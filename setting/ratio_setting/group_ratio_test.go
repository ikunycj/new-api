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

func TestCheckPricingGroupOrderRejectsInvalidGroups(t *testing.T) {
	previousRatios := GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(previousRatios))
	})
	require.NoError(t, UpdateGroupRatioByJSONString(`{"alpha":1,"beta":1}`))

	require.NoError(t, CheckPricingGroupOrder(`["beta","alpha"]`))
	require.Error(t, CheckPricingGroupOrder(`["alpha","alpha"]`))
	require.Error(t, CheckPricingGroupOrder(`["unknown"]`))
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
	require.NoError(t, CheckPricingGroupRetryPolicy(value))
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
		`{"unknown":{"mode":"fixed","retry_times":1}}`,
	} {
		require.Error(t, CheckPricingGroupRetryPolicy(invalid), invalid)
	}
}
