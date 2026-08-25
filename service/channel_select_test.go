package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
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
	weight := uint(100)
	channels := []model.Channel{
		{Id: 92001, Name: "Claude Pro", Key: "pro", Status: common.ChannelStatusEnabled, Models: "claude-test", Group: "claude", Weight: &weight},
		{Id: 92002, Name: "Claude Official", Key: "official", Status: common.ChannelStatusEnabled, Models: "claude-test", Group: "claude", Weight: &weight},
	}
	require.NoError(t, model.DB.Create(&channels).Error)
	for _, channel := range channels {
		require.NoError(t, model.DB.Create(&model.Ability{Group: "claude", Model: "claude-test", ChannelId: channel.Id, Enabled: true, Weight: weight}).Error)
	}
	route := model.BillingGroupRoute{Id: 81, BillingGroup: "claude", Name: "Claude", Enabled: true, MaxTotalAttempts: 3, TotalTimeoutMs: 30000, CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60, CircuitHalfOpenRequests: 1}
	require.NoError(t, model.DB.Create(&route).Error)
	require.NoError(t, model.DB.Create(&[]model.BillingGroupChannel{
		{BillingGroupRouteId: route.Id, ChannelId: channels[0].Id, Priority: 100, Weight: 100, MaxAttempts: 2, Enabled: true, CostFactor: 0.6},
		{BillingGroupRouteId: route.Id, ChannelId: channels[1].Id, Priority: 90, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 1.1},
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

func TestCrossGroupRetryExhaustsEarlierGroupsFirst(t *testing.T) {
	channels := setupChannelRoute(t)
	weight := uint(100)
	cheaperChannel := model.Channel{
		Id:                  92003,
		Name:                "Economy",
		Key:                 "economy",
		Status:              common.ChannelStatusEnabled,
		Models:              "claude-test",
		Group:               "economy",
		Weight:              &weight,
		PriceMultiplier:     0.1,
		PriceMultiplierMode: model.ChannelPriceMultiplierModeUSD,
	}
	require.NoError(t, model.DB.Create(&cheaperChannel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "economy",
		Model:     "claude-test",
		ChannelId: cheaperChannel.Id,
		Enabled:   true,
		Weight:    weight,
	}).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupCandidates, []string{"claude", "economy"})
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, true)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "claude-test"}

	first, firstGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, channels[0].Id, first.Id)
	assert.Equal(t, "claude", firstGroup)

	param.MarkChannelAttempted(first.Id)
	param.HandleChannelFailure(first.Id, "switch_channel")
	require.True(t, param.AdvanceRetry())

	second, secondGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, channels[1].Id, second.Id)
	assert.Equal(t, "claude", secondGroup)

	param.MarkChannelAttempted(second.Id)
	param.HandleChannelFailure(second.Id, "switch_channel")
	require.True(t, param.AdvanceRetry())

	third, thirdGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, third)
	assert.Equal(t, cheaperChannel.Id, third.Id)
	assert.Equal(t, "economy", thirdGroup)
}

