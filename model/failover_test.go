package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFailoverTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(
		&Cluster{},
		&ClusterPool{},
		&FailoverPolicy{},
		&FailoverPolicyStep{},
		&FailoverGroup{},
		&FailoverGroupMember{},
		&FailoverRule{},
		&UpstreamErrorMapping{},
	))
	for _, table := range []string{
		"upstream_error_mappings",
		"failover_rules",
		"failover_group_members",
		"failover_groups",
		"failover_policy_steps",
		"failover_policies",
		"cluster_pools",
		"clusters",
	} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{
			"upstream_error_mappings",
			"failover_rules",
			"failover_group_members",
			"failover_groups",
			"failover_policy_steps",
			"failover_policies",
			"cluster_pools",
			"clusters",
		} {
			_ = DB.Exec("DELETE FROM " + table).Error
		}
		InitFailoverCache()
	})
}

func TestSaveFailoverConfigNormalizesAndArchivesRemovedRecords(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&Cluster{Id: 9, Name: "old", Type: "custom", Status: ClusterStatusEnabled}).Error)
	require.NoError(t, DB.Create(&ClusterPool{Id: 9, ClusterId: 9, Tier: PoolTierFree, Name: "old", Status: ClusterStatusEnabled}).Error)
	require.NoError(t, DB.Create(&FailoverPolicy{Id: 9, Name: "old", Mode: FailoverModeBalanced, Enabled: true}).Error)
	require.NoError(t, DB.Create(&UpstreamErrorMapping{Id: 9, ClusterType: "custom", RawCode: "old", AlltokenCode: 200001, Category: "unknown", FailureScope: "cluster", Action: "failover", Enabled: true}).Error)

	config := &FailoverConfig{
		Clusters: []Cluster{{Id: 17, Name: " Cluster A ", Type: " IKUN ", Status: ClusterStatusEnabled}},
		Pools:    []ClusterPool{{Id: 17, ClusterId: 17, Tier: PoolTierPremium, Name: "Pro", Status: ClusterStatusEnabled, CostFactor: 1.5}},
		Policies: []FailoverPolicy{{
			Id: 17, Name: "", Mode: "unknown", Enabled: true,
			MaxPoolAttempts: 3, MaxClusterAttempts: 3, MaxTotalAttempts: 2, TotalFailoverBudgetMs: 5000,
			CircuitFailureThreshold: 3, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 60,
			CircuitHalfOpenRequests: 1,
		}},
		ErrorMappings: []UpstreamErrorMapping{{
			Id: 17, ClusterType: " IKUN ", RawCode: " POOL_EXHAUSTED ", StatusCode: 503,
			AlltokenCode: 205004, Category: "upstream", FailureScope: "channel", Action: "failover", Retryable: true, Enabled: true,
		}},
	}

	require.NoError(t, SaveFailoverConfig(config))
	InitFailoverCache()

	var policy FailoverPolicy
	require.NoError(t, DB.First(&policy, "id = ?", 17).Error)
	assert.Equal(t, FailoverModeBalanced, policy.Mode)
	assert.Equal(t, FailoverModeBalanced, policy.Name)
	assert.Equal(t, 3, policy.MaxTotalAttempts)
	assert.Equal(t, float64(1), policy.MaxCostMultiplier)
	runtimePolicy := GetRuntimeFailoverPolicy(FailoverModeBalanced)
	assert.Equal(t, 3, runtimePolicy.MaxTotalAttempts)
	assert.Equal(t, 5000, runtimePolicy.TotalFailoverBudgetMs)

	var oldCluster Cluster
	require.NoError(t, DB.First(&oldCluster, "id = ?", 9).Error)
	assert.True(t, oldCluster.Archived)
	assert.Equal(t, ClusterStatusDisabled, oldCluster.Status)

	var oldPool ClusterPool
	require.NoError(t, DB.First(&oldPool, "id = ?", 9).Error)
	assert.Equal(t, ClusterStatusDisabled, oldPool.Status)

	var oldPolicy FailoverPolicy
	require.NoError(t, DB.First(&oldPolicy, "id = ?", 9).Error)
	assert.False(t, oldPolicy.Enabled)

	var oldMapping UpstreamErrorMapping
	require.NoError(t, DB.First(&oldMapping, "id = ?", 9).Error)
	assert.False(t, oldMapping.Enabled)
}

