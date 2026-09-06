package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFailoverMonitoringSnapshotAggregatesMetricsAndAlerts(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	common.MonitoringEnabled = true
	t.Cleanup(func() { common.MonitoringEnabled = previousMonitoringEnabled })

	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		query := r.URL.Query().Get("query")
		values := map[string]string{
			`sum(rate(new_api_relay_requests_total[5m]))`: "12.5",
			`sum(rate(new_api_relay_requests_total{outcome="error"}[5m])) / clamp_min(sum(rate(new_api_relay_requests_total[5m])), 0.001)`: "0.025",
			`histogram_quantile(0.95, sum by (le) (rate(new_api_relay_request_duration_seconds_bucket[5m])))`:                              "0.56",
			`sum(new_api_relay_in_flight)`:                           "4",
			`sum(increase(alltoken_channel_switch_total[5m]))`:       "3",
			`sum(alltoken_channel_circuit_state{state="open"} == 1)`: "1",
			`new_api_database_connections{database="main",state="in_use"} / clamp_min(new_api_database_connections{database="main",state="max_open"}, 1)`: "0.4",
			`sum(increase(new_api_redis_pool_timeouts_total[5m]))`: "0",
		}
		value, ok := values[query]
		if !ok {
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1786773070,"%s"]}]}}`, value)
	}))
	t.Cleanup(prometheus.Close)

	alertmanager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"labels":{"alertname":"AllTokenChannelCircuitOpen","severity":"critical","channel_id":"38","instance":"new-api:8006"},"annotations":{"summary":"Channel circuit is open","description":"Traffic moved to another channel."},"startsAt":"2026-08-15T05:51:49Z","fingerprint":"abc","status":{"state":"active"}}]`))
	}))
	t.Cleanup(alertmanager.Close)

	t.Setenv("FAILOVER_PROMETHEUS_URL", prometheus.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", alertmanager.URL)
	t.Setenv("FAILOVER_GRAFANA_PUBLIC_URL", "https://monitoring.example.com/d/failover?orgId=1")
	t.Setenv("FAILOVER_MONITORING_BEARER_TOKEN", "test-token")

	snapshot := GetFailoverMonitoringSnapshot(context.Background())

	assert.Equal(t, "5m", snapshot.Window)
	assert.Equal(t, 12.5, snapshot.Metrics.RequestRPS)
	assert.Equal(t, 0.025, snapshot.Metrics.ErrorRate)
	assert.Equal(t, 0.56, snapshot.Metrics.P95LatencySeconds)
	assert.Equal(t, 3.0, snapshot.Metrics.ChannelSwitches)
	assert.Equal(t, 1.0, snapshot.Metrics.OpenCircuits)
	assert.Equal(t, 0.4, snapshot.Metrics.DatabaseUsage)
	require.Len(t, snapshot.Alerts, 1)
	assert.Equal(t, "AllTokenChannelCircuitOpen", snapshot.Alerts[0].Name)
	assert.Equal(t, "critical", snapshot.Alerts[0].Severity)
	assert.Equal(t, "38", snapshot.Alerts[0].ChannelID)
	assert.Equal(t, "https://monitoring.example.com/d/failover?orgId=1", snapshot.GrafanaURL)
	for _, source := range snapshot.Sources {
		assert.Equal(t, "healthy", source.Status)
	}
}

