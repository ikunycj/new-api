package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChannelTokenUsageAtAggregatesCalendarDayAndMonth(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Log{})
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.Local)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	channels := []Channel{{Name: "usage-a", Key: "key-a"}, {Name: "usage-b", Key: "key-b"}}
	require.NoError(t, DB.Create(&channels).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: channels[0].Id, Type: LogTypeConsume, CreatedAt: todayStart.Add(time.Hour).Unix(), PromptTokens: 100, CompletionTokens: 20},
		{ChannelId: channels[0].Id, Type: LogTypeConsume, CreatedAt: monthStart.Add(time.Hour).Unix(), PromptTokens: 200, CompletionTokens: 50},
		{ChannelId: channels[0].Id, Type: LogTypeError, CreatedAt: todayStart.Add(2 * time.Hour).Unix(), PromptTokens: 999, CompletionTokens: 999},
		{ChannelId: channels[0].Id, Type: LogTypeConsume, CreatedAt: monthStart.Add(-time.Second).Unix(), PromptTokens: 999, CompletionTokens: 999},
		{ChannelId: channels[1].Id, Type: LogTypeConsume, CreatedAt: todayStart.Add(3 * time.Hour).Unix(), PromptTokens: 30, CompletionTokens: 10},
	}).Error)

	usage, err := GetChannelTokenUsageAt([]int{channels[0].Id, channels[1].Id}, now)
	require.NoError(t, err)
	assert.Equal(t, ChannelTokenUsage{DailyTokens: 120, MonthlyTokens: 370}, usage[channels[0].Id])
	assert.Equal(t, ChannelTokenUsage{DailyTokens: 40, MonthlyTokens: 40}, usage[channels[1].Id])
}
