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
	config := readPeers()
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
			time.Sleep(5000 * time.Millisecond)
			server.processUpdates()
			fmt.Println(server.data)
		}
	}()

	http.HandleFunc("/", server.handleHTTP)

	log.Fatal(http.ListenAndServe(config.APIs[id], nil))
}

func readPeers() *config {
	bytes, err := os.ReadFile("peers.json")
	if err != nil {
		fmt.Println("error reading peer file")
		return nil
	}

	var result config
	json.Unmarshal(bytes, &result)

	return &result
}

type Request struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Response struct {
	Leader string `json:"leader"`
	Request
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.handleSet(w, r)
	case http.MethodGet:
		s.handleGet(w, r)
	default:
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Problem parsing", http.StatusInternalServerError)
		return
	}

	// Submit to raft
	entry := fmt.Sprintf("%v:%v", req.Key, req.Value)
	result := s.raft.Submit(entry)
	resp := Response{
		Leader: s.config.APIs[result.Leader],
	}
	if result.Success {
		resp.Key = req.Key
		resp.Value = req.Value
	}

	// Write response
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	s.processUpdates()

	key := r.URL.Query()["key"][0]
	val := s.data.Get(key)

	resp := Response{
		Leader: s.config.APIs[s.raft.Leader()],
		Request: Request{
			Key:   key,
			Value: val,
		},
	}

	// Write response
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
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
