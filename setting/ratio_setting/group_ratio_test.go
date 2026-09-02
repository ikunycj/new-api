package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingGroupOrderKeepsConfiguredOrderAndAppendsMissingGroups(t *testing.T) {
	previousRatios := GroupRatio2JSONString()
	previousOrder := PricingGroupOrder2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, UpdatePricingGroupOrderByJSONString(previousOrder))
	})

	require.NoError(t, UpdateGroupRatioByJSONString(`{"alpha":1,"beta":1,"gamma":1}`))
	require.NoError(t, UpdatePricingGroupOrderByJSONString(`["gamma","alpha","removed"]`))

	assert.Equal(t, []string{"gamma", "alpha", "beta"}, GetPricingGroupOrder())
}

func TestPricingGroupRetryPolicyValidationAndNormalization(t *testing.T) {
	previousRatios := GroupRatio2JSONString()
	previousPolicies := PricingGroupRetryPolicy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
	})
	require.NoError(t, UpdateGroupRatioByJSONString(`{"alpha":1,"beta":1}`))

	value := `{
		"alpha":{"mode":"fixed","retry_times":5},
		"beta":{"mode":"active_channels","retry_times":99}
	}`
	require.NoError(t, UpdatePricingGroupRetryPolicyByJSONString(value))

	alpha, exists := GetPricingGroupRetryPolicy("alpha")
	require.True(t, exists)
	assert.Equal(t, PricingGroupRetryPolicy{Mode: PricingGroupRetryModeFixed, RetryTimes: 5}, alpha)
	beta, exists := GetPricingGroupRetryPolicy("beta")
	require.True(t, exists)
	assert.Equal(t, PricingGroupRetryPolicy{Mode: PricingGroupRetryModeActiveChannels, RetryTimes: 0}, beta)

	for _, invalid := range []string{
		`{"alpha":{"mode":"fixed","retry_times":-1}}`,
		`{"alpha":{"mode":"fixed","retry_times":101}}`,
		`{"alpha":{"mode":"unknown","retry_times":1}}`,
	} {
		require.Error(t, UpdatePricingGroupRetryPolicyByJSONString(invalid), invalid)
	}
}

func TestParsePricingGroupConfigurationRequiresMatchingGroups(t *testing.T) {
	configuration, err := ParsePricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		`{"alpha":true,"beta":false}`,
		`["beta","alpha"]`,
		`{
			"alpha":{"mode":"fixed","retry_times":4},
			"beta":{"mode":"active_channels","retry_times":99}
			}`,
		`{}`,
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"beta", "alpha"}, configuration.GroupOrder)
	assert.Equal(t, map[string]bool{"alpha": true, "beta": false}, configuration.GroupEnabled)
	assert.Equal(t, PricingGroupRetryPolicy{
		Mode:       PricingGroupRetryModeActiveChannels,
		RetryTimes: 0,
	}, configuration.RetryPolicies["beta"])

	for _, testCase := range []struct {
		name       string
		ratios     string
		enabled    string
		order      string
		policies   string
		strategies string
	}{
		{
			name:       "empty group name",
			ratios:     `{"":1}`,
			enabled:    `{"":true}`,
			order:      `[""]`,
			policies:   `{"":{"mode":"fixed","retry_times":1}}`,
			strategies: `{}`,
		},
		{
			name:       "missing order entry",
			ratios:     `{"alpha":1,"beta":1}`,
			enabled:    `{"alpha":true,"beta":true}`,
			order:      `["alpha"]`,
			policies:   `{"alpha":{"mode":"fixed","retry_times":1},"beta":{"mode":"fixed","retry_times":1}}`,
			strategies: `{}`,
		},
		{
			name:       "missing retry policy",
			ratios:     `{"alpha":1,"beta":1}`,
			enabled:    `{"alpha":true,"beta":true}`,
			order:      `["alpha","beta"]`,
			policies:   `{"alpha":{"mode":"fixed","retry_times":1}}`,
			strategies: `{}`,
		},
		{
			name:       "unknown retry policy group",
			ratios:     `{"alpha":1}`,
			enabled:    `{"alpha":true}`,
			order:      `["alpha"]`,
			policies:   `{"unknown":{"mode":"fixed","retry_times":1}}`,
			strategies: `{}`,
		},
		{
			name:       "missing enabled entry",
			ratios:     `{"alpha":1,"beta":1}`,
			enabled:    `{"alpha":true}`,
			order:      `["alpha","beta"]`,
			policies:   `{"alpha":{"mode":"fixed","retry_times":1},"beta":{"mode":"fixed","retry_times":1}}`,
			strategies: `{}`,
		},
		{
			name:       "unknown enabled group",
			ratios:     `{"alpha":1}`,
			enabled:    `{"unknown":true}`,
			order:      `["alpha"]`,
			policies:   `{"alpha":{"mode":"fixed","retry_times":1}}`,
			strategies: `{}`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParsePricingGroupConfiguration(testCase.ratios, testCase.enabled, testCase.order, testCase.policies, testCase.strategies)
			require.Error(t, err)
		})
	}
}

