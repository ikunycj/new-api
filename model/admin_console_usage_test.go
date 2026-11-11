package model

import (
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdminConsoleUsageTest(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, testDB.AutoMigrate(&Log{}, &QuotaData{}))

	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousExport := common.DataExportEnabled
	DB, LOG_DB = testDB, testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.DataExportEnabled = previousExport
	})
}

func TestAdminConsoleUsageCombinesQuotaDataHistoryWithTodaysLogs(t *testing.T) {
	setupAdminConsoleUsageTest(t)
	common.DataExportEnabled = true
	now := time.Date(2026, 11, 20, 15, 0, 0, 0, time.Local)
	todayStart, _, _, _, monthStart := adminConsoleTimeBounds(now)
	day := int64(24 * 3600)

	require.NoError(t, DB.Create(&[]QuotaData{
		// Last month, outside the rolling 30 days.
		{UserID: 1, CreatedAt: monthStart - 20*day, Count: 5, TokenUsed: 500, Quota: 50},
		// This month, inside the last 7 days.
		{UserID: 2, CreatedAt: todayStart - 2*day, Count: 3, TokenUsed: 300, Quota: 30},
		// This month, older than 7 days.
		{UserID: 3, CreatedAt: todayStart - 10*day, Count: 2, TokenUsed: 200, Quota: 20},
		// Today's bucket overlaps the logs read below and must not be counted twice.
		{UserID: 4, CreatedAt: todayStart + 3600, Count: 4, TokenUsed: 400, Quota: 40},
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Type: LogTypeConsume, UserId: 4, TokenId: 7, CreatedAt: todayStart + 3600, PromptTokens: 10, CompletionTokens: 20, Quota: 5},
		{Type: LogTypeConsume, UserId: 2, TokenId: 8, CreatedAt: todayStart + 7200, PromptTokens: 1, CompletionTokens: 2, Quota: 1},
		{Type: LogTypeError, UserId: 9, TokenId: 9, CreatedAt: todayStart + 7200},
		// Already covered by quota_data history.
		{Type: LogTypeConsume, UserId: 2, TokenId: 8, CreatedAt: todayStart - 2*day, PromptTokens: 100, CompletionTokens: 200, Quota: 30},
	}).Error)

	row, err := getAdminConsoleUsage(now)
	require.NoError(t, err)

	assert.Equal(t, [3]int64{2, 7, 12}, [3]int64{row.RequestsToday, row.RequestsMonth, row.RequestsTotal})
	assert.Equal(t, [3]int64{33, 533, 1033}, [3]int64{row.TokensToday, row.TokensMonth, row.TokensTotal})
	assert.Equal(t, [3]int64{6, 56, 106}, [3]int64{row.QuotaToday, row.QuotaMonth, row.QuotaTotal})
	assert.Equal(t, int64(2), row.ActiveKeysToday)
	assert.Equal(t, int64(2), row.ActiveUsersToday)
	assert.Equal(t, int64(2), row.ActiveUsersWeek)
	assert.Equal(t, int64(3), row.ActiveUsersMonth)
}

func TestAdminConsoleUsageReadsLogsWhenDataExportIsDisabled(t *testing.T) {
	setupAdminConsoleUsageTest(t)
	common.DataExportEnabled = false
	now := time.Date(2026, 11, 20, 15, 0, 0, 0, time.Local)
	todayStart, _, _, _, _ := adminConsoleTimeBounds(now)
	day := int64(24 * 3600)

	// quota_data is not maintained while the dashboard is off; stale rows must be ignored.
	require.NoError(t, DB.Create(&QuotaData{UserID: 1, CreatedAt: todayStart - 2*day, Count: 99, TokenUsed: 999, Quota: 99}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Type: LogTypeConsume, UserId: 2, TokenId: 8, CreatedAt: todayStart - 2*day, PromptTokens: 100, CompletionTokens: 200, Quota: 30},
		{Type: LogTypeConsume, UserId: 4, TokenId: 7, CreatedAt: todayStart + 3600, PromptTokens: 10, CompletionTokens: 20, Quota: 5},
	}).Error)

	row, err := getAdminConsoleUsage(now)
	require.NoError(t, err)

	assert.Equal(t, [3]int64{1, 2, 2}, [3]int64{row.RequestsToday, row.RequestsMonth, row.RequestsTotal})
	assert.Equal(t, int64(330), row.TokensTotal)
	assert.Equal(t, int64(35), row.QuotaTotal)
	assert.Equal(t, int64(2), row.ActiveUsersWeek)
}

func TestCachedAdminConsoleConcurrencyKeepsLastValueWhenRefreshFails(t *testing.T) {
	adminConsoleConcurrencyCache.Lock()
	adminConsoleConcurrencyCache.expiresAt, adminConsoleConcurrencyCache.value = time.Time{}, [3]int64{}
	adminConsoleConcurrencyCache.Unlock()
	now := time.Now()

	first := cachedAdminConsoleConcurrency(now, func() ([3]int64, error) { return [3]int64{3, 7, 9}, nil })
	failed := cachedAdminConsoleConcurrency(now.Add(adminConsoleConcurrencyCacheTTL), func() ([3]int64, error) {
		return [3]int64{}, errors.New("canceling statement due to statement timeout")
	})

	assert.Equal(t, [3]int64{3, 7, 9}, first)
	assert.Equal(t, first, failed)
}
