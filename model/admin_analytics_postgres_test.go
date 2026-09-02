package model

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupPostgresAnalyticsTestDB(t *testing.T, tables ...any) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL analytics tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)

	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		initCol()
		_ = tx.Rollback().Error
		_ = sqlDB.Close()
	})

	// PostgreSQL DDL is transactional, so this isolated schema disappears with
	// the rollback and never touches tables already present in the test database.
	schema := fmt.Sprintf("new_api_analytics_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)

	DB, LOG_DB = tx, tx
	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
	initCol()
	require.NoError(t, tx.AutoMigrate(tables...))
}

func TestGetAdminConsoleStatsAggregatesPostgresData(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Token{}, &Channel{}, &User{}, &Log{}, &TopUp{})

	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.Local)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Unix()

	users := []User{
		{Id: 1, Username: "today-one", AffCode: "today-one", CreatedAt: startOfDay + 60},
		{Id: 2, Username: "today-two", AffCode: "today-two", CreatedAt: startOfDay + 120},
		{Id: 3, Username: "old-user", AffCode: "old-user", CreatedAt: startOfDay - 86400},
		{Id: 4, Username: "week-user", AffCode: "week-user", CreatedAt: startOfDay - 6*86400},
		{Id: 5, Username: "month-user", AffCode: "month-user", CreatedAt: startOfDay - 29*86400},
		{Id: 6, Username: "inactive-user", AffCode: "inactive-user", CreatedAt: startOfDay - 30*86400},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Create(&[]Token{
		{UserId: 1, Key: "key-1", Status: common.TokenStatusEnabled},
		{UserId: 1, Key: "key-2", Status: common.TokenStatusEnabled},
		{UserId: 2, Key: "key-3", Status: common.TokenStatusExpired},
	}).Error)
	require.NoError(t, DB.Create(&[]Channel{
		{Name: "enabled-1", Key: "channel-key-1", Status: common.ChannelStatusEnabled},
		{Name: "enabled-2", Key: "channel-key-2", Status: common.ChannelStatusEnabled},
		{Name: "manual", Key: "channel-key-3", Status: common.ChannelStatusManuallyDisabled},
		{Name: "auto", Key: "channel-key-4", Status: common.ChannelStatusAutoDisabled},
	}).Error)

	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: startOfDay + 60, PromptTokens: 100, CompletionTokens: 20, Quota: 1000, UseTime: 10},
		{UserId: 2, TokenId: 2, Type: LogTypeConsume, CreatedAt: startOfDay + 3600, PromptTokens: 30, CompletionTokens: 10, Quota: 2000, UseTime: 20},
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: now.Unix() - 30, PromptTokens: 3, CompletionTokens: 2, Quota: 10, UseTime: 4},
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: now.Unix() + 30, PromptTokens: 300, CompletionTokens: 200, Quota: 5000, UseTime: 40},
		{UserId: 3, TokenId: 3, Type: LogTypeConsume, CreatedAt: startOfDay - 86400, PromptTokens: 200, CompletionTokens: 50, Quota: 3000, UseTime: 8},
		{UserId: 4, TokenId: 1, Type: LogTypeConsume, CreatedAt: startOfDay - 6*86400 + 60, PromptTokens: 10, CompletionTokens: 5, Quota: 400, UseTime: 6},
		{UserId: 5, TokenId: 2, Type: LogTypeConsume, CreatedAt: startOfDay - 29*86400 + 60, PromptTokens: 20, CompletionTokens: 5, Quota: 500, UseTime: 7},
		{UserId: 6, TokenId: 3, Type: LogTypeConsume, CreatedAt: startOfDay - 30*86400 + 60, PromptTokens: 30, CompletionTokens: 10, Quota: 600, UseTime: 9},
		{UserId: 1, Type: LogTypeError, CreatedAt: startOfDay + 7200, PromptTokens: 999, CompletionTokens: 999, Quota: 9999, UseTime: 99},
	}).Error)

	require.NoError(t, DB.Create(&[]TopUp{
		{UserId: 1, Money: 999, PaidCents: 10000, TradeNo: "today-paid", PaymentMethod: PaymentMethodStripe, CreateTime: startOfDay, CompleteTime: startOfDay + 60, Status: common.TopUpStatusSuccess},
		{UserId: 2, Money: 53.25, TradeNo: "today-legacy", PaymentMethod: "alipay", CreateTime: startOfDay, CompleteTime: startOfDay + 120, Status: common.TopUpStatusSuccess},
		{UserId: 1, Money: 999, TradeNo: "today-pending", PaymentMethod: "alipay", CreateTime: startOfDay, Status: common.TopUpStatusPending},
		{UserId: 3, Money: 25, TradeNo: "month-paid", PaymentMethod: PaymentMethodCreem, CreateTime: startOfDay - 5*86400, CompleteTime: startOfDay - 5*86400, Status: common.TopUpStatusSuccess},
		{UserId: 4, Money: 4.75, TradeNo: "month-legacy-time", PaymentMethod: "wxpay", CreateTime: startOfDay - 10*86400, Status: common.TopUpStatusSuccess},
		{UserId: 5, Money: 10, TradeNo: "previous-month", PaymentMethod: PaymentMethodStripe, CreateTime: startOfMonth - 60, CompleteTime: startOfMonth - 1, Status: common.TopUpStatusSuccess},
		{UserId: 1, Money: 20, TradeNo: "balance-payment", PaymentMethod: PaymentMethodBalance, CreateTime: startOfDay, CompleteTime: startOfDay + 180, Status: common.TopUpStatusSuccess},
		{UserId: 1, Money: 777, TradeNo: "failed-payment", PaymentMethod: "alipay", CreateTime: startOfDay, CompleteTime: startOfDay + 240, Status: common.TopUpStatusFailed},
	}).Error)

	stats, err := getAdminConsoleStatsAt(now)
	require.NoError(t, err)
	assert.Equal(t, AdminConsoleKeyStats{Total: 3, Active: 2, Enabled: 2}, stats.APIKeys)
	assert.Equal(t, AdminConsoleChannelStats{Total: 4, Enabled: 2, AutoDisabled: 1}, stats.Channels)
	assert.Equal(t, AdminConsolePeriodStats{Today: 4, Month: 6, Total: 8}, stats.Requests)
	assert.Equal(t, AdminConsoleUserStats{Today: 2, Total: 6, ActiveToday: 2, ActiveWeek: 4, ActiveMonth: 5}, stats.Users)
	assert.Equal(t, AdminConsolePeriodStats{Today: 665, Month: 930, Total: 995}, stats.Tokens)
	assert.Equal(t, AdminConsolePeriodStats{Today: 8010, Month: 11410, Total: 12510}, stats.Quota)
	assert.InDelta(t, 153.25, stats.Revenue.Today, 0.001)
	assert.InDelta(t, 183, stats.Revenue.Month, 0.001)
	assert.InDelta(t, 193, stats.Revenue.Total, 0.001)
	assert.InDelta(t, 15, stats.Performance.TodayResponseP50Seconds, 0.001)
	assert.InDelta(t, 34, stats.Performance.TodayResponseP90Seconds, 0.001)
	assert.InDelta(t, 39.4, stats.Performance.TodayResponseP99Seconds, 0.001)
	assert.Zero(t, stats.Performance.TodayRPMP50)
	assert.Zero(t, stats.Performance.TodayRPMP90)
	assert.Zero(t, stats.Performance.TodayRPMP99)
	assert.Zero(t, stats.Performance.TodayTPMP50)
	assert.Zero(t, stats.Performance.TodayTPMP90)
	assert.Zero(t, stats.Performance.TodayTPMP99)
	assert.Zero(t, stats.Performance.MonthConcurrencyP50)
	assert.Zero(t, stats.Performance.MonthConcurrencyP90)
	assert.Zero(t, stats.Performance.MonthConcurrencyP95)

	realtime, err := getAdminConsoleRealtimeStatsAt(now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), realtime.RPM)
	assert.Equal(t, int64(5), realtime.TPM)
	assert.InDelta(t, 4, realtime.ResponseSeconds, 0.001)
}

