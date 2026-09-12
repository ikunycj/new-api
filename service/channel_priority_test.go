package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateChannelPriorityScoresUsesMultiplierAndFullReference(t *testing.T) {
	channels := []*model.Channel{
		{
			Id:                          95001,
			Status:                      common.ChannelStatusEnabled,
			Group:                       "priority-test",
			PriceMultiplier:             0.10,
			PreviousDayProbeSuccessRate: 95,
			PreviousDayProbeSampleCount: 100,
		},
		{
			Id:                          95002,
			Status:                      common.ChannelStatusEnabled,
			Group:                       "priority-test",
			PriceMultiplier:             0.11,
			PreviousDayProbeSuccessRate: 95,
			PreviousDayProbeSampleCount: 100,
		},
		{
			Id:                          95003,
			Status:                      common.ChannelStatusEnabled,
			Group:                       "priority-test",
			PriceMultiplier:             0.14,
			PreviousDayProbeSuccessRate: 95,
			PreviousDayProbeSampleCount: 100,
		},
	}

	// Simulate a paginated response: the cheapest channel is only in the
	// reference set, so the page's scores must still use its multiplier.
	page := []*model.Channel{channels[1], channels[2]}
	CalculateChannelPriorityScores(page, channels, "priority-test")

	require.NotNil(t, page[0].PriorityScore)
	require.NotNil(t, page[1].PriorityScore)
	assert.InDelta(t, 94.3636, *page[0].PriorityScore, 0.0001)
	assert.InDelta(t, 86.5714, *page[1].PriorityScore, 0.0001)
}

func TestSortChannelsByPriorityOrdersScoresInRequestedDirection(t *testing.T) {
	channels := []*model.Channel{
		{
			Id:                          95011,
			Status:                      common.ChannelStatusEnabled,
			Group:                       "priority-sort-test",
			PriceMultiplier:             0.10,
			PreviousDayProbeSuccessRate: 95,
			PreviousDayProbeSampleCount: 100,
		},
		{
			Id:                          95012,
			Status:                      common.ChannelStatusEnabled,
			Group:                       "priority-sort-test",
			PriceMultiplier:             0.14,
			PreviousDayProbeSuccessRate: 95,
			PreviousDayProbeSampleCount: 100,
		},
	}

	descending := []*model.Channel{channels[1], channels[0]}
	SortChannelsByPriority(descending, channels, "priority-sort-test", true)
	assert.Equal(t, 95011, descending[0].Id)
	assert.Equal(t, 95012, descending[1].Id)

	ascending := []*model.Channel{channels[0], channels[1]}
	SortChannelsByPriority(ascending, channels, "priority-sort-test", false)
	assert.Equal(t, 95012, ascending[0].Id)
	assert.Equal(t, 95011, ascending[1].Id)
}

func TestChannelPriorityUsesRecentTestTTFTFromFullReferenceSet(t *testing.T) {
	const group = "priority-recent-ttft-test"
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousStrategies := ratio_setting.PricingGroupRoutingStrategy2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupRoutingStrategyByJSONString(previousStrategies))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"`+group+`":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupRoutingStrategyByJSONString(`{
		"strategies":{"recent":{"name":"最近测试 TTFT","price_weight":0,"availability_weight":0,"load_weight":0,"ttft_weight":0,"recent_test_ttft_weight":100}},
		"group_bindings":{"`+group+`":"recent"}
	}`))

	fast := &model.Channel{
		Id:             95021,
		Status:         common.ChannelStatusEnabled,
		Group:          group,
		LastTestTTFTMs: 100,
	}
	medium := &model.Channel{
		Id:             95022,
		Status:         common.ChannelStatusEnabled,
		Group:          group,
		LastTestTTFTMs: 200,
	}
	slow := &model.Channel{
		Id:             95023,
		Status:         common.ChannelStatusEnabled,
		Group:          group,
		LastTestTTFTMs: 400,
	}
	reference := []*model.Channel{fast, medium, slow}
	page := []*model.Channel{slow, medium}

	CalculateChannelPriorityScores(page, reference, group)
	require.NotNil(t, page[0].PriorityScore)
	require.NotNil(t, page[1].PriorityScore)
	// The omitted fast channel establishes the 100ms baseline. If scoring used
	// only the paginated page, the medium channel would incorrectly score 100.
	assert.InDelta(t, 25, *page[0].PriorityScore, 0.0001)
	assert.InDelta(t, 50, *page[1].PriorityScore, 0.0001)

	SortChannelsByPriority(page, reference, group, true)
	assert.Equal(t, medium.Id, page[0].Id)
	assert.Equal(t, slow.Id, page[1].Id)
	assert.InDelta(t, 50, *page[0].PriorityScore, 0.0001)
	assert.InDelta(t, 25, *page[1].PriorityScore, 0.0001)
}
