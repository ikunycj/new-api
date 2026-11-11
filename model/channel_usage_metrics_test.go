package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetChannelTokenUsageAtReadsQuotaData(t *testing.T) {
	truncateTables(t)
	resetChannelTokenUsageCache()
	t.Cleanup(resetChannelTokenUsageCache)

	now := time.Date(2026, 10, 28, 12, 30, 0, 0, time.Local)
	todayHour := time.Date(2026, 10, 28, 9, 0, 0, 0, time.Local).Unix()
	earlierThisMonth := time.Date(2026, 10, 3, 15, 0, 0, 0, time.Local).Unix()
	lastMonth := time.Date(2026, 9, 30, 23, 0, 0, 0, time.Local).Unix()

	rows := []QuotaData{
		{UserID: 1, ModelName: "m", ChannelID: 7, CreatedAt: todayHour, TokenUsed: 100},
		{UserID: 2, ModelName: "m", ChannelID: 7, CreatedAt: todayHour, TokenUsed: 50},
		{UserID: 1, ModelName: "m", ChannelID: 7, CreatedAt: earlierThisMonth, TokenUsed: 1000},
		{UserID: 1, ModelName: "m", ChannelID: 7, CreatedAt: lastMonth, TokenUsed: 99999},
		{UserID: 1, ModelName: "m", ChannelID: 8, CreatedAt: earlierThisMonth, TokenUsed: 30},
		{UserID: 1, ModelName: "m", ChannelID: 9, CreatedAt: todayHour, TokenUsed: 5},
	}
	for i := range rows {
		require.NoError(t, DB.Create(&rows[i]).Error)
	}

	usage, err := GetChannelTokenUsageAt([]int{7, 8, 10}, now)
	require.NoError(t, err)
	require.Equal(t, ChannelTokenUsage{DailyTokens: 150, MonthlyTokens: 1150}, usage[7])
	require.Equal(t, ChannelTokenUsage{DailyTokens: 0, MonthlyTokens: 30}, usage[8])
	_, ok := usage[10]
	require.False(t, ok)
	_, ok = usage[9]
	require.False(t, ok, "only requested channels are returned")

	// Within the TTL a new row is not visible; after it expires it is.
	require.NoError(t, DB.Create(&QuotaData{UserID: 3, ModelName: "m", ChannelID: 8, CreatedAt: todayHour, TokenUsed: 1}).Error)
	usage, err = GetChannelTokenUsageAt([]int{8}, now.Add(10*time.Second))
	require.NoError(t, err)
	require.Equal(t, int64(30), usage[8].MonthlyTokens)
	usage, err = GetChannelTokenUsageAt([]int{8}, now.Add(2*channelTokenUsageCacheTTL))
	require.NoError(t, err)
	require.Equal(t, ChannelTokenUsage{DailyTokens: 1, MonthlyTokens: 31}, usage[8])
}
