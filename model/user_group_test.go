package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserGroupCatalogAndDeletionRules(t *testing.T) {
	truncateTables(t)

	require.NoError(t, EnsureDefaultUserGroup())
	group, err := CreateUserGroup("enterprise")
	require.NoError(t, err)
	assert.Equal(t, "enterprise", group.Name)

	_, err = CreateUserGroup("enterprise")
	require.Error(t, err)
	assert.Equal(t, "用户分组已存在", err.Error())

	require.NoError(t, DB.Create(&User{
		Id:       9901,
		Username: "enterprise-user",
		Password: "password123",
		Group:    "enterprise",
		AffCode:  "enterprise-user",
	}).Error)

	summaries, err := ListUserGroupSummaries()
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, DefaultUserGroup, summaries[0].Name)
	assert.Equal(t, int64(0), summaries[0].UserCount)
	assert.Equal(t, "enterprise", summaries[1].Name)
	assert.Equal(t, int64(1), summaries[1].UserCount)

	err = DeleteUserGroup("enterprise")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能删除")

	require.NoError(t, DB.Delete(&User{}, 9901).Error)
	require.NoError(t, DeleteUserGroup("enterprise"))
	require.Error(t, DeleteUserGroup(DefaultUserGroup))
}

func TestUserGroupConfigurationUpdatesDefaultAndCustomGroups(t *testing.T) {
	truncateTables(t)
	previousTopupRatios := common.TopupGroupRatio2JSONString()
	previousPricingGroups := setting.UserGroupPricingGroups2JSONString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(previousTopupRatios))
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPricingGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":0.8}`))
	require.NoError(t, EnsureDefaultUserGroup())
	custom, err := CreateUserGroup("enterprise")
	require.NoError(t, err)

	selected := []string{"vip"}
	_, err = UpdateUserGroupConfiguration(DefaultUserGroup, UserGroupUpdate{
		TopupRatio:    float64Ptr(0.75),
		PricingGroups: &selected,
	})
	require.NoError(t, err)
	defaultSummary, err := GetUserGroupSummary(DefaultUserGroup)
	require.NoError(t, err)
	assert.Equal(t, 0.75, defaultSummary.TopupRatio)
	assert.Equal(t, []string{"vip"}, defaultSummary.PricingGroups)
	assert.False(t, defaultSummary.PricingGroupsAll)

	customSummary, err := GetUserGroupSummary(custom.Name)
	require.NoError(t, err)
	assert.Equal(t, float64(1), customSummary.TopupRatio)
	assert.True(t, customSummary.PricingGroupsAll)

	_, err = UpdateUserGroupConfiguration(custom.Name, UserGroupUpdate{
		TopupRatio:    float64Ptr(1.2),
		PricingGroups: &[]string{"vip"},
	})
	require.NoError(t, err)
	require.NoError(t, DeleteUserGroup(custom.Name))
	_, pricingConfigExists := setting.GetUserGroupPricingGroupsCopy()[custom.Name]
	assert.False(t, pricingConfigExists)
	assert.Equal(t, float64(1), common.GetTopupGroupRatio(custom.Name))
	require.Error(t, DeleteUserGroup(DefaultUserGroup))
}

