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
		`["beta","alpha"]`,
		`{
			"alpha":{"mode":"fixed","retry_times":4},
			"beta":{"mode":"active_channels","retry_times":99}
		}`,
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"beta", "alpha"}, configuration.GroupOrder)
	assert.Equal(t, PricingGroupRetryPolicy{
		Mode:       PricingGroupRetryModeActiveChannels,
		RetryTimes: 0,
	}, configuration.RetryPolicies["beta"])

	for _, testCase := range []struct {
		name     string
		ratios   string
		order    string
		policies string
	}{
		{
			name:     "empty group name",
			ratios:   `{"":1}`,
			order:    `[""]`,
			policies: `{"":{"mode":"fixed","retry_times":1}}`,
		},
		{
			name:     "missing order entry",
			ratios:   `{"alpha":1,"beta":1}`,
			order:    `["alpha"]`,
			policies: `{"alpha":{"mode":"fixed","retry_times":1},"beta":{"mode":"fixed","retry_times":1}}`,
		},
		{
			name:     "missing retry policy",
			ratios:   `{"alpha":1,"beta":1}`,
			order:    `["alpha","beta"]`,
			policies: `{"alpha":{"mode":"fixed","retry_times":1}}`,
		},
		{
			name:     "unknown retry policy group",
			ratios:   `{"alpha":1}`,
			order:    `["alpha"]`,
			policies: `{"unknown":{"mode":"fixed","retry_times":1}}`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParsePricingGroupConfiguration(testCase.ratios, testCase.order, testCase.policies)
			require.Error(t, err)
		})
	}
}

func TestParsePersistedPricingGroupConfigurationBuildsCompleteSnapshot(t *testing.T) {
	configuration, err := ParsePersistedPricingGroupConfiguration(
		`{"alpha":1,"beta":2}`,
		`["retired","beta"]`,
		`{
			"alpha":{"mode":"fixed","retry_times":4},
			"retired":{"mode":"fixed","retry_times":9}
		}`,
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"beta", "alpha"}, configuration.GroupOrder)
	assert.Equal(t, map[string]PricingGroupRetryPolicy{
		"alpha": {Mode: PricingGroupRetryModeFixed, RetryTimes: 4},
		"beta":  {Mode: PricingGroupRetryModeActiveChannels},
	}, configuration.RetryPolicies)
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
