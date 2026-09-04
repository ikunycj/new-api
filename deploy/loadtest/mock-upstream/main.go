package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
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
	ttft            time.Duration
	response        time.Duration
	streamChunks    int
	streamInterval  time.Duration
	errorRate       float64
	consumeTokens   int64
	usage           usageConfig
	failureStatuses []int
	failureStatus   []failureDistribution
}

type usageConfig struct {
	inputTokens  int64 `json:"input_tokens"`
	outputTokens int64 `json:"output_tokens"`
	totalTokens  int64 `json:"total_tokens"`
}

type failureDistribution struct {
	Status int     `json:"status"`
	Rate   float64 `json:"rate"`
}

type loadTestFileConfig struct {
	Mock struct {
		ResponseUsage    usageConfig           `json:"response_usage"`
		ConsumeTokens    int64                 `json:"consume_tokens"`
		StreamChunks     int                   `json:"stream_chunks"`
		StreamIntervalMS int                   `json:"stream_interval_ms"`
		FailureStatuses  []int                 `json:"failure_statuses"`
		Channels         []loadTestFileChannel `json:"channels"`
	} `json:"mock"`
}

type loadTestFileChannel struct {
	ID                  int                   `json:"id"`
	Name                string                `json:"name"`
	Tokens              int64                 `json:"tokens"`
	TTFTMS              int                   `json:"ttft_ms"`
	ResponseMS          int                   `json:"response_ms"`
	ErrorRate           float64               `json:"error_rate"`
	FailureDistribution []failureDistribution `json:"failure_distribution"`
}

type requestConfig struct {
	errorRate     float64
	errorStatus   int
	latency       time.Duration
	failureStatus []failureDistribution
}

type mockChannelConfig struct {
	Slot          int     `json:"slot"`
	FailureRate   float64 `json:"failure_rate"`
	FailureStatus int     `json:"failure_status"`
	LatencyMS     int     `json:"latency_ms"`
}

const (
	mockFailureRateHeader   = "X-Alltoken-Mock-Failure-Rate"
	mockFailureStatusHeader = "X-Alltoken-Mock-Failure-Status"
	mockLatencyMSHeader     = "X-Alltoken-Mock-Latency-Ms"
	mockChannelsHeader      = "X-Alltoken-Mock-Channels"
	maxMockLatencyMS        = 120000
	maxMockChannelsBytes    = 4096
)

var mockFailureStatuses = [...]int{
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
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
		consumeTokens:  30,
		usage:          usageConfig{inputTokens: 10, outputTokens: 20, totalTokens: 30},
	}
	state := &channelState{
		id:        intFromEnv("CHANNEL_ID", 1),
		name:      envOrDefault("CHANNEL_NAME", "mock-channel-a"),
		remaining: int64FromEnv("CHANNEL_TOKENS", 300),
	}
	if err := applyLoadTestFileConfig(&cfg, state, os.Getenv("LOADTEST_CONFIG_FILE")); err != nil {
		log.Fatal(err)
	}
	applyMockEnvironmentOverrides(&cfg, state)
	if len(cfg.failureStatuses) == 0 {
		cfg.failureStatuses = append([]int(nil), mockFailureStatuses[:]...)
	}
	if err := validateLoadTestConfig(cfg); err != nil {
		log.Fatal(err)
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
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
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
	cfg = normalizeConfig(cfg)
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
	requestCfg, err := loadRequestConfig(r.Header, cfg, state.id)
	if err != nil {
		errorsTotal.Add(1)
		writeChannelError(w, state, err.Error(), "invalid_mock_config", http.StatusBadRequest)
		return
	}
	var input request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		errorsTotal.Add(1)
		http.Error(w, `{"error":{"message":"invalid request"}}`, http.StatusBadRequest)
		return
	}
	if requestCfg.latency > 0 {
		time.Sleep(requestCfg.latency)
	}
	if shouldInjectFailure(requestCfg.errorRate, rand.Float64()) {
		errorsTotal.Add(1)
		time.Sleep(cfg.ttft)
		failureStatus := requestCfg.errorStatus
		if failureStatus == 0 {
			failureStatus = resolveDistributedFailureStatus(requestCfg.failureStatus, cfg.failureStatuses, rand.Float64())
		}
		writeChannelError(w, state, "injected load-test failure", "mock_error", failureStatus)
		return
	}
	if !state.consume(cfg.consumeTokens) {
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
	switch r.URL.Path {
	case "/v1/messages":
		_, _ = fmt.Fprintf(w, `{"id":"msg_loadtest","type":"message","role":"assistant","model":%q,"content":[{"type":"text","text":"deterministic load-test response"}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":%d,"output_tokens":%d}}`, input.Model, cfg.usage.inputTokens, cfg.usage.outputTokens)
	case "/v1/responses":
		_, _ = fmt.Fprintf(w, `{"id":"resp_loadtest","object":"response","created_at":%d,"status":"completed","model":%q,"output":[{"id":"msg_loadtest","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"deterministic load-test response","annotations":[]}]}],"usage":{"input_tokens":%d,"output_tokens":%d,"total_tokens":%d}}`, time.Now().Unix(), input.Model, cfg.usage.inputTokens, cfg.usage.outputTokens, cfg.usage.totalTokens)
	default:
		_, _ = fmt.Fprintf(w, `{"id":"chatcmpl-loadtest","object":"chat.completion","created":%d,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":"deterministic load-test response"},"finish_reason":"stop"}],"usage":{"prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d}}`, time.Now().Unix(), input.Model, cfg.usage.inputTokens, cfg.usage.outputTokens, cfg.usage.totalTokens)
	}
}

