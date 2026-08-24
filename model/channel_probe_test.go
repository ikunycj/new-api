package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelProbeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&ChannelProbeState{}, &ChannelProbeHistory{}))

	previousDB := DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	DB = testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		initCol()
	})
	return testDB
}

func TestClaimChannelProbeAllowsOnlyOneActiveLease(t *testing.T) {
	db := setupChannelProbeTestDB(t)

	claimed, err := ClaimChannelProbe(1001, 100, 300)
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = ClaimChannelProbe(1001, 100, 300)
	require.NoError(t, err)
	assert.False(t, claimed)

	var state ChannelProbeState
	require.NoError(t, db.First(&state, "channel_id = ?", 1001).Error)
	assert.Equal(t, int64(400), state.LeaseUntil)

	claimed, err = ClaimChannelProbe(1001, 400, 300)
	require.NoError(t, err)
	assert.True(t, claimed)
	assert.Equal(t, int64(700), readProbeLease(t, db, 1001))
}

func TestReleaseChannelProbeDoesNotClearAReplacementLease(t *testing.T) {
	db := setupChannelProbeTestDB(t)

	claimed, err := ClaimChannelProbe(1002, 100, 300)
	require.NoError(t, err)
	require.True(t, claimed)
	claimed, err = ClaimChannelProbe(1002, 401, 300)
	require.NoError(t, err)
	require.True(t, claimed)

	require.NoError(t, ReleaseChannelProbe(1002, 400))
	assert.Equal(t, int64(701), readProbeLease(t, db, 1002))

	require.NoError(t, ReleaseChannelProbe(1002, 701))
	assert.Zero(t, readProbeLease(t, db, 1002))
}

func TestSaveChannelProbeResultWithLeaseRejectsStaleOwner(t *testing.T) {
	db := setupChannelProbeTestDB(t)

	claimed, err := ClaimChannelProbe(1003, 100, 300)
	require.NoError(t, err)
	require.True(t, claimed)
	claimed, err = ClaimChannelProbe(1003, 401, 300)
	require.NoError(t, err)
	require.True(t, claimed)

	err = SaveChannelProbeResultWithLease(1003, ChannelProbeHistory{
		ChannelID: 1003,
		Success:   false,
		CheckedAt: 450,
	}, 500, 400)
	require.Error(t, err)
	assert.Equal(t, int64(701), readProbeLease(t, db, 1003))

	var historyCount int64
	require.NoError(t, db.Model(&ChannelProbeHistory{}).Where("channel_id = ?", 1003).Count(&historyCount).Error)
	assert.Zero(t, historyCount, "a stale write must roll back its history row")

	require.NoError(t, SaveChannelProbeResultWithLease(1003, ChannelProbeHistory{
		ChannelID: 1003,
		Success:   true,
		LatencyMs: 42,
		CheckedAt: 500,
	}, 800, 701))
	assert.Zero(t, readProbeLease(t, db, 1003))

	var stored ChannelProbeHistory
	require.NoError(t, db.Where("channel_id = ?", 1003).First(&stored).Error)
	assert.True(t, stored.Success)
	assert.Equal(t, int64(42), stored.LatencyMs)
	var state ChannelProbeState
	require.NoError(t, db.First(&state, "channel_id = ?", 1003).Error)
	assert.Equal(t, int64(800), state.NextProbeAt)
	assert.Equal(t, int64(500), state.LastProbeAt)
}

func TestSaveChannelProbeResultUsesAuthoritativeChannelIDAndCleansGlobalHistory(t *testing.T) {
	db := setupChannelProbeTestDB(t)
	require.NoError(t, db.Create(&ChannelProbeState{ChannelID: 1006}).Error)
	require.NoError(t, db.Create(&ChannelProbeHistory{ChannelID: 2000, CheckedAt: 1}).Error)

	require.NoError(t, SaveChannelProbeResult(1006, ChannelProbeHistory{
		ChannelID: 9999,
		Success:   true,
		CheckedAt: int64(channelProbeHistoryRetention/time.Second) + 10,
	}, int64(channelProbeHistoryRetention/time.Second)+20))

	var stored []ChannelProbeHistory
	require.NoError(t, db.Order("id ASC").Find(&stored).Error)
	require.Len(t, stored, 1)
	assert.Equal(t, 1006, stored[0].ChannelID)
}

func TestChannelProbeSettingsKeepIntervalsAndRetryDefaultsSeparate(t *testing.T) {
	retries := 0
	channel := &Channel{
		ProbeIntervalSeconds:             17,
		AutoDisabledProbeIntervalSeconds: 43,
		UpstreamMaxRetries:               &retries,
	}
	assert.Equal(t, 17, channel.GetProbeIntervalSeconds())
	assert.Equal(t, 43, channel.GetAutoDisabledProbeIntervalSeconds())
	assert.Zero(t, channel.GetUpstreamMaxRetries())

	channel.ProbeIntervalSeconds = 0
	channel.AutoDisabledProbeIntervalSeconds = 0
	channel.UpstreamMaxRetries = nil
	assert.Equal(t, DefaultChannelProbeIntervalSeconds, channel.GetProbeIntervalSeconds())
	assert.Equal(t, DefaultAutoDisabledProbeIntervalSeconds, channel.GetAutoDisabledProbeIntervalSeconds())
	assert.Equal(t, DefaultChannelUpstreamMaxRetries, channel.GetUpstreamMaxRetries())

	falseValue := false
	trueValue := true
	channel.ProbeFailureAutoBan = &falseValue
	channel.ProbeSuccessAutoEnable = &trueValue
	assert.False(t, channel.ShouldProbeFailureAutoBan())
	assert.True(t, channel.ShouldProbeSuccessAutoEnable())
}

func TestPreviousNaturalDaySuccessRateUsesPreviousLocalDayAndDefaultsTo100(t *testing.T) {
	db := setupChannelProbeTestDB(t)
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.Local)
	start, end := PreviousNaturalDayBounds(now)

	require.NoError(t, db.Create(&[]ChannelProbeHistory{
		{ChannelID: 1004, Success: true, CheckedAt: start + 1},
		{ChannelID: 1004, Success: false, CheckedAt: end - 1},
		{ChannelID: 1004, Success: true, CheckedAt: end},
	}).Error)

	rate, err := GetPreviousDayChannelProbeSuccessRate(1004, now)
	require.NoError(t, err)
	assert.Equal(t, float64(50), rate)

	rate, err = GetPreviousDayChannelProbeSuccessRate(1005, now)
	require.NoError(t, err)
	assert.Equal(t, float64(100), rate)
}

func readProbeLease(t *testing.T, db *gorm.DB, channelID int) int64 {
	t.Helper()
	var state ChannelProbeState
	require.NoError(t, db.First(&state, "channel_id = ?", channelID).Error)
	return state.LeaseUntil
}
