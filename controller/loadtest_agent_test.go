package controller

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidLoadTestProgress(t *testing.T) {
	tests := []struct {
		name        string
		sent        int64
		completed   int64
		successes   int64
		failures    int64
		dropped     int64
		currentRPS  float64
		errorCounts map[string]int64
		want        bool
	}{
		{name: "heartbeat without completed requests", currentRPS: 20, want: true},
		{name: "complete counters", sent: 100, completed: 98, successes: 90, failures: 8, dropped: 2, currentRPS: 20, errorCounts: map[string]int64{"429": 8}, want: true},
		{name: "completed exceeds sent", sent: 10, completed: 11, want: false},
		{name: "classified results exceed completed", sent: 10, completed: 10, successes: 9, failures: 2, want: false},
		{name: "negative error count", errorCounts: map[string]int64{"500": -1}, want: false},
		{name: "non finite rate", currentRPS: math.Inf(1), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := validLoadTestProgress(test.sent, test.completed, test.successes, test.failures, test.dropped, test.currentRPS, 0, 0, 0, test.errorCounts)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestValidateLoadTestAgentCapacity(t *testing.T) {
	tests := []struct {
		name    string
		agent   model.LoadTestAgent
		request createLoadTestRunRequest
		message string
	}{
		{
			name:    "managed capacity accepts bounded task",
			agent:   model.LoadTestAgent{Managed: true, MaxRPS: 200, MaxConcurrency: 500},
			request: createLoadTestRunRequest{RequestsPerSecond: 100, Concurrency: 300},
		},
		{
			name:    "managed agent must report capacity",
			agent:   model.LoadTestAgent{Managed: true},
			request: createLoadTestRunRequest{RequestsPerSecond: 1, Concurrency: 1},
			message: "has not reported capacity",
		},
		{
			name:    "RPS cannot exceed agent capacity",
			agent:   model.LoadTestAgent{Managed: true, MaxRPS: 100, MaxConcurrency: 500},
			request: createLoadTestRunRequest{RequestsPerSecond: 101, Concurrency: 100},
			message: "at most 100 RPS",
		},
		{
			name:    "concurrency cannot exceed agent capacity",
			agent:   model.LoadTestAgent{Managed: true, MaxRPS: 200, MaxConcurrency: 50},
			request: createLoadTestRunRequest{RequestsPerSecond: 20, Concurrency: 51},
			message: "at most 50 concurrent requests",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateLoadTestAgentCapacity(&test.agent, test.request)
			if test.message == "" {
				require.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, test.message)
		})
	}
}
