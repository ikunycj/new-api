package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetLogAnalyticsAggregatesRequestsAndSearchesUserIDs(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&User{}, &Log{}))

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

	require.NoError(t, DB.Create(&User{Id: 7, Username: "alice", Email: "alice@example.com", Remark: "Finance"}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 7, Username: "alice", Type: LogTypeConsume, CreatedAt: 100, UseTime: 2, PromptTokens: 10, CompletionTokens: 5, InputTokensTotal: 20, CacheReadTokens: 5, CacheWriteTokens: 2, Quota: 100, Group: "paid", ModelName: "model-a"},
		{UserId: 8, Username: "bob", Type: LogTypeConsume, CreatedAt: 200, UseTime: 4, PromptTokens: 20, CompletionTokens: 10, InputTokensTotal: 30, CacheReadTokens: 15, Quota: 200, Group: "free", ModelName: "model-b"},
		{UserId: 7, Username: "alice", Type: LogTypeError, CreatedAt: 300, UseTime: 8, Quota: 50, Group: "paid", ModelName: "model-a"},
		{UserId: 999, Username: "ghost", Type: LogTypeConsume, CreatedAt: 400, PromptTokens: 1, CompletionTokens: 1, Quota: 1},
	}).Error)

	analytics, err := GetLogAnalytics(LogAnalyticsFilters{StartTimestamp: 1, EndTimestamp: 300, Granularity: "day", UserLimit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), analytics.Summary.TotalRequests)
	assert.Equal(t, int64(2), analytics.Summary.SuccessRequests)
	assert.Equal(t, int64(1), analytics.Summary.FailedRequests)
	assert.Equal(t, int64(30), analytics.Summary.InputTokens)
	assert.Equal(t, int64(15), analytics.Summary.OutputTokens)
	assert.Equal(t, int64(20), analytics.Summary.CacheReadTokens)
	assert.Equal(t, int64(2), analytics.Summary.CacheWriteTokens)
	assert.Equal(t, int64(45), analytics.Summary.TotalTokens)
	assert.Equal(t, int64(300), analytics.Summary.TotalQuota)
	assert.InDelta(t, 8, analytics.Summary.P90UseTime, 0.001)
	assert.InDelta(t, 8, analytics.Summary.P99UseTime, 0.001)
	assert.Len(t, analytics.GroupDistribution, 2)

	filtered, err := GetLogAnalytics(LogAnalyticsFilters{Keyword: "999", StartTimestamp: 1, EndTimestamp: 500, UserLimit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), filtered.Summary.TotalRequests)
	assert.Equal(t, int64(2), filtered.Summary.TotalTokens)
}
