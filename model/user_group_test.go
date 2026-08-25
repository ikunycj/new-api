package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	previousUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(previousTopupRatios))
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPricingGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousUsableGroups))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":0.8}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认","vip":"VIP"}`))
	require.NoError(t, EnsureDefaultUserGroup())
	custom, err := CreateUserGroup("enterprise")
	require.NoError(t, err)

	selected := []string{"vip"}
	require.NoError(t, UpdateUserGroupConfiguration(DefaultUserGroup, UserGroupUpdate{
		TopupRatio:    float64Ptr(0.75),
		PricingGroups: &selected,
	}))
	defaultSummary, err := GetUserGroupSummary(DefaultUserGroup)
	require.NoError(t, err)
	assert.Equal(t, 0.75, defaultSummary.TopupRatio)
	assert.Equal(t, []string{"vip"}, defaultSummary.PricingGroups)
	assert.False(t, defaultSummary.PricingGroupsAll)

	customSummary, err := GetUserGroupSummary(custom.Name)
	require.NoError(t, err)
	assert.Equal(t, float64(1), customSummary.TopupRatio)
	assert.True(t, customSummary.PricingGroupsAll)

	require.NoError(t, UpdateUserGroupConfiguration(custom.Name, UserGroupUpdate{
		TopupRatio:    float64Ptr(1.2),
		PricingGroups: &[]string{"vip"},
	}))
	require.NoError(t, DeleteUserGroup(custom.Name))
	_, pricingConfigExists := setting.GetUserGroupPricingGroupsCopy()[custom.Name]
	assert.False(t, pricingConfigExists)
	assert.Equal(t, float64(1), common.GetTopupGroupRatio(custom.Name))
	require.Error(t, DeleteUserGroup(DefaultUserGroup))
}
