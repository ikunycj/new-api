package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestChannelCircuitOpensAndRecoversAfterSuccess(t *testing.T) {
	policy := model.DefaultRuntimeRoutingPolicy(model.RoutingModeBalanced)
	policy.CircuitFailureThreshold = 2
	route := "/test/circuit"
	channelID := 991
	RecordChannelCircuitSuccess(channelID, route)

	assert.True(t, ChannelCircuitAllows(channelID, route, policy))
	RecordChannelCircuitFailure(channelID, route, policy)
	assert.True(t, ChannelCircuitAllows(channelID, route, policy))
	RecordChannelCircuitFailure(channelID, route, policy)
	assert.False(t, ChannelCircuitAllows(channelID, route, policy))

	RecordChannelCircuitSuccess(channelID, route)
	assert.True(t, ChannelCircuitAllows(channelID, route, policy))
}