func TestCrossGroupRetryUsesIndependentRouteAttemptBudgets(t *testing.T) {
	channels := setupChannelRoute(t)
	weight := uint(100)
	economyChannel := model.Channel{
		Id:     92003,
		Name:   "Economy",
		Key:    "economy",
		Status: common.ChannelStatusEnabled,
		Models: "claude-test",
		Group:  "economy",
		Weight: &weight,
	}
	require.NoError(t, model.DB.Create(&economyChannel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "economy",
		Model:     "claude-test",
		ChannelId: economyChannel.Id,
		Enabled:   true,
		Weight:    weight,
	}).Error)
	require.NoError(t, model.DB.Model(&model.BillingGroupRoute{}).
		Where("id = ?", 81).
		Update("max_total_attempts", 2).Error)
	economyRoute := model.BillingGroupRoute{
		Id:                      82,
		BillingGroup:            "economy",
		Name:                    "Economy",
		Enabled:                 true,
		MaxTotalAttempts:        1,
		TotalTimeoutMs:          30000,
		CircuitFailureThreshold: 5,
		CircuitWindowSeconds:    60,
		CircuitCooldownSeconds:  60,
		CircuitHalfOpenRequests: 1,
	}
	require.NoError(t, model.DB.Create(&economyRoute).Error)
	require.NoError(t, model.DB.Create(&model.BillingGroupChannel{
		BillingGroupRouteId: economyRoute.Id,
		ChannelId:           economyChannel.Id,
		Priority:            100,
		Weight:              100,
		MaxAttempts:         1,
		Enabled:             true,
		CostFactor:          1,
	}).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupCandidates, []string{"claude", "economy"})
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, true)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "claude-test"}

	for _, expectedChannelID := range []int{channels[0].Id, channels[1].Id} {
		channel, group, err := CacheGetRandomSatisfiedChannel(param)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, expectedChannelID, channel.Id)
		assert.Equal(t, "claude", group)
		param.MarkChannelAttempted(channel.Id)
		param.HandleChannelFailure(channel.Id, "switch_channel")
		require.True(t, param.HasNextRetry())
		require.True(t, param.AdvanceRetry())
	}

	channel, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, economyChannel.Id, channel.Id)
	assert.Equal(t, "economy", group)
	assert.Equal(t, 2, param.groupAttemptCounts["claude"])
	assert.Zero(t, param.groupAttemptCounts["economy"])
}

func TestConfiguredRouteWithoutMemoryCacheHonorsRoutePriority(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("price_multiplier", 100).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("price_multiplier", 0.1).Error)
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

func TestConfiguredRoutePriorityWinsOverDynamicChannelSignals(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("price_multiplier", 100).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("price_multiplier", 0.1).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channels[0].Id, selected.Id)
}

func TestConfiguredRouteUsesWeightWithinEqualPriority(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Model(&model.BillingGroupChannel{}).
		Where("billing_group_route_id = ? AND channel_id = ?", 81, channels[0].Id).
		Update("weight", 0).Error)
	require.NoError(t, model.DB.Model(&model.BillingGroupChannel{}).
		Where("billing_group_route_id = ? AND channel_id = ?", 81, channels[1].Id).
		Updates(map[string]any{"priority": 100, "weight": 50}).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{
		Ctx:        ctx,
		TokenGroup: "claude",
		ModelName:  "claude-test",
		randomIntn: func(total int) int {
			assert.Equal(t, 150, total)
			return 100
		},
	}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channels[1].Id, selected.Id)
}

func TestUnconfiguredRouteUsesChannelWeightWithinEqualRank(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Exec("DELETE FROM billing_group_channels").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM billing_group_routes").Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("weight", 0).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("weight", 90).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{
		Ctx:        ctx,
		TokenGroup: "claude",
		ModelName:  "claude-test",
		randomIntn: func(total int) int {
			assert.Equal(t, 110, total)
			return 10
		},
	}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channels[1].Id, selected.Id)
}

func TestMarkChannelAttemptedInitializesDefaultBudget(t *testing.T) {
	param := &RetryParam{}

	assert.NotPanics(t, func() { param.MarkChannelAttempted(92001) })
	assert.Equal(t, 1, param.attemptCounts[92001])
	assert.Equal(t, model.DefaultChannelUpstreamMaxRetries+1, param.channelLimits[92001])
}

func TestRetryParamTracksIndependentGroupRetryBudgets(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupRetryTimes, map[string]int{
		"openai": 0,
		"claude": 2,
	})
	param := &RetryParam{
		Ctx:           ctx,
		TokenGroup:    "auto",
		channelGroups: map[int]string{1: "openai", 2: "claude"},
		channelLimits: map[int]int{1: 10, 2: 10},
	}

	assert.True(t, param.groupHasBudget("openai"))
	assert.True(t, param.groupHasBudget("claude"))
	param.MarkChannelAttempted(1)
	assert.False(t, param.groupHasBudget("openai"))
	assert.True(t, param.groupHasBudget("claude"))

	param.MarkChannelAttempted(2)
	param.MarkChannelAttempted(2)
	assert.True(t, param.groupHasBudget("claude"))
	param.MarkChannelAttempted(2)
	assert.False(t, param.groupHasBudget("claude"))
}

