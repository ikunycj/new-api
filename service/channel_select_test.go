package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOrderedRoutingChannels(t *testing.T, groups ...string) map[string]int {
	t.Helper()

	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
	require.NoError(t, model.DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM channels").Error)

	ids := make(map[string]int, len(groups))
	priority := int64(10)
	weight := uint(100)
	for i, group := range groups {
		channel := &model.Channel{
			Id:       91000 + i,
			Name:     "routing-" + group,
			Key:      "test-key",
			Status:   common.ChannelStatusEnabled,
			Models:   "shared-model",
			Group:    group,
			Priority: &priority,
			Weight:   &weight,
		}
		require.NoError(t, model.DB.Create(channel).Error)
		require.NoError(t, model.DB.Create(&model.Ability{
			Group:     group,
			Model:     "shared-model",
			ChannelId: channel.Id,
			Enabled:   true,
			Priority:  &priority,
			Weight:    weight,
		}).Error)
		ids[group] = channel.Id
	}

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	model.InitChannelCache()
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		_ = model.DB.Exec("DELETE FROM abilities").Error
		_ = model.DB.Exec("DELETE FROM channels").Error
	})
	return ids
}

func orderedRoutingContext(groups []string, crossGroupRetry bool) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupCandidates, groups)
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, crossGroupRetry)
	return ctx
}

func orderedRoutingContextWithRetryTimes(groups []string, retryTimes map[string]int) *gin.Context {
	ctx := orderedRoutingContext(groups, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroupRetryTimes, retryTimes)
	return ctx
}

func TestCacheGetRandomSatisfiedChannelUsesExplicitCandidateOrder(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "openai", "claude")
	ctx := orderedRoutingContext([]string{"openai", "claude"}, true)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "shared-model"}

	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, ids["openai"], channel.Id)
	assert.Equal(t, "openai", selectedGroup)
}

func TestCacheGetRandomSatisfiedChannelSkipsUnavailableCandidatesBeforeAttempt(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "claude")
	ctx := orderedRoutingContext([]string{"openai", "claude"}, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "shared-model"}

	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, ids["claude"], channel.Id)
	assert.Equal(t, "claude", selectedGroup)
}

func TestRetryParamCrossGroupTransitionIgnoresGlobalRetryCount(t *testing.T) {
	setupOrderedRoutingChannels(t, "openai", "claude")
	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 0
	t.Cleanup(func() { common.RetryTimes = originalRetryTimes })

	ctx := orderedRoutingContext([]string{"openai", "claude"}, true)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "shared-model"}
	first, firstGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "openai", firstGroup)

	param.MarkAttempted()
	require.True(t, param.HasNextRetry())
	require.True(t, param.AdvanceRetry())
	second, secondGroup, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, "claude", secondGroup)
}

func TestRetryParamUsesConfiguredPerGroupRetryLimits(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "openai", "claude")
	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 10
	t.Cleanup(func() { common.RetryTimes = originalRetryTimes })

	ctx := orderedRoutingContextWithRetryTimes(
		[]string{"openai", "claude"},
		map[string]int{"openai": 0, "claude": 3},
	)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "shared-model"}

	first, firstGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, ids["openai"], first.Id)
	assert.Equal(t, "openai", firstGroup)

	param.MarkChannelAttempted(first.Id, first.ClusterId)
	require.True(t, param.HasNextRetry())
	require.True(t, param.AdvanceRetry())

	second, secondGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, ids["claude"], second.Id)
	assert.Equal(t, "claude", secondGroup)
}

func TestRetryParamAllowsConfiguredGroupRetries(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "openai")
	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 10
	t.Cleanup(func() { common.RetryTimes = originalRetryTimes })

	ctx := orderedRoutingContextWithRetryTimes([]string{"openai"}, map[string]int{"openai": 3})
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "shared-model"}

	for attempt := 0; attempt < 4; attempt++ {
		channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, ids["openai"], channel.Id)
		assert.Equal(t, "openai", selectedGroup)

		param.MarkChannelAttempted(channel.Id, channel.ClusterId)
		if attempt < 3 {
			require.True(t, param.HasNextRetry())
			require.True(t, param.AdvanceRetry())
		} else {
			assert.False(t, param.HasNextRetry())
			assert.False(t, param.AdvanceRetry())
		}
	}
}

