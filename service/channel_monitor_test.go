package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const channelMonitorTestPricingGroup = "monitor-pricing"

func setupChannelMonitorServiceTest(t *testing.T) {
	t.Helper()

	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	previousRetryPolicies := ratio_setting.PricingGroupRetryPolicy2JSONString()
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		for _, table := range []string{
			"channel_monitor_histories",
			"channel_monitors",
			"billing_group_channels",
			"billing_group_routes",
			"abilities",
			"channels",
		} {
			assert.NoError(t, model.DB.Exec("DELETE FROM "+table).Error)
		}
		assert.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		assert.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(previousRetryPolicies))
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		model.InitChannelRoutingCache()
		if previousMemoryCacheEnabled {
			model.InitChannelCache()
		}
	})

	require.NoError(t, model.DB.AutoMigrate(
		&model.Channel{},
		&model.Ability{},
		&model.BillingGroupRoute{},
		&model.BillingGroupChannel{},
		&model.ChannelMonitor{},
		&model.ChannelMonitorHistory{},
	))
	for _, table := range []string{
		"channel_monitor_histories",
		"channel_monitors",
		"billing_group_channels",
		"billing_group_routes",
		"abilities",
		"channels",
	} {
		require.NoError(t, model.DB.Exec("DELETE FROM "+table).Error)
	}
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"monitor-pricing":1}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(
		`{"monitor-pricing":{"mode":"fixed","retry_times":3}}`,
	))
	common.MemoryCacheEnabled = false
	model.InitChannelRoutingCache()
}

func validChannelMonitorInput() ChannelMonitorInput {
	return ChannelMonitorInput{
		PricingGroup:             channelMonitorTestPricingGroup,
		TestModel:                "gpt-test",
		IntervalSeconds:          60,
		TimeoutSeconds:           15,
		Enabled:                  true,
		Visible:                  true,
		AvailabilityBoostPercent: 0,
		CreatedBy:                1,
	}
}

func seedChannelMonitorCandidate(t *testing.T) *model.Channel {
	t.Helper()
	weight := uint(100)
	channel := &model.Channel{
		Id:     93001,
		Name:   "Monitor candidate",
		Key:    "sk-internal",
		Status: common.ChannelStatusEnabled,
		Models: "gpt-test",
		Group:  channelMonitorTestPricingGroup,
		Weight: &weight,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     channelMonitorTestPricingGroup,
		Model:     "gpt-test",
		ChannelId: channel.Id,
		Enabled:   true,
		Weight:    weight,
	}).Error)
	return channel
}

func seedAdditionalChannelMonitorCandidate(t *testing.T, id int) *model.Channel {
	t.Helper()
	weight := uint(100)
	channel := &model.Channel{
		Id:     id,
		Name:   "Additional monitor candidate",
		Key:    "sk-internal-additional",
		Status: common.ChannelStatusEnabled,
		Models: "gpt-test",
		Group:  channelMonitorTestPricingGroup,
		Weight: &weight,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     channelMonitorTestPricingGroup,
		Model:     "gpt-test",
		ChannelId: channel.Id,
		Enabled:   true,
		Weight:    weight,
	}).Error)
	return channel
}

func TestChannelMonitorAvailabilityBoostValidation(t *testing.T) {
	require.NoError(t, validateChannelMonitorAvailabilityBoost(99.95))

	for _, invalid := range []float64{-0.01, 100.01, math.NaN(), math.Inf(1)} {
		t.Run("invalid", func(t *testing.T) {
			require.Error(t, validateChannelMonitorAvailabilityBoost(invalid))
		})
	}
}

func TestApplyChannelMonitorAvailabilityBoostUsesFailureGap(t *testing.T) {
	tests := []struct {
		name     string
		raw      float64
		boost    float64
		expected float64
	}{
		{name: "zero boost preserves raw value", raw: 80, boost: 0, expected: 80},
		{name: "ten percent recovers ten percent of failures", raw: 80, boost: 10, expected: 82},
		{name: "twenty percent recovers twenty percent of failures", raw: 95, boost: 20, expected: 96},
		{name: "result keeps two decimals", raw: 99.5, boost: 10, expected: 99.55},
		{name: "one hundred stays capped", raw: 100, boost: 100, expected: 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := applyChannelMonitorAvailabilityBoost(&test.raw, test.boost)
			require.NotNil(t, resolved)
			assert.Equal(t, test.expected, *resolved)
		})
	}
	assert.Nil(t, applyChannelMonitorAvailabilityBoost(nil, 100))
}

func TestCreateChannelMonitorRequiresExistingPricingGroup(t *testing.T) {
	setupChannelMonitorServiceTest(t)

	input := validChannelMonitorInput()
	input.PricingGroup = "missing-pricing"
	_, err := CreateChannelMonitor(input)
	require.EqualError(t, err, "pricing group does not exist")

	monitor, err := CreateChannelMonitor(validChannelMonitorInput())
	require.NoError(t, err)
	assert.Equal(t, channelMonitorTestPricingGroup, monitor.PricingGroup)
}

