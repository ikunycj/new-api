package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLoadTestChannelStatsRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader("{"))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "invalid request_ids", response.Message)
}

func TestGetLoadTestChannelStatsRejectsTooManyRequestIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestIDs := strings.Repeat(`"request",`, maxLoadTestStatsRequestIDs) + `"request"`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/log/self/loadtest-stats", strings.NewReader(`{"request_ids":[`+requestIDs+`]}`))

	GetLoadTestChannelStats(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "too many request_ids", response.Message)
}
