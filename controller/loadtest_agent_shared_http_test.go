package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSharedLoadTestHTTPDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.LoadTestAgent{}, &model.LoadTestRun{}, &model.LoadTestRunWorker{}))
	t.Cleanup(func() {
		model.DB = previousDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func callLoadTestAgentJSON(t *testing.T, handler gin.HandlerFunc, method, path, secret string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := common.Marshal(payload)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(data))
	ctx.Request.Header.Set("Authorization", "Bearer "+secret)
	if marker := strings.Index(path, "/runs/"); marker >= 0 {
		id := strings.TrimSuffix(strings.TrimPrefix(path[marker+len("/runs/"):], "/"), "/progress")
		id = strings.TrimSuffix(id, "/complete")
		ctx.Params = gin.Params{{Key: "id", Value: id}}
	}
	handler(ctx)
	return recorder
}

func TestSharedLoadTestAgentHTTPFlowSplitsAndAggregatesWorkers(t *testing.T) {
	db := setupSharedLoadTestHTTPDB(t)
	gin.SetMode(gin.TestMode)
	agent, err := model.NewManagedLoadTestAgentPairing(1, "SHAREDHTTP", common.GetTimestamp()+300)
	require.NoError(t, err)
	_, err = model.PairLoadTestAgent("SHAREDHTTP", "agent-secret", model.LoadTestAgentRuntime{
		Name: "shared", Platform: "linux", Version: "0.5.0", MaxRPS: 100, MaxConcurrency: 40,
	})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Token{
		Id: 77, UserId: 8, Key: "test-key", Name: "test", Group: "default",
		Status: common.TokenStatusEnabled,
	}).Error)
	run := &model.LoadTestRun{
		UserID: 8, AgentID: agent.ID, TokenID: 77, KeyName: "test", PackageName: "default",
		Model: "gpt-test", Endpoint: "openai", Prompt: "OK", TargetURL: "https://example.com",
		DurationSeconds: 30, RequestsPerSecond: 100, Concurrency: 40,
		ExecutionMode: model.LoadTestExecutionShared, ExpectedWorkers: 2,
	}
	require.NoError(t, model.CreateLoadTestRun(run))

	first := callLoadTestAgentJSON(t, PollLoadTestAgent, http.MethodPost, "/api/loadtest-agent/poll", "agent-secret", map[string]any{
		"worker_id": "worker-a", "name": "a", "platform": "linux", "version": "0.5.0",
	})
	assert.Equal(t, http.StatusOK, first.Code)
	var firstResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Command string `json:"command"`
			Task    struct {
				WorkerID          string `json:"worker_id"`
				RequestsPerSecond int    `json:"requests_per_second"`
				Concurrency       int    `json:"concurrency"`
				ExpectedWorkers   int    `json:"expected_workers"`
			} `json:"task"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(first.Body.Bytes(), &firstResponse))
	assert.True(t, firstResponse.Success)
	assert.Equal(t, "run", firstResponse.Data.Command)
	assert.Equal(t, "worker-a", firstResponse.Data.Task.WorkerID)
	assert.Equal(t, 50, firstResponse.Data.Task.RequestsPerSecond)
	assert.Equal(t, 20, firstResponse.Data.Task.Concurrency)
	assert.Equal(t, 2, firstResponse.Data.Task.ExpectedWorkers)

	second := callLoadTestAgentJSON(t, PollLoadTestAgent, http.MethodPost, "/api/loadtest-agent/poll", "agent-secret", map[string]any{
		"worker_id": "worker-b", "name": "b", "platform": "linux", "version": "0.5.0",
	})
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Contains(t, second.Body.String(), `"worker_id":"worker-b"`)

	progressPayload := map[string]any{
		"worker_id": "worker-a", "sent": 50, "completed": 50, "successes": 49,
		"failures": 1, "dropped": 0, "current_rps": 50,
	}
	progress := callLoadTestAgentJSON(t, UpdateLoadTestRunProgress, http.MethodPost, "/api/loadtest-agent/runs/"+run.ID+"/progress", "agent-secret", progressPayload)
	assert.Equal(t, http.StatusOK, progress.Code)
	progressPayload["worker_id"] = "worker-b"
	progressPayload["successes"] = 50
	progressPayload["failures"] = 0
	progress = callLoadTestAgentJSON(t, UpdateLoadTestRunProgress, http.MethodPost, "/api/loadtest-agent/runs/"+run.ID+"/progress", "agent-secret", progressPayload)
	assert.Equal(t, http.StatusOK, progress.Code)

	finishPayload := map[string]any{
		"worker_id": "worker-a", "status": model.LoadTestRunCompleted,
		"sent": 50, "completed": 50, "successes": 49, "failures": 1,
	}
	finish := callLoadTestAgentJSON(t, FinishLoadTestRun, http.MethodPost, "/api/loadtest-agent/runs/"+run.ID+"/complete", "agent-secret", finishPayload)
	assert.Equal(t, http.StatusOK, finish.Code)
	finishPayload["worker_id"] = "worker-b"
	finishPayload["successes"] = 50
	finishPayload["failures"] = 0
	finish = callLoadTestAgentJSON(t, FinishLoadTestRun, http.MethodPost, "/api/loadtest-agent/runs/"+run.ID+"/complete", "agent-secret", finishPayload)
	assert.Equal(t, http.StatusOK, finish.Code)

	stored, err := model.GetLoadTestRun(8, run.ID)
	require.NoError(t, err)
	assert.Equal(t, model.LoadTestRunCompleted, stored.Status)
	assert.Equal(t, int64(100), stored.Sent)
	assert.Equal(t, int64(99), stored.Successes)
	assert.Equal(t, int64(1), stored.Failures)
}
