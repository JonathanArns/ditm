package main

import (
	"bytes"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Request struct {
	From             string      `json:"from"`
	FromName         string      `json:"from_name"`
	To               string      `json:"to"`
	ToName           string      `json:"to_name"`
	StreamIdentifier string      `json:"stream_identifier"`
	Method           string      `json:"method"`
	Timestamp        time.Time   `json:"timestamp"`
	BodyLength       int         `json:"body_length"`
	Blocked          bool        `json:"blocked"`
	BlockedResponse  bool        `json:"blocked_response"`
	FromOutside      bool        `json:"from_outside"`
	Body             []byte      `json:"body"`
	ResponseBody     []byte      `json:"response_body"`
	Header           http.Header `json:"header"`
}

type Recording struct {
	Requests     []*Request    `json:"requests"`
	Logs         []LogEntry    `json:"logs"`
	Volumes      string        `json:"volumes"`
	BlockConfigs []BlockConfig `json:"block_configs"`
	StartTime    time.Time     `json:"start_time"`
}

type LogEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Message       string    `json:"message"`
	ContainerName string    `json:"container_name"`
}

func (r *Recording) getStream(streamIdentifier string) []*Request {
	ret := []*Request{}
	for _, request := range r.Requests {
		if request.StreamIdentifier == streamIdentifier {
			ret = append(ret, request)
		}
	}
	return ret
}

type Proxy struct {
	mu            *sync.Mutex
	hostNames     map[string]string
	isRecording   bool
	isReplaying   bool
	isInspecting  bool
	blockConfig   BlockConfig
	recording     Recording
	replayingFrom Recording
	replayTimer   *time.Timer
	lastSavedId   int
	matcher       Matcher
	endReplayC    chan struct{}
}

type BlockConfig struct {
	Mode         string     `json:"mode"`
	Partitions   [][]string `json:"partitions"`
	Percentage   int        `json:"percentage"`
	Matcher      string     `json:"matcher"`
	Timestamp    time.Time  `json:"timestamp"`
	previousMode string
}

// returns true when the request is allowed by the current partitions
func (b BlockConfig) checkPartitions(request *Request) bool {
	for _, partition := range b.Partitions {
		from := false
		to := false
		for _, node := range partition {
			if node == request.FromName {
				from = true
			} else if node == request.ToName {
				to = true
			}
		}
		if from && to {
			return true
		}
	}
	return false
}

func (b BlockConfig) Block(r *Request, replayBlock func(*Request) (bool, bool)) (request bool, response bool) {
	if r.FromOutside {
		return false, false
	}
	switch b.Mode {
	case "none":
		break
	case "random":
		request = rand.Float32() < float32(b.Percentage)/100
		if !request {
			response = rand.Float32() < float32(b.Percentage)/100
		}
		return request, response
	case "partitions":
		return !b.checkPartitions(r), false
	case "replay":
		return replayBlock(r)
	}
	return false, false
}

func (p *Proxy) replayBlock(request *Request) (bool, bool) {
	bestMatch := p.matcher.Match(request, p.recording, p.replayingFrom)
	if bestMatch != nil {
		return bestMatch.Blocked, bestMatch.BlockedResponse
	}
	return false, false
}

