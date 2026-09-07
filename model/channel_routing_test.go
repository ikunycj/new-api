package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelRoutingTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &BillingGroupRoute{}, &BillingGroupChannel{}, &UpstreamErrorMapping{}))
	for _, table := range []string{"channel_error_mappings", "billing_group_channels", "billing_group_routes", "abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"channel_error_mappings", "billing_group_channels", "billing_group_routes", "abilities", "channels"} {
			_ = DB.Exec("DELETE FROM " + table).Error
		}
		InitChannelRoutingCache()
	})
}

func TestSaveChannelRoutingConfigPersistsOrderedChannelsAndRemovesMissingRows(t *testing.T) {
	setupChannelRoutingTables(t)
	proWeight := uint(37)
	officialWeight := uint(83)
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 38, Name: "Pro", Group: "claude", Weight: &proWeight},
		{Id: 40, Name: "Official", Group: "claude", Weight: &officialWeight},
	}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "old", Name: "old", Enabled: true}).Error)

	config := &ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: 17, BillingGroup: " claude ", Name: " Claude ", Mode: RoutingModeStabilityFirst, Enabled: true,
			MaxTotalAttempts: 2, ProfitGuardMode: ProfitGuardModeWarn, MinimumProfitMargin: 12.5,
		}},
		RouteChannels: []BillingGroupChannel{
			{Id: 1, BillingGroupRouteId: 17, ChannelId: 38, Priority: 100, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 0.6},
			{Id: 2, BillingGroupRouteId: 17, ChannelId: 40, Priority: 100, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 1.1},
		},
		ErrorMappings: []UpstreamErrorMapping{{
			Id: 1, ChannelId: 38, RawCode: " RATE_LIMIT_ERROR ", StatusCode: 429,
			AlltokenCode: 204001, Category: "rate_limit", FailureScope: "channel", Action: "switch_channel", Retryable: true, Enabled: true,
		}},
	}

	require.NoError(t, SaveChannelRoutingConfig(config))
	InitChannelRoutingCache()
	policy, channels, ok := ResolveBillingGroupRoute("claude")
	require.True(t, ok)
	assert.Equal(t, RoutingModeStabilityFirst, policy.Mode)
	assert.Equal(t, ProfitGuardModeWarn, policy.ProfitGuardMode)
	assert.InDelta(t, 12.5, policy.MinimumProfitMargin, 0.0001)
	assert.Equal(t, 2, policy.MaxTotalAttempts)
	require.Len(t, channels, 2)
	assert.Equal(t, 38, channels[0].ChannelId)
	assert.Equal(t, 40, channels[1].ChannelId)
	assert.Equal(t, 2, channels[0].Priority)
	assert.Equal(t, 1, channels[1].Priority)
	assert.Zero(t, channels[0].Weight)
	assert.Zero(t, channels[1].Weight)
	assert.InDelta(t, 0.6, ResolveChannelCostFactor("claude", 38), 0.0001)

	var oldCount int64
	require.NoError(t, DB.Model(&BillingGroupRoute{}).Where("id = ?", 9).Count(&oldCount).Error)
	assert.Zero(t, oldCount)
}

func TestSaveBillingGroupRoutePreservesExplicitTotalAttemptBudget(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 38, Name: "Claude primary", Group: "claude"},
		{Id: 40, Name: "Claude fallback", Group: "claude"},
	}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{
		Id: 9, BillingGroup: "claude", Name: "Claude", Enabled: true,
		MaxTotalAttempts: 2, RetryPolicy: `{"rate_limit_action":"switch_channel","upstream_action":"retry_channel"}`,
	}).Error)

	config := &BillingGroupRouteConfig{
		Route: BillingGroupRoute{
			Id: 9, BillingGroup: "claude", Name: "Claude", Enabled: true,
			MaxTotalAttempts: 2, RetryPolicy: `{"rate_limit_action":"switch_channel","upstream_action":"retry_channel"}`,
		},
		RouteChannels: []BillingGroupChannel{
			{BillingGroupRouteId: 9, ChannelId: 38, Priority: 2, MaxAttempts: 3, Enabled: true, CostFactor: 1},
			{BillingGroupRouteId: 9, ChannelId: 40, Priority: 1, MaxAttempts: 3, Enabled: true, CostFactor: 1},
		},
	}
	require.NoError(t, SaveBillingGroupRouteConfig(config))

	InitChannelRoutingCache()
	policy, _, ok := ResolveBillingGroupRoute("claude")
	require.True(t, ok)
	assert.Equal(t, 2, policy.MaxTotalAttempts)
	assert.Equal(t, "retry_channel", policy.RetryAction("upstream", 502, "switch_channel"))
	assert.Equal(t, "retry_channel", policy.RetryAction("network", 504, "switch_channel"))
}

