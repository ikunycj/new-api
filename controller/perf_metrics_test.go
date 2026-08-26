package controller

import (
	"testing"

	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterActiveGroupsExcludesDisabledPricingGroups(t *testing.T) {
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(previousEnabled))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"active":1,"disabled":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"active":true,"disabled":false}`))

	filtered := filterActiveGroups([]perfmetrics.GroupResult{
		{Group: "active"},
		{Group: "disabled"},
		{Group: "retired"},
		{Group: "auto"},
	})
	assert.Equal(t, []perfmetrics.GroupResult{{Group: "active"}, {Group: "auto"}}, filtered)
}
