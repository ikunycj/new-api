package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

func managedUserGroups() []string {
	// Account groups are deliberately independent from pricing and routing
	// groups. Keep this contract stable so a pricing package can never appear
	// in the user editor or be accepted by the account API.
	return []string{"default", "vip"}
}

func normalizeManagedUserGroup(group string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(group)) {
	case "vip", "tob", "enterprise":
		return "vip", true
	case "default":
		return "default", true
	default:
		return "", false
	}
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
	pricingGroups, err := model.GetPricingGroupNames()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, groupName := range pricingGroups {
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
