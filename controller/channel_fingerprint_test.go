package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	lmfpd "github.com/Ikaleio/lm-detector/sdk/go"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fingerprintTestRequest struct {
	path          string
	authorization string
	model         string
	prompt        string
	messageCount  int
	messageRole   string
}

type fingerprintTestTransport struct {
	mu       sync.Mutex
	status   int
	reply    string
	requests []fingerprintTestRequest
}

func (transport *fingerprintTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != "127.0.0.1" {
		return nil, fmt.Errorf("fingerprint test attempted external request: %s", request.URL.Host)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	requestRecord := fingerprintTestRequest{path: request.URL.Path, authorization: request.Header.Get("Authorization"), model: payload.Model, messageCount: len(payload.Messages)}
	if len(payload.Messages) > 0 {
		requestRecord.prompt = payload.Messages[0].Content
		requestRecord.messageRole = payload.Messages[0].Role
	}
	transport.mu.Lock()
	transport.requests = append(transport.requests, requestRecord)
	transport.mu.Unlock()

	status := transport.status
	if status == 0 {
		status = http.StatusOK
	}
	responseBody := `{"error":{"message":"fingerprint test upstream failure","type":"server_error"}}`
	if status == http.StatusOK {
		responseBody = fmt.Sprintf(`{"id":"fingerprint-test","object":"chat.completion","model":"mapped-model","choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":320,"total_tokens":321}}`, transport.reply)
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(responseBody))}, nil
}

func (transport *fingerprintTestTransport) snapshot() []fingerprintTestRequest {
	transport.mu.Lock()
	defer transport.mu.Unlock()
	requests := make([]fingerprintTestRequest, len(transport.requests))
	copy(requests, transport.requests)
	return requests
}

func fingerprintTestNumberSequence() string {
	var sequence strings.Builder
	for number := 1; number <= 320; number++ {
		if number > 1 {
			sequence.WriteByte(' ')
		}
		sequence.WriteString(fmt.Sprint(number))
	}
	return sequence.String()
}

func setupFingerprintTest(t *testing.T, status int) (*model.Channel, int, *fingerprintTestTransport, chan struct{}) {
	t.Helper()
	_, channel, userID, _, responseTimeUpdated := setupChannelRecoveryControllerDB(t)
	modelMapping := `{"gpt-4o":"mapped-model"}`
	channel.ModelMapping = common.GetPointer(modelMapping)
	channel.NotifyOnStatusChange = common.GetPointer(false)
	require.NoError(t, model.DB.Save(channel).Error)

	client := service.GetHttpClient()
	require.NotNil(t, client)
	previousTransport := client.Transport
	transport := &fingerprintTestTransport{status: status, reply: fingerprintTestNumberSequence()}
	client.Transport = transport
	t.Cleanup(func() { client.Transport = previousTransport })
	return channel, userID, transport, responseTimeUpdated
}

func TestFingerprintChallengesUseRelayMappingAndHeaders(t *testing.T) {
	channel, userID, transport, _ := setupFingerprintTest(t, http.StatusOK)
	challenges, err := lmfpd.GenerateChallenges(3)
	require.NoError(t, err)

	for _, challenge := range challenges {
		result := testChannelWithPromptEngine(
			context.Background(), channel, userID, "gpt-4o", "", false,
			modelTestTokenName, "", challenge.Prompt, channelIQTestEngineFingerprint,
		)
		require.NoError(t, result.localErr)
		require.NotEmpty(t, result.response)
	}

	requests := transport.snapshot()
	require.Len(t, requests, 3)
	for _, request := range requests {
		assert.Equal(t, "/v1/chat/completions", request.path)
		assert.Equal(t, "Bearer first", request.authorization)
		assert.Equal(t, "mapped-model", request.model)
		assert.Equal(t, 1, request.messageCount)
		assert.Equal(t, "user", request.messageRole)
		assert.NotEmpty(t, request.prompt)
		assert.NotContains(t, strings.ToLower(request.prompt), "pelican")
		assert.NotContains(t, strings.ToLower(request.prompt), "html")
		assert.NotContains(t, strings.ToLower(request.prompt), "svg")
	}
}

