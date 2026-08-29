package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadK6SummaryBuildsTerminalResult(t *testing.T) {
	summary := `{
  "metrics": {
	"http_reqs": {"count": 100, "rate": 20},
	"http_req_failed": {"value": 0.03, "passes": 3, "fails": 97},
	"dropped_iterations": {"count": 2, "rate": 0.4},
	"http_req_duration": {"med": 120, "p(95)": 400, "p(99)": 800},
	"alltoken_status_429": {"count": 2},
	"alltoken_status_500": {"count": 1},
	"alltoken_errors{code:alltoken.upstream_exhausted}": {"count": 2},
	"alltoken_errors{code:provider_error}": {"count": 1},
	"alltoken_input_tokens": {"count": 1000},
	"alltoken_output_tokens": {"count": 200},
	"alltoken_cache_read_tokens": {"count": 600},
	"alltoken_cache_write_tokens": {"count": 100}
  }
}`
	path := filepath.Join(t.TempDir(), "summary.json")
	require.NoError(t, os.WriteFile(path, []byte(summary), 0o600))

	result, err := readK6Summary(path, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(102), result.Sent)
	assert.Equal(t, int64(100), result.Completed)
	assert.Equal(t, int64(97), result.Successes)
	assert.Equal(t, int64(3), result.Failures)
	assert.Equal(t, map[string]int64{
		"http_429": 2, "http_500": 1,
		"alltoken.upstream_exhausted": 2, "provider_error": 1,
	}, result.ErrorCounts)
	assert.Equal(t, float64(400), result.P95MS)
	assert.Equal(t, int64(600), result.CacheReadTokens)
}

func TestReadK6SummarySupportsLegacyValuesObject(t *testing.T) {
	summary := `{
  "metrics": {
    "http_reqs": {"values": {"count": 10}},
    "http_req_failed": {"values": {"rate": 0.2, "passes": 2, "fails": 8}},
    "http_req_duration": {"values": {"med": 50, "p(95)": 90, "p(99)": 100}}
  }
}`
	path := filepath.Join(t.TempDir(), "summary.json")
	require.NoError(t, os.WriteFile(path, []byte(summary), 0o600))

	result, err := readK6Summary(path, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(10), result.Completed)
	assert.Equal(t, int64(8), result.Successes)
	assert.Equal(t, int64(2), result.Failures)
	assert.Equal(t, float64(90), result.P95MS)
}

func TestReadK6SummaryRejectsRunWithoutRequests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"metrics": {}}`), 0o600))

	_, err := readK6Summary(path, 5)
	require.EqualError(t, err, "k6 completed without issuing any HTTP requests")
}

func TestValidateServerURLRequiresHTTPSOutsideLoopback(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{url: "https://alltokenapi.com", want: true},
		{url: "http://127.0.0.1:3000", want: true},
		{url: "http://localhost:3000", want: true},
		{url: "http://[::1]:3000", want: true},
		{url: "http://alltokenapi.com", want: false},
		{url: "http://localhost.example.com", want: false},
		{url: "https://user:password@example.com", want: false},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			err := validateServerURL(test.url)
			assert.Equal(t, test.want, err == nil)
		})
	}
}

func TestValidateTaskRejectsInvalidMockSettings(t *testing.T) {
	valid := loadTestTask{
		RunID: "run-1", TargetURL: "https://alltokenapi.com", APIKey: "sk-test",
		Model: "gpt-test", Endpoint: "openai", Prompt: "OK", DurationSeconds: 5,
		RequestsPerSecond: 1, Concurrency: 1, MockEnabled: true,
		MockToken:       "signed-mock-token",
		MockFailureRate: 0.25, MockFailureStatus: 503, MockLatencyMS: 100,
	}
	require.NoError(t, validateTask(valid))

	invalidRate := valid
	invalidRate.MockFailureRate = 1.1
	assert.ErrorContains(t, validateTask(invalidRate), "mock settings")

	invalidStatus := valid
	invalidStatus.MockFailureStatus = 418
	assert.ErrorContains(t, validateTask(invalidStatus), "failure status")

	valid.MockFailureStatus = 0
	require.NoError(t, validateTask(valid))

	valid.MockFailureRate = 0
	valid.MockLatencyMS = 0
	valid.MockChannels = []loadTestMockChannel{
		{Slot: 1, FailureRate: 0.1, FailureStatus: 503, LatencyMS: 50},
		{Slot: 2, FailureRate: 0.2, FailureStatus: 0, LatencyMS: 100},
		{Slot: 3, FailureRate: 0, FailureStatus: 429, LatencyMS: 0},
	}
	require.NoError(t, validateTask(valid))
}

func TestPreflightMockRequiresInternalExecutionMarker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Alltoken-Mock-Executed", "true")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	task := loadTestTask{RunID: "run-1", TargetURL: server.URL, APIKey: "sk-test", Model: "gpt-test", Endpoint: "openai", Prompt: "ping", MockEnabled: true, MockToken: "signed"}
	require.NoError(t, preflightMock(t.Context(), task))

	missingMarker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer missingMarker.Close()
	task.TargetURL = missingMarker.URL
	err := preflightMock(t.Context(), task)
	assert.ErrorContains(t, err, "mock preflight rejected")
}

func TestK6MockRequestsKeepSignedRunIDAndUseUniqueTraceID(t *testing.T) {
	// The mock signature is bound to the exact run ID. Per-request uniqueness
	// belongs in X-Request-ID so logs remain correlatable without invalidating
	// the server-issued signature.
	require.Contains(t, k6TaskScript, "'X-Load-Test-ID': __ENV.ALLTOKEN_RUN_ID,")
	require.Contains(t, k6TaskScript, "'X-Request-ID': __ENV.ALLTOKEN_RUN_ID + '-' + __VU + '-' + __ITER,")
	require.NotContains(t, k6TaskScript, "'X-Load-Test-ID': __ENV.ALLTOKEN_RUN_ID + '-' + __VU + '-' + __ITER,")
	require.Contains(t, k6TaskScript, "X-Alltoken-Mock-Token")
}
