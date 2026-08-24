package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
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
	weight := uint(100)
	channels := []model.Channel{
		{Id: 92001, Name: "Claude Pro", Key: "pro", Status: common.ChannelStatusEnabled, Models: "claude-test", Group: "claude", Priority: &priority, Weight: &weight},
		{Id: 92002, Name: "Claude Official", Key: "official", Status: common.ChannelStatusEnabled, Models: "claude-test", Group: "claude", Priority: &priority, Weight: &weight},
	}
	require.NoError(t, model.DB.Create(&channels).Error)
	for _, channel := range channels {
		require.NoError(t, model.DB.Create(&model.Ability{Group: "claude", Model: "claude-test", ChannelId: channel.Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
	}
	route := model.BillingGroupRoute{Id: 81, BillingGroup: "claude", Name: "Claude", Mode: model.RoutingModeBalanced, Enabled: true, MaxTotalAttempts: 3, TotalTimeoutMs: 30000, CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60, CircuitHalfOpenRequests: 1}
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

func TestConfiguredRouteWithoutMemoryCacheHonorsRoutePriority(t *testing.T) {
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