func TestSaveUpstreamErrorMappingsPreservesRoutesAndBindings(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Claude", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{
		Id: 9, BillingGroup: "claude", Name: "Claude", Enabled: true, MaxTotalAttempts: 7,
	}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 2, Enabled: true, CostFactor: 1,
	}).Error)
	require.NoError(t, DB.Create(&[]UpstreamErrorMapping{
		{Id: 1, RawCode: "rate_limit", StatusCode: 429, AlltokenCode: 204001, Category: "rate_limit", FailureScope: "channel", Action: "switch_channel", Retryable: true, Enabled: true},
		{Id: 2, RawCode: "overloaded", StatusCode: 503, AlltokenCode: 205004, Category: "upstream", FailureScope: "provider", Action: "retry_later", Retryable: true, Enabled: true},
	}).Error)

	mappings := []UpstreamErrorMapping{{
		Id: 1, RawCode: " RATE_LIMIT ", StatusCode: 429, AlltokenCode: 204001,
		Category: "rate_limit", FailureScope: "channel", Action: "switch_channel", Retryable: true, Enabled: true,
	}}
	require.NoError(t, SaveUpstreamErrorMappings(mappings))

	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.Equal(t, 7, route.MaxTotalAttempts)
	var binding BillingGroupChannel
	require.NoError(t, DB.First(&binding, 1).Error)
	assert.Equal(t, 2, binding.MaxAttempts)
	var savedMapping UpstreamErrorMapping
	require.NoError(t, DB.First(&savedMapping, 1).Error)
	assert.Equal(t, "rate_limit", savedMapping.RawCode)
	assert.ErrorIs(t, DB.First(&UpstreamErrorMapping{}, 2).Error, gorm.ErrRecordNotFound)
}

func TestSaveChannelRoutingConfigRejectsInvalidMinimumProfitMargin(t *testing.T) {
	setupChannelRoutingTables(t)

	err := SaveChannelRoutingConfig(&ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: -10, BillingGroup: "claude", MinimumProfitMargin: 100,
		}},
	})

	require.EqualError(t, err, "minimum_profit_margin must be between 0 and 100")
}

func TestGetChannelRoutingConfigNormalizesLegacyProfitGuardMode(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&BillingGroupRoute{
		Id: 18, BillingGroup: "legacy", ProfitGuardMode: "",
	}).Error)

	config, err := GetChannelRoutingConfig()
	require.NoError(t, err)
	require.Len(t, config.Routes, 1)
	assert.Equal(t, ProfitGuardModeOff, config.Routes[0].ProfitGuardMode)
	assert.Equal(t, GetChannelCircuitConfig().Default, config.CircuitDefaults)
	assert.Equal(t, GetChannelCircuitConfig().Presets, config.CircuitPresets)
}

