package controller

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	loadTestPairingTTLSeconds  = 300
	loadTestAgentOnlineSeconds = 30
	loadTestAgentBodyLimit     = 1 << 20
	loadTestMaxPromptChars     = 8000
	loadTestMaxErrorMessage    = 4096
	loadTestMaxErrorKinds      = 100
	loadTestMaxMockLatencyMS   = 120000
	loadTestMockChannelCount   = 3
	loadTestMockAgentVersion   = "0.5.0"
	loadTestSharedMinWorkers   = 2
	loadTestSharedMaxWorkers   = 256
)

var loadTestEndpoints = map[string]struct{}{
	"anthropic":               {},
	"openai":                  {},
	"openai-response":         {},
	"openai-response-compact": {},
}

type pairLoadTestAgentRequest struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Platform       string `json:"platform"`
	Version        string `json:"version"`
	CPUCores       int    `json:"cpu_cores"`
	MemoryBytes    int64  `json:"memory_bytes"`
	MaxRPS         int    `json:"max_rps"`
	MaxConcurrency int    `json:"max_concurrency"`
}

type createLoadTestRunRequest struct {
	AgentID           string                      `json:"agent_id"`
	ExecutionMode     model.LoadTestExecutionMode `json:"execution_mode"`
	ExpectedWorkers   int                         `json:"expected_workers"`
	TokenID           int                         `json:"token_id"`
	Model             string                      `json:"model"`
	Endpoint          string                      `json:"endpoint"`
	Prompt            string                      `json:"prompt"`
	PromptCache       bool                        `json:"prompt_cache"`
	MockEnabled       bool                        `json:"mock_enabled"`
	MockFailureRate   float64                     `json:"mock_failure_rate"`
	MockFailureStatus int                         `json:"mock_failure_status"`
	MockLatencyMS     int                         `json:"mock_latency_ms"`
	MockChannels      []model.LoadTestMockChannel `json:"mock_channels"`
	DurationSeconds   int                         `json:"duration_seconds"`
	RequestsPerSecond int                         `json:"requests_per_second"`
	Concurrency       int                         `json:"concurrency"`
}

type loadTestAgentPollRequest struct {
	Name           string `json:"name"`
	Platform       string `json:"platform"`
	Version        string `json:"version"`
	WorkerID       string `json:"worker_id"`
	CurrentRunID   string `json:"current_run_id"`
	CPUCores       int    `json:"cpu_cores"`
	MemoryBytes    int64  `json:"memory_bytes"`
	MaxRPS         int    `json:"max_rps"`
	MaxConcurrency int    `json:"max_concurrency"`
}

type updateManagedLoadTestAgentCapacityRequest struct {
	MaxRPS         int `json:"max_rps"`
	MaxConcurrency int `json:"max_concurrency"`
}

type loadTestRunProgressRequest struct {
	WorkerID    string           `json:"worker_id"`
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

type finishLoadTestRunRequest struct {
	WorkerID         string                  `json:"worker_id"`
	Status           model.LoadTestRunStatus `json:"status"`
	Sent             int64                   `json:"sent"`
	Completed        int64                   `json:"completed"`
	Successes        int64                   `json:"successes"`
	Failures         int64                   `json:"failures"`
	Dropped          int64                   `json:"dropped"`
	InputTokens      int64                   `json:"input_tokens"`
	OutputTokens     int64                   `json:"output_tokens"`
	CacheReadTokens  int64                   `json:"cache_read_tokens"`
	CacheWriteTokens int64                   `json:"cache_write_tokens"`
	CurrentRPS       float64                 `json:"current_rps"`
	P50MS            float64                 `json:"p50_ms"`
	P95MS            float64                 `json:"p95_ms"`
	P99MS            float64                 `json:"p99_ms"`
	ErrorCounts      map[string]int64        `json:"error_counts"`
	ErrorMessage     string                  `json:"error_message"`
}

func requireLoadTestUser(c *gin.Context) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if !user.IsToB() {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "load test demo is only available to ToB users"})
		return false
	}
	return true
}

func CreateLoadTestAgentPairing(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	code, err := common.GenerateRandomCharsKey(8)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	code = strings.ToUpper(code)
	expiresAt := common.GetTimestamp() + loadTestPairingTTLSeconds
	agent, err := model.NewLoadTestAgentPairing(c.GetInt("id"), code, expiresAt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"agent_id": agent.ID, "code": code, "expires_at": expiresAt})
}

