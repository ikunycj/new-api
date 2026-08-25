package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingGroupMetricsAggregateUsageAndChannelCounts(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Log{})
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.Local)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	require.NoError(t, DB.Create(&[]Channel{
		{Name: "shared", Key: "key-1", Group: "paid,backup", Status: common.ChannelStatusEnabled},
		{Name: "disabled", Key: "key-2", Group: "paid", Status: common.ChannelStatusManuallyDisabled},
		{Name: "backup", Key: "key-3", Group: "backup", Status: common.ChannelStatusEnabled},
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Type: LogTypeConsume, Group: "paid", CreatedAt: todayStart.Add(time.Hour).Unix(), PromptTokens: 100, CompletionTokens: 20, Quota: 300},
		{Type: LogTypeConsume, Group: "paid", CreatedAt: yesterdayStart.Add(time.Hour).Unix(), PromptTokens: 200, CompletionTokens: 50, Quota: 600},
		{Type: LogTypeConsume, Group: "paid", CreatedAt: yesterdayStart.AddDate(0, 0, -1).Unix(), PromptTokens: 10, CompletionTokens: 5, Quota: 40},
		{Type: LogTypeConsume, Group: "backup", CreatedAt: todayStart.Add(2 * time.Hour).Unix(), PromptTokens: 30, CompletionTokens: 10, Quota: 90},
		{Type: LogTypeError, Group: "paid", CreatedAt: todayStart.Add(3 * time.Hour).Unix(), PromptTokens: 999, CompletionTokens: 999, Quota: 999},
	}).Error)

	usage, err := GetPricingGroupUsageAt(now)
	require.NoError(t, err)
	assert.Equal(t, PricingGroupUsage{
		Today:     PricingGroupUsagePeriod{Tokens: 120, Quota: 300},
		Yesterday: PricingGroupUsagePeriod{Tokens: 250, Quota: 600},
		Total:     PricingGroupUsagePeriod{Tokens: 385, Quota: 940},
	}, usage["paid"])
	assert.Equal(t, PricingGroupUsagePeriod{Tokens: 40, Quota: 90}, usage["backup"].Today)

	channels, err := GetPricingGroupChannelCounts()
	require.NoError(t, err)
	assert.Equal(t, PricingGroupChannelCount{Available: 1, Total: 2}, channels["paid"])
	assert.Equal(t, PricingGroupChannelCount{Available: 2, Total: 2}, channels["backup"])
}
