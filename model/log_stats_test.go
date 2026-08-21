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

func TestSumUsedQuotaAggregatesCacheHitRateFromEligibleLogs(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Log{}))

	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	DB, LOG_DB = testDB, testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		initCol()
	})

	now := time.Now().Unix()
	require.NoError(t, LOG_DB.Create(&[]Log{
		{
			UserId:              7,
			Type:                LogTypeConsume,
			CreatedAt:           now,
			InputTokensTotal:    100,
			CacheReadTokens:     40,
			CacheWriteTokens:    10,
			CacheStatsAvailable: true,
		},
		{
			UserId:              7,
			Type:                LogTypeConsume,
			CreatedAt:           now,
			InputTokensTotal:    50,
			CacheStatsAvailable: true,
		},
		{
			UserId:           7,
			Type:             LogTypeConsume,
			CreatedAt:        now,
			InputTokensTotal: 100,
			CacheReadTokens:  100,
		},
	}).Error)

	stat, err := SumUsedQuota(LogTypeConsume, now-1, now+1, "", "", "", 0, "", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 150, stat.CacheInputTokens)
	assert.EqualValues(t, 40, stat.CacheReadTokens)
	assert.EqualValues(t, 10, stat.CacheWriteTokens)
	assert.EqualValues(t, 1, stat.CacheHitRequests)
	assert.EqualValues(t, 2, stat.CacheEligibleRequests)
	assert.InDelta(t, 26.6667, stat.CacheHitRate, 0.0001)
}
