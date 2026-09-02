package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEligibleChannelsLoadProbeRatesWithoutMemoryCache(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &ChannelProbeHistory{}, &Log{}))
	for _, table := range []string{"channel_probe_histories", "logs", "abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"channel_probe_histories", "logs", "abilities", "channels"} {
			_ = DB.Exec("DELETE FROM " + table).Error
		}
	})

	channels := []Channel{
		{Id: 94001, Name: "probe-low", Key: "probe-low", Models: "shared-model", Group: "probe", Status: common.ChannelStatusEnabled},
		{Id: 94002, Name: "probe-high", Key: "probe-high", Models: "shared-model", Group: "probe", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&channels).Error)
	for index := range channels {
		require.NoError(t, channels[index].UpdateAbilities(DB))
	}
	start, _ := PreviousNaturalDayBounds(time.Now())
	require.NoError(t, DB.Create(&[]ChannelProbeHistory{
		{ChannelID: channels[0].Id, Success: false, CheckedAt: start + 1},
		{ChannelID: channels[1].Id, Success: true, CheckedAt: start + 1},
	}).Error)
	ttftOther, err := common.Marshal(map[string]any{"frt": 120.5})
	require.NoError(t, err)
	require.NoError(t, LOG_DB.Create(&Log{
		ChannelId: channels[1].Id,
		Type:      LogTypeConsume,
		IsStream:  true,
		CreatedAt: start + 2,
		Other:     string(ttftOther),
	}).Error)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCacheEnabled })

	eligible, err := GetEligibleChannels("probe", "shared-model", "", nil)
	require.NoError(t, err)
	require.Len(t, eligible, 2)
	assert.Equal(t, float64(0), eligible[0].PreviousDayProbeSuccessRate)
	assert.Equal(t, float64(100), eligible[1].PreviousDayProbeSuccessRate)
	assert.InDelta(t, 120.5, eligible[1].PreviousDayAverageTTFTMs, 0.000001)
	assert.Zero(t, eligible[0].PreviousDayAverageTTFTMs)
}

func TestMemoryChannelCacheLoadsPreviousDayAverageTTFT(t *testing.T) {
	const channelID = 94005
	group := "ttft-cache"
	modelName := "ttft-model"
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &Log{}))
	_ = DB.Where("channel_id = ?", channelID).Delete(&Ability{}).Error
	_ = DB.Where("channel_id = ?", channelID).Delete(&Log{}).Error
	_ = DB.Where("id = ?", channelID).Delete(&Channel{}).Error

	channel := &Channel{
		Id:     channelID,
		Name:   "ttft-cache",
		Key:    "ttft-cache-key",
		Models: modelName,
		Group:  group,
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: channelID,
		Enabled:   true,
	}).Error)
	start, _ := PreviousNaturalDayBounds(time.Now())
	firstOther, err := common.Marshal(map[string]any{"frt": 80})
	require.NoError(t, err)
	secondOther, err := common.Marshal(map[string]any{"frt": 120})
	require.NoError(t, err)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: channelID, Type: LogTypeConsume, IsStream: true, CreatedAt: start + 1, Other: string(firstOther)},
		{ChannelId: channelID, Type: LogTypeConsume, IsStream: true, CreatedAt: start + 2, Other: string(secondOther)},
	}).Error)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		_ = DB.Where("channel_id = ?", channelID).Delete(&Ability{}).Error
		_ = DB.Where("channel_id = ?", channelID).Delete(&Log{}).Error
		_ = DB.Where("id = ?", channelID).Delete(&Channel{}).Error
		common.MemoryCacheEnabled = true
		InitChannelCache()
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	InitChannelCache()
	eligible, err := GetEligibleChannels(group, modelName, "", nil)
	require.NoError(t, err)
	require.Len(t, eligible, 1)
	assert.InDelta(t, 100, eligible[0].PreviousDayAverageTTFTMs, 0.000001)
}

func TestPreviousNaturalDayBoundsUseLocalCalendarMidnights(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	previousLocal := time.Local
	time.Local = location
	t.Cleanup(func() { time.Local = previousLocal })

	tests := []struct {
		name        string
		now         time.Time
		expectedGap int64
	}{
		{
			name:        "spring daylight saving transition",
			now:         time.Date(2026, time.March, 9, 12, 0, 0, 0, location),
			expectedGap: 23 * 60 * 60,
		},
		{
			name:        "autumn daylight saving transition",
			now:         time.Date(2026, time.November, 2, 12, 0, 0, 0, location),
			expectedGap: 25 * 60 * 60,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end := PreviousNaturalDayBounds(test.now)
			todayStart := time.Date(2026, test.now.Month(), test.now.Day(), 0, 0, 0, 0, location)
			assert.Equal(t, todayStart.AddDate(0, 0, -1).Unix(), start)
			assert.Equal(t, todayStart.Unix(), end)
			assert.Equal(t, test.expectedGap, end-start)
		})
	}
}

func TestMemoryChannelCacheHonorsDisabledAbilities(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}))
	for _, table := range []string{"abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"abilities", "channels"} {
			_ = DB.Exec("DELETE FROM " + table).Error
		}
		InitChannelCache()
	})

	channel := &Channel{
		Id:     94004,
		Name:   "disabled-ability",
		Key:    "disabled-ability-key",
		Models: "shared-model",
		Group:  "pricing",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "pricing",
		Model:     "shared-model",
		ChannelId: channel.Id,
		Enabled:   false,
	}).Error)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	InitChannelCache()
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCacheEnabled })

	assert.False(t, IsChannelEnabledForGroupModel("pricing", "shared-model", channel.Id))
	channels, err := GetEligibleChannels("pricing", "shared-model", "", nil)
	require.NoError(t, err)
	assert.Empty(t, channels)
}
