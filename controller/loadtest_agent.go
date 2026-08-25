package controller

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
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
)

var loadTestEndpoints = map[string]struct{}{
	"anthropic":               {},
	"openai":                  {},
	"openai-response":         {},
	"openai-response-compact": {},
}

type pairLoadTestAgentRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

type createLoadTestRunRequest struct {
	AgentID           string `json:"agent_id"`
	TokenID           int    `json:"token_id"`
	Model             string `json:"model"`
	Endpoint          string `json:"endpoint"`
	Prompt            string `json:"prompt"`
	PromptCache       bool   `json:"prompt_cache"`
	DurationSeconds   int    `json:"duration_seconds"`
	RequestsPerSecond int    `json:"requests_per_second"`
	Concurrency       int    `json:"concurrency"`
}

type loadTestAgentPollRequest struct {
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	CurrentRunID string `json:"current_run_id"`
}

type loadTestRunProgressRequest struct {
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

func ListLoadTestAgents(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	agents, err := model.ListLoadTestAgents(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"agents": agents, "online_before": common.GetTimestamp() - loadTestAgentOnlineSeconds})
}

func GetLoadTestState(c *gin.Context) {
	if !requireLoadTestUser(c) {
		return
	}
	agents, err := model.ListLoadTestAgents(c.GetInt("id"))
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
		"agents":        agents,
		"online_before": common.GetTimestamp() - loadTestAgentOnlineSeconds,
		"runs":          runs,
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
	if len(request.Name) < 1 || len(request.Name) > 128 || len(request.Platform) > 64 || len(request.Version) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid agent metadata"})
		return
	}
	secret, err := common.GenerateRandomCharsKey(48)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	agent, err := model.PairLoadTestAgent(request.Code, secret, request.Name, request.Platform, request.Version)
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
	agent, err := model.GetLoadTestAgent(c.GetInt("id"), request.AgentID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if agent.LastSeenAt < common.GetTimestamp()-loadTestAgentOnlineSeconds {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "load-test agent is offline"})
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
	run := &model.LoadTestRun{
		UserID: c.GetInt("id"), AgentID: agent.ID, TokenID: token.Id,
		KeyName: token.Name, PackageName: packageName, Model: request.Model,
		Endpoint: request.Endpoint, Prompt: request.Prompt, PromptCache: request.PromptCache,
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
	if err := model.TouchLoadTestAgent(agent.ID, request.Name, request.Platform, request.Version); err != nil {
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
	if strings.TrimSpace(request.CurrentRunID) != "" {
		common.ApiSuccess(c, gin.H{"command": "wait"})
		return
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
			"requests_per_second": run.RequestsPerSecond, "concurrency": run.Concurrency,
		},
	})
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
	err = model.UpdateLoadTestRunProgress(agent.ID, c.Param("id"), map[string]any{
		"sent": request.Sent, "completed": request.Completed, "successes": request.Successes,
		"failures": request.Failures, "dropped": request.Dropped, "current_rps": request.CurrentRPS,
		"p50_ms": request.P50MS, "p95_ms": request.P95MS, "p99_ms": request.P99MS,
		"error_counts_json": errorJSON,
	})
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
	err = model.FinishLoadTestRun(agent.ID, c.Param("id"), request.Status, map[string]any{
		"sent": request.Sent, "completed": request.Completed, "successes": request.Successes,
		"failures": request.Failures, "dropped": request.Dropped, "input_tokens": request.InputTokens,
		"output_tokens": request.OutputTokens, "cache_read_tokens": request.CacheReadTokens,
		"cache_write_tokens": request.CacheWriteTokens, "current_rps": request.CurrentRPS,
		"p50_ms": request.P50MS, "p95_ms": request.P95MS, "p99_ms": request.P99MS,
		"error_counts_json": errorJSON, "error_message": strings.TrimSpace(request.ErrorMessage),
	})
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
