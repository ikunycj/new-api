package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetLoadTestChannelStatsAggregatesUserLogsByChannel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}, &BillingGroupRoute{}, &BillingGroupChannel{}, &Log{}))

	previousDB, previousLogDB := DB, LOG_DB
	DB, LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	})

	channels := []Channel{
		{Name: "pro"},
		{Name: "official"},
	}
	require.NoError(t, db.Create(&channels).Error)
	route := BillingGroupRoute{BillingGroup: "claude", Name: "Claude", Mode: RoutingModeBalanced, Enabled: true}
	require.NoError(t, db.Create(&route).Error)
	require.NoError(t, db.Create(&[]BillingGroupChannel{
		{BillingGroupRouteId: route.Id, ChannelId: channels[0].Id, Priority: 100, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 0.6},
		{BillingGroupRouteId: route.Id, ChannelId: channels[1].Id, Priority: 90, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 1.1},
	}).Error)
	InitChannelRoutingCache()
	require.NoError(t, db.Create(&[]Log{
		{UserId: 7, Type: LogTypeConsume, RequestId: "run-a", ChannelId: channels[0].Id, Group: "claude", PromptTokens: 100, CompletionTokens: 20},
		{UserId: 7, Type: LogTypeConsume, RequestId: "run-b", ChannelId: channels[0].Id, Group: "claude", PromptTokens: 50, InputTokensTotal: 75, CompletionTokens: 10, CacheReadTokens: 25},
		{UserId: 7, Type: LogTypeConsume, RequestId: "run-c", ChannelId: channels[1].Id, Group: "claude", PromptTokens: 40, CompletionTokens: 8},
		{UserId: 8, Type: LogTypeConsume, RequestId: "run-d", ChannelId: channels[1].Id, PromptTokens: 999, CompletionTokens: 999},
	}).Error)

	stats, err := GetLoadTestChannelStats(7, []string{"run-a", "run-b", "run-c", "run-c"})
	require.NoError(t, err)
	require.Len(t, stats, 2)
	require.Equal(t, channels[0].Id, stats[0].ChannelID)
	require.Equal(t, int64(2), stats[0].Requests)
	require.Equal(t, int64(150), stats[0].InputTokens)
	require.Equal(t, int64(75), stats[0].InputTokensTotal)
	require.Equal(t, int64(30), stats[0].OutputTokens)
	require.Equal(t, int64(25), stats[0].CacheReadTokens)
	require.InDelta(t, 0.6, stats[0].CostFactor, 0.0001)
	require.Equal(t, channels[1].Id, stats[1].ChannelID)
	require.Equal(t, int64(1), stats[1].Requests)
	require.InDelta(t, 1.1, stats[1].CostFactor, 0.0001)
}
