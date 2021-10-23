package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	LEADER = iota + 1
	CANDIDATE
	FOLLOWER

	ADD    = 1
	REMOVE = -1

	// A big enough buffer size to avoid blocking on sending the RPC.
	defaultBufferSize            = 1000
	rpcTimeout                   = time.Duration(100) * time.Millisecond
	heartbeatTimeout             = time.Duration(1000) * time.Millisecond
	electionTimeout              = time.Duration(500) * time.Millisecond
	minElectionTimeoutMultiplier = 4
	maxElectionTimeoutMultiplier = 10
)

type response struct {
	// LeaderId int // only used to redirect client request.
	Term    int   `json:"term"`
	Success bool  `json:"success"`
	Err     error `json:"err"`
}

type clientRequest struct {
	Val int `json:"val"`
}

type appendEntryRequest struct {
	Term           int       `json:"term"`
	LeaderId       int       `json:"leaderId"`
	PrevLogIndex   int       `json:"prevLogIndex"`
	PrevLogTerm    int       `json:"prevLogTerm"`
	CommittedIndex int       `json:"committedIndex"`
	Entry          *logEntry `json:"entry"`
}

type requestVoteRequest struct {
	Term        int `json:"term"`
	CandidateId int `json:"candidateId"`
	LogSize     int `json:"logSize"`
	LastLogTerm int `json:"lastLogTerm"`
}

type addServerRequest struct {
	ServerAddr string `json:"serverAddr"`
	ServerId   int    `json:"serverId"`
}

type removeServerRequest struct {
	ServerId int `json:"serverId"`
}

type logEntry struct {
	Term       int    `json:"term"`
	Val        int    `json:"val"`
	ServerAddr string `json:"serverAddr"`
	ServerID   int    `json:"serverID"`
}

// server struct represents a server in Raft protocol.
// It could be a leader, follower, or candidate.
type server struct {
	id           int
	initialPeers map[int]string
	mu           sync.Mutex
	running      bool
	done         chan struct{}

	// Persistent state that won't be reset when killed/restart.
	currentTerm int
	votedFor    map[int]int // term -> voted for candidateId.
	logs        []logEntry

	// Volatile state that will be reset when killed/restart.
	leaderId                    int
	nextIndex                   map[int]int // The next log entry to send to peers.
	role                        int
	committedIndex              int
	cummulativeHeartbeatTimeout time.Duration

	inbound chan struct{}
}

func (s *server) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func newServer(id int) *server {
	return &server{
		id:       id,
		votedFor: map[int]int{},
		done:     make(chan struct{}, 1),
		inbound:  make(chan struct{}, 1),
	}
}

// kill shuts down the server.
// This can be used to simulate crash.
func (s *server) kill() {
	s.mu.Lock()
	if s.running {
		s.running = false
		defer func() {
			<-s.done
		}()
	}
	s.mu.Unlock()
}

// start launches the server.
// This resets the volatile state.
func (s *server) start(peers map[int]string, forceLeader bool) error {
	s.mu.Lock()
	if s.running {
		return fmt.Errorf("server is already running; call kill() first")
	}
	s.running = true
	if forceLeader {
		s.role = LEADER
	} else {
		s.role = FOLLOWER
	}
	s.initialPeers = map[int]string{}
	s.nextIndex = map[int]int{}
	for id, svr := range peers {
		if s.role == LEADER && s.id != id {
			// Initialize to be the current last log entry + 1.
			s.nextIndex[id] = len(s.logs)
		}
		if id == s.id {
			continue
		}
		s.initialPeers[id] = svr
	}
	s.committedIndex = -1
	s.mu.Unlock()
	go s.run()
	http.HandleFunc("/", s.handle)
	http.ListenAndServe(":80", http.DefaultServeMux)
	return nil
}

// getCurrentPeers returns the current set of peers.
// Peers may change from initialPeers due to addServer or removeServer RPCs.
func (s *server) getCurrentPeers() map[int]string {
	ret := map[int]string{}
	for id, svr := range s.initialPeers {
		ret[id] = svr
	}
	for _, e := range s.logs {
		if e.ServerAddr != "" {
			if e.Val == ADD && e.ServerID != s.id {
				ret[e.ServerID] = e.ServerAddr
			} else {
				delete(ret, e.ServerID)
			}
		}
	}
	return ret
}

