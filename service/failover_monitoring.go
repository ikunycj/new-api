package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"golang.org/x/sync/singleflight"
)

const (
	failoverMonitoringWindow   = "5m"
	failoverMonitoringTimeout  = 4 * time.Second
	failoverMonitoringCacheTTL = 8 * time.Second
)

var errMonitoringNotConfigured = errors.New("monitoring source is not configured")

type FailoverMonitoringSource struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type FailoverMonitoringMetrics struct {
	RequestRPS        float64 `json:"request_rps"`
	ErrorRate         float64 `json:"error_rate"`
	P95LatencySeconds float64 `json:"p95_latency_seconds"`
	InFlight          float64 `json:"in_flight"`
	ChannelSwitches   float64 `json:"channel_switches"`
	OpenCircuits      float64 `json:"open_circuits"`
	DatabaseUsage     float64 `json:"database_usage"`
	RedisTimeouts     float64 `json:"redis_timeouts"`
}

type FailoverMonitoringAlert struct {
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	ChannelID   string `json:"channel_id,omitempty"`
	Instance    string `json:"instance,omitempty"`
	StartedAt   string `json:"started_at"`
}

type FailoverMonitoringSnapshot struct {
	UpdatedAt  int64                      `json:"updated_at"`
	Window     string                     `json:"window"`
	Metrics    FailoverMonitoringMetrics  `json:"metrics"`
	Alerts     []FailoverMonitoringAlert  `json:"alerts"`
	Sources    []FailoverMonitoringSource `json:"sources"`
	GrafanaURL string                     `json:"grafana_url,omitempty"`
}

type failoverMonitoringConfig struct {
	prometheusURL   string
	alertmanagerURL string
	grafanaURL      string
	username        string
	password        string
	bearerToken     string
}

type cachedFailoverMonitoringSnapshot struct {
	snapshot  FailoverMonitoringSnapshot
	expiresAt time.Time
}

var failoverMonitoringCache = struct {
	sync.Mutex
	values map[string]cachedFailoverMonitoringSnapshot
}{values: make(map[string]cachedFailoverMonitoringSnapshot)}

var failoverMonitoringRefresh singleflight.Group

type prometheusQueryResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		Result []struct {
			Value []any `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type alertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	Fingerprint string            `json:"fingerprint"`
	Status      struct {
		State string `json:"state"`
	} `json:"status"`
}

func GetFailoverMonitoringSnapshot(ctx context.Context) FailoverMonitoringSnapshot {
	config := failoverMonitoringConfigFromEnv()
	cacheKey := failoverMonitoringCacheKey(config)
	now := time.Now()
	failoverMonitoringCache.Lock()
	if cached, ok := failoverMonitoringCache.values[cacheKey]; ok && now.Before(cached.expiresAt) {
		failoverMonitoringCache.Unlock()
		return cached.snapshot
	}
	failoverMonitoringCache.Unlock()

	value, _, _ := failoverMonitoringRefresh.Do(cacheKey, func() (any, error) {
		now := time.Now()
		failoverMonitoringCache.Lock()
		if cached, ok := failoverMonitoringCache.values[cacheKey]; ok && now.Before(cached.expiresAt) {
			failoverMonitoringCache.Unlock()
			return cached.snapshot, nil
		}
		failoverMonitoringCache.Unlock()

		snapshot := getFailoverMonitoringSnapshot(ctx, config)
		failoverMonitoringCache.Lock()
		failoverMonitoringCache.values[cacheKey] = cachedFailoverMonitoringSnapshot{
			snapshot:  snapshot,
			expiresAt: time.Now().Add(failoverMonitoringCacheTTL),
		}
		failoverMonitoringCache.Unlock()
		return snapshot, nil
	})
	return value.(FailoverMonitoringSnapshot)
}

func getFailoverMonitoringSnapshot(ctx context.Context, config failoverMonitoringConfig) FailoverMonitoringSnapshot {
	snapshot := FailoverMonitoringSnapshot{
		UpdatedAt: time.Now().UnixMilli(),
		Window:    failoverMonitoringWindow,
		Alerts:    []FailoverMonitoringAlert{},
		Sources: []FailoverMonitoringSource{
			monitoringSource("prometheus", config.prometheusURL),
			monitoringSource("alertmanager", config.alertmanagerURL),
			monitoringSource("grafana", config.grafanaURL),
		},
	}

	client := &http.Client{Timeout: failoverMonitoringTimeout}
	if config.prometheusURL != "" {
		metrics, errCount, err := fetchFailoverMonitoringMetrics(ctx, client, config)
		snapshot.Metrics = metrics
		setMonitoringSource(&snapshot.Sources[0], err, errCount)
	}
	if config.alertmanagerURL != "" {
		alerts, err := fetchFailoverMonitoringAlerts(ctx, client, config)
		snapshot.Alerts = alerts
		setMonitoringSource(&snapshot.Sources[1], err, 0)
	}
	if grafanaURL, err := validateMonitoringURL(config.grafanaURL); err == nil {
		snapshot.GrafanaURL = grafanaURL
		setMonitoringSource(&snapshot.Sources[2], nil, 0)
	} else if config.grafanaURL != "" {
		setMonitoringSource(&snapshot.Sources[2], err, 0)
	}

	return snapshot
}

func failoverMonitoringCacheKey(config failoverMonitoringConfig) string {
	material := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s", config.prometheusURL, config.alertmanagerURL, config.grafanaURL, config.username, config.password+"\x00"+config.bearerToken)
	digest := sha256.Sum256([]byte(material))
	return fmt.Sprintf("channel:%x", digest[:8])
}

func failoverMonitoringConfigFromEnv() failoverMonitoringConfig {
	return failoverMonitoringConfig{
		prometheusURL:   common.GetEnvOrDefaultString("FAILOVER_PROMETHEUS_URL", ""),
		alertmanagerURL: common.GetEnvOrDefaultString("FAILOVER_ALERTMANAGER_URL", ""),
		grafanaURL:      common.GetEnvOrDefaultString("FAILOVER_GRAFANA_PUBLIC_URL", ""),
		username:        common.GetEnvOrDefaultString("FAILOVER_MONITORING_USERNAME", ""),
		password:        common.GetEnvOrDefaultString("FAILOVER_MONITORING_PASSWORD", ""),
		bearerToken:     common.GetEnvOrDefaultString("FAILOVER_MONITORING_BEARER_TOKEN", ""),
	}
}

func monitoringSource(name, target string) FailoverMonitoringSource {
	status := "not_configured"
	message := "not configured"
	if target != "" {
		status = "pending"
		message = ""
	}
	return FailoverMonitoringSource{Name: name, Status: status, Message: message}
}

func setMonitoringSource(source *FailoverMonitoringSource, err error, errorCount int) {
	if err == nil && errorCount == 0 {
		source.Status = "healthy"
		source.Message = ""
		return
	}
	if errors.Is(err, errMonitoringNotConfigured) {
		source.Status = "not_configured"
		source.Message = "not configured"
		return
	}
	if errorCount > 0 && errorCount < 9 {
		source.Status = "degraded"
		source.Message = fmt.Sprintf("%d metric queries failed", errorCount)
		return
	}
	source.Status = "unavailable"
	source.Message = "request failed"
}

func fetchFailoverMonitoringMetrics(ctx context.Context, client *http.Client, config failoverMonitoringConfig) (FailoverMonitoringMetrics, int, error) {
	queries := []struct {
		name  string
		query string
	}{
		{name: "request_rps", query: `sum(rate(new_api_relay_requests_total[5m]))`},
		{name: "error_rate", query: `sum(rate(new_api_relay_requests_total{outcome="error"}[5m])) / clamp_min(sum(rate(new_api_relay_requests_total[5m])), 0.001)`},
		{name: "p95_latency_seconds", query: `histogram_quantile(0.95, sum by (le) (rate(new_api_relay_request_duration_seconds_bucket[5m])))`},
		{name: "in_flight", query: `sum(new_api_relay_in_flight)`},
		{name: "channel_switches", query: `sum(increase(new_api_routing_channel_switch_total[5m]))`},
		{name: "open_circuits", query: `sum(new_api_routing_channel_circuit_state{state="open"} == 1)`},
		{name: "database_usage", query: `new_api_database_connections{database="main",state="in_use"} / clamp_min(new_api_database_connections{database="main",state="max_open"}, 1)`},
		{name: "redis_timeouts", query: `sum(increase(new_api_redis_pool_timeouts_total[5m]))`},
	}
	values := make([]float64, len(queries))
	errorsByQuery := make([]error, len(queries))
	var waitGroup sync.WaitGroup
	for index, query := range queries {
		index, query := index, query
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			values[index], errorsByQuery[index] = queryPrometheus(ctx, client, config, query.query)
		}()
	}
	waitGroup.Wait()

	metrics := FailoverMonitoringMetrics{}
	for index, query := range queries {
		if errorsByQuery[index] != nil {
			continue
		}
		switch query.name {
		case "request_rps":
			metrics.RequestRPS = values[index]
		case "error_rate":
			metrics.ErrorRate = values[index]
		case "p95_latency_seconds":
			metrics.P95LatencySeconds = values[index]
		case "in_flight":
			metrics.InFlight = values[index]
		case "channel_switches":
			metrics.ChannelSwitches = values[index]
		case "open_circuits":
			metrics.OpenCircuits = values[index]
		case "database_usage":
			metrics.DatabaseUsage = values[index]
		case "redis_timeouts":
			metrics.RedisTimeouts = values[index]
		}
	}
	failed := 0
	for _, err := range errorsByQuery {
		if err != nil {
			failed++
		}
	}
	return metrics, failed, nil
}

func queryPrometheus(ctx context.Context, client *http.Client, config failoverMonitoringConfig, query string) (float64, error) {
	endpoint, err := monitoringEndpoint(config.prometheusURL, "/api/v1/query")
	if err != nil {
		return 0, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return 0, err
	}
	queryValues := parsed.Query()
	queryValues.Set("query", query)
	parsed.RawQuery = queryValues.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return 0, err
	}
	applyMonitoringAuth(request, config)
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("monitoring request returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	var payload prometheusQueryResponse
	if err := common.DecodeJson(bytes.NewReader(body), &payload); err != nil {
		return 0, err
	}
	if payload.Status != "success" {
		return 0, errors.New("prometheus query failed")
	}
	if len(payload.Data.Result) == 0 || len(payload.Data.Result[0].Value) < 2 {
		return 0, nil
	}
	return parseMonitoringNumber(payload.Data.Result[0].Value[1])
}

func fetchFailoverMonitoringAlerts(ctx context.Context, client *http.Client, config failoverMonitoringConfig) ([]FailoverMonitoringAlert, error) {
	endpoint, err := monitoringEndpoint(config.alertmanagerURL, "/api/v2/alerts")
	if err != nil {
		return []FailoverMonitoringAlert{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return []FailoverMonitoringAlert{}, err
	}
	applyMonitoringAuth(request, config)
	response, err := client.Do(request)
	if err != nil {
		return []FailoverMonitoringAlert{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return []FailoverMonitoringAlert{}, fmt.Errorf("monitoring request returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return []FailoverMonitoringAlert{}, err
	}
	var payload []alertmanagerAlert
	if err := common.DecodeJson(bytes.NewReader(body), &payload); err != nil {
		return []FailoverMonitoringAlert{}, err
	}
	alerts := make([]FailoverMonitoringAlert, 0, len(payload))
	for _, alert := range payload {
		status := alert.Status.State
		if status == "" {
			status = "active"
		}
		alerts = append(alerts, FailoverMonitoringAlert{
			Fingerprint: alert.Fingerprint,
			Name:        alert.Labels["alertname"],
			Severity:    alert.Labels["severity"],
			Status:      status,
			Summary:     alert.Annotations["summary"],
			Description: alert.Annotations["description"],
			ChannelID:   alert.Labels["channel_id"],
			Instance:    alert.Labels["instance"],
			StartedAt:   alert.StartsAt,
		})
	}
	sort.SliceStable(alerts, func(i, j int) bool {
		severityRank := func(value string) int {
			switch value {
			case "critical":
				return 0
			case "warning":
				return 1
			default:
				return 2
			}
		}
		if severityRank(alerts[i].Severity) != severityRank(alerts[j].Severity) {
			return severityRank(alerts[i].Severity) < severityRank(alerts[j].Severity)
		}
		return alerts[i].StartedAt > alerts[j].StartedAt
	})
	return alerts, nil
}

func monitoringEndpoint(rawBaseURL, path string) (string, error) {
	if strings.TrimSpace(rawBaseURL) == "" {
		return "", errMonitoringNotConfigured
	}
	parsed, err := validateMonitoringURL(rawBaseURL)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(parsed)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	base.RawQuery = ""
	base.Fragment = ""
	return base.String(), nil
}

func validateMonitoringURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return "", errors.New("invalid monitoring URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("monitoring URL must use http or https")
	}
	parsed.User = nil
	return strings.TrimRight(parsed.String(), "/"), nil
}

func applyMonitoringAuth(request *http.Request, config failoverMonitoringConfig) {
	if config.bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+config.bearerToken)
		return
	}
	if config.username != "" {
		request.SetBasicAuth(config.username, config.password)
	}
}

func parseMonitoringNumber(value any) (float64, error) {
	var number float64
	switch parsed := value.(type) {
	case float64:
		number = parsed
	case string:
		var err error
		number, err = strconv.ParseFloat(parsed, 64)
		if err != nil {
			return 0, err
		}
	default:
		return 0, errors.New("invalid monitoring metric value")
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, errors.New("monitoring metric value must be finite")
	}
	return number, nil
}
