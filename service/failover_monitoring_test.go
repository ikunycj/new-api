package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFailoverMonitoringSnapshotAggregatesMetricsAndAlerts(t *testing.T) {
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
			`sum(increase(alltoken_cluster_failover_total[5m]))`:     "3",
			`sum(increase(alltoken_pool_failover_total[5m]))`:        "8",
			`sum(alltoken_cluster_circuit_state{state="open"} == 1)`: "1",
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
		_, _ = w.Write([]byte(`[{"labels":{"alertname":"AllTokenClusterCircuitOpen","severity":"critical","cluster_code":"1","instance":"new-api:8006"},"annotations":{"summary":"Cluster C1 circuit is open","description":"Traffic moved to another cluster."},"startsAt":"2026-08-15T05:51:49Z","fingerprint":"abc","status":{"state":"active"}}]`))
	}))
	t.Cleanup(alertmanager.Close)

	t.Setenv("FAILOVER_PROMETHEUS_URL", prometheus.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", alertmanager.URL)
	t.Setenv("FAILOVER_GRAFANA_PUBLIC_URL", "https://monitoring.example.com/d/failover?orgId=1")
	t.Setenv("FAILOVER_MONITORING_BEARER_TOKEN", "test-token")

	snapshot := GetFailoverMonitoringSnapshot(context.Background(), 0)

	assert.Equal(t, "5m", snapshot.Window)
	assert.Equal(t, 12.5, snapshot.Metrics.RequestRPS)
	assert.Equal(t, 0.025, snapshot.Metrics.ErrorRate)
	assert.Equal(t, 0.56, snapshot.Metrics.P95LatencySeconds)
	assert.Equal(t, 3.0, snapshot.Metrics.ClusterFailovers)
	assert.Equal(t, 8.0, snapshot.Metrics.PoolFailovers)
	assert.Equal(t, 1.0, snapshot.Metrics.OpenCircuits)
	assert.Equal(t, 0.4, snapshot.Metrics.DatabaseUsage)
	require.Len(t, snapshot.Alerts, 1)
	assert.Equal(t, "AllTokenClusterCircuitOpen", snapshot.Alerts[0].Name)
	assert.Equal(t, "critical", snapshot.Alerts[0].Severity)
	assert.Equal(t, "1", snapshot.Alerts[0].ClusterCode)
	assert.Equal(t, "https://monitoring.example.com/d/failover?orgId=1", snapshot.GrafanaURL)
	for _, source := range snapshot.Sources {
		assert.Equal(t, "healthy", source.Status)
	}
}

func TestGetFailoverMonitoringSnapshotReportsMissingConfiguration(t *testing.T) {
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

	snapshot := GetFailoverMonitoringSnapshot(context.Background(), 0)

	assert.Empty(t, snapshot.GrafanaURL)
	assert.Empty(t, snapshot.Alerts)
	require.Len(t, snapshot.Sources, 3)
	for _, source := range snapshot.Sources {
		assert.Equal(t, "not_configured", source.Status)
		assert.True(t, strings.Contains(source.Message, "not configured"))
	}
}

func TestGetFailoverMonitoringSnapshotFiltersMetricsAndAlertsByCluster(t *testing.T) {
	prometheus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		expectedQueries := map[string]struct{}{
			`sum(rate(alltoken_pool_requests_total{cluster_code="7"}[5m]))`: {},
			`sum(rate(alltoken_pool_requests_total{cluster_code="7",outcome="error"}[5m])) / clamp_min(sum(rate(alltoken_pool_requests_total{cluster_code="7"}[5m])), 0.001)`: {},
			`histogram_quantile(0.95, sum by (le) (rate(new_api_relay_request_duration_seconds_bucket[5m])))`:                                                                 {},
			`sum(new_api_relay_in_flight)`: {},
			`sum(increase(alltoken_cluster_failover_total{from_cluster="7"}[5m])) + sum(increase(alltoken_cluster_failover_total{to_cluster="7"}[5m]))`:   {},
			`sum(increase(alltoken_pool_failover_total{cluster_code="7"}[5m]))`:                                                                           {},
			`sum(alltoken_cluster_circuit_state{cluster_code="7",state="open"} == 1)`:                                                                     {},
			`new_api_database_connections{database="main",state="in_use"} / clamp_min(new_api_database_connections{database="main",state="max_open"}, 1)`: {},
			`sum(increase(new_api_redis_pool_timeouts_total[5m]))`:                                                                                        {},
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
			{"labels":{"alertname":"Selected","severity":"critical","cluster_code":"7"},"fingerprint":"selected","status":{"state":"active"}},
			{"labels":{"alertname":"Other","severity":"critical","cluster_code":"8"},"fingerprint":"other","status":{"state":"active"}}
		]`))
	}))
	t.Cleanup(alertmanager.Close)

	t.Setenv("FAILOVER_PROMETHEUS_URL", prometheus.URL)
	t.Setenv("FAILOVER_ALERTMANAGER_URL", alertmanager.URL)

	snapshot := GetFailoverMonitoringSnapshot(context.Background(), 7)

	assert.Equal(t, 7, snapshot.ClusterCode)
	require.Len(t, snapshot.Alerts, 2)
	assert.Equal(t, "Selected", snapshot.Alerts[0].Name)
	assert.Equal(t, "Global", snapshot.Alerts[1].Name)
	for _, source := range snapshot.Sources[:2] {
		assert.Equal(t, "healthy", source.Status)
	}
}

func TestGetFailoverMonitoringSnapshotRejectsNonFiniteMetrics(t *testing.T) {
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

	snapshot := GetFailoverMonitoringSnapshot(context.Background(), 0)

	assert.Zero(t, snapshot.Metrics.P95LatencySeconds)
	require.Len(t, snapshot.Sources, 3)
	assert.Equal(t, "degraded", snapshot.Sources[0].Status)
	assert.Equal(t, "1 metric queries failed", snapshot.Sources[0].Message)
	_, err := common.Marshal(snapshot)
	require.NoError(t, err)
}
