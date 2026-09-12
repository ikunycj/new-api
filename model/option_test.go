package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preservePricingGroupConfiguration(t *testing.T) {
	t.Helper()
	var configuration ratio_setting.PricingGroupConfiguration
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.GroupRatio2JSONString(), &configuration.GroupRatios))
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupEnabled2JSONString(), &configuration.GroupEnabled))
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupDisplayName2JSONString(), &configuration.GroupDisplayNames))
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupRemark2JSONString(), &configuration.GroupRemarks))
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupOrder2JSONString(), &configuration.GroupOrder))
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupRetryPolicy2JSONString(), &configuration.RetryPolicies))
	var routingConfiguration ratio_setting.PricingGroupRoutingConfiguration
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupRoutingStrategy2JSONString(), &routingConfiguration))
	configuration.RoutingStrategies = routingConfiguration.Strategies
	configuration.RoutingStrategyBindings = routingConfiguration.GroupBindings

	optionValues := make(map[string]string, len(pricingGroupOptionKeys))
	optionExists := make(map[string]bool, len(pricingGroupOptionKeys))
	common.OptionMapRWMutex.RLock()
	for _, key := range pricingGroupOptionKeys {
		optionValues[key], optionExists[key] = common.OptionMap[key]
	}
	common.OptionMapRWMutex.RUnlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if common.OptionMap == nil {
			common.OptionMap = make(map[string]string)
		}
		for _, key := range pricingGroupOptionKeys {
			if optionExists[key] {
				common.OptionMap[key] = optionValues[key]
			} else {
				delete(common.OptionMap, key)
			}
		}
		ratio_setting.ApplyPricingGroupConfiguration(&configuration)
		common.OptionMapRWMutex.Unlock()
	})
}

func TestRetiredOptionsAreRemovedAndCannotBeRestored(t *testing.T) {
	truncateTables(t)

	retiredKeys := []string{
		"AutomaticEnableChannelEnabled",
		"RetryTimes",
		"UserUsableGroups",
		"group_ratio_setting.group_ratio",
		"group_ratio_setting.pricing_group_retry_policy",
		"monitor_setting.auto_test_channel_enabled",
		"monitor_setting.auto_test_channel_minutes",
		"monitor_setting.channel_test_mode",
	}
	activeOption := Option{Key: "WaffoReturnUrl", Value: "/wallet/return"}
	require.NoError(t, DB.Create(&activeOption).Error)
	for _, key := range retiredKeys {
		require.NoError(t, DB.Create(&Option{Key: key, Value: `{}`}).Error)
	}

	require.NoError(t, removeRetiredOptions())

	var retiredCount int64
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" IN ?", retiredKeys).Count(&retiredCount).Error)
	assert.Zero(t, retiredCount)

	var activeCount int64
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" = ?", activeOption.Key).Count(&activeCount).Error)
	assert.EqualValues(t, 1, activeCount)

	for _, key := range retiredKeys {
		require.ErrorIs(t, UpdateOption(key, `{}`), errRetiredOption)
	}
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" IN ?", retiredKeys).Count(&retiredCount).Error)
	assert.Zero(t, retiredCount)
}

func TestUpdatePricingGroupConfigurationPersistsAllSettings(t *testing.T) {
	truncateTables(t)
	preservePricingGroupConfiguration(t)

	require.NoError(t, UpdatePricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		`{"alpha":true,"beta":false}`,
		`["beta","alpha"]`,
		`{
			"alpha":{"mode":"fixed","retry_times":2},
			"beta":{"mode":"active_channels","retry_times":99}
			}`,
		`{}`,
	))

	_, alphaExists := ratio_setting.GetPricingGroupRetryPolicy("alpha")
	_, betaExists := ratio_setting.GetPricingGroupRetryPolicy("beta")
	assert.True(t, alphaExists)
	assert.True(t, betaExists)
	assert.Equal(t, []string{"beta", "alpha"}, ratio_setting.GetPricingGroupOrder())
	assert.Equal(t, float64(2), ratio_setting.GetGroupRatio("beta"))
	assert.False(t, ratio_setting.IsPricingGroupEnabled("beta"))

	var stored []Option
	require.NoError(t, DB.Where(commonKeyCol+" IN ?", pricingGroupOptionKeys).Find(&stored).Error)
	require.Len(t, stored, 7)
	var retryPolicyValue string
	var enabledValue string
	for _, option := range stored {
		if option.Key == "PricingGroupRetryPolicy" {
			retryPolicyValue = option.Value
		}
		if option.Key == "PricingGroupEnabled" {
			enabledValue = option.Value
		}
	}
	assert.JSONEq(t, `{"alpha":true,"beta":false}`, enabledValue)
	var persisted map[string]ratio_setting.PricingGroupRetryPolicy
	require.NoError(t, common.UnmarshalJsonStr(retryPolicyValue, &persisted))
	assert.Equal(t, ratio_setting.PricingGroupRetryPolicy{
		Mode:       ratio_setting.PricingGroupRetryModeFollowChannels,
		RetryTimes: 0,
	}, persisted["beta"])
}

