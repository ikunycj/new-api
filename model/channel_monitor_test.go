package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestChannelMonitor(pricingGroup string) *ChannelMonitor {
	return &ChannelMonitor{
		PricingGroup:             pricingGroup,
		TestModel:                "gpt-test",
		IntervalSeconds:          60,
		TimeoutSeconds:           15,
		Enabled:                  true,
		Visible:                  true,
		AvailabilityBoostPercent: 12.5,
	}
}

func TestChannelMonitorIsOwnedByUniquePricingGroup(t *testing.T) {
	truncateTables(t)

	monitor := newTestChannelMonitor("enterprise-pricing")
	require.NoError(t, CreateChannelMonitor(monitor))

	persisted, err := GetChannelMonitorByID(monitor.Id)
	require.NoError(t, err)
	assert.Equal(t, "enterprise-pricing", persisted.PricingGroup)
	assert.Equal(t, "gpt-test", persisted.TestModel)

	duplicate := newTestChannelMonitor("enterprise-pricing")
	require.Error(t, CreateChannelMonitor(duplicate))
}

func TestChannelMonitorAvailabilityAndRecentHistory(t *testing.T) {
	truncateTables(t)

	monitor := newTestChannelMonitor("availability-pricing")
	require.NoError(t, CreateChannelMonitor(monitor))

	results := []*ChannelMonitorHistory{
		{MonitorId: monitor.Id, Success: true, LatencyMs: 100, StatusCode: 200, CheckedAt: 1000},
		{MonitorId: monitor.Id, Success: false, LatencyMs: 200, StatusCode: 500, CheckedAt: 1100},
		{MonitorId: monitor.Id, Success: true, LatencyMs: 150, StatusCode: 200, CheckedAt: 1200},
	}
	for _, result := range results {
		require.NoError(t, DB.Create(result).Error)
	}

	availability, err := GetChannelMonitorAvailability(monitor.Id, 900)
	require.NoError(t, err)
	require.NotNil(t, availability)
	assert.Equal(t, 66.67, *availability)

	history, err := ListChannelMonitorHistory(monitor.Id, 2)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, int64(1200), history[0].CheckedAt)
	assert.Equal(t, int64(1100), history[1].CheckedAt)
}

func TestClaimDueChannelMonitorsOnlyClaimsOncePerLease(t *testing.T) {
	truncateTables(t)

	now := int64(2000)
	monitor := newTestChannelMonitor("lease-pricing")
	monitor.NextCheckAt = &now
	require.NoError(t, CreateChannelMonitor(monitor))

	first, err := ClaimDueChannelMonitors(now, 180, 10)
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, monitor.Id, first[0].Id)
	require.NotNil(t, first[0].LeaseExpiresAt)
	assert.Equal(t, int64(2180), *first[0].LeaseExpiresAt)

	second, err := ClaimDueChannelMonitors(now, 180, 10)
	require.NoError(t, err)
	assert.Empty(t, second)
}

func TestUpdateChannelMonitorSettings(t *testing.T) {
	truncateTables(t)

	monitor := newTestChannelMonitor("boost-pricing")
	require.NoError(t, CreateChannelMonitor(monitor))

	monitor.AvailabilityBoostPercent = 7.75
	require.NoError(t, UpdateChannelMonitor(monitor))

	updated, err := GetChannelMonitorByID(monitor.Id)
	require.NoError(t, err)
	assert.Equal(t, 7.75, updated.AvailabilityBoostPercent)
}

func TestDeleteChannelMonitorsOutsidePricingGroupsRemovesOwnedHistory(t *testing.T) {
	truncateTables(t)

	kept := newTestChannelMonitor("kept-pricing")
	removed := newTestChannelMonitor("removed-pricing")
	require.NoError(t, CreateChannelMonitor(kept))
	require.NoError(t, CreateChannelMonitor(removed))
	require.NoError(t, DB.Create(&[]ChannelMonitorHistory{
		{MonitorId: kept.Id, Success: true, CheckedAt: 1000},
		{MonitorId: removed.Id, Success: false, CheckedAt: 1000},
	}).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return DeleteChannelMonitorsOutsidePricingGroups(tx, []string{"kept-pricing"})
	}))

	_, err := GetChannelMonitorByID(kept.Id)
	require.NoError(t, err)
	_, err = GetChannelMonitorByID(removed.Id)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var keptHistoryCount int64
	require.NoError(t, DB.Model(&ChannelMonitorHistory{}).Where("monitor_id = ?", kept.Id).Count(&keptHistoryCount).Error)
	assert.Equal(t, int64(1), keptHistoryCount)
	var removedHistoryCount int64
	require.NoError(t, DB.Model(&ChannelMonitorHistory{}).Where("monitor_id = ?", removed.Id).Count(&removedHistoryCount).Error)
	assert.Zero(t, removedHistoryCount)
}

