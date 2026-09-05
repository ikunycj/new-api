package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type selfChannelView struct {
	Id           int     `json:"id"`
	Type         int     `json:"type"`
	Name         string  `json:"name"`
	BaseURL      *string `json:"base_url"`
	Models       string  `json:"models"`
	TestModel    *string `json:"test_model"`
	ModelMapping *string `json:"model_mapping"`
	Remark       *string `json:"remark"`
	Group        string  `json:"group"`
	Status       int     `json:"status"`
}

func requireToBUser(c *gin.Context) (*model.User, bool) {
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil || !user.IsToB() {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "仅 ToB 用户可查看分组渠道"})
		return nil, false
	}
	return user, true
}

func GetSelfChannels(c *gin.Context) {
	user, ok := requireToBUser(c)
	if !ok {
		return
	}
	usableGroups := service.GetUserUsableGroups(user.Group)
	pricingGroups, err := model.GetPricingGroupNames()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	allowedGroups := make(map[string]struct{})
	for _, group := range pricingGroups {
		if _, usable := usableGroups[group]; usable && model.IsBillingGroupToB(group) {
			allowedGroups[group] = struct{}{}
		}
	}
	var channels []selfChannelView
	if err = model.DB.Model(&model.Channel{}).Select("id", "type", "name", "base_url", "models", "test_model", "model_mapping", "remark", "group", "status").Order("id desc").Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	filtered := make([]selfChannelView, 0, len(channels))
	for _, channel := range channels {
		for _, group := range strings.Split(channel.Group, ",") {
			if _, allowed := allowedGroups[strings.TrimSpace(group)]; allowed {
				filtered = append(filtered, channel)
				break
			}
		}
	}
	common.ApiSuccess(c, filtered)
}

func GetSelfChannel(c *gin.Context) { GetSelfChannels(c) }

func TestSelfChannel(c *gin.Context) {
	user, ok := requireToBUser(c)
	if !ok {
		return
	}

	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.CacheGetChannel(channelID)
	if err != nil {
		channel, err = model.GetChannelById(channelID, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	usableGroups := service.GetUserUsableGroups(user.Group)
	pricingGroups, err := model.GetPricingGroupNames()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	allowedGroups := make(map[string]struct{})
	for _, group := range pricingGroups {
		if _, usable := usableGroups[group]; usable && model.IsBillingGroupToB(group) {
			allowedGroups[group] = struct{}{}
		}
	}
	channelAllowed := false
	for _, group := range strings.Split(channel.Group, ",") {
		if _, allowed := allowedGroups[strings.TrimSpace(group)]; allowed {
			channelAllowed = true
			break
		}
	}
	if !channelAllowed {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权测试该渠道"})
		return
	}

	tik := time.Now()
	result := testChannel(c.Request.Context(), channel, user.Id, c.Query("model"), c.Query("endpoint_type"), c.Query("stream") == "true")
	respondChannelTest(c, result, tik)
}

func respondChannelTest(c *gin.Context, result testResult, startedAt time.Time) {
	if result.localErr != nil {
		resp := gin.H{"success": false, "message": result.localErr.Error(), "time": 0.0}
		if result.newAPIError != nil {
			resp["error_code"] = result.newAPIError.GetErrorCode()
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	responseTime := time.Since(startedAt).Milliseconds()
	consumedTime := float64(responseTime) / 1000.0
	if result.newAPIError != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    result.newAPIError.Error(),
			"time":       consumedTime,
			"error_code": result.newAPIError.GetErrorCode(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "time": consumedTime})
}
