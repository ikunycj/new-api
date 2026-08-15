package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupFailoverBindingTestDB(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Channel{}, &Cluster{}, &ClusterPool{}))
	previousDB := DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	DB = testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
	})
}

func TestSaveChannelFailoverBindingsUpdatesOnlyBindingFields(t *testing.T) {
	setupFailoverBindingTestDB(t)
	channels := []Channel{
		{Name: "IKUN plus", Key: "secret-a", Status: common.ChannelStatusEnabled, CreatedTime: 1},
		{Name: "IKUN free", Key: "secret-b", Status: common.ChannelStatusManuallyDisabled, CreatedTime: 1, ClusterId: 9, ClusterPoolId: 99},
	}
	require.NoError(t, DB.Create(&channels).Error)
	require.NoError(t, DB.Create(&Cluster{Id: 7, Name: "IKUN HK", Type: "ikun", Status: ClusterStatusEnabled}).Error)
	require.NoError(t, DB.Create(&ClusterPool{Id: 71, ClusterId: 7, Tier: PoolTierPremium, Name: "Pro/Plus", Status: ClusterStatusEnabled}).Error)

	err := SaveChannelFailoverBindings([]ChannelFailoverBindingUpdate{
		{ChannelId: channels[0].Id, ClusterId: 7, ClusterPoolId: 71},
		{ChannelId: channels[1].Id},
	})
	require.NoError(t, err)

	var stored []Channel
	require.NoError(t, DB.Order("id ASC").Find(&stored).Error)
	require.Len(t, stored, 2)
	assert.Equal(t, "secret-a", stored[0].Key)
	assert.Equal(t, 7, stored[0].ClusterId)
	assert.Equal(t, 71, stored[0].ClusterPoolId)
	assert.Equal(t, "IKUN free", stored[1].Name)
	assert.Zero(t, stored[1].ClusterId)
	assert.Zero(t, stored[1].ClusterPoolId)

	bindings, err := GetChannelFailoverBindings()
	require.NoError(t, err)
	require.Len(t, bindings, 2)
	assert.Equal(t, "IKUN plus", bindings[0].ChannelName)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, bindings[1].Status)
}

func TestSaveChannelFailoverBindingsRejectsInvalidRelationshipsAtomically(t *testing.T) {
	setupFailoverBindingTestDB(t)
	channel := Channel{Name: "IKUN", Key: "secret", Status: common.ChannelStatusEnabled, CreatedTime: 1}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&[]Cluster{
		{Id: 1, Name: "Cluster A", Type: "ikun", Status: ClusterStatusEnabled},
		{Id: 2, Name: "Cluster B", Type: "ikun", Status: ClusterStatusEnabled},
	}).Error)
	require.NoError(t, DB.Create(&ClusterPool{Id: 21, ClusterId: 2, Tier: PoolTierFree, Name: "Free", Status: ClusterStatusEnabled}).Error)

	tests := []struct {
		name     string
		bindings []ChannelFailoverBindingUpdate
		message  string
	}{
		{name: "partial binding", bindings: []ChannelFailoverBindingUpdate{{ChannelId: channel.Id, ClusterId: 1}}, message: "must set both"},
		{name: "duplicate channel", bindings: []ChannelFailoverBindingUpdate{{ChannelId: channel.Id}, {ChannelId: channel.Id}}, message: "duplicated"},
		{name: "pool from another cluster", bindings: []ChannelFailoverBindingUpdate{{ChannelId: channel.Id, ClusterId: 1, ClusterPoolId: 21}}, message: "does not belong"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := SaveChannelFailoverBindings(test.bindings)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
			var stored Channel
			require.NoError(t, DB.First(&stored, channel.Id).Error)
			assert.Zero(t, stored.ClusterId)
			assert.Zero(t, stored.ClusterPoolId)
		})
	}
}