func TestResolveRuntimeFailoverUsesNamedMixedChannelGroup(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&[]Cluster{
		{Id: 21, Name: "primary", Type: "ikun", Status: ClusterStatusEnabled, BillingGroup: "mixed-claude"},
		{Id: 22, Name: "backup", Type: "claude", Status: ClusterStatusEnabled, BillingGroup: "mixed-claude"},
	}).Error)
	require.NoError(t, DB.Create(&FailoverPolicy{Id: 21, Name: "mixed", Mode: FailoverModeBalanced, Strategy: RoutingStrategyBalanced, Enabled: true, MaxPoolAttempts: 2, MaxClusterAttempts: 2, MaxTotalAttempts: 4, TotalFailoverBudgetMs: 5000, CircuitFailureThreshold: 3, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 30, CircuitHalfOpenRequests: 1}).Error)
	require.NoError(t, DB.Create(&FailoverGroup{Id: 21, Name: "mixed-claude", PolicyId: 21, Enabled: true}).Error)
	require.NoError(t, DB.Create(&[]FailoverGroupMember{
		{FailoverGroupId: 21, ClusterId: 22, Priority: 200, Weight: 100},
		{FailoverGroupId: 21, ClusterId: 21, Priority: 100, Weight: 100},
	}).Error)

	InitFailoverCache()
	policy, order, direct := ResolveRuntimeFailover("", "claude-opus", "/v1/messages", "default", "mixed-claude")
	assert.Equal(t, RoutingStrategyBalanced, policy.Strategy)
	assert.Equal(t, []int{22, 21}, order)
	assert.True(t, direct)
}

func TestSaveFailoverConfigAllowsCrossBillingGroupMembers(t *testing.T) {
	setupFailoverTables(t)
	config := &FailoverConfig{
		Clusters: []Cluster{
			{Id: 101, Name: "primary", Type: "ikun", Status: ClusterStatusEnabled, BillingGroup: "Cluster_1"},
			{Id: 102, Name: "secondary", Type: "ikun2", Status: ClusterStatusEnabled, BillingGroup: "Cluster_2"},
		},
		Policies: []FailoverPolicy{{
			Id: 101, Name: "cross-group", Mode: FailoverModeBalanced, Enabled: true,
			MaxPoolAttempts: 4, MaxClusterAttempts: 2, MaxTotalAttempts: 8, TotalFailoverBudgetMs: 12000,
			CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60,
			CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
		}},
		Groups: []FailoverGroup{{Id: 101, Name: "primary-to-secondary", PolicyId: 101, Enabled: true}},
		GroupMembers: []FailoverGroupMember{
			{FailoverGroupId: 101, ClusterId: 101, Priority: 100, Weight: 100},
			{FailoverGroupId: 101, ClusterId: 102, Priority: 90, Weight: 100},
		},
		Rules: []FailoverRule{{
			FailoverGroupId: 101, ModelPattern: "gpt-*", RoutePattern: "/v1/chat/completions", UserGroup: "*", Priority: 100, Enabled: true,
		}},
	}

	require.NoError(t, SaveFailoverConfig(config))
	InitFailoverCache()

	_, clusterOrder, rulePolicy := ResolveRuntimeFailover("", "gpt-5.4-mini", "/v1/chat/completions", "default", "Cluster_1")
	assert.Equal(t, []int{101, 102}, clusterOrder)
	assert.True(t, rulePolicy)
}

func TestMatchUpstreamErrorMappingUsesMostSpecificRule(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&Cluster{Id: 23, Name: "IKUN A", Type: "ikun", Status: ClusterStatusEnabled}).Error)
	require.NoError(t, DB.Create(&[]UpstreamErrorMapping{
		{Id: 1, ClusterType: "*", RawCode: "*", StatusCode: 503, AlltokenCode: 205002, Category: "upstream", FailureScope: "cluster", Action: "failover", Retryable: true, Enabled: true},
		{Id: 2, ClusterType: "ikun", RawCode: "pool_exhausted", StatusCode: 503, AlltokenCode: 205004, Category: "upstream", FailureScope: "channel", Action: "failover", Retryable: true, Enabled: true},
	}).Error)
	InitFailoverCache()

	mapping, ok := MatchUpstreamErrorMapping(23, "POOL_EXHAUSTED", 503)

	require.True(t, ok)
	assert.Equal(t, 205004, mapping.AlltokenCode)
	assert.Equal(t, "channel", mapping.FailureScope)
}

