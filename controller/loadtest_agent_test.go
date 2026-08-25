package controller

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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