func TestUpdateChannelMonitorEnablesAndDisablesScheduling(t *testing.T) {
	setupChannelMonitorServiceTest(t)

	input := validChannelMonitorInput()
	input.Enabled = false
	monitor, err := CreateChannelMonitor(input)
	require.NoError(t, err)
	assert.Nil(t, monitor.NextCheckAt)

	input.Enabled = true
	enabled, err := UpdateChannelMonitor(monitor.Id, input)
	require.NoError(t, err)
	require.NotNil(t, enabled.NextCheckAt)

	input.Enabled = false
	disabled, err := UpdateChannelMonitor(monitor.Id, input)
	require.NoError(t, err)
	assert.Nil(t, disabled.NextCheckAt)
}

func TestRunUserChannelMonitorTestPersistsProbeResultAndEnforcesCooldown(t *testing.T) {
	setupChannelMonitorServiceTest(t)
	channel := seedChannelMonitorCandidate(t)
	monitor, err := CreateChannelMonitor(validChannelMonitorInput())
	require.NoError(t, err)

	probeCalls := 0
	probe := func(ctx context.Context, selected *model.Channel, selectedMonitor *model.ChannelMonitor) (int, int, error) {
		probeCalls++
		assert.Equal(t, channel.Id, selected.Id)
		assert.Equal(t, channelMonitorTestPricingGroup, selectedMonitor.PricingGroup)
		assert.Equal(t, "gpt-test", selectedMonitor.TestModel)
		_, hasDeadline := ctx.Deadline()
		assert.True(t, hasDeadline)
		return 204, 37, nil
	}

	result, err := RunUserChannelMonitorTest(context.Background(), monitor.Id, probe)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 37, result.LatencyMs)
	assert.Equal(t, 1, probeCalls)
	assert.GreaterOrEqual(t, result.NextTestAt, result.CheckedAt+ChannelMonitorUserTestCooldownSeconds)

	history, err := model.ListChannelMonitorHistory(monitor.Id, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.True(t, history[0].Success)
	assert.Equal(t, 204, history[0].StatusCode)
	assert.Equal(t, 37, history[0].LatencyMs)
	assert.Equal(t, result.CheckedAt, history[0].CheckedAt)

	updated, err := model.GetChannelMonitorByID(monitor.Id)
	require.NoError(t, err)
	require.NotNil(t, updated.LastCheckedAt)
	assert.Equal(t, result.CheckedAt, *updated.LastCheckedAt)
	require.NotNil(t, updated.NextCheckAt)
	assert.Equal(t, result.CheckedAt+int64(monitor.IntervalSeconds), *updated.NextCheckAt)

	_, err = RunUserChannelMonitorTest(context.Background(), monitor.Id, probe)
	var cooldownErr *ChannelMonitorUserTestCooldownError
	require.True(t, errors.As(err, &cooldownErr))
	assert.Equal(t, result.NextTestAt, cooldownErr.NextTestAt)
	assert.Equal(t, 1, probeCalls)
}

func TestRunChannelMonitorCheckPersistsFailureWhenNoChannelIsAvailable(t *testing.T) {
	setupChannelMonitorServiceTest(t)
	monitor, err := CreateChannelMonitor(validChannelMonitorInput())
	require.NoError(t, err)

	probeCalled := false
	result, err := RunChannelMonitorCheck(context.Background(), monitor.Id, func(context.Context, *model.Channel, *model.ChannelMonitor) (int, int, error) {
		probeCalled = true
		return 200, 1, nil
	})
	require.NoError(t, err)
	assert.False(t, probeCalled)
	assert.False(t, result.Success)
	assert.Zero(t, result.StatusCode)
	assert.Contains(t, result.ErrorMessage, "no enabled channel")
	assert.Contains(t, result.ErrorMessage, channelMonitorTestPricingGroup)

	history, err := model.ListChannelMonitorHistory(monitor.Id, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, result.ErrorMessage, history[0].ErrorMessage)
	assert.False(t, history[0].Success)
}

func TestRunChannelMonitorCheckUsesPricingGroupRetryPolicyAcrossDifferentChannels(t *testing.T) {
	setupChannelMonitorServiceTest(t)
	seedChannelMonitorCandidate(t)
	seedAdditionalChannelMonitorCandidate(t, 93002)
	seedAdditionalChannelMonitorCandidate(t, 93003)
	require.NoError(t, ratio_setting.UpdatePricingGroupRetryPolicyByJSONString(
		`{"monitor-pricing":{"mode":"fixed","retry_times":1}}`,
	))
	monitor, err := CreateChannelMonitor(validChannelMonitorInput())
	require.NoError(t, err)

	attempted := make(map[int]int)
	result, err := RunChannelMonitorCheck(context.Background(), monitor.Id, func(_ context.Context, channel *model.Channel, _ *model.ChannelMonitor) (int, int, error) {
		attempted[channel.Id]++
		return 503, 8, errors.New("upstream unavailable")
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Len(t, attempted, 2)
	for _, attempts := range attempted {
		assert.Equal(t, 1, attempts)
	}
}
