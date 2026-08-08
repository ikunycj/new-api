package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type alertStore struct {
	mu      sync.RWMutex
	payload json.RawMessage
}

func main() {
	store := &alertStore{payload: json.RawMessage(`{"status":"waiting","alerts":[]}`)}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var payload json.RawMessage
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
			if err := decoder.Decode(&payload); err != nil {
				http.Error(w, "invalid alert payload", http.StatusBadRequest)
				return
			}
			store.mu.Lock()
			store.payload = append(store.payload[:0], payload...)
			store.mu.Unlock()
			formatted, _ := json.MarshalIndent(payload, "", "  ")
			log.Printf("ALERTMANAGER NOTIFICATION:\n%s", formatted)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		store.mu.RLock()
		defer store.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(store.payload)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("local alert sink listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
