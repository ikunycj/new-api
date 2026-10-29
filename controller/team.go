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

// GetTeamGroups returns the usable groups for a team member (or the parent
// itself when user_id is omitted). Sub-accounts share the same group
// eligibility rules as their parent, so we resolve against the queried user's
// own group assignment.
// GET /api/team/groups?user_id=<int>
func GetTeamGroups(c *gin.Context) {
	parentId := c.GetInt("id")
	targetId := parentId
	if raw := c.Query("user_id"); raw != "" {
		uid, err := strconv.Atoi(raw)
		if err != nil || uid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 user_id"})
			return
		}
		if !middleware.AssertChildOf(c, parentId, uid) {
			return
		}
		targetId = uid
	}

	userGroup, _ := model.GetUserGroup(targetId, false)
	isToBUser := false
	if user, err := model.GetUserById(targetId, false); err == nil {
		isToBUser = user.IsToB()
	}
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	pricingGroups, err := model.GetPricingGroupNames()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	usableGroups := make(map[string]map[string]interface{})
	for _, groupName := range pricingGroups {
		if isToBUser && !model.IsBillingGroupToB(groupName) {
			continue
		}
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
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": usableGroups})
}

// GetTeamTokens lists tokens owned by one member (user_id param) or all team
// members (no user_id). Includes tokens with no usage history.
// GET /api/team/tokens?user_id=<int>
func GetTeamTokens(c *gin.Context) {
	parentId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)

	if raw := c.Query("user_id"); raw != "" {
		uid, err := strconv.Atoi(raw)
		if err != nil || uid <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 user_id"})
			return
		}
		if !middleware.AssertChildOf(c, parentId, uid) {
			return
		}
		tokens, err := model.GetAllUserTokens(uid, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
		if err != nil {
			common.ApiError(c, err)
			return
		}
		total, _ := model.CountUserTokens(uid)
		pageInfo.SetTotal(int(total))
		pageInfo.SetItems(buildMaskedTokenResponses(tokens))
		common.ApiSuccess(c, pageInfo)
		return
	}

	// No user_id: list tokens for the whole team (parent + all children).
	children, err := model.GetUserChildren(parentId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userIds := make([]int, 0, len(children)+1)
	userIds = append(userIds, parentId)
	for _, ch := range children {
		userIds = append(userIds, ch.Id)
	}
	tokens, total, err := model.GetTokensByUserIds(userIds, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

// GetTeamTokenKey returns the plaintext key of a token owned by a team member.
// POST /api/team/token/:id/key
func GetTeamTokenKey(c *gin.Context) {
	parentId := c.GetInt("id")
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil || tokenId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 token id"})
		return
	}
	token, err := model.GetTokenById(tokenId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Token 不存在"})
		return
	}
	if !middleware.AssertChildOf(c, parentId, token.UserId) {
		return
	}
	recordManageAuditFor(c, token.UserId, "team.token_key_view", map[string]interface{}{
		"token_id":   token.Id,
		"token_name": token.Name,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"key": token.GetFullKey()}})
}

// placeholder — following interfaces added in subsequent commits
var _ = operation_setting.GetMaxUserTokens
