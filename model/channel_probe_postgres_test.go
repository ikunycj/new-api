package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestChannelStatusNotificationSettingPersists(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{})
	channel := Channel{
		Id:     10,
		Name:   "notification-test",
		Key:    "channel-key",
		Models: "gpt-4o",
		Group:  "default",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, DB.Create(&channel).Error)

	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.False(t, stored.ShouldNotifyOnStatusChange())

	channel.NotifyOnStatusChange = common.GetPointer(true)
	require.NoError(t, channel.Update("notify_on_status_change"))
	stored = Channel{}
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.True(t, stored.ShouldNotifyOnStatusChange())

	channel.NotifyOnStatusChange = common.GetPointer(false)
	require.NoError(t, channel.Update("notify_on_status_change"))
	stored = Channel{}
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.False(t, stored.ShouldNotifyOnStatusChange())
}

func TestGetChannelProbeCandidatesFiltersAndOrdersPeriodGroups(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{})
	channels := []Channel{
		{Id: 1, Key: "never-expose", AutoProbeEnabled: common.GetPointer(true), Status: common.ChannelStatusEnabled, ProbePeriodMinutes: 2},
		{Id: 2, AutoProbeEnabled: common.GetPointer(true), Status: common.ChannelStatusAutoDisabled, ProbePeriodMinutes: 1},
		{Id: 3, AutoProbeEnabled: common.GetPointer(true), Status: common.ChannelStatusEnabled, ProbePeriodMinutes: 3},
		{Id: 4, AutoProbeEnabled: common.GetPointer(false), Status: common.ChannelStatusEnabled, ProbePeriodMinutes: 1},
		{Id: 5, AutoProbeEnabled: common.GetPointer(true), Status: common.ChannelStatusManuallyDisabled, ProbePeriodMinutes: 1},
	}
	require.NoError(t, DB.Create(&channels).Error)
	candidates, err := GetChannelProbeCandidates(context.Background())
	require.NoError(t, err)
	require.Len(t, candidates, 3)
	assert.Equal(t, []int{2, 1, 3}, []int{candidates[0].Id, candidates[1].Id, candidates[2].Id})
	assert.Empty(t, candidates[1].Key)
}

func TestSaveChannelIQProbeAnalysisPersistsManualTestConfidence(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &ChannelProbeHistory{})
	previousCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })

	testModel := "model-b"
	channel := Channel{Id: 10, Key: "channel-key", TestModel: &testModel, Status: common.ChannelStatusEnabled, OtherInfo: `{"preserved":"value"}`}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&ChannelProbeHistory{ChannelID: channel.Id, Success: true, CheckedAt: 100}).Error)

	_, _, err := CompleteManualChannelIQTest(context.Background(), channel.Id, testModel, "", &ChannelIQProbeAnalysis{
		Prediction: "model-a",
		ModelConfidences: map[string]float64{
			"model-a": 0.91,
			"model-b": 0.04,
		},
	})
	require.NoError(t, err)

	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, "value", stored.GetOtherInfo()["preserved"])
	assert.Equal(t, "model-a", stored.GetOtherInfo()["iq_probe_prediction"])
	assert.True(t, stored.GetOtherInfo()["iq_probe_degraded"].(bool))
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	var historyCount int64
	require.NoError(t, DB.Model(&ChannelProbeHistory{}).Where("channel_id = ?", channel.Id).Count(&historyCount).Error)
	assert.Equal(t, int64(1), historyCount)

	EnrichChannelIQProbeMetrics([]*Channel{&stored})
	require.NotNil(t, stored.IQTestConfidence)
	assert.InDelta(t, 0.04, *stored.IQTestConfidence, 0.0001)
	require.NotNil(t, stored.IQProbeConfidence)
	assert.InDelta(t, 0.91, *stored.IQProbeConfidence, 0.0001)

	_, _, err = CompleteManualChannelIQTest(context.Background(), channel.Id, testModel, "", &ChannelIQProbeAnalysis{
		Prediction: "model-b",
		ModelConfidences: map[string]float64{
			"model-a": 0.02,
			"model-b": 0.88,
		},
	})
	require.NoError(t, err)
	stored = Channel{}
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.False(t, stored.GetOtherInfo()["iq_probe_degraded"].(bool))
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	require.NoError(t, DB.Model(&ChannelProbeHistory{}).Where("channel_id = ?", channel.Id).Count(&historyCount).Error)
	assert.Equal(t, int64(1), historyCount)
}

