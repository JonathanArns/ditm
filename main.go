package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	log.Println("starting ditm")
	proxy := InitProxy()
	helloSrv := &http.Server{
		Handler:      http.HandlerFunc(hello),
		Addr:         ":81",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	srv := &http.Server{
		Handler:      http.HandlerFunc(proxy.Handler),
		Addr:         ":80",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	go func() {
		log.Fatal(helloSrv.ListenAndServe())
	}()
	log.Fatal(srv.ListenAndServe())
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from ditm!"))
}

func InitProxy() *Proxy {
	proxy := &Proxy{
		hostNames: map[string]string{},
		recording: Recording{},
		replay:    Recording{},
	}

	blockPercentage, err := strconv.Atoi(os.Getenv("BLOCK_PERCENTAGE"))
	if err != nil {
		panic(err)
	}
	proxy.blockPercentage = blockPercentage

	hostNames := strings.Split(os.Getenv("CONTAINER_HOST_NAMES"), ",")
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
