package controller

import (
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func managedUserGroups() []string {
	groupSet := map[string]struct{}{
		"default": {},
		"toB":     {},
	}
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupSet[groupName] = struct{}{}
	}
	groupNames := make([]string, 0, len(groupSet))
	for groupName := range groupSet {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)
	return groupNames
}

func GetGroups(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    managedUserGroups(),
	})
}

func GetPricingGroups(c *gin.Context) {
	groups, err := model.GetPricingGroupNames()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, groups)
}

func GetManagedUserGroups(c *gin.Context) {
	groups, err := model.ListUserGroupSummaries()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, groups)
}

type userGroupRequest struct {
	Name string `json:"name"`
}
type updateUserGroupRequest struct {
	Name             *string   `json:"name"`
	TopupRatio       *float64  `json:"topup_ratio"`
	PricingGroups    *[]string `json:"pricing_groups"`
	PricingGroupsAll *bool     `json:"pricing_groups_all"`
}

func CreateManagedUserGroup(c *gin.Context) {
	var req userGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		common.ApiErrorMsg(c, "用户分组名称不能为空")
		return
	}
	group, err := model.CreateUserGroup(req.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	summary, err := model.GetUserGroupSummary(group.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func UpdateManagedUserGroup(c *gin.Context) {
	var req updateUserGroupRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的用户分组配置")
		return
	}
	name, err := model.UpdateUserGroupConfiguration(c.Param("name"), model.UserGroupUpdate{
		Name: req.Name, TopupRatio: req.TopupRatio,
		PricingGroups: req.PricingGroups, PricingGroupsAll: req.PricingGroupsAll,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	summary, err := model.GetUserGroupSummary(name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func DeleteManagedUserGroup(c *gin.Context) {
	if err := model.DeleteUserGroup(c.Param("name")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	isToBUser := false
	if user, err := model.GetUserById(userId, false); err == nil {
		isToBUser = user.IsToB()
	}
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		if isToBUser && !model.IsBillingGroupToB(groupName) {
			continue
		}
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": service.GetUserGroupRatio(userGroup, groupName),
				"desc":  desc,
			}
		}
	}
	if !isToBUser {
		if _, ok := userUsableGroups["auto"]; ok {
			usableGroups["auto"] = map[string]interface{}{
				"ratio": "自动",
				"desc":  setting.GetUsableGroupDescription("auto"),
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