func TestChannelEditPreservesLatestIQProbeResult(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
	testModel := "model-b"
	channel := Channel{
		Key: "test-key", Models: testModel, Group: "default", TestModel: &testModel,
		IQProbeEnabled: true, IQAutoDisableDegraded: true, OtherInfo: `{"provider":{"id":9007199254740995}}`,
	}
	require.NoError(t, channel.Insert())
	// The edit was prepared before the scheduled probe disabled the channel.
	edit := channel
	lease := common.GetTimestamp() + 300
	require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, IQLeaseUntil: lease}).Error)
	observed, changed, err := CompleteChannelIQProbe(t.Context(), channel.Id, &ChannelProbeHistory{Success: true}, lease, "", testModel, &ChannelIQProbeAnalysis{
		Prediction: "model-a", ModelConfidences: map[string]float64{"model-a": 0.95, testModel: 0.05},
	})
	require.NoError(t, err)
	require.True(t, changed)
	assert.Contains(t, observed.OtherInfo, `"id":9007199254740995`)
	edit.OtherInfo = `{"provider":{"id":9007199254740993}}`
	require.NoError(t, edit.Update("other_info"))
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Contains(t, stored.OtherInfo, `"id":9007199254740993`)
	EnrichChannelIQProbeMetrics([]*Channel{&stored})
	require.NotNil(t, stored.IQTestConfidence)
	assert.Equal(t, 0.05, *stored.IQTestConfidence)
	assert.Equal(t, "model-a", stored.IQProbePrediction)
	assert.True(t, stored.IQProbeDegraded())
	// Editing unrelated metadata must not remove the reason required for manual recovery.
	recovered, changed, err := CompleteManualChannelIQTest(t.Context(), channel.Id, testModel, "", &ChannelIQProbeAnalysis{
		Prediction: testModel, ModelConfidences: map[string]float64{testModel: 1},
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, common.ChannelStatusEnabled, recovered.Status)
	assert.Contains(t, recovered.OtherInfo, `"id":9007199254740993`)
}

func TestSuccessfulManualIQTestRecoversDegradedChannel(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{})
	previousCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })

	testModel := "model-b"
	degradedInfo := `{"status_reason":"IQ probe detected model degradation"}`
	channels := []Channel{
		{Id: 20, TestModel: &testModel, Status: common.ChannelStatusAutoDisabled, IQAutoDisableDegraded: true, OtherInfo: degradedInfo},
		{Id: 21, TestModel: &testModel, Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true), IQAutoDisableDegraded: true, OtherInfo: degradedInfo},
	}
	require.NoError(t, DB.Create(&channels).Error)
	require.NoError(t, DB.Create(&[]Ability{
		{ChannelId: 20, Group: "test", Model: testModel, Enabled: false},
		{ChannelId: 21, Group: "test", Model: testModel, Enabled: false},
	}).Error)

	stored, changed, err := CompleteManualChannelIQTest(context.Background(), 20, testModel, "", &ChannelIQProbeAnalysis{
		Prediction:       testModel,
		ModelConfidences: map[string]float64{"model-a": 0.02, testModel: 0.88},
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)

	var recoveredAbility Ability
	require.NoError(t, DB.First(&recoveredAbility, "channel_id = ?", 20).Error)
	assert.True(t, recoveredAbility.Enabled)

	stored, changed, err = CompleteManualChannelIQTest(context.Background(), 21, testModel, "", &ChannelIQProbeAnalysis{
		Prediction:       "model-a",
		ModelConfidences: map[string]float64{"model-a": 0.91, testModel: 0.04},
	})
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	// A later, matching result for another model is diagnostic only, even with regular probing on.
	previous := stored.OtherInfo
	_, changed, err = CompleteManualChannelIQTest(context.Background(), 21, "model-a", "", &ChannelIQProbeAnalysis{
		Prediction: "model-a", ModelConfidences: map[string]float64{"model-a": 1},
	})
	require.NoError(t, err)
	assert.False(t, changed)
	require.NoError(t, DB.First(&stored, 21).Error)
	assert.Equal(t, previous, stored.OtherInfo)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	var disabledAbility Ability
	require.NoError(t, DB.First(&disabledAbility, "channel_id = ?", 21).Error)
	assert.False(t, disabledAbility.Enabled)
}