func TestRenameUserGroupMovesUsersAndConfiguration(t *testing.T) {
	truncateTables(t)
	previousTopupRatios := common.TopupGroupRatio2JSONString()
	previousPricingGroups := setting.UserGroupPricingGroups2JSONString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(previousTopupRatios))
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPricingGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":0.8}`))
	require.NoError(t, EnsureDefaultUserGroup())
	_, err := CreateUserGroup("enterprise")
	require.NoError(t, err)
	require.NoError(t, DB.Create(&User{
		Id:       9911,
		Username: "renamed-group-user",
		Password: "password123",
		Group:    "enterprise",
		AffCode:  "renamed-group-user",
	}).Error)

	newName := "business"
	selected := []string{"vip"}
	updatedName, err := UpdateUserGroupConfiguration("enterprise", UserGroupUpdate{
		Name:          &newName,
		TopupRatio:    float64Ptr(1.25),
		PricingGroups: &selected,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updatedName)

	var user User
	require.NoError(t, DB.Unscoped().First(&user, 9911).Error)
	assert.Equal(t, newName, user.Group)
	oldGroupExists, err := IsUserGroupName("enterprise")
	require.NoError(t, err)
	assert.False(t, oldGroupExists)
	newGroupExists, err := IsUserGroupName(newName)
	require.NoError(t, err)
	assert.True(t, newGroupExists)

	ratioMap := make(map[string]float64)
	require.NoError(t, common.UnmarshalJsonStr(common.TopupGroupRatio2JSONString(), &ratioMap))
	_, oldRatioExists := ratioMap["enterprise"]
	assert.False(t, oldRatioExists)
	assert.Equal(t, 1.25, ratioMap[newName])
	pricingMap := setting.GetUserGroupPricingGroupsCopy()
	_, oldPricingExists := pricingMap["enterprise"]
	assert.False(t, oldPricingExists)
	assert.Equal(t, selected, pricingMap[newName])
	var ratioOption Option
	require.NoError(t, DB.First(&ratioOption, &Option{Key: "TopupGroupRatio"}).Error)
	persistedRatioMap := make(map[string]float64)
	require.NoError(t, common.UnmarshalJsonStr(ratioOption.Value, &persistedRatioMap))
	_, oldPersistedRatioExists := persistedRatioMap["enterprise"]
	assert.False(t, oldPersistedRatioExists)
	assert.Equal(t, 1.25, persistedRatioMap[newName])
	var pricingOption Option
	require.NoError(t, DB.First(&pricingOption, &Option{Key: "UserGroupPricingGroups"}).Error)
	persistedPricingMap := make(map[string][]string)
	require.NoError(t, common.UnmarshalJsonStr(pricingOption.Value, &persistedPricingMap))
	_, oldPersistedPricingExists := persistedPricingMap["enterprise"]
	assert.False(t, oldPersistedPricingExists)
	assert.Equal(t, selected, persistedPricingMap[newName])

	summary, err := GetUserGroupSummary(newName)
	require.NoError(t, err)
	assert.Equal(t, int64(1), summary.UserCount)
	assert.Equal(t, 1.25, summary.TopupRatio)
	assert.Equal(t, selected, summary.PricingGroups)
}

func TestRenameUserGroupRejectsReservedAndDuplicateNames(t *testing.T) {
	truncateTables(t)
	require.NoError(t, EnsureDefaultUserGroup())
	_, err := CreateUserGroup("enterprise")
	require.NoError(t, err)
	_, err = CreateUserGroup("business")
	require.NoError(t, err)

	duplicateName := "business"
	_, err = UpdateUserGroupConfiguration("enterprise", UserGroupUpdate{Name: &duplicateName})
	require.Error(t, err)
	assert.Equal(t, "用户分组已存在", err.Error())

	defaultRename := "standard"
	_, err = UpdateUserGroupConfiguration(DefaultUserGroup, UserGroupUpdate{Name: &defaultRename})
	require.Error(t, err)
	assert.Equal(t, "default 分组不能重命名", err.Error())
	defaultExists, err := IsUserGroupName(DefaultUserGroup)
	require.NoError(t, err)
	assert.True(t, defaultExists)
}

func TestUpdateUserGroupConfigurationPreservesPersistedOptionChanges(t *testing.T) {
	truncateTables(t)
	previousTopupRatios := common.TopupGroupRatio2JSONString()
	previousPricingGroups := setting.UserGroupPricingGroups2JSONString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(previousTopupRatios))
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPricingGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":0.8}`))
	require.NoError(t, EnsureDefaultUserGroup())
	_, err := CreateUserGroup("enterprise")
	require.NoError(t, err)
	_, err = CreateUserGroup("business")
	require.NoError(t, err)

	persistedRatios, err := common.Marshal(map[string]float64{
		DefaultUserGroup: 1,
		"enterprise":     1,
		"business":       1.4,
	})
	require.NoError(t, err)
	persistedPricingGroups, err := common.Marshal(map[string][]string{
		DefaultUserGroup: {setting.AllPricingGroups},
		"enterprise":     {setting.AllPricingGroups},
		"business":       {"vip"},
	})
	require.NoError(t, err)
	require.NoError(t, DB.Model(&Option{}).
		Where(commonKeyCol+" = ?", topupGroupRatioOption).
		Update("value", string(persistedRatios)).Error)
	require.NoError(t, DB.Model(&Option{}).
		Where(commonKeyCol+" = ?", userGroupPricingGroupsOption).
		Update("value", string(persistedPricingGroups)).Error)

	_, err = UpdateUserGroupConfiguration("enterprise", UserGroupUpdate{
		TopupRatio: float64Ptr(1.25),
	})
	require.NoError(t, err)

	var ratioOption Option
	require.NoError(t, DB.First(&ratioOption, &Option{Key: topupGroupRatioOption}).Error)
	actualRatios := make(map[string]float64)
	require.NoError(t, common.UnmarshalJsonStr(ratioOption.Value, &actualRatios))
	assert.Equal(t, 1.25, actualRatios["enterprise"])
	assert.Equal(t, 1.4, actualRatios["business"])

	var pricingOption Option
	require.NoError(t, DB.First(&pricingOption, &Option{Key: userGroupPricingGroupsOption}).Error)
	actualPricingGroups := make(map[string][]string)
	require.NoError(t, common.UnmarshalJsonStr(pricingOption.Value, &actualPricingGroups))
	assert.Equal(t, []string{"vip"}, actualPricingGroups["business"])
	assert.Equal(t, 1.4, common.GetTopupGroupRatio("business"))
	assert.Equal(t, []string{"vip"}, setting.GetUserGroupPricingGroups("business"))
}