func TestUpdatePricingGroupConfigurationPersistsDisplayNames(t *testing.T) {
	truncateTables(t)
	preservePricingGroupConfiguration(t)

	require.NoError(t, UpdatePricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		`{"alpha":true,"beta":true}`,
		`["alpha","beta"]`,
		`{"alpha":{"mode":"active_channels"},"beta":{"mode":"active_channels"}}`,
		`{}`,
		`{}`,
		`{"alpha":"企业套餐","beta":"  "}`,
	))

	assert.Equal(t, "企业套餐", ratio_setting.GetPricingGroupDisplayName("alpha"))
	assert.Equal(t, "企业套餐", ratio_setting.GetPricingGroupDisplayNameOrName("alpha"))
	assert.Empty(t, ratio_setting.GetPricingGroupDisplayName("beta"))
	assert.Equal(t, "beta", ratio_setting.GetPricingGroupDisplayNameOrName("beta"))

	var stored Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", pricingGroupDisplayNameOptionKey).First(&stored).Error)
	assert.JSONEq(t, `{"alpha":"企业套餐"}`, stored.Value)
}

func TestUpdatePricingGroupConfigurationPersistsRoutingStrategies(t *testing.T) {
	truncateTables(t)
	previous := ratio_setting.PricingGroupRoutingStrategy2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdatePricingGroupRoutingStrategyByJSONString(previous)
	})

	require.NoError(t, UpdatePricingGroupConfiguration(
		`{"alpha":1}`,
		`{"alpha":true}`,
		`["alpha"]`,
		`{"alpha":{"mode":"active_channels","retry_times":0}}`,
		`{
			"strategies":{"enterprise":{"name":"企业策略","price_weight":45,"availability_weight":20,"load_weight":10,"ttft_weight":5,"recent_test_ttft_weight":20}},
			"group_bindings":{"alpha":"enterprise"}
		}`,
	))
	strategy, exists := ratio_setting.GetPricingGroupRoutingStrategy("alpha")
	require.True(t, exists)
	assert.Equal(t, "enterprise", strategy.Strategy)
	assert.Equal(t, float64(45), strategy.PriceWeight)
	assert.Equal(t, float64(20), strategy.AvailabilityWeight)
	assert.Equal(t, float64(10), strategy.LoadWeight)
	assert.Equal(t, float64(5), strategy.TTFTWeight)
	assert.Equal(t, float64(20), strategy.RecentTestTTFTWeight)

	var stored Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", "PricingGroupRoutingStrategy").First(&stored).Error)
	var persisted ratio_setting.PricingGroupRoutingConfiguration
	require.NoError(t, common.UnmarshalJsonStr(stored.Value, &persisted))
	persistedStrategy, exists := persisted.Strategies["enterprise"]
	require.True(t, exists)
	assert.Equal(t, float64(45), persistedStrategy.PriceWeight)
	assert.Equal(t, float64(20), persistedStrategy.AvailabilityWeight)
	assert.Equal(t, float64(10), persistedStrategy.LoadWeight)
	assert.Equal(t, float64(5), persistedStrategy.TTFTWeight)
	assert.Equal(t, float64(20), persistedStrategy.RecentTestTTFTWeight)
	assert.Equal(t, "enterprise", persisted.GroupBindings["alpha"])
}

func TestUpdatePricingGroupConfigurationPersistsRemarks(t *testing.T) {
	truncateTables(t)
	preservePricingGroupConfiguration(t)

	require.NoError(t, UpdatePricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		`{"alpha":true,"beta":true}`,
		`["alpha","beta"]`,
		`{"alpha":{"mode":"active_channels"},"beta":{"mode":"active_channels"}}`,
		`{}`,
		`{"alpha":"企业套餐","beta":"备用套餐"}`,
	))

	assert.Equal(t, "企业套餐", ratio_setting.GetPricingGroupRemark("alpha"))
	var stored Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", pricingGroupRemarkOptionKey).First(&stored).Error)
	assert.JSONEq(t, `{"alpha":"企业套餐","beta":"备用套餐"}`, stored.Value)
}

func TestLoadOptionsPublishesCompletePricingGroupConfiguration(t *testing.T) {
	truncateTables(t)
	preservePricingGroupConfiguration(t)

	require.NoError(t, DB.Create(&[]Option{
		{Key: "GroupRatio", Value: `{"alpha":1,"beta":2}`},
		{Key: "PricingGroupOrder", Value: `["beta","alpha"]`},
		{Key: "PricingGroupRetryPolicy", Value: `{"alpha":{"mode":"fixed","retry_times":4},"beta":{"mode":"active_channels"}}`},
	}).Error)

	loadOptionsFromDatabase()

	assert.Equal(t, []string{"beta", "alpha"}, ratio_setting.GetPricingGroupOrder())
	assert.Equal(t, float64(2), ratio_setting.GetGroupRatio("beta"))
	policy, exists := ratio_setting.GetPricingGroupRetryPolicy("beta")
	require.True(t, exists)
	assert.Equal(t, ratio_setting.PricingGroupRetryPolicy{
		Mode: ratio_setting.PricingGroupRetryModeFollowChannels,
	}, policy)

	common.OptionMapRWMutex.RLock()
	storedPolicies := common.OptionMap["PricingGroupRetryPolicy"]
	storedEnabled := common.OptionMap["PricingGroupEnabled"]
	common.OptionMapRWMutex.RUnlock()
	var policies map[string]ratio_setting.PricingGroupRetryPolicy
	require.NoError(t, common.UnmarshalJsonStr(storedPolicies, &policies))
	assert.Equal(t, policy, policies["beta"])
	assert.JSONEq(t, `{"alpha":true,"beta":true}`, storedEnabled)
	assert.True(t, ratio_setting.IsPricingGroupEnabled("alpha"))
	assert.True(t, ratio_setting.IsPricingGroupEnabled("beta"))
}
