package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shirou/gopsutil/mem"
)

const agentVersion = "0.3.0"

var agentHTTPClient = &http.Client{Timeout: 15 * time.Second}

type agentConfig struct {
	ServerURL string `json:"server_url"`
	AgentID   string `json:"agent_id"`
	Secret    string `json:"secret"`
	Name      string `json:"name"`
}

type apiResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type pairResponse struct {
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
}

type pollResponse struct {
	Command string       `json:"command"`
	RunID   string       `json:"run_id"`
	Task    loadTestTask `json:"task"`
}

type loadTestTask struct {
	RunID             string                `json:"run_id"`
	TargetURL         string                `json:"target_url"`
	APIKey            string                `json:"api_key"`
	Model             string                `json:"model"`
	Endpoint          string                `json:"endpoint"`
	Prompt            string                `json:"prompt"`
	PromptCache       bool                  `json:"prompt_cache"`
	MockEnabled       bool                  `json:"mock_enabled"`
	MockFailureRate   float64               `json:"mock_failure_rate"`
	MockFailureStatus int                   `json:"mock_failure_status"`
	MockLatencyMS     int                   `json:"mock_latency_ms"`
	MockChannels      []loadTestMockChannel `json:"mock_channels"`
	DurationSeconds   int                   `json:"duration_seconds"`
	RequestsPerSecond int                   `json:"requests_per_second"`
	Concurrency       int                   `json:"concurrency"`
}

type loadTestMockChannel struct {
	Slot          int     `json:"slot"`
	MaxRPS        int     `json:"max_rps"`
	FailureRate   float64 `json:"failure_rate"`
	FailureStatus int     `json:"failure_status"`
	LatencyMS     int     `json:"latency_ms"`
}

type agentRuntime struct {
	TargetURL      string
	CPUCores       int
	MemoryBytes    int64
	MaxRPS         int
	MaxConcurrency int
}

type agentHeartbeat struct {
	Name           string `json:"name"`
	Platform       string `json:"platform"`
	Version        string `json:"version"`
	CurrentRunID   string `json:"current_run_id"`
	CPUCores       int    `json:"cpu_cores"`
	MemoryBytes    int64  `json:"memory_bytes"`
	MaxRPS         int    `json:"max_rps"`
	MaxConcurrency int    `json:"max_concurrency"`
}

type progressPayload struct {
	Sent        int64            `json:"sent"`
	Completed   int64            `json:"completed"`
	Successes   int64            `json:"successes"`
	Failures    int64            `json:"failures"`
	Dropped     int64            `json:"dropped"`
	CurrentRPS  float64          `json:"current_rps"`
	P50MS       float64          `json:"p50_ms"`
	P95MS       float64          `json:"p95_ms"`
	P99MS       float64          `json:"p99_ms"`
	ErrorCounts map[string]int64 `json:"error_counts"`
}

type finishPayload struct {
	Status           string           `json:"status"`
	Sent             int64            `json:"sent"`
	Completed        int64            `json:"completed"`
	Successes        int64            `json:"successes"`
	Failures         int64            `json:"failures"`
	Dropped          int64            `json:"dropped"`
	InputTokens      int64            `json:"input_tokens"`
	OutputTokens     int64            `json:"output_tokens"`
	CacheReadTokens  int64            `json:"cache_read_tokens"`
	CacheWriteTokens int64            `json:"cache_write_tokens"`
	CurrentRPS       float64          `json:"current_rps"`
	P50MS            float64          `json:"p50_ms"`
	P95MS            float64          `json:"p95_ms"`
	P99MS            float64          `json:"p99_ms"`
	ErrorCounts      map[string]int64 `json:"error_counts"`
	ErrorMessage     string           `json:"error_message"`
}

type k6Metric struct {
	Values map[string]float64 `json:"values"`
	Count  float64            `json:"count"`
	Rate   float64            `json:"rate"`
	Passes float64            `json:"passes"`
	Fails  float64            `json:"fails"`
	Value  float64            `json:"value"`
	Min    float64            `json:"min"`
	Med    float64            `json:"med"`
	Max    float64            `json:"max"`
	Avg    float64            `json:"avg"`
	P90    float64            `json:"p(90)"`
	P95    float64            `json:"p(95)"`
	P99    float64            `json:"p(99)"`
}