func TestGetAdminConsoleStatsCalculatesThroughputAndConcurrencyPercentiles(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Token{}, &Channel{}, &User{}, &Log{}, &TopUp{})

	now := time.Date(2026, time.August, 1, 0, 2, 59, 0, time.Local)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Type: LogTypeConsume, CreatedAt: startOfDay + 100, CompletionTokens: 100, UseTime: 100},
		{Type: LogTypeConsume, CreatedAt: startOfDay + 100, CompletionTokens: 200, UseTime: 50},
		{Type: LogTypeConsume, CreatedAt: startOfDay + 130, CompletionTokens: 50, UseTime: 10},
	}).Error)

	stats, err := getAdminConsoleStatsAt(now)
	require.NoError(t, err)
	assert.InDelta(t, 50, stats.Performance.TodayResponseP50Seconds, 0.001)
	assert.InDelta(t, 90, stats.Performance.TodayResponseP90Seconds, 0.001)
	assert.InDelta(t, 99, stats.Performance.TodayResponseP99Seconds, 0.001)
	assert.InDelta(t, 1, stats.Performance.TodayRPMP50, 0.001)
	assert.InDelta(t, 1.8, stats.Performance.TodayRPMP90, 0.001)
	assert.InDelta(t, 1.98, stats.Performance.TodayRPMP99, 0.001)
	assert.InDelta(t, 50, stats.Performance.TodayTPMP50, 0.001)
	assert.InDelta(t, 250, stats.Performance.TodayTPMP90, 0.001)
	assert.InDelta(t, 295, stats.Performance.TodayTPMP99, 0.001)
	assert.Equal(t, int64(1), stats.Performance.MonthConcurrencyP50)
	assert.Equal(t, int64(2), stats.Performance.MonthConcurrencyP90)
	assert.Equal(t, int64(2), stats.Performance.MonthConcurrencyP95)
}

