package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type config struct {
	ttft           time.Duration
	response       time.Duration
	streamChunks   int
	streamInterval time.Duration
	errorRate      float64
}

type request struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

type channelState struct {
	sync.Mutex
	id        int
	name      string
	disabled  bool
	remaining int64
	consumed  uint64
}

var (
	activeRequests atomic.Int64
	requests       atomic.Uint64
	errorsTotal    atomic.Uint64
	durationNanos  atomic.Uint64
)

func main() {
	cfg := config{
		ttft:           durationFromMillis("TTFT_MS", 100),
		response:       durationFromMillis("RESPONSE_MS", 500),
		streamChunks:   intFromEnv("STREAM_CHUNKS", 20),
		streamInterval: durationFromMillis("STREAM_INTERVAL_MS", 50),
		errorRate:      floatFromEnv("ERROR_RATE", 0),
	}
	state := &channelState{
		id:        intFromEnv("CHANNEL_ID", 1),
		name:      envOrDefault("CHANNEL_NAME", "mock-channel-a"),
		remaining: int64FromEnv("CHANNEL_TOKENS", 300),
	}
	if cfg.streamChunks < 1 {
		cfg.streamChunks = 1
	}
	if cfg.errorRate < 0 || cfg.errorRate > 1 {
		log.Fatal("ERROR_RATE must be between 0 and 1")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics(w, r)
		state.writeMetrics(w)
	})
	mux.HandleFunc("/control/state", state.handleState)
	mux.HandleFunc("/control/reset", state.handleReset)
	mux.HandleFunc("/control/exhaust", state.handleExhaust)
	mux.HandleFunc("/control/disable", state.handleDisable)
	mux.HandleFunc("/control/enable", state.handleEnable)
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg, state)
	})
	mux.HandleFunc("/v1/responses", func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg, state)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("mock channel %s/CH%d listening on %s (ttft=%s response=%s chunks=%d interval=%s error_rate=%.3f)", state.name, state.id, server.Addr, cfg.ttft, cfg.response, cfg.streamChunks, cfg.streamInterval, cfg.errorRate)
	log.Fatal(server.ListenAndServe())
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg config, state *channelState) {
	startedAt := time.Now()
	requests.Add(1)
	activeRequests.Add(1)
	defer func() {
		activeRequests.Add(-1)
		durationNanos.Add(uint64(time.Since(startedAt)))
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		errorsTotal.Add(1)
		http.Error(w, `{"error":{"message":"invalid request"}}`, http.StatusBadRequest)
		return
	}
	if cfg.errorRate > 0 && rand.Float64() < cfg.errorRate {
		errorsTotal.Add(1)
		time.Sleep(cfg.ttft)
		writeChannelError(w, state, "injected load-test failure", "mock_error", http.StatusServiceUnavailable)
		return
	}
	if !state.consume(30) {
		errorsTotal.Add(1)
		writeChannelError(w, state, "mock channel is exhausted", "channel_exhausted", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Mock-Channel", strconv.Itoa(state.id))
	if input.Stream {
		streamResponse(w, cfg)
		return
	}
	time.Sleep(cfg.response)
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"id":"chatcmpl-loadtest","object":"chat.completion","created":%d,"model":"gpt-3.5-turbo","choices":[{"index":0,"message":{"role":"assistant","content":"deterministic load-test response"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`, time.Now().Unix())
}

func writeChannelError(w http.ResponseWriter, state *channelState, message, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Mock-Channel", strconv.Itoa(state.id))
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error":{"message":%q,"type":"channel_error","code":%q,"source":"channel","failure_scope":"channel"}}`, message, code)
}

func (s *channelState) consume(tokens int64) bool {
	s.Lock()
	defer s.Unlock()
	if s.disabled || s.remaining < tokens {
		return false
	}
	s.remaining -= tokens
	s.consumed += uint64(tokens)
	return true
}

func (s *channelState) handleState(w http.ResponseWriter, _ *http.Request) {
	s.Lock()
	defer s.Unlock()
	writeJSON(w, map[string]any{
		"channel_id": s.id, "channel_name": s.name, "disabled": s.disabled,
		"remaining_tokens": s.remaining, "consumed_tokens": s.consumed,
	})
}

func (s *channelState) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.Lock()
	s.disabled = false
	s.remaining = queryInt64(r, "tokens", int64FromEnv("CHANNEL_TOKENS", 300))
	s.consumed = 0
	s.Unlock()
	s.handleState(w, r)
}

func (s *channelState) handleExhaust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.Lock()
	s.remaining = 0
	s.Unlock()
	s.handleState(w, r)
}

func (s *channelState) handleDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.Lock()
	s.disabled = true
	s.Unlock()
	s.handleState(w, r)
}

func (s *channelState) handleEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.Lock()
	s.disabled = false
	s.Unlock()
	s.handleState(w, r)
}

