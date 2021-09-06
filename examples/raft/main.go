package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
)

func main() {
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
	server.start(peers, forceLeader)
}