func TestRetryParamDoesNotCrossGroupsWhenDisabled(t *testing.T) {
	setupOrderedRoutingChannels(t, "openai", "claude")
	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 0
	t.Cleanup(func() { common.RetryTimes = originalRetryTimes })

	ctx := orderedRoutingContext([]string{"openai", "claude"}, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "auto", ModelName: "shared-model"}
	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, "openai", selectedGroup)

	param.MarkAttempted()
	assert.False(t, param.HasNextRetry())
	assert.False(t, param.AdvanceRetry())
}

func TestRetryParamSwitchesToUntriedChannelWithoutGlobalRetryBudget(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "ikun")
	priority := int64(10)
	weight := uint(100)
	second := &model.Channel{
		Id:       91999,
		Name:     "routing-ikun-secondary",
		Key:      "test-key-secondary",
		Status:   common.ChannelStatusEnabled,
		Models:   "shared-model",
		Group:    "ikun",
		Priority: &priority,
		Weight:   &weight,
	}
	require.NoError(t, model.DB.Create(second).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "ikun",
		Model:     "shared-model",
		ChannelId: second.Id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
	model.InitChannelCache()

	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 0
	t.Cleanup(func() { common.RetryTimes = originalRetryTimes })

	ctx := orderedRoutingContext(nil, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "ikun", ModelName: "shared-model"}
	first, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	param.MarkChannelAttempted(first.Id, first.ClusterId)

	require.True(t, param.HasNextRetry())
	require.True(t, param.AdvanceRetry())
	next, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, "ikun", selectedGroup)
	assert.NotEqual(t, first.Id, next.Id)
	assert.Contains(t, []int{ids["ikun"], second.Id}, next.Id)
	param.MarkChannelAttempted(next.Id, next.ClusterId)
	assert.False(t, param.HasNextRetry())
}

func TestCacheGetRandomSatisfiedChannelHonorsExplicitGroupOverride(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "openai", "claude")
	ctx := orderedRoutingContext([]string{"openai", "claude"}, true)
	param := &RetryParam{Ctx: ctx, TokenGroup: "claude", ModelName: "shared-model"}

	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, ids["claude"], channel.Id)
	assert.Equal(t, "claude", selectedGroup)
	assert.False(t, param.IsAutoRouting())
}

