package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
)

func TestSupportsChannelTest(t *testing.T) {
	unsupported := []int{
		constant.ChannelTypeMidjourney,
		constant.ChannelTypeMidjourneyPlus,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu,
	}
	for _, channelType := range unsupported {
		assert.False(t, supportsChannelTest(channelType), "channel type %d", channelType)
	}
	assert.True(t, supportsChannelTest(constant.ChannelTypeOpenAI))
}

func TestShouldRunChannelProbeRequiresSupportedRecoverableChannel(t *testing.T) {
	testModel := common.GetPointer("gpt-4o")
	autoProbeDisabled := false
	autoProbeEnabled := true
	assert.False(t, shouldRunChannelProbe(nil))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, TestModel: testModel, AutoProbeEnabled: &autoProbeDisabled}))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusManuallyDisabled, TestModel: testModel, AutoProbeEnabled: &autoProbeEnabled}))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeKling, Status: common.ChannelStatusEnabled}))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, TestModel: testModel, AutoProbeEnabled: &autoProbeEnabled}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusAutoDisabled, TestModel: testModel, AutoProbeEnabled: &autoProbeEnabled}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{
		Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled,
		TestModel: testModel, AutoProbeEnabled: &autoProbeEnabled,
		ChannelInfo: model.ChannelInfo{IsMultiKey: true},
	}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{
		Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusAutoDisabled,
		TestModel: testModel, AutoProbeEnabled: &autoProbeEnabled,
		ChannelInfo: model.ChannelInfo{IsMultiKey: true},
	}))
}

func TestShouldUseStreamForAutomaticChannelTest(t *testing.T) {
	assert.False(t, shouldUseStreamForAutomaticChannelTest(nil))
	assert.False(t, shouldUseStreamForAutomaticChannelTest(&model.Channel{Type: constant.ChannelTypeOpenAI}))
	assert.True(t, shouldUseStreamForAutomaticChannelTest(&model.Channel{Type: constant.ChannelTypeOpenAI, ProbeStreamEnabled: true}))
	assert.True(t, shouldUseStreamForAutomaticChannelTest(&model.Channel{Type: constant.ChannelTypeCodex}))
}

func TestChannelTestHandlerIsOnDemandOnly(t *testing.T) {
	_, scheduled := any(channelTestHandler{}).(service.ScheduledSystemTaskHandler)
	assert.False(t, scheduled)
}