func TestGetAdminConsoleStatsAtRangeFiltersRangeMetrics(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Token{}, &Channel{}, &User{}, &Log{}, &TopUp{})

	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.Local)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	rangeStart := startOfDay + 100
	rangeEnd := startOfDay + 200
	require.NoError(t, DB.Create(&[]User{
		{Id: 1, Username: "inside", CreatedAt: rangeStart + 1},
		{Id: 2, Username: "before", CreatedAt: rangeStart - 1},
		{Id: 3, Username: "after", CreatedAt: rangeEnd + 1},
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: rangeStart + 20, PromptTokens: 10, CompletionTokens: 20, Quota: 100, UseTime: 10},
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: rangeEnd - 1, PromptTokens: 20, CompletionTokens: 30, Quota: 200, UseTime: 20},
		{UserId: 2, TokenId: 2, Type: LogTypeConsume, CreatedAt: rangeStart - 1, PromptTokens: 100, CompletionTokens: 100, Quota: 1000, UseTime: 30},
		{UserId: 3, TokenId: 3, Type: LogTypeConsume, CreatedAt: rangeEnd + 2, PromptTokens: 200, CompletionTokens: 200, Quota: 2000, UseTime: 1},
	}).Error)
	require.NoError(t, DB.Create(&[]TopUp{
		{UserId: 1, Money: 5, TradeNo: "inside", PaymentMethod: "stripe", CreateTime: rangeStart + 10, CompleteTime: rangeStart + 10, Status: common.TopUpStatusSuccess},
		{UserId: 2, Money: 7, TradeNo: "before", PaymentMethod: "stripe", CreateTime: rangeStart - 1, CompleteTime: rangeStart - 1, Status: common.TopUpStatusSuccess},
		{UserId: 3, Money: 9, TradeNo: "after", PaymentMethod: "stripe", CreateTime: rangeEnd + 1, CompleteTime: rangeEnd + 1, Status: common.TopUpStatusSuccess},
	}).Error)

	selectedRange, err := normalizeAdminConsoleTimeRange(now, rangeStart, rangeEnd)
	require.NoError(t, err)
	stats, err := getAdminConsoleStatsAtRange(now, selectedRange)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Users.Today)
	assert.Equal(t, int64(2), stats.Requests.Today)
	assert.Equal(t, int64(80), stats.Tokens.Today)
	assert.Equal(t, int64(300), stats.Quota.Today)
	assert.InDelta(t, 5, stats.Revenue.Today, 0.001)
	assert.InDelta(t, 15, stats.Performance.AverageResponseSeconds, 0.001)
	assert.InDelta(t, 15, stats.Performance.TodayResponseP50Seconds, 0.001)
	assert.InDelta(t, 19, stats.Performance.TodayResponseP90Seconds, 0.001)
	assert.InDelta(t, 19.9, stats.Performance.TodayResponseP99Seconds, 0.001)
	assert.InDelta(t, 1, stats.Performance.TodayRPMP50, 0.001)
	assert.InDelta(t, 1, stats.Performance.TodayRPMP90, 0.001)
	assert.InDelta(t, 1, stats.Performance.TodayRPMP99, 0.001)
	assert.InDelta(t, 30, stats.Performance.TodayTPMP50, 0.001)
	assert.InDelta(t, 46, stats.Performance.TodayTPMP90, 0.001)
	assert.InDelta(t, 49.6, stats.Performance.TodayTPMP99, 0.001)
	assert.Equal(t, int64(0), stats.Performance.ConcurrencyP50)
	assert.Equal(t, int64(1), stats.Performance.ConcurrencyP90)
	assert.Equal(t, int64(1), stats.Performance.ConcurrencyP99)
}