func TestRetryParamExcludesEveryChannelInFailedCluster(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "ikun")
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", ids["ikun"]).Update("cluster_id", 1).Error)
	priority := int64(10)
	weight := uint(100)
	channels := []model.Channel{
		{Id: 92001, Name: "cluster-one-a", Key: "a", Status: common.ChannelStatusEnabled, Models: "shared-model", Group: "ikun", Priority: &priority, Weight: &weight, ClusterId: 1},
		{Id: 92002, Name: "cluster-one-b", Key: "b", Status: common.ChannelStatusEnabled, Models: "shared-model", Group: "ikun", Priority: &priority, Weight: &weight, ClusterId: 1},
		{Id: 92003, Name: "cluster-two", Key: "c", Status: common.ChannelStatusEnabled, Models: "shared-model", Group: "ikun", Priority: &priority, Weight: &weight, ClusterId: 2},
	}
	for i := range channels {
		require.NoError(t, model.DB.Create(&channels[i]).Error)
		require.NoError(t, model.DB.Create(&model.Ability{Group: "ikun", Model: "shared-model", ChannelId: channels[i].Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
	}
	model.InitChannelCache()

	ctx := orderedRoutingContext(nil, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "ikun", ModelName: "shared-model"}
	param.MarkChannelAttempted(channels[0].Id, 1)
	param.ExcludeCluster(1)

	require.True(t, param.HasNextRetry())
	next, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, 2, next.ClusterId)
}

func TestRetryParamSkipsOpenClusterWithoutGlobalRetryBudget(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "ikun")
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", ids["ikun"]).Update("cluster_id", 1).Error)
	priority := int64(10)
	weight := uint(100)
	second := &model.Channel{
		Id:        92004,
		Name:      "cluster-two",
		Key:       "cluster-two-key",
		Status:    common.ChannelStatusEnabled,
		Models:    "shared-model",
		Group:     "ikun",
		Priority:  &priority,
		Weight:    &weight,
		ClusterId: 2,
	}
	require.NoError(t, model.DB.Create(second).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group: "ikun", Model: "shared-model", ChannelId: second.Id,
		Enabled: true, Priority: &priority, Weight: weight,
	}).Error)
	model.InitChannelCache()

	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 0
	t.Cleanup(func() { common.RetryTimes = originalRetryTimes })

	policy := model.DefaultRuntimeFailoverPolicy(model.FailoverModeBalanced)
	ctx := orderedRoutingContext(nil, false)
	param := &RetryParam{
		Ctx: ctx, TokenGroup: "ikun", ModelName: "shared-model",
		runtimePolicy: &policy, clusterOrder: []int{1, 2}, startedAt: time.Now(),
	}
	first, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Equal(t, 1, first.ClusterId)

	param.ExcludeCluster(first.ClusterId)
	require.True(t, param.HasNextRetry())
	require.True(t, param.AdvanceRetry())
	next, _, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, 2, next.ClusterId)
}

func TestRetryParamKeepsClusterAvailableForChannelScopedFailure(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "ikun")
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", ids["ikun"]).Updates(map[string]any{"cluster_id": 1, "cluster_pool_id": 1}).Error)
	priority := int64(9)
	weight := uint(100)
	second := &model.Channel{
		Id: 92011, Name: "cluster-one-premium", Key: "premium", Status: common.ChannelStatusEnabled,
		Models: "shared-model", Group: "ikun", Priority: &priority, Weight: &weight, ClusterId: 1, ClusterPoolId: 2,
	}
	require.NoError(t, model.DB.Create(second).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "ikun", Model: "shared-model", ChannelId: second.Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
	model.InitChannelCache()

	ctx := orderedRoutingContext(nil, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "ikun", ModelName: "shared-model"}
	first, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	param.MarkChannelAttempted(first.Id, first.ClusterId)

	require.True(t, param.HasNextRetry())
	require.True(t, param.AdvanceRetry())
	next, _, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, 1, next.ClusterId)
	assert.NotEqual(t, first.Id, next.Id)
}

func TestRetryParamClusterLimitStillAllowsVisitedClusterPools(t *testing.T) {
	policy := model.DefaultRuntimeFailoverPolicy(model.FailoverModeBalanced)
	policy.MaxClusterAttempts = 1
	param := &RetryParam{runtimePolicy: &policy}
	param.MarkChannelAttempted(1, 17)

	assert.True(t, param.CanAttemptCluster(17))
	assert.False(t, param.CanAttemptCluster(23))
}

