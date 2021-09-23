package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var peer string

func main() {
	peer = os.Getenv("PEER")
	router := http.NewServeMux()
	router.HandleFunc("/", home)
	router.HandleFunc("/loop", loop)
	srv := &http.Server{
		Handler: router,
		Addr:    ":80",
	}
	log.Fatal(srv.ListenAndServe())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func home(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("v")
	fmt.Println(v)
	w.Write([]byte(v))
}

func loop(w http.ResponseWriter, r *http.Request) {
	count, _ := strconv.Atoi(r.FormValue("count"))
	max_shift, _ := strconv.Atoi(r.FormValue("max_shift"))
	sendTimestamp, _ := strconv.ParseBool(r.FormValue("send_timestamp"))
	async, _ := strconv.ParseBool(r.FormValue("async"))
	post, _ := strconv.ParseBool(r.FormValue("post"))
	client := http.DefaultClient
	if disableKeepAlive, _ := strconv.ParseBool(r.FormValue("disable_keep_alive")); disableKeepAlive {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.DisableKeepAlives = true
		client = &http.Client{Transport: t}
	}

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
	tmp := make([]int, count)
	list := make([]int, count)
	for i := 0; i < count; i++ {
		tmp[i] = i
	}
	n := max_shift + 1
	for i := 0; i < count; i++ {
		randIndex := rand.Intn(min(len(tmp), n))
		list[i] = tmp[randIndex]
		tmp = append(tmp[:randIndex], tmp[1+randIndex:]...)
		if randIndex > 0 || n <= max_shift {
			n -= 1
		}
		if n == 0 {
			n = max_shift + 1
		}
	}
	log.Println(list)
	for _, i := range list {
		if async {
			go send(targetUrl, i)
		} else {
			send(targetUrl, i)
		}
	}
}