func TestGetAdminConsoleRealtimeStatsUsesSixtySecondWindow(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Log{})

	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.Local)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Type: LogTypeConsume, CreatedAt: now.Unix() - 60, CompletionTokens: 100, UseTime: 1},
		{Type: LogTypeConsume, CreatedAt: now.Unix() - 59, CompletionTokens: 20, UseTime: 2},
		{Type: LogTypeConsume, CreatedAt: now.Unix(), CompletionTokens: 30, UseTime: 4},
		{Type: LogTypeConsume, CreatedAt: now.Unix() + 1, CompletionTokens: 200, UseTime: 8},
	}).Error)

	stats, err := getAdminConsoleRealtimeStatsAt(now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.RPM)
	assert.Equal(t, int64(50), stats.TPM)
	assert.InDelta(t, 3, stats.ResponseSeconds, 0.001)
}

func TestGetLogAnalyticsUsesPostgresPercentilesAndUserSearch(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &User{}, &Log{}, &Channel{})
	require.NoError(t, DB.Create(&[]User{
		{Id: 7, Username: "alice", Email: "alice@example.com", Remark: "Finance", AffCode: "alice"},
		{Id: 999, Username: "ghost", Email: "ghost@example.com", AffCode: "ghost"},
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 7, Username: "alice", Type: LogTypeConsume, CreatedAt: 100, UseTime: 2, PromptTokens: 10, CompletionTokens: 5, InputTokensTotal: 20, CacheReadTokens: 5, CacheWriteTokens: 2, Quota: 100, Group: "paid", ModelName: "model-a"},
		{UserId: 8, Username: "bob", Type: LogTypeConsume, CreatedAt: 200, UseTime: 4, PromptTokens: 20, CompletionTokens: 10, InputTokensTotal: 30, CacheReadTokens: 15, Quota: 200, Group: "free", ModelName: "model-b"},
		{UserId: 7, Username: "alice", Type: LogTypeError, CreatedAt: 300, UseTime: 8, Quota: 50, Group: "paid", ModelName: "model-a"},
		{UserId: 999, Username: "ghost", Type: LogTypeConsume, CreatedAt: 400, PromptTokens: 1, CompletionTokens: 1, Quota: 1},
		{UserId: 7, Username: "alice", Type: LogTypeConsume, CreatedAt: time.Date(2026, time.August, 22, 15, 30, 0, 0, time.UTC).Unix(), PromptTokens: 1},
		{UserId: 7, Username: "alice", Type: LogTypeConsume, CreatedAt: time.Date(2026, time.August, 22, 16, 30, 0, 0, time.UTC).Unix(), PromptTokens: 1},
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
	assert.InDelta(t, 14.0/3.0, analytics.Summary.AverageUseTime, 0.001)
	assert.InDelta(t, 7.2, analytics.Summary.P90UseTime, 0.001)
	assert.InDelta(t, 7.92, analytics.Summary.P99UseTime, 0.001)
	require.Len(t, analytics.TokenTrend, 1)
	assert.InDelta(t, 40, analytics.TokenTrend[0].CacheHitRate, 0.001)
	assert.Len(t, analytics.GroupDistribution, 2)
	assert.Len(t, analytics.ModelDistribution, 2)

	filtered, err := GetLogAnalytics(LogAnalyticsFilters{Keyword: "999", StartTimestamp: 1, EndTimestamp: 500, UserLimit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), filtered.Summary.TotalRequests)
	assert.Equal(t, int64(2), filtered.Summary.TotalTokens)

	localDay, err := GetLogAnalytics(LogAnalyticsFilters{
		StartTimestamp: time.Date(2026, time.August, 22, 15, 0, 0, 0, time.UTC).Unix(),
		EndTimestamp:   time.Date(2026, time.August, 22, 17, 0, 0, 0, time.UTC).Unix(),
		Granularity:    "day",
		TimezoneOffset: -8 * 60,
		UserLimit:      10,
	})
	require.NoError(t, err)
	require.Len(t, localDay.TokenTrend, 2)
	assert.Equal(t, time.Date(2026, time.August, 21, 16, 0, 0, 0, time.UTC).Unix(), localDay.TokenTrend[0].Timestamp)
	assert.Equal(t, time.Date(2026, time.August, 22, 16, 0, 0, 0, time.UTC).Unix(), localDay.TokenTrend[1].Timestamp)
}

func TestGetLogCacheTrendAggregatesByDimension(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Log{})

	base := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC).Unix()
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 1, Name: "primary"},
		{Id: 2, Name: "secondary"},
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{
			Type:                LogTypeConsume,
			CreatedAt:           base + 60,
			Group:               "paid",
			ChannelId:           1,
			InputTokensTotal:    100,
			CacheReadTokens:     60,
			CacheWriteTokens:    10,
			CacheStatsAvailable: true,
		},
		{
			Type:                LogTypeConsume,
			CreatedAt:           base + 120,
			Group:               "paid",
			ChannelId:           1,
			InputTokensTotal:    50,
			CacheWriteTokens:    5,
			CacheStatsAvailable: true,
		},
		{
			Type:                LogTypeConsume,
			CreatedAt:           base + 3600 + 60,
			Group:               "paid",
			ChannelId:           2,
			InputTokensTotal:    200,
			CacheReadTokens:     50,
			CacheStatsAvailable: true,
		},
		{
			Type:                LogTypeConsume,
			CreatedAt:           base + 60,
			Group:               "legacy",
			ChannelId:           2,
			InputTokensTotal:    999,
			CacheReadTokens:     999,
			CacheStatsAvailable: false,
		},
		{
			Type:                LogTypeError,
			CreatedAt:           base + 120,
			Group:               "paid",
			ChannelId:           1,
			InputTokensTotal:    500,
			CacheReadTokens:     500,
			CacheStatsAvailable: true,
		},
		{
			Type:                LogTypeConsume,
			CreatedAt:           base + 7200 + 60,
			Group:               "zero",
			InputTokensTotal:    0,
			CacheStatsAvailable: true,
		},
	}).Error)

	filters := LogCacheTrendFilters{
		StartTimestamp: base,
		EndTimestamp:   base + 3*3600,
		Granularity:    "hour",
		TimezoneOffset: 0,
	}

	groupFilters := filters
	groupFilters.Dimension = LogCacheTrendDimensionGroup
	groupPoints, err := GetLogCacheTrend(groupFilters)
	require.NoError(t, err)
	require.Len(t, groupPoints, 3)
	assert.Equal(t, base, groupPoints[0].Timestamp)
	assert.Equal(t, "paid", groupPoints[0].Name)
	assert.Equal(t, int64(150), groupPoints[0].CacheInputTokens)
	assert.Equal(t, int64(60), groupPoints[0].CacheReadTokens)
	assert.Equal(t, int64(15), groupPoints[0].CacheWriteTokens)
	assert.Equal(t, int64(1), groupPoints[0].CacheHitRequests)
	assert.Equal(t, int64(2), groupPoints[0].CacheEligibleRequests)
	assert.InDelta(t, 40, groupPoints[0].CacheHitRate, 0.001)
	assert.Equal(t, base+3600, groupPoints[1].Timestamp)
	assert.Equal(t, "paid", groupPoints[1].Name)
	assert.InDelta(t, 25, groupPoints[1].CacheHitRate, 0.001)
	assert.Equal(t, base+7200, groupPoints[2].Timestamp)
	assert.Equal(t, "zero", groupPoints[2].Name)
	assert.Zero(t, groupPoints[2].CacheHitRate)

	channelFilters := filters
	channelFilters.Dimension = LogCacheTrendDimensionChannel
	channelPoints, err := GetLogCacheTrend(channelFilters)
	require.NoError(t, err)
	require.Len(t, channelPoints, 3)
	assert.Equal(t, "primary", channelPoints[0].Name)
	assert.Equal(t, 1, channelPoints[0].ChannelID)
	assert.Equal(t, int64(150), channelPoints[0].CacheInputTokens)
	assert.InDelta(t, 40, channelPoints[0].CacheHitRate, 0.001)
	assert.Equal(t, "secondary", channelPoints[1].Name)
	assert.Equal(t, 2, channelPoints[1].ChannelID)
	assert.InDelta(t, 25, channelPoints[1].CacheHitRate, 0.001)
	assert.Equal(t, "未记录渠道", channelPoints[2].Name)
	assert.Zero(t, channelPoints[2].ChannelID)
}

