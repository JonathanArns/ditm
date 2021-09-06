package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mholt/archiver/v3"
)

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
}

type Recording struct {
	Requests []*Request `json:"requests"`
	Logs     []LogEntry `json:"logs"`
	Volumes  string     `json:"volumes"`
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

type BlockConfig struct {
	Mode         string     `json:"mode"`
	Partitions   [][]string `json:"partitions"`
	Percentage   int        `json:"percentage"`
	timesUsed    int
	previousMode string
}

type Request struct {
	From             string      `json:"from"`
	FromName         string      `json:"from_name"`
	To               string      `json:"to"`
	ToName           string      `json:"to_name"`
	StreamIdentifier string      `json:"stream_identifier"`
	Method           string      `json:"method"`
	Timestamp        time.Time   `json:"timestamp"`
	BodyLength       int         `json:"body_length"`
	TLS              bool        `json:"tls"`
	Blocked          bool        `json:"blocked"`
	BlockedResponse  bool        `json:"blocked_response"`
	FromOutside      bool        `json:"from_outside"`
	Body             []byte      `json:"body"`
	Header           http.Header `json:"header"`
	seen             bool        `json:"-"`
}

func (p *Proxy) replayBlock(request *Request) (bool, bool) {
	recording := p.recording.getStream(request.StreamIdentifier)
	replayingFrom := p.replayingFrom.getStream(request.StreamIdentifier)

	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(replayingFrom)) // a faktor to relativize constant score components
	log.Println(recording)
	log.Println(replayingFrom)
	for i, r := range replayingFrom {
		// TODO: this doesn't work at all right now
		if r.seen || r.FromOutside {
			continue
		}
		if i == len(recording) {
			log.Println("hello")
			bestMatch = r
			break
		}
		score := 0.0
		score -= math.Abs(float64(i - len(recording)))
		if r.To == request.To {
			score += 1 * faktor
		}
		if score > highScore {
			highScore = score
			bestMatch = r
		}
	}
	if bestMatch != nil {
		bestMatch.seen = true
		return bestMatch.Blocked, bestMatch.BlockedResponse
	}
	// log.Println("WE ARE SEEING MORE REQUESTS FOR THIS STREAM THAN IN THE ORIGINAL RECORDING")
	return false, false
}

func (p *Proxy) Block(r *Request) (request bool, response bool) {
	if r.FromOutside {
		return false, false
	}
	switch p.blockConfig.Mode {
	case "none":
		break
	case "random":
		request = rand.Float32() < float32(p.blockConfig.Percentage)/100
		if !request {
			response = rand.Float32() < float32(p.blockConfig.Percentage)/100
		}
		return request, response
	case "partitions":
		return !p.checkPartitions(r), false
	case "replay":
		return p.replayBlock(r)
	}
	return false, false
}

// This is where all the proxy requests are handled.
// The handler records requests and decides wether or not to block them,
// before either proxying the request or calling panig() to close the connection.
func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.ResetReplayTimer()

	var proto string
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
		BodyLength: len(body),
		Body:       body,
		Method:     r.Method,
		Timestamp:  time.Now(),
	}
	if r.TLS == nil {
		request.TLS = false
		proto = "http://"
	} else {
		request.TLS = true
		proto = "https://"
	}

	// perform reverse lookup
	ip, _, _ := net.SplitHostPort(request.From)
	if fromName, ok := p.hostNames[ip]; ok {
		request.FromName = fromName
	} else {
		request.FromOutside = true
		request.FromName = "outside"
	}
	request.StreamIdentifier = request.FromName + "->" + request.To

	request.Blocked, request.BlockedResponse = p.Block(request)
	if !p.isInspecting {
		p.record(request)
	}
	p.mu.Unlock()
	if request.Blocked {
		panic("We want to block this request")
	}

	// proxy the request
	remoteHost, err := url.Parse(proto + r.URL.Host)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(remoteHost)
	if request.BlockedResponse {
		proxy.ModifyResponse = func(r *http.Response) error {
			panic("We want to block this response")
		}
	}
	proxy.ServeHTTP(w, r)
	if p.isReplaying {
		p.nextOutsideRequest(false)
	}
}