func TestChannelAllowedByFailoverPolicyEnforcesTierAndCost(t *testing.T) {
	setupFailoverTables(t)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCacheEnabled })
	require.NoError(t, DB.Create(&ClusterPool{Id: 41, ClusterId: 1, Tier: PoolTierPremium, Name: "Pro", Status: ClusterStatusEnabled, CostFactor: 1.5}).Error)
	channel := &Channel{ClusterPoolId: 41}
	policy := DefaultRuntimeFailoverPolicy(FailoverModeBalanced)

	policy.AllowPaidEscalation = false
	assert.False(t, ChannelAllowedByFailoverPolicy(channel, policy))

	policy.AllowPaidEscalation = true
	policy.MaxCostMultiplier = 1
	assert.False(t, ChannelAllowedByFailoverPolicy(channel, policy))

	policy.MaxCostMultiplier = 2
	assert.True(t, ChannelAllowedByFailoverPolicy(channel, policy))
}

func TestResolveRuntimeFailoverAppliesRulePolicyAndClusterGroup(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&FailoverPolicy{
		Id: 31, Name: "rule-policy", Mode: FailoverModeAggressive, Enabled: true,
		MaxPoolAttempts: 3, MaxClusterAttempts: 2, MaxTotalAttempts: 5, TotalFailoverBudgetMs: 4500,
		CircuitFailureThreshold: 3, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 60,
		CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
	}).Error)
	require.NoError(t, DB.Create(&FailoverGroup{Id: 41, Name: "primary", PolicyId: 31, Enabled: true}).Error)
	require.NoError(t, DB.Create(&[]FailoverGroupMember{
		{FailoverGroupId: 41, ClusterId: 17, Priority: 100, Weight: 100},
		{FailoverGroupId: 41, ClusterId: 23, Priority: 90, Weight: 100},
	}).Error)
	require.NoError(t, DB.Create(&FailoverRule{
		FailoverGroupId: 41, ModelPattern: "gpt-*", RoutePattern: "/v1/*", UserGroup: "pro", Priority: 100, Enabled: true,
	}).Error)
	InitFailoverCache()

	policy, clusterOrder, rulePolicy := ResolveRuntimeFailover("", "gpt-5", "/v1/chat", "pro", "")

	assert.Equal(t, FailoverModeAggressive, policy.Mode)
	assert.Equal(t, 5, policy.MaxTotalAttempts)
	assert.Equal(t, []int{17, 23}, clusterOrder)
	assert.True(t, rulePolicy)
}

func TestResolveRuntimeFailoverUsesBillingGroupClusterOrder(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&FailoverPolicy{
		Id: 61, Name: "cluster-policy", Mode: FailoverModeConservative, Enabled: true,
		MaxPoolAttempts: 3, MaxClusterAttempts: 2, MaxTotalAttempts: 4, TotalFailoverBudgetMs: 7000,
		CircuitFailureThreshold: 4, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 60,
		CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
	}).Error)
	require.NoError(t, DB.Create(&FailoverPolicy{
		Id: 62, Name: "secondary-policy", Mode: FailoverModeBalanced, Enabled: true,
		MaxPoolAttempts: 4, MaxClusterAttempts: 3, MaxTotalAttempts: 6, TotalFailoverBudgetMs: 9000,
		CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60,
		CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
	}).Error)
	require.NoError(t, DB.Create(&[]Cluster{
		{Id: 71, Name: "secondary", Type: "ikun", Status: ClusterStatusEnabled, BillingGroup: "cluster_pro", PolicyId: 62, FailoverPriority: 50},
		{Id: 72, Name: "primary", Type: "ikun", Status: ClusterStatusEnabled, BillingGroup: "cluster_pro", PolicyId: 61, FailoverPriority: 100},
	}).Error)
	InitFailoverCache()

	policy, clusterOrder, rulePolicy := ResolveRuntimeFailover("", "gpt-5", "/v1/chat", "user", "cluster_pro")

	assert.Equal(t, FailoverModeConservative, policy.Mode)
	assert.Equal(t, []int{72, 71}, clusterOrder)
	assert.False(t, rulePolicy)
	assert.Equal(t, FailoverModeBalanced, GetRuntimeFailoverPolicyForCluster(71, DefaultRuntimeFailoverPolicy("")).Mode)
}

