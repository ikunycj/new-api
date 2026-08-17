package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPackageComparisonStatsAggregatesConsumptionAndErrors(t *testing.T) {
	now := time.Now().Unix()
	plans := []SubscriptionPlan{
		{Title: "comparison-basic", PriceAmount: 10, Currency: "USD", TotalAmount: 1000, RoutingStrategy: RoutingStrategyCostFirst},
		{Title: "comparison-pro", PriceAmount: 20, Currency: "USD", TotalAmount: 2000, RoutingStrategy: RoutingStrategyStabilityFirst},
	}
	require.NoError(t, DB.Create(&plans).Error)

	logs := []Log{
		{CreatedAt: now, Type: LogTypeConsume, SubscriptionPlanId: plans[0].Id, PromptTokens: 100, CompletionTokens: 40, Quota: 140, UseTime: 2},
		{CreatedAt: now, Type: LogTypeError, SubscriptionPlanId: plans[0].Id, UseTime: 1},
		{CreatedAt: now, Type: LogTypeConsume, SubscriptionPlanId: plans[1].Id, PromptTokens: 20, CompletionTokens: 10, Quota: 30, UseTime: 3},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	stats, err := GetPackageComparisonStats(
		[]int{plans[0].Id, plans[1].Id},
		now-1,
		now+1,
		"",
		"",
	)
	require.NoError(t, err)
	require.Len(t, stats, 2)

	assert.Equal(t, int64(2), stats[0].Requests)
	assert.Equal(t, int64(1), stats[0].SuccessRequests)
	assert.Equal(t, int64(1), stats[0].ErrorRequests)
	assert.Equal(t, int64(100), stats[0].PromptTokens)
	assert.Equal(t, int64(40), stats[0].CompletionTokens)
	assert.Equal(t, int64(140), stats[0].Quota)
	assert.InDelta(t, 0.5, stats[0].ChannelHitRate, 0.001)
	assert.InDelta(t, 0.5, stats[0].FailureRate, 0.001)
	assert.InDelta(t, 1_000_000, stats[0].QuotaPerMillionTokens, 0.001)
	assert.Equal(t, RoutingStrategyCostFirst, stats[0].RoutingStrategy)
	assert.InDelta(t, 2000, stats[0].AverageLatencyMs, 0.001)

	assert.Equal(t, int64(1), stats[1].Requests)
	assert.Equal(t, int64(30), stats[1].TotalTokens)
	assert.Equal(t, "comparison-pro", stats[1].PlanTitle)
	assert.Equal(t, 20.0, stats[1].PlanPrice)
}
