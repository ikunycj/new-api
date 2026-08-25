package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPricingGroupNamesUsesConfiguredGroupsOnly(t *testing.T) {
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	previousGroupOrder := ratio_setting.PricingGroupOrder2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(previousGroupOrder))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"configured-ratio":0.5,"shared":1,"auto":0.8}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(`["shared","auto","configured-ratio"]`))

	groups, err := GetPricingGroupNames()
	require.NoError(t, err)
	assert.Equal(t, []string{"shared", "configured-ratio"}, groups)
	assert.NotContains(t, groups, "auto")
}