func CreateManagedLoadTestAgentPairing(c *gin.Context) {
	code, err := common.GenerateRandomCharsKey(8)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	code = strings.ToUpper(code)
	expiresAt := common.GetTimestamp() + loadTestPairingTTLSeconds
	agent, err := model.NewManagedLoadTestAgentPairing(c.GetInt("id"), code, expiresAt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"agent_id": agent.ID, "code": code, "expires_at": expiresAt})
}

func ListLoadTestAgents(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	localAgents, err := model.ListLoadTestAgents(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	managedAgents, err := model.ListManagedLoadTestAgents()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"local_agents": localAgents, "managed_agents": managedAgents, "online_before": common.GetTimestamp() - loadTestAgentOnlineSeconds})
}

func GetLoadTestState(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	localAgents, err := model.ListLoadTestAgents(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	managedAgents, err := model.ListManagedLoadTestAgents()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	runs, err := model.ListLoadTestRuns(c.GetInt("id"), 30)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"local_agents":   localAgents,
		"managed_agents": managedAgents,
		"online_before":  common.GetTimestamp() - loadTestAgentOnlineSeconds,
		"runs":           runs,
	})
}

func DeleteLoadTestAgent(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	if err := model.RevokeLoadTestAgent(c.GetInt("id"), c.Param("id")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "load-test agent not found"})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteManagedLoadTestAgent(c *gin.Context) {
	if err := model.RevokeManagedLoadTestAgent(c.Param("id")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "managed load-test agent not found"})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func UpdateManagedLoadTestAgentCapacity(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, loadTestAgentBodyLimit)
	var request updateManagedLoadTestAgentCapacityRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid agent capacity"})
		return
	}
	settings := operation_setting.GetLoadTestSetting()
	if request.MaxRPS < 1 || request.MaxRPS > settings.MaxRPS || request.MaxConcurrency < 1 || request.MaxConcurrency > settings.MaxConcurrency {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "agent capacity exceeds load-test limits"})
		return
	}
	agent, err := model.UpdateManagedLoadTestAgentCapacity(c.Param("id"), request.MaxRPS, request.MaxConcurrency)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "managed load-test agent not found"})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, agent)
}

func PairLoadTestAgent(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, loadTestAgentBodyLimit)
	var request pairLoadTestAgentRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid pairing request"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Platform = strings.TrimSpace(request.Platform)
	request.Version = strings.TrimSpace(request.Version)
	if len(request.Name) < 1 || len(request.Name) > 128 || len(request.Platform) > 64 || len(request.Version) > 32 || !validLoadTestAgentResources(request.CPUCores, request.MemoryBytes, request.MaxRPS, request.MaxConcurrency) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid agent metadata"})
		return
	}
	secret, err := common.GenerateRandomCharsKey(48)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	agent, err := model.PairLoadTestAgent(request.Code, secret, loadTestAgentRuntime(request.Name, request.Platform, request.Version, request.CPUCores, request.MemoryBytes, request.MaxRPS, request.MaxConcurrency))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, gin.H{"agent_id": agent.ID, "agent_secret": secret})
}

