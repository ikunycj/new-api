package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTokenUsageMetricsAtAggregatesTodayAndLifetimeConsumeUsage(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})

	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.Local)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	require.NoError(t, LOG_DB.Create(&[]Log{
		{TokenId: 101, Type: LogTypeConsume, CreatedAt: todayStart.Add(time.Hour).Unix(), PromptTokens: 100, CompletionTokens: 20, Quota: 1100},
		{TokenId: 101, Type: LogTypeConsume, CreatedAt: yesterdayStart.Add(time.Hour).Unix(), PromptTokens: 200, CompletionTokens: 50, Quota: 2200},
		{TokenId: 101, Type: LogTypeError, CreatedAt: todayStart.Add(2 * time.Hour).Unix(), PromptTokens: 999, CompletionTokens: 999, Quota: 9999},
		{TokenId: 101, Type: LogTypeRefund, CreatedAt: todayStart.Add(3 * time.Hour).Unix(), PromptTokens: 500, CompletionTokens: 500, Quota: 5000},
		{TokenId: 202, Type: LogTypeConsume, CreatedAt: todayStart.Add(4 * time.Hour).Unix(), PromptTokens: 30, CompletionTokens: 10, Quota: 400},
		{TokenId: 202, Type: LogTypeConsume, CreatedAt: todayStart.AddDate(0, 0, 1).Unix(), PromptTokens: 700, CompletionTokens: 80, Quota: 7000},
	}).Error)

	usage, err := GetTokenUsageMetricsAt([]int{101, 202, 303, 101}, now)
	require.NoError(t, err)
	assert.Equal(t, TokenUsageMetrics{DailyTokens: 120, TotalTokens: 370, DailyQuota: 1100, TotalQuota: 3300}, usage[101])
	assert.Equal(t, TokenUsageMetrics{DailyTokens: 40, TotalTokens: 820, DailyQuota: 400, TotalQuota: 7400}, usage[202])
	assert.Equal(t, TokenUsageMetrics{}, usage[303])
}
