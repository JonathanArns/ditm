package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/", handle)
	router.HandleFunc("/hello", proxy)

	srv := &http.Server{
		Handler:      router,
		Addr:         ":80",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func proxy(w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse("http://fuzznet:81/")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ServeHTTP(w, r)
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