func TestCompleteChannelIQProbeAutoDisablesDegradedUntilAHealthyIQProbe(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
	previousCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })

	testModel := "model-b"
	channel := Channel{
		Id:                    12,
		Key:                   "channel-key",
		TestModel:             &testModel,
		Status:                common.ChannelStatusEnabled,
		AutoProbeEnabled:      common.GetPointer(true),
		IQProbeEnabled:        true,
		IQAutoDisableDegraded: true,
	}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: testModel, Enabled: true}).Error)
	lease := common.GetTimestamp() + 300
	require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, IQLeaseUntil: lease}).Error)

	stored, changed, err := CompleteChannelIQProbe(context.Background(), channel.Id, &ChannelProbeHistory{Success: true}, lease, "", testModel, &ChannelIQProbeAnalysis{
		Prediction:       "model-a",
		ModelConfidences: map[string]float64{"model-a": 0.91, "model-b": 0.04},
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	assert.Equal(t, "IQ probe detected model degradation", stored.GetOtherInfo()["status_reason"])

	var history ChannelProbeHistory
	require.NoError(t, DB.First(&history, "channel_id = ?", channel.Id).Error)
	assert.True(t, history.Success)
	var ability Ability
	require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
	assert.False(t, ability.Enabled)

	availabilityLease := common.GetTimestamp() + 400
	require.NoError(t, DB.Model(&ChannelProbeState{}).Where("channel_id = ?", channel.Id).Update("lease_until", availabilityLease).Error)
	stored, changed, err = CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: true}, availabilityLease, "")
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)

	healthyIQLease := common.GetTimestamp() + 500
	require.NoError(t, DB.Model(&ChannelProbeState{}).Where("channel_id = ?", channel.Id).Update("iq_lease_until", healthyIQLease).Error)
	stored, changed, err = CompleteChannelIQProbe(context.Background(), channel.Id, &ChannelProbeHistory{Success: true}, healthyIQLease, "", testModel, &ChannelIQProbeAnalysis{
		Prediction:       testModel,
		ModelConfidences: map[string]float64{"model-a": 0.02, testModel: 0.88},
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
	assert.True(t, ability.Enabled)
}

func TestEligibleChannelsIncludeIQConfidenceForDynamicRouting(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeHistory{}, &Log{})
	previousCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })

	testModel := "model-b"
	channel := Channel{
		Id:        11,
		Key:       "channel-key",
		TestModel: &testModel,
		OtherInfo: `{"iq_probe_at":123,"iq_probe_prediction":"model-a","iq_probe_model_confidences":{"model-a":0.91,"model-b":0.04}}`,
		Status:    common.ChannelStatusEnabled,
	}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "shared-model", Enabled: true}).Error)

	candidates, err := GetEligibleChannels("test", "shared-model", "", nil)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.NotNil(t, candidates[0].IQTestConfidence)
	assert.InDelta(t, 0.04, *candidates[0].IQTestConfidence, 0.0001)
}

