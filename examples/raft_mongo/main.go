package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))
	sid := os.Getenv("RAFT_ID")
	id, err := strconv.Atoi(sid)
	if err != nil {
		log.Fatal(err)
	}
	var forceLeader bool
	sForceLeader := os.Getenv("RAFT_FORCE_LEADER")
	if sForceLeader == "true" {
		forceLeader = true
	}
	peers := map[int]string{}
	err = json.Unmarshal([]byte(os.Getenv("RAFT_PEERS")), &peers)

	server := newServer(id)
	time.Sleep(1 * time.Second)
	server.start(peers, forceLeader)
}
