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
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupOrder2JSONString(), &configuration.GroupOrder))
	require.NoError(t, common.UnmarshalJsonStr(ratio_setting.PricingGroupRetryPolicy2JSONString(), &configuration.RetryPolicies))

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
		"UserUsableGroups",
		"group_ratio_setting.group_ratio",
		"group_ratio_setting.pricing_group_retry_policy",
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
		`["beta","alpha"]`,
		`{
			"alpha":{"mode":"fixed","retry_times":2},
			"beta":{"mode":"active_channels","retry_times":99}
		}`,
	))

	_, alphaExists := ratio_setting.GetPricingGroupRetryPolicy("alpha")
	_, betaExists := ratio_setting.GetPricingGroupRetryPolicy("beta")
	assert.True(t, alphaExists)
	assert.True(t, betaExists)
	assert.Equal(t, []string{"beta", "alpha"}, ratio_setting.GetPricingGroupOrder())
	assert.Equal(t, float64(2), ratio_setting.GetGroupRatio("beta"))

	var stored []Option
	require.NoError(t, DB.Where(commonKeyCol+" IN ?", pricingGroupOptionKeys).Find(&stored).Error)
	require.Len(t, stored, 3)
	var retryPolicyValue string
	for _, option := range stored {
		if option.Key == "PricingGroupRetryPolicy" {
			retryPolicyValue = option.Value
		}
	}
	var persisted map[string]ratio_setting.PricingGroupRetryPolicy
	require.NoError(t, common.UnmarshalJsonStr(retryPolicyValue, &persisted))
	assert.Equal(t, ratio_setting.PricingGroupRetryPolicy{
		Mode:       ratio_setting.PricingGroupRetryModeActiveChannels,
		RetryTimes: 0,
	}, persisted["beta"])
}

func TestLoadOptionsPublishesCompletePricingGroupConfiguration(t *testing.T) {
	truncateTables(t)
	preservePricingGroupConfiguration(t)

	require.NoError(t, DB.Create(&[]Option{
		{Key: "GroupRatio", Value: `{"alpha":1,"beta":2}`},
		{Key: "PricingGroupOrder", Value: `["beta","alpha"]`},
		{Key: "PricingGroupRetryPolicy", Value: `{"alpha":{"mode":"fixed","retry_times":4}}`},
	}).Error)

	loadOptionsFromDatabase()

	assert.Equal(t, []string{"beta", "alpha"}, ratio_setting.GetPricingGroupOrder())
	assert.Equal(t, float64(2), ratio_setting.GetGroupRatio("beta"))
	policy, exists := ratio_setting.GetPricingGroupRetryPolicy("beta")
	require.True(t, exists)
	assert.Equal(t, ratio_setting.PricingGroupRetryPolicy{
		Mode: ratio_setting.PricingGroupRetryModeActiveChannels,
	}, policy)

	common.OptionMapRWMutex.RLock()
	storedPolicies := common.OptionMap["PricingGroupRetryPolicy"]
	common.OptionMapRWMutex.RUnlock()
	var policies map[string]ratio_setting.PricingGroupRetryPolicy
	require.NoError(t, common.UnmarshalJsonStr(storedPolicies, &policies))
	assert.Equal(t, policy, policies["beta"])
}
