package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var post bool
var counter int
var mu sync.Mutex

// this small code sample shows a possible bug discovered by ditm
func main() {
	if sb := os.Getenv("POST"); sb == "true" {
		post = true
	}
	router := http.NewServeMux()
	router.HandleFunc("/", handle)
	go func() {
		log.Fatal(http.ListenAndServe(":8000", router))
	}()
	for i := 0; i < 3; i++ {
		if post {
			res, err := http.PostForm("http://localhost:8000", url.Values{})
			if err == nil {
				// we don't need to close the body for this example to work,
				// we are just being good citizens
				res.Body.Close()
			}
		} else {
			res, err := http.Get("http://localhost:8000")
			if err == nil {
				res.Body.Close()
			}
		}
	}
	time.Sleep(10 * time.Millisecond)
}

func handle(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	counter += 1
	log.Println("HELLO WORLD!")
	if counter%2 == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	panic("closing connection")
}
