package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateUnifiedClusterBillingGroup(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&Channel{}, &Ability{}, &Cluster{}, &Token{}, &Option{}))
	previousDB := DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	DB = testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
	})

	groupRatio, err := common.Marshal(map[string]float64{
		"Cluster_1": 0.8,
		"Cluster_2": 0.9,
		"default":   1,
	})
	require.NoError(t, err)
	usableGroups, err := common.Marshal(map[string]string{
		"Cluster_1": "旧套餐 1",
		"Cluster_2": "旧套餐 2",
		"default":   "默认",
	})
	require.NoError(t, err)
	require.NoError(t, testDB.Create(&[]Option{
		{Key: "GroupRatio", Value: string(groupRatio)},
		{Key: "UserUsableGroups", Value: string(usableGroups)},
	}).Error)
	require.NoError(t, testDB.Create(&[]Cluster{
		{Id: 1, Name: "one", BillingGroup: "Cluster_1"},
		{Id: 2, Name: "two", BillingGroup: "Cluster_2"},
	}).Error)
	channels := []Channel{
		{Id: 11, Name: "one-key", Models: "gpt-5", Group: "Cluster_1,default", Status: common.ChannelStatusEnabled, ClusterId: 1},
		{Id: 12, Name: "two-key", Models: "gpt-5", Group: "Cluster_2", Status: common.ChannelStatusEnabled, ClusterId: 2},
	}
	require.NoError(t, testDB.Create(&channels).Error)
	token := Token{Id: 21, Group: "Cluster_2", DeletedAt: gorm.DeletedAt{Valid: true, Time: time.Now()}}
	require.NoError(t, token.SetGroupCandidates([]string{"Cluster_1", "default", "Cluster_2"}))
	require.NoError(t, testDB.Create(&token).Error)

	require.NoError(t, migrateUnifiedClusterBillingGroup())
	require.NoError(t, migrateUnifiedClusterBillingGroup())

	var storedClusters []Cluster
	require.NoError(t, testDB.Order("id ASC").Find(&storedClusters).Error)
	for _, cluster := range storedClusters {
		assert.Equal(t, UnifiedClusterBillingGroup, cluster.BillingGroup)
	}
	var storedChannels []Channel
	require.NoError(t, testDB.Order("id ASC").Find(&storedChannels).Error)
	assert.Equal(t, UnifiedClusterBillingGroup+",default", storedChannels[0].Group)
	assert.Equal(t, UnifiedClusterBillingGroup, storedChannels[1].Group)

	var abilities []Ability
	require.NoError(t, testDB.Order("channel_id ASC").Find(&abilities).Error)
	assert.Len(t, abilities, 3)
	assert.Equal(t, UnifiedClusterBillingGroup, abilities[0].Group)

	var storedToken Token
	require.NoError(t, testDB.Unscoped().First(&storedToken, token.Id).Error)
	assert.Equal(t, UnifiedClusterBillingGroup, storedToken.Group)
	candidates, err := storedToken.GetGroupCandidates()
	require.NoError(t, err)
	assert.Equal(t, []string{UnifiedClusterBillingGroup, "default"}, candidates)

	var ratioOption Option
	require.NoError(t, testDB.First(&ratioOption, "key = ?", "GroupRatio").Error)
	var ratios map[string]float64
	require.NoError(t, common.UnmarshalJsonStr(ratioOption.Value, &ratios))
	assert.Equal(t, 0.8, ratios[UnifiedClusterBillingGroup])
	assert.NotContains(t, ratios, "Cluster_1")
	assert.NotContains(t, ratios, "Cluster_2")

	var groupsOption Option
	require.NoError(t, testDB.First(&groupsOption, "key = ?", "UserUsableGroups").Error)
	var groups map[string]string
	require.NoError(t, common.UnmarshalJsonStr(groupsOption.Value, &groups))
	assert.Equal(t, "通用套餐", groups[UnifiedClusterBillingGroup])
	assert.NotContains(t, groups, "Cluster_1")
	assert.NotContains(t, groups, "Cluster_2")
}
