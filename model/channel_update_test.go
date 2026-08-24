package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelUpdateTestDB(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Channel{}, &Ability{}))

	previousDB := DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	DB = testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		initCol()
	})
}

func TestChannelUpdateSelectedFieldsPersistsExplicitZeroValues(t *testing.T) {
	setupChannelUpdateTestDB(t)
	retries := 4
	channel := &Channel{
		Id:                               95001,
		Name:                             "keep-this-name",
		Key:                              "keep-this-key",
		Models:                           "model-a",
		Group:                            "pricing-a",
		Status:                           common.ChannelStatusEnabled,
		ProbeIntervalSeconds:             120,
		AutoDisabledProbeIntervalSeconds: 180,
		PriceMultiplier:                  2.5,
		PriceMultiplierMode:              ChannelPriceMultiplierModeCNY,
		UpstreamMaxRetries:               &retries,
	}
	require.NoError(t, DB.Create(channel).Error)

	channel.ProbeIntervalSeconds = 0
	channel.AutoDisabledProbeIntervalSeconds = 0
	channel.PriceMultiplier = 0
	channel.PriceMultiplierMode = ""
	channel.UpstreamMaxRetries = nil
	require.NoError(t, channel.Update(
		"probe_interval_seconds",
		"auto_disabled_probe_interval_seconds",
		"price_multiplier",
		"price_multiplier_mode",
		"upstream_max_retries",
	))

	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Zero(t, stored.ProbeIntervalSeconds)
	assert.Zero(t, stored.AutoDisabledProbeIntervalSeconds)
	assert.Zero(t, stored.PriceMultiplier)
	assert.Empty(t, stored.PriceMultiplierMode)
	assert.Nil(t, stored.UpstreamMaxRetries)
	assert.Equal(t, "keep-this-name", stored.Name)
	assert.Equal(t, "keep-this-key", stored.Key)
}

func TestChannelUpdatePersistsRecalculatedMultiKeyInfo(t *testing.T) {
	setupChannelUpdateTestDB(t)
	channel := &Channel{
		Id:     95002,
		Key:    "key-1\nkey-2",
		Models: "model-a",
		Group:  "pricing-a",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:         true,
			MultiKeySize:       2,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusEnabled, 1: common.ChannelStatusEnabled},
		},
	}
	require.NoError(t, DB.Create(channel).Error)

	channel.Key = "key-1"
	channel.ChannelInfo.MultiKeySize = 99
	require.NoError(t, channel.Update("key", "channel_info"))

	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, "key-1", stored.Key)
	assert.Equal(t, 1, stored.ChannelInfo.MultiKeySize)
	_, retainedRemovedKey := stored.ChannelInfo.MultiKeyStatusList[1]
	assert.False(t, retainedRemovedKey)
}

func TestChannelUpdateReconcilesMultiKeyAvailabilityWithAbilities(t *testing.T) {
	setupChannelUpdateTestDB(t)
	channel := &Channel{
		Id:     95003,
		Key:    "key-1\nkey-2",
		Models: "model-a",
		Group:  "pricing-a",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:         true,
			MultiKeySize:       2,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusManuallyDisabled},
		},
	}
	require.NoError(t, channel.Insert())

	channel.ChannelInfo.MultiKeyStatusList[1] = common.ChannelStatusManuallyDisabled
	channel.ReconcileMultiKeyAvailability(false)
	require.NoError(t, channel.Update("channel_info", "status", "other_info"))
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
	var ability Ability
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).First(&ability).Error)
	assert.False(t, ability.Enabled)

	delete(channel.ChannelInfo.MultiKeyStatusList, 0)
	channel.ReconcileMultiKeyAvailability(true)
	require.NoError(t, channel.Update("channel_info", "status", "other_info"))
	assert.Equal(t, common.ChannelStatusEnabled, channel.Status)
	ability = Ability{}
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).First(&ability).Error)
	assert.True(t, ability.Enabled)

	assert.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusAutoDisabled, "probe failed"))
	eligible, err := GetEligibleChannels("pricing-a", "model-a", "", nil)
	require.NoError(t, err)
	assert.Empty(t, eligible)
	ability = Ability{}
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).First(&ability).Error)
	assert.False(t, ability.Enabled)
}

func TestReconcileMultiKeyAvailabilityPreservesManualDisable(t *testing.T) {
	channel := &Channel{
		Key:    "key-1",
		Status: common.ChannelStatusManuallyDisabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:         true,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusManuallyDisabled},
		},
	}

	channel.ReconcileMultiKeyAvailability(false)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, channel.Status)
	delete(channel.ChannelInfo.MultiKeyStatusList, 0)
	channel.ReconcileMultiKeyAvailability(true)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, channel.Status)
}

