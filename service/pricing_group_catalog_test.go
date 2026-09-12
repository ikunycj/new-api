package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisiblePricingGroupsIgnoreRuntimeEnabledState(t *testing.T) {
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	previousOrder := ratio_setting.PricingGroupOrder2JSONString()
	previousPermissions := setting.UserGroupPricingGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(previousEnabled))
		require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(previousOrder))
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPermissions))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"disabled":0.8,"auto":0.5}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"default":true,"disabled":false,"auto":true}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(`["disabled","auto","default"]`))
	require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(`{"default":["default","disabled","auto"]}`))

	visible := GetUserGroupVisiblePricingGroups("default")
	assert.Equal(t, map[string]string{"default": "default", "disabled": "disabled"}, visible)

	usable := GetUserGroupPricingGroups("default")
	assert.Contains(t, usable, "default")
	assert.NotContains(t, usable, "disabled")
}
