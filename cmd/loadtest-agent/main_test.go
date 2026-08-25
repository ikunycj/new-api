package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadK6SummaryBuildsTerminalResult(t *testing.T) {
	summary := `{
  "metrics": {
    "http_reqs": {"values": {"count": 100}},
    "http_req_failed": {"values": {"rate": 0.03, "passes": 3, "fails": 97}},
    "dropped_iterations": {"values": {"count": 2}},
    "http_req_duration": {"values": {"med": 120, "p(95)": 400, "p(99)": 800}},
    "alltoken_status_429": {"values": {"count": 2}},
    "alltoken_status_500": {"values": {"count": 1}},
    "alltoken_errors{code:alltoken.upstream_exhausted}": {"values": {"count": 2}},
    "alltoken_errors{code:provider_error}": {"values": {"count": 1}},
    "alltoken_input_tokens": {"values": {"count": 1000}},
    "alltoken_output_tokens": {"values": {"count": 200}},
    "alltoken_cache_read_tokens": {"values": {"count": 600}},
    "alltoken_cache_write_tokens": {"values": {"count": 100}}
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
