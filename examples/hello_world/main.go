package main

import (
	"crypto/tls"
	"log"
	"net/http"
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
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	res, err := http.Get("https://ditm:81/hello")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println(res.Status)
	x := []byte{}
	i, err := res.Body.Read(x)
	log.Println(i, err, string(x))
	w.Write(x)
}