// returns true when the request is allowed by the current partitions
func (p *Proxy) checkPartitions(request *Request) bool {
	for _, partition := range p.blockConfig.Partitions {
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

// appends a request to the current recording
// p.mu has to be locked when calling record
func (p *Proxy) record(request *Request) {
	p.recording.Requests = append(p.recording.Requests, request)
}

func (p *Proxy) writeRecording() (int, error) {
	filename, fileId := newFilePath("/recordings/", ".json")
	p.lastSavedId = fileId
	bytes, err := json.MarshalIndent(p.recording, "", " ")
	if err != nil {
		return 0, err
	}
	return fileId, os.WriteFile(filename, bytes, 0b_110110110)
}

func newFilePath(prefix, postfix string) (string, int) {
	fileId := 1
	for {
		if _, err := os.Stat(prefix + strconv.Itoa(fileId) + postfix); os.IsNotExist(err) {
			log.Println(fileId)
			return prefix + strconv.Itoa(fileId) + postfix, fileId
		}
		fileId += 1
	}
}

func ListFileIDs(dirname, fileext string) []int {
	ret := sort.IntSlice{}
	files, err := os.ReadDir(dirname)
	if err != nil {
		log.Fatal(err)
	}
	for _, info := range files {
		if !info.IsDir() {
			name := strings.Trim(info.Name(), fileext)
			if id, err := strconv.Atoi(name); err == nil {
				ret = append(ret, id)
			}
		}
	}
	ret.Sort()
	return ret
}

func ListRecordings() []int {
	return ListFileIDs("/recordings", ".json")
}

func ListVolumesSnapshots() []int {
	return ListFileIDs("/volumes", ".zip")
}

func loadRecording(id string) (*Recording, error) {
	filepath := "/recordings/" + id + ".json"
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	recording := &Recording{}
	err = json.Unmarshal(bytes, recording)
	if err != nil {
		return nil, err
	}
	return recording, nil
}

func (p *Proxy) ResetReplayTimer() {
	if p.replayTimer == nil {
		return
	}
	p.replayTimer.Reset(time.Duration(3) * time.Second)
}

// Checks if the next unseen request in the recording is an
// outside request, if so, sends the request.
// If alwaysSend is true, the next unseen request is always sent,
// even if there are unseen requests from inside are before it.
func (p *Proxy) nextOutsideRequest(alwaysSend bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, request := range p.replayingFrom.Requests {
		if !request.seen && request.FromOutside {
			request.seen = true
			p.mu.Unlock()
			_, err := send(request)
			if err != nil {
				log.Println(err)
			}
			r := *request // make a copy of request, to record it with new timestamp
			r.Timestamp = time.Now()
			p.mu.Lock()
			p.record(&r)
			return
		} else if !alwaysSend && !request.seen {
			return // exit because we need to see some other requests first
		}
	}
}

func send(r *Request) (*http.Response, error) {
	url, err := url.Parse(r.To)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	request := &http.Request{
		Method: r.Method,
		URL:    url,
		Body:   io.NopCloser(bytes.NewReader(r.Body)),
		Header: r.Header,
	}
	return http.DefaultClient.Do(request)
}

func (p *Proxy) EndReplay() {
	log.Println("end replay")
	p.mu.Lock()
	for _, r := range p.replayingFrom.Requests {
		if !r.seen {
			p.ResetReplayTimer()
			p.mu.Unlock()
			p.nextOutsideRequest(true)
			return
		}
	}
	p.writeRecording()
	p.isReplaying = false
	p.blockConfig.Mode = p.blockConfig.previousMode
	p.replayTimer = nil
	p.mu.Unlock()
	log.Println("replay finished")
}

func writeVolumes() (int, error) {
	filename, fileId := newFilePath("/snapshots/", ".zip")
	err := archiver.Archive([]string{"/volumes"}, filename)
	return fileId, err
}

func loadVolumes(id string) error {
	if id == "" {
		return errors.New("No Volumes Snapshot")
	}
	filepath := "/volumes/" + id + ".zip"
	err := archiver.Unarchive(filepath, "/volumes")
	return err
}

func latestVolumes() string {
	files, _ := os.ReadDir("/snapshots")
	latest := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.Trim(file.Name(), ".zip")
		if id, err := strconv.Atoi(name); err == nil {
			if id > latest {
				latest = id
			}
		}
	}
	return strconv.Itoa(latest)
}

type TemplateData struct {
	ModeDefault     bool
	ModeRecording   bool
	ModeReplaying   bool
	ModeInspecting  bool
	Recordings      []int
	Volumes         []int
	Partitions      string
	Percentage      int
	BlockNone       bool
	BlockPartitions bool
	BlockRandom     bool
}

func (p *Proxy) NewTemplateData() TemplateData {
	partitions, _ := json.Marshal(p.blockConfig.Partitions)
	return TemplateData{
		ModeDefault:     !p.isRecording && !p.isReplaying && !p.isInspecting,
		ModeRecording:   p.isRecording,
		ModeReplaying:   p.isReplaying,
		ModeInspecting:  p.isInspecting,
		Recordings:      ListRecordings(),
		Volumes:         ListVolumesSnapshots(),
		Partitions:      string(partitions),
		Percentage:      p.blockConfig.Percentage,
		BlockNone:       p.blockConfig.Mode == "none",
		BlockPartitions: p.blockConfig.Mode == "partitions",
		BlockRandom:     p.blockConfig.Mode == "random",
	}
}

type TemplateEvent struct {
	IsRequest bool
	Request   Request
	Log       LogEntry
}

//go:embed templates/main.html
var mainTemplate string

//go:embed templates/recording.html
var recordingTemplate string

func (p *Proxy) HomeHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	if p.isReplaying {
		p.replayTimer.Stop()
		p.replayTimer = nil
		p.isReplaying = false
		log.Println("replay canceled")
	}
	p.recording = Recording{Requests: []*Request{}}
	p.isInspecting = false

	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	err := t.Execute(w, p.NewTemplateData())
	if err != nil {
		log.Println(err)
	}
}

