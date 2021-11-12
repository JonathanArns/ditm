package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TemplateData struct {
	ModeDefault      bool
	ModeRecording    bool
	ModeReplaying    bool
	ModeInspecting   bool
	Recordings       []int
	Volumes          []int
	Partitions       string
	Percentage       int
	BlockNone        bool
	BlockPartitions  bool
	BlockRandom      bool
	MatcherHeuristic bool
	MatcherExact     bool
	MatcherMix       bool
	MatcherCounting  bool
	MatcherTiming    bool
}

func (p *Proxy) NewTemplateData() TemplateData {
	p.mu.Lock()
	partitions, _ := json.Marshal(p.blockConfig.Partitions)
	ret := TemplateData{
		ModeDefault:      !p.isRecording && !p.isReplaying && !p.isInspecting,
		ModeRecording:    p.isRecording,
		ModeReplaying:    p.isReplaying,
		ModeInspecting:   p.isInspecting,
		Recordings:       ListRecordings(),
		Volumes:          ListVolumesSnapshots(),
		Partitions:       string(partitions),
		Percentage:       p.blockConfig.Percentage,
		BlockNone:        p.blockConfig.Mode == "none",
		BlockPartitions:  p.blockConfig.Mode == "partitions",
		BlockRandom:      p.blockConfig.Mode == "random",
		MatcherHeuristic: p.blockConfig.Matcher == "heuristic",
		MatcherExact:     p.blockConfig.Matcher == "exact",
		MatcherMix:       p.blockConfig.Matcher == "mix",
		MatcherCounting:  p.blockConfig.Matcher == "counting",
		MatcherTiming:    p.blockConfig.Matcher == "timing",
	}
	p.mu.Unlock()
	return ret
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
	p.recording = Recording{Requests: []*Request{}, StartTime: time.Now()}
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
	if p.isReplaying {
		p.replayTimer.Stop()
		p.replayTimer = nil
		p.isReplaying = false
		log.Println("replay canceled")
	}
	p.isRecording = false
	p.isInspecting = true
	p.recording = *recording
	p.mu.Unlock()

	t := template.New("main")
	t.Parse(mainTemplate)
	err = t.Execute(w, p.NewTemplateData())
	if err != nil {
		log.Println(err)
	}
}

func (p *Proxy) BlockConfigHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
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
	if matcher := r.FormValue("matcher"); matcher != "" {
		switch matcher {
		case "heuristic":
			p.matcher = &heuristicMatcher{map[*Request]struct{}{}}
			p.blockConfig.Matcher = matcher
		case "exact":
			p.matcher = &exactMatcher{map[*Request]struct{}{}}
			p.blockConfig.Matcher = matcher
		case "mix":
			p.matcher = &mixMatcher{map[*Request]struct{}{}}
			p.blockConfig.Matcher = matcher
		case "counting":
			p.matcher = &countingMatcher{map[*Request]struct{}{}}
			p.blockConfig.Matcher = matcher
		case "timing":
			p.matcher = &timingMatcher{map[*Request]struct{}{}}
			p.blockConfig.Matcher = matcher
		}
	}
	p.blockConfig.Timestamp = time.Now()
	p.recording.BlockConfigs = append(p.recording.BlockConfigs, p.blockConfig)
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	err := t.Execute(w, p.NewTemplateData())
	if err != nil {
		log.Println(err)
	}
}

func (p *Proxy) StartRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.isRecording = true
	p.isInspecting = false
	i, err := writeVolumes()
	if err != nil {
		log.Println(err)
	}
	p.recording = Recording{Requests: []*Request{}, StartTime: time.Now(), Volumes: strconv.Itoa(i)}
	p.replayingFrom = Recording{Requests: []*Request{}, StartTime: time.Now()}
	blockConf := p.blockConfig
	blockConf.Timestamp = time.Now()
	p.recording.BlockConfigs = append(p.recording.BlockConfigs, blockConf)
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) EndRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	_, err := p.writeRecording()
	if err != nil {
		log.Println(err)
	}
	p.recording = Recording{Requests: []*Request{}, StartTime: time.Now()}
	p.isRecording = false
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) SaveVolumesHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	_, err := writeVolumes()
	if err != nil {
		log.Println(err)
	}
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) LoadVolumesHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	filename := r.FormValue("id")
	if filename == "" {
		filename = latestVolumes()
	}
	err := loadVolumes(filename)
	if err != nil {
		log.Println(err)
	}
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
}

func (p *Proxy) StartReplayHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	id := r.FormValue("id")
	keepMode, _ := strconv.ParseBool(r.FormValue("keep_mode"))
	p.isRecording = false
	recording, err := loadRecording(id)
	if err != nil {
		log.Println(err)
	}
	p.replayingFrom = *recording
	p.recording = Recording{Requests: []*Request{}, StartTime: time.Now()}
	err = loadVolumes(p.replayingFrom.Volumes)
	if err != nil {
		log.Println(err)
	}
	p.isReplaying = true
	p.isInspecting = false
	p.blockConfig.previousMode = p.blockConfig.Mode
	if !keepMode {
		p.blockConfig.Mode = "replay"
	}
	p.replayTimer = time.AfterFunc(time.Duration(3)*time.Second, p.EndReplay)
	p.mu.Unlock()
	t := template.New("main")
	t.Parse(mainTemplate)
	t.Execute(w, p.NewTemplateData())
	time.Sleep(100 * time.Millisecond)
	p.nextOutsideRequest(true)
}

func (p *Proxy) LiveUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	hostNames := make([]string, len(p.hostNames))
	i := 0
	for k := range p.hostNames {
		hostNames[i] = p.hostNames[k]
		i++
	}
	t := template.New("recording").Funcs(template.FuncMap{
		"abbreviate": func(str string, i int) template.HTML {
			if len(str) <= i {
				return template.HTML(strings.TrimSpace(template.HTMLEscapeString(str)))
			}
			return template.HTML("<abbr title=\"" + strings.TrimSpace(template.HTMLEscapeString(str)) + "\">" + template.HTMLEscapeString(str[0:i]) + "</abbr>")
		},
		"string": func(data []byte) string {
			return string(data)
		},
		"colormap": func(str string) string {
			for i, name := range hostNames {
				if strings.Contains(str, name) {
					return []string{"#3582c4", "#b342f5", "#FFC300", "#FF5733", "#009A20"}[i%5]
				}
			}
			return "#111"
		},
	})
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
		case <-time.After(100 * time.Millisecond):
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
					p.mu.Unlock()
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
					p.mu.Unlock()
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

func (p *Proxy) DiffHandler(w http.ResponseWriter, r *http.Request) {
	a := r.FormValue("a")
	b := r.FormValue("b")
	ra, err := loadRecording(a)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	rb, err := loadRecording(b)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	diff := Diff(ra, rb)
	log.Println(diff)
	// TODO: render diff
}

func (p *Proxy) StatusHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	var status string
	if p.isReplaying {
		status = "replaying"
	} else if p.isInspecting {
		status = "inspecting"
	} else if p.isRecording {
		status = "recording"
	} else {
		status = "none"
	}
	p.mu.Unlock()
	w.Write([]byte(status))
}

func (p *Proxy) LatestRecordingHandler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	w.Write([]byte(strconv.Itoa(p.lastSavedId)))
	p.mu.Unlock()
}
