package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelUsageReconstructsReportedCostAndIncludesProbes(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})
	previousUnit := common.QuotaPerUnit
	t.Cleanup(func() { common.QuotaPerUnit = previousUnit })
	common.QuotaPerUnit = 500000
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.Local)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: 60, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 8164502, CompletionTokens: 50604, Quota: 5586840, Other: `{"group_ratio":0.06,"billing_usd_to_cny_rate":7}`},
		{ChannelId: 60, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 76408, CompletionTokens: 855, Quota: 156113, Content: "渠道探测", Other: `{"group_ratio":1,"billing_usd_to_cny_rate":7}`},
		{ChannelId: 60, Type: LogTypeError, CreatedAt: now.Unix(), PromptTokens: 999, Quota: 99999},
		{ChannelId: 61, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 999, Quota: 99999},
	}).Error)
	usage, err := GetChannelUsageAt([]int{60}, now)
	require.NoError(t, err)
	require.Len(t, usage, 1)
	row := usage[60]
	assert.EqualValues(t, 8292369, row.DailyTokens)
	assert.Equal(t, row.DailyTokens, row.TotalTokens)
	assert.False(t, row.DailyCostIncomplete)
	assert.False(t, row.TotalCostIncomplete)
	usd := &Channel{PriceMultiplier: 0.2, PriceMultiplierMode: ChannelPriceMultiplierModeUSD}
	cny := &Channel{PriceMultiplier: 0.2, PriceMultiplierMode: ChannelPriceMultiplierModeCNY}
	for _, cost := range []*float64{
		usd.EstimateCostCNY(row.DailyBaseCostUSD, row.DailyBaseCostCNY),
		usd.EstimateCostCNY(row.TotalBaseCostUSD, row.TotalBaseCostCNY),
	} {
		require.NotNil(t, cost)
		assert.InDelta(t, 5.329720742857143, *cost, 1e-9)
	}
	cost := cny.EstimateCostCNY(row.TotalBaseCostUSD, row.TotalBaseCostCNY)
	require.NotNil(t, cost)
	assert.InDelta(t, 37.3080452, *cost, 1e-9)
}

func TestChannelUsageUsesCalendarDayAndFrozenPricing(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})
	previousUnit := common.QuotaPerUnit
	t.Cleanup(func() { common.QuotaPerUnit = previousUnit })
	common.QuotaPerUnit = 500000
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.Local)
	start := time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local)
	require.NoError(t, LOG_DB.Create(&[]Log{
		// Each log represents $1 of model usage, with different sale terms.
		{ChannelId: 1, Type: LogTypeConsume, CreatedAt: start.Unix(), PromptTokens: 10, CompletionTokens: 1, Quota: 210000, Other: `{"group_ratio":0.06,"billing_usd_to_cny_rate":7}`},
		{ChannelId: 1, Type: LogTypeConsume, CreatedAt: start.Add(23 * time.Hour).Unix(), PromptTokens: 20, CompletionTokens: 2, Quota: 400000, Other: `{"group_ratio":0.1,"billing_usd_to_cny_rate":8}`},
		{ChannelId: 1, Type: LogTypeConsume, CreatedAt: start.Add(-time.Second).Unix(), PromptTokens: 30, CompletionTokens: 3, Quota: 3500000, Other: `{"group_ratio":1,"billing_usd_to_cny_rate":7}`},
		{ChannelId: 1, Type: LogTypeConsume, CreatedAt: start.AddDate(0, 0, 1).Unix(), PromptTokens: 40, CompletionTokens: 4, Quota: 2000000, Other: `{"group_ratio":0.5,"billing_usd_to_cny_rate":8}`},
		// Per-call billing has no token volume but still has a real cost.
		{ChannelId: 2, Type: LogTypeConsume, CreatedAt: now.Unix(), Quota: 5250000, Other: `{"group_ratio":0.06,"billing_usd_to_cny_rate":7,"model_price":25}`},
	}).Error)
	usage, err := GetChannelUsageAt([]int{1, 2}, now)
	require.NoError(t, err)
	assert.Equal(t, ChannelUsage{DailyTokens: 33, TotalTokens: 110, DailyBaseCostUSD: 2, TotalBaseCostUSD: 4, DailyBaseCostCNY: 15, TotalBaseCostCNY: 30}, usage[1])
	assert.InDelta(t, 25, usage[2].DailyBaseCostUSD, 1e-9)
	assert.InDelta(t, 175, usage[2].DailyBaseCostCNY, 1e-9)
	assert.Zero(t, usage[2].DailyTokens)
}

func TestChannelUsageMarksUnrecoverableCostsWithoutLosingTokens(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.Local)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: 1, Type: LogTypeConsume, CreatedAt: now.AddDate(0, 0, -1).Unix(), PromptTokens: 100, Other: ""},
		{ChannelId: 1, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 20, Other: `{"group_ratio":1,"billing_usd_to_cny_rate":7}`},
		{ChannelId: 2, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 30, Other: `{"group_ratio":0,"billing_usd_to_cny_rate":7}`},
		{ChannelId: 3, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 40, Other: `{"group_ratio":1,"billing_usd_to_cny_rate":0}`},
		{ChannelId: 4, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 50, Other: `{"group_ratio":"NaN"}`},
		{ChannelId: 5, Type: LogTypeConsume, CreatedAt: now.Unix(), PromptTokens: 60, Other: `{"group_ratio":1}`},
	}).Error)
	usage, err := GetChannelUsageAt([]int{1, 2, 3, 4, 5}, now)
	require.NoError(t, err)
	assert.EqualValues(t, 120, usage[1].TotalTokens)
	assert.EqualValues(t, 20, usage[1].DailyTokens)
	assert.True(t, usage[1].TotalCostIncomplete)
	assert.False(t, usage[1].DailyCostIncomplete)
	for _, id := range []int{2, 3, 4} {
		assert.True(t, usage[id].DailyCostIncomplete)
		assert.True(t, usage[id].TotalCostIncomplete)
		assert.Positive(t, usage[id].TotalTokens)
	}
	assert.False(t, usage[5].TotalCostIncomplete)
}
