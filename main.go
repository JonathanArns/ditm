package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("starting ditm")
	proxy := InitProxy()
	srv := &http.Server{
		Handler: http.HandlerFunc(proxy.Handler),
		Addr:    ":5000",
	}
	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	r := mux.NewRouter()
	r.Path("/").HandlerFunc(proxy.HomeHandler)
	r.Path("/log").HandlerFunc(proxy.LogHandler)
	r.Path("/live_updates").HandlerFunc(proxy.LiveUpdatesHandler)
	r.Path("/start_recording").HandlerFunc(proxy.StartRecordingHandler)
	r.Path("/end_recording").HandlerFunc(proxy.EndRecordingHandler)
	r.Path("/load_recording").HandlerFunc(proxy.InspectHandler)
	r.Path("/start_replay").HandlerFunc(proxy.StartReplayHandler)
	r.Path("/save_volumes").HandlerFunc(proxy.SaveVolumesHandler)
	r.Path("/load_volumes").HandlerFunc(proxy.LoadVolumesHandler)
	r.Path("/block_config").HandlerFunc(proxy.BlockConfigHandler)
	r.Path("/api/status").HandlerFunc(proxy.StatusHandler)
	r.Path("/api/latest_recording").HandlerFunc(proxy.LatestRecordingHandler)

	apiSrv := &http.Server{
		Handler: r,
		Addr:    ":80",
	}
	log.Fatal(apiSrv.ListenAndServe())
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from ditm!"))
}

func InitProxy() *Proxy {
	m := &sync.Mutex{}
	proxy := &Proxy{
		mu:            m,
		hostNames:     map[string]string{},
		recording:     Recording{Requests: []*Request{}},
		replayingFrom: Recording{Requests: []*Request{}},
		blockConfig:   BlockConfig{Percentage: 50, Mode: "none", Matcher: "heuristic"},
		matcher:       &heuristicMatcher{map[*Request]struct{}{}},
		endReplayC:    make(chan struct{}, 1),
	}

	hostNames := strings.Split(os.Getenv("CONTAINER_HOST_NAMES"), ",")
	proxy.blockConfig.Partitions = [][]string{hostNames}
	for _, name := range hostNames {
		addrs, err := net.DefaultResolver.LookupHost(context.Background(), name)
		if err != nil {
			log.Println(err)
			panic("Unknown hostname: " + name)
		}
		log.Println("Added target with name:", name, ", addrs:", addrs)
		for _, addr := range addrs {
			proxy.hostNames[addr] = name
		}
	}
	return proxy
}
