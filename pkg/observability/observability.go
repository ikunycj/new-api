package observability

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ProviderOther = "other"

	ContextErrorClassKey = "observability_error_class"

	ErrorSuccess              = "success"
	ErrorAuth                 = "auth"
	ErrorUserRateLimit        = "user_rate_limit"
	ErrorQuota                = "quota"
	ErrorNoChannel            = "no_channel"
	ErrorUpstreamPendingLimit = "upstream_pending_limit"
	ErrorUpstreamRateLimit    = "upstream_rate_limit"
	ErrorUpstream4xx          = "upstream_4xx"
	ErrorUpstream5xx          = "upstream_5xx"
	ErrorConnectTimeout       = "connect_timeout"
	ErrorResponseTimeout      = "response_timeout"
	ErrorClientCancelled      = "client_cancelled"
	ErrorParse                = "parse_error"
	ErrorInternal             = "internal"
)

var (
	relayRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "requests_total",
		Help: "Final relay request outcomes.",
	}, []string{"provider", "channel_id", "outcome", "error_class", "final_status"})
	relayRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "request_duration_seconds",
		Help:    "Relay request duration from entry to final outcome.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"provider", "channel_id", "outcome"})
	relayAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "attempts_total",
		Help: "Individual upstream relay attempt outcomes.",
	}, []string{"provider", "channel_id", "outcome", "error_class", "upstream_status"})
	relayAttemptDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "attempt_duration_seconds",
		Help:    "Individual upstream relay attempt duration.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"provider", "channel_id", "outcome"})
	relayRetries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "retries_total",
		Help: "Relay retries by bounded reason.",
	}, []string{"provider", "channel_id", "reason"})
	relayInFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "in_flight",
		Help: "Current upstream relay attempts.",
	}, []string{"provider", "channel_id"})
	relayClientCancellations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "client_cancellations_total",
		Help: "Client cancellations by bounded phase.",
	}, []string{"provider", "channel_id", "phase"})
	errorEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "error_events_total",
		Help: "Structured error events by bounded error and routing dimensions.",
	}, []string{"event_kind", "stable_code", "category", "channel_id"})
	channelRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "channel_requests_total",
		Help: "Upstream attempts by channel and outcome.",
	}, []string{"channel_id", "outcome"})
	channelSwitches = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "channel_switch_total",
		Help: "Ordered transitions between channels.",
	}, []string{"from_channel", "to_channel"})
	finalErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "final_errors_total",
		Help: "Errors returned to clients after failover is complete.",
	}, []string{"stable_code", "category", "channel_id"})
	authFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "auth_failures_total",
		Help: "Gateway authentication failures by bounded reason and route.",
	}, []string{"reason", "route", "status"})
	channelCircuitState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "channel_circuit_state",
		Help: "Channel circuit state represented as one hot state labels.",
	}, []string{"channel_id", "route", "state"})
	failoverDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api", Subsystem: "routing", Name: "failover_duration_seconds",
		Help:    "Time spent before a successful failover or final exhaustion.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30},
	}, []string{"outcome"})
)

// Collectors returns all relay collectors for registration in the application registry.
func Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		relayRequests, relayRequestDuration, relayAttempts, relayAttemptDuration,
		relayRetries, relayInFlight, relayClientCancellations,
		errorEvents, channelRequests, channelSwitches,
		finalErrors, authFailures, channelCircuitState, failoverDuration,
	}
}

func RecordAuthFailure(route, reason string, status int) {
	if route == "" {
		route = "unmatched"
	}
	authFailures.WithLabelValues(normalizeAuthFailureReason(reason), route, statusLabel(status)).Inc()
}

func RecordErrorEvent(eventKind string, apiErr *types.NewAPIError) {
	if apiErr == nil {
		return
	}
	code := strconv.Itoa(apiErr.StableCode())
	channel := strconv.Itoa(apiErr.ChannelID())
	errorEvents.WithLabelValues(normalizeEventKind(eventKind), code, apiErr.ErrorCategory(), channel).Inc()
	if eventKind == "final_response" {
		finalErrors.WithLabelValues(code, apiErr.ErrorCategory(), channel).Inc()
	}
}

func RecordChannelRequest(channelID int, outcome string) {
	if outcome != "success" {
		outcome = "error"
	}
	channelRequests.WithLabelValues(strconv.Itoa(channelID), outcome).Inc()
}

func RecordChannelSwitch(fromChannel, toChannel int) {
	channelSwitches.WithLabelValues(strconv.Itoa(fromChannel), strconv.Itoa(toChannel)).Inc()
}

