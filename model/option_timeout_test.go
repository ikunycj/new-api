package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutOptionUpdatesRuntimeState(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	previousMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	previousRelay := common.RelayTimeout
	previousStreaming := constant.StreamingTimeout
	previousIdle := common.RelayIdleConnTimeout
	previousWrite := common.StreamClientWriteTimeout
	previousShutdown := common.ShutdownTimeoutSeconds
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousMap
		common.OptionMapRWMutex.Unlock()
		common.RelayTimeout = previousRelay
		constant.StreamingTimeout = previousStreaming
		common.RelayIdleConnTimeout = previousIdle
		common.StreamClientWriteTimeout = previousWrite
		common.ShutdownTimeoutSeconds = previousShutdown
	})

	timeoutValues := map[string]string{
		"RelayTimeout":             "45",
		"StreamingTimeout":         "120",
		"RelayIdleConnTimeout":     "30",
		"StreamClientWriteTimeout": "20",
		"ShutdownTimeoutSeconds":   "90",
	}
	for key, value := range timeoutValues {
		require.NoError(t, updateOptionMap(key, value))
	}

	assert.Equal(t, 45, common.RelayTimeout)
	assert.Equal(t, 120, constant.StreamingTimeout)
	assert.Equal(t, 30, common.RelayIdleConnTimeout)
	assert.Equal(t, 20, common.StreamClientWriteTimeout)
	assert.Equal(t, 90, common.ShutdownTimeoutSeconds)
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	for key, value := range timeoutValues {
		assert.Equal(t, value, common.OptionMap[key])
	}
}
