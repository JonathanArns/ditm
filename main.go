package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func main() {
	log.Println("starting fuzznet")

	helloSrv := &http.Server{
		Handler:      http.HandlerFunc(hello),
		Addr:         ":81",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	srv := &http.Server{
		Handler:      http.HandlerFunc(proxyHandler),
		Addr:         ":80",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	go func() {
		log.Fatal(helloSrv.ListenAndServe())
	}()
	log.Fatal(srv.ListenAndServe())
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Proxy: From: " + r.RemoteAddr)
	log.Println("Proxy: To: " + r.URL.String())
	host, err := url.Parse("http://" + r.URL.Host + "/")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Proxying to: " + host.String())
	httputil.NewSingleHostReverseProxy(host).ServeHTTP(w, r)
	// res, err := http.Get("http://fuzznet/hello")
	// if err != nil {
	// 	log.Println(err)
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	// x := []byte{}
	// res.Body.Read(x)
	// w.Write(x)
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from Fuzznet!"))
}
