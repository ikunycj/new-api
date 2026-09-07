package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPreviousDayChannelAverageTTFTsUsesValidStreamingConsumeLogs(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.Local)
	start, end := PreviousNaturalDayBounds(now)

	validOther1, err := common.Marshal(map[string]any{"frt": 100.5})
	require.NoError(t, err)
	validOther2, err := common.Marshal(map[string]any{"frt": 200.5})
	require.NoError(t, err)
	todayOther, err := common.Marshal(map[string]any{"frt": 900})
	require.NoError(t, err)
	nonStreamOther, err := common.Marshal(map[string]any{"frt": 400})
	require.NoError(t, err)
	negativeOther, err := common.Marshal(map[string]any{"frt": -1})
	require.NoError(t, err)

	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: 1001, Type: LogTypeConsume, IsStream: true, CreatedAt: start + 1, Other: string(validOther1)},
		{ChannelId: 1001, Type: LogTypeConsume, IsStream: true, CreatedAt: end - 1, Other: string(validOther2)},
		{ChannelId: 1001, Type: LogTypeConsume, IsStream: true, CreatedAt: end, Other: string(todayOther)},
		{ChannelId: 1001, Type: LogTypeError, IsStream: true, CreatedAt: start + 2, Other: string(todayOther)},
		{ChannelId: 1001, Type: LogTypeConsume, IsStream: false, CreatedAt: start + 3, Other: string(nonStreamOther)},
		{ChannelId: 1001, Type: LogTypeConsume, IsStream: true, CreatedAt: start + 4, Other: string(negativeOther)},
		{ChannelId: 1001, Type: LogTypeConsume, IsStream: true, CreatedAt: start + 5, Other: "not-json"},
	}).Error)

	averages, err := GetPreviousDayChannelAverageTTFTs([]int{1001, 1002}, now)
	require.NoError(t, err)
	assert.InDelta(t, 150.5, averages[1001], 0.000001)
	assert.Zero(t, averages[1002])
}

func TestGetLatestChannelTestTTFTsIncludesManualAndAutomaticTests(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})

	manualTTFT, err := common.Marshal(map[string]any{"frt": 80})
	require.NoError(t, err)
	automaticTTFT, err := common.Marshal(map[string]any{"frt": 120})
	require.NoError(t, err)
	olderTTFT, err := common.Marshal(map[string]any{"frt": 90})
	require.NoError(t, err)
	invalidTTFT, err := common.Marshal(map[string]any{"frt": -1})
	require.NoError(t, err)

	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: 1001, Type: LogTypeConsume, TokenName: ChannelTestTokenName, CreatedAt: 100, Other: string(manualTTFT)},
		{ChannelId: 1001, Type: LogTypeConsume, TokenName: ChannelProbeTokenName, CreatedAt: 200, Other: string(automaticTTFT)},
		{ChannelId: 1001, Type: LogTypeConsume, TokenName: "普通请求", CreatedAt: 300, Other: string(invalidTTFT)},
		{ChannelId: 1002, Type: LogTypeConsume, TokenName: ChannelTestTokenName, CreatedAt: 100, Other: string(olderTTFT)},
		{ChannelId: 1002, Type: LogTypeConsume, TokenName: ChannelProbeTokenName, CreatedAt: 200, Other: string(invalidTTFT)},
	}).Error)

	ttfts, err := GetLatestChannelTestTTFTs([]int{1001, 1002, 1003})
	require.NoError(t, err)
	assert.Equal(t, float64(120), ttfts[1001])
	assert.Equal(t, float64(90), ttfts[1002])
	assert.Zero(t, ttfts[1003])
}
