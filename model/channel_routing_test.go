package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelRoutingTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &BillingGroupRoute{}, &BillingGroupChannel{}, &ChannelProbeState{}, &ChannelProbeHistory{}, &ChannelIQTestRun{}))
	for _, table := range []string{"channel_iq_test_runs", "channel_probe_histories", "channel_probe_states", "billing_group_channels", "billing_group_routes", "abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"channel_iq_test_runs", "channel_probe_histories", "channel_probe_states", "billing_group_channels", "billing_group_routes", "abilities", "channels"} {
			_ = DB.Exec("DELETE FROM " + table).Error
		}
		InitChannelRoutingCache()
	})
}

func TestResolveBillingGroupRouteLoadsEnabledChannels(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 17, BillingGroup: "claude", Name: "Claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&[]BillingGroupChannel{
		{Id: 1, BillingGroupRouteId: 17, ChannelId: 38, Priority: 1, Weight: 100, Enabled: true},
		{Id: 2, BillingGroupRouteId: 17, ChannelId: 40, Priority: 2, Weight: 50, Enabled: true},
		{Id: 3, BillingGroupRouteId: 17, ChannelId: 41, Priority: 3, Weight: 0, Enabled: false},
	}).Error)

	InitChannelRoutingCache()
	policy, channels, configured := ResolveBillingGroupRoute(" claude ")

	require.True(t, configured)
	assert.Equal(t, ratio_setting.DefaultPricingGroupRoutingStrategy(), policy.RoutingStrategy)
	require.Len(t, channels, 2)
	assert.Equal(t, 38, channels[0].ChannelId)
	assert.Equal(t, 40, channels[1].ChannelId)
}

func TestResolveBillingGroupRouteReturnsStrategyForUnconfiguredGroup(t *testing.T) {
	setupChannelRoutingTables(t)

	policy, channels, configured := ResolveBillingGroupRoute("missing")

	assert.False(t, configured)
	assert.Empty(t, channels)
	assert.Equal(t, ratio_setting.DefaultPricingGroupRoutingStrategy(), policy.RoutingStrategy)
}

func TestPricingGroupModelAllowlistCandidatesFollowEnabledRouteMembershipPostgres(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Channel{}, &Ability{}, &BillingGroupRoute{}, &BillingGroupChannel{})
	preservePricingGroupConfiguration(t)
	require.NoError(t, ratio_setting.UpdatePricingGroupModelAllowlistByJSONString(
		`{"default":["selected","unbound","zero-weight","disabled-channel"]}`,
	))
	channelRoutingLookup.Lock()
	previousRouting := channelRoutingLookup.value
	channelRoutingLookup.Unlock()
	t.Cleanup(func() {
		channelRoutingLookup.Lock()
		channelRoutingLookup.value = previousRouting
		channelRoutingLookup.Unlock()
	})

	require.NoError(t, DB.Create(&[]Channel{
		{Id: 1, Name: "packed", Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "unbound", Status: common.ChannelStatusEnabled},
		{Id: 3, Name: "zero-weight", Status: common.ChannelStatusEnabled},
		{Id: 4, Name: "forced", Status: common.ChannelStatusEnabled, ForcePriority: common.GetPointer(true)},
		{Id: 5, Name: "disabled", Status: common.ChannelStatusManuallyDisabled},
	}).Error)
	require.NoError(t, DB.Create(&[]Ability{
		{Group: "default", Model: "also-available", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "selected", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "selected", ChannelId: 2, Enabled: true},
		{Group: "default", Model: "unbound", ChannelId: 2, Enabled: true},
		{Group: "default", Model: "zero-weight", ChannelId: 3, Enabled: true},
		{Group: "default", Model: "forced", ChannelId: 4, Enabled: true},
		{Group: "default", Model: "disabled-channel", ChannelId: 5, Enabled: true},
		{Group: "default", Model: "disabled-ability", ChannelId: 1, Enabled: false},
		{Group: "another", Model: "another-group", ChannelId: 1, Enabled: true},
	}).Error)
	InitChannelRoutingCache()
	models, err := GetPricingGroupModelAllowlistCandidates("default")
	require.NoError(t, err)
	assert.Equal(t, []string{"also-available", "forced", "selected", "unbound", "zero-weight"}, models)
	visibleModels, err := GetGroupEnabledModels("default")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"selected", "unbound", "zero-weight"}, visibleModels)

	route := BillingGroupRoute{BillingGroup: "default", Enabled: true}
	require.NoError(t, DB.Create(&route).Error)
	require.NoError(t, DB.Create(&[]BillingGroupChannel{
		{BillingGroupRouteId: route.Id, ChannelId: 1, Weight: 10, Enabled: true},
		{BillingGroupRouteId: route.Id, ChannelId: 2, Weight: 10, Enabled: false},
		{BillingGroupRouteId: route.Id, ChannelId: 3, Weight: 0, Enabled: true},
		{BillingGroupRouteId: route.Id, ChannelId: 4, Weight: 0, Enabled: true},
		{BillingGroupRouteId: route.Id, ChannelId: 5, Weight: 10, Enabled: true},
	}).Error)
	InitChannelRoutingCache()
	models, err = GetPricingGroupModelAllowlistCandidates("default")
	require.NoError(t, err)
	assert.Equal(t, []string{"also-available", "forced", "selected"}, models)
	visibleModels, err = GetGroupEnabledModels("default")
	require.NoError(t, err)
	assert.Equal(t, []string{"selected"}, visibleModels)

	routeMember := DB.Model(&BillingGroupChannel{}).Where("billing_group_route_id = ? AND channel_id = ?", route.Id, 1)
	require.NoError(t, routeMember.Update("enabled", false).Error)
	InitChannelRoutingCache()
	visibleModels, err = GetGroupEnabledModels("default")
	require.NoError(t, err)
	assert.Empty(t, visibleModels)

	require.NoError(t, routeMember.Update("enabled", true).Error)
	InitChannelRoutingCache()
	visibleModels, err = GetGroupEnabledModels("default")
	require.NoError(t, err)
	assert.Equal(t, []string{"selected"}, visibleModels)

	channel := DB.Model(&Channel{}).Where("id = ?", 1)
	require.NoError(t, channel.Update("status", common.ChannelStatusManuallyDisabled).Error)
	visibleModels, err = GetGroupEnabledModels("default")
	require.NoError(t, err)
	assert.Empty(t, visibleModels)

	require.NoError(t, channel.Update("status", common.ChannelStatusEnabled).Error)
	visibleModels, err = GetGroupEnabledModels("default")
	require.NoError(t, err)
	assert.Equal(t, []string{"selected"}, visibleModels)

	models, err = GetPricingGroupModelAllowlistCandidates("empty")
	require.NoError(t, err)
	assert.Equal(t, []string{}, models)
}

