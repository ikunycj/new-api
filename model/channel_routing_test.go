package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelRoutingTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &BillingGroupRoute{}, &BillingGroupChannel{}, &UpstreamErrorMapping{}, &ChannelProbeState{}, &ChannelProbeHistory{}))
	for _, table := range []string{"channel_probe_histories", "channel_probe_states", "channel_error_mappings", "billing_group_channels", "billing_group_routes", "abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"channel_probe_histories", "channel_probe_states", "channel_error_mappings", "billing_group_channels", "billing_group_routes", "abilities", "channels"} {
			_ = DB.Exec("DELETE FROM " + table).Error
		}
		InitChannelRoutingCache()
	})
}

func TestSaveChannelRoutingConfigPersistsOrderedChannelsAndRemovesMissingRows(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&[]Channel{{Id: 38, Name: "Pro", Group: "claude"}, {Id: 40, Name: "Official", Group: "claude"}}).Error)
	require.NoError(t, DB.Create(&BillingGroupRoute{Id: 9, BillingGroup: "old", Name: "old", Enabled: true}).Error)

	config := &ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: 17, BillingGroup: " claude ", Name: " Claude ", Enabled: true,
		}},
		RouteChannels: []BillingGroupChannel{
			{Id: 1, BillingGroupRouteId: 17, ChannelId: 38, Priority: 100, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 0.6},
			{Id: 2, BillingGroupRouteId: 17, ChannelId: 40, Priority: 90, Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 1.1},
		},
		ErrorMappings: []UpstreamErrorMapping{{
			Id: 1, ChannelId: 38, RawCode: " RATE_LIMIT_ERROR ", StatusCode: 429,
			StableCode: 204001, Category: "rate_limit", FailureScope: "channel", Action: "switch_channel", Retryable: true, Enabled: true,
		}},
	}

	require.NoError(t, SaveChannelRoutingConfig(config))
	InitChannelRoutingCache()
	policy, channels, ok := ResolveBillingGroupRoute("claude")
	require.True(t, ok)
	assert.Equal(t, 4, policy.MaxTotalAttempts)
	require.Len(t, channels, 2)
	assert.Equal(t, 38, channels[0].ChannelId)
	assert.Equal(t, 40, channels[1].ChannelId)
	assert.InDelta(t, 0.6, ResolveChannelCostFactor("claude", 38), 0.0001)

	var oldCount int64
	require.NoError(t, DB.Model(&BillingGroupRoute{}).Where("id = ?", 9).Count(&oldCount).Error)
	assert.Zero(t, oldCount)
}

func TestSaveChannelRoutingConfigRemapsTemporaryIDs(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Group: "claude"}).Error)

	config := &ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: -10, BillingGroup: "claude", Name: "Claude", Enabled: true,
		}},
		RouteChannels: []BillingGroupChannel{{
			Id: -20, BillingGroupRouteId: -10, ChannelId: 38, Priority: 100,
			Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 0.6,
		}},
	}

	require.NoError(t, SaveChannelRoutingConfig(config))
	require.Positive(t, config.Routes[0].Id)
	require.Positive(t, config.RouteChannels[0].Id)
	assert.Equal(t, config.Routes[0].Id, config.RouteChannels[0].BillingGroupRouteId)
}

func TestSaveChannelRoutingConfigRejectsChannelOutsideBillingGroup(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 38, Name: "Pro", Group: "default"}).Error)

	err := SaveChannelRoutingConfig(&ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{Id: -10, BillingGroup: "claude", Enabled: true}},
		RouteChannels: []BillingGroupChannel{{
			Id: -20, BillingGroupRouteId: -10, ChannelId: 38, Priority: 100,
			Weight: 100, MaxAttempts: 1, Enabled: true, CostFactor: 1,
		}},
	})

	require.EqualError(t, err, "route channel does not belong to its billing group")
}

func TestSaveChannelRoutingConfigRejectsEnabledRouteWithoutChannel(t *testing.T) {
	setupChannelRoutingTables(t)

	err := SaveChannelRoutingConfig(&ChannelRoutingConfig{
		Routes: []BillingGroupRoute{{
			Id: -10, BillingGroup: "claude", Enabled: true,
		}},
	})

	require.EqualError(t, err, "enabled billing group route requires an enabled channel")
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
	channel := Channel{Id: 39, Name: "Disabled", Key: "key", Group: "claude", Status: common.ChannelStatusManuallyDisabled}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&Ability{Group: "claude", Model: "claude-test", ChannelId: channel.Id}).Error)
	require.NoError(t, DB.Create(&ChannelProbeState{ChannelID: channel.Id}).Error)
	require.NoError(t, DB.Create(&ChannelProbeHistory{ChannelID: channel.Id, CheckedAt: 1}).Error)
	require.NoError(t, DB.Create(&UpstreamErrorMapping{ChannelId: channel.Id, RawCode: "test", StatusCode: 500}).Error)
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
		"error mapping": &UpstreamErrorMapping{},
		"cost entry":    &ChannelCostEntry{},
	} {
		var count int64
		require.NoError(t, DB.Model(value).Where("channel_id = ?", channel.Id).Count(&count).Error, name)
		assert.Zero(t, count, name)
	}
}

func TestMatchUpstreamErrorMappingPrefersExactChannel(t *testing.T) {
	setupChannelRoutingTables(t)
	require.NoError(t, DB.Create(&[]UpstreamErrorMapping{
		{Id: 1, RawCode: "*", StatusCode: 503, StableCode: 205002, Category: "upstream", FailureScope: "provider", Action: "switch_channel", Retryable: true, Enabled: true},
		{Id: 2, ChannelId: 38, ChannelType: 14, RawCode: "overloaded_error", StatusCode: 503, StableCode: 205004, Category: "upstream", FailureScope: "channel", Action: "switch_channel", Retryable: true, Enabled: true},
	}).Error)
	InitChannelRoutingCache()

	mapping, ok := MatchUpstreamErrorMapping(38, 14, "OVERLOADED_ERROR", 503)
	require.True(t, ok)
	assert.Equal(t, 205004, mapping.StableCode)
	assert.Equal(t, "channel", mapping.FailureScope)
}
