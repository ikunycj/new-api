package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func requireToBUser(c *gin.Context) bool {
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil || !user.IsToB() {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "仅 ToB 用户可管理自己的渠道"})
		return false
	}
	return true
}

func selfChannelQuery(c *gin.Context) *gorm.DB {
	return model.DB.Where("owner_user_id = ?", c.GetInt("id"))
}

type selfChannelRequest struct {
	Type               int     `json:"type"`
	Name               string  `json:"name"`
	Key                string  `json:"key"`
	BaseURL            *string `json:"base_url"`
	Models             string  `json:"models"`
	ModelMapping       *string `json:"model_mapping"`
	OpenAIOrganization *string `json:"openai_organization"`
	Remark             *string `json:"remark"`
}

func (request selfChannelRequest) channel(ownerUserID int) model.Channel {
	return model.Channel{
		OwnerUserId:        ownerUserID,
		Type:               request.Type,
		Name:               strings.TrimSpace(request.Name),
		Key:                strings.TrimSpace(request.Key),
		BaseURL:            request.BaseURL,
		Models:             strings.TrimSpace(request.Models),
		ModelMapping:       request.ModelMapping,
		OpenAIOrganization: request.OpenAIOrganization,
		Remark:             request.Remark,
		Group:              "default",
		PriceMultiplier:    1,
		Status:             common.ChannelStatusManuallyDisabled,
	}
}

func validateSelfChannelRequest(request selfChannelRequest) error {
	if request.Name == "" {
		return errors.New("渠道名称不能为空")
	}
	if request.Key == "" {
		return errors.New("API Key 不能为空")
	}
	if request.Models == "" {
		return errors.New("模型列表不能为空")
	}
	return nil
}

func GetSelfChannels(c *gin.Context) {
	if !requireToBUser(c) {
		return
	}
	var channels []*model.Channel
	if err := selfChannelQuery(c).Order("id desc").Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, channels)
}

func GetSelfChannel(c *gin.Context) {
	if !requireToBUser(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var channel model.Channel
	if err = selfChannelQuery(c).Where("id = ?", id).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	common.ApiSuccess(c, channel)
}

func CreateSelfChannel(c *gin.Context) {
	if !requireToBUser(c) {
		return
	}
	var request selfChannelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateSelfChannelRequest(request); err != nil {
		common.ApiError(c, err)
		return
	}
	channel := request.channel(c.GetInt("id"))
	if err := model.BatchInsertChannels([]model.Channel{channel}); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	common.ApiSuccess(c, nil)
}

func UpdateSelfChannel(c *gin.Context) {
	if !requireToBUser(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request selfChannelRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err = validateSelfChannelRequest(request); err != nil {
		common.ApiError(c, err)
		return
	}
	patch := request.channel(c.GetInt("id"))
	if err = selfChannelQuery(c).Where("id = ?", id).Select(
		"type", "name", "key", "base_url", "models", "model_mapping", "openai_organization", "remark",
	).Updates(&patch).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	common.ApiSuccess(c, nil)
}

func DeleteSelfChannel(c *gin.Context) {
	if !requireToBUser(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if result := selfChannelQuery(c).Where("id = ?", id).Delete(&model.Channel{}); result.Error != nil {
		common.ApiError(c, result.Error)
		return
	}
	model.InitChannelCache()
	common.ApiSuccess(c, nil)
}
