package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mholt/archiver/v3"
)

type Proxy struct {
	mu              sync.Mutex
	hostNames       map[string]string
	isRecording     bool
	isReplaying     bool
	blockPercentage int
	recording       Recording
	replayingFrom   Recording
	replayTimer     *time.Timer
}

type Recording struct {
	Requests []*Request `json:"requests"`
	Volumes  string     `json:"volumes"`
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

type Request struct {
	From             string      `json:"from"`
	FromName         string      `json:"from_name"`
	To               string      `json:"to"`
	StreamIdentifier string      `json:"stream_identifier"`
	URI              string      `json:"uri"`
	Method           string      `json:"method"`
	Timestamp        time.Time   `json:"timestamp"`
	BodyLength       int         `json:"body_length"`
	TLS              bool        `json:"tls"`
	Blocked          bool        `json:"blocked"`
	FromOutside      bool        `json:"from_outside"`
	Body             []byte      `json:"body"`
	Header           http.Header `json:"header"`
	seen             bool        `json:"-"`
}

func (p *Proxy) Block(request *Request) bool {
	if p.isRecording {
		return rand.Float32() < float32(p.blockPercentage)/100
	} else if !p.isReplaying {
		return false
	}

	recording := p.recording.getStream(request.StreamIdentifier)
	replayingFrom := p.replayingFrom.getStream(request.StreamIdentifier)

	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(replayingFrom)) // a faktor to relativize constant score components
	for i, r := range replayingFrom {
		if r.seen {
			continue
		}
		score := 0.0
		score -= math.Abs(float64(i - len(recording)))
		if r.URI == request.URI {
			score += 1 * faktor
		}
		if score > highScore {
			highScore = score
			bestMatch = r
		}
	}
	if bestMatch != nil {
		bestMatch.seen = true
		return bestMatch.Blocked
	}
	log.Println("WE ARE SEEING MORE REQUESTS FOR THIS STREAM THAN IN THE ORIGINAL RECORDING")
	return false
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ResetReplayTimer()

	var proto string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	request := &Request{
		From:       r.RemoteAddr,
		To:         r.URL.String(),
		BodyLength: len(body),
		URI:        r.URL.RequestURI(),
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

	request.Blocked = p.Block(request)
	p.record(request)
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
	proxy.ServeHTTP(w, r)
	p.nextOutsideRequest(false)
}

func (p *Proxy) record(request *Request) {
	p.recording.Requests = append(p.recording.Requests, request)
}

func (p *Proxy) writeRecording() (int, error) {
	filename, fileId := newFilePath("/recordings/", ".json")
	bytes, err := json.MarshalIndent(p.recording, "", " ")
	if err != nil {
		return 0, err
	}
	log.Println(string(bytes))
	return fileId, os.WriteFile(filename, bytes, 0b_110110110)
}

func newFilePath(prefix, postfix string) (string, int) {
	fileId := 1
	for {
		if _, err := os.Stat(prefix + strconv.Itoa(fileId) + postfix); os.IsNotExist(err) {
			return prefix + strconv.Itoa(fileId) + postfix, fileId
		}
		fileId += 1
	}
}

func loadRecording(id string) (*Recording, error) {
	filepath := "/recordings/" + id + ".json"
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	recording := Recording{Requests: []*Request{}}
	err = json.Unmarshal(bytes, &recording)
	if err != nil {
		return nil, err
	}
	return &recording, nil
}

func (p *Proxy) ResetReplayTimer() {
	if p.replayTimer == nil {
		return
	}
	p.replayTimer.Reset(time.Duration(3) * time.Second)
}

func (p *Proxy) nextOutsideRequest(alwaysSend bool) {
	for _, request := range p.replayingFrom.Requests {
		if !request.seen && request.FromOutside {
			request.seen = true
			send(request)
			r := *request // make a copy of request, to record it with new timestamp
			r.Timestamp = time.Now()
			p.record(&r)
		} else if !alwaysSend && !request.seen {
			return // exit because we need to see some other requests first
		}
	}
}

func send(r *Request) (*http.Response, error) {
	url, err := url.Parse(r.URI)
	if err != nil {
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
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.recording.Requests {
		if !r.seen {
			p.ResetReplayTimer()
			return
		}
	}
	p.writeRecording()
	p.isReplaying = false
	p.replayTimer = nil
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

func (p *Proxy) StartRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.isRecording = true
	p.recording = Recording{Requests: []*Request{}}
	p.replayingFrom = Recording{Requests: []*Request{}}
}

func (p *Proxy) EndRecordingHandler(w http.ResponseWriter, r *http.Request) {
	_, err := p.writeRecording()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	p.recording = Recording{Requests: []*Request{}}
}

func (p *Proxy) SaveVolumesHandler(w http.ResponseWriter, r *http.Request) {
	_, err := writeVolumes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (p *Proxy) LoadVolumesHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("id")
	if filename == "" {
		filename = latestVolumes()
	}
	err := loadVolumes(filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (p *Proxy) StartReplayHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	p.isRecording = false
	recording, err := loadRecording(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	p.replayingFrom = *recording
	p.recording = Recording{Requests: []*Request{}}
	err = loadVolumes(p.recording.Volumes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	p.isReplaying = true
	p.replayTimer = time.AfterFunc(time.Duration(3)*time.Second, p.EndReplay)
	p.nextOutsideRequest(true)
}

func (p *Proxy) CurrentRecordingHandler(w http.ResponseWriter, r *http.Request) {

}
