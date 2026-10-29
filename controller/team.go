package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// teamMemberView is the per-member payload returned by GET /api/team/members.
type teamMemberView struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	FeishuId    string `json:"feishu_id"`
	UsedQuota   int    `json:"used_quota"`
	Status      int    `json:"status"`
}

// GetTeamMembers lists all sub-accounts belonging to the calling parent account.
// GET /api/team/members
func GetTeamMembers(c *gin.Context) {
	parentId := c.GetInt("id")
	children, err := model.GetUserChildren(parentId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	views := make([]teamMemberView, 0, len(children))
	for _, u := range children {
		views = append(views, teamMemberView{
			Id:          u.Id,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Email:       u.Email,
			FeishuId:    u.FeishuId,
			UsedQuota:   u.UsedQuota,
			Status:      u.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": views})
}

// placeholder — following interfaces added in subsequent commits
var _ = strconv.Atoi
var _ = service.GetUserUsableGroups
var _ = setting.GetUsableGroupDescription
var _ = operation_setting.GetMaxUserTokens
var _ = middleware.AssertChildOf
