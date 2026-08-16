package observability

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ProviderIkun  = "ikun"
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
	}, []string{"provider", "outcome", "error_class", "final_status"})
	relayRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "request_duration_seconds",
		Help:    "Relay request duration from entry to final outcome.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"provider", "outcome"})
	relayAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "attempts_total",
		Help: "Individual upstream relay attempt outcomes.",
	}, []string{"provider", "outcome", "error_class", "upstream_status"})
	relayAttemptDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "attempt_duration_seconds",
		Help:    "Individual upstream relay attempt duration.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"provider", "outcome"})
	relayRetries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "retries_total",
		Help: "Relay retries by bounded reason.",
	}, []string{"provider", "reason"})
	relayInFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "in_flight",
		Help: "Current upstream relay attempts.",
	}, []string{"provider"})
	relayClientCancellations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "new_api", Subsystem: "relay", Name: "client_cancellations_total",
		Help: "Client cancellations by bounded phase.",
	}, []string{"provider", "phase"})
	errorEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alltoken", Name: "error_events_total",
		Help: "Structured error events by bounded error and routing dimensions.",
	}, []string{"event_kind", "alltoken_code", "category", "cluster_code", "pool_tier"})
	poolRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alltoken", Name: "pool_requests_total",
		Help: "Upstream attempts by cluster pool and outcome.",
	}, []string{"cluster_code", "pool_tier", "outcome"})
	poolFailovers = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alltoken", Name: "pool_failover_total",
		Help: "Pool transitions inside the same cluster.",
	}, []string{"cluster_code", "from_pool", "to_pool", "mode"})
	clusterFailovers = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alltoken", Name: "cluster_failover_total",
		Help: "Cross-cluster routing transitions.",
	}, []string{"from_cluster", "to_cluster", "mode"})
	finalErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alltoken", Name: "final_errors_total",
		Help: "Errors returned to clients after failover is complete.",
	}, []string{"alltoken_code", "category", "cluster_code"})
	clusterCircuitState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "alltoken", Name: "cluster_circuit_state",
		Help: "Cluster circuit state represented as one hot state labels.",
	}, []string{"cluster_code", "route", "state"})
	clusterInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "alltoken", Name: "cluster_info",
		Help: "Configured active clusters available for monitoring filters.",
	}, []string{"cluster_code"})
	failoverDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "alltoken", Name: "failover_duration_seconds",
		Help:    "Time spent before a successful failover or final exhaustion.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30},
	}, []string{"outcome", "mode"})
)

// Collectors returns all relay collectors for registration in the application registry.
func Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		relayRequests, relayRequestDuration, relayAttempts, relayAttemptDuration,
		relayRetries, relayInFlight, relayClientCancellations,
		errorEvents, poolRequests, poolFailovers, clusterFailovers,
		finalErrors, clusterCircuitState, clusterInfo, failoverDuration,
	}
}

func ReplaceClusterInfo(clusterCodes []int) {
	clusterInfo.Reset()
	for _, clusterCode := range clusterCodes {
		if clusterCode > 0 {
			clusterInfo.WithLabelValues(strconv.Itoa(clusterCode)).Set(1)
		}
	}
}

func RecordErrorEvent(eventKind string, apiErr *types.NewAPIError) {
	if apiErr == nil {
		return
	}
	code := strconv.Itoa(apiErr.AlltokenCode())
	cluster := strconv.Itoa(apiErr.ClusterCode())
	pool := strconv.Itoa(apiErr.PoolTier())
	errorEvents.WithLabelValues(normalizeEventKind(eventKind), code, apiErr.ErrorCategory(), cluster, pool).Inc()
	if eventKind == "final_response" {
		finalErrors.WithLabelValues(code, apiErr.ErrorCategory(), cluster).Inc()
	}
}

func RecordPoolRequest(clusterCode int, poolTier int, outcome string) {
	if outcome != "success" {
		outcome = "error"
	}
	poolRequests.WithLabelValues(strconv.Itoa(clusterCode), strconv.Itoa(poolTier), outcome).Inc()
}

func RecordPoolFailover(clusterCode, fromPool, toPool int, mode string) {
	poolFailovers.WithLabelValues(strconv.Itoa(clusterCode), strconv.Itoa(fromPool), strconv.Itoa(toPool), normalizeMode(mode)).Inc()
}

