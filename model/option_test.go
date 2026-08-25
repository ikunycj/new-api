package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetiredOptionsAreRemovedAndCannotBeRestored(t *testing.T) {
	truncateTables(t)

	const retiredKey = "UserUsableGroups"
	activeOption := Option{Key: "WaffoReturnUrl", Value: "/wallet/return"}
	retiredOption := Option{Key: retiredKey, Value: `{}`}
	require.NoError(t, DB.Create(&activeOption).Error)
	require.NoError(t, DB.Create(&retiredOption).Error)

	require.NoError(t, removeRetiredOptions())

	var retiredCount int64
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" = ?", retiredKey).Count(&retiredCount).Error)
	assert.Zero(t, retiredCount)

	var activeCount int64
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" = ?", activeOption.Key).Count(&activeCount).Error)
	assert.EqualValues(t, 1, activeCount)

	require.ErrorIs(t, UpdateOption(retiredKey, `{}`), errRetiredOption)
	require.NoError(t, DB.Model(&Option{}).Where(commonKeyCol+" = ?", retiredKey).Count(&retiredCount).Error)
	assert.Zero(t, retiredCount)
}

func TestUpdateGroupRatioRemovesDeletedRetryPolicies(t *testing.T) {
	truncateTables(t)
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousPolicies := ratio_setting.PricingGroupRetryPolicy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(previousPolicies))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"alpha":1,"beta":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(`{
		"alpha":{"mode":"fixed","retry_times":2},
		"beta":{"mode":"active_channels","retry_times":0}
	}`))
	require.NoError(t, UpdateOption("GroupRatio", `{"alpha":1}`))

	_, alphaExists := ratio_setting.GetPricingGroupRetryPolicy("alpha")
	_, betaExists := ratio_setting.GetPricingGroupRetryPolicy("beta")
	assert.True(t, alphaExists)
	assert.False(t, betaExists)

	var stored Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", "PricingGroupRetryPolicy").First(&stored).Error)
	var persisted map[string]ratio_setting.PricingGroupRetryPolicy
	require.NoError(t, common.UnmarshalJsonStr(stored.Value, &persisted))
	assert.Contains(t, persisted, "alpha")
	assert.NotContains(t, persisted, "beta")
}