func TestCompleteChannelProbeUsesCurrentStatusAndUnifiedPeriod(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		success    bool
		autoProbe  bool
		wantStatus int
	}{
		{"recover", common.ChannelStatusAutoDisabled, true, true, common.ChannelStatusEnabled},
		{"still failing", common.ChannelStatusAutoDisabled, false, true, common.ChannelStatusAutoDisabled},
		{"disable independently of business auto ban", common.ChannelStatusEnabled, false, true, common.ChannelStatusAutoDisabled},
		{"manual disable wins on success", common.ChannelStatusManuallyDisabled, true, true, common.ChannelStatusManuallyDisabled},
		{"manual disable wins on failure", common.ChannelStatusManuallyDisabled, false, true, common.ChannelStatusManuallyDisabled},
		{"auto probe switched off on success", common.ChannelStatusAutoDisabled, true, false, common.ChannelStatusAutoDisabled},
		{"auto probe switched off on failure", common.ChannelStatusEnabled, false, false, common.ChannelStatusEnabled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
			previousCacheEnabled := common.MemoryCacheEnabled
			common.MemoryCacheEnabled = false
			t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })
			previousAutoDisable := common.AutomaticDisableChannelEnabled
			common.AutomaticDisableChannelEnabled = false
			t.Cleanup(func() { common.AutomaticDisableChannelEnabled = previousAutoDisable })
			channel := Channel{Id: 11, Status: tc.status, AutoProbeEnabled: &tc.autoProbe, AutoBan: common.GetPointer(0), ProbePeriodMinutes: 2}
			require.NoError(t, DB.Create(&channel).Error)
			require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "gpt-4o", Enabled: tc.status == common.ChannelStatusEnabled}).Error)
			now := common.GetTimestamp()
			require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, LeaseUntil: now + 300}).Error)
			stored, changed, err := CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: tc.success, ErrorMessage: "probe failed", LatencyMs: 42}, now+300, "")
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, stored.Status)
			assert.Equal(t, tc.status != tc.wantStatus, changed)
			var state ChannelProbeState
			require.NoError(t, DB.First(&state, "channel_id = ?", channel.Id).Error)
			assert.Zero(t, state.LeaseUntil)
			assert.Equal(t, tc.success, state.LastSuccess)
			var history []ChannelProbeHistory
			require.NoError(t, DB.Find(&history).Error)
			require.Len(t, history, 1)
			assert.Equal(t, state.LastProbeAt, history[0].CheckedAt)
			var ability Ability
			require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, tc.wantStatus == common.ChannelStatusEnabled, ability.Enabled)
		})
	}
}

func TestCompleteChannelProbeRejectsExpiredOrReplacedLeaseWithoutChangingStatus(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		t.Run(map[bool]string{false: "expired", true: "replaced"}[replaced], func(t *testing.T) {
			setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
			channel := Channel{Id: 12, Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true)}
			require.NoError(t, DB.Create(&channel).Error)
			now := common.GetTimestamp()
			lease := now - 1
			storedLease := lease
			if replaced {
				lease = now + 200
				storedLease = now + 300
			}
			require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, LeaseUntil: storedLease}).Error)
			_, changed, err := CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: true}, lease, "")
			require.Error(t, err)
			assert.False(t, changed)
			require.NoError(t, DB.First(&channel, channel.Id).Error)
			assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
			var count int64
			require.NoError(t, DB.Model(&ChannelProbeHistory{}).Count(&count).Error)
			assert.Zero(t, count)
			var state ChannelProbeState
			require.NoError(t, DB.First(&state, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, storedLease, state.LeaseUntil)
			assert.Zero(t, state.LastProbeAt)
		})
	}
}

func TestRelayAutoDisableDoesNotCreateOrRescheduleProbeState(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "no state", true: "active lease"}[existing], func(t *testing.T) {
			setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{})
			previousCacheEnabled := common.MemoryCacheEnabled
			common.MemoryCacheEnabled = false
			t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })
			channel := Channel{Id: 13, Status: common.ChannelStatusEnabled, AutoProbeEnabled: common.GetPointer(true), ProbePeriodMinutes: 2}
			require.NoError(t, DB.Create(&channel).Error)
			state := ChannelProbeState{ChannelID: channel.Id, LastProbeAt: 50, LeaseUntil: 800}
			if existing {
				require.NoError(t, DB.Create(&state).Error)
			}

			require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusAutoDisabled, "upstream failure"))
			var count int64
			require.NoError(t, DB.Model(&ChannelProbeState{}).Where("channel_id = ?", channel.Id).Count(&count).Error)
			if !existing {
				assert.Zero(t, count)
				return
			}
			assert.Equal(t, int64(1), count)
			var after ChannelProbeState
			require.NoError(t, DB.First(&after, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, state, after)
		})
	}
}

