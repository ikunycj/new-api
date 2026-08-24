package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
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
	assert.False(t, shouldRunChannelProbe(nil))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Status: common.ChannelStatusManuallyDisabled}))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeKling, Status: common.ChannelStatusEnabled}))
	assert.False(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, TestModel: testModel}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusAutoDisabled, TestModel: testModel}))
	assert.True(t, shouldRunChannelProbe(&model.Channel{
		Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled,
		TestModel:   testModel,
		ChannelInfo: model.ChannelInfo{IsMultiKey: true},
	}))
	assert.False(t, shouldRunChannelProbe(&model.Channel{
		Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusAutoDisabled,
		TestModel:   testModel,
		ChannelInfo: model.ChannelInfo{IsMultiKey: true},
	}))
}

func TestChannelProbeIntervalUsesResultingStatus(t *testing.T) {
	channel := &model.Channel{
		ProbeIntervalSeconds:             11,
		AutoDisabledProbeIntervalSeconds: 29,
	}

	assert.Equal(t, 11, channelProbeIntervalSeconds(channel, common.ChannelStatusEnabled))
	assert.Equal(t, 29, channelProbeIntervalSeconds(channel, common.ChannelStatusAutoDisabled))
}