func CreateLoadTestRun(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, loadTestAgentBodyLimit)
	var request createLoadTestRunRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid load-test request"})
		return
	}
	settings := operation_setting.GetLoadTestSetting()
	if request.DurationSeconds < operation_setting.LoadTestMinDurationSeconds || request.DurationSeconds > settings.MaxDurationSeconds ||
		request.RequestsPerSecond < 1 || request.RequestsPerSecond > settings.MaxRPS ||
		request.Concurrency < 1 || request.Concurrency > settings.MaxConcurrency {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "load-test limits exceeded"})
		return
	}
	request.Model = strings.TrimSpace(request.Model)
	request.Endpoint = strings.TrimSpace(request.Endpoint)
	request.Prompt = strings.TrimSpace(request.Prompt)
	if len(request.Model) < 1 || len(request.Model) > 128 || len(request.Prompt) < 1 || len(request.Prompt) > loadTestMaxPromptChars {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid model or prompt"})
		return
	}
	if _, ok := loadTestEndpoints[request.Endpoint]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "unsupported load-test endpoint"})
		return
	}
	agent, err := model.GetUsableLoadTestAgent(c.GetInt("id"), request.AgentID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if agent.LastSeenAt < common.GetTimestamp()-loadTestAgentOnlineSeconds {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "load-test agent is offline"})
		return
	}
	if request.ExecutionMode == "" {
		request.ExecutionMode = model.LoadTestExecutionSingle
	}
	if err := validateLoadTestExecutionSettings(agent, request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if request.ExecutionMode == model.LoadTestExecutionSingle {
		if err := validateLoadTestAgentCapacity(agent, request); err != nil {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	if err := validateLoadTestMockSettings(agent, request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	token, err := model.GetTokenByIds(request.TokenID, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if token.Status != common.TokenStatusEnabled {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "API key is not enabled"})
		return
	}
	targetURL := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	parsedTarget, err := url.Parse(targetURL)
	if err != nil || (parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https") || parsedTarget.Host == "" {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "server address is not configured"})
		return
	}
	packageName := strings.TrimSpace(token.Group)
	if packageName == "" {
		candidates, candidateErr := token.GetGroupCandidates()
		if candidateErr == nil && len(candidates) > 0 {
			packageName = candidates[0]
		}
	}
	mockChannelsJSON, err := model.EncodeLoadTestMockChannels(request.MockChannels)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	run := &model.LoadTestRun{
		UserID: c.GetInt("id"), AgentID: agent.ID, TokenID: token.Id,
		KeyName: token.Name, PackageName: packageName, Model: request.Model,
		Endpoint: request.Endpoint, Prompt: request.Prompt, PromptCache: request.PromptCache, AgentManaged: agent.Managed,
		ExecutionMode: request.ExecutionMode, ExpectedWorkers: request.ExpectedWorkers,
		MockEnabled: request.MockEnabled, MockFailureRate: request.MockFailureRate,
		MockFailureStatus: request.MockFailureStatus, MockLatencyMS: request.MockLatencyMS,
		MockChannelsJSON: mockChannelsJSON, MockChannels: request.MockChannels,
		TargetURL: targetURL, DurationSeconds: request.DurationSeconds,
		RequestsPerSecond: request.RequestsPerSecond, Concurrency: request.Concurrency,
	}
	if err := model.CreateLoadTestRun(run); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, run)
}

func ListLoadTestRuns(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	runs, err := model.ListLoadTestRuns(c.GetInt("id"), limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, runs)
}

func GetLoadTestRun(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	run, err := model.GetLoadTestRun(c.GetInt("id"), c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, run)
}

func CancelLoadTestRun(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	if err := model.RequestLoadTestRunCancellation(c.GetInt("id"), c.Param("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func PollLoadTestAgent(c *gin.Context) {
	agent, ok := authenticateLoadTestAgentRequest(c)
	if !ok {
		return
	}
	var request loadTestAgentPollRequest
	if c.Request.ContentLength > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, loadTestAgentBodyLimit)
		if err := common.DecodeJson(c.Request.Body, &request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid poll request"})
			return
		}
	}
	if !validLoadTestAgentResources(request.CPUCores, request.MemoryBytes, request.MaxRPS, request.MaxConcurrency) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid agent resources"})
		return
	}
	if err := model.TouchLoadTestAgent(agent.ID, loadTestAgentRuntime(request.Name, request.Platform, request.Version, request.CPUCores, request.MemoryBytes, request.MaxRPS, request.MaxConcurrency)); err != nil {
		common.ApiError(c, err)
		return
	}
	cancelRun, err := model.GetAgentCancelRequest(agent.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if cancelRun != nil {
		common.ApiSuccess(c, gin.H{"command": "stop", "run_id": cancelRun.ID})
		return
	}
	runtimeInfo := loadTestAgentRuntime(request.Name, request.Platform, request.Version, request.CPUCores, request.MemoryBytes, request.MaxRPS, request.MaxConcurrency)
	workerID := strings.TrimSpace(request.WorkerID)
	if workerID != "" && !validLoadTestWorkerID(workerID) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid worker id"})
		return
	}
	if strings.TrimSpace(request.CurrentRunID) != "" {
		currentRun, runErr := model.GetLoadTestRunByID(request.CurrentRunID)
		if runErr != nil {
			common.ApiError(c, runErr)
			return
		}
		if currentRun.Status != model.LoadTestRunDispatched && currentRun.Status != model.LoadTestRunRunning {
			common.ApiSuccess(c, gin.H{"command": "stop", "run_id": currentRun.ID})
			return
		}
		if workerID != "" && currentRun.ExecutionMode == model.LoadTestExecutionShared {
			if err := model.TouchLoadTestRunWorker(request.CurrentRunID, workerID, runtimeInfo); err != nil {
				common.ApiError(c, err)
				return
			}
		}
		common.ApiSuccess(c, gin.H{"command": "wait"})
		return
	}
	if workerID != "" {
		worker, err := model.ClaimLoadTestRunForWorker(agent.ID, workerID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if worker != nil {
			run, err := model.GetLoadTestRunByID(worker.RunID)
			if err != nil {
				common.ApiError(c, err)
				return
			}
			token, err := model.GetTokenByIds(run.TokenID, run.UserID)
			if err != nil {
				_ = model.FinishLoadTestRunWorker(worker.RunID, worker.WorkerID, model.LoadTestRunWorkerFailed, map[string]any{"error_message": "API key is unavailable"})
				common.ApiError(c, err)
				return
			}
			common.ApiSuccess(c, gin.H{
				"command": "run",
				"task": gin.H{
					"run_id": run.ID, "worker_id": worker.WorkerID, "target_url": run.TargetURL, "api_key": "sk-" + token.GetFullKey(),
					"model": run.Model, "endpoint": run.Endpoint, "prompt": run.Prompt,
					"prompt_cache": run.PromptCache, "duration_seconds": run.DurationSeconds,
					"mock_enabled": run.MockEnabled, "mock_failure_rate": run.MockFailureRate,
					"mock_failure_status": run.MockFailureStatus, "mock_latency_ms": run.MockLatencyMS,
					"mock_channels":       run.MockChannels,
					"mock_token":          service.MockLoadTestSignature(run.ID, run.TokenID, run.MockChannelsJSON, run.MockFailureRate, run.MockFailureStatus, run.MockLatencyMS),
					"requests_per_second": worker.AssignedRPS, "concurrency": worker.AssignedConcurrency,
					"start_at": run.StartAt, "expected_workers": run.ExpectedWorkers,
				},
			})
			return
		}
	}
	run, err := model.ClaimLoadTestRun(agent.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if run == nil {
		common.ApiSuccess(c, gin.H{"command": "wait"})
		return
	}
	token, err := model.GetTokenByIds(run.TokenID, run.UserID)
	if err != nil {
		_ = model.FinishLoadTestRun(agent.ID, run.ID, model.LoadTestRunFailed, map[string]any{"error_message": "API key is unavailable"})
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"command": "run",
		"task": gin.H{
			"run_id": run.ID, "target_url": run.TargetURL, "api_key": "sk-" + token.GetFullKey(),
			"model": run.Model, "endpoint": run.Endpoint, "prompt": run.Prompt,
			"prompt_cache": run.PromptCache, "duration_seconds": run.DurationSeconds,
			"mock_enabled": run.MockEnabled, "mock_failure_rate": run.MockFailureRate,
			"mock_failure_status": run.MockFailureStatus, "mock_latency_ms": run.MockLatencyMS,
			"mock_channels":       run.MockChannels,
			"mock_token":          service.MockLoadTestSignature(run.ID, run.TokenID, run.MockChannelsJSON, run.MockFailureRate, run.MockFailureStatus, run.MockLatencyMS),
			"requests_per_second": run.RequestsPerSecond, "concurrency": run.Concurrency,
		},
	})
}

func validateLoadTestMockSettings(agent *model.LoadTestAgent, request createLoadTestRunRequest) error {
	if !request.MockEnabled {
		if request.MockFailureRate != 0 || request.MockFailureStatus != 0 || request.MockLatencyMS != 0 || len(request.MockChannels) != 0 {
			return errors.New("mock settings require mock mode")
		}
		return nil
	}
	if !agent.Managed {
		return errors.New("mock mode is only available on server load-test agents")
	}
	if !loadTestAgentSupportsMockChannels(agent.Version) {
		return fmt.Errorf("server load-test agent must be updated to version %s or newer", loadTestMockAgentVersion)
	}
	if len(request.MockChannels) > 0 {
		if request.MockFailureRate != 0 || request.MockFailureStatus != 0 || request.MockLatencyMS != 0 {
			return errors.New("per-channel mock settings cannot be combined with legacy mock settings")
		}
		if len(request.MockChannels) != loadTestMockChannelCount {
			return fmt.Errorf("mock mode requires exactly %d channel configurations", loadTestMockChannelCount)
		}
		seenSlots := make(map[int]struct{}, loadTestMockChannelCount)
		for _, channel := range request.MockChannels {
			if channel.Slot < 1 || channel.Slot > loadTestMockChannelCount {
				return fmt.Errorf("mock channel slot must be between 1 and %d", loadTestMockChannelCount)
			}
			if _, exists := seenSlots[channel.Slot]; exists {
				return errors.New("mock channel slots must be unique")
			}
			seenSlots[channel.Slot] = struct{}{}
			if math.IsNaN(channel.FailureRate) || math.IsInf(channel.FailureRate, 0) || channel.FailureRate < 0 || channel.FailureRate > 1 {
				return errors.New("mock channel failure rate must be between 0 and 1")
			}
			if channel.LatencyMS < 0 || channel.LatencyMS > loadTestMaxMockLatencyMS {
				return fmt.Errorf("mock channel latency must be between 0 and %d milliseconds", loadTestMaxMockLatencyMS)
			}
			if !validLoadTestMockFailureStatus(channel.FailureStatus) {
				return errors.New("unsupported mock channel failure status")
			}
		}
		return nil
	}
	if math.IsNaN(request.MockFailureRate) || math.IsInf(request.MockFailureRate, 0) || request.MockFailureRate < 0 || request.MockFailureRate > 1 {
		return errors.New("mock failure rate must be between 0 and 1")
	}
	if request.MockLatencyMS < 0 || request.MockLatencyMS > loadTestMaxMockLatencyMS {
		return fmt.Errorf("mock latency must be between 0 and %d milliseconds", loadTestMaxMockLatencyMS)
	}
	if !validLoadTestMockFailureStatus(request.MockFailureStatus) {
		return errors.New("unsupported mock failure status")
	}
	return nil
}

func validLoadTestMockFailureStatus(status int) bool {
	switch status {
	case 0, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func loadTestAgentSupportsMockChannels(version string) bool {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	patch, patchErr := strconv.Atoi(parts[2])
	if majorErr != nil || minorErr != nil || patchErr != nil || major < 0 || minor < 0 || patch < 0 {
		return false
	}
	return major > 0 || minor > 4 || (minor == 4 && patch >= 0)
}

func validateLoadTestMockToken(token *model.Token, modelName, endpoint string) error {
	requestPath := "/v1/responses"
	switch endpoint {
	case "anthropic":
		requestPath = "/v1/messages"
	case "openai":
		requestPath = "/v1/chat/completions"
	case "openai-response-compact":
		requestPath = "/v1/responses/compact"
	}
	groups := []string{strings.TrimSpace(token.Group)}
	if token.Group == "auto" {
		var err error
		groups, err = token.GetGroupCandidates()
		if err != nil {
			return fmt.Errorf("invalid API key billing groups: %w", err)
		}
	}
	if len(groups) == 0 {
		return errors.New("mock API key has no billing group")
	}
	for _, group := range groups {
		_, routeChannels, ok := model.ResolveBillingGroupRoute(group)
		if !ok || len(routeChannels) == 0 {
			return fmt.Errorf("billing group %q is not configured for mock load tests", group)
		}
		channelIDs := make([]int, 0, len(routeChannels))
		for _, routeChannel := range routeChannels {
			if routeChannel.Enabled {
				channelIDs = append(channelIDs, routeChannel.ChannelId)
			}
		}
		if len(channelIDs) == 0 {
			return fmt.Errorf("billing group %q has no enabled mock channels", group)
		}
		channels, err := model.GetChannelsByIds(channelIDs)
		if err != nil {
			return fmt.Errorf("load mock channels for billing group %q: %w", group, err)
		}
		if len(channels) != len(channelIDs) {
			return fmt.Errorf("billing group %q contains unavailable channels", group)
		}
		channelsByID := make(map[int]*model.Channel, len(channels))
		for _, channel := range channels {
			channelsByID[channel.Id] = channel
		}
		mockRouteChannels := make([]model.BillingGroupChannel, 0, len(routeChannels))
		for _, routeChannel := range routeChannels {
			channel := channelsByID[routeChannel.ChannelId]
			if routeChannel.Enabled && channel != nil && channel.Status == common.ChannelStatusEnabled && channel.GetSetting().MockLoadTest {
				mockRouteChannels = append(mockRouteChannels, routeChannel)
			}
		}
		if len(mockRouteChannels) < loadTestMockChannelCount {
			return fmt.Errorf("billing group %q requires %d enabled mock channels", group, loadTestMockChannelCount)
		}
		excluded := make(map[int]struct{}, loadTestMockChannelCount)
		for slot := 1; slot <= loadTestMockChannelCount; slot++ {
			selected, err := model.GetConfiguredRouteChannel(group, modelName, requestPath, mockRouteChannels, excluded)
			if err != nil {
				return fmt.Errorf("resolve mock channel %d for billing group %q: %w", slot, group, err)
			}
			if selected == nil || !selected.GetSetting().MockLoadTest {
				return fmt.Errorf("billing group %q requires %d mock channels for model %q and endpoint %q", group, loadTestMockChannelCount, modelName, endpoint)
			}
			excluded[selected.Id] = struct{}{}
		}
	}
	return nil
}

func UpdateLoadTestRunProgress(c *gin.Context) {
	agent, ok := authenticateLoadTestAgentRequest(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, loadTestAgentBodyLimit)
	var request loadTestRunProgressRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid progress payload"})
		return
	}
	if !validLoadTestProgress(request.Sent, request.Completed, request.Successes, request.Failures, request.Dropped, request.CurrentRPS, request.P50MS, request.P95MS, request.P99MS, request.ErrorCounts) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid progress values"})
		return
	}
	errorJSON, err := model.EncodeLoadTestErrorCounts(request.ErrorCounts)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	runID := c.Param("id")
	run, err := model.GetLoadTestRunByID(runID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if run.ExecutionMode == model.LoadTestExecutionShared {
		if !validLoadTestWorkerID(request.WorkerID) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "worker id is required for shared load-test runs"})
			return
		}
		err = model.UpdateLoadTestRunWorkerProgress(runID, request.WorkerID, map[string]any{
			"sent": request.Sent, "completed": request.Completed, "successes": request.Successes,
			"failures": request.Failures, "dropped": request.Dropped, "current_rps": request.CurrentRPS,
			"p50_ms": request.P50MS, "p95_ms": request.P95MS, "p99_ms": request.P99MS,
			"error_counts_json": errorJSON,
		})
	} else {
		err = model.UpdateLoadTestRunProgress(agent.ID, runID, map[string]any{
			"sent": request.Sent, "completed": request.Completed, "successes": request.Successes,
			"failures": request.Failures, "dropped": request.Dropped, "current_rps": request.CurrentRPS,
			"p50_ms": request.P50MS, "p95_ms": request.P95MS, "p99_ms": request.P99MS,
			"error_counts_json": errorJSON,
		})
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func FinishLoadTestRun(c *gin.Context) {
	agent, ok := authenticateLoadTestAgentRequest(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, loadTestAgentBodyLimit)
	var request finishLoadTestRunRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid completion payload"})
		return
	}
	if len(request.ErrorMessage) > loadTestMaxErrorMessage || request.InputTokens < 0 || request.OutputTokens < 0 || request.CacheReadTokens < 0 || request.CacheWriteTokens < 0 ||
		!validLoadTestProgress(request.Sent, request.Completed, request.Successes, request.Failures, request.Dropped, request.CurrentRPS, request.P50MS, request.P95MS, request.P99MS, request.ErrorCounts) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid completion values"})
		return
	}
	errorJSON, err := model.EncodeLoadTestErrorCounts(request.ErrorCounts)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	runID := c.Param("id")
	run, err := model.GetLoadTestRunByID(runID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if run.ExecutionMode == model.LoadTestExecutionShared {
		if !validLoadTestWorkerID(request.WorkerID) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "worker id is required for shared load-test runs"})
			return
		}
		workerStatus := model.LoadTestRunWorkerCompleted
		if request.Status == model.LoadTestRunFailed {
			workerStatus = model.LoadTestRunWorkerFailed
		} else if request.Status == model.LoadTestRunCancelled {
			workerStatus = model.LoadTestRunWorkerCancelled
		}
		err = model.FinishLoadTestRunWorker(runID, request.WorkerID, workerStatus, map[string]any{
			"sent": request.Sent, "completed": request.Completed, "successes": request.Successes,
			"failures": request.Failures, "dropped": request.Dropped, "input_tokens": request.InputTokens,
			"output_tokens": request.OutputTokens, "cache_read_tokens": request.CacheReadTokens,
			"cache_write_tokens": request.CacheWriteTokens, "current_rps": request.CurrentRPS,
			"p50_ms": request.P50MS, "p95_ms": request.P95MS, "p99_ms": request.P99MS,
			"error_counts_json": errorJSON, "error_message": strings.TrimSpace(request.ErrorMessage),
		})
	} else {
		err = model.FinishLoadTestRun(agent.ID, runID, request.Status, map[string]any{
			"sent": request.Sent, "completed": request.Completed, "successes": request.Successes,
			"failures": request.Failures, "dropped": request.Dropped, "input_tokens": request.InputTokens,
			"output_tokens": request.OutputTokens, "cache_read_tokens": request.CacheReadTokens,
			"cache_write_tokens": request.CacheWriteTokens, "current_rps": request.CurrentRPS,
			"p50_ms": request.P50MS, "p95_ms": request.P95MS, "p99_ms": request.P99MS,
			"error_counts_json": errorJSON, "error_message": strings.TrimSpace(request.ErrorMessage),
		})
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func authenticateLoadTestAgentRequest(c *gin.Context) (*model.LoadTestAgent, bool) {
	credential := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(credential), "bearer ") {
		credential = strings.TrimSpace(credential[7:])
	}
	agent, err := model.AuthenticateLoadTestAgent(credential)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return nil, false
	}
	return agent, true
}