func (p *Proxy) InspectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	recording, err := loadRecording(id)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isReplaying {
		p.replayTimer.Stop()
		p.replayTimer = nil
		p.isReplaying = false
		log.Println("replay canceled")
	}
	p.isRecording = false
	p.isInspecting = true
	p.recording = *recording

	t := template.New("main")
	t.Parse(mainTemplate)
	err = t.Execute(w, p.NewTemplateData())
	if err != nil {
		log.Println(err)
	}
}

func (p *Proxy) BlockConfigHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isReplaying {
		return
	}
	if mode := r.FormValue("mode"); mode != "" {
		p.blockConfig.previousMode = p.blockConfig.Mode
		p.blockConfig.Mode = mode
	}
	if partitions := r.FormValue("partitions"); partitions != "" {
		parts := [][]string{}
		json.Unmarshal([]byte(partitions), &parts)
		p.blockConfig.Partitions = parts
	}
	if percentage := r.FormValue("percentage"); percentage != "" {
		if i, err := strconv.Atoi(percentage); err == nil {
			p.blockConfig.Percentage = i
		}
	}
	t := template.New("main")
	t.Parse(mainTemplate)
	err := t.Execute(w, p.NewTemplateData())
	if err != nil {
		log.Println(err)
	}
}

func (p *Proxy) StartRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isRecording = true
	p.isInspecting = false
	p.recording = Recording{Requests: []*Request{}}
	p.replayingFrom = Recording{Requests: []*Request{}}
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) EndRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.writeRecording()
	if err != nil {
		log.Println(err)
	}
	p.recording = Recording{Requests: []*Request{}}
	p.isRecording = false
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) SaveVolumesHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := writeVolumes()
	if err != nil {
		log.Println(err)
	}
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) LoadVolumesHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	filename := r.FormValue("id")
	if filename == "" {
		filename = latestVolumes()
	}
	err := loadVolumes(filename)
	if err != nil {
		log.Println(err)
	}
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) StartReplayHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	id := r.FormValue("id")
	p.isRecording = false
	recording, err := loadRecording(id)
	if err != nil {
		log.Println(err)
	}
	p.replayingFrom = *recording
	p.recording = Recording{Requests: []*Request{}}
	err = loadVolumes(p.recording.Volumes)
	if err != nil {
		log.Println(err)
	}
	p.isReplaying = true
	p.isInspecting = false
	p.blockConfig.previousMode = p.blockConfig.Mode
	p.blockConfig.Mode = "replay"
	p.replayTimer = time.AfterFunc(time.Duration(3)*time.Second, p.EndReplay)
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
	p.nextOutsideRequest(true)
}

func (p *Proxy) LiveUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	t := template.New("recording")
	t.Parse("data: " + strings.ReplaceAll(recordingTemplate, "\n", "") + "\n\n")
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher := w.(http.Flusher)
	isReplaying := p.isReplaying
	writtenRequests := 0
	writtenLogs := 0
	var event TemplateEvent
	for {
		select {
		case <-time.After(1 * time.Second):
			p.mu.Lock()
			if isReplaying != p.isReplaying {
				w.Write([]byte(fmt.Sprintf("event: finished\ndata: %v\n\n", p.lastSavedId)))
				flusher.Flush()
				p.mu.Unlock()
				return
			}
			event.IsRequest = false
			for writtenLogs < len(p.recording.Logs) {
				event.Log = p.recording.Logs[writtenLogs]
				err := t.Execute(w, event)
				if err != nil {
					log.Println(err)
					return
				}
				writtenLogs += 1
			}
			event.IsRequest = true
			for writtenRequests < len(p.recording.Requests) {
				event.Request = *p.recording.Requests[writtenRequests]
				err := t.Execute(w, event)
				if err != nil {
					log.Println(err)
					return
				}
				writtenRequests += 1
			}
			p.mu.Unlock()
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (p *Proxy) LogHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}
	log.Println("log:", string(data))
	msg := LogEntry{}
	err = json.Unmarshal(data, &msg)
	if err != nil {
		log.Println(err)
	}
	p.mu.Lock()
	if !p.isInspecting {
		p.recording.Logs = append(p.recording.Logs, msg)
	}
	p.mu.Unlock()
}
