package controller

import (
	"net/http"
	"strings"

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
	if err = model.DB.Model(&model.Channel{}).Select("id", "type", "name", "base_url", "models", "model_mapping", "remark", "group", "status").Order("id desc").Find(&channels).Error; err != nil {
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