func RecordClusterFailover(fromCluster, toCluster int, mode string) {
	clusterFailovers.WithLabelValues(strconv.Itoa(fromCluster), strconv.Itoa(toCluster), normalizeMode(mode)).Inc()
}

func SetClusterCircuitState(clusterCode int, route string, state string) {
	if state != "open" && state != "half_open" {
		state = "closed"
	}
	cluster := strconv.Itoa(clusterCode)
	if route == "" {
		route = "unknown"
	}
	for _, candidate := range []string{"closed", "open", "half_open"} {
		value := float64(0)
		if candidate == state {
			value = 1
		}
		clusterCircuitState.WithLabelValues(cluster, route, candidate).Set(value)
	}
}

func RecordFailoverDuration(outcome, mode string, duration time.Duration) {
	if outcome != "success" {
		outcome = "exhausted"
	}
	failoverDuration.WithLabelValues(outcome, normalizeMode(mode)).Observe(duration.Seconds())
}

func normalizeEventKind(kind string) string {
	if kind == "upstream_attempt" {
		return kind
	}
	return "final_response"
}

func normalizeMode(mode string) string {
	switch mode {
	case "conservative", "aggressive":
		return mode
	default:
		return "balanced"
	}
}

// ProviderFromBaseURL maps arbitrary upstream URLs into a fixed provider set.
func ProviderFromBaseURL(raw string) string {
	host := ""
	if parsed, err := url.Parse(strings.TrimSpace(raw)); err == nil {
		host = parsed.Hostname()
	}
	if host == "" {
		host = strings.TrimSpace(raw)
		if splitHost, _, err := net.SplitHostPort(host); err == nil {
			host = splitHost
		}
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "ikun.love" || strings.HasSuffix(host, ".ikun.love") {
		return ProviderIkun
	}
	return ProviderOther
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

func RecordRequest(provider, errorClass string, status int, duration time.Duration) {
	provider = normalizeProvider(provider)
	errorClass = normalizeErrorClass(errorClass)
	outcome := Outcome(errorClass)
	relayRequests.WithLabelValues(provider, outcome, errorClass, statusLabel(status)).Inc()
	relayRequestDuration.WithLabelValues(provider, outcome).Observe(duration.Seconds())
}

func RecordAttempt(provider, errorClass string, status int, duration time.Duration) {
	provider = normalizeProvider(provider)
	errorClass = normalizeErrorClass(errorClass)
	outcome := Outcome(errorClass)
	relayAttempts.WithLabelValues(provider, outcome, errorClass, statusLabel(status)).Inc()
	relayAttemptDuration.WithLabelValues(provider, outcome).Observe(duration.Seconds())
}

func RecordRetry(provider, reason string) {
	relayRetries.WithLabelValues(normalizeProvider(provider), normalizeReason(reason)).Inc()
}

func IncInFlight(provider string) func() {
	provider = normalizeProvider(provider)
	relayInFlight.WithLabelValues(provider).Inc()
	return func() { relayInFlight.WithLabelValues(provider).Dec() }
}

func RecordClientCancellation(provider, phase string) {
	if phase != "before_upstream" && phase != "upstream" && phase != "response" {
		phase = "unknown"
	}
	relayClientCancellations.WithLabelValues(normalizeProvider(provider), phase).Inc()
}

func normalizeProvider(provider string) string {
	if provider == ProviderIkun {
		return ProviderIkun
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
	AlltokenCode      int    `json:"alltoken_code,omitempty"`
	ErrorRef          string `json:"error_ref,omitempty"`
	Category          string `json:"category,omitempty"`
	ClusterCode       int    `json:"cluster_code,omitempty"`
	PoolTier          int    `json:"pool_tier,omitempty"`
	FailureScope      string `json:"failure_scope,omitempty"`
	Action            string `json:"action,omitempty"`
	FailoverMode      string `json:"failover_mode,omitempty"`
}

func LogEvent(ctx context.Context, event Event) {
	event.Provider = normalizeProvider(event.Provider)
	encoded, err := json.Marshal(event)
	if err != nil {
		return
	}
	logger.LogInfo(ctx, "observability_event="+string(encoded))
}
