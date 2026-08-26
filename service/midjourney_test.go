package service

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMidjourneyChannel(t *testing.T, channelID int, maxConcurrency int) {
	t.Helper()
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	channelConcurrencyStates.Delete(channelID)
	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
	require.NoError(t, model.DB.Where("id = ?", channelID).Delete(&model.Channel{}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:             channelID,
		Name:           "Midjourney test channel",
		Status:         common.ChannelStatusEnabled,
		MaxConcurrency: &maxConcurrency,
	}).Error)
	model.InitChannelCache()
	t.Cleanup(func() {
		channelConcurrencyStates.Delete(channelID)
		_ = model.DB.Where("id = ?", channelID).Delete(&model.Channel{}).Error
		model.InitChannelCache()
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})
}

func beginMidjourneyRoutingSelection(c *gin.Context, channelID int) *RetryParam {
	retryParam := &RetryParam{Ctx: c}
	retryParam.beginRoutingSelection()
	retryParam.finishRoutingSelection(channelID)
	return retryParam
}

func TestDoMidjourneyHttpRequestFailsClosedWhenSelectedChannelIsMissing(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCacheEnabled })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/mj/submit", nil)
	c.Set("channel_id", math.MaxInt32)

	response, body, err := DoMidjourneyHttpRequest(c, time.Second, "http://unused.invalid")

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	assert.Equal(t, "channel_unavailable", response.Response.Description)
	assert.Empty(t, body)
}

func TestDoMidjourneyHttpRequestLeavesRoutingSelectionPendingWhenConcurrencyIsFull(t *testing.T) {
	const channelID = 995001
	setupMidjourneyChannel(t, channelID, 1)
	require.True(t, TryAcquireChannelConcurrency(channelID, 1))
	t.Cleanup(func() { ReleaseChannelConcurrency(channelID) })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/mj/submit", nil)
	c.Set("channel_id", channelID)
	retryParam := beginMidjourneyRoutingSelection(c, channelID)
	t.Cleanup(retryParam.CancelRoutingSelection)

	response, body, err := DoMidjourneyHttpRequest(c, time.Second, "http://unused.invalid")

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusTooManyRequests, response.StatusCode)
	assert.Equal(t, "channel_concurrency_limit", response.Response.Description)
	assert.Empty(t, body)
	assert.NotNil(t, routingScheduleTicketFromContext(c))
	assert.Equal(t, 1, CurrentChannelConcurrency(channelID))
}

func TestDoMidjourneyHttpRequestLeavesRoutingSelectionPendingOnLocalRequestFailure(t *testing.T) {
	const channelID = 995002
	setupMidjourneyChannel(t, channelID, 1)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", strings.NewReader("{"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("channel_id", channelID)
	retryParam := beginMidjourneyRoutingSelection(c, channelID)
	t.Cleanup(retryParam.CancelRoutingSelection)

	response, body, err := DoMidjourneyHttpRequest(c, time.Second, "http://unused.invalid")

	require.Error(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Equal(t, "read_request_body_failed", response.Response.Description)
	assert.Empty(t, body)
	assert.NotNil(t, routingScheduleTicketFromContext(c))
	assert.Equal(t, 0, CurrentChannelConcurrency(channelID))
}

func TestDoMidjourneyHttpRequestCommitsSelectionAndReleasesConcurrencyOnAttempt(t *testing.T) {
	const channelID = 995003
	setupMidjourneyChannel(t, channelID, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"description":"ok","result":"task-1"}`))
	}))
	t.Cleanup(upstream.Close)
	originalHTTPClient := httpClient
	httpClient = upstream.Client()
	t.Cleanup(func() { httpClient = originalHTTPClient })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/mj/submit", nil)
	c.Set("channel_id", channelID)
	retryParam := beginMidjourneyRoutingSelection(c, channelID)
	t.Cleanup(retryParam.CancelRoutingSelection)

	response, body, err := DoMidjourneyHttpRequest(c, time.Second, upstream.URL)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, 1, response.Response.Code)
	assert.NotEmpty(t, body)
	assert.Nil(t, routingScheduleTicketFromContext(c))
	assert.Equal(t, 0, CurrentChannelConcurrency(channelID))
}
