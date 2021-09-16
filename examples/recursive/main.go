package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var sum1 int
var sum2 int

var peer string
var sendTimestamp bool
var sendBody bool

var receive chan struct{}
var mu sync.Mutex

func main() {
	peer = os.Getenv("PEER")
	if sts := os.Getenv("SEND_TIMESTAMP"); sts == "true" {
		sendTimestamp = true
	}
	if sts := os.Getenv("SEND_BODY"); sts == "true" {
		sendBody = true
	}
	receive = make(chan struct{}, 100)
	router := http.NewServeMux()
	router.HandleFunc("/", home)
	router.HandleFunc("/recurse", callRecurse)
	srv := &http.Server{
		Handler: router,
		Addr:    ":80",
	}
	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				sum1 = 0
				sum2 = 0
			case <-receive:
				break
			}
		}
	}()
	log.Fatal(srv.ListenAndServe())
}

func home(w http.ResponseWriter, r *http.Request) {
	receive <- struct{}{}
	v := r.FormValue("v")
	vs := strings.Split(v, "-")
	v1, _ := strconv.Atoi(vs[0])
	v2, _ := strconv.Atoi(vs[1])
	mu.Lock()
	sum1 += v1
	sum2 += v2
	mu.Unlock()
	fmt.Printf("%v-%v\n", sum1, sum2)
	w.Write([]byte(v))
}

func callRecurse(w http.ResponseWriter, r *http.Request) {
	maxDepth, _ := strconv.Atoi(r.FormValue("depth"))
	c := make(chan string)
	go recurse(maxDepth, 0, 0, c)
	s1 := 0
	s2 := 0
loop:
	for {
		select {
		case v := <-c:
			vs := strings.Split(v, "-")
			v1, _ := strconv.Atoi(vs[0])
			v2, _ := strconv.Atoi(vs[1])
			s1 += v1
			s2 += v2
		case <-time.After(100 * time.Millisecond):
			break loop
		}
	}
	fmt.Fprintf(w, "%v-%v\n", s1, s2)
}

func recurse(maxDepth, depth, id int, c chan string) {
	var err error
	var res *http.Response
	if depth < maxDepth {
		go recurse(maxDepth, depth+1, id|1<<depth, c)
		go recurse(maxDepth, depth+1, id, c)
	}

	v := fmt.Sprintf("%v-%v", depth, id)
	log.Println(v)
	url := "http://" + peer + ":80"
	uri := url + "?v=" + v
	if sendTimestamp {
		uri += "&ts=" + time.Now().Format(time.StampNano)
	}
	if sendBody {
		filler := []string{"abcdef", "abcdefghijklm", "abcdefghijklmopqrstuvw"}[(depth+id)%3]
		res, err = http.PostForm(url, map[string][]string{"v": {v}, "filler": {filler}})
	} else {
		res, err = http.Get(uri)
	}
	if err == nil {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
		}
		c <- string(data)
	} else {
		log.Println(err)
	}
}
