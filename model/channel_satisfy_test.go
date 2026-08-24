package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEligibleChannelsLoadProbeRatesWithoutMemoryCache(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &ChannelProbeHistory{}))
	for _, table := range []string{"channel_probe_histories", "abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"channel_probe_histories", "abilities", "channels"} {
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

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCacheEnabled })

	eligible, err := GetEligibleChannels("probe", "shared-model", "", nil)
	require.NoError(t, err)
	require.Len(t, eligible, 2)
	assert.Equal(t, float64(0), eligible[0].PreviousDayProbeSuccessRate)
	assert.Equal(t, float64(100), eligible[1].PreviousDayProbeSuccessRate)
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