func TestRemoveLegacyChannelProbePolicyColumnsPreservesUnifiedPolicy(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{})
	require.NoError(t, DB.Exec(`ALTER TABLE channels ADD COLUMN probe_failure_auto_ban boolean, ADD COLUMN probe_success_auto_enable boolean`).Error)
	channels := []Channel{
		{Id: 21, AutoProbeEnabled: common.GetPointer(true), AutoBan: common.GetPointer(0)},
		{Id: 22, AutoProbeEnabled: common.GetPointer(false), AutoBan: common.GetPointer(1)},
		{Id: 23},
	}
	require.NoError(t, DB.Create(&channels).Error)
	require.NoError(t, DB.Exec(`UPDATE channels SET probe_failure_auto_ban = true, probe_success_auto_enable = false`).Error)
	require.NoError(t, removeLegacyChannelProbePolicyColumns())
	require.NoError(t, removeLegacyChannelProbePolicyColumns())
	require.NoError(t, DB.AutoMigrate(&Channel{}))
	assert.False(t, DB.Migrator().HasColumn(&Channel{}, "probe_failure_auto_ban"))
	assert.False(t, DB.Migrator().HasColumn(&Channel{}, "probe_success_auto_enable"))
	var stored []Channel
	require.NoError(t, DB.Order("id").Find(&stored).Error)
	require.Len(t, stored, 3)
	assert.Equal(t, common.GetPointer(true), stored[0].AutoProbeEnabled)
	assert.Equal(t, common.GetPointer(false), stored[1].AutoProbeEnabled)
	assert.Nil(t, stored[2].AutoProbeEnabled)
	assert.False(t, stored[0].GetAutoBan())
	assert.True(t, stored[1].GetAutoBan())
	payload, err := common.Marshal(stored)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "probe_failure_auto_ban")
	assert.NotContains(t, string(payload), "probe_success_auto_enable")
}

func TestRemoveLegacyChannelProbePolicyColumnsHandlesConcurrentDrop(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{})
	require.NoError(t, DB.Exec(`ALTER TABLE channels ADD COLUMN probe_failure_auto_ban boolean, ADD COLUMN probe_success_auto_enable boolean`).Error)
	columns := []string{"probe_failure_auto_ban", "probe_success_auto_enable"}
	var dropped []string
	require.NoError(t, DB.Callback().Raw().Before("gorm:raw").Register("test:concurrent_probe_policy_drop", func(query *gorm.DB) {
		sql := query.Statement.SQL.String()
		if !strings.HasPrefix(sql, `ALTER TABLE "channels" DROP COLUMN `) {
			return
		}
		for _, column := range columns {
			if strings.HasSuffix(sql, `"`+column+`"`) {
				// 在存在性检查后、实际 DDL 前删列，确定性复现另一 master 抢先完成的窗口。
				// 直接使用连接执行，避免再次进入当前回调；两次 DDL 都由真实 PostgreSQL 执行。
				_, err := query.Statement.ConnPool.ExecContext(query.Statement.Context, `ALTER TABLE "channels" DROP COLUMN "`+column+`"`)
				query.AddError(err)
				dropped = append(dropped, column)
				return
			}
		}
	}))
	t.Cleanup(func() { assert.NoError(t, DB.Callback().Raw().Remove("test:concurrent_probe_policy_drop")) })

	require.NoError(t, removeLegacyChannelProbePolicyColumns())
	assert.Equal(t, columns, dropped)
	require.NoError(t, removeLegacyChannelProbePolicyColumns())
	assert.Equal(t, columns, dropped, "重复迁移不应再次尝试删除已移除列")
	for _, column := range columns {
		assert.False(t, DB.Migrator().HasColumn(&Channel{}, column))
	}
}