func SetChannelCircuitState(channelID int, route string, state string) {
	if state != "open" && state != "half_open" {
		state = "closed"
	}
	channel := strconv.Itoa(channelID)
	if route == "" {
		route = "unknown"
	}
	for _, candidate := range []string{"closed", "open", "half_open"} {
		value := float64(0)
		if candidate == state {
			value = 1
		}
		channelCircuitState.WithLabelValues(channel, route, candidate).Set(value)
	}
}

func RecordFailoverDuration(outcome string, duration time.Duration) {
	if outcome != "success" {
		outcome = "exhausted"
	}
	failoverDuration.WithLabelValues(outcome).Observe(duration.Seconds())
}

func normalizeEventKind(kind string) string {
	if kind == "upstream_attempt" {
		return kind
	}
	return "final_response"
}

// ErrorClass converts all errors into a deliberately bounded category set.
func ErrorClass(apiErr *types.NewAPIError, contextErr error) string {
	// A relay can finish writing a valid response just before the client closes
	// the connection. In that case Gin's request context is already canceled,
	// but the relay outcome is still a success and must not be reported as 499.
	if apiErr == nil {
		return ErrorSuccess
	}
	if contextErr != nil {
		if errors.Is(contextErr, context.Canceled) {
			return ErrorClientCancelled
		}
		if errors.Is(contextErr, context.DeadlineExceeded) {
			return ErrorResponseTimeout
		}
	}
	code := strings.ToLower(string(apiErr.GetErrorCode()))
	message := strings.ToLower(apiErr.Error())
	switch apiErr.GetErrorCode() {
	case types.ErrorCodeInsufficientUserQuota, types.ErrorCodePreConsumeTokenQuotaFailed:
		return ErrorQuota
	case types.ErrorCodeGetChannelFailed, types.ErrorCodeChannelNoAvailableKey:
		return ErrorNoChannel
	case types.ErrorCodeReadRequestBodyFailed, types.ErrorCodeConvertRequestFailed,
		types.ErrorCodeReadResponseBodyFailed, types.ErrorCodeBadResponseBody:
		return ErrorParse
	}
	if strings.Contains(code, "rate_limit") && apiErr.GetErrorType() == types.ErrorTypeNewAPIError {
		return ErrorUserRateLimit
	}
	if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
		if apiErr.GetErrorType() == types.ErrorTypeNewAPIError {
			return ErrorAuth
		}
		return ErrorUpstream4xx
	}
	if apiErr.StatusCode == 429 {
		if strings.Contains(message, "pending") || strings.Contains(message, "wait queue") || strings.Contains(message, "queue full") {
			return ErrorUpstreamPendingLimit
		}
		return ErrorUpstreamRateLimit
	}
	if strings.Contains(message, "connect") && (strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded")) {
		return ErrorConnectTimeout
	}
	if strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded") {
		return ErrorResponseTimeout
	}
	if apiErr.GetErrorType() == types.ErrorTypeNewAPIError {
		return ErrorInternal
	}
	if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		return ErrorUpstream4xx
	}
	if apiErr.StatusCode >= 500 && apiErr.StatusCode < 600 {
		return ErrorUpstream5xx
	}
	return ErrorInternal
}

func Outcome(errorClass string) string {
	switch errorClass {
	case ErrorSuccess:
		return "success"
	case ErrorClientCancelled:
		return "cancelled"
	default:
		return "error"
	}
}

func MarkContextError(c interface{ Set(string, any) }, errorClass string) {
	if c != nil {
		c.Set(ContextErrorClassKey, errorClass)
	}
}

func statusLabel(status int) string {
	if status <= 0 || status > 599 {
		return "0"
	}
	return strconv.Itoa(status)
}

func RecordRequest(provider string, channelID int, errorClass string, status int, duration time.Duration) {
	provider = normalizeProvider(provider)
	errorClass = normalizeErrorClass(errorClass)
	outcome := Outcome(errorClass)
	channel := channelLabel(channelID)
	relayRequests.WithLabelValues(provider, channel, outcome, errorClass, statusLabel(status)).Inc()
	relayRequestDuration.WithLabelValues(provider, channel, outcome).Observe(duration.Seconds())
}

