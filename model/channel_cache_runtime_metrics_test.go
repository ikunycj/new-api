package model

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCachedChannelTestTTFT(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	channelSyncLock.Lock()
	previousChannels := channelsIDM
	common.MemoryCacheEnabled = true
	channelsIDM = map[int]*Channel{
		95009: {Id: 95009, LastTestTTFTMs: 120},
	}
	channelSyncLock.Unlock()
	t.Cleanup(func() {
		channelSyncLock.Lock()
		channelsIDM = previousChannels
		channelSyncLock.Unlock()
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
	})

	UpdateCachedChannelTestTTFT(95009, 80.5)
	channelSyncLock.RLock()
	updated := channelsIDM[95009]
	channelSyncLock.RUnlock()
	require.NotNil(t, updated)
	assert.InDelta(t, 80.5, updated.LastTestTTFTMs, 0.000001)

	for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		UpdateCachedChannelTestTTFT(95009, invalid)
	}
	channelSyncLock.RLock()
	unchanged := channelsIDM[95009]
	channelSyncLock.RUnlock()
	require.NotNil(t, unchanged)
	assert.InDelta(t, 80.5, unchanged.LastTestTTFTMs, 0.000001)

	UpdateCachedChannelTestTTFT(95010, 50)
	channelSyncLock.RLock()
	_, inserted := channelsIDM[95010]
	channelSyncLock.RUnlock()
	assert.False(t, inserted)
}

func TestUpdateCachedChannelTestTTFTIsDisabledWithoutMemoryCache(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	channelSyncLock.Lock()
	previousChannels := channelsIDM
	common.MemoryCacheEnabled = false
	channelsIDM = map[int]*Channel{
		95011: {Id: 95011, LastTestTTFTMs: 120},
	}
	channelSyncLock.Unlock()
	t.Cleanup(func() {
		channelSyncLock.Lock()
		channelsIDM = previousChannels
		channelSyncLock.Unlock()
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
	})

	UpdateCachedChannelTestTTFT(95011, 60)
	channelSyncLock.RLock()
	unchanged := channelsIDM[95011]
	channelSyncLock.RUnlock()
	require.NotNil(t, unchanged)
	assert.InDelta(t, 120, unchanged.LastTestTTFTMs, 0.000001)
}