func TestPricingGroupRetryPolicyCapsTokenRetryBudget(t *testing.T) {
	previousPolicies := ratio_setting.PricingGroupRetryPolicy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
	})
	require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(
		`{"openai":{"mode":"fixed","retry_times":2}}`,
	))

	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupRetryTimes, map[string]int{"openai": 5})
	param := &RetryParam{Ctx: ctx, TokenGroup: "openai"}
	limit, configured := param.groupRetryLimit("openai")
	require.True(t, configured)
	assert.Equal(t, 3, limit)

	ctx, _ = gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupRetryTimes, map[string]int{"openai": 1})
	param = &RetryParam{Ctx: ctx, TokenGroup: "openai"}
	limit, configured = param.groupRetryLimit("openai")
	require.True(t, configured)
	assert.Equal(t, 2, limit)
}

func TestActiveChannelRetryPolicyUsesRequestScopedChannelCount(t *testing.T) {
	channels := setupChannelRoute(t)
	previousPolicies := ratio_setting.PricingGroupRetryPolicy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
	})
	require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(
		`{"claude":{"mode":"active_channels","retry_times":0}}`,
	))

	param := &RetryParam{TokenGroup: "claude", ModelName: "claude-test", RequestPath: "/v1/messages"}
	limit, configured := param.groupRetryLimit("claude")
	require.True(t, configured)
	assert.Equal(t, 3, limit)

	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[1].Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
	model.InitChannelCache()
	cachedLimit, configured := param.groupRetryLimit("claude")
	require.True(t, configured)
	assert.Equal(t, 3, cachedLimit)

	newRequest := &RetryParam{TokenGroup: "claude", ModelName: "claude-test", RequestPath: "/v1/messages"}
	newLimit, configured := newRequest.groupRetryLimit("claude")
	require.True(t, configured)
	assert.Equal(t, 2, newLimit)
}

func TestActiveChannelRetryPolicyCountsOnlyConfiguredRouteChannels(t *testing.T) {
	channels := setupChannelRoute(t)
	previousPolicies := ratio_setting.PricingGroupRetryPolicy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
	})
	require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(
		`{"claude":{"mode":"active_channels","retry_times":0}}`,
	))
	require.NoError(t, model.DB.Model(&model.BillingGroupChannel{}).
		Where("billing_group_route_id = ? AND channel_id = ?", 81, channels[1].Id).
		Update("enabled", false).Error)
	model.InitChannelCache()

	param := &RetryParam{TokenGroup: "claude", ModelName: "claude-test", RequestPath: "/v1/messages"}
	limit, configured := param.groupRetryLimit("claude")

	require.True(t, configured)
	assert.Equal(t, 2, limit)
}

func TestUnconfiguredRouteHonorsChannelRetryBudget(t *testing.T) {
	channels := setupChannelRoute(t)
	// Remove the optional billing-group route so the selector must rely on the
	// channel-level retry setting. The default runtime policy must not truncate
	// an explicitly larger channel budget.
	require.NoError(t, model.DB.Exec("DELETE FROM billing_group_channels").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM billing_group_routes").Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("upstream_max_retries", 10).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}
	param.ExcludeChannel(channels[1].Id)
	for attempt := 1; attempt <= 11; attempt++ {
		channel, _, err := CacheGetRandomSatisfiedChannel(param)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, channels[0].Id, channel.Id)
		param.MarkChannelAttempted(channel.Id)
		if attempt < 11 {
			require.True(t, param.HasNextRetry(), "attempt %d should still have the channel retry budget", attempt)
			require.True(t, param.AdvanceRetry())
		} else {
			assert.False(t, param.HasNextRetry())
		}
	}
}

