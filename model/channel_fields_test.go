package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelGetGroupsTrimsDropsEmptyAndDeduplicates(t *testing.T) {
	channel := &Channel{Group: " alpha, beta,alpha, , beta,gamma "}
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, channel.GetGroups())
	channel.NormalizeGroups()
	assert.Equal(t, "alpha,beta,gamma", channel.Group)
}

func TestChannelSelectionFieldDefaults(t *testing.T) {
	channel := &Channel{}
	assert.Equal(t, float64(1), channel.GetPriceMultiplier())
	assert.Equal(t, ChannelPriceMultiplierModeUSD, channel.GetPriceMultiplierMode())
	assert.Equal(t, ChannelForcePriorityScopeGroup, channel.GetForcePriorityScope())

	channel.PriceMultiplier = math.Inf(1)
	channel.PriceMultiplierMode = " CNY "
	channel.ForcePriorityScope = " CROSS_GROUP "
	assert.Equal(t, float64(1), channel.GetPriceMultiplier())
	assert.Equal(t, ChannelPriceMultiplierModeCNY, channel.GetPriceMultiplierMode())
	assert.Equal(t, ChannelForcePriorityScopeCrossGroup, channel.GetForcePriorityScope())
}

func TestChannelEstimateCostCNYUsesUpstreamCreditPurchaseMode(t *testing.T) {
	for _, tt := range []struct {
		name string
		mode string
		want float64
	}{
		{"five yuan buys five dollars of credit", ChannelPriceMultiplierModeUSD, 5},
		{"RMB credit costs the official RMB price times the multiplier", ChannelPriceMultiplierModeCNY, 35},
	} {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{PriceMultiplier: 0.2, PriceMultiplierMode: tt.mode}
			cost := channel.EstimateCostCNY(25, 175)
			require.NotNil(t, cost)
			assert.InDelta(t, tt.want, *cost, 1e-9)
		})
	}
}

func TestChannelEstimateCostCNYRejectsInvalidAmounts(t *testing.T) {
	channel := &Channel{PriceMultiplier: 0.2}
	for _, base := range []float64{-1, math.Inf(1), math.NaN()} {
		assert.Nil(t, channel.EstimateCostCNY(base, base))
	}
	cost := channel.EstimateCostCNY(0, 0)
	require.NotNil(t, cost)
	assert.Zero(t, *cost)
}

func TestChannelGetTestModel(t *testing.T) {
	testModel := "  gpt-4o-mini  "
	channel := &Channel{TestModel: &testModel}

	assert.Equal(t, "gpt-4o-mini", channel.GetTestModel())
	assert.Empty(t, (&Channel{}).GetTestModel())
}

func TestEnrichChannelIQProbeMetricsSelectsCurrentModelConfidence(t *testing.T) {
	testModel := "model-b"
	channel := &Channel{
		TestModel: &testModel,
		OtherInfo: `{"iq_probe_at":123,"iq_probe_prediction":"model-a","iq_probe_confidence":0.92,"iq_probe_model_confidences":{"model-a":0.92,"model-b":0.03}}`,
	}

	EnrichChannelIQProbeMetrics([]*Channel{channel})

	require.NotNil(t, channel.IQTestConfidence)
	assert.InDelta(t, 0.03, *channel.IQTestConfidence, 0.0001)
	assert.Equal(t, "model-a", channel.IQProbePrediction)
	require.NotNil(t, channel.IQProbeConfidence)
	assert.InDelta(t, 0.92, *channel.IQProbeConfidence, 0.0001)
	assert.Equal(t, map[string]float64{"model-a": 0.92, "model-b": 0.03}, channel.IQProbeModelConfidences)
}

func TestEnrichChannelIQProbeMetricsUsesNeutralConfidenceWithoutResults(t *testing.T) {
	channel := &Channel{}

	EnrichChannelIQProbeMetrics([]*Channel{channel})

	require.NotNil(t, channel.IQTestConfidence)
	assert.Equal(t, float64(1), *channel.IQTestConfidence)
	assert.Nil(t, channel.IQProbeConfidence)
	assert.Empty(t, channel.IQProbeModelConfidences)
}

func TestEnrichChannelIQProbeMetricsUsesZeroForUnlistedModelAfterTest(t *testing.T) {
	testModel := "unlisted-model"
	channel := &Channel{
		TestModel: &testModel,
		OtherInfo: `{"iq_probe_at":123,"iq_probe_model_confidences":{"model-a":0.92}}`,
	}

	EnrichChannelIQProbeMetrics([]*Channel{channel})

	require.NotNil(t, channel.IQTestConfidence)
	assert.Zero(t, *channel.IQTestConfidence)
}

func TestSetChannelIQProbeAnalysisIgnoresMissingAnalysis(t *testing.T) {
	channel := &Channel{OtherInfo: `{"iq_probe_at":123,"iq_probe_prediction":"model-a","iq_probe_model_confidences":{"model-a":0.92}}`}
	original := channel.OtherInfo

	require.Error(t, setChannelIQProbeAnalysis(channel, "model-b", nil, 456))

	assert.Equal(t, original, channel.OtherInfo)
}

func TestSetChannelIQProbeAnalysisWithNullMetadata(t *testing.T) {
	testModel := "model-a"
	channel := &Channel{OtherInfo: "null", TestModel: &testModel}
	require.NoError(t, setChannelIQProbeAnalysis(channel, testModel, &ChannelIQProbeAnalysis{
		Prediction: "model-a", ModelConfidences: map[string]float64{"model-a": 0.9},
	}, 456))
	EnrichChannelIQProbeMetrics([]*Channel{channel})
	require.NotNil(t, channel.IQTestConfidence)
	assert.Equal(t, 0.9, *channel.IQTestConfidence)
	assert.Equal(t, int64(456), channel.IQProbeAt)
}