func TestParsePersistedPricingGroupConfigurationBuildsCompleteSnapshot(t *testing.T) {
	configuration, err := ParsePersistedPricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		"",
		`["retired","beta"]`,
		`{"alpha":{"mode":"fixed","retry_times":4},"beta":{"mode":"active_channels"}}`,
		`{
			"strategies":{"cheap":{"name":"低价","price_weight":65,"availability_weight":20,"load_weight":15}},
			"group_bindings":{"alpha":"cheap","beta":"cheap"}
		}`,
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"beta", "alpha"}, configuration.GroupOrder)
	assert.Equal(t, map[string]bool{"alpha": true, "beta": true}, configuration.GroupEnabled)
	assert.Equal(t, map[string]PricingGroupRetryPolicy{
		"alpha": {Mode: PricingGroupRetryModeFixed, RetryTimes: 4},
		"beta":  {Mode: PricingGroupRetryModeActiveChannels},
	}, configuration.RetryPolicies)
}

func TestParsePersistedPricingGroupConfigurationRejectsInvalidStoredEnabledMap(t *testing.T) {
	_, err := ParsePersistedPricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		`{"alpha":true}`,
		`["alpha","beta"]`,
		`{"alpha":{"mode":"fixed","retry_times":1},"beta":{"mode":"fixed","retry_times":1}}`,
		`{}`,
	)
	require.ErrorContains(t, err, "启用状态必须覆盖全部定价分组")
}

func TestPricingGroupRetryPolicyDefaultsToActiveChannels(t *testing.T) {
	previousRatios := GroupRatio2JSONString()
	previousPolicies := PricingGroupRetryPolicy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
	})
	require.NoError(t, UpdateGroupRatioByJSONString(`{"alpha":1}`))
	require.NoError(t, UpdatePricingGroupRetryPolicyByJSONString(`{}`))

	policy, exists := GetPricingGroupRetryPolicy("alpha")
	require.True(t, exists)
	assert.Equal(t, PricingGroupRetryPolicy{
		Mode: PricingGroupRetryModeActiveChannels,
	}, policy)

	_, exists = GetPricingGroupRetryPolicy("unknown")
	assert.False(t, exists)
}

func TestPricingGroupRoutingStrategyPresetsAndDefaults(t *testing.T) {
	assert.Equal(t, PricingGroupRoutingStrategy{
		Strategy:           PricingGroupRoutingStrategyPriceFirst,
		PriceWeight:        65,
		AvailabilityWeight: 20,
		LoadWeight:         15,
		TTFTWeight:         0,
	}, PricingGroupRoutingStrategyPreset(PricingGroupRoutingStrategyPriceFirst))
	assert.Equal(t, DefaultPricingGroupRoutingStrategy(), PricingGroupRoutingStrategyPreset("unknown"))

	previousRatios := GroupRatio2JSONString()
	previousOrder := PricingGroupOrder2JSONString()
	previousPolicies := PricingGroupRetryPolicy2JSONString()
	previousStrategies := PricingGroupRoutingStrategy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, UpdatePricingGroupOrderByJSONString(previousOrder))
		require.NoError(t, UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
		require.NoError(t, UpdatePricingGroupRoutingStrategyByJSONString(previousStrategies))
	})

	require.NoError(t, UpdateGroupRatioByJSONString(`{"alpha":1}`))
	require.NoError(t, UpdatePricingGroupRoutingStrategyByJSONString(`{}`))
	strategy, exists := GetPricingGroupRoutingStrategy("alpha")
	require.True(t, exists)
	assert.Equal(t, DefaultPricingGroupRoutingStrategy(), strategy)
}