func TestFingerprintRequestOmitsPelicanOutputContract(t *testing.T) {
	for _, endpointType := range []string{"openai", "openai-response"} {
		t.Run(endpointType, func(t *testing.T) {
			request := buildTestRequestWithIQEngine(
				"gpt-4o", endpointType, nil, false,
				"choose a sequence of integers", channelIQTestEngineFingerprint,
			)
			body, err := common.Marshal(request)
			require.NoError(t, err)
			assert.NotContains(t, strings.ToLower(string(body)), "pelican")
			assert.NotContains(t, strings.ToLower(string(body)), "html")
			assert.NotContains(t, strings.ToLower(string(body)), "svg")

			var payload map[string]any
			require.NoError(t, common.Unmarshal(body, &payload))
			if endpointType == "openai" {
				messages, ok := payload["messages"].([]any)
				require.True(t, ok)
				require.Len(t, messages, 1)
			} else {
				input, ok := payload["input"].([]any)
				require.True(t, ok)
				require.Len(t, input, 1)
				assert.NotContains(t, payload, "instructions")
			}
		})
	}
}

func TestFingerprintTestPropagatesCancellation(t *testing.T) {
	channel, userID, transport, _ := setupFingerprintTest(t, http.StatusOK)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := testChannelWithFingerprint(ctx, channel, userID, "gpt-4o", "", modelTestTokenName)
	require.Error(t, result.localErr)
	assert.ErrorIs(t, result.localErr, context.Canceled)
	assert.Empty(t, transport.snapshot())
}

func TestFingerprintTestPropagatesUpstreamFailure(t *testing.T) {
	channel, userID, transport, _ := setupFingerprintTest(t, http.StatusServiceUnavailable)
	channel.Status, channel.IQProbeEnabled = common.ChannelStatusEnabled, true
	require.NoError(t, model.DB.Save(channel).Error)
	require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", true).Error)
	now := common.GetTimestamp()
	before := model.ChannelProbeState{ChannelID: channel.Id, IQLeaseUntil: now + 300, LeaseUntil: now + 220, LastSuccess: true}
	require.NoError(t, model.DB.Create(&before).Error)

	result := testChannelWithPromptEngine(
		context.Background(), channel, userID, "gpt-4o", "", false,
		modelTestTokenName, "", "generate an integer sequence", channelIQTestEngineFingerprint,
	)
	require.Error(t, result.localErr)
	assert.Equal(t, http.StatusServiceUnavailable, result.upstreamFailureStatus)
	assert.NotEmpty(t, transport.snapshot())
	completed := completeFingerprintProbe(context.Background(), channel.Id, channel.GetTestModel(), before.IQLeaseUntil, result, 123*time.Millisecond)
	require.NoError(t, completed.err)
	assert.Equal(t, channelProbeSummary{Checked: 1, Failed: 1, Disabled: 1}, completed.summary)
	var history []model.ChannelProbeHistory
	require.NoError(t, model.DB.Where("channel_id = ?", channel.Id).Find(&history).Error)
	require.Len(t, history, 1)
	assert.False(t, history[0].Success)
	assert.Equal(t, http.StatusServiceUnavailable, history[0].StatusCode)
	assert.Equal(t, int64(123), history[0].LatencyMs)
	state, err := model.GetChannelProbeState(channel.Id)
	require.NoError(t, err)
	assert.Equal(t, history[0].CheckedAt, state.LastIQProbeAt)
	assert.Zero(t, state.IQLeaseUntil)
	assert.Equal(t, before.LeaseUntil, state.LeaseUntil)
	var stored model.Channel
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	var ability model.Ability
	require.NoError(t, model.DB.First(&ability, "channel_id = ?", channel.Id).Error)
	assert.False(t, ability.Enabled)
}

