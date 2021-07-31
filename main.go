package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	log.Println("starting fuzznet")
	proxy := Proxy{
		recording: true,
		filter:    Filter{},
	}

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
	w.Write([]byte("Hello from Fuzznet!"))
}