func TestChannelCircuitConfigControlsRoutingDefaultsAndPresets(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previous := common.OptionMap[ChannelCircuitConfigOptionKey]
	common.OptionMap[ChannelCircuitConfigOptionKey] = `{
		"default":{"failure_threshold":11,"window_seconds":75,"cooldown_seconds":95,"half_open_requests":4},
		"modes":{
			"cost_first":{"failure_threshold":12,"window_seconds":76,"cooldown_seconds":96,"half_open_requests":5},
			"stability_first":{"failure_threshold":13,"window_seconds":77,"cooldown_seconds":97,"half_open_requests":6}
		},
		"presets":[{"key":"custom","label":"Custom","failure_threshold":14,"window_seconds":78,"cooldown_seconds":98,"half_open_requests":7}]
	}`
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap[ChannelCircuitConfigOptionKey] = previous
		common.OptionMapRWMutex.Unlock()
	})

	balanced := DefaultRuntimeRoutingPolicy(RoutingModeBalanced)
	assert.Equal(t, 4, balanced.MaxTotalAttempts)
	assert.Equal(t, 11, balanced.CircuitFailureThreshold)
	costFirst := DefaultRuntimeRoutingPolicy(RoutingModeCostFirst)
	assert.Equal(t, 6, costFirst.MaxTotalAttempts)
	assert.Equal(t, 12, costFirst.CircuitFailureThreshold)
	stabilityFirst := DefaultRuntimeRoutingPolicy(RoutingModeStabilityFirst)
	assert.Equal(t, 3, stabilityFirst.MaxTotalAttempts)
	assert.Equal(t, 97, stabilityFirst.CircuitCooldownSeconds)
	assert.Equal(t, "custom", GetChannelCircuitConfig().Presets[0].Key)
}

func TestNormalizeChannelCircuitConfigRejectsIncompleteOrOutOfRangeValues(t *testing.T) {
	_, err := NormalizeChannelCircuitConfigJSONString(`{"default":{}}`)
	require.EqualError(t, err, "ChannelCircuitConfig contains missing or out-of-range values")

	_, err = NormalizeChannelCircuitConfigJSONString(`not-json`)
	require.EqualError(t, err, "ChannelCircuitConfig must be valid JSON")
}

func TestSaveChannelRoutingConfigRemapsTemporaryIDs(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Group: "claude"}).Error)

	config := &ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: -10, BillingGroup: "claude", Name: "Claude", Mode: RoutingModeBalanced, Enabled: true,
		}},
		RouteChannels: []BillingGroupChannel{{
			Id: -20, BillingGroupRouteId: -10, ChannelId: 38, Priority: 100,
			Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 0.6,
		}},
	}

	require.NoError(t, SaveChannelRoutingConfig(config))
	require.Positive(t, config.Routes[0].Id)
	require.Positive(t, config.RouteChannels[0].Id)
	assert.Equal(t, config.Routes[0].Id, config.RouteChannels[0].BillingGroupRouteId)
}

func TestSaveChannelRoutingConfigPreservesWeightsForWeightedStrategy(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Group: "claude"}).Error)

	config := &ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: -10, BillingGroup: "claude", Enabled: true,
			StrategyConfig: `{"type":"weighted"}`,
		}},
		RouteChannels: []BillingGroupChannel{{
			Id: -20, BillingGroupRouteId: -10, ChannelId: 38,
			Priority: 1, Weight: 73, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	}

	require.NoError(t, SaveChannelRoutingConfig(config))
	var saved BillingGroupChannel
	require.NoError(t, DB.First(&saved, config.RouteChannels[0].Id).Error)
	assert.Equal(t, 73, saved.Weight)
	InitChannelRoutingCache()
	policy, _, ok := ResolveBillingGroupRoute("claude")
	assert.True(t, ok)
	assert.Equal(t, RoutingStrategyWeighted, policy.Strategy)
	assert.InDelta(t, 40, policy.StrategyConfig.PriceWeight, 0.0001)
	assert.InDelta(t, 40, policy.StrategyConfig.AvailabilityWeight, 0.0001)
	assert.InDelta(t, 20, policy.StrategyConfig.LoadWeight, 0.0001)
}

func TestWeightedStrategyDefaultsAndNormalizesDynamicWeights(t *testing.T) {
	defaults := parseRoutingStrategyConfig(`{"type":"weighted"}`)
	assert.Equal(t, RoutingStrategyWeighted, defaults.Type)
	assert.InDelta(t, 40, defaults.PriceWeight, 0.0001)
	assert.InDelta(t, 40, defaults.AvailabilityWeight, 0.0001)
	assert.InDelta(t, 20, defaults.LoadWeight, 0.0001)

	normalized := parseRoutingStrategyConfig(`{"type":"weighted","price_weight":2,"availability_weight":1,"load_weight":1}`)
	assert.InDelta(t, 50, normalized.PriceWeight, 0.0001)
	assert.InDelta(t, 25, normalized.AvailabilityWeight, 0.0001)
	assert.InDelta(t, 25, normalized.LoadWeight, 0.0001)
}

