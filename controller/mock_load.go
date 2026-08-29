package controller

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	projectcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const mockLoadTestOutputTokens = 8

// executeInternalMockLoadTest is intentionally independent of channel
// selection. It never initializes ChannelMeta, reads channel settings, or
// invokes a relay handler, so a mock run cannot reach an IKUN/other provider
// even when the database contains real channel URLs and keys.
func executeInternalMockLoadTest(c *gin.Context, info *relaycommon.RelayInfo, relayFormat types.RelayFormat, promptTokens int) *types.NewAPIError {
	channelsJSON := strings.TrimSpace(c.GetHeader("X-Alltoken-Mock-Channels"))
	failureRate := parseMockFloatHeader(c, "X-Alltoken-Mock-Failure-Rate")
	failureStatus := parseMockIntHeader(c, "X-Alltoken-Mock-Failure-Status")
	latencyMS := parseMockIntHeader(c, "X-Alltoken-Mock-Latency-Ms")
	if err := service.VerifyMockLoadTestRequest(c, channelsJSON, failureRate, failureStatus, latencyMS); err != nil {
		if apiErr, ok := err.(*types.NewAPIError); ok {
			return apiErr
		}
		return types.NewErrorWithStatusCode(err, types.ErrorCodeAccessDenied, http.StatusForbidden, types.ErrOptionWithSkipRetry())
	}
	// Set the marker before simulating failures as well as successes. The
	// agent uses it as a hard guard that the request stayed inside alltoken.
	c.Header("X-Alltoken-Mock-Executed", "true")
	c.Header("X-Alltoken-Mock-Upstream", "internal")

	channels := make([]model.LoadTestMockChannel, 0, 3)
	if channelsJSON != "" && channelsJSON != "null" && channelsJSON != "[]" {
		if err := projectcommon.UnmarshalJsonStr(channelsJSON, &channels); err != nil || len(channels) == 0 {
			return types.NewErrorWithStatusCode(errors.New("invalid mock channel configuration"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
	} else {
		channels = append(channels, model.LoadTestMockChannel{Slot: 1, FailureRate: failureRate, FailureStatus: failureStatus, LatencyMS: latencyMS})
	}
	if len(channels) > 3 {
		return types.NewErrorWithStatusCode(errors.New("too many mock channel configurations"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].Slot < channels[j].Slot })

	requestID := c.GetString(projectcommon.RequestIdKey)
	if requestID == "" {
		requestID = c.GetHeader("X-Load-Test-ID")
	}
	if requestID == "" {
		requestID = c.GetHeader(constant.MockLoadTestRunHeader)
	}
	seenSlots := make(map[int]struct{}, len(channels))
	for index, channel := range channels {
		if channel.Slot < 1 || channel.Slot > 3 || channel.FailureRate < 0 || channel.FailureRate > 1 || math.IsNaN(channel.FailureRate) || math.IsInf(channel.FailureRate, 0) || channel.LatencyMS < 0 {
			return types.NewErrorWithStatusCode(errors.New("invalid mock channel configuration"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if _, exists := seenSlots[channel.Slot]; exists {
			return types.NewErrorWithStatusCode(errors.New("duplicate mock channel slot"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		seenSlots[channel.Slot] = struct{}{}
		if channel.FailureStatus != 0 && channel.FailureStatus != http.StatusTooManyRequests && channel.FailureStatus != http.StatusInternalServerError && channel.FailureStatus != http.StatusBadGateway && channel.FailureStatus != http.StatusServiceUnavailable && channel.FailureStatus != http.StatusGatewayTimeout {
			return types.NewErrorWithStatusCode(errors.New("invalid mock failure status"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if channel.LatencyMS > 120000 {
			return types.NewErrorWithStatusCode(errors.New("mock latency exceeds limit"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if channel.LatencyMS > 0 {
			time.Sleep(time.Duration(channel.LatencyMS) * time.Millisecond)
		}
		if mockFailureForRequest(requestID, channel.Slot, channel.FailureRate) {
			status := channel.FailureStatus
			if status == 0 {
				status = http.StatusServiceUnavailable
			}
			if index == len(channels)-1 {
				return types.NewErrorWithStatusCode(fmt.Errorf("mock slot %d failed with status %d", channel.Slot, status), types.ErrorCodeUpstreamExhausted, status, types.ErrOptionWithSkipRetry())
			}
			continue
		}
		writeInternalMockSuccess(c, info, relayFormat, promptTokens, channel.Slot)
		return nil
	}
	return types.NewErrorWithStatusCode(errors.New("mock load-test has no executable slots"), types.ErrorCodeUpstreamExhausted, http.StatusServiceUnavailable, types.ErrOptionWithSkipRetry())
}

func parseMockFloatHeader(c *gin.Context, name string) float64 {
	value := strings.TrimSpace(c.GetHeader(name))
	if value == "" {
		return 0
	}
	var parsed float64
	if _, err := fmt.Sscanf(value, "%f", &parsed); err != nil {
		return math.NaN()
	}
	return parsed
}

func parseMockIntHeader(c *gin.Context, name string) int {
	value := strings.TrimSpace(c.GetHeader(name))
	if value == "" {
		return 0
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return -1
	}
	return parsed
}

func mockFailureForRequest(requestID string, slot int, rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", requestID, slot)))
	value := float64(binary.BigEndian.Uint64(sum[:8])) / float64(^uint64(0))
	return value < rate
}

func writeInternalMockSuccess(c *gin.Context, info *relaycommon.RelayInfo, relayFormat types.RelayFormat, promptTokens, slot int) {
	if promptTokens < 0 {
		promptTokens = 0
	}
	completionTokens := mockLoadTestOutputTokens
	totalTokens := promptTokens + completionTokens
	responseID := fmt.Sprintf("mock-%s-%d", c.GetString(projectcommon.RequestIdKey), slot)
	modelName := info.OriginModelName
	c.Header("X-Alltoken-Mock-Executed", "true")
	c.Header("X-Alltoken-Mock-Slot", fmt.Sprintf("%d", slot))
	c.Header("X-Alltoken-Mock-Upstream", "internal")
	usage := map[string]any{
		"prompt_tokens": promptTokens, "completion_tokens": completionTokens, "total_tokens": totalTokens,
		"input_tokens": promptTokens, "output_tokens": completionTokens,
	}
	if relayFormat == types.RelayFormatClaude {
		body := gin.H{"id": responseID, "type": "message", "role": "assistant", "model": modelName,
			"content": []gin.H{{"type": "text", "text": "mock response"}}, "stop_reason": "end_turn",
			"usage": gin.H{"input_tokens": promptTokens, "output_tokens": completionTokens}}
		writeMockResponse(c, body, relayFormat)
		return
	}
	if relayFormat == types.RelayFormatOpenAIResponses || relayFormat == types.RelayFormatOpenAIResponsesCompaction {
		body := gin.H{"id": responseID, "object": "response", "created_at": time.Now().Unix(), "status": "completed",
			"model": modelName, "output": []gin.H{{"type": "message", "id": responseID + "-msg", "role": "assistant", "content": []gin.H{{"type": "output_text", "text": "mock response", "annotations": []any{}}}}}, "usage": usage}
		writeMockResponse(c, body, relayFormat)
		return
	}
	body := gin.H{"id": responseID, "object": "chat.completion", "created": time.Now().Unix(), "model": modelName,
		"choices": []gin.H{{"index": 0, "message": gin.H{"role": "assistant", "content": "mock response"}, "finish_reason": "stop"}}, "usage": usage}
	writeMockResponse(c, body, relayFormat)
}

func writeMockResponse(c *gin.Context, body gin.H, relayFormat types.RelayFormat) {
	if !projectcommon.GetContextKeyBool(c, constant.ContextKeyIsStream) {
		c.JSON(http.StatusOK, body)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	data, err := projectcommon.Marshal(body)
	if err != nil {
		return
	}
	if relayFormat == types.RelayFormatClaude {
		_, _ = c.Writer.Write([]byte("event: message_start\ndata: " + string(data) + "\n\n"))
	} else {
		_, _ = c.Writer.Write([]byte("data: " + string(data) + "\n\ndata: [DONE]\n\n"))
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
