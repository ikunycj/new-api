package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
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

func setupClusterConfigurationTestDB(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Channel{}, &Ability{}, &Cluster{}, &ClusterPool{}, &FailoverPolicy{}, &FailoverGroupMember{}, &Option{}))
	previousDB := DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousGroups := setting.UserUsableGroups2JSONString()
	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	DB = testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousGroups))
		DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
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

func TestSaveClusterConfigurationCreatesBillingGroupPoolsAndOrderedRoutes(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	channels := []Channel{
		{Name: "free-key", Key: "key-a", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "plus-key", Key: "key-b", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "fallback-key", Key: "key-c", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&channels).Error)
	policy := FailoverPolicy{Name: "balanced", Mode: FailoverModeBalanced, Enabled: true, MaxPoolAttempts: 4}
	require.NoError(t, DB.Create(&policy).Error)

	config := &ClusterConfiguration{
		Name: "IKUN Hong Kong", Type: "IKUN", Status: ClusterStatusEnabled,
		BillingGroup: "cluster_hk", BillingGroupDescription: "Hong Kong billing", BillingGroupRatio: 1.25,
		PolicyId: policy.Id, FailoverPriority: 100,
		Routes: []ClusterRouteConfig{
			{ChannelId: channels[0].Id, PoolTier: 1, RouteOrder: 1, PoolName: "Free", CostFactor: 0.5, Weight: 100},
			{ChannelId: channels[1].Id, PoolTier: 2, RouteOrder: 2, PoolName: "Pro/Plus", CostFactor: 1, Weight: 80},
			{ChannelId: channels[2].Id, PoolTier: 3, RouteOrder: 3, PoolName: "Fallback", CostFactor: 1.5, Weight: 60},
		},
	}
	require.NoError(t, SaveClusterConfiguration(config))
	require.Positive(t, config.Id)

	var storedCluster Cluster
	require.NoError(t, DB.First(&storedCluster, config.Id).Error)
	assert.Equal(t, "ikun", storedCluster.Type)
	assert.Equal(t, "cluster_hk", storedCluster.BillingGroup)
	assert.Equal(t, policy.Id, storedCluster.PolicyId)

	var storedChannels []Channel
	require.NoError(t, DB.Order("id ASC").Find(&storedChannels).Error)
	require.Len(t, storedChannels, 3)
	for index, channel := range storedChannels {
		assert.Equal(t, config.Id, channel.ClusterId)
		assert.Equal(t, "cluster_hk", channel.Group)
		require.NotNil(t, channel.Priority)
		assert.Equal(t, int64((3-index)*100), *channel.Priority)
	}

	var abilities []Ability
	require.NoError(t, DB.Order("channel_id ASC").Find(&abilities).Error)
	require.Len(t, abilities, 3)
	for _, ability := range abilities {
		assert.Equal(t, "cluster_hk", ability.Group)
		assert.Equal(t, "gpt-5", ability.Model)
	}
	assert.Equal(t, 1.25, ratio_setting.GetGroupRatio("cluster_hk"))
	assert.Equal(t, "Hong Kong billing", setting.GetUsableGroupDescription("cluster_hk"))

	snapshot, err := GetClusterConfigurationSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot.Clusters, 1)
	require.Len(t, snapshot.Clusters[0].Routes, 3)
	assert.Equal(t, channels[0].Id, snapshot.Clusters[0].Routes[0].ChannelId)

	config.Status = ClusterStatusDisabled
	require.NoError(t, SaveClusterConfiguration(config))
	var pools []ClusterPool
	require.NoError(t, DB.Where("cluster_id = ?", config.Id).Find(&pools).Error)
	require.Len(t, pools, 3)
	for _, pool := range pools {
		assert.Equal(t, ClusterStatusDisabled, pool.Status)
	}
}

