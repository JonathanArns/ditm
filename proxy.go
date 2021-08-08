package main

import (
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
	"strings"
	"time"

	"github.com/mholt/archiver/v3"
)

type Proxy struct {
	hostNames       map[string]string
	isRecording     bool
	isReplaying     bool
	blockPercentage int
	recording       Recording
	replay          Recording
}

type Recording struct {
	requests map[string][]Request
	snapshot string
}

type Request struct {
	From             string `json:"from"`
	FromName         string `json:"from_name"`
	To               string `json:"to"`
	StreamIdentifier string `json:"stream_identifier"`
	URI              string `json:"uri"`
	BodyLength       int    `json:"body_length"`
	TLS              bool   `json:"tls"`
	Blocked          bool   `json:"blocked"`
	seen             bool   `json:"-"`
}

func (p *Proxy) Block(request *Request) bool {
	if p.isRecording {
		return rand.Float32() < float32(p.blockPercentage)/100
	} else if !p.isReplaying {
		return false
	}

	recording, ok := p.recording.requests[request.StreamIdentifier]
	if !ok {
		log.Println("FUCK WE DON'T KNOW THIS REQUEST")
	}
	replay, ok := p.replay.requests[request.StreamIdentifier]
	if !ok {
		log.Println("FUCK WE DON'T KNOW THIS REQUEST")
	}

	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(recording)) // a faktor to relativize constant score components
	for i, r := range recording {
		if r.seen {
			continue
		}
		score := 0.0
		score -= math.Abs(float64(i - len(replay)))
		if r.URI == request.URI {
			score += 1 * faktor
		}
		if score > highScore {
			highScore = score
			bestMatch = &r
		}
	}
	if bestMatch != nil {
		bestMatch.seen = true
		return bestMatch.Blocked
	}
	log.Println("WE ARE SEEING MORE REQUESTS THAN IN THE RECORDING")
	return false
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	var proto string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	request := Request{
		From:       r.RemoteAddr,
		To:         r.URL.String(),
		BodyLength: len(body),
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
	request.FromName = p.hostNames[ip]
	request.StreamIdentifier = request.FromName + "->" + request.To

	request.Blocked = p.Block(&request)
	if request.Blocked {
		panic("We want to block this request")
	}

	log.Printf("%+v", request)

	// proxy the request
	remoteHost, err := url.Parse(proto + r.URL.Host)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(remoteHost)
	proxy.ServeHTTP(w, r)
}

func (p *Proxy) writeRecording(recording Recording) error {
	filename := "/recordings/" + time.Now().Format(time.StampMicro) + ".json"
	bytes, err := json.MarshalIndent(recording, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, bytes, 666)
}

func (p *Proxy) loadRecording(filename string) error {
	filepath := "/recordings/" + filename
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	recording := Recording{}
	err = json.Unmarshal(bytes, &recording)
	if err != nil {
		return err
	}
	p.recording = recording
	return nil
}

func writeVolumes() error {
	filename := "/snapshots/" + time.Now().Format(time.StampMicro) + ".zip"
	err := archiver.Archive([]string{"/volumes"}, filename)
	return err
}

func loadVolumes(file string) error {
	if file == "" {
		return errors.New("No Volumes Snapshot")
	}
	filepath := "/volumes/" + file
	err := archiver.Unarchive(filepath, "/volumes")
	return err
}

func latestVolumes() string {
	files, _ := os.ReadDir("/snapshots")
	latestTime := time.Time{}
	latestName := ""
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if stamp, err := time.Parse(time.StampMicro, strings.Trim(name, ".zip")); err == nil {
			if stamp.After(latestTime) {
				latestName = name
				latestTime = stamp
			}
		}
	}
	return latestName
}

func (p *Proxy) StartRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.isRecording = true
	p.recording = Recording{}
	p.replay = Recording{}
}

func (p *Proxy) EndRecordingHandler(w http.ResponseWriter, r *http.Request) {
	err := p.writeRecording(p.recording)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (p *Proxy) SaveVolumesHandler(w http.ResponseWriter, r *http.Request) {
	err := writeVolumes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (p *Proxy) LoadVolumesHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("filename")
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
	filename := r.FormValue("filename")
	p.isRecording = false
	err := p.loadRecording(filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	loadVolumes(p.recording.snapshot)
	p.isReplaying = true
}

func (p *Proxy) CurrentRecordingHandler(w http.ResponseWriter, r *http.Request) {

}
