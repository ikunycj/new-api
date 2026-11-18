package controller

import (
	"testing"
	"time"

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

func TestShouldRunChannelIQProbeRequiresEnabledSupportedChannel(t *testing.T) {
	testModel := common.GetPointer("gpt-4o")
	assert.False(t, shouldRunChannelIQProbe(nil))
	assert.False(t, shouldRunChannelIQProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, TestModel: testModel}))
	assert.False(t, shouldRunChannelIQProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusManuallyDisabled, TestModel: testModel, IQProbeEnabled: true}))
	assert.False(t, shouldRunChannelIQProbe(&model.Channel{Type: constant.ChannelTypeKling, Status: common.ChannelStatusEnabled, TestModel: testModel, IQProbeEnabled: true}))
	assert.True(t, shouldRunChannelIQProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, TestModel: testModel, IQProbeEnabled: true}))
	assert.True(t, shouldRunChannelIQProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusAutoDisabled, TestModel: testModel, IQProbeEnabled: true}))
	for _, nonText := range []string{"text-embedding-3-small", "rerank-v3", "gpt-5-openai-compact", "gpt-image-1"} {
		assert.False(t, shouldRunChannelIQProbe(&model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, TestModel: &nonText, IQProbeEnabled: true}), nonText)
	}
	for _, nonText := range []int{constant.ChannelTypeSora, constant.ChannelTypeJina, constant.ChannelTypeMokaAI} {
		assert.False(t, shouldRunChannelIQProbe(&model.Channel{Type: nonText, Status: common.ChannelStatusEnabled, TestModel: testModel, IQProbeEnabled: true}))
	}
}

func TestChannelIQProbeHandlerIsScheduled(t *testing.T) {
	scheduled, ok := any(channelIQProbeHandler{}).(service.ScheduledSystemTaskHandler)
	assert.True(t, ok)
	assert.Equal(t, model.SystemTaskTypeChannelIQProbe, scheduled.Type())
	assert.Equal(t, time.Minute, scheduled.Interval())
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
