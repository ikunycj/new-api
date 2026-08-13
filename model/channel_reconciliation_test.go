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

func TestValidateChannelCostEntry(t *testing.T) {
	entry := &ChannelCostEntry{ChannelId: 1, StartAt: 100, EndAt: 200, AmountUSD: 2}
	require.NoError(t, ValidateChannelCostEntry(entry))
	assert.Equal(t, "USD", entry.Currency)
	assert.Equal(t, "manual", entry.Source)

	entry.AmountUSD = -1
	assert.Error(t, ValidateChannelCostEntry(entry))
	entry.AmountUSD = 1
	entry.EndAt = entry.StartAt + int64(MaxChannelCostEntryRange/time.Second) + 1
	assert.Error(t, ValidateChannelCostEntry(entry))
}

func TestAggregateChannelReconciliationLogUsesBillingSnapshot(t *testing.T) {
	summary := ChannelReconciliationSummary{}
	daily := map[string]*ChannelReconciliationBucket{}
	models := map[string]*ChannelReconciliationBucket{}
	inbound := map[string]*ChannelReconciliationBucket{}
	upstream := map[string]*ChannelReconciliationBucket{}
	log := &Log{
		CreatedAt:        1_700_000_000,
		PromptTokens:     100,
		CompletionTokens: 50,
		Quota:            int(2 * common.QuotaPerUnit * 7.3),
		UseTime:          2,
		ModelName:        "model-a",
		Other: common.MapToJsonStr(map[string]any{
			"billing_usd_to_cny_rate":  7.3,
			"group_ratio":              2.0,
			"user_group_ratio":         0.5,
			"cache_tokens":             20,
			"cache_creation_tokens_5m": 3,
			"cache_creation_tokens_1h": 4,
			"request_path":             "/v1/chat/completions",
			"upstream_request_path":    "/chat/completions",
		}),
	}
	aggregateChannelReconciliationLog(log, &summary, daily, models, inbound, upstream)

	assert.Equal(t, int64(1), summary.Requests)
	assert.Equal(t, int64(20), summary.CacheReadTokens)
	assert.Equal(t, int64(7), summary.CacheWriteTokens)
	assert.InDelta(t, 2.0, summary.UserChargeUSD, 0.0001)
	assert.InDelta(t, 4.0, summary.EstimatedCostUSD, 0.0001)
	assert.Equal(t, int64(1), models["model-a"].Requests)
	assert.Equal(t, int64(1), inbound["/v1/chat/completions"].Requests)
	assert.Equal(t, int64(1), upstream["/chat/completions"].Requests)
}

func TestProratedCostUsesRangeIntersection(t *testing.T) {
	entry := ChannelCostEntry{StartAt: 100, EndAt: 200, AmountUSD: 10}
	assert.InDelta(t, 5, proratedCost(entry, 150, 250), 0.0001)
	assert.Zero(t, proratedCost(entry, 200, 250))
}

func TestAllocateCostToDailyIncludesDaysWithoutRequests(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	daily := map[string]*ChannelReconciliationBucket{}
	allocateCostToDaily(2, start, start+48*60*60, daily)
	assert.Contains(t, daily, "2026-01-02")
	assert.InDelta(t, 1, daily["2026-01-01"].ActualCostUSD, 0.0001)
	assert.InDelta(t, 1, daily["2026-01-02"].ActualCostUSD, 0.0001)
}

func TestGetChannelReconciliationIsolatesChannelAndRejectsOverlappingCost(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Channel{}, &ChannelCostEntry{}, &Log{}))

	previousDB, previousLogDB := DB, LOG_DB
	DB, LOG_DB = testDB, testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	})

	channels := []Channel{
		{Name: "reconciliation-a", Key: "key-a", CreatedTime: 1},
		{Name: "reconciliation-b", Key: "key-b", CreatedTime: 1},
	}
	require.NoError(t, DB.Create(&channels).Error)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	require.NoError(t, LOG_DB.Create(&[]Log{
		{ChannelId: channels[0].Id, Type: LogTypeConsume, CreatedAt: start + 60, RequestId: "request-a", Quota: int(common.QuotaPerUnit), ModelName: "model-a", Other: common.MapToJsonStr(map[string]any{"group_ratio": 1})},
		{ChannelId: channels[1].Id, Type: LogTypeConsume, CreatedAt: start + 60, RequestId: "request-b", Quota: int(9 * common.QuotaPerUnit), ModelName: "model-b", Other: common.MapToJsonStr(map[string]any{"group_ratio": 1})},
	}).Error)

	entry := &ChannelCostEntry{ChannelId: channels[0].Id, StartAt: start, EndAt: start + 86400, AmountUSD: 3, CreatedBy: 1}
	require.NoError(t, CreateChannelCostEntry(entry))
	overlap := &ChannelCostEntry{ChannelId: channels[0].Id, StartAt: start + 3600, EndAt: start + 7200, AmountUSD: 1, CreatedBy: 1}
	assert.ErrorIs(t, CreateChannelCostEntry(overlap), ErrChannelCostEntryOverlap)

	result, err := GetChannelReconciliation(channels[0].Id, start, start+86400)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Summary.Requests)
	assert.InDelta(t, 1, result.Summary.UserChargeUSD, 0.0001)
	assert.InDelta(t, 3, result.Summary.ActualCostUSD, 0.0001)
	require.Len(t, result.Models, 1)
	assert.Equal(t, "model-a", result.Models[0].Name)
}