func TestUpdateChannelStatusEnablesKeyWhileChannelIsAlreadyEnabled(t *testing.T) {
	setupChannelUpdateTestDB(t)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCacheEnabled })
	channel := &Channel{
		Id:     95006,
		Key:    "key-1\nkey-2",
		Models: "model-a",
		Group:  "pricing-a",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:         true,
			MultiKeySize:       2,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled},
		},
	}
	require.NoError(t, channel.Insert())

	assert.True(t, UpdateChannelStatus(channel.Id, "key-1", common.ChannelStatusEnabled, ""))
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	_, disabled := stored.ChannelInfo.MultiKeyStatusList[0]
	assert.False(t, disabled)
}

func TestUpdateChannelStatusDoesNotEnableChannelWithoutUsableKey(t *testing.T) {
	setupChannelUpdateTestDB(t)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = originalMemoryCacheEnabled })
	channel := &Channel{
		Id:     95007,
		Key:    "key-1",
		Models: "model-a",
		Group:  "pricing-a",
		Status: common.ChannelStatusAutoDisabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:         true,
			MultiKeySize:       1,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled},
		},
	}
	require.NoError(t, channel.Insert())

	assert.False(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, ""))
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
}

func TestUpdateChannelStatusRestoresMemoryCacheMembership(t *testing.T) {
	setupChannelUpdateTestDB(t)
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		group2model2channels = nil
		channelsIDM = nil
	})
	channel := &Channel{
		Id:     95008,
		Key:    "key-1",
		Models: "model-a",
		Group:  "pricing-a",
		Status: common.ChannelStatusAutoDisabled,
	}
	require.NoError(t, channel.Insert())
	InitChannelCache()
	channelsIDM[channel.Id].PreviousDayProbeSuccessRate = 42.5

	eligible, err := GetEligibleChannels("pricing-a", "model-a", "", nil)
	require.NoError(t, err)
	assert.Empty(t, eligible)
	assert.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, "manual operation"))
	eligible, err = GetEligibleChannels("pricing-a", "model-a", "", nil)
	require.NoError(t, err)
	require.Len(t, eligible, 1)
	assert.Equal(t, channel.Id, eligible[0].Id)
	assert.Equal(t, 42.5, eligible[0].PreviousDayProbeSuccessRate)

	var ability Ability
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).First(&ability).Error)
	assert.True(t, ability.Enabled)
}

func TestEditChannelByTagOnlyUpdatesOriginallyMatchedChannels(t *testing.T) {
	setupChannelUpdateTestDB(t)
	oldTag := "old"
	newTag := "new"
	zeroWeight := uint(0)
	require.NoError(t, DB.Create(&[]Channel{
		{Id: 95004, Tag: &oldTag, Models: "model-old", Group: "pricing-a", Status: common.ChannelStatusEnabled, Weight: common.GetPointer(uint(10))},
		{Id: 95005, Tag: &newTag, Models: "model-existing", Group: "pricing-a", Status: common.ChannelStatusEnabled, Weight: common.GetPointer(uint(20))},
	}).Error)
	require.NoError(t, DB.Create(&[]Ability{
		{Group: "pricing-a", Model: "model-old", ChannelId: 95004, Enabled: true, Weight: 10, Tag: &oldTag},
		{Group: "pricing-a", Model: "model-existing", ChannelId: 95005, Enabled: true, Weight: 20, Tag: &newTag},
	}).Error)

	models := "model-updated"
	require.NoError(t, EditChannelByTag(oldTag, &newTag, nil, &models, nil, &zeroWeight, nil, nil))

	var updated Channel
	require.NoError(t, DB.First(&updated, 95004).Error)
	assert.Equal(t, newTag, *updated.Tag)
	assert.Equal(t, models, updated.Models)
	require.NotNil(t, updated.Weight)
	assert.Zero(t, *updated.Weight)
	var updatedAbilities []Ability
	require.NoError(t, DB.Where("channel_id = ?", updated.Id).Find(&updatedAbilities).Error)
	require.Len(t, updatedAbilities, 1)
	assert.Equal(t, models, updatedAbilities[0].Model)
	assert.Zero(t, updatedAbilities[0].Weight)

	var existingAbilities []Ability
	require.NoError(t, DB.Where("channel_id = ?", 95005).Find(&existingAbilities).Error)
	require.Len(t, existingAbilities, 1)
	assert.Equal(t, "model-existing", existingAbilities[0].Model)
	assert.Equal(t, uint(20), existingAbilities[0].Weight)
}
