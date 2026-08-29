package controller

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func TestValidateLoadTestMockSettings(t *testing.T) {
	managed := &model.LoadTestAgent{Managed: true, Version: loadTestMockAgentVersion}
	require.NoError(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockFailureRate: 0.25, MockFailureStatus: 503, MockLatencyMS: 500,
	}))
	require.NoError(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockFailureRate: 0.25, MockFailureStatus: 0,
	}))
	assert.ErrorContains(t, validateLoadTestMockSettings(&model.LoadTestAgent{}, createLoadTestRunRequest{
		MockEnabled: true, MockFailureStatus: 503,
	}), "only available on server")
	assert.ErrorContains(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockFailureRate: 1.1, MockFailureStatus: 503,
	}), "between 0 and 1")
	assert.ErrorContains(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockFailureStatus: 418,
	}), "unsupported")
	assert.ErrorContains(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockFailureStatus: 503,
	}), "require mock mode")
	assert.ErrorContains(t, validateLoadTestMockSettings(&model.LoadTestAgent{Managed: true, Version: "0.2.0"}, createLoadTestRunRequest{
		MockEnabled: true,
	}), "must be updated")

	channels := []model.LoadTestMockChannel{
		{Slot: 1, FailureRate: 0.1, FailureStatus: 503, LatencyMS: 50},
		{Slot: 2, FailureRate: 0.2, FailureStatus: 0, LatencyMS: 100},
		{Slot: 3, FailureRate: 0, FailureStatus: 429, LatencyMS: 0},
	}
	require.NoError(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockChannels: channels,
	}))
	assert.ErrorContains(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockChannels: channels[:2],
	}), "exactly 3")
	invalidChannels := append([]model.LoadTestMockChannel(nil), channels...)
	invalidChannels[2].Slot = 2
	assert.ErrorContains(t, validateLoadTestMockSettings(managed, createLoadTestRunRequest{
		MockEnabled: true, MockChannels: invalidChannels,
	}), "unique")
}

func TestValidateLoadTestMockTokenAllowsMixedRealChannels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.BillingGroupRoute{}, &model.BillingGroupChannel{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		model.InitChannelRoutingCache()
	})

	mockSetting := `{"mock_load_test":true}`
	realSetting := `{}`
	channels := []model.Channel{
		{Id: 101, Name: "mock-a", Status: common.ChannelStatusEnabled, Setting: &mockSetting},
		{Id: 102, Name: "real", Status: common.ChannelStatusEnabled, Setting: &realSetting},
		{Id: 103, Name: "mock-b", Status: common.ChannelStatusEnabled, Setting: &mockSetting},
		{Id: 104, Name: "mock-c", Status: common.ChannelStatusEnabled, Setting: &mockSetting},
	}
	require.NoError(t, db.Create(&channels).Error)
	route := model.BillingGroupRoute{BillingGroup: "mock-only", Enabled: true}
	require.NoError(t, db.Create(&route).Error)
	require.NoError(t, db.Create(&[]model.BillingGroupChannel{
		{BillingGroupRouteId: route.Id, ChannelId: 101, Priority: 100, Enabled: true},
		{BillingGroupRouteId: route.Id, ChannelId: 102, Priority: 90, Enabled: true},
		{BillingGroupRouteId: route.Id, ChannelId: 103, Priority: 80, Enabled: true},
		{BillingGroupRouteId: route.Id, ChannelId: 104, Priority: 70, Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "mock-only", Model: "gpt-test", ChannelId: 101, Enabled: true},
		{Group: "mock-only", Model: "gpt-test", ChannelId: 103, Enabled: true},
		{Group: "mock-only", Model: "gpt-test", ChannelId: 104, Enabled: true},
	}).Error)
	model.InitChannelRoutingCache()

	token := &model.Token{Group: "mock-only"}
	require.NoError(t, validateLoadTestMockToken(token, "gpt-test", "openai"))

	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", 101).Update("setting", realSetting).Error)
	model.InitChannelRoutingCache()
	assert.ErrorContains(t, validateLoadTestMockToken(token, "gpt-test", "openai"), "requires 3 enabled mock channels")
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