func TestCacheGetRandomSatisfiedChannelUsesPolicyPoolOrder(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "strategy")
	require.NoError(t, model.DB.AutoMigrate(&model.ClusterPool{}))
	require.NoError(t, model.DB.Create(&[]model.ClusterPool{
		{Id: 81, ClusterId: 1, Tier: model.PoolTierFree, Name: "cheap", Status: model.ClusterStatusEnabled, CostFactor: 1},
		{Id: 82, ClusterId: 1, Tier: model.PoolTierFallback, Name: "stable", Status: model.ClusterStatusEnabled, CostFactor: 2},
	}).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", ids["strategy"]).Updates(map[string]any{"cluster_id": 1, "cluster_pool_id": 81}).Error)
	priority := int64(1)
	weight := uint(100)
	stable := &model.Channel{
		Id: 93001, Name: "stable", Key: "stable", Status: common.ChannelStatusEnabled,
		Models: "shared-model", Group: "strategy", Priority: &priority, Weight: &weight, ClusterId: 1, ClusterPoolId: 82,
	}
	require.NoError(t, model.DB.Create(stable).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "strategy", Model: "shared-model", ChannelId: stable.Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
	model.InitChannelCache()

	policy := model.DefaultRuntimeFailoverPolicy(model.FailoverModeBalanced)
	policy.PoolTiers = []int{model.PoolTierFallback, model.PoolTierFree}
	param := &RetryParam{Ctx: orderedRoutingContext(nil, false), TokenGroup: "strategy", ModelName: "shared-model", runtimePolicy: &policy}
	channel, _, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, stable.Id, channel.Id)
}

func TestCacheGetRandomSatisfiedChannelUsesExplicitPolicyChannelOrder(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "direct")
	priority := int64(1)
	weight := uint(100)
	second := &model.Channel{
		Id: 93002, Name: "direct-second", Key: "second", Status: common.ChannelStatusEnabled,
		Models: "shared-model", Group: "direct", Priority: &priority, Weight: &weight,
	}
	require.NoError(t, model.DB.Create(second).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "direct", Model: "shared-model", ChannelId: second.Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
	model.InitChannelCache()

	policy := model.DefaultRuntimeFailoverPolicy(model.FailoverModeBalanced)
	policy.ChannelIDs = []int{second.Id, ids["direct"]}
	policy.PoolTiers = nil
	param := &RetryParam{Ctx: orderedRoutingContext(nil, false), TokenGroup: "direct", ModelName: "shared-model", runtimePolicy: &policy, startedAt: time.Now()}
	channel, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, second.Id, channel.Id)

	param.MarkChannelAttempted(channel.Id, channel.ClusterId)
	assert.True(t, param.HasNextRetry())
	assert.True(t, param.AdvanceRetry())
	next, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, ids["direct"], next.Id)
}

func TestRetryParamEnforcesPoolRetryAndTierBudgets(t *testing.T) {
	policy := model.DefaultRuntimeFailoverPolicy(model.FailoverModeBalanced)
	policy.SamePoolRetries = 1
	policy.MaxPoolAttempts = 2
	param := &RetryParam{runtimePolicy: &policy}

	assert.True(t, param.CanAttemptPool(17, model.PoolTierFree))
	param.MarkPoolAttempted(17, model.PoolTierFree)
	assert.True(t, param.CanAttemptPool(17, model.PoolTierFree))
	param.MarkPoolAttempted(17, model.PoolTierFree)
	assert.False(t, param.CanAttemptPool(17, model.PoolTierFree))

	assert.True(t, param.CanAttemptPool(17, model.PoolTierPremium))
	param.MarkPoolAttempted(17, model.PoolTierPremium)
	assert.False(t, param.CanAttemptPool(17, model.PoolTierFallback))
}