func TestGetBillingGroupTypesUsesRouteMembershipIncludingDisabledRoutes(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&[]BillingGroupRoute{
		{Id: 1, BillingGroup: "enterprise", Enabled: true},
		{Id: 2, BillingGroup: "internal", Enabled: false},
	}).Error)
	InitChannelRoutingCache()

	assert.Equal(t, map[string]string{
		"default":    BillingGroupTypeToC,
		"enterprise": BillingGroupTypeToB,
		"internal":   BillingGroupTypeToB,
	}, GetBillingGroupTypes(map[string]float64{
		"default": 1, "enterprise": 1.2, "internal": 0.8,
	}))
}

func TestUpdateBillingGroupTypeChangesOnlyGroupType(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&BillingGroupRoute{
		Id: 9, BillingGroup: "claude", Name: "Claude", Mode: RoutingModeStabilityFirst,
		GroupType: BillingGroupTypeToB, Enabled: true, MaxTotalAttempts: 7,
	}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Enabled: true,
	}).Error)

	require.NoError(t, UpdateBillingGroupType(" claude ", BillingGroupTypeToC))

	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.Equal(t, BillingGroupTypeToC, route.GroupType)
	assert.Equal(t, "Claude", route.Name)
	assert.Equal(t, RoutingModeStabilityFirst, route.Mode)
	assert.True(t, route.Enabled)
	assert.Equal(t, 7, route.MaxTotalAttempts)
	var channelCount int64
	require.NoError(t, DB.Model(&BillingGroupChannel{}).Where("billing_group_route_id = ?", 9).Count(&channelCount).Error)
	assert.Equal(t, int64(1), channelCount)
}

func TestUpdateBillingGroupTypeCreatesDisabledRouteForPricingGroup(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, UpdateBillingGroupType("enterprise", BillingGroupTypeToB))

	var route BillingGroupRoute
	require.NoError(t, DB.Where("billing_group = ?", "enterprise").First(&route).Error)
	assert.Equal(t, BillingGroupTypeToB, route.GroupType)
	assert.Equal(t, "enterprise", route.Name)
	assert.False(t, route.Enabled)
	assert.Equal(t, RoutingModeBalanced, route.Mode)
	assert.NotEmpty(t, route.StrategyConfig)
}

func TestUpdateBillingGroupTypeRejectsInvalidType(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", GroupType: BillingGroupTypeToB}).Error)

	err := UpdateBillingGroupType("claude", "invalid")
	require.EqualError(t, err, "group_type must be toB or toC")

	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.Equal(t, BillingGroupTypeToB, route.GroupType)
}

func TestSaveChannelRoutingConfigRejectsChannelOutsideBillingGroup(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Group: "default"}).Error)

	err := SaveChannelRoutingConfig(&ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{Id: -10, BillingGroup: "claude", Enabled: true}},
		RouteChannels: []BillingGroupChannel{{
			Id: -20, BillingGroupRouteId: -10, ChannelId: 38, Priority: 100,
			Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	})

	require.EqualError(t, err, "route channel does not belong to its billing group")
}

func TestSaveChannelRoutingConfigRejectsEnabledRouteWithoutChannel(t *testing.T) {
	setupChannelRoutingTables(t)

	err := SaveChannelRoutingConfig(&ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: -10, BillingGroup: "claude", Enabled: true,
		}},
	})

	require.EqualError(t, err, "enabled billing group route requires an enabled channel")
}

func TestSaveBillingGroupRouteConfigCleansStalePersistedBinding(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Moved", Group: "other"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
	}).Error)

	config := &BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	}

	require.NoError(t, SaveBillingGroupRouteConfig(config))
	assert.Empty(t, config.RouteChannels)
	var count int64
	require.NoError(t, DB.Model(&BillingGroupChannel{}).Where("billing_group_route_id = ?", 9).Count(&count).Error)
	assert.Zero(t, count)
	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.False(t, route.Enabled)
}

func TestSaveBillingGroupRouteConfigCleansStaleBindingWithLegacyInvalidValues(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Moved", Group: "other"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: false}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 0, Enabled: false, CostFactor: 0,
	}).Error)

	config := &BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: false},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 0, Enabled: false, CostFactor: 0,
		}},
	}
	require.NoError(t, SaveBillingGroupRouteConfig(config))
	assert.Empty(t, config.RouteChannels)
}

