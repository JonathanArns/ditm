package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

var peer string
var sendTimestamp bool
var post bool
var async bool

var client *http.Client = http.DefaultClient

var mu sync.Mutex

func main() {
	peer = os.Getenv("PEER")
	if sts := os.Getenv("SEND_TIMESTAMP"); sts == "true" {
		sendTimestamp = true
	}
	if sb := os.Getenv("POST"); sb == "true" {
		post = true
	}
	if asy := os.Getenv("ASYNC"); asy == "true" {
		async = true
	}
	if disableKeepAlive := os.Getenv("DISABLE_KEEP_ALIVE"); disableKeepAlive == "true" {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.DisableKeepAlives = true
		client = &http.Client{Transport: t}
	}
	router := http.NewServeMux()
	router.HandleFunc("/", home)
	router.HandleFunc("/recurse", callRecurse)
	router.HandleFunc("/loop", loop)
	srv := &http.Server{
		Handler: router,
		Addr:    ":80",
	}
	log.Fatal(srv.ListenAndServe())
}

func home(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("v")
	fmt.Println(v)
	w.Write([]byte(v))
}

func loop(w http.ResponseWriter, r *http.Request) {
	count, _ := strconv.Atoi(r.FormValue("count"))
	targetUrl := "http://" + peer
	send := func(targetUrl string, i int) {
		var err error
		var res *http.Response
		uri := targetUrl + "?v=" + strconv.Itoa(i)
		if sendTimestamp {
			uri += "&ts=" + url.QueryEscape(time.Now().Format(time.StampNano))
		}
		if post {
			filler := []string{"abcdef", "abcdefghijklm", "abcdefghijklmopqrstuvw"}[(i)%3]
			res, err = client.PostForm(targetUrl, map[string][]string{"v": {strconv.Itoa(i)}, "filler": {filler}, "ts": {time.Now().Format(time.StampNano)}})
		} else {
			res, err = client.Get(uri)
		}
		if err == nil {
			data, err := io.ReadAll(res.Body)
			if err != nil {
				log.Println(err)
			}
			fmt.Println(string(data))
		} else {
			log.Println(err)
		}
	}
	for i := 0; i < count; i++ {
		if async {
			go send(targetUrl, i)
		} else {
			send(targetUrl, i)
		}
	}
}

func callRecurse(w http.ResponseWriter, r *http.Request) {
	maxDepth, _ := strconv.Atoi(r.FormValue("depth"))
	go recurse(maxDepth, 0, 0)
}

func recurse(maxDepth, depth, id int) {
	var err error
	var res *http.Response
	v := fmt.Sprintf("%v-%v", depth, id)
	log.Println(v)
	targetUrl := "http://" + peer
	uri := targetUrl + "?v=" + v
	if sendTimestamp {
		uri += "&ts=" + url.QueryEscape(time.Now().Format(time.StampNano))
	}
	if post {
		filler := []string{"abcdef", "abcdefghijklm", "abcdefghijklmopqrstuvw"}[(depth+id)%3]
		res, err = http.PostForm(targetUrl, map[string][]string{"v": {v}, "filler": {filler}})
	} else {
		res, err = http.Get(uri)
	}
	if err == nil {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(string(data))
	} else {
		log.Println(err)
	}
	if depth < maxDepth {
		// time.Sleep(1 * time.Millisecond)
		if async {
			go recurse(maxDepth, depth+1, id|1<<depth)
			go recurse(maxDepth, depth+1, id)
		} else {
			recurse(maxDepth, depth+1, id|1<<depth)
			recurse(maxDepth, depth+1, id)
		}
	}
}
