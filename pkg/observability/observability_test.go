package observability

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderFromBaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "ikun https", url: "https://ikun.love/v1", want: ProviderIkun},
		{name: "ikun subdomain", url: "https://api.ikun.love", want: ProviderIkun},
		{name: "case and trailing dot", url: "https://IKUN.LOVE./v1", want: ProviderIkun},
		{name: "unknown", url: "https://example.com/v1", want: ProviderOther},
		{name: "invalid", url: "not a url", want: ProviderOther},
		{name: "empty", url: "", want: ProviderOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ProviderFromBaseURL(tt.url))
		})
	}
}

func TestErrorClass(t *testing.T) {
	tests := []struct {
		name       string
		apiErr     *types.NewAPIError
		contextErr error
		want       string
	}{
		{name: "success", want: ErrorSuccess},
		{name: "successful response ignores late cancellation", contextErr: context.Canceled, want: ErrorSuccess},
		{name: "cancelled", apiErr: types.NewError(context.Canceled, types.ErrorCodeDoRequestFailed), contextErr: context.Canceled, want: ErrorClientCancelled},
		{name: "deadline", apiErr: types.NewError(context.DeadlineExceeded, types.ErrorCodeDoRequestFailed), contextErr: context.DeadlineExceeded, want: ErrorResponseTimeout},
		{name: "quota", apiErr: types.NewError(errors.New("quota"), types.ErrorCodeInsufficientUserQuota), want: ErrorQuota},
		{name: "no channel", apiErr: types.NewError(errors.New("none"), types.ErrorCodeGetChannelFailed), want: ErrorNoChannel},
		{name: "parse", apiErr: types.NewError(errors.New("bad json"), types.ErrorCodeReadResponseBodyFailed), want: ErrorParse},
		{name: "upstream pending", apiErr: types.NewOpenAIError(errors.New("account wait queue full"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests), want: ErrorUpstreamPendingLimit},
		{name: "upstream rate limit", apiErr: types.NewOpenAIError(errors.New("rate limited"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests), want: ErrorUpstreamRateLimit},
		{name: "upstream 4xx", apiErr: types.NewOpenAIError(errors.New("bad request"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest), want: ErrorUpstream4xx},
		{name: "upstream 5xx", apiErr: types.NewOpenAIError(errors.New("bad gateway"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway), want: ErrorUpstream5xx},
		{name: "connect timeout", apiErr: types.NewError(errors.New("dial connect timeout"), types.ErrorCodeDoRequestFailed), want: ErrorConnectTimeout},
		{name: "response timeout", apiErr: types.NewError(errors.New("response deadline exceeded"), types.ErrorCodeDoRequestFailed), want: ErrorResponseTimeout},
		{name: "arbitrary becomes internal", apiErr: types.NewError(errors.New("user-controlled-random-value"), types.ErrorCode("random-code")), want: ErrorInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ErrorClass(tt.apiErr, tt.contextErr))
		})
	}
}

func TestCollectorLabelsAreBounded(t *testing.T) {
	for _, collector := range Collectors() {
		descCh := make(chan *prometheus.Desc, 10)
		collector.Describe(descCh)
		close(descCh)
		for desc := range descCh {
			description := desc.String()
			for _, forbidden := range []string{
				"request_id",
				"client_trace_id",
				"upstream_request_id",
				"channel_name",
				"url",
				"error_message",
			} {
				assert.False(t, strings.Contains(description, forbidden), description)
			}
		}
	}
}

func TestNormalizeErrorClassIsBounded(t *testing.T) {
	assert.Equal(t, ErrorSuccess, normalizeErrorClass(ErrorSuccess))
	assert.Equal(t, ErrorClientCancelled, normalizeErrorClass(ErrorClientCancelled))
	assert.Equal(t, ErrorInternal, normalizeErrorClass("arbitrary-user-controlled-class"))
}

func TestNormalizeReasonIsBounded(t *testing.T) {
	assert.Equal(t, ErrorUpstream5xx, normalizeReason(ErrorUpstream5xx))
	assert.Equal(t, ErrorInternal, normalizeReason("arbitrary-user-controlled-reason"))
}

func TestAuthFailureReasonIsBounded(t *testing.T) {
	assert.Equal(t, "token_expired", normalizeAuthFailureReason("token_expired"))
	assert.Equal(t, "token_invalid", normalizeAuthFailureReason("raw-token-value"))
}

func TestRecordAuthFailureUsesBoundedLabels(t *testing.T) {
	RecordAuthFailure("/v1/chat/completions", "token_not_found", http.StatusUnauthorized)
	registry := prometheus.NewRegistry()
	for _, collector := range Collectors() {
		registry.MustRegister(collector)
	}
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range metricFamilies {
		if family.GetName() != "alltoken_auth_failures_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			assert.Equal(t, "token_not_found", labels["reason"])
			assert.Equal(t, "/v1/chat/completions", labels["route"])
			assert.Equal(t, "401", labels["status"])
			return
		}
	}
	t.Fatal("alltoken_auth_failures_total metric was not gathered")
}

func TestLogEventSkipsStructuredLogsWhenMonitoringIsDisabled(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	previousEventLogEnabled := common.ObservabilityEventLogEnabled
	previousWriter := gin.DefaultWriter
	common.MonitoringEnabled = false
	common.ObservabilityEventLogEnabled = true
	t.Cleanup(func() {
		common.MonitoringEnabled = previousMonitoringEnabled
		common.ObservabilityEventLogEnabled = previousEventLogEnabled
		gin.DefaultWriter = previousWriter
	})

	var output bytes.Buffer
	gin.DefaultWriter = &output
	LogEvent(context.Background(), Event{
		Event:      "request_finished",
		Provider:   ProviderOther,
		Route:      "/v1/responses",
		Status:     200,
		DurationMS: 10,
	})

	assert.Empty(t, output.String())
}

func TestLogEventSkipsStructuredLogsWhenEventLoggingIsDisabled(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	previousEventLogEnabled := common.ObservabilityEventLogEnabled
	previousWriter := gin.DefaultWriter
	common.MonitoringEnabled = true
	common.ObservabilityEventLogEnabled = false
	t.Cleanup(func() {
		common.MonitoringEnabled = previousMonitoringEnabled
		common.ObservabilityEventLogEnabled = previousEventLogEnabled
		gin.DefaultWriter = previousWriter
	})

	var output bytes.Buffer
	gin.DefaultWriter = &output
	LogEvent(context.Background(), Event{Event: "upstream_attempt", Provider: ProviderOther})

	assert.Empty(t, output.String())
}