func TestDeleteBoundChannelRequiresRemovingItFromRouting(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Key: "key", Group: "claude"}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "claude", Enabled: true}).Error)
	require.NoError(t, DB.Create(&BillingGroupChannel{Id: 1, BillingGroupRouteId: 9, ChannelId: 38, Enabled: true}).Error)

	err := (&Channel{Id: 38}).Delete()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by billing-group routing")

	require.NoError(t, DB.Delete(&BillingGroupChannel{}, 1).Error)
	require.NoError(t, (&Channel{Id: 38}).Delete())
	require.ErrorIs(t, DB.First(&Channel{}, 38).Error, gorm.ErrRecordNotFound)
}

func TestDeleteDisabledChannelRemovesOwnedChannelRecords(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Exec(`CREATE TABLE IF NOT EXISTS channel_cost_entries (
		id integer primary key,
		channel_id integer not null,
		start_at bigint not null,
		end_at bigint not null,
		amount_usd real not null,
		currency text not null,
		source text not null,
		note text,
		created_by integer not null,
		created_at bigint not null,
		updated_at bigint not null
	)`).Error)
	require.NoError(t, DB.Exec("DELETE FROM channel_cost_entries").Error)
	t.Cleanup(func() { _ = DB.Exec("DELETE FROM channel_cost_entries").Error })
	channel := Channel{Id: 39, Name: "Disabled", Key: "key", Group: "claude", Status: 2}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&Ability{Group: "claude", Model: "claude-test", ChannelId: channel.Id}).Error)
	require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id}).Error)
	require.NoError(t, DB.Create(&ChannelProbeHistory{ChannelID: channel.Id, CheckedAt: 1}).Error)
	require.NoError(t, DB.Create(&ChannelIQTestRun{ChannelID: channel.Id, TestModel: "claude-test", CompletedAt: 1}).Error)
	require.NoError(t, DB.Create(&ChannelCostEntry{ChannelId: channel.Id, StartAt: 1, EndAt: 2, AmountUSD: 1, Currency: "USD", Source: "manual"}).Error)

	deleted, err := DeleteDisabledChannel()
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var channelCount int64
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Count(&channelCount).Error)
	assert.Zero(t, channelCount)
	for name, value := range map[string]any{
		"ability":       &Ability{},
		"probe state":   &ChannelProbeState{},
		"probe history": &ChannelProbeHistory{},
		"IQ run":        &ChannelIQTestRun{},
		"cost entry":    &ChannelCostEntry{},
	} {
		var count int64
		require.NoError(t, DB.Model(value).Where("channel_id = ?", channel.Id).Count(&count).Error, name)
		assert.Zero(t, count, name)
	}
}
