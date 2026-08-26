package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestChannelCalculateTokenCostUSDUsesMultiplierCurrency(t *testing.T) {
	usdChannel := &Channel{PriceMultiplier: 2, PriceMultiplierMode: ChannelPriceMultiplierModeUSD}
	assert.InDelta(t, 3, usdChannel.CalculateTokenCostUSD(1_500_000, 7.5), 0.000001)

	cnyChannel := &Channel{PriceMultiplier: 15, PriceMultiplierMode: ChannelPriceMultiplierModeCNY}
	assert.InDelta(t, 3, cnyChannel.CalculateTokenCostUSD(1_500_000, 7.5), 0.000001)
	assert.Zero(t, cnyChannel.CalculateTokenCostUSD(0, 7.5))
}

func TestChannelGetTestModel(t *testing.T) {
	testModel := "  gpt-4o-mini  "
	channel := &Channel{TestModel: &testModel}

	assert.Equal(t, "gpt-4o-mini", channel.GetTestModel())
	assert.Empty(t, (&Channel{}).GetTestModel())
}
