package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	recording bool
	filter    Filter
}

type Request struct {
	From       string   `json:"from"`
	FromNames  []string `json:"from_names"`
	To         string   `json:"to"`
	URI        string   `json:"uri"`
	BodyLength int      `json:"body_length"`
	TLS        bool     `json:"tls"`
	Blocked    bool     `json:"blocked"`
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	var proto string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	request := Request{
		From:       r.RemoteAddr,
		To:         r.URL.String(),
		BodyLength: len(body),
	}
	if r.TLS == nil {
		request.TLS = false
		proto = "http://"
	} else {
		request.TLS = true
		proto = "https://"
	}

	// perform reverse lookup
	// TODO: cache this in a map, never call in handler, because too slow
	ip, _, err := net.ParseCIDR(r.RemoteAddr)
	names, err := net.DefaultResolver.LookupAddr(context.Background(), ip.String())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	request.FromNames = names

	request.Blocked := p.filter.Block(&request)

	// proxy the request
	host, err := url.Parse(proto + r.URL.Host)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(host)
	proxy.ServeHTTP(w, r)
}

type Filter struct {
	recording  bool
	percentage int
}

func (f Filter) Block(request *Request) bool {
	if f.recording {
		return rand.Float32() > float32(f.percentage)/100
	}
	return false
}