func newElectionCountdown() time.Duration {
	x := rand.Intn(maxElectionTimeoutMultiplier - minElectionTimeoutMultiplier)
	return electionTimeout * time.Duration(x+minElectionTimeoutMultiplier)
}

func (s *server) shouldStepDown() bool {
	if s.role != LEADER {
		return false
	}
	for i := len(s.logs) - 1; i >= 0; i-- {
		if s.logs[i].ServerID != s.id {
			continue
		}
		if s.logs[i].Val == ADD {
			// It's an ADD.
			return false
		}
		// It's remove.
		// Leader only steps down then the entry is committed.
		if s.committedIndex >= i {
			return true
		}
		break
	}
	return false
}

func (s *server) run() {
	// TODO needs locking
	electionCountdown := newElectionCountdown()
	for s.isRunning() {
		if s.shouldStepDown() {
			s.mu.Lock()
			s.running = false
			log.Println("STEPPING DOWN")
			s.mu.Unlock()
			break
		}
		select {
		case <-s.inbound:
			break // restart timer, since we received a request
		case <-time.After(heartbeatTimeout):
			switch s.role {
			case FOLLOWER:
				s.mu.Lock()
				// No request from leader. Count down election.
				s.cummulativeHeartbeatTimeout += heartbeatTimeout
				if electionCountdown < s.cummulativeHeartbeatTimeout {
					// Start election.
					s.role = CANDIDATE
				}
				s.mu.Unlock()
			case LEADER:
				// No request from client. Need to send heartbeat.
				s.mu.Lock()
				s.broadcastEntry()
				s.mu.Unlock()
			case CANDIDATE:
				s.mu.Lock()
				if electionCountdown < s.cummulativeHeartbeatTimeout {
					// Clear election countdown.
					electionCountdown = newElectionCountdown()
					s.cummulativeHeartbeatTimeout = 0
					// Run election.
					if s.runElection() {
						// Become leader.
						s.role = LEADER
						s.leaderId = s.id
						log.Printf("server %d becomes leader", s.id)
						for id, _ := range s.getCurrentPeers() {
							// Initialize to be the current last log entry + 1.
							s.nextIndex[id] = len(s.logs)
						}
					}
				}
				s.mu.Unlock()
			}
		}
	}
	s.done <- struct{}{}
}

func sendHelper(hostname string, req interface{}) response {
	data, _ := json.Marshal(req)
	url := "http://" + hostname
	switch req.(type) {
	case clientRequest:
		url += "/client"
	case addServerRequest:
		url += "/add_server"
	case removeServerRequest:
		url += "/remove_server"
	case appendEntryRequest:
		url += "/append_entry"
	case requestVoteRequest:
		url += "/request_vote"
	default:
		return response{Err: errors.New("bad internal request")}
	}
	log.Println(url)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return response{Err: err}
	}
	body, _ := io.ReadAll(resp.Body)
	log.Println(string(body))
	re := response{}
	err = json.Unmarshal(body, &re)
	if err != nil {
		return response{Err: err}
	}
	return re
}

func (s *server) send(reqs map[int]interface{}) map[int]response {
	var mu sync.Mutex
	resps := map[int]response{}
	var wg sync.WaitGroup
	for id, req := range reqs {
		sid := id   // record peer ID.
		sreq := req // record the request.
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := sendHelper(s.getCurrentPeers()[sid], sreq)
			mu.Lock()
			resps[sid] = resp
			mu.Unlock()
		}()
	}
	wg.Wait()
	return resps
}