func normalizeConfig(cfg config) config {
	if cfg.consumeTokens < 1 {
		cfg.consumeTokens = 30
	}
	if cfg.usage.inputTokens == 0 && cfg.usage.outputTokens == 0 && cfg.usage.totalTokens == 0 {
		cfg.usage = usageConfig{inputTokens: 10, outputTokens: 20, totalTokens: 30}
	}
	if cfg.streamChunks < 1 {
		cfg.streamChunks = 20
	}
	if len(cfg.failureStatuses) == 0 {
		cfg.failureStatuses = append([]int(nil), mockFailureStatuses[:]...)
	}
	return cfg
}

func loadRequestConfig(header http.Header, cfg config, channelSlot int) (requestConfig, error) {
	requestCfg := requestConfig{errorRate: cfg.errorRate, failureStatus: cfg.failureStatus}
	if value := strings.TrimSpace(header.Get(mockChannelsHeader)); value != "" {
		if len(value) > maxMockChannelsBytes {
			return requestConfig{}, fmt.Errorf("%s is too large", mockChannelsHeader)
		}
		var channels []mockChannelConfig
		if err := json.Unmarshal([]byte(value), &channels); err != nil || len(channels) != 3 {
			return requestConfig{}, fmt.Errorf("%s must contain three channel configurations", mockChannelsHeader)
		}
		seenSlots := make(map[int]struct{}, 3)
		for _, channel := range channels {
			if channel.Slot < 1 || channel.Slot > 3 ||
				math.IsNaN(channel.FailureRate) || math.IsInf(channel.FailureRate, 0) || channel.FailureRate < 0 || channel.FailureRate > 1 ||
				channel.LatencyMS < 0 || channel.LatencyMS > maxMockLatencyMS || !allowedFailureStatus(channel.FailureStatus) {
				return requestConfig{}, fmt.Errorf("%s contains an invalid channel configuration", mockChannelsHeader)
			}
			if _, exists := seenSlots[channel.Slot]; exists {
				return requestConfig{}, fmt.Errorf("%s contains duplicate channel slots", mockChannelsHeader)
			}
			seenSlots[channel.Slot] = struct{}{}
			if channel.Slot == channelSlot {
				requestCfg.errorRate = channel.FailureRate
				requestCfg.errorStatus = channel.FailureStatus
				requestCfg.latency = time.Duration(channel.LatencyMS) * time.Millisecond
			}
		}
		return requestCfg, nil
	}
	if value := strings.TrimSpace(header.Get(mockFailureRateHeader)); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 || parsed > 1 {
			return requestConfig{}, fmt.Errorf("%s must be between 0 and 1", mockFailureRateHeader)
		}
		requestCfg.errorRate = parsed
	}
	if value := strings.TrimSpace(header.Get(mockFailureStatusHeader)); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || !allowedFailureStatus(parsed) {
			return requestConfig{}, fmt.Errorf("%s is unsupported", mockFailureStatusHeader)
		}
		requestCfg.errorStatus = parsed
	}
	if value := strings.TrimSpace(header.Get(mockLatencyMSHeader)); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 || parsed > maxMockLatencyMS {
			return requestConfig{}, fmt.Errorf("%s must be between 0 and %d", mockLatencyMSHeader, maxMockLatencyMS)
		}
		requestCfg.latency = time.Duration(parsed) * time.Millisecond
	}
	return requestCfg, nil
}

