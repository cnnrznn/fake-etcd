package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cnnrznn/fake-etcd/store"
	"github.com/cnnrznn/raft"
)

type config struct {
	Peers []string `json:"peers"`
	APIs  []string `json:"apis"`
}

type Server struct {
	raft     *raft.Raft
	config   *config
	data     *store.Store
	lastSeen int
}

func main() {
	// Read peer file and my address
	config := readConfig()
	args := os.Args
	id, _ := strconv.Atoi(args[1])

	// Make raft instance
	raft := raft.New(id, config.Peers)

	// Run raft
	go raft.Run()

	server := Server{
		raft:     raft,
		config:   config,
		data:     store.New(),
		lastSeen: 0,
	}

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			server.processUpdates()
		}
	}()

	go func() {
		for {
			time.Sleep(5000 * time.Millisecond)
			fmt.Println(server.data)
		}
	}()

	http.HandleFunc("/", server.handleHTTP)
	log.Fatal(http.ListenAndServe(config.APIs[id], nil))
}

func readConfig() *config {
	bytes, err := os.ReadFile("peers.json")
	if err != nil {
		fmt.Println("error reading peer file")
		return nil
	}

	var result config
	json.Unmarshal(bytes, &result)

	return &result
}

func (s *Server) processUpdates() {
	entries := s.raft.Retrieve(s.lastSeen)
	s.lastSeen += len(entries)

	for _, e := range entries {
		sp := strings.Split(e.Msg, ":")
		key, value := sp[0], sp[1]
		s.data.Set(key, value)
	}
}
