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

func TestGetDistinctUserGroupsReturnsSortedActiveGroups(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&User{}))

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

	require.NoError(t, testDB.Create(&[]User{
		{Username: "group-vip", Password: "password", AffCode: "GROUPVIP", Group: "vip"},
		{Username: "group-default", Password: "password", AffCode: "GROUPDEFAULT", Group: "default"},
		{Username: "group-vip-2", Password: "password", AffCode: "GROUPVIP2", Group: "vip"},
		{Username: "deleted-group", Password: "password", AffCode: "GROUPDELETED", Group: "archived", DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}},
	}).Error)

	groups, err := GetDistinctUserGroups()
	require.NoError(t, err)
	assert.Equal(t, []string{"default", "vip"}, groups)
}