func loadTestAgentRuntime(name, platform, version string, cpuCores int, memoryBytes int64, maxRPS, maxConcurrency int) model.LoadTestAgentRuntime {
	return model.LoadTestAgentRuntime{
		Name: name, Platform: platform, Version: version, CPUCores: cpuCores,
		MemoryBytes: memoryBytes, MaxRPS: maxRPS, MaxConcurrency: maxConcurrency,
	}
}

func validLoadTestAgentResources(cpuCores int, memoryBytes int64, maxRPS, maxConcurrency int) bool {
	return cpuCores >= 0 && cpuCores <= 4096 && memoryBytes >= 0 && memoryBytes <= 1<<60 && maxRPS >= 0 && maxRPS <= 1_000_000 && maxConcurrency >= 0 && maxConcurrency <= 1_000_000
}

func validateLoadTestAgentCapacity(agent *model.LoadTestAgent, request createLoadTestRunRequest) error {
	if agent.Managed && (agent.MaxRPS < 1 || agent.MaxConcurrency < 1) {
		return errors.New("managed load-test agent has not reported capacity")
	}
	if agent.MaxRPS > 0 && request.RequestsPerSecond > agent.MaxRPS {
		return fmt.Errorf("load-test agent supports at most %d RPS", agent.MaxRPS)
	}
	if agent.MaxConcurrency > 0 && request.Concurrency > agent.MaxConcurrency {
		return fmt.Errorf("load-test agent supports at most %d concurrent requests", agent.MaxConcurrency)
	}
	return nil
}