func TestListUserGroupSummariesCountsActiveUsersByCurrentGroup(t *testing.T) {
	truncateTables(t)

	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	logSQLDB, err := logDB.DB()
	require.NoError(t, err)
	logSQLDB.SetMaxOpenConns(1)
	require.NoError(t, logDB.AutoMigrate(&Log{}))

	previousLogDB := LOG_DB
	LOG_DB = logDB
	t.Cleanup(func() {
		LOG_DB = previousLogDB
		require.NoError(t, logSQLDB.Close())
	})

	require.NoError(t, DB.Create(&[]UserGroup{
		{Name: DefaultUserGroup},
		{Name: "enterprise"},
	}).Error)
	users := []User{
		{Id: 1101, Username: "active-today", Password: "password123", Group: "enterprise", AffCode: "active-today"},
		{Id: 1102, Username: "active-month-boundary", Password: "password123", Group: "enterprise", AffCode: "active-month-boundary"},
		{Id: 1103, Username: "before-month-boundary", Password: "password123", Group: "enterprise", AffCode: "before-month-boundary"},
		{Id: 1104, Username: "at-tomorrow-boundary", Password: "password123", Group: "enterprise", AffCode: "at-tomorrow-boundary"},
		{Id: 1105, Username: "error-log-only", Password: "password123", Group: "enterprise", AffCode: "error-log-only"},
		{Id: 1106, Username: "deleted-active", Password: "password123", Group: "enterprise", AffCode: "deleted-active"},
		{Id: 1107, Username: "default-active", Password: "password123", Group: DefaultUserGroup, AffCode: "default-active"},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Delete(&users[5]).Error)

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.Local)
	todayStart, tomorrowStart, _, rollingMonthStart, _ := adminConsoleTimeBounds(now)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 1101, Type: LogTypeConsume, CreatedAt: todayStart + 1, Group: "pricing-group-not-user-group"},
		{UserId: 1101, Type: LogTypeConsume, CreatedAt: todayStart + 2, Group: "pricing-group-not-user-group"},
		{UserId: 1102, Type: LogTypeConsume, CreatedAt: rollingMonthStart, Group: DefaultUserGroup},
		{UserId: 1103, Type: LogTypeConsume, CreatedAt: rollingMonthStart - 1, Group: "enterprise"},
		{UserId: 1104, Type: LogTypeConsume, CreatedAt: tomorrowStart, Group: "enterprise"},
		{UserId: 1105, Type: LogTypeError, CreatedAt: todayStart + 1, Group: "enterprise"},
		{UserId: 1106, Type: LogTypeConsume, CreatedAt: todayStart + 1, Group: "enterprise"},
		{UserId: 1107, Type: LogTypeConsume, CreatedAt: todayStart + 1, Group: "unrelated-pricing-group"},
		{UserId: 9999, Type: LogTypeConsume, CreatedAt: todayStart + 1, Group: "enterprise"},
		{UserId: 0, Type: LogTypeConsume, CreatedAt: todayStart + 1, Group: "enterprise"},
	}).Error)

	summaries, err := listUserGroupSummariesAt(now)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	assert.Equal(t, DefaultUserGroup, summaries[0].Name)
	assert.Equal(t, int64(1), summaries[0].UserCount)
	assert.Equal(t, int64(1), summaries[0].ActiveToday)
	assert.Equal(t, int64(1), summaries[0].ActiveMonth)

	assert.Equal(t, "enterprise", summaries[1].Name)
	assert.Equal(t, int64(5), summaries[1].UserCount)
	assert.Equal(t, int64(1), summaries[1].ActiveToday)
	assert.Equal(t, int64(2), summaries[1].ActiveMonth)

	LOG_DB = nil
	enterprise, err := GetUserGroupSummary("enterprise")
	require.NoError(t, err)
	assert.Equal(t, int64(5), enterprise.UserCount)
	assert.Zero(t, enterprise.ActiveToday)
	assert.Zero(t, enterprise.ActiveMonth)
}
