package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

const defaultUserGroup = model.DefaultUserGroup

func managedUserGroups() []string {
	groups, err := model.ListUserGroupNames()
	if err != nil || len(groups) == 0 {
		return []string{defaultUserGroup}
	}
	return groups
}

func isManagedUserGroup(group string) bool {
	group = strings.TrimSpace(group)
	if group == "" {
		return false
	}
	known, err := model.IsUserGroupName(group)
	if err != nil {
		return group == defaultUserGroup
	}
	return known
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
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": groups})
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
	if err := c.ShouldBindJSON(&req); err != nil {
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
	updatedName, err := model.UpdateUserGroupConfiguration(c.Param("name"), model.UserGroupUpdate{
		Name:             req.Name,
		TopupRatio:       req.TopupRatio,
		PricingGroups:    req.PricingGroups,
		PricingGroupsAll: req.PricingGroupsAll,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	group, err := model.GetUserGroupSummary(updatedName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, group)
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
	pricingGroups := service.GetUserGroupPricingGroups(userGroup)
	for order, groupName := range ratio_setting.GetPricingGroupOrder() {
		if _, ok := pricingGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": ratio_setting.GetGroupRatio(groupName),
				"desc":  groupName,
				"order": order,
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