func TestChannelTestRecoveryUsesCurrentPolicyAndPreservesProbeState(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		enabled    *bool
		wantStatus int
	}{
		{"auto disabled", common.ChannelStatusAutoDisabled, common.GetPointer(true), common.ChannelStatusEnabled},
		{"manually disabled during request", common.ChannelStatusManuallyDisabled, common.GetPointer(true), common.ChannelStatusManuallyDisabled},
		{"switched off during request", common.ChannelStatusAutoDisabled, common.GetPointer(false), common.ChannelStatusAutoDisabled},
		{"unset", common.ChannelStatusAutoDisabled, nil, common.ChannelStatusAutoDisabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
			previousCache := common.MemoryCacheEnabled
			common.MemoryCacheEnabled = false
			t.Cleanup(func() { common.MemoryCacheEnabled = previousCache })
			channel := Channel{Id: 24, Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true)}
			require.NoError(t, DB.Create(&channel).Error)
			require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "gpt-4o"}).Error)
			state := ChannelProbeState{ChannelID: channel.Id, LastProbeAt: 500, LeaseUntil: 800}
			require.NoError(t, DB.Create(&state).Error)
			// 请求开始后修改状态/策略，恢复必须以最新持久化值为准。
			require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Updates(map[string]any{
				"status": tc.status, "auto_probe_enabled": tc.enabled,
			}).Error)
			stored, changed, err := RecoverChannelAfterTest(context.Background(), channel.Id, "")
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, stored.Status)
			assert.Equal(t, tc.status != tc.wantStatus, changed)
			_, changed, err = RecoverChannelAfterTest(context.Background(), channel.Id, "")
			require.NoError(t, err)
			assert.False(t, changed, "重复成功不能再次触发恢复通知")
			var ability Ability
			require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, tc.wantStatus == common.ChannelStatusEnabled, ability.Enabled)
			var after ChannelProbeState
			require.NoError(t, DB.First(&after, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, state, after)
			var count int64
			require.NoError(t, DB.Model(&ChannelProbeHistory{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestChannelProbeAndManualRecoveryPreserveMultiKeyIsolation(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		for _, tc := range []struct {
			name        string
			key         string
			statuses    map[int]int
			wantEnabled bool
		}{
			{"healthy key", "first", map[int]int{1: common.ChannelStatusAutoDisabled}, true},
			{"manually isolated sibling", "first", map[int]int{1: common.ChannelStatusManuallyDisabled}, true},
			{"successful key disabled in flight", "first", map[int]int{0: common.ChannelStatusAutoDisabled}, false},
			{"successful key manually disabled", "first", map[int]int{0: common.ChannelStatusManuallyDisabled}, false},
			{"all keys disabled", "first", map[int]int{0: common.ChannelStatusAutoDisabled, 1: common.ChannelStatusAutoDisabled}, false},
			{"key replaced in flight", "removed", nil, false},
			{"missing tested key", "", nil, false},
		} {
			t.Run(map[bool]string{true: "probe/", false: "manual/"}[automatic]+tc.name, func(t *testing.T) {
				setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
				previousCache := common.MemoryCacheEnabled
				common.MemoryCacheEnabled = false
				t.Cleanup(func() { common.MemoryCacheEnabled = previousCache })
				channel := Channel{Id: 25, Key: "first\nsecond", Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true),
					ChannelInfo: ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyStatusList: tc.statuses,
						MultiKeyDisabledReason: map[int]string{1: "isolated"}, MultiKeyDisabledTime: map[int]int64{1: 123}}}
				require.NoError(t, DB.Create(&channel).Error)
				require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "gpt-4o"}).Error)
				lease := common.GetTimestamp() + 300
				require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, LeaseUntil: lease}).Error)
				var changed bool
				var err error
				if automatic {
					_, changed, err = CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: true}, lease, tc.key)
				} else {
					_, changed, err = RecoverChannelAfterTest(context.Background(), channel.Id, tc.key)
				}
				require.NoError(t, err)
				assert.Equal(t, tc.wantEnabled, changed)
				var stored Channel
				require.NoError(t, DB.First(&stored, channel.Id).Error)
				assert.Equal(t, tc.wantEnabled, stored.Status == common.ChannelStatusEnabled)
				assert.Equal(t, channel.ChannelInfo, stored.ChannelInfo)
				var ability Ability
				require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
				assert.Equal(t, tc.wantEnabled, ability.Enabled)
			})
		}
	}
}

func TestChannelProbeFailureDisablesMultiKeyChannelWithoutChangingKeys(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
	previousCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCache })
	channel := Channel{Id: 26, Key: "first\nsecond", Status: common.ChannelStatusEnabled, AutoProbeEnabled: common.GetPointer(true),
		ChannelInfo: ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyStatusList: map[int]int{1: common.ChannelStatusManuallyDisabled}}}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "gpt-4o", Enabled: true}).Error)
	lease := common.GetTimestamp() + 300
	require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, LeaseUntil: lease}).Error)
	stored, changed, err := CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{ErrorMessage: "final failure"}, lease, "first")
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	assert.Equal(t, channel.ChannelInfo, stored.ChannelInfo)
	assert.Equal(t, "final failure", stored.GetOtherInfo()["status_reason"])
	var ability Ability
	require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
	assert.False(t, ability.Enabled)
}