func TestSaveClusterConfigurationExpandsTwoRouteCluster(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	channels := []Channel{
		{Name: "pro-key", Key: "key-a", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "plus-key", Key: "key-b", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "fallback-key", Key: "key-c", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&channels).Error)

	config := &ClusterConfiguration{
		Name: "IKUN", Type: "ikun", Status: ClusterStatusEnabled,
		BillingGroup: "cluster_ikun", BillingGroupDescription: "IKUN billing", BillingGroupRatio: 1,
		Routes: []ClusterRouteConfig{
			{ChannelId: channels[0].Id, PoolTier: 1, RouteOrder: 1, PoolName: "Pro", CostFactor: 1, Weight: 100},
			{ChannelId: channels[1].Id, PoolTier: 2, RouteOrder: 2, PoolName: "Plus", CostFactor: 1.5, Weight: 100},
		},
	}
	require.NoError(t, SaveClusterConfiguration(config))

	config.Name = "IKUN updated"
	config.Routes = append(config.Routes, ClusterRouteConfig{
		ChannelId: channels[2].Id, PoolTier: 3, RouteOrder: 3,
		PoolName: "Fallback", CostFactor: 2, Weight: 100,
	})
	require.NoError(t, SaveClusterConfiguration(config))

	var storedCluster Cluster
	require.NoError(t, DB.First(&storedCluster, config.Id).Error)
	assert.Equal(t, "IKUN updated", storedCluster.Name)
	var storedChannels []Channel
	require.NoError(t, DB.Order("id ASC").Find(&storedChannels).Error)
	require.Len(t, storedChannels, 3)
	for index, channel := range storedChannels {
		assert.Equal(t, config.Id, channel.ClusterId)
		require.NotNil(t, channel.Priority)
		assert.Equal(t, int64((3-index)*100), *channel.Priority)
	}
}

func TestSaveClusterConfigurationReactivatesArchivedClusterWithNewRoutes(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	cluster := Cluster{
		Name: "Cluster1", Type: "ikun", Status: ClusterStatusDisabled,
		BillingGroup: "Cluster_1", FailoverPriority: 100, Archived: true,
	}
	require.NoError(t, DB.Create(&cluster).Error)
	for tier := 1; tier <= 4; tier++ {
		require.NoError(t, DB.Create(&ClusterPool{
			ClusterId: cluster.Id, Tier: tier, Name: defaultPoolName(tier),
			Status: ClusterStatusEnabled, CostFactor: float64(tier-1) * 0.5,
		}).Error)
	}
	oldChannels := []Channel{
		{Name: "old-pro", Key: "old-a", Models: "gpt-5.4-mini", Group: "Cluster_1", Status: common.ChannelStatusEnabled, ClusterId: cluster.Id, ClusterPoolId: 2},
		{Name: "old-plus", Key: "old-b", Models: "gpt-5.4-mini", Group: "Cluster_1", Status: common.ChannelStatusEnabled, ClusterId: cluster.Id, ClusterPoolId: 3},
	}
	newChannels := []Channel{
		{Name: "new-plus", Key: "new-a", Models: "gpt-5.4-mini", Group: "Cluster_1", Status: common.ChannelStatusEnabled},
		{Name: "new-pro", Key: "new-b", Models: "gpt-5.4-mini", Group: "Cluster_1", Status: common.ChannelStatusEnabled},
		{Name: "new-fallback", Key: "new-c", Models: "gpt-5.4-mini", Group: "Cluster_1", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&oldChannels).Error)
	require.NoError(t, DB.Create(&newChannels).Error)

	snapshot, err := GetClusterConfigurationSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot.Clusters, 1)
	config := &snapshot.Clusters[0]
	require.Len(t, config.Routes, 2)
	for index, route := range config.Routes {
		assert.Equal(t, index+1, route.PoolTier)
		assert.Equal(t, index+1, route.RouteOrder)
	}
	config.BillingGroupDescription = "Cluster 1 routes"
	config.Routes[0].ChannelId = newChannels[0].Id
	config.Routes[1].ChannelId = newChannels[1].Id
	config.Routes = append(config.Routes, ClusterRouteConfig{
		ChannelId: newChannels[2].Id, PoolTier: 3, RouteOrder: 3,
		PoolName: "Fallback", Weight: 100, CostFactor: 1.5,
	})
	require.NoError(t, SaveClusterConfiguration(config))

	var stored Cluster
	require.NoError(t, DB.First(&stored, cluster.Id).Error)
	assert.False(t, stored.Archived)
	for _, channel := range newChannels {
		require.NoError(t, DB.First(&channel, channel.Id).Error)
		assert.Equal(t, cluster.Id, channel.ClusterId)
	}
	for _, channel := range oldChannels {
		require.NoError(t, DB.First(&channel, channel.Id).Error)
		assert.Zero(t, channel.ClusterId)
		assert.Zero(t, channel.ClusterPoolId)
	}
}

