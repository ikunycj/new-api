package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelCircuitOptionUpdatesRuntimeState(t *testing.T) {
	previous := common.IsChannelCircuitEnabled()
	t.Cleanup(func() { common.SetChannelCircuitEnabled(previous) })

	require.NoError(t, updateOptionMap(common.ChannelCircuitEnabledOptionKey, "true"))
	assert.True(t, common.IsChannelCircuitEnabled())
	require.NoError(t, updateOptionMap(common.ChannelCircuitEnabledOptionKey, "false"))
	assert.False(t, common.IsChannelCircuitEnabled())
}

func TestChannelCircuitOptionRejectsInvalidValues(t *testing.T) {
	require.Error(t, updateOptionMap(common.ChannelCircuitEnabledOptionKey, "enabled"))
	require.Error(t, updateOptionMap(ChannelCircuitConfigOptionKey, `{"default":{}}`))
}

func TestChannelCircuitConfigSyncOnlyNotifiesOnChange(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[ChannelCircuitConfigOptionKey] = DefaultChannelCircuitConfigJSONString()
	common.OptionMapRWMutex.Unlock()
	before := common.ChannelCircuitConfigVersion()

	require.NoError(t, updateOptionMap(ChannelCircuitConfigOptionKey, DefaultChannelCircuitConfigJSONString()))
	assert.Equal(t, before, common.ChannelCircuitConfigVersion())

	changed := `{"default":{"failure_threshold":6,"window_seconds":61,"cooldown_seconds":62,"half_open_requests":2},"modes":{"cost_first":{"failure_threshold":8,"window_seconds":60,"cooldown_seconds":60,"half_open_requests":1},"stability_first":{"failure_threshold":3,"window_seconds":60,"cooldown_seconds":90,"half_open_requests":1}},"presets":[{"key":"standard","label":"Standard","failure_threshold":20,"window_seconds":60,"cooldown_seconds":30,"half_open_requests":1}]}`
	require.NoError(t, updateOptionMap(ChannelCircuitConfigOptionKey, changed))
	assert.Equal(t, before+1, common.ChannelCircuitConfigVersion())
}