func TestGetLogFilterOptionsUsesConfiguredPricingGroups(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Log{})

	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	previousGroupEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	previousGroupOrder := ratio_setting.PricingGroupOrder2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(previousGroupEnabled))
		require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(previousGroupOrder))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"configured":0.5,"disabled":1,"auto":0.8}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"configured":true,"disabled":false,"auto":true}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(`["configured","auto","disabled"]`))

	require.NoError(t, DB.Create(&Channel{Id: 7, Name: "configured-channel"}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Type: LogTypeConsume, Group: "configured", ChannelId: 7},
		{Type: LogTypeConsume, Group: "auto", ChannelId: 7},
		{Type: LogTypeConsume, Group: "legacy", ChannelId: 7},
	}).Error)

	options, err := GetLogFilterOptions()
	require.NoError(t, err)
	assert.Equal(t, []string{"configured", "disabled"}, options.Groups)
	assert.Equal(t, []LogFilterChannel{{ID: 7, Name: "configured-channel"}}, options.Channels)
}

func TestGetAllQuotaDatesPreservesGroupAndChannelTrendDimensions(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &QuotaData{}, &Channel{})
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 1, Name: "east"},
		{Id: 2, Name: "west"},
	}).Error)
	require.NoError(t, DB.Create(&[]QuotaData{
		{Username: "alice", ModelName: "gpt-a", UseGroup: "vip", ChannelID: 1, CreatedAt: 1000, TokenUsed: 40, Quota: 100, Count: 2},
		{Username: "alice", ModelName: "gpt-a", UseGroup: "vip", ChannelID: 1, CreatedAt: 1000, TokenUsed: 20, Quota: 50, Count: 1},
		{Username: "alice", ModelName: "gpt-a", UseGroup: "vip", ChannelID: 2, CreatedAt: 1000, TokenUsed: 10, Quota: 25, Count: 1},
		{Username: "bob", ModelName: "gpt-b", UseGroup: "default", ChannelID: 1, CreatedAt: 2000, TokenUsed: 30, Quota: 70, Count: 3},
	}).Error)

	rows, err := GetAllQuotaDates(900, 2100, "")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	byDimension := make(map[string]*QuotaData, len(rows))
	for _, row := range rows {
		key := fmt.Sprintf("%s/%d/%d", row.UseGroup, row.ChannelID, row.CreatedAt)
		byDimension[key] = row
	}
	require.Contains(t, byDimension, "vip/1/1000")
	assert.Equal(t, "east", byDimension["vip/1/1000"].ChannelName)
	assert.Equal(t, 60, byDimension["vip/1/1000"].TokenUsed)
	assert.Equal(t, 150, byDimension["vip/1/1000"].Quota)
	assert.Equal(t, "west", byDimension["vip/2/1000"].ChannelName)
	assert.Equal(t, "default", byDimension["default/1/2000"].UseGroup)

	userRows, err := GetQuotaDataGroupByUser(900, 2100)
	require.NoError(t, err)
	require.Len(t, userRows, 2)
	userQuotaByName := make(map[string]int, len(userRows))
	for _, row := range userRows {
		userQuotaByName[row.Username] = row.Quota
	}
	assert.Equal(t, 175, userQuotaByName["alice"])
	assert.Equal(t, 70, userQuotaByName["bob"])

	filtered, err := GetAllQuotaDates(900, 2100, "alice")
	require.NoError(t, err)
	require.Len(t, filtered, 2)
	for _, row := range filtered {
		assert.Equal(t, "alice", row.Username)
		assert.Equal(t, "vip", row.UseGroup)
		assert.NotEmpty(t, row.ChannelName)
	}
}