func allowedFailureStatus(status int) bool {
	if status == 0 {
		return true
	}
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func resolveFailureStatus(configuredStatus, randomIndex int, statuses ...[]int) int {
	if configuredStatus != 0 {
		return configuredStatus
	}
	available := mockFailureStatuses[:]
	if len(statuses) > 0 && len(statuses[0]) > 0 {
		available = statuses[0]
	}
	if randomIndex < 0 || randomIndex >= len(available) {
		return http.StatusServiceUnavailable
	}
	return available[randomIndex]
}

func resolveDistributedFailureStatus(distribution []failureDistribution, statuses []int, randomValue float64) int {
	if len(distribution) == 0 {
		if len(statuses) == 0 {
			return http.StatusServiceUnavailable
		}
		return resolveFailureStatus(0, rand.IntN(len(statuses)), statuses)
	}
	cumulative := 0.0
	for _, item := range distribution {
		cumulative += item.Rate
		if randomValue < cumulative {
			return item.Status
		}
	}
	return distribution[len(distribution)-1].Status
}

func shouldInjectFailure(rate, randomValue float64) bool {
	return rate > 0 && randomValue < rate
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
	_, _ = fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-loadtest\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d}}\n\n", time.Now().Unix(), cfg.usage.inputTokens, cfg.usage.outputTokens, cfg.usage.totalTokens)
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func applyLoadTestFileConfig(cfg *config, state *channelState, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read load-test config: %w", err)
	}
	var fileConfig loadTestFileConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("parse load-test config: %w", err)
	}
	if fileConfig.Mock.ConsumeTokens > 0 {
		cfg.consumeTokens = fileConfig.Mock.ConsumeTokens
	}
	if fileConfig.Mock.ResponseUsage.inputTokens > 0 {
		cfg.usage = fileConfig.Mock.ResponseUsage
	}
	if fileConfig.Mock.StreamChunks > 0 {
		cfg.streamChunks = fileConfig.Mock.StreamChunks
	}
	if fileConfig.Mock.StreamIntervalMS >= 0 {
		cfg.streamInterval = time.Duration(fileConfig.Mock.StreamIntervalMS) * time.Millisecond
	}
	for _, channel := range fileConfig.Mock.Channels {
		if channel.ID != state.id {
			continue
		}
		if channel.Name != "" {
			state.name = channel.Name
		}
		if channel.Tokens >= 0 {
			state.remaining = channel.Tokens
		}
		if channel.TTFTMS >= 0 {
			cfg.ttft = time.Duration(channel.TTFTMS) * time.Millisecond
		}
		if channel.ResponseMS >= 0 {
			cfg.response = time.Duration(channel.ResponseMS) * time.Millisecond
		}
		cfg.errorRate = channel.ErrorRate
		cfg.failureStatus = append([]failureDistribution(nil), channel.FailureDistribution...)
		break
	}
	cfg.failureStatuses = append([]int(nil), fileConfig.Mock.FailureStatuses...)
	return nil
}

func applyMockEnvironmentOverrides(cfg *config, state *channelState) {
	if value := os.Getenv("TTFT_MS"); value != "" {
		cfg.ttft = durationFromMillis("TTFT_MS", int(cfg.ttft/time.Millisecond))
	}
	if value := os.Getenv("RESPONSE_MS"); value != "" {
		cfg.response = durationFromMillis("RESPONSE_MS", int(cfg.response/time.Millisecond))
	}
	if value := os.Getenv("STREAM_CHUNKS"); value != "" {
		cfg.streamChunks = intFromEnv("STREAM_CHUNKS", cfg.streamChunks)
	}
	if value := os.Getenv("STREAM_INTERVAL_MS"); value != "" {
		cfg.streamInterval = durationFromMillis("STREAM_INTERVAL_MS", int(cfg.streamInterval/time.Millisecond))
	}
	if value := os.Getenv("ERROR_RATE"); value != "" {
		cfg.errorRate = floatFromEnv("ERROR_RATE", cfg.errorRate)
	}
	if value := os.Getenv("CHANNEL_TOKENS"); value != "" {
		state.remaining = int64FromEnv("CHANNEL_TOKENS", state.remaining)
	}
	if value := strings.TrimSpace(os.Getenv("CHANNEL_NAME")); value != "" {
		state.name = value
	}
}

func validateLoadTestConfig(cfg config) error {
	if cfg.consumeTokens < 1 || cfg.usage.inputTokens < 0 || cfg.usage.outputTokens < 0 || cfg.usage.totalTokens < 0 || cfg.usage.inputTokens > cfg.usage.totalTokens || cfg.usage.outputTokens > cfg.usage.totalTokens-cfg.usage.inputTokens {
		return errors.New("load-test config has invalid usage values")
	}
	if cfg.streamChunks < 1 || cfg.streamInterval < 0 || cfg.ttft < 0 || cfg.response < 0 || math.IsNaN(cfg.errorRate) || math.IsInf(cfg.errorRate, 0) || cfg.errorRate < 0 || cfg.errorRate > 1 {
		return errors.New("load-test config has invalid mock values")
	}
	statusSet := make(map[int]struct{}, len(cfg.failureStatuses))
	for _, status := range cfg.failureStatuses {
		if status == 0 || !allowedFailureStatus(status) {
			return errors.New("load-test config has unsupported failure status")
		}
		statusSet[status] = struct{}{}
	}
	if len(statusSet) == 0 {
		return errors.New("load-test config must define at least one failure status")
	}
	total := 0.0
	for _, item := range cfg.failureStatus {
		if math.IsNaN(item.Rate) || math.IsInf(item.Rate, 0) || item.Rate < 0 || item.Rate > 1 || item.Status == 0 {
			return errors.New("load-test config has invalid failure distribution")
		}
		if _, ok := statusSet[item.Status]; !ok {
			return errors.New("load-test config has unsupported failure status")
		}
		total += item.Rate
	}
	if len(cfg.failureStatus) > 0 && (total < 0.999999 || total > 1.000001) {
		return errors.New("load-test failure distribution must sum to 1")
	}
	return nil
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