// This is where all the proxy requests are handled.
// The handler records requests and decides wether or not to block them,
// before either proxying the request or calling panig() to close the connection.
func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	log.Println("lock handler")
	p.mu.Lock()
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)
	body, err := io.ReadAll(tee)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	r.Body = io.NopCloser(&buf)
	request := &Request{
		From:       r.RemoteAddr,
		To:         r.URL.String(),
		ToName:     strings.Split(r.URL.Host, ":")[0],
		BodyLength: len(body),
		Body:       body,
		Method:     r.Method,
		Timestamp:  time.Now(),
		Header:     r.Header,
	}

	// perform reverse lookup
	ip, _, _ := net.SplitHostPort(request.From)
	if fromName, ok := p.hostNames[ip]; ok {
		request.FromName = fromName
	} else {
		request.FromOutside = true
		request.FromName = "outside"
	}
	request.StreamIdentifier = request.FromName + " " + request.ToName

	request.Blocked, request.BlockedResponse = p.blockConfig.Block(request, p.replayBlock)
	if !p.isInspecting {
		p.record(request)
	}
	log.Println("unlock handler")
	p.mu.Unlock()
	if request.Blocked {
		panic("We want to block this request")
	}

	// proxy the request
	remoteHost, err := url.Parse("http://" + r.URL.Host)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(remoteHost)
	proxy.ModifyResponse = func(r *http.Response) error {
		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)
		body, err := io.ReadAll(tee)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(&buf)
		log.Println("lock response handler")
		p.mu.Lock()
		request.ResponseBody = body
		log.Println("unlock response handler")
		p.mu.Unlock()
		if request.BlockedResponse {
			panic("We want to block this response")
		}
		return nil
	}
	proxy.ServeHTTP(w, r)
	if p.isReplaying {
		if p.nextOutsideRequest(false) {
			p.ResetReplayTimer()
		}
	}
}

// appends a request to the current recording
// p.mu has to be locked when calling record
func (p *Proxy) record(request *Request) {
	p.recording.Requests = append(p.recording.Requests, request)
}

func (p *Proxy) ResetReplayTimer() {
	if p.replayTimer == nil {
		return
	}
	p.replayTimer.Reset(time.Duration(4000) * time.Millisecond)
}

// Checks if the next unseen request in the recording is an
// outside request, if so, sends the request.
// If alwaysSend is true, the next unseen request is always sent,
// even if there are unseen requests from inside are before it.
func (p *Proxy) nextOutsideRequest(alwaysSend bool) bool {
	log.Println("lock nextOutsideRequest")
	p.mu.Lock()
	unseenFlag := false
	for _, request := range p.replayingFrom.Requests {
		if !p.matcher.Seen(request) {
			if request.FromOutside {
				p.matcher.MarkSeen(request)
				r := *request // make a copy of request, to record it with new timestamp
				r.Timestamp = time.Now()
				p.record(&r)
				go send(&r)
				if unseenFlag {
					log.Println("unlock nextOutsideRequest1")
					p.mu.Unlock()
					return true
				}
			} else if !alwaysSend {
				log.Println("unlock nextOutsideRequest2")
				p.mu.Unlock()
				return true // exit because we need to see some other requests first
			}
			unseenFlag = true
		}
	}
	log.Println("unlock nextOutsideRequest3")
	p.mu.Unlock()
	p.endReplayC <- struct{}{}
	return false
}

func send(r *Request) {
	request, err := http.NewRequest(r.Method, r.To, bytes.NewReader(r.Body))
	if err != nil {
		log.Println(err)
		return
	}
	request.Header = r.Header
	resp, err := http.DefaultClient.Do(request)
	if err == nil {
		r.ResponseBody, _ = io.ReadAll(resp.Body)
	} else {
		r.ResponseBody = []byte{}
		log.Println(err)
	}
}

func (p *Proxy) EndReplay() {
	log.Println("lock EndReplay")
	p.mu.Lock()
	select {
	case <-p.endReplayC:
		break
	default:
		for _, r := range p.replayingFrom.Requests {
			if !p.matcher.Seen(r) {
				p.ResetReplayTimer()
				log.Println("unlock EndReplay1")
				p.mu.Unlock()
				p.nextOutsideRequest(true)
				return
			}
		}
	}
	log.Println("unlock EndReplay2")
	p.mu.Unlock()
	time.Sleep(time.Millisecond * 100)
	log.Println("lock EndReplay2")
	p.mu.Lock()
	p.writeRecording()
	p.isReplaying = false
	p.blockConfig.Mode = p.blockConfig.previousMode
	p.replayTimer = nil
	log.Println("unlock EndReplay3")
	p.mu.Unlock()
	log.Println("replay finished")
}