func TestConfiguredRouteTotalBudgetCapsChannelRetryBudget(t *testing.T) {
	channels := setupChannelRoute(t)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channels[0].Id).Update("upstream_max_retries", 10).Error)
	require.NoError(t, model.DB.Model(&model.BillingGroupChannel{}).
		Where("billing_group_route_id = ? AND channel_id = ?", 81, channels[0].Id).
		Update("max_attempts", 20).Error)
	model.InitChannelCache()

	ctx, _ := gin.CreateTestContext(nil)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "claude-test"}
	for attempt := 1; attempt <= 3; attempt++ {
		channel, _, err := CacheGetRandomSatisfiedChannel(param)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, channels[0].Id, channel.Id)
		param.MarkChannelAttempted(channel.Id)
		if attempt < 3 {
			require.True(t, param.HasNextRetry())
			require.True(t, param.AdvanceRetry())
		} else {
			assert.False(t, param.HasNextRetry())
		}
	}
}

func TestDynamicCandidateRanking(t *testing.T) {
	force := true
	notForced := false
	base := &model.Channel{Id: 1, PriceMultiplier: 1, PriceMultiplierMode: model.ChannelPriceMultiplierModeUSD, PreviousDayProbeSuccessRate: 90, ForcePriority: &notForced}

	t.Run("cross-group force wins before price", func(t *testing.T) {
		forced := *base
		forced.Id = 2
		forced.PriceMultiplier = 100
		forced.ForcePriority = &force
		forced.ForcePriorityScope = model.ChannelForcePriorityScopeCrossGroup
		assert.True(t, dynamicCandidateLess(
			dynamicChannelCandidate{channel: &forced, groupIndex: 1},
			dynamicChannelCandidate{channel: base, groupIndex: 0},
		))
	})

	t.Run("CNY and USD prices are comparable", func(t *testing.T) {
		previousRate := operation_setting.BillingUSDToCNYRate
		operation_setting.BillingUSDToCNYRate = 7.2
		t.Cleanup(func() { operation_setting.BillingUSDToCNYRate = previousRate })

		cny := *base
		cny.Id = 2
		cny.PriceMultiplier = 6
		cny.PriceMultiplierMode = model.ChannelPriceMultiplierModeCNY
		assert.True(t, dynamicCandidateLess(
			dynamicChannelCandidate{channel: &cny},
			dynamicChannelCandidate{channel: base},
		))
	})

	t.Run("probe rate breaks equal-cost ties", func(t *testing.T) {
		healthier := *base
		healthier.Id = 2
		healthier.PreviousDayProbeSuccessRate = 99
		assert.True(t, dynamicCandidateLess(
			dynamicChannelCandidate{channel: &healthier},
			dynamicChannelCandidate{channel: base},
		))
	})

	t.Run("group-scoped force does not overtake an earlier group", func(t *testing.T) {
		laterForced := *base
		laterForced.Id = 2
		laterForced.PriceMultiplier = 0.1
		laterForced.PreviousDayProbeSuccessRate = 100
		laterForced.ForcePriority = &force
		laterForced.ForcePriorityScope = model.ChannelForcePriorityScopeGroup
		assert.True(t, dynamicCandidateLess(
			dynamicChannelCandidate{channel: base, groupIndex: 0},
			dynamicChannelCandidate{channel: &laterForced, groupIndex: 1},
		))
	})

	t.Run("group-scoped force wins before price within one group", func(t *testing.T) {
		forced := *base
		forced.Id = 2
		forced.PriceMultiplier = 100
		forced.PreviousDayProbeSuccessRate = 1
		forced.ForcePriority = &force
		forced.ForcePriorityScope = model.ChannelForcePriorityScopeGroup
		assert.True(t, dynamicCandidateLess(
			dynamicChannelCandidate{channel: &forced, groupIndex: 0},
			dynamicChannelCandidate{channel: base, groupIndex: 0},
		))
	})
}