func TestChannelRecoveryRollsBackStatusWhenAbilityUpdateFails(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		t.Run(map[bool]string{true: "probe", false: "manual"}[automatic], func(t *testing.T) {
			setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
			require.NoError(t, DB.Exec(`ALTER TABLE abilities ADD CONSTRAINT keep_disabled CHECK (NOT enabled)`).Error)
			channel := Channel{Id: 27, Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true)}
			require.NoError(t, DB.Create(&channel).Error)
			require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "gpt-4o"}).Error)
			state := ChannelProbeState{ChannelID: channel.Id, LeaseUntil: common.GetTimestamp() + 300}
			require.NoError(t, DB.Create(&state).Error)
			var changed bool
			var err error
			if automatic {
				_, changed, err = CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: true}, state.LeaseUntil, "")
			} else {
				_, changed, err = RecoverChannelAfterTest(context.Background(), channel.Id, "")
			}
			require.Error(t, err)
			assert.False(t, changed)
			require.NoError(t, DB.First(&channel, channel.Id).Error)
			assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
			assert.Empty(t, channel.OtherInfo)
			var after ChannelProbeState
			require.NoError(t, DB.First(&after, "channel_id = ?", channel.Id).Error)
			assert.Equal(t, state, after)
			var count int64
			require.NoError(t, DB.Model(&ChannelProbeHistory{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestChannelRecoveryRefreshesCachedRoutingStatus(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		t.Run(fmt.Sprint(automatic), func(t *testing.T) {
			setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{})
			previousCache := common.MemoryCacheEnabled
			previousChannels, previousGroups, previousConfigs := channelsIDM, group2model2channels, channel2advancedCustomConfig
			channelsIDM, group2model2channels, channel2advancedCustomConfig = nil, nil, nil
			common.MemoryCacheEnabled = true
			t.Cleanup(func() {
				common.MemoryCacheEnabled = previousCache
				channelsIDM, group2model2channels, channel2advancedCustomConfig = previousChannels, previousGroups, previousConfigs
			})
			channel := Channel{Id: 30, Key: "first", Group: "test", Models: "gpt-4o", Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true)}
			require.NoError(t, channel.Insert())
			SyncChannelCacheEntry(&channel)
			var changed bool
			var err error
			if automatic {
				lease := common.GetTimestamp() + 300
				require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, LeaseUntil: lease}).Error)
				_, changed, err = CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: true}, lease, "first")
			} else {
				_, changed, err = RecoverChannelAfterTest(context.Background(), channel.Id, "first")
			}
			require.NoError(t, err)
			require.True(t, changed)
			cached, err := CacheGetChannel(channel.Id)
			require.NoError(t, err)
			assert.Equal(t, common.ChannelStatusEnabled, cached.Status)
			eligible, err := GetEligibleChannels("test", "gpt-4o", "", nil)
			require.NoError(t, err)
			require.Len(t, eligible, 1)
			assert.Equal(t, channel.Id, eligible[0].Id)
		})
	}
}

func TestChannelPollingSnapshotCannotClearKeyIsolationBeforeRecovery(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{})
	previousCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCache })
	channel := Channel{Id: 28, Key: "first\nsecond", Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true),
		ChannelInfo: ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyMode: constant.MultiKeyModePolling}}
	require.NoError(t, DB.Create(&channel).Error)
	isolated := channel.ChannelInfo
	isolated.MultiKeyStatusList = map[int]int{0: common.ChannelStatusAutoDisabled, 1: common.ChannelStatusManuallyDisabled}
	isolated.MultiKeyDisabledReason = map[int]string{0: "isolated", 1: "manual"}
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Update("channel_info", isolated).Error)
	// 旧快照选 Key 后保存轮询游标，不能把当前所有 Key 的隔离抹掉。
	usingKey, _, apiErr := channel.GetNextEnabledKey()
	require.Nil(t, apiErr)
	stored, changed, err := RecoverChannelAfterTest(context.Background(), channel.Id, usingKey)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	assert.Equal(t, isolated.MultiKeyStatusList, stored.ChannelInfo.MultiKeyStatusList)
	assert.Equal(t, isolated.MultiKeyDisabledReason, stored.ChannelInfo.MultiKeyDisabledReason)
	assert.False(t, stored.HasEnabledKey())
}