func validateLoadTestExecutionSettings(agent *model.LoadTestAgent, request createLoadTestRunRequest) error {
	mode := request.ExecutionMode
	if mode == "" {
		mode = model.LoadTestExecutionSingle
	}
	switch mode {
	case model.LoadTestExecutionSingle:
		if request.ExpectedWorkers != 0 {
			return errors.New("expected workers is only valid for shared mode")
		}
	case model.LoadTestExecutionShared:
		if agent == nil || !agent.Managed {
			return errors.New("shared mode requires a managed load-test agent")
		}
		if request.ExpectedWorkers < loadTestSharedMinWorkers || request.ExpectedWorkers > loadTestSharedMaxWorkers {
			return fmt.Errorf("shared mode requires at least %d and at most %d workers", loadTestSharedMinWorkers, loadTestSharedMaxWorkers)
		}
	default:
		return errors.New("unsupported load-test execution mode")
	}
	return nil
}

func validLoadTestProgress(sent, completed, successes, failures, dropped int64, currentRPS, p50MS, p95MS, p99MS float64, errorCounts map[string]int64) bool {
	if sent < 0 || completed < 0 || successes < 0 || failures < 0 || dropped < 0 || completed > sent || successes+failures > completed {
		return false
	}
	for _, value := range []float64{currentRPS, p50MS, p95MS, p99MS} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	if len(errorCounts) > loadTestMaxErrorKinds {
		return false
	}
	for code, count := range errorCounts {
		if strings.TrimSpace(code) == "" || len(code) > 128 || count < 0 {
			return false
		}
	}
	return true
}

func validLoadTestWorkerID(workerID string) bool {
	workerID = strings.TrimSpace(workerID)
	return workerID != "" && len(workerID) <= 96
}