func TestSaveBillingGroupRouteConfigPreservesValidBinding(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Claude", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
	}).Error)

	config := &BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 2, Enabled: true, CostFactor: 1,
		}},
	}

	require.NoError(t, SaveBillingGroupRouteConfig(config))
	require.Len(t, config.RouteChannels, 1)
	var entry BillingGroupChannel
	require.NoError(t, DB.First(&entry, 1).Error)
	assert.Equal(t, 2, entry.MaxAttempts)
}

func TestSaveBillingGroupRouteConfigRejectsMissingPersistedRoute(t *testing.T) {
	setupChannelRoutingTables(t)

	err := SaveBillingGroupRouteConfig(&BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 404, BillingGroup: "claude", Enabled: false},
	})

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestSaveBillingGroupRouteConfigRejectsNewCrossGroupBinding(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Other", Group: "other"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: false}).Error)

	err := SaveBillingGroupRouteConfig(&BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: false},
		RouteChannels: []BillingGroupChannel{{
			BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	})

	require.EqualError(t, err, `route channel does not belong to billing group "claude" (channel_id=38)`)
}

func TestSaveBillingGroupRouteConfigRejectsRouteRenameWithValidOldBinding(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Claude", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
	}).Error)

	err := SaveBillingGroupRouteConfig(&BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "other", Enabled: true},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	})

	require.EqualError(t, err, `route channel does not belong to billing group "other" (channel_id=38)`)
	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.Equal(t, "claude", route.BillingGroup)
}

func TestSaveBillingGroupRouteConfigPreservesBindingMovedToRenamedGroup(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Moved", Group: "other"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
	}).Error)

	config := &BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "other", Enabled: true},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	}
	require.NoError(t, SaveBillingGroupRouteConfig(config))
	require.Len(t, config.RouteChannels, 1)

	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.Equal(t, "other", route.BillingGroup)
	var entry BillingGroupChannel
	require.NoError(t, DB.First(&entry, 1).Error)
	assert.Equal(t, 38, entry.ChannelId)
}

func TestSaveBillingGroupRouteConfigNormalizesNegativeWeightForPriorityStrategy(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Claude", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)

	config := &BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true},
		RouteChannels: []BillingGroupChannel{{
			BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, Weight: -1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	}
	require.NoError(t, SaveBillingGroupRouteConfig(config))
	require.Len(t, config.RouteChannels, 1)
	assert.Zero(t, config.RouteChannels[0].Weight)
}

func TestSaveBillingGroupRouteConfigDisablesRouteAfterRemovingLastEnabledChannel(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Claude", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
	}).Error)

	config := &BillingGroupRouteConfig{Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}}
	require.NoError(t, SaveBillingGroupRouteConfig(config))
	assert.False(t, config.Route.Enabled)
	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.False(t, route.Enabled)
	var count int64
	require.NoError(t, DB.Model(&BillingGroupChannel{}).Where("billing_group_route_id = ?", 9).Count(&count).Error)
	assert.Zero(t, count)
}

func TestSaveBillingGroupRouteConfigDoesNotReplaceOtherRoutes(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 38, Name: "Claude", Group: "claude"},
		{Id: 39, Name: "Other", Group: "other"},
	}).Error)
	require.NoError(t, DB.Create(&[]BillingGroupRoute{
		{Id: 9, BillingGroup: "claude", Enabled: true},
		{Id: 10, BillingGroup: "other", Enabled: true},
	}).Error)
	require.NoError(t, DB.Create(&[]BillingGroupChannel{
		{Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1},
		{Id: 2, BillingGroupRouteId: 10, ChannelId: 39, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1},
	}).Error)

	require.NoError(t, SaveBillingGroupRouteConfig(&BillingGroupRouteConfig{
		Route: BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 2, Enabled: true, CostFactor: 1,
		}},
	}))

	var otherRoute BillingGroupRoute
	require.NoError(t, DB.First(&otherRoute, 10).Error)
	var otherEntry BillingGroupChannel
	require.NoError(t, DB.First(&otherEntry, 2).Error)
	assert.Equal(t, 10, otherEntry.BillingGroupRouteId)
	assert.Equal(t, 39, otherEntry.ChannelId)
}