func TestFingerprintProbeWithoutCalibratedResultPreservesAvailability(t *testing.T) {
	inconclusive := channelProbeSummary{Checked: 1, Inconclusive: 1}
	for _, tc := range []struct {
		name    string
		status  int
		result  testResult
		summary channelProbeSummary
	}{
		{"detector error", common.ChannelStatusEnabled, testResult{localErr: errors.New("detector data unavailable")}, channelProbeSummary{Checked: 1, Failed: 1}},
		{"partial ranking", common.ChannelStatusAutoDisabled, testResult{fingerprintAnalysis: &lmfpd.Analysis{Decision: "partial", Prediction: "gpt-4o"}}, inconclusive},
		{"unscorable", common.ChannelStatusAutoDisabled, testResult{fingerprintAnalysis: &lmfpd.Analysis{Decision: "unscorable"}}, inconclusive},
		{"non model", common.ChannelStatusAutoDisabled, testResult{fingerprintAnalysis: &lmfpd.Analysis{Decision: "not_model"}}, inconclusive},
		{"uncalibrated ranking", common.ChannelStatusAutoDisabled, testResult{fingerprintAnalysis: &lmfpd.Analysis{Decision: "scored", Prediction: "gpt-4o", ProbabilityStatus: "unavailable"}}, inconclusive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, channel, _, _, _ := setupChannelRecoveryControllerDB(t)
			channel.Status, channel.IQProbeEnabled, channel.IQAutoDisableDegraded = tc.status, true, true
			channel.NotifyOnStatusChange = common.GetPointer(false)
			channel.OtherInfo = `{"iq_probe_at":123,"iq_probe_prediction":"gpt-4o","iq_probe_degraded":false,"iq_probe_model_confidences":{"gpt-4o":0.82}}`
			require.NoError(t, db.Save(channel).Error)
			enabled := tc.status == common.ChannelStatusEnabled
			require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", enabled).Error)
			now := common.GetTimestamp()
			before := model.ChannelProbeState{ChannelID: channel.Id, LastProbeAt: now - 60, LastSuccess: true, LastLatencyMs: 42, LeaseUntil: now + 220, LastIQProbeAt: now - 3600, IQLeaseUntil: now + 300}
			require.NoError(t, db.Create(&before).Error)

			result := completeFingerprintProbe(context.Background(), channel.Id, channel.GetTestModel(), before.IQLeaseUntil, tc.result, 123*time.Millisecond)
			require.NoError(t, result.err)
			assert.Equal(t, tc.summary, result.summary)
			state, err := model.GetChannelProbeState(channel.Id)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, state.LastIQProbeAt, now)
			assert.Zero(t, state.IQLeaseUntil)
			assert.Equal(t, before.LastProbeAt, state.LastProbeAt)
			assert.Equal(t, before.LastSuccess, state.LastSuccess)
			assert.Equal(t, before.LastLatencyMs, state.LastLatencyMs)
			assert.Equal(t, before.LeaseUntil, state.LeaseUntil)
			var count int64
			require.NoError(t, db.Model(&model.ChannelProbeHistory{}).Where("channel_id = ?", channel.Id).Count(&count).Error)
			assert.Zero(t, count)
			var stored model.Channel
			require.NoError(t, db.First(&stored, channel.Id).Error)
			assert.Equal(t, tc.status, stored.Status)
			assert.JSONEq(t, channel.OtherInfo, stored.OtherInfo)
			var ability model.Ability
			require.NoError(t, db.First(&ability, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, enabled, ability.Enabled)
		})
	}
}

func TestManualIQTestEngineSelection(t *testing.T) {
	for _, tc := range []struct{ name, query string }{
		{"default pelican", ""}, {"explicit pelican", "&iq_test_engine=pelican"}, {"unknown", "&iq_test_engine=unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			channel, userID, transport, responseTimeUpdated := setupFingerprintTest(t, http.StatusOK)
			// A previous healthy fingerprint must not recover an auto-disabled channel on a Pelican run.
			channel.OtherInfo = `{"status_reason":"IQ probe detected model degradation","iq_probe_at":123,"iq_probe_prediction":"gpt-4o","iq_probe_degraded":false,"iq_probe_model_confidences":{"gpt-4o":0.82}}`
			require.NoError(t, model.DB.Save(channel).Error)
			transport.reply = `<html><body><svg><circle /><animate attributeName="x" values="0;10" dur="1s" repeatCount="indefinite" /></svg></body></html>`
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/test?iq_test=true"+tc.query, nil)
			ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(channel.Id)}}
			ctx.Set("id", userID)
			TestChannel(ctx)
			var response struct {
				Success bool
				Message string
				Data    struct{ Engine, Response string }
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, http.StatusOK, recorder.Code)
			if tc.name == "unknown" {
				assert.False(t, response.Success)
				assert.Contains(t, response.Message, "无效的降智测试引擎")
				assert.Empty(t, transport.snapshot())
				return
			}
			require.True(t, response.Success, response.Message)
			waitChannelRecoveryEffect(t, responseTimeUpdated)
			assert.Equal(t, "pelican", response.Data.Engine)
			assert.Equal(t, transport.reply, response.Data.Response)
			require.Len(t, transport.snapshot(), 1)
			var stored model.Channel
			require.NoError(t, model.DB.First(&stored, channel.Id).Error)
			assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
			assert.JSONEq(t, channel.OtherInfo, stored.OtherInfo)
		})
	}
}

