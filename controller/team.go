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

// UpdateTeamTokenStatus enables or disables a token owned by a team member.
// PUT /api/team/token/status  body: {"id": <int>, "status": <1|2>}
func UpdateTeamTokenStatus(c *gin.Context) {
	parentId := c.GetInt("id")
	var body struct {
		Id     int `json:"id" binding:"required"`
		Status int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if body.Status != 1 && body.Status != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status 只能为 1（启用）或 2（停用）"})
		return
	}
	token, err := model.GetTokenById(body.Id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Token 不存在"})
		return
	}
	if !middleware.AssertChildOf(c, parentId, token.UserId) {
		return
	}
	token.Status = body.Status
	if err := token.SelectUpdate(); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, token.UserId, "team.token_status", map[string]interface{}{
		"token_id":   token.Id,
		"token_name": token.Name,
		"status":     body.Status,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// CreateTeamToken creates an API key under a team member's account.
// POST /api/team/token
// Body: user_id, name, group, remain_quota (-1 = unlimited), expired_time (-1 = never)
func CreateTeamToken(c *gin.Context) {
	parentId := c.GetInt("id")
	var body struct {
		UserId         int    `json:"user_id" binding:"required"`
		Name           string `json:"name" binding:"required"`
		Group          string `json:"group"`
		RemainQuota    int    `json:"remain_quota"`
		UnlimitedQuota bool   `json:"unlimited_quota"`
		ExpiredTime    int64  `json:"expired_time"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if !middleware.AssertChildOf(c, parentId, body.UserId) {
		return
	}
	if len(body.Name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Key 名称不能超过 50 个字符"})
		return
	}
	if !body.UnlimitedQuota {
		if body.RemainQuota < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "额度不能为负数"})
			return
		}
		maxQuota := int(1000000000 * common.QuotaPerUnit)
		if body.RemainQuota > maxQuota {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "额度超出最大值"})
			return
		}
	}

	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(body.UserId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该成员已达到最大 Key 数量限制"})
		return
	}

	// Validate the group against the member's own group.
	memberGroup, err := model.GetUserGroup(body.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if body.Group != "" {
		if err := service.ValidateTokenGroup(memberGroup, body.Group); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
			return
		}
	} else {
		body.Group = memberGroup
	}

	expiredTime := body.ExpiredTime
	if expiredTime == 0 {
		expiredTime = -1
	}

	key, err := common.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "生成 Key 失败"})
		return
	}

	token := model.Token{
		UserId:         body.UserId,
		Name:           body.Name,
		Key:            key,
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    expiredTime,
		RemainQuota:    body.RemainQuota,
		UnlimitedQuota: body.UnlimitedQuota,
		Group:          body.Group,
	}
	if err := token.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, body.UserId, "team.token_create", map[string]interface{}{
		"token_id":   token.Id,
		"token_name": token.Name,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":   token.Id,
			"name": token.Name,
			"key":  token.GetFullKey(),
		},
	})
}