func TestSaveChannelRoutingConfigCleansStalePersistedBinding(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Moved", Group: "other"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{
		Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
	}).Error)

	config := &ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{Id: 9, BillingGroup: "claude", Enabled: true}},
		RouteChannels: []BillingGroupChannel{{
			Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	}
	require.NoError(t, SaveChannelRoutingConfig(config))
	assert.Empty(t, config.RouteChannels)
	var count int64
	require.NoError(t, DB.Model(&BillingGroupChannel{}).Where("billing_group_route_id = ?", 9).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDeleteBillingGroupRouteAndCleanupRejectNegativeRouteID(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{Id: 1, BillingGroupRouteId: 9, ChannelId: 38}).Error)

	require.NoError(t, DeleteBillingGroupRoute(9))
	assert.ErrorIs(t, DB.First(&BillingGroupRoute{}, 9).Error, gorm.ErrRecordNotFound)
	assert.ErrorIs(t, DB.First(&BillingGroupChannel{}, 1).Error, gorm.ErrRecordNotFound)
	_, err := CleanupStaleBillingGroupRoutes(-1)
	require.EqualError(t, err, "billing group route id must be non-negative")
}

func TestCleanupStaleBillingGroupRoutesRemovesOrphansAndDisablesEmptyRoutes(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Moved", Group: "other"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&[]BillingGroupChannel{
		{Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1},
		{Id: 2, BillingGroupRouteId: 999, ChannelId: 404, Priority: 1, MaxAttempts: 1, Enabled: true, CostFactor: 1},
	}).Error)

	result, err := CleanupStaleBillingGroupRoutes(0)
	require.NoError(t, err)
	assert.Equal(t, 2, result.RemovedRouteChannels)
	assert.Equal(t, 1, result.DisabledRoutes)

	var entryCount int64
	require.NoError(t, DB.Model(&BillingGroupChannel{}).Count(&entryCount).Error)
	assert.Zero(t, entryCount)
	var route BillingGroupRoute
	require.NoError(t, DB.First(&route, 9).Error)
	assert.False(t, route.Enabled)
}

func TestCleanupStaleBillingGroupRoutesRemovesNullRouteBinding(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Exec(`
		INSERT INTO billing_group_channels
			(id, billing_group_route_id, channel_id, priority, weight, max_attempts, enabled, cost_factor)
		VALUES (?, NULL, ?, ?, ?, ?, ?, ?)
	`, 1, 38, 1, 0, 1, true, 1).Error)

	result, err := CleanupStaleBillingGroupRoutes(0)
	require.NoError(t, err)
	assert.Equal(t, 1, result.RemovedRouteChannels)

	var count int64
	require.NoError(t, DB.Model(&BillingGroupChannel{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDeleteBoundChannelRequiresRemovingItFromRouting(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Key: "key", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Enabled: true}).Error)

	err := (&Channel{Id: 38}).Delete()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by billing-group routing")

	require.NoError(t, DB.Delete(&BillingGroupChannel{}, 1).Error)
	require.NoError(t, (&Channel{Id: 38}).Delete())
	require.ErrorIs(t, DB.First(&Channel{}, 38).Error, gorm.ErrRecordNotFound)
}

func TestMatchUpstreamErrorMappingPrefersExactChannel(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&[]UpstreamErrorMapping{
		{Id: 1, RawCode: "*", StatusCode: 503, AlltokenCode: 205002, Category: "upstream", FailureScope: "provider", Action: "switch_channel", Retryable: true, Enabled: true},
		{Id: 2, ChannelId: 38, ChannelType: 14, RawCode: "overloaded_error", StatusCode: 503, AlltokenCode: 205004, Category: "upstream", FailureScope: "channel", Action: "switch_channel", Retryable: true, Enabled: true},
	}).Error)
	InitChannelRoutingCache()

	mapping, ok := MatchUpstreamErrorMapping(38, 14, "OVERLOADED_ERROR", 503)
	require.True(t, ok)
	assert.Equal(t, 205004, mapping.AlltokenCode)
	assert.Equal(t, "channel", mapping.FailureScope)
}