func TestChannelRecoveryHoldsRowLockUntilStatusAndAbilitiesCommit(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		t.Run(fmt.Sprint(automatic), func(t *testing.T) {
			dsn := os.Getenv("TEST_POSTGRES_DSN")
			if dsn == "" {
				t.Skip("set TEST_POSTGRES_DSN to run isolated PostgreSQL lock tests")
			}
			db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
			require.NoError(t, err)
			pool, err := db.DB()
			require.NoError(t, err)
			pool.SetMaxOpenConns(1)
			other, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
			require.NoError(t, err)
			otherPool, err := other.DB()
			require.NoError(t, err)
			otherPool.SetMaxOpenConns(1)
			// 此测试需要两条连接，使用独立已提交 schema，结束时完整删除。
			schema := fmt.Sprintf("channel_recovery_lock_%d", time.Now().UnixNano())
			require.NoError(t, db.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
			previousDB, previousType, previousCache := DB, common.MainDatabaseType(), common.MemoryCacheEnabled
			t.Cleanup(func() {
				DB = previousDB
				common.SetMainDatabaseType(previousType)
				common.MemoryCacheEnabled = previousCache
				initCol()
				assert.NoError(t, db.Exec(`DROP SCHEMA "`+schema+`" CASCADE`).Error)
				_ = otherPool.Close()
				_ = pool.Close()
			})
			require.NoError(t, db.Exec(`SET search_path TO "`+schema+`"`).Error)
			require.NoError(t, other.Exec(`SET search_path TO "`+schema+`"`).Error)
			DB, common.MemoryCacheEnabled = db, false
			common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
			initCol()
			require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &ChannelProbeState{}, &ChannelProbeHistory{}))
			channel := Channel{Id: 29, Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true)}
			require.NoError(t, DB.Create(&channel).Error)
			require.NoError(t, DB.Create(&Ability{ChannelId: channel.Id, Group: "test", Model: "gpt-4o"}).Error)
			lease := common.GetTimestamp() + 300
			require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id, LeaseUntil: lease}).Error)
			readCurrent, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var once, releaseOnce sync.Once
			require.NoError(t, DB.Callback().Query().After("gorm:query").Register("test:hold_recovery_read", func(query *gorm.DB) {
				if query.Statement.Table == "channels" {
					once.Do(func() { close(readCurrent); <-release })
				}
			}))
			var recoveryErr error
			go func() {
				defer close(finished)
				if automatic {
					_, _, recoveryErr = CompleteChannelProbe(context.Background(), channel.Id, ChannelProbeHistory{Success: true}, lease, "")
				} else {
					_, _, recoveryErr = RecoverChannelAfterTest(context.Background(), channel.Id, "")
				}
			}()
			t.Cleanup(func() { releaseOnce.Do(func() { close(release) }); <-finished })
			select {
			case <-readCurrent:
			case <-time.After(5 * time.Second):
				t.Fatal("等待恢复读取当前状态超时")
			}
			// NOWAIT 确定性验证事务持有行锁，不用睡眠猜测并发时序。
			err = other.Exec("SELECT id FROM channels WHERE id = ? FOR UPDATE NOWAIT", channel.Id).Error
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			assert.Equal(t, "55P03", pgErr.Code)
			releaseOnce.Do(func() { close(release) })
			<-finished
			require.NoError(t, recoveryErr)
			require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusManuallyDisabled, "manual"))
			stored, changed, err := RecoverChannelAfterTest(context.Background(), channel.Id, "")
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
			var ability Ability
			require.NoError(t, DB.First(&ability, "channel_id = ?", channel.Id).Error)
			assert.False(t, ability.Enabled)
		})
	}
}

func TestListSystemTasksKeepsOldActiveDispatcherVisible(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &SystemTask{})
	require.NoError(t, DB.Create(&[]SystemTask{
		{ID: 1, TaskID: "probe-running", Type: SystemTaskTypeChannelProbe, Status: SystemTaskStatusRunning},
		{ID: 2, TaskID: "other-pending", Status: SystemTaskStatusPending},
		{ID: 3, TaskID: "other-done", Status: SystemTaskStatusSucceeded},
	}).Error)
	tasks, err := ListSystemTasks(2)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, "other-pending", tasks[0].TaskID)
	assert.Equal(t, "probe-running", tasks[1].TaskID)
}
