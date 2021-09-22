package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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