func TestGetFailoverMonitoringSnapshotCachesRepeatedRefreshes(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	common.MonitoringEnabled = true
	t.Cleanup(func() { common.MonitoringEnabled = previousMonitoringEnabled })

	var prometheusRequests atomic.Int32
	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		prometheusRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1786773070,"1"]}]}}`))
	}))
	t.Cleanup(prometheus.Close)

	alertmanager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(alertmanager.Close)

	t.Setenv("FAILOVER_PROMETHEUS_URL", prometheus.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", alertmanager.URL)
	t.Setenv("FAILOVER_GRAFANA_PUBLIC_URL", "")
	t.Setenv("FAILOVER_MONITORING_USERNAME", "")
	t.Setenv("FAILOVER_MONITORING_PASSWORD", "")
	t.Setenv("FAILOVER_MONITORING_BEARER_TOKEN", "")

	first := GetFailoverMonitoringSnapshot(context.Background())
	second := GetFailoverMonitoringSnapshot(context.Background())

	assert.Equal(t, first.Metrics, second.Metrics)
	assert.Equal(t, int32(8), prometheusRequests.Load())
}

func TestGetFailoverMonitoringSnapshotReportsMissingConfiguration(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	common.MonitoringEnabled = true
	t.Cleanup(func() { common.MonitoringEnabled = previousMonitoringEnabled })

	for _, name := range []string{
		"FAILOVER_PROMETHEUS_URL",
		"FAILOVER_ALERTMANAGER_URL",
		"FAILOVER_GRAFANA_PUBLIC_URL",
		"FAILOVER_MONITORING_USERNAME",
		"FAILOVER_MONITORING_PASSWORD",
		"FAILOVER_MONITORING_BEARER_TOKEN",
	} {
		t.Setenv(name, "")
	}

	snapshot := GetFailoverMonitoringSnapshot(context.Background())

	assert.Empty(t, snapshot.GrafanaURL)
	assert.Empty(t, snapshot.Alerts)
	require.Len(t, snapshot.Sources, 3)
	for _, source := range snapshot.Sources {
		assert.Equal(t, "not_configured", source.Status)
		assert.True(t, strings.Contains(source.Message, "not configured"))
	}
}

func TestGetFailoverMonitoringSnapshotReturnsChannelAlerts(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	common.MonitoringEnabled = true
	t.Cleanup(func() { common.MonitoringEnabled = previousMonitoringEnabled })

	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		expectedQueries := map[string]struct{}{
			`sum(rate(new_api_relay_requests_total[5m]))`: {},
			`sum(rate(new_api_relay_requests_total{outcome="error"}[5m])) / clamp_min(sum(rate(new_api_relay_requests_total[5m])), 0.001)`: {},
			`histogram_quantile(0.95, sum by (le) (rate(new_api_relay_request_duration_seconds_bucket[5m])))`:                              {},
			`sum(new_api_relay_in_flight)`:                           {},
			`sum(increase(alltoken_channel_switch_total[5m]))`:       {},
			`sum(alltoken_channel_circuit_state{state="open"} == 1)`: {},
			`new_api_database_connections{database="main",state="in_use"} / clamp_min(new_api_database_connections{database="main",state="max_open"}, 1)`: {},
			`sum(increase(new_api_redis_pool_timeouts_total[5m]))`: {},
		}
		if _, ok := expectedQueries[query]; !ok {
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1786773070,"1"]}]}}`))
	}))
	t.Cleanup(prometheus.Close)

	alertmanager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"labels":{"alertname":"Global","severity":"warning"},"fingerprint":"global","status":{"state":"active"}},
			{"labels":{"alertname":"Selected","severity":"critical","channel_id":"7"},"fingerprint":"selected","status":{"state":"active"}},
			{"labels":{"alertname":"Other","severity":"critical","channel_id":"8"},"fingerprint":"other","status":{"state":"active"}}
		]`))
	}))
	t.Cleanup(alertmanager.Close)

	t.Setenv("FAILOVER_PROMETHEUS_URL", prometheus.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", alertmanager.URL)

	snapshot := GetFailoverMonitoringSnapshot(context.Background())

	require.Len(t, snapshot.Alerts, 3)
	assert.Equal(t, "Selected", snapshot.Alerts[0].Name)
	assert.Equal(t, "Other", snapshot.Alerts[1].Name)
	for _, source := range snapshot.Sources[:2] {
		assert.Equal(t, "healthy", source.Status)
	}
}

func TestGetFailoverMonitoringSnapshotRejectsNonFiniteMetrics(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	common.MonitoringEnabled = true
	t.Cleanup(func() { common.MonitoringEnabled = previousMonitoringEnabled })

	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := "1"
		if strings.Contains(r.URL.Query().Get("query"), "histogram_quantile") {
			value = "NaN"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1786773070,"%s"]}]}}`, value)
	}))
	t.Cleanup(prometheus.Close)

	t.Setenv("FAILOVER_PROMETHEUS_URL", prometheus.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", "")
	t.Setenv("FAILOVER_GRAFANA_PUBLIC_URL", "")

	snapshot := GetFailoverMonitoringSnapshot(context.Background())

	assert.Zero(t, snapshot.Metrics.P95LatencySeconds)
	require.Len(t, snapshot.Sources, 3)
	assert.Equal(t, "degraded", snapshot.Sources[0].Status)
	assert.Equal(t, "1 metric queries failed", snapshot.Sources[0].Message)
	_, err := common.Marshal(snapshot)
	require.NoError(t, err)
}

func TestGetFailoverMonitoringSnapshotDisabledSkipsMonitoringSources(t *testing.T) {
	previousMonitoringEnabled := common.MonitoringEnabled
	common.MonitoringEnabled = false
	t.Cleanup(func() { common.MonitoringEnabled = previousMonitoringEnabled })

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("FAILOVER_PROMETHEUS_URL", server.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", server.URL)
	t.Setenv("FAILOVER_GRAFANA_PUBLIC_URL", server.URL)

	snapshot := GetFailoverMonitoringSnapshot(context.Background())

	assert.False(t, snapshot.Enabled)
	assert.Equal(t, "disabled", snapshot.Status)
	assert.Zero(t, requests.Load())
	require.Len(t, snapshot.Sources, 3)
	for _, source := range snapshot.Sources {
		assert.Equal(t, "disabled", source.Status)
		assert.Equal(t, "monitoring disabled", source.Message)
	}
}