func RecordAttempt(provider string, channelID int, errorClass string, status int, duration time.Duration) {
	provider = normalizeProvider(provider)
	errorClass = normalizeErrorClass(errorClass)
	outcome := Outcome(errorClass)
	channel := channelLabel(channelID)
	relayAttempts.WithLabelValues(provider, channel, outcome, errorClass, statusLabel(status)).Inc()
	relayAttemptDuration.WithLabelValues(provider, channel, outcome).Observe(duration.Seconds())
}

func RecordRetry(provider string, channelID int, reason string) {
	relayRetries.WithLabelValues(normalizeProvider(provider), channelLabel(channelID), normalizeReason(reason)).Inc()
}

func IncInFlight(provider string, channelID int) func() {
	provider = normalizeProvider(provider)
	channel := channelLabel(channelID)
	relayInFlight.WithLabelValues(provider, channel).Inc()
	return func() { relayInFlight.WithLabelValues(provider, channel).Dec() }
}

func RecordClientCancellation(provider string, channelID int, phase string) {
	if phase != "before_upstream" && phase != "upstream" && phase != "response" {
		phase = "unknown"
	}
	relayClientCancellations.WithLabelValues(normalizeProvider(provider), channelLabel(channelID), phase).Inc()
}

func channelLabel(channelID int) string {
	if channelID <= 0 {
		return "0"
	}
	return strconv.Itoa(channelID)
}

func normalizeProvider(provider string) string {
	if provider == ProviderOther {
		return provider
	}
	return ProviderOther
}

func normalizeErrorClass(errorClass string) string {
	switch errorClass {
	case ErrorSuccess, ErrorAuth, ErrorUserRateLimit, ErrorQuota, ErrorNoChannel,
		ErrorUpstreamPendingLimit, ErrorUpstreamRateLimit, ErrorUpstream4xx,
		ErrorUpstream5xx, ErrorConnectTimeout, ErrorResponseTimeout,
		ErrorClientCancelled, ErrorParse, ErrorInternal:
		return errorClass
	default:
		return ErrorInternal
	}
}

func normalizeReason(reason string) string {
	switch reason {
	case ErrorUpstreamPendingLimit, ErrorUpstreamRateLimit, ErrorUpstream4xx,
		ErrorUpstream5xx, ErrorConnectTimeout, ErrorResponseTimeout, ErrorInternal:
		return reason
	default:
		return ErrorInternal
	}
}

func normalizeAuthFailureReason(reason string) string {
	switch reason {
	case "token_not_provided", "token_not_found", "token_invalid", "token_disabled",
		"token_expired", "token_exhausted", "ip_restricted", "user_disabled",
		"group_forbidden", "token_config_invalid", "specific_channel_forbidden":
		return reason
	default:
		return "token_invalid"
	}
}

// Event is serialized as one JSON object. IDs remain log fields and never labels.
type Event struct {
	Event             string `json:"event"`
	RequestID         string `json:"request_id,omitempty"`
	ClientTraceID     string `json:"client_trace_id,omitempty"`
	UpstreamRequestID string `json:"upstream_request_id,omitempty"`
	Provider          string `json:"provider"`
	ChannelID         int    `json:"channel_id,omitempty"`
	RetryIndex        int    `json:"retry_index,omitempty"`
	Route             string `json:"route,omitempty"`
	Model             string `json:"model,omitempty"`
	Stream            bool   `json:"stream,omitempty"`
	Status            int    `json:"status,omitempty"`
	UpstreamStatus    int    `json:"upstream_status,omitempty"`
	ErrorClass        string `json:"error_class"`
	ErrorCode         string `json:"error_code,omitempty"`
	ErrorSource       string `json:"error_source,omitempty"`
	SourceCode        string `json:"source_code,omitempty"`
	DurationMS        int64  `json:"duration_ms,omitempty"`
	AttemptDurationMS int64  `json:"attempt_duration_ms,omitempty"`
	Retried           bool   `json:"retried,omitempty"`
	StableCode        int    `json:"stable_code,omitempty"`
	ErrorRef          string `json:"error_ref,omitempty"`
	Category          string `json:"category,omitempty"`
	ChannelName       string `json:"channel_name,omitempty"`
	BillingGroup      string `json:"billing_group,omitempty"`
	FailureScope      string `json:"failure_scope,omitempty"`
	Action            string `json:"action,omitempty"`
	AuthFailureReason string `json:"auth_failure_reason,omitempty"`
}

func LogEvent(ctx context.Context, event Event) {
	event.Provider = normalizeProvider(event.Provider)
	encoded, err := json.Marshal(event)
	if err != nil {
		return
	}
	logger.LogInfo(ctx, "observability_event="+string(encoded))
}