func TestUpdatePricingGroupConfigurationRemovesDeletedMonitorAndHistory(t *testing.T) {
	truncateTables(t)

	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	originalGroupEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	originalGroupOrder := ratio_setting.PricingGroupOrder2JSONString()
	originalRetryPolicies := ratio_setting.PricingGroupRetryPolicy2JSONString()
	originalStrategies := ratio_setting.PricingGroupRoutingStrategy2JSONString()
	t.Cleanup(func() {
		assert.NoError(t, updateOptionMap("GroupRatio", originalGroupRatios))
		assert.NoError(t, updateOptionMap("PricingGroupEnabled", originalGroupEnabled))
		assert.NoError(t, updateOptionMap("PricingGroupOrder", originalGroupOrder))
		assert.NoError(t, updateOptionMap("PricingGroupRetryPolicy", originalRetryPolicies))
		assert.NoError(t, updateOptionMap("PricingGroupRoutingStrategy", originalStrategies))
	})

	initialGroupRatios := `{"kept-pricing":1,"removed-pricing":1}`
	require.NoError(t, UpdatePricingGroupConfiguration(
		initialGroupRatios,
		`{"kept-pricing":true,"removed-pricing":true}`,
		`["kept-pricing","removed-pricing"]`,
		`{
			"kept-pricing":{"mode":"fixed","retry_times":3},
			"removed-pricing":{"mode":"fixed","retry_times":3}
			}`,
		`{}`,
	))

	kept := newTestChannelMonitor("kept-pricing")
	removed := newTestChannelMonitor("removed-pricing")
	require.NoError(t, CreateChannelMonitor(kept))
	require.NoError(t, CreateChannelMonitor(removed))
	require.NoError(t, DB.Create(&[]ChannelMonitorHistory{
		{MonitorId: kept.Id, Success: true, CheckedAt: 1000},
		{MonitorId: removed.Id, Success: false, CheckedAt: 1000},
	}).Error)

	updatedGroupRatios := `{"kept-pricing":1}`
	require.NoError(t, UpdatePricingGroupConfiguration(
		updatedGroupRatios,
		`{"kept-pricing":false}`,
		`["kept-pricing"]`,
		`{"kept-pricing":{"mode":"fixed","retry_times":3}}`,
		`{}`,
	))

	var stored Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", "GroupRatio").First(&stored).Error)
	assert.JSONEq(t, updatedGroupRatios, stored.Value)
	assert.True(t, ratio_setting.ContainsGroupRatio("kept-pricing"))
	assert.False(t, ratio_setting.IsPricingGroupEnabled("kept-pricing"))
	assert.False(t, ratio_setting.ContainsGroupRatio("removed-pricing"))

	_, err := GetChannelMonitorByID(kept.Id)
	require.NoError(t, err)
	_, err = GetChannelMonitorByID(removed.Id)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var keptHistoryCount int64
	require.NoError(t, DB.Model(&ChannelMonitorHistory{}).Where("monitor_id = ?", kept.Id).Count(&keptHistoryCount).Error)
	assert.Equal(t, int64(1), keptHistoryCount)
	var removedHistoryCount int64
	require.NoError(t, DB.Model(&ChannelMonitorHistory{}).Where("monitor_id = ?", removed.Id).Count(&removedHistoryCount).Error)
	assert.Zero(t, removedHistoryCount)
}

func TestClaimChannelMonitorUserTestEnforcesSharedCooldown(t *testing.T) {
	truncateTables(t)

	monitor := newTestChannelMonitor("cooldown-pricing")
	require.NoError(t, CreateChannelMonitor(monitor))

	claimed, err := ClaimChannelMonitorUserTest(monitor.Id, 100, 125)
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = ClaimChannelMonitorUserTest(monitor.Id, 110, 135)
	require.NoError(t, err)
	assert.False(t, claimed)

	require.NoError(t, CompleteChannelMonitorUserTest(monitor.Id, 120))
	claimed, err = ClaimChannelMonitorUserTest(monitor.Id, 119, 144)
	require.NoError(t, err)
	assert.False(t, claimed)

	claimed, err = ClaimChannelMonitorUserTest(monitor.Id, 120, 145)
	require.NoError(t, err)
	assert.True(t, claimed)
}