func TestWeightedDynamicCandidateIndexBoundaries(t *testing.T) {
	zeroWeight := uint(0)
	ninetyWeight := uint(90)
	channelOne := &model.Channel{Id: 1, Weight: &zeroWeight}
	channelTwo := &model.Channel{Id: 2, Weight: &ninetyWeight}

	tests := []struct {
		name       string
		candidates []dynamicChannelCandidate
		draw       int
		want       int
	}{
		{
			name: "configured zero weight uses one hundred default",
			candidates: []dynamicChannelCandidate{
				{channel: channelOne, routeConfigured: true, routeWeight: 0},
				{channel: channelTwo, routeConfigured: true, routeWeight: 50},
			},
			draw: 99,
			want: 0,
		},
		{
			name: "configured second interval starts at default boundary",
			candidates: []dynamicChannelCandidate{
				{channel: channelOne, routeConfigured: true, routeWeight: -1},
				{channel: channelTwo, routeConfigured: true, routeWeight: 50},
			},
			draw: 100,
			want: 1,
		},
		{
			name: "unconfigured zero channel weight retains plus ten baseline",
			candidates: []dynamicChannelCandidate{
				{channel: channelOne},
				{channel: channelTwo},
			},
			draw: 9,
			want: 0,
		},
		{
			name: "unconfigured second interval starts after plus ten baseline",
			candidates: []dynamicChannelCandidate{
				{channel: channelOne},
				{channel: channelTwo},
			},
			draw: 10,
			want: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, weightedDynamicCandidateIndex(test.candidates, test.draw))
		})
	}
}

func TestRankDynamicCandidatesWeightsOnlyBestEquivalentSet(t *testing.T) {
	zeroWeight := uint(0)
	ninetyWeight := uint(90)
	highWeight := uint(1000)
	first := &model.Channel{Id: 1, Weight: &zeroWeight, PriceMultiplier: 1, PreviousDayProbeSuccessRate: 90}
	second := &model.Channel{Id: 2, Weight: &ninetyWeight, PriceMultiplier: 1, PreviousDayProbeSuccessRate: 90}
	worse := &model.Channel{Id: 3, Weight: &highWeight, PriceMultiplier: 2, PreviousDayProbeSuccessRate: 90}
	draws := 0
	param := &RetryParam{randomIntn: func(total int) int {
		draws++
		assert.Equal(t, 110, total)
		return 10
	}}

	ranked := param.rankDynamicCandidates([]dynamicChannelCandidate{
		{channel: worse, group: "test", groupIndex: 0},
		{channel: first, group: "test", groupIndex: 0},
		{channel: second, group: "test", groupIndex: 0},
	})

	require.Len(t, ranked, 3)
	assert.Equal(t, 1, draws)
	assert.Equal(t, []int{2, 1, 3}, []int{ranked[0].channel.Id, ranked[1].channel.Id, ranked[2].channel.Id})
}

func TestRankDynamicCandidatesKeepsConcreteRetryFirst(t *testing.T) {
	zeroWeight := uint(0)
	ninetyWeight := uint(90)
	first := &model.Channel{Id: 1, Weight: &zeroWeight, PriceMultiplier: 1, PreviousDayProbeSuccessRate: 90}
	second := &model.Channel{Id: 2, Weight: &ninetyWeight, PriceMultiplier: 1, PreviousDayProbeSuccessRate: 90}
	param := &RetryParam{
		concreteAttempted: true,
		currentChannelID:  1,
		attemptCounts:     map[int]int{1: 1},
		channelLimits:     map[int]int{1: 2},
		channelGroups:     map[int]string{1: "test"},
		randomIntn:        func(int) int { return 10 },
	}

	ranked := param.rankDynamicCandidates([]dynamicChannelCandidate{
		{channel: first, group: "test", groupIndex: 0},
		{channel: second, group: "test", groupIndex: 0},
	})

	require.Len(t, ranked, 2)
	assert.Equal(t, 1, ranked[0].channel.Id)
}
