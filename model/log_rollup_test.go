package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestParseFrozenPricingDefaultsMissingBillingRate(t *testing.T) {
	previousUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousUnit })
	group, rate, incomplete := parseFrozenPricing(`{"group_ratio":"0.5"}`)
	require.Equal(t, 0.5, group)
	require.Equal(t, 1.0, rate)
	require.False(t, incomplete)
}

func TestAddRollupLogAggregatesByChannelAndDay(t *testing.T) {
	previousUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = previousUnit })
	day := time.Date(2026, 9, 18, 12, 0, 0, 0, time.Local)
	rows := make(map[int]*rollupAccumulator)
	addRollupLog(rows, &Log{CreatedAt: day.Unix(), ChannelId: 9, Type: LogTypeConsume, PromptTokens: 2, CompletionTokens: 3, Quota: 500000, Other: `{"group_ratio":1,"billing_usd_to_cny_rate":7}`}, 1)
	addRollupLog(rows, &Log{CreatedAt: day.Add(time.Hour).Unix(), ChannelId: 9, Type: LogTypeConsume, PromptTokens: 5, CompletionTokens: 7, Quota: 1000000, Other: `{"group_ratio":0.5,"billing_usd_to_cny_rate":7}`}, 2)
	require.Len(t, rows, 1)
	row := rows[9]
	require.EqualValues(t, 2, row.RequestCount)
	require.EqualValues(t, 7, row.PromptTokens)
	require.EqualValues(t, 10, row.CompletionTokens)
	require.EqualValues(t, 1500000, row.Quota)
	require.InDelta(t, 5.0/7.0, row.BaseCostUSD, 1e-9)
	require.Equal(t, localDayStart(day), row.BucketStart)
}

func TestRequiredRollupBoundaryForLogCleanup(t *testing.T) {
	day := localDayStart(time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local))
	require.Equal(t, day, requiredRollupBoundaryForLogCleanup(day))
	require.Equal(t, nextLocalDay(day), requiredRollupBoundaryForLogCleanup(day+3600))
}