func (s *server) handleRequestHelper(w http.ResponseWriter, r *http.Request, potentialLogEntry logEntry) {
	switch s.role {
	case FOLLOWER:
		// proxy client request to leader.
		leaderHost := s.getCurrentPeers()[s.leaderId]
		req := &http.Request{Method: r.Method, Body: r.Body, Header: r.Header.Clone()}
		req.URL, _ = url.Parse("http://" + leaderHost + "/" + strings.TrimPrefix(r.URL.Path, "/"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		io.Copy(w, resp.Body)
	case CANDIDATE:
		// No leader yet, returns error to client.
		data, _ := json.Marshal(response{Err: fmt.Errorf("no leader; please try later")})
		w.Write(data)
	case LEADER:
		s.logs = append(s.logs, potentialLogEntry)
		if s.broadcastEntry() {
			data, _ := json.Marshal(response{Term: s.currentTerm, Success: true})
			w.Write(data)
		} else {
			data, _ := json.Marshal(response{Term: s.currentTerm})
			w.Write(data)
		}
	}
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	log.Println(path)
	s.inbound <- struct{}{}
	buf := bytes.NewBuffer([]byte{})
	tee := io.TeeReader(r.Body, buf)
	data, err := io.ReadAll(tee)
	r.Body = io.NopCloser(buf)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch path {
	case "add_server":
		req := addServerRequest{}
		err = json.Unmarshal(data, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.handleRequestHelper(w, r, logEntry{Term: s.currentTerm, Val: ADD, ServerAddr: req.ServerAddr, ServerID: req.ServerId})
	case "remove_server":
		req := removeServerRequest{}
		err = json.Unmarshal(data, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.handleRequestHelper(w, r, logEntry{Term: s.currentTerm, Val: REMOVE, ServerID: req.ServerId, ServerAddr: "x"})
	case "client":
		if r.Method == http.MethodGet {
			data, _ := json.Marshal(s.logs[0 : s.committedIndex+1])
			w.Write(data)
			return
		}
		req := clientRequest{}
		err = json.Unmarshal(data, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.handleRequestHelper(w, r, logEntry{Term: s.currentTerm, Val: req.Val})
	case "append_entry":
		req := appendEntryRequest{}
		err = json.Unmarshal(data, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Term < s.currentTerm {
			data, _ := json.Marshal(response{Term: s.currentTerm})
			w.Write(data)
			return
		} else if req.Term >= s.currentTerm {
			s.role = FOLLOWER
			s.currentTerm = req.Term
		}
		s.cummulativeHeartbeatTimeout = 0
		s.leaderId = req.LeaderId
		s.committedIndex = req.CommittedIndex
		// Check if preceding entry matches.
		if req.PrevLogIndex == -1 || req.PrevLogIndex < len(s.logs) && s.logs[req.PrevLogIndex].Term == req.PrevLogTerm {
			if len(s.logs) > 0 {
				s.logs = s.logs[0 : req.PrevLogIndex+1]
			}
			if req.Entry != nil {
				s.logs = append(s.logs, *req.Entry)
			}
			data, _ := json.Marshal(response{Term: s.currentTerm, Success: true})
			w.Write(data)
			return
		} else {
			data, _ := json.Marshal(response{Term: s.currentTerm})
			w.Write(data)
			return
		}
	case "request_vote":
		req := requestVoteRequest{}
		err = json.Unmarshal(data, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Term < s.currentTerm || s.cummulativeHeartbeatTimeout < minElectionTimeoutMultiplier*electionTimeout {
			data, _ := json.Marshal(response{Term: s.currentTerm})
			w.Write(data)
			return
		} else if req.Term > s.currentTerm {
			s.role = FOLLOWER
			s.currentTerm = req.Term
		}
		if votedFor, has := s.votedFor[s.currentTerm]; !has || votedFor == req.CandidateId {
			var vote bool
			// Compare logs.
			var ownLastTerm int
			if len(s.logs) > 0 {
				ownLastTerm = s.logs[len(s.logs)-1].Term
			}
			if ownLastTerm < req.LastLogTerm {
				vote = true
			}
			if ownLastTerm == req.LastLogTerm && len(s.logs) <= req.LogSize {
				vote = true
			}
			if vote {
				s.votedFor[s.currentTerm] = req.CandidateId
				data, _ := json.Marshal(response{Term: s.currentTerm, Success: true})
				w.Write(data)
				return
			} else {
				data, _ := json.Marshal(response{Term: s.currentTerm})
				w.Write(data)
				return
			}
		} else {
			// Already voted for somebody else in this term.
			data, _ := json.Marshal(response{Term: s.currentTerm})
			w.Write(data)
			return
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// broadcastEntry sends one entry to each follower based on next index.
// The function returns true if majority of the followers respond success.
func (s *server) broadcastEntry() bool {
	log.Printf("server %d sending broadcast with logs: %v; nextIndex: %v", s.id, s.logs, s.nextIndex)
	reqs := map[int]interface{}{}
	for id, _ := range s.getCurrentPeers() {
		req := appendEntryRequest{
			Term:           s.currentTerm,
			LeaderId:       s.id,
			PrevLogIndex:   s.nextIndex[id] - 1,
			CommittedIndex: s.committedIndex,
		}
		if s.nextIndex[id] < len(s.logs) {
			// next index could be equal to current log size in which case this is a heartbeat.
			req.Entry = &s.logs[s.nextIndex[id]]
		}
		if req.PrevLogIndex >= 0 {
			req.PrevLogTerm = s.logs[req.PrevLogIndex].Term
		}
		reqs[id] = req
	}
	resps := s.send(reqs)
	// Go over results
	successCount := 1 // count itself.
	var successFromCurrentTerm []int
	for id, resp := range resps {
		if resp.Err != nil {
			// Skip this time and retry later.
			log.Printf("server %d failed to receive broadcast response from server %d", s.id, id)
			continue
		}
		if resp.Term > s.currentTerm {
			// Convert to follower.
			log.Printf("server %d current term %d detects broadcast term conflict %d", s.id, s.currentTerm, resp.Term)
			s.currentTerm = resp.Term
			s.role = FOLLOWER
			return false
		}
		if resp.Success {
			successCount += 1
			// Advance next index for this follower.
			if s.nextIndex[id] < len(s.logs) {
				s.nextIndex[id] += 1
			}
			if s.nextIndex[id] > 0 && s.logs[s.nextIndex[id]-1].Term == s.currentTerm {
				successFromCurrentTerm = append(successFromCurrentTerm, s.nextIndex[id]-1)
			}
		} else {
			// Backtrack next index for this follower.
			s.nextIndex[id] -= 1
		}
	}
	// Update committed index.
	if len(successFromCurrentTerm) >= len(s.getCurrentPeers())/2 && len(successFromCurrentTerm) > 0 {
		sort.Ints(successFromCurrentTerm)
		// Committed index is the largest index that has received majority ack for the current term.
		s.committedIndex = successFromCurrentTerm[len(successFromCurrentTerm)-len(s.getCurrentPeers())/2]
	}
	if len(s.getCurrentPeers()) == 0 {
		s.committedIndex = len(s.logs) - 1
	}
	return successCount >= len(s.getCurrentPeers())/2+1
}

// runElection runs for leader.
// The function returns true if successfully elected.
func (s *server) runElection() bool {
	s.currentTerm += 1
	s.votedFor[s.currentTerm] = s.id
	log.Printf("server %d starts election for term %d", s.id, s.currentTerm)
	reqs := map[int]interface{}{}
	for id, _ := range s.getCurrentPeers() {
		req := requestVoteRequest{
			Term:        s.currentTerm,
			CandidateId: s.id,
			LogSize:     len(s.logs),
		}
		if req.LogSize > 0 {
			req.LastLogTerm = s.logs[req.LogSize-1].Term
		}
		reqs[id] = req
	}
	resps := s.send(reqs)
	// Go over results
	successCount := 1 // count itself.
	for id, resp := range resps {
		if resp.Err != nil {
			// Skip this time and retry later.
			log.Printf("server %d failed to receive vote response from server %d", s.id, id)
			continue
		}
		if resp.Term > s.currentTerm {
			// Convert to follower.
			log.Printf("server %d current term %d detects election term conflict %d", s.id, s.currentTerm, resp.Term)
			s.currentTerm = resp.Term
			s.role = FOLLOWER
			return false
		}
		if resp.Success {
			successCount += 1
		}
	}
	return successCount >= len(s.getCurrentPeers())/2+1
}