func TestFingerprintManualAndScheduledResults(t *testing.T) {
	dataDir := os.Getenv("TEST_LMFPD_DATA_DIR")
	if dataDir == "" {
		t.Skip("set TEST_LMFPD_DATA_DIR to a local calibrated detector bundle for the offline SDK integration test")
	}
	channel, userID, transport, responseTimeUpdated := setupFingerprintTest(t, http.StatusOK)
	dataDir, err := filepath.Abs(dataDir)
	require.NoError(t, err)
	detector, err := lmfpd.Load(t.Context(), lmfpd.LoadOptions{Sources: []string{dataDir}})
	require.NoError(t, err)
	fingerprintDetectorState.Lock()
	previousDetector := fingerprintDetectorState.detector
	fingerprintDetectorState.detector = detector
	fingerprintDetectorState.Unlock()
	t.Cleanup(func() {
		fingerprintDetectorState.Lock()
		fingerprintDetectorState.detector = previousDetector
		fingerprintDetectorState.Unlock()
	})
	require.NoError(t, model.LOG_DB.AutoMigrate(&model.Log{}))
	previousExport := common.DataExportEnabled
	common.DataExportEnabled, common.LogConsumeEnabled = false, true
	t.Cleanup(func() { common.DataExportEnabled = previousExport })
	channel.Status, channel.IQProbeEnabled, channel.IQProbeSampleCount = common.ChannelStatusEnabled, true, 1
	channel.OtherInfo = `{"iq_probe_at":123,"iq_probe_prediction":"gpt-4o","iq_probe_model_confidences":{"gpt-4o":0.82}}`
	require.NoError(t, model.DB.Save(channel).Error)
	transport.reply = strings.Repeat("27 139 264 75 316 188 43 251 ", 36)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/channel/test?iq_test=true&iq_test_engine=fingerprint", nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(channel.Id)}}
	c.Set("id", userID)
	TestChannel(c)
	var response struct {
		Success bool
		Message string
		Data    struct {
			Engine   string
			Analysis lmfpd.Analysis
		}
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	waitChannelRecoveryEffect(t, responseTimeUpdated)
	assert.Equal(t, "fingerprint", response.Data.Engine)
	assert.Equal(t, "partial", response.Data.Analysis.Decision)
	assert.Nil(t, response.Data.Analysis.Probability)
	var stored model.Channel
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	assert.JSONEq(t, channel.OtherInfo, stored.OtherInfo)

	require.NoError(t, model.DB.Model(channel).Update("iq_probe_sample_count", 3).Error)
	completed := runChannelIQProbe(t.Context(), channel.Id, userID)
	require.NoError(t, completed.err)
	assert.Equal(t, channelProbeSummary{Checked: 1, Succeeded: 1}, completed.summary)
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	model.EnrichChannelIQProbeMetrics([]*model.Channel{&stored})
	assert.Greater(t, stored.IQProbeAt, int64(123))
	assert.NotEmpty(t, stored.IQProbeModelConfidences)
	var sources []string
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("channel_id = ? AND type = ?", channel.Id, model.LogTypeConsume).Order("id").Pluck("token_name", &sources).Error)
	assert.Equal(t, []string{modelTestTokenName, channelProbeTokenName, channelProbeTokenName, channelProbeTokenName}, sources)
}

func TestCopiedChannelStartsWithoutFingerprintHistory(t *testing.T) {
	db, source, userID, _, _ := setupChannelRecoveryControllerDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	source.IQProbeEnabled, source.IQProbeSampleCount = true, 2
	source.OtherInfo = `{"provider":{"id":9007199254740993},"iq_probe_engine":"fingerprint","iq_probe_test_model":"gpt-4o","iq_probe_at":123,"iq_probe_prediction":"other","iq_probe_confidence":0.99,"iq_probe_degraded":true,"iq_probe_model_confidences":{"other":0.99}}`
	require.NoError(t, db.Save(source).Error)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/channel/copy", nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(source.Id)}}
	c.Set("id", userID)
	CopyChannel(c)
	var response struct {
		Success bool
		Message string
		Data    struct{ ID int }
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	var clone model.Channel
	require.NoError(t, db.First(&clone, response.Data.ID).Error)
	assert.True(t, clone.IQProbeEnabled)
	assert.Equal(t, source.IQProbeSampleCount, clone.IQProbeSampleCount)
	assert.Contains(t, clone.OtherInfo, "9007199254740993")
	assert.NotContains(t, clone.OtherInfo, "iq_probe_")
	model.EnrichChannelIQProbeMetrics([]*model.Channel{&clone})
	require.NotNil(t, clone.IQTestConfidence)
	assert.Equal(t, float64(1), *clone.IQTestConfidence)
	assert.Zero(t, clone.IQProbeAt)
	assert.False(t, clone.IQProbeDegraded())
	var original model.Channel
	require.NoError(t, db.First(&original, source.Id).Error)
	assert.Equal(t, source.OtherInfo, original.OtherInfo)
}
