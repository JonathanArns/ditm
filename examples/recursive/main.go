package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/", home)
	router.HandleFunc("/recurse", callRecurse)
	srv := &http.Server{
		Handler: router,
		Addr:    ":80",
	}
	log.Fatal(srv.ListenAndServe())
}

func home(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("v")
	vs := strings.Split(v, "-")
	v1, _ := strconv.Atoi(vs[0])
	v2, _ := strconv.Atoi(vs[1])
	w.Write([]byte(strconv.Itoa(v1 + v2)))
}

func callRecurse(w http.ResponseWriter, r *http.Request) {
	c := make(chan int)
	go recurse(0, 0, c)
	i := 0
loop:
	for {
		select {
		case x := <-c:
			i += x
		case <-time.After(100 * time.Millisecond):
			break loop
		}
	}
	w.Write([]byte(strconv.Itoa(i)))
}

func recurse(depth, id int, c chan int) {
	if depth < 5 {
		go recurse(depth+1, id|1<<depth, c)
		go recurse(depth+1, id, c)
	}

	v := fmt.Sprintf("%v-%v", depth, id)
	log.Println(v)
	if res, err := http.Get("http://target2:80?v=" + v); err == nil {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
		}
		i, err := strconv.Atoi(string(data))
		if err != nil {
			log.Println(err)
		}
		c <- i
	} else {
		log.Println(err)
	}
}
