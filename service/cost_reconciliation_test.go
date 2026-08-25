package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProviderBaseCostUsesActualUsageAndIgnoresUserGroupRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	usage := &dto.Usage{PromptTokens: 1000, CompletionTokens: 500}

	calculate := func(groupRatio float64) textQuotaSummary {
		return calculateTextQuotaSummary(ctx, &relaycommon.RelayInfo{
			OriginModelName: "cost-model",
			StartTime:       time.Now(),
			PriceData: types.PriceData{
				ModelRatio:          2,
				CompletionRatio:     2,
				BillingUSDToCNYRate: 7.3,
				GroupRatioInfo:      types.GroupRatioInfo{GroupRatio: groupRatio},
			},
		}, usage)
	}

	toC := calculate(0.5)
	toB := calculate(2)

	require.True(t, toC.ProviderCostAvailable)
	require.True(t, toB.ProviderCostAvailable)
	assert.InDelta(t, 0.008, toC.ProviderBaseCostUSD, 0.000000001)
	assert.InDelta(t, toC.ProviderBaseCostUSD, toB.ProviderBaseCostUSD, 0.000000001)
	assert.NotEqual(t, toC.Quota, toB.Quota)
}

func TestCostReconciliationAppliesEachAttemptChannelFactor(t *testing.T) {
	setupCostReconciliationRouting(t)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("use_channel", []string{"11", "12"})
	ctx.Set("use_channel_groups", []string{"toB", "toB"})

	snapshot := BuildCostReconciliationSnapshot(
		ctx,
		&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 12}, UsingGroup: "toB"},
		1000,
		500,
		7500,
		0.01,
		true,
		"actual_usage_model_pricing_x_channel_cost_factor",
		1,
	)

	assert.Equal(t, "estimated", snapshot["status"])
	assert.Equal(t, int64(15_000), snapshot["user_charge_usd_micros"])
	assert.Equal(t, int64(5_000), snapshot["retry_cost_usd_micros"])
	assert.Equal(t, int64(12_000), snapshot["successful_cost_usd_micros"])
	assert.Equal(t, int64(17_000), snapshot["estimated_cost_usd_micros"])
}

func TestCostReconciliationUsesFailedAttemptActualUsage(t *testing.T) {
	setupCostReconciliationRouting(t)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("use_channel", []string{"11", "12"})
	ctx.Set("use_channel_groups", []string{"toB", "toB"})
	info := &relaycommon.RelayInfo{
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 12},
		UsingGroup:      "toB",
		OriginModelName: "cost-model",
		StartTime:       time.Now(),
		PriceData: types.PriceData{
			ModelRatio:      2,
			CompletionRatio: 2,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
	}

	RecordFailedAttemptUsage(ctx, info, "toB", 11, &dto.Usage{PromptTokens: 250})
	snapshot := BuildCostReconciliationSnapshot(ctx, info, 1000, 500, 7500, 0.01, true, "actual_usage_model_pricing_x_channel_cost_factor", 1)

	assert.Equal(t, int64(0), snapshot["retry_cost_usd_micros"])
	assert.Equal(t, int64(500), snapshot["failed_partial_usage_cost_usd_micros"])
	assert.Equal(t, int64(12_500), snapshot["estimated_cost_usd_micros"])
}

func setupCostReconciliationRouting(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = testDB
	model.InitChannelRoutingCache()
	t.Cleanup(func() {
		model.DB = previousDB
		model.InitChannelRoutingCache()
	})
	require.NoError(t, testDB.AutoMigrate(&model.BillingGroupRoute{}, &model.BillingGroupChannel{}))
	require.NoError(t, testDB.Create(&model.BillingGroupRoute{Id: 1, BillingGroup: "toB", Enabled: true}).Error)
	require.NoError(t, testDB.Create(&[]model.BillingGroupChannel{
		{BillingGroupRouteId: 1, ChannelId: 11, Priority: 2, MaxAttempts: 1, Enabled: true, CostFactor: 0.5},
		{BillingGroupRouteId: 1, ChannelId: 12, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1.2},
	}).Error)
	model.InitChannelRoutingCache()
}
