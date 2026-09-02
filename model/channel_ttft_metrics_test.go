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