func TestSaveClusterConfigurationRejectsUnsupportedRouteCount(t *testing.T) {
	config := &ClusterConfiguration{
		Name: "IKUN", Type: "ikun", Status: ClusterStatusEnabled,
		BillingGroup: "cluster_ikun", BillingGroupRatio: 1,
	}
	tests := []struct {
		name   string
		routes []ClusterRouteConfig
	}{
		{name: "no routes"},
		{name: "more than four routes", routes: make([]ClusterRouteConfig, 5)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config.Routes = test.routes
			err := SaveClusterConfiguration(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "between one and four")
		})
	}
}

func TestSaveClusterConfigurationRejectsMultiKeyAndExistingClusterBinding(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	channels := []Channel{
		{Name: "multi", Key: "a\nb", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled, ChannelInfo: ChannelInfo{IsMultiKey: true}},
		{Name: "bound", Key: "b", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled, ClusterId: 77},
		{Name: "normal", Key: "c", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "normal-2", Key: "d", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&channels).Error)

	tests := []struct {
		name      string
		channelID int
		message   string
	}{
		{name: "multi key", channelID: channels[0].Id, message: "multiple keys"},
		{name: "other cluster", channelID: channels[1].Id, message: "already belongs"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &ClusterConfiguration{
				Name: "invalid", Type: "ikun", Status: ClusterStatusEnabled,
				BillingGroup: "invalid_group", BillingGroupRatio: 1,
				Routes: []ClusterRouteConfig{
					{ChannelId: test.channelID, PoolTier: 1, RouteOrder: 1},
					{ChannelId: channels[2].Id, PoolTier: 2, RouteOrder: 2},
					{ChannelId: channels[3].Id, PoolTier: 3, RouteOrder: 3},
				},
			}
			err := SaveClusterConfiguration(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)

			var clusterCount int64
			require.NoError(t, DB.Model(&Cluster{}).Count(&clusterCount).Error)
			assert.Zero(t, clusterCount)
		})
	}
}

func TestClusterConfigurationIncludesArchivedClusterAndCanReclaimItsChannel(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	archivedCluster := Cluster{
		Id: 77, Name: "legacy", Type: "ikun", Status: ClusterStatusDisabled,
		BillingGroup: "legacy_group", Archived: true,
	}
	require.NoError(t, DB.Create(&archivedCluster).Error)
	legacyPool := ClusterPool{ClusterId: archivedCluster.Id, Tier: PoolTierFree, Name: "Free", Status: ClusterStatusDisabled}
	require.NoError(t, DB.Create(&legacyPool).Error)
	channels := []Channel{
		{Name: "legacy-key", Key: "a", Models: "gpt-5", Group: "legacy_group", Status: common.ChannelStatusEnabled, ClusterId: archivedCluster.Id, ClusterPoolId: legacyPool.Id},
		{Name: "plus-key", Key: "b", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "fallback-key", Key: "c", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&channels).Error)

	snapshot, err := GetClusterConfigurationSnapshot()
	require.NoError(t, err)
	require.Len(t, snapshot.Clusters, 1)
	assert.True(t, snapshot.Clusters[0].Archived)
	require.Len(t, snapshot.Clusters[0].Routes, 1)
	assert.Equal(t, channels[0].Id, snapshot.Clusters[0].Routes[0].ChannelId)

	config := &ClusterConfiguration{
		Name: "replacement", Type: "ikun", Status: ClusterStatusEnabled,
		BillingGroup: "replacement_group", BillingGroupRatio: 1,
		Routes: []ClusterRouteConfig{
			{ChannelId: channels[0].Id, PoolTier: 1, RouteOrder: 1},
			{ChannelId: channels[1].Id, PoolTier: 2, RouteOrder: 2},
			{ChannelId: channels[2].Id, PoolTier: 3, RouteOrder: 3},
		},
	}
	require.NoError(t, SaveClusterConfiguration(config))

	var reclaimed Channel
	require.NoError(t, DB.First(&reclaimed, channels[0].Id).Error)
	assert.Equal(t, config.Id, reclaimed.ClusterId)
	assert.Equal(t, "replacement_group", reclaimed.Group)
}

