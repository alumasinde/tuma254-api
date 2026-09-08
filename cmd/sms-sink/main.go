package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type message struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

type store struct {
	mu     sync.RWMutex
	latest message
	ok     bool
}

func main() {
	secret := strings.TrimSpace(os.Getenv("SMS_SINK_TOKEN"))
	if secret == "" {
		log.Fatal("SMS_SINK_TOKEN is required")
	}
	s := &store{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		defer r.Body.Close()
		var m message
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&m); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.latest = m
		s.ok = true
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /messages/latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		if !s.ok {
			http.Error(w, "no message", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(s.latest)
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:8090", mux))
}
