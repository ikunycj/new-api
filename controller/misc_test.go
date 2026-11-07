package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statusReleaseResponse struct {
	Success bool `json:"success"`
	Data    struct {
		BuildCommit     string `json:"build_commit"`
		BuildRelease    string `json:"build_release"`
		BuildTime       string `json:"build_time"`
		CustomerService string `json:"customer_service"`
	} `json:"data"`
}

func TestGetStatusExposesBuildMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldCommit, oldRelease, oldTime := common.BuildCommit, common.BuildRelease, common.BuildTime
	common.BuildCommit, common.BuildRelease, common.BuildTime = "abc123", "abc123-v1", "2026-08-02T00:00:00Z"
	t.Cleanup(func() {
		common.BuildCommit, common.BuildRelease, common.BuildTime = oldCommit, oldRelease, oldTime
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	GetStatus(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response statusReleaseResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, "abc123", response.Data.BuildCommit)
	assert.Equal(t, "abc123-v1", response.Data.BuildRelease)
	assert.Equal(t, "2026-08-02T00:00:00Z", response.Data.BuildTime)
}

func TestGetStatusExposesCustomerService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"CustomerService": "Email: support@example.com\nHours: 09:00-18:00",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	GetStatus(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response statusReleaseResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "Email: support@example.com\nHours: 09:00-18:00", response.Data.CustomerService)
}
