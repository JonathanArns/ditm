package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
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
	r.Path("/start_replay").HandlerFunc(proxy.StartReplayHandler)
	r.Path("/save_volumes").HandlerFunc(proxy.SaveVolumesHandler)
	r.Path("/load_volumes").HandlerFunc(proxy.LoadVolumesHandler)
	r.Path("/block_config").HandlerFunc(proxy.BlockConfigHandler)
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
	proxy := &Proxy{
		mu:            sync.Mutex{},
		hostNames:     map[string]string{},
		recording:     Recording{Requests: []*Request{}},
		replayingFrom: Recording{Requests: []*Request{}},
		blockConfig:   BlockConfig{},
	}

	blockPercentage, err := strconv.Atoi(os.Getenv("BLOCK_PERCENTAGE"))
	if err != nil {
		panic(err)
	}
	proxy.blockConfig.Percentage = blockPercentage

	hostNames := strings.Split(os.Getenv("CONTAINER_HOST_NAMES"), ",")
	proxy.blockConfig.Partitions = [][]string{hostNames}
	for _, name := range hostNames {
		addrs, err := net.DefaultResolver.LookupHost(context.Background(), name)
		if err != nil {
			log.Println(err)
			panic("Unknown hostname: " + name)
		}
		log.Println("NAME:", name, ", addrs:", addrs)
		for _, addr := range addrs {
			proxy.hostNames[addr] = name
		}
	}

	return proxy
}