func (metric k6Metric) get(name string) float64 {
	if metric.Values != nil {
		return metric.Values[name]
	}
	switch name {
	case "count":
		return metric.Count
	case "rate":
		if metric.Rate != 0 {
			return metric.Rate
		}
		return metric.Value
	case "passes":
		return metric.Passes
	case "fails":
		return metric.Fails
	case "value":
		return metric.Value
	case "min":
		return metric.Min
	case "med":
		return metric.Med
	case "max":
		return metric.Max
	case "avg":
		return metric.Avg
	case "p(90)":
		return metric.P90
	case "p(95)":
		return metric.P95
	case "p(99)":
		return metric.P99
	default:
		return 0
	}
}

type k6Summary struct {
	Metrics map[string]k6Metric `json:"metrics"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "pair":
		err = pair(os.Args[2:])
	case "run":
		err = runAgent(os.Args[2:])
	case "status":
		err = printStatus()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "load-test agent:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  alltoken-loadtest-agent pair --server https://alltokenapi.com CODE")
	fmt.Println("  alltoken-loadtest-agent run [--max-rps N] [--max-concurrency N] [--target-url URL]")
	fmt.Println("  alltoken-loadtest-agent status")
}

func pair(args []string) error {
	flags := flag.NewFlagSet("pair", flag.ContinueOnError)
	serverURL := flags.String("server", "https://alltokenapi.com", "AllToken server URL")
	name := flags.String("name", defaultAgentName(), "agent display name")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("pairing code is required")
	}
	server := normalizeServerURL(*serverURL)
	if err := validateServerURL(server); err != nil {
		return err
	}
	body := map[string]string{
		"code": flags.Arg(0), "name": strings.TrimSpace(*name),
		"platform": runtime.GOOS + "/" + runtime.GOARCH, "version": agentVersion,
	}
	var response apiResponse[pairResponse]
	if err := requestJSON(context.Background(), http.MethodPost, server+"/api/loadtest-agent/pair", "", body, &response); err != nil {
		return err
	}
	if !response.Success {
		return errors.New(response.Message)
	}
	config := agentConfig{ServerURL: server, AgentID: response.Data.AgentID, Secret: response.Data.AgentSecret, Name: strings.TrimSpace(*name)}
	if err := saveConfig(config); err != nil {
		return err
	}
	fmt.Printf("paired agent %s (%s)\n", config.Name, config.AgentID)
	return nil
}

func runAgent(args []string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("k6"); err != nil {
		return errors.New("k6 is not installed or not available in PATH")
	}
	defaultMaxRPS := max(1, runtime.NumCPU()*50)
	defaultMaxConcurrency := max(1, runtime.NumCPU()*100)
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	maxRPS := flags.Int("max-rps", defaultMaxRPS, "maximum accepted requests per second")
	maxConcurrency := flags.Int("max-concurrency", defaultMaxConcurrency, "maximum accepted concurrent requests")
	targetURL := flags.String("target-url", "", "override task target URL; HTTPS or loopback HTTP only")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *maxRPS < 1 || *maxRPS > 1_000_000 || *maxConcurrency < 1 || *maxConcurrency > 1_000_000 {
		return errors.New("agent capacity is invalid")
	}
	normalizedTargetURL := normalizeServerURL(*targetURL)
	if normalizedTargetURL != "" {
		if err := validateServerURL(normalizedTargetURL); err != nil {
			return fmt.Errorf("invalid target URL override: %w", err)
		}
	}
	memoryBytes := int64(0)
	if memory, memoryErr := mem.VirtualMemory(); memoryErr == nil && memory.Total <= math.MaxInt64 {
		memoryBytes = int64(memory.Total)
	}
	runtimeInfo := agentRuntime{
		TargetURL: normalizedTargetURL, CPUCores: runtime.NumCPU(), MemoryBytes: memoryBytes,
		MaxRPS: *maxRPS, MaxConcurrency: *maxConcurrency,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("agent %s is online; capacity %d RPS / %d concurrent\n", config.Name, runtimeInfo.MaxRPS, runtimeInfo.MaxConcurrency)
	var activeRunID string
	var activeCancel context.CancelFunc
	done := make(chan string, 1)
	for {
		select {
		case <-ctx.Done():
			if activeCancel != nil {
				activeCancel()
			}
			return nil
		case completedRunID := <-done:
			if completedRunID == activeRunID {
				activeRunID = ""
				activeCancel = nil
			}
		case <-time.After(2 * time.Second):
		}
		response, err := poll(ctx, config, activeRunID, runtimeInfo)
		if err != nil {
			fmt.Fprintln(os.Stderr, "poll failed:", err)
			continue
		}
		if response.Command == "stop" {
			if activeCancel != nil && response.RunID == activeRunID {
				activeCancel()
			} else if response.RunID != "" {
				if err := postCompletion(context.Background(), config, response.RunID, finishPayload{Status: "cancelled", ErrorMessage: "cancelled after agent restart"}); err != nil {
					fmt.Fprintln(os.Stderr, "cancel acknowledgement failed:", err)
				}
			}
			continue
		}
		if response.Command != "run" || activeRunID != "" {
			continue
		}
		if runtimeInfo.TargetURL != "" {
			response.Task.TargetURL = runtimeInfo.TargetURL
		}
		runCtx, cancel := context.WithCancel(ctx)
		activeRunID = response.Task.RunID
		activeCancel = cancel
		go func(task loadTestTask) {
			defer func() { done <- task.RunID }()
			if err := executeTask(runCtx, config, task); err != nil {
				fmt.Fprintf(os.Stderr, "run %s failed: %v\n", task.RunID, err)
			}
		}(response.Task)
	}
}

func printStatus() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	fmt.Printf("agent_id=%s\nname=%s\nserver=%s\nversion=%s\n", config.AgentID, config.Name, config.ServerURL, agentVersion)
	return nil
}

func poll(ctx context.Context, config agentConfig, currentRunID string, runtimeInfo agentRuntime) (pollResponse, error) {
	var response apiResponse[pollResponse]
	body := agentHeartbeat{
		Name: config.Name, Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Version: agentVersion, CurrentRunID: currentRunID, CPUCores: runtimeInfo.CPUCores,
		MemoryBytes: runtimeInfo.MemoryBytes, MaxRPS: runtimeInfo.MaxRPS, MaxConcurrency: runtimeInfo.MaxConcurrency,
	}
	err := requestJSON(ctx, http.MethodPost, config.ServerURL+"/api/loadtest-agent/poll", config.Secret, body, &response)
	if err != nil {
		return pollResponse{}, err
	}
	if !response.Success {
		return pollResponse{}, errors.New(response.Message)
	}
	return response.Data, nil
}

func executeTask(ctx context.Context, config agentConfig, task loadTestTask) (runErr error) {
	terminalReported := false
	defer func() {
		if runErr == nil || terminalReported {
			return
		}
		reportErr := postCompletion(context.Background(), config, task.RunID, finishPayload{
			Status:       "failed",
			ErrorMessage: runErr.Error(),
		})
		if reportErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("report terminal status: %w", reportErr))
		}
	}()
	if err := validateTask(task); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "alltoken-loadtest-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	keyPath := filepath.Join(tempDir, "api-key")
	if err := os.WriteFile(keyPath, []byte(task.APIKey), 0o600); err != nil {
		return err
	}
	scriptPath := filepath.Join(tempDir, "task.js")
	if err := os.WriteFile(scriptPath, []byte(k6TaskScript), 0o600); err != nil {
		return err
	}
	summaryPath := filepath.Join(tempDir, "summary.json")
	stdoutPath := filepath.Join(tempDir, "stdout.log")
	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer stdoutFile.Close()
	command := exec.CommandContext(ctx, "k6", "run", "--summary-export", summaryPath, scriptPath)
	command.Stdout = stdoutFile
	command.Stderr = stdoutFile
	mockChannelsJSON, err := common.Marshal(task.MockChannels)
	if err != nil {
		return fmt.Errorf("encode mock channels: %w", err)
	}
	command.Env = append(os.Environ(),
		"ALLTOKEN_RUN_ID="+task.RunID,
		"ALLTOKEN_TARGET_URL="+task.TargetURL,
		"ALLTOKEN_API_KEY_FILE="+keyPath,
		"ALLTOKEN_MODEL="+task.Model,
		"ALLTOKEN_ENDPOINT="+task.Endpoint,
		"ALLTOKEN_PROMPT="+task.Prompt,
		"ALLTOKEN_PROMPT_CACHE="+strconv.FormatBool(task.PromptCache),
		"ALLTOKEN_MOCK_ENABLED="+strconv.FormatBool(task.MockEnabled),
		"ALLTOKEN_MOCK_FAILURE_RATE="+strconv.FormatFloat(task.MockFailureRate, 'f', -1, 64),
		"ALLTOKEN_MOCK_FAILURE_STATUS="+strconv.Itoa(task.MockFailureStatus),
		"ALLTOKEN_MOCK_LATENCY_MS="+strconv.Itoa(task.MockLatencyMS),
		"ALLTOKEN_MOCK_CHANNELS="+string(mockChannelsJSON),
		"ALLTOKEN_DURATION_SECONDS="+strconv.Itoa(task.DurationSeconds),
		"ALLTOKEN_RPS="+strconv.Itoa(task.RequestsPerSecond),
		"ALLTOKEN_CONCURRENCY="+strconv.Itoa(task.Concurrency),
	)
	startedAt := time.Now()
	if err := command.Start(); err != nil {
		return err
	}
	fmt.Printf("run %s started (%d RPS, %ds, concurrency %d)\n", task.RunID, task.RequestsPerSecond, task.DurationSeconds, task.Concurrency)
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = postProgress(context.Background(), config, task.RunID, progressPayload{CurrentRPS: float64(task.RequestsPerSecond)})
		case commandErr := <-done:
			payload, summaryErr := readK6Summary(summaryPath, task.RequestsPerSecond)
			payload.ErrorMessage = ""
			payload.Status = "completed"
			if errors.Is(ctx.Err(), context.Canceled) {
				payload.Status = "cancelled"
				payload.ErrorMessage = "cancelled by user"
			} else if commandErr != nil || summaryErr != nil {
				payload.Status = "failed"
				if commandErr != nil {
					payload.ErrorMessage = commandErr.Error()
				} else {
					payload.ErrorMessage = summaryErr.Error()
				}
			}
			if err := postCompletion(context.Background(), config, task.RunID, payload); err != nil {
				return err
			}
			terminalReported = true
			fmt.Printf("run %s %s in %s (%d completed, %d failed)\n", task.RunID, payload.Status, time.Since(startedAt).Round(time.Second), payload.Completed, payload.Failures)
			return nil
		}
	}
}

func readK6Summary(path string, configuredRPS int) (finishPayload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return finishPayload{}, err
	}
	var summary k6Summary
	if err := common.Unmarshal(data, &summary); err != nil {
		return finishPayload{}, err
	}
	metric := func(name, key string) float64 { return summary.Metrics[name].get(key) }
	count := func(name string) int64 { return int64(metric(name, "count")) }
	completed := count("http_reqs")
	if completed == 0 {
		return finishPayload{}, errors.New("k6 completed without issuing any HTTP requests")
	}
	failed := int64(metric("http_req_failed", "passes"))
	if failed == 0 {
		failed = int64(metric("http_req_failed", "rate") * float64(completed))
	}
	successes := completed - failed
	if successes < 0 {
		successes = 0
	}
	errorCounts := map[string]int64{}
	for _, status := range []string{"0", "400", "401", "403", "408", "409", "429", "500", "502", "503", "504"} {
		if statusCount := count("alltoken_status_" + status); statusCount > 0 {
			if status == "0" {
				errorCounts["network_error"] = statusCount
			} else {
				errorCounts["http_"+status] = statusCount
			}
		}
	}
	for name, values := range summary.Metrics {
		const prefix = "alltoken_errors{code:"
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, "}") {
			code := strings.TrimSuffix(strings.TrimPrefix(name, prefix), "}")
			if code != "" {
				errorCounts[code] += int64(values.get("count"))
			}
		}
	}
	if failed > 0 && len(errorCounts) == 0 {
		errorCounts["request_failed"] = failed
	}
	return finishPayload{
		Sent: completed + count("dropped_iterations"), Completed: completed, Successes: successes,
		Failures: failed, Dropped: count("dropped_iterations"), CurrentRPS: float64(configuredRPS),
		P50MS: metric("http_req_duration", "med"), P95MS: metric("http_req_duration", "p(95)"),
		P99MS: metric("http_req_duration", "p(99)"), InputTokens: count("alltoken_input_tokens"),
		OutputTokens: count("alltoken_output_tokens"), CacheReadTokens: count("alltoken_cache_read_tokens"),
		CacheWriteTokens: count("alltoken_cache_write_tokens"), ErrorCounts: errorCounts,
	}, nil
}

func postProgress(ctx context.Context, config agentConfig, runID string, payload progressPayload) error {
	var response apiResponse[any]
	if err := requestJSON(ctx, http.MethodPost, config.ServerURL+"/api/loadtest-agent/runs/"+runID+"/progress", config.Secret, payload, &response); err != nil {
		return err
	}
	if !response.Success {
		return errors.New(response.Message)
	}
	return nil
}

func postCompletion(ctx context.Context, config agentConfig, runID string, payload finishPayload) error {
	var response apiResponse[any]
	if err := requestJSON(ctx, http.MethodPost, config.ServerURL+"/api/loadtest-agent/runs/"+runID+"/complete", config.Secret, payload, &response); err != nil {
		return err
	}
	if !response.Success {
		return errors.New(response.Message)
	}
	return nil
}

func requestJSON(ctx context.Context, method, endpoint, secret string, body any, output any) error {
	encoded, err := common.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if secret != "" {
		request.Header.Set("Authorization", "Bearer "+secret)
	}
	response, err := agentHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("server returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	return common.DecodeJson(response.Body, output)
}

func configPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "alltoken", "loadtest-agent.json"), nil
}

func saveConfig(config agentConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := common.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func loadConfig() (agentConfig, error) {
	path, err := configPath()
	if err != nil {
		return agentConfig{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return agentConfig{}, fmt.Errorf("agent is not paired: %w", err)
	}
	var config agentConfig
	if err := common.Unmarshal(data, &config); err != nil {
		return agentConfig{}, err
	}
	return config, nil
}

func defaultAgentName() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return runtime.GOOS + " load-test agent"
	}
	return hostname
}

func normalizeServerURL(value string) string { return strings.TrimRight(strings.TrimSpace(value), "/") }

func validateServerURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return errors.New("server URL is invalid")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	hostname := parsed.Hostname()
	if parsed.Scheme == "http" && (hostname == "127.0.0.1" || hostname == "localhost" || hostname == "::1") {
		return nil
	}
	return errors.New("server must use HTTPS, except localhost development")
}

func validateTask(task loadTestTask) error {
	if strings.TrimSpace(task.RunID) == "" || len(task.RunID) > 64 || strings.TrimSpace(task.APIKey) == "" || len(task.APIKey) > 256 {
		return errors.New("task identity or API key is invalid")
	}
	if err := validateServerURL(task.TargetURL); err != nil {
		return fmt.Errorf("invalid task target: %w", err)
	}
	switch task.Endpoint {
	case "anthropic", "openai", "openai-response", "openai-response-compact":
	default:
		return errors.New("task endpoint is invalid")
	}
	if strings.TrimSpace(task.Model) == "" || len(task.Model) > 128 || strings.TrimSpace(task.Prompt) == "" || len(task.Prompt) > 8000 {
		return errors.New("task model or prompt is invalid")
	}
	if task.DurationSeconds < 1 || task.RequestsPerSecond < 1 || task.Concurrency < 1 {
		return errors.New("task load settings are invalid")
	}
	if task.MockEnabled {
		if len(task.MockChannels) > 0 {
			if task.MockFailureRate != 0 || task.MockFailureStatus != 0 || task.MockLatencyMS != 0 || len(task.MockChannels) != 3 {
				return errors.New("task mock channel settings are invalid")
			}
			seenSlots := make(map[int]struct{}, 3)
			for _, channel := range task.MockChannels {
				if channel.Slot < 1 || channel.Slot > 3 || channel.MaxRPS < 1 || channel.MaxRPS > 1_000_000 ||
					math.IsNaN(channel.FailureRate) || math.IsInf(channel.FailureRate, 0) || channel.FailureRate < 0 || channel.FailureRate > 1 ||
					channel.LatencyMS < 0 || channel.LatencyMS > 120000 {
					return errors.New("task mock channel settings are invalid")
				}
				if _, exists := seenSlots[channel.Slot]; exists {
					return errors.New("task mock channel settings are invalid")
				}
				seenSlots[channel.Slot] = struct{}{}
				switch channel.FailureStatus {
				case 0, 429, 500, 502, 503, 504:
				default:
					return errors.New("task mock channel failure status is invalid")
				}
			}
			return nil
		}
		if math.IsNaN(task.MockFailureRate) || math.IsInf(task.MockFailureRate, 0) || task.MockFailureRate < 0 || task.MockFailureRate > 1 || task.MockLatencyMS < 0 || task.MockLatencyMS > 120000 {
			return errors.New("task mock settings are invalid")
		}
		switch task.MockFailureStatus {
		case 0, 429, 500, 502, 503, 504:
		default:
			return errors.New("task mock failure status is invalid")
		}
	}
	return nil
}

const k6TaskScript = `
import http from 'k6/http';
import { Counter } from 'k6/metrics';

http.setResponseCallback(http.expectedStatuses(200));

const apiKey = open(__ENV.ALLTOKEN_API_KEY_FILE).trim();
const target = __ENV.ALLTOKEN_TARGET_URL.replace(/\/+$/, '');
const endpoint = __ENV.ALLTOKEN_ENDPOINT;
const durationSeconds = Number(__ENV.ALLTOKEN_DURATION_SECONDS);
const targetRPS = Number(__ENV.ALLTOKEN_RPS);
const concurrency = Number(__ENV.ALLTOKEN_CONCURRENCY);
const promptCache = __ENV.ALLTOKEN_PROMPT_CACHE === 'true';
const mockEnabled = __ENV.ALLTOKEN_MOCK_ENABLED === 'true';
const mockChannels = __ENV.ALLTOKEN_MOCK_CHANNELS || '';
const cachePrefix = Array.from(
  { length: 48 },
  (_, index) => 'Stable load-test context section ' + (index + 1) + ': keep this deterministic prefix unchanged so provider prompt caching can reuse it across requests. The demo measures gateway routing and usage reporting only.'
).join('\n');
const inputTokens = new Counter('alltoken_input_tokens');
const outputTokens = new Counter('alltoken_output_tokens');
const cacheReadTokens = new Counter('alltoken_cache_read_tokens');
const cacheWriteTokens = new Counter('alltoken_cache_write_tokens');
const alltokenErrors = new Counter('alltoken_errors');
const errorStatusCounters = {
  0: new Counter('alltoken_status_0'), 400: new Counter('alltoken_status_400'), 401: new Counter('alltoken_status_401'),
  403: new Counter('alltoken_status_403'), 408: new Counter('alltoken_status_408'),
  409: new Counter('alltoken_status_409'), 429: new Counter('alltoken_status_429'),
  500: new Counter('alltoken_status_500'), 502: new Counter('alltoken_status_502'),
  503: new Counter('alltoken_status_503'), 504: new Counter('alltoken_status_504'),
};

export const options = {
  discardResponseBodies: false,
  scenarios: {
    load: {
      executor: 'constant-arrival-rate',
      rate: targetRPS,
      timeUnit: '1s',
      duration: durationSeconds + 's',
      preAllocatedVUs: Math.min(concurrency, Math.max(1, targetRPS)),
      maxVUs: concurrency,
    },
  },
  summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max'],
};

export default function () {
  const requestPath = endpoint === 'anthropic' ? '/v1/messages'
    : endpoint === 'openai' ? '/v1/chat/completions'
    : endpoint === 'openai-response-compact' ? '/v1/responses/compact'
    : '/v1/responses';
  const headers = {
    Authorization: 'Bearer ' + apiKey,
    'Content-Type': 'application/json',
    'X-Load-Test-ID': __ENV.ALLTOKEN_RUN_ID + '-' + __VU + '-' + __ITER,
  };
  if (mockEnabled) {
    headers['X-Alltoken-Mock-Load-Test'] = 'true';
    if (mockChannels && mockChannels !== 'null' && mockChannels !== '[]') {
      headers['X-Alltoken-Mock-Channels'] = mockChannels;
    } else {
      headers['X-Alltoken-Mock-Failure-Rate'] = __ENV.ALLTOKEN_MOCK_FAILURE_RATE;
      headers['X-Alltoken-Mock-Failure-Status'] = __ENV.ALLTOKEN_MOCK_FAILURE_STATUS;
      headers['X-Alltoken-Mock-Latency-Ms'] = __ENV.ALLTOKEN_MOCK_LATENCY_MS;
    }
  }
  if (endpoint === 'anthropic') {
    headers['x-api-key'] = apiKey;
    headers['anthropic-version'] = '2023-06-01';
  }
  let body;
  if (endpoint === 'anthropic') {
    body = { model: __ENV.ALLTOKEN_MODEL, max_tokens: 32, messages: [{ role: 'user', content: __ENV.ALLTOKEN_PROMPT }] };
    if (promptCache) {
      body.system = [{ type: 'text', text: cachePrefix, cache_control: { type: 'ephemeral' } }];
    }
  } else if (endpoint === 'openai') {
    body = { model: __ENV.ALLTOKEN_MODEL, max_tokens: 32, temperature: 0, stream: false, messages: [{ role: 'user', content: __ENV.ALLTOKEN_PROMPT }] };
    if (promptCache) {
      body.messages = [{ role: 'system', content: cachePrefix }, { role: 'user', content: __ENV.ALLTOKEN_PROMPT }];
    }
  } else {
    body = {
      model: __ENV.ALLTOKEN_MODEL,
      input: promptCache
        ? [{ role: 'system', content: cachePrefix }, { role: 'user', content: __ENV.ALLTOKEN_PROMPT }]
        : [{ role: 'user', content: __ENV.ALLTOKEN_PROMPT }],
    };
    if (endpoint === 'openai-response') {
      body.max_output_tokens = 32;
      body.stream = false;
    }
  }
  const response = http.post(target + requestPath, JSON.stringify(body), { headers, timeout: '120s' });
  if (response.status !== 200) {
    if (errorStatusCounters[response.status]) errorStatusCounters[response.status].add(1);
    let payload;
    try { payload = response.json(); } catch (_) { payload = {}; }
    const responseError = payload && typeof payload.error === 'object' ? payload.error : {};
    const rawErrorCode = responseError.code || responseError.type ||
      response.headers['X-Alltoken-Error-Code'] || response.headers['X-Alltoken-Code'] ||
      (response.status === 0 ? 'network_error' : 'http_' + response.status);
    const errorCode = String(rawErrorCode).replace(/[^A-Za-z0-9._:-]/g, '_').slice(0, 128) || 'unknown_error';
    alltokenErrors.add(1, { code: errorCode });
    return;
  }
  let usage;
  try { usage = response.json().usage; } catch (_) { return; }
  const totalInput = Number(usage.input_tokens || usage.prompt_tokens || 0);
  const promptDetails = usage.prompt_tokens_details || {};
  const inputDetails = usage.input_tokens_details || {};
  const cacheRead = Number(usage.cache_read_input_tokens || promptDetails.cached_tokens || inputDetails.cached_tokens || 0);
  const cacheCreation = usage.cache_creation || {};
  const cacheWrite = Number(
    usage.cache_creation_input_tokens || usage.cache_write_tokens ||
    (Number(cacheCreation.ephemeral_5m_input_tokens || 0) + Number(cacheCreation.ephemeral_1h_input_tokens || 0))
  );
  inputTokens.add(endpoint === 'anthropic' ? totalInput : Math.max(0, totalInput - cacheRead - cacheWrite));
  outputTokens.add(Number(usage.output_tokens || usage.completion_tokens || 0));
  cacheReadTokens.add(cacheRead);
  cacheWriteTokens.add(cacheWrite);
}
`
