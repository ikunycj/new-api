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

func TestGetAdminConsoleStatsAggregatesCurrentAndHistoricalUsage(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Token{}, &Channel{}, &User{}, &Log{}, &TopUp{}))

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

	tokens := []Token{
		{UserId: 1, Key: "key-1", Status: common.TokenStatusEnabled},
		{UserId: 1, Key: "key-2", Status: common.TokenStatusEnabled},
		{UserId: 2, Key: "key-3", Status: common.TokenStatusExpired},
	}
	require.NoError(t, DB.Create(&tokens).Error)

	channels := []Channel{
		{Name: "enabled-1", Key: "channel-key-1", Status: common.ChannelStatusEnabled},
		{Name: "enabled-2", Key: "channel-key-2", Status: common.ChannelStatusEnabled},
		{Name: "manual", Key: "channel-key-3", Status: common.ChannelStatusManuallyDisabled},
		{Name: "auto", Key: "channel-key-4", Status: common.ChannelStatusAutoDisabled},
	}
	require.NoError(t, DB.Create(&channels).Error)

	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: startOfDay + 60, PromptTokens: 100, CompletionTokens: 20, Quota: 1000, UseTime: 10},
		{UserId: 2, TokenId: 2, Type: LogTypeConsume, CreatedAt: startOfDay + 3600, PromptTokens: 30, CompletionTokens: 10, Quota: 2000, UseTime: 20},
		{UserId: 1, TokenId: 1, Type: LogTypeConsume, CreatedAt: now.Unix() - 30, PromptTokens: 3, CompletionTokens: 2, Quota: 10, UseTime: 4},
		{UserId: 3, TokenId: 3, Type: LogTypeConsume, CreatedAt: startOfDay - 86400, PromptTokens: 200, CompletionTokens: 50, Quota: 3000, UseTime: 8},
		{UserId: 4, TokenId: 1, Type: LogTypeConsume, CreatedAt: startOfDay - 6*86400 + 60, PromptTokens: 10, CompletionTokens: 5, Quota: 400, UseTime: 6},
		{UserId: 5, TokenId: 2, Type: LogTypeConsume, CreatedAt: startOfDay - 29*86400 + 60, PromptTokens: 20, CompletionTokens: 5, Quota: 500, UseTime: 7},
		{UserId: 6, TokenId: 3, Type: LogTypeConsume, CreatedAt: startOfDay - 30*86400 + 60, PromptTokens: 30, CompletionTokens: 10, Quota: 600, UseTime: 9},
		{UserId: 1, Type: LogTypeError, CreatedAt: startOfDay + 7200, PromptTokens: 999, CompletionTokens: 999, Quota: 9999, UseTime: 99},
	}).Error)

	require.NoError(t, DB.Create(&[]TopUp{
		{UserId: 1, Money: 100, TradeNo: "today-paid", PaymentMethod: PaymentMethodStripe, CreateTime: startOfDay, CompleteTime: startOfDay + 60, Status: common.TopUpStatusSuccess},
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

	assert.Equal(t, int64(3), stats.APIKeys.Total)
	assert.Equal(t, int64(2), stats.APIKeys.Active)
	assert.Equal(t, int64(2), stats.APIKeys.Enabled)
	assert.Equal(t, int64(4), stats.Accounts.Total)
	assert.Equal(t, int64(2), stats.Accounts.Enabled)
	assert.Equal(t, int64(1), stats.Accounts.AutoDisabled)
	assert.Equal(t, int64(3), stats.Requests.Today)
	assert.Equal(t, int64(7), stats.Requests.Total)
	assert.Equal(t, int64(2), stats.Users.Today)
	assert.Equal(t, int64(6), stats.Users.Total)
	assert.Equal(t, int64(2), stats.Users.ActiveToday)
	assert.Equal(t, int64(4), stats.Users.ActiveWeek)
	assert.Equal(t, int64(5), stats.Users.ActiveMonth)
	assert.Equal(t, int64(165), stats.Tokens.Today)
	assert.Equal(t, int64(495), stats.Tokens.Total)
	assert.Equal(t, int64(3010), stats.Quota.Today)
	assert.Equal(t, int64(7510), stats.Quota.Total)
	assert.InDelta(t, 153.25, stats.Revenue.Today, 0.001)
	assert.InDelta(t, 183, stats.Revenue.Month, 0.001)
	assert.InDelta(t, 193, stats.Revenue.Total, 0.001)
	assert.Equal(t, int64(1), stats.Performance.RPM)
	assert.Equal(t, int64(5), stats.Performance.TPM)
	assert.InDelta(t, 34.0/3.0, stats.Performance.AverageResponseSeconds, 0.001)
}
