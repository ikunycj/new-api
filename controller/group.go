package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

const defaultUserGroup = model.DefaultUserGroup

func managedUserGroups() []string {
	return []string{defaultUserGroup}
}

func isManagedUserGroup(group string) bool {
	return group == defaultUserGroup
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

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": ratio_setting.GetGroupRatio(groupName),
				"desc":  desc,
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
