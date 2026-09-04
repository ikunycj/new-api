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
			ProfitGuardMode: ProfitGuardModeWarn, MinimumProfitMargin: 12.5,
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