func streamResponse(w http.ResponseWriter, cfg config) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	time.Sleep(cfg.ttft)
	for index := 0; index < cfg.streamChunks; index++ {
		_, _ = fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-loadtest\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"token-%d \"},\"finish_reason\":null}]}\n\n", time.Now().Unix(), index)
		flusher.Flush()
		time.Sleep(cfg.streamInterval)
	}
	_, _ = fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-loadtest\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":20,\"total_tokens\":30}}\n\n", time.Now().Unix())
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	requestCount := requests.Load()
	durationSeconds := float64(durationNanos.Load()) / float64(time.Second)
	_, _ = fmt.Fprintf(w, "# HELP mock_openai_requests_active Current requests handled by the mock upstream.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mock_openai_requests_active gauge\nmock_openai_requests_active %d\n", activeRequests.Load())
	_, _ = fmt.Fprintf(w, "# HELP mock_openai_requests_total Requests handled by the mock upstream.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mock_openai_requests_total counter\nmock_openai_requests_total %d\n", requestCount)
	_, _ = fmt.Fprintf(w, "# HELP mock_openai_errors_total Injected or validation errors.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mock_openai_errors_total counter\nmock_openai_errors_total %d\n", errorsTotal.Load())
	_, _ = fmt.Fprintf(w, "# HELP mock_openai_request_duration_seconds Total request handling duration.\n")
	_, _ = fmt.Fprintf(w, "# TYPE mock_openai_request_duration_seconds summary\nmock_openai_request_duration_seconds_sum %.6f\nmock_openai_request_duration_seconds_count %d\n", durationSeconds, requestCount)
}

func (s *channelState) writeMetrics(w http.ResponseWriter) {
	s.Lock()
	defer s.Unlock()
	_, _ = fmt.Fprintf(w, "mock_channel_remaining_tokens{channel_id=\"%d\",channel_name=\"%s\"} %d\n", s.id, s.name, s.remaining)
	_, _ = fmt.Fprintf(w, "mock_channel_consumed_tokens_total{channel_id=\"%d\",channel_name=\"%s\"} %d\n", s.id, s.name, s.consumed)
	disabled := 0
	if s.disabled {
		disabled = 1
	}
	_, _ = fmt.Fprintf(w, "mock_channel_disabled{channel_id=\"%d\",channel_name=\"%s\"} %d\n", s.id, s.name, disabled)
}

func durationFromMillis(name string, fallback int) time.Duration {
	return time.Duration(intFromEnv(name, fallback)) * time.Millisecond
}

func intFromEnv(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("%s must be an integer: %v", name, err)
	}
	return parsed
}

func floatFromEnv(name string, fallback float64) float64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Fatalf("%s must be a number: %v", name, err)
	}
	return parsed
}

func int64FromEnv(name string, fallback int64) int64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		log.Fatalf("%s must be a non-negative integer", name)
	}
	return parsed
}

func envOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func queryInt64(r *http.Request, name string, fallback int64) int64 {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
