package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setCircuitEnabledForTest(t *testing.T, enabled bool) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[common.ChannelCircuitEnabledOptionKey] = "false"
	if enabled {
		common.OptionMap[common.ChannelCircuitEnabledOptionKey] = "true"
	}
	common.SetChannelCircuitEnabled(enabled)
	common.OptionMapRWMutex.Unlock()
	clearChannelCircuitState()
	circuitConfigState.Lock()
	circuitConfigState.initialized = false
	circuitConfigState.enabled = false
	circuitConfigState.version = 0
	circuitConfigState.Unlock()
}

func TestChannelCircuitOpensAndRecoversAfterSuccess(t *testing.T) {
	setCircuitEnabledForTest(t, true)
	t.Cleanup(func() { setCircuitEnabledForTest(t, false) })
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

func TestChannelCircuitDisabledBypassesAndDoesNotRecordFailures(t *testing.T) {
	setCircuitEnabledForTest(t, false)
	t.Cleanup(func() { setCircuitEnabledForTest(t, false) })
	policy := model.DefaultRuntimeRoutingPolicy(model.RoutingModeBalanced)
	policy.CircuitFailureThreshold = 1
	channelID := 992
	route := "/test/circuit-disabled"

	RecordChannelCircuitFailure(channelID, route, policy)
	require.True(t, ChannelCircuitAllows(channelID, route, policy))

	setCircuitEnabledForTest(t, true)
	assert.True(t, ChannelCircuitAllows(channelID, route, policy))
}

func TestChannelCircuitToggleClearsOpenState(t *testing.T) {
	setCircuitEnabledForTest(t, true)
	t.Cleanup(func() { setCircuitEnabledForTest(t, false) })
	policy := model.DefaultRuntimeRoutingPolicy(model.RoutingModeBalanced)
	policy.CircuitFailureThreshold = 1
	channelID := 993
	route := "/test/circuit-toggle"

	RecordChannelCircuitFailure(channelID, route, policy)
	require.False(t, ChannelCircuitAllows(channelID, route, policy))
	setCircuitEnabledForTest(t, false)
	require.True(t, ChannelCircuitAllows(channelID, route, policy))
	setCircuitEnabledForTest(t, true)
	assert.True(t, ChannelCircuitAllows(channelID, route, policy))
}