func TestDefaultRuntimeFailoverPolicyUsesStrategyPoolOrder(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		strategy  string
		poolTiers []int
	}{
		{name: "cost", mode: FailoverModeAggressive, strategy: RoutingStrategyCostFirst, poolTiers: []int{PoolTierFree, PoolTierPremium, PoolTierFallback, PoolTierEmergency}},
		{name: "balanced", mode: FailoverModeBalanced, strategy: RoutingStrategyBalanced, poolTiers: []int{PoolTierPremium, PoolTierFree, PoolTierFallback, PoolTierEmergency}},
		{name: "stability", mode: FailoverModeConservative, strategy: RoutingStrategyStabilityFirst, poolTiers: []int{PoolTierFallback, PoolTierEmergency, PoolTierPremium, PoolTierFree}},
		{name: "pro cost", mode: FailoverModeBalanced, strategy: RoutingStrategyProCostFirst, poolTiers: []int{PoolTierPremium, PoolTierFallback}},
		{name: "pro stability", mode: FailoverModeBalanced, strategy: RoutingStrategyProStabilityFirst, poolTiers: []int{PoolTierPremium, PoolTierFallback}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := DefaultRuntimeFailoverPolicy(test.mode)
			policy.Strategy = test.strategy
			policy = withDefaultPoolOrder(policy)
			assert.Equal(t, test.strategy, policy.Strategy)
			assert.Equal(t, test.poolTiers, policy.PoolTiers)
		})
	}
}

func TestResolveRuntimeFailoverKeepsCustomStrategySeparateFromModePolicy(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&[]FailoverPolicy{
		{
			Id: 101, Name: "balanced-default", Mode: FailoverModeBalanced, Strategy: RoutingStrategyBalanced, Enabled: true,
			MaxPoolAttempts: 4, MaxClusterAttempts: 3, MaxTotalAttempts: 6, TotalFailoverBudgetMs: 10000,
			CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60,
			CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
		},
		{
			Id: 102, Name: "claude-pro-cost", Mode: FailoverModeBalanced, Strategy: RoutingStrategyProCostFirst, Enabled: true,
			MaxPoolAttempts: 2, MaxClusterAttempts: 2, MaxTotalAttempts: 8, TotalFailoverBudgetMs: 30000,
			CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60,
			CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2,
		},
	}).Error)
	require.NoError(t, DB.Create(&[]FailoverPolicyStep{
		{PolicyId: 102, StepOrder: 1, PoolTier: PoolTierPremium, MaxAttempts: 3},
		{PolicyId: 102, StepOrder: 2, PoolTier: PoolTierFallback, MaxAttempts: 1},
	}).Error)
	InitFailoverCache()

	modePolicy := GetRuntimeFailoverPolicy(FailoverModeBalanced)
	assert.Equal(t, RoutingStrategyBalanced, modePolicy.Strategy)

	strategyPolicy, _, _ := ResolveRuntimeFailoverWithStrategy("", RoutingStrategyProCostFirst, "claude-sonnet-4-6", "/v1/messages", "default", "claude")
	assert.Equal(t, RoutingStrategyProCostFirst, strategyPolicy.Strategy)
	assert.Equal(t, []int{PoolTierPremium, PoolTierFallback}, strategyPolicy.PoolTiers)
	assert.Equal(t, 3, strategyPolicy.PoolAttemptsByTier[PoolTierPremium])
	assert.Equal(t, 1, strategyPolicy.PoolAttemptsByTier[PoolTierFallback])
}

func TestResolveRuntimeFailoverUsesPackageStrategyPolicySteps(t *testing.T) {
	setupFailoverTables(t)
	require.NoError(t, DB.Create(&FailoverPolicy{
		Id: 91, Name: "stable-package", Mode: FailoverModeBalanced, Strategy: RoutingStrategyStabilityFirst, Enabled: true,
		MaxPoolAttempts: 4, MaxClusterAttempts: 2, MaxTotalAttempts: 6, TotalFailoverBudgetMs: 8000,
		CircuitFailureThreshold: 4, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 60,
		CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 3,
	}).Error)
	require.NoError(t, DB.Create(&[]FailoverPolicyStep{
		{PolicyId: 91, StepOrder: 1, PoolTier: PoolTierEmergency, MaxAttempts: 2},
		{PolicyId: 91, StepOrder: 2, PoolTier: PoolTierFallback, MaxAttempts: 1},
	}).Error)
	require.NoError(t, DB.Create(&Cluster{Id: 91, Name: "package-cluster", Type: "ikun", Status: ClusterStatusEnabled, BillingGroup: "package_group", PolicyId: 91, FailoverPriority: 100}).Error)
	InitFailoverCache()

	policy, clusterOrder, rulePolicy := ResolveRuntimeFailoverWithStrategy("", RoutingStrategyStabilityFirst, "gpt-5.4-mini", "/v1/chat/completions", "default", "package_group")

	assert.Equal(t, RoutingStrategyStabilityFirst, policy.Strategy)
	assert.Equal(t, []int{PoolTierEmergency, PoolTierFallback}, policy.PoolTiers)
	assert.Equal(t, 2, policy.PoolAttemptsByTier[PoolTierEmergency])
	assert.Equal(t, []int{91}, clusterOrder)
	assert.False(t, rulePolicy)
}
