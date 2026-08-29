package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChannelRoute(t *testing.T) []model.Channel {
	t.Helper()
	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}, &model.BillingGroupRoute{}, &model.BillingGroupChannel{}))
	for _, table := range []string{"billing_group_channels", "billing_group_routes", "abilities", "channels"} {
		require.NoError(t, model.DB.Exec("DELETE FROM "+table).Error)
	}
	priority := int64(10)
	firstWeight := uint(1)
	secondWeight := uint(10000)
	channels := []model.Channel{
		{Id: 92001, Name: "Claude Pro", Key: "pro", Status: common.ChannelStatusEnabled, Models: "claude-test", Group: "claude", Priority: &priority, Weight: &firstWeight},
		{Id: 92002, Name: "Claude Official", Key: "official", Status: common.ChannelStatusEnabled, Models: "claude-test", Group: "claude", Priority: &priority, Weight: &secondWeight},
	}
	require.NoError(t, model.DB.Create(&channels).Error)
	for _, channel := range channels {
		require.NoError(t, model.DB.Create(&model.Ability{Group: "claude", Model: "claude-test", ChannelId: channel.Id, Enabled: true, Priority: &priority, Weight: 100}).Error)
	}
	route := model.BillingGroupRoute{Id: 81, BillingGroup: "claude", Name: "Claude", Mode: model.RoutingModeBalanced, Enabled: true, MaxTotalAttempts: 3, TotalTimeoutMs: 30000, CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60, CircuitHalfOpenRequests: 1}
	require.NoError(t, model.DB.Create(&route).Error)
	require.NoError(t, model.DB.Create(&[]model.BillingGroupChannel{
		{BillingGroupRouteId: route.Id, ChannelId: channels[0].Id, Priority: 100, Weight: 1, MaxAttempts: 2, Enabled: true, CostFactor: 0.6},
		{BillingGroupRouteId: route.Id, ChannelId: channels[1].Id, Priority: 100, Weight: 10000, MaxAttempts: 1, Enabled: true, CostFactor: 1.1},
	}).Error)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	model.InitChannelCache()
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		for _, table := range []string{"billing_group_channels", "billing_group_routes", "abilities", "channels"} {
			_ = model.DB.Exec("DELETE FROM " + table).Error
		}
	})
	return channels
}

func TestConfiguredRouteRetriesChannelThenSwitchesInOrder(t *testing.T) {
	channels := setupChannelRoute(t)
	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

	first, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "claude", group)
	assert.Equal(t, channels[0].Id, first.Id)

	param.MarkChannelAttempted(first.Id)
	param.HandleChannelFailure(first.Id, "retry_channel")
	require.True(t, param.HasNextRetry())
	require.True(t, param.AdvanceRetry())
	second, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	assert.Equal(t, channels[0].Id, second.Id)

	param.MarkChannelAttempted(second.Id)
	param.HandleChannelFailure(second.Id, "retry_channel")
	require.True(t, param.AdvanceRetry())
	third, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	assert.Equal(t, channels[1].Id, third.Id)

	param.MarkChannelAttempted(third.Id)
	assert.False(t, param.HasNextRetry())
}

func TestMockMarkerCannotAlterRealRouting(t *testing.T) {
	channels := setupChannelRoute(t)
	mockSetting := `{"mock_load_test":true}`
	realSetting := `{}`
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("setting", mockSetting).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("setting", realSetting).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	ctx.Request.Header.Set(constant.MockLoadTestHeader, "true")
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channels[1].Id, selected.Id)
}

func TestRealRouteSkipsMockLoadTestChannels(t *testing.T) {
	channels := setupChannelRoute(t)
	mockSetting := `{"mock_load_test":true}`
	realSetting := `{}`
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("setting", mockSetting).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("setting", realSetting).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channels[1].Id, selected.Id)
}

func TestConfiguredRouteIgnoresWeightsAtEqualPriority(t *testing.T) {
	channels := setupChannelRoute(t)
	for _, memoryCacheEnabled := range []bool{true, false} {
		common.MemoryCacheEnabled = memoryCacheEnabled
		ctx, _ := gin.CreateTestContext(nil)
		param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

		for range 10 {
			selected, _, err := CacheGetRandomSatisfiedChannel(param)
			require.NoError(t, err)
			require.NotNil(t, selected)
			assert.Equal(t, channels[0].Id, selected.Id)
		}
	}
}

func TestConfiguredRouteSwitchActionSkipsRemainingChannelAttempts(t *testing.T) {
	channels := setupChannelRoute(t)
	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

	first, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, channels[0].Id, first.Id)

	param.MarkChannelAttempted(first.Id)
	param.HandleChannelFailure(first.Id, "switch_channel")
	require.True(t, param.AdvanceRetry())
	second, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, channels[1].Id, second.Id)
}

func TestCrossGroupRetryRequiresTokenPermission(t *testing.T) {
	setupChannelRoute(t)
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupCandidates, []string{"claude", "missing"})
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "claude-test"}
	channel, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, channel)
	param.ExcludeChannel(channel.Id)
	param.ExcludeChannel(92002)
	param.MarkAttempted()
	assert.False(t, param.AdvanceRetry())
}

func TestConfiguredRouteWithoutMemoryCacheHonorsRouteOrder(t *testing.T) {
	channels := setupChannelRoute(t)
	common.MemoryCacheEnabled = false
	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

	first, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, channels[0].Id, first.Id)

	param.ExcludeChannel(first.Id)
	second, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, channels[1].Id, second.Id)
}

func TestProfitGuardWarnsWithoutBlockingTraffic(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Model(&model.BillingGroupRoute{}).
		Where("billing_group = ?", "claude").
		Updates(map[string]any{
			"profit_guard_mode":     model.ProfitGuardModeWarn,
			"minimum_profit_margin": 0,
		}).Error)
	model.InitChannelRoutingCache()
	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}
	priceData := types.PriceData{
		EstimatedProviderBaseCostUSD: 0.01,
		GroupRatioInfo:               types.GroupRatioInfo{GroupRatio: 0.8},
	}

	first := param.EvaluateProfitGuard(channels[0].Id, priceData)
	second := param.EvaluateProfitGuard(channels[0].Id, priceData)

	assert.Equal(t, "allow", first.Decision)
	assert.InDelta(t, 25, first.ProjectedProfitMargin, 0.0001)
	assert.Equal(t, "warn", second.Decision)
	assert.InDelta(t, -50, second.ProjectedProfitMargin, 0.0001)
}

func TestProfitGuardEnforceBlocksUnprofitableAttempt(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Model(&model.BillingGroupRoute{}).
		Where("billing_group = ?", "claude").
		Updates(map[string]any{
			"profit_guard_mode":     model.ProfitGuardModeEnforce,
			"minimum_profit_margin": 10,
		}).Error)
	model.InitChannelRoutingCache()
	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}
	priceData := types.PriceData{
		EstimatedProviderBaseCostUSD: 0.01,
		GroupRatioInfo:               types.GroupRatioInfo{GroupRatio: 0.8},
	}

	first := param.EvaluateProfitGuard(channels[0].Id, priceData)
	second := param.EvaluateProfitGuard(channels[0].Id, priceData)

	assert.Equal(t, "allow", first.Decision)
	assert.Equal(t, "block", second.Decision)
	assert.InDelta(t, -50, second.ProjectedProfitMargin, 0.0001)
}