func TestFailoverGroupPriorityOverridesChannelPriorityAcrossClusters(t *testing.T) {
	ids := setupOrderedRoutingChannels(t, "ikun")
	require.NoError(t, model.DB.AutoMigrate(
		&model.Cluster{}, &model.ClusterPool{}, &model.FailoverPolicy{}, &model.FailoverGroup{},
		&model.FailoverGroupMember{}, &model.FailoverRule{}, &model.UpstreamErrorMapping{},
	))
	for _, table := range []string{"failover_rules", "failover_group_members", "failover_groups", "failover_policies", "clusters"} {
		require.NoError(t, model.DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"failover_rules", "failover_group_members", "failover_groups", "failover_policies", "clusters"} {
			_ = model.DB.Exec("DELETE FROM " + table).Error
		}
		model.InitFailoverCache()
	})
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", ids["ikun"]).Update("cluster_id", 1).Error)
	highPriority := int64(100)
	weight := uint(100)
	second := &model.Channel{
		Id: 92021, Name: "cluster-two-higher-channel-priority", Key: "cluster-two", Status: common.ChannelStatusEnabled,
		Models: "shared-model", Group: "ikun", Priority: &highPriority, Weight: &weight, ClusterId: 2,
	}
	require.NoError(t, model.DB.Create(second).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "ikun", Model: "shared-model", ChannelId: second.Id, Enabled: true, Priority: &highPriority, Weight: weight}).Error)
	require.NoError(t, model.DB.Create(&model.FailoverPolicy{
		Id: 51, Name: "ordered", Mode: model.FailoverModeBalanced, Enabled: true,
		MaxPoolAttempts: 3, MaxClusterAttempts: 2, MaxTotalAttempts: 6, TotalFailoverBudgetMs: 10000,
		CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60,
		CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
	}).Error)
	require.NoError(t, model.DB.Create(&model.FailoverGroup{Id: 51, Name: "ordered", PolicyId: 51, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&[]model.FailoverGroupMember{
		{FailoverGroupId: 51, ClusterId: 1, Priority: 100, Weight: 100},
		{FailoverGroupId: 51, ClusterId: 2, Priority: 90, Weight: 100},
	}).Error)
	require.NoError(t, model.DB.Create(&model.FailoverRule{FailoverGroupId: 51, ModelPattern: "shared-*", RoutePattern: "/v1/*", UserGroup: "*", Priority: 100, Enabled: true}).Error)
	model.InitChannelCache()

	ctx := orderedRoutingContext(nil, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "ikun", ModelName: "shared-model", RequestPath: "/v1/chat"}
	channel, _, err := CacheGetRandomSatisfiedChannel(param)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 1, channel.ClusterId)
}

func TestRetryParamUsesPolicyOfSelectedCluster(t *testing.T) {
	require.NoError(t, model.DB.AutoMigrate(&model.Cluster{}, &model.FailoverPolicy{}))
	for _, table := range []string{"failover_policies", "clusters"} {
		require.NoError(t, model.DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"failover_policies", "clusters"} {
			_ = model.DB.Exec("DELETE FROM " + table).Error
		}
		model.InitFailoverCache()
	})
	require.NoError(t, model.DB.Create(&[]model.FailoverPolicy{
		{Id: 81, Name: "conservative", Mode: model.FailoverModeConservative, Enabled: true, MaxPoolAttempts: 4, MaxClusterAttempts: 2, MaxTotalAttempts: 4, TotalFailoverBudgetMs: 12000, CircuitFailureThreshold: 8, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60, CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2},
		{Id: 82, Name: "aggressive", Mode: model.FailoverModeAggressive, Enabled: true, MaxPoolAttempts: 4, MaxClusterAttempts: 4, MaxTotalAttempts: 8, TotalFailoverBudgetMs: 6000, CircuitFailureThreshold: 3, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 90, CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2},
	}).Error)
	require.NoError(t, model.DB.Create(&[]model.Cluster{
		{Id: 81, Name: "primary", Type: "ikun", Status: model.ClusterStatusEnabled, BillingGroup: "cluster_pro", PolicyId: 81, FailoverPriority: 100},
		{Id: 82, Name: "secondary", Type: "ikun", Status: model.ClusterStatusEnabled, BillingGroup: "cluster_pro", PolicyId: 82, FailoverPriority: 90},
	}).Error)
	model.InitFailoverCache()

	ctx := orderedRoutingContext(nil, false)
	param := &RetryParam{Ctx: ctx, TokenGroup: "cluster_pro", ModelName: "shared-model"}

	assert.Equal(t, model.FailoverModeConservative, param.RuntimePolicyForCluster(81).Mode)
	assert.Equal(t, model.FailoverModeAggressive, param.RuntimePolicyForCluster(82).Mode)
}