func TestPricingGroupRoutingConfigurationCRUDValidation(t *testing.T) {
	valid := `{
		"strategies":{
			"balanced":{"name":"均衡","price_weight":40,"availability_weight":40,"load_weight":20},
			"custom":{"name":"企业稳定","price_weight":20,"availability_weight":65,"load_weight":15}
		},
		"group_bindings":{"alpha":"custom","beta":"balanced"}
	}`
	configuration, err := ParsePricingGroupRoutingConfiguration(valid, map[string]float64{"alpha": 1, "beta": 1})
	require.NoError(t, err)
	assert.Equal(t, "custom", configuration.GroupBindings["alpha"])
	assert.Equal(t, "企业稳定", configuration.Strategies["custom"].Name)

	for _, value := range []string{
		`{"strategies":{"bad":{"name":"错误","price_weight":50,"availability_weight":20,"load_weight":20}},"group_bindings":{"alpha":"bad","beta":"bad"}}`,
		`{"strategies":{"one":{"name":"重复","price_weight":40,"availability_weight":40,"load_weight":20},"two":{"name":"重复","price_weight":20,"availability_weight":60,"load_weight":20}},"group_bindings":{"alpha":"one","beta":"two"}}`,
		`{"strategies":{"one":{"name":"存在","price_weight":40,"availability_weight":40,"load_weight":20}},"group_bindings":{"alpha":"missing","beta":"one"}}`,
		`{"strategies":{"one":{"name":"存在","price_weight":40,"availability_weight":40,"load_weight":20}},"group_bindings":{"alpha":"one"}}`,
		`{"alpha":{"strategy":"balanced","price_weight":40,"availability_weight":40,"load_weight":20}}`,
	} {
		_, err := ParsePricingGroupRoutingConfiguration(value, map[string]float64{"alpha": 1, "beta": 1})
		require.Error(t, err, value)
	}
}

func TestPricingGroupRoutingConfigurationSupportsTTFTWeight(t *testing.T) {
	configuration, err := ParsePricingGroupRoutingConfiguration(`{
		"strategies":{"latency":{"name":"低延迟","price_weight":20,"availability_weight":20,"load_weight":10,"ttft_weight":50}},
		"group_bindings":{"alpha":"latency"}
	}`, map[string]float64{"alpha": 1})
	require.NoError(t, err)
	assert.Equal(t, float64(50), configuration.Strategies["latency"].TTFTWeight)

	// Existing persisted configurations omit ttft_weight and retain their
	// original three-weight total.
	legacy, err := ParsePricingGroupRoutingConfiguration(`{
		"strategies":{"legacy":{"name":"旧策略","price_weight":40,"availability_weight":40,"load_weight":20}},
		"group_bindings":{"alpha":"legacy"}
	}`, map[string]float64{"alpha": 1})
	require.NoError(t, err)
	assert.Zero(t, legacy.Strategies["legacy"].TTFTWeight)

	_, err = ParsePricingGroupRoutingConfiguration(`{
		"strategies":{"invalid":{"name":"无效","price_weight":30,"availability_weight":30,"load_weight":20,"ttft_weight":10}},
		"group_bindings":{"alpha":"invalid"}
	}`, map[string]float64{"alpha": 1})
	require.Error(t, err)
}

func TestPricingGroupRoutingConfigurationRejectsDeletingReferencedStrategy(t *testing.T) {
	_, err := ParsePricingGroupRoutingConfiguration(`{
		"strategies":{"remaining":{"name":"保留","price_weight":40,"availability_weight":40,"load_weight":20}},
		"group_bindings":{"alpha":"deleted"}
	}`, map[string]float64{"alpha": 1})
	require.ErrorContains(t, err, "引用了不存在的策略")
}

func TestGetPricingGroupRoutingStrategyResolvesGroupBinding(t *testing.T) {
	previousRatios := GroupRatio2JSONString()
	previousStrategies := PricingGroupRoutingStrategy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, UpdatePricingGroupRoutingStrategyByJSONString(previousStrategies))
	})
	require.NoError(t, UpdateGroupRatioByJSONString(`{"alpha":1,"beta":1}`))
	require.NoError(t, UpdatePricingGroupRoutingStrategyByJSONString(`{
		"strategies":{"shared":{"name":"共享策略","price_weight":55,"availability_weight":30,"load_weight":15}},
		"group_bindings":{"alpha":"shared","beta":"shared"}
	}`))

	alpha, alphaExists := GetPricingGroupRoutingStrategy("alpha")
	beta, betaExists := GetPricingGroupRoutingStrategy("beta")
	require.True(t, alphaExists)
	require.True(t, betaExists)
	assert.Equal(t, "shared", alpha.Strategy)
	assert.Equal(t, alpha, beta)
}