func TestSaveClusterConfigurationRejectsChannelFromActiveCluster(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	owner := Cluster{Name: "active", Type: "ikun", Status: ClusterStatusEnabled, BillingGroup: "active_group"}
	require.NoError(t, DB.Create(&owner).Error)
	channels := []Channel{
		{Name: "owned", Key: "a", Models: "gpt-5", Group: "active_group", Status: common.ChannelStatusEnabled, ClusterId: owner.Id},
		{Name: "plus-key", Key: "b", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
		{Name: "fallback-key", Key: "c", Models: "gpt-5", Group: "default", Status: common.ChannelStatusEnabled},
	}
	require.NoError(t, DB.Create(&channels).Error)
	config := &ClusterConfiguration{
		Name: "replacement", Type: "ikun", Status: ClusterStatusEnabled,
		BillingGroup: "replacement_group", BillingGroupRatio: 1,
		Routes: []ClusterRouteConfig{
			{ChannelId: channels[0].Id, PoolTier: 1, RouteOrder: 1},
			{ChannelId: channels[1].Id, PoolTier: 2, RouteOrder: 2},
			{ChannelId: channels[2].Id, PoolTier: 3, RouteOrder: 3},
		},
	}

	err := SaveClusterConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already belongs to active cluster")
	var clusterCount int64
	require.NoError(t, DB.Model(&Cluster{}).Count(&clusterCount).Error)
	assert.Equal(t, int64(1), clusterCount)
}

func TestDeleteClusterConfigurationArchivesClusterAndReleasesChannels(t *testing.T) {
	setupClusterConfigurationTestDB(t)
	cluster := Cluster{
		Name: "legacy", Type: "ikun", Status: ClusterStatusEnabled,
		BillingGroup: "legacy_group",
	}
	require.NoError(t, DB.Create(&cluster).Error)
	pools := []ClusterPool{
		{ClusterId: cluster.Id, Tier: PoolTierFree, Name: "Free", Status: ClusterStatusEnabled},
		{ClusterId: cluster.Id, Tier: PoolTierPremium, Name: "Pro/Plus", Status: ClusterStatusEnabled},
	}
	require.NoError(t, DB.Create(&pools).Error)
	channels := []Channel{
		{Name: "free-key", Key: "a", Models: "gpt-5", Group: "default,legacy_group", Status: common.ChannelStatusEnabled, ClusterId: cluster.Id, ClusterPoolId: pools[0].Id},
		{Name: "plus-key", Key: "b", Models: "gpt-5", Group: "legacy_group", Status: common.ChannelStatusEnabled, ClusterId: cluster.Id, ClusterPoolId: pools[1].Id},
	}
	require.NoError(t, DB.Create(&channels).Error)
	for index := range channels {
		require.NoError(t, channels[index].UpdateAbilities(DB))
	}
	require.NoError(t, DB.Create(&FailoverGroupMember{FailoverGroupId: 9, ClusterId: cluster.Id, Priority: 100}).Error)

	require.NoError(t, DeleteClusterConfiguration(cluster.Id))

	var storedCluster Cluster
	require.ErrorIs(t, DB.First(&Cluster{}, cluster.Id).Error, gorm.ErrRecordNotFound)
	require.NoError(t, DB.Unscoped().First(&storedCluster, cluster.Id).Error)
	assert.True(t, storedCluster.Archived)
	assert.Equal(t, ClusterStatusDisabled, storedCluster.Status)
	assert.True(t, storedCluster.DeletedAt.Valid)
	snapshot, err := GetClusterConfigurationSnapshot()
	require.NoError(t, err)
	assert.Empty(t, snapshot.Clusters)

	var storedPools []ClusterPool
	require.NoError(t, DB.Where("cluster_id = ?", cluster.Id).Find(&storedPools).Error)
	require.Len(t, storedPools, 2)
	for _, pool := range storedPools {
		assert.Equal(t, ClusterStatusDisabled, pool.Status)
	}

	var storedChannels []Channel
	require.NoError(t, DB.Order("id ASC").Find(&storedChannels).Error)
	require.Len(t, storedChannels, 2)
	for _, channel := range storedChannels {
		assert.Zero(t, channel.ClusterId)
		assert.Zero(t, channel.ClusterPoolId)
		assert.Equal(t, "default", channel.Group)
	}
	var abilities []Ability
	require.NoError(t, DB.Find(&abilities).Error)
	require.Len(t, abilities, 2)
	for _, ability := range abilities {
		assert.Equal(t, "default", ability.Group)
	}
	var memberCount int64
	require.NoError(t, DB.Model(&FailoverGroupMember{}).Where("cluster_id = ?", cluster.Id).Count(&memberCount).Error)
	assert.Zero(t, memberCount)
}
