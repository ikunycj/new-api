package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func IsChannelEnabledForGroupModel(group string, modelName string, channelID int) bool {
	if group == "" || modelName == "" || channelID <= 0 {
		return false
	}
	if !common.MemoryCacheEnabled {
		return isChannelEnabledForGroupModelDB(group, modelName, channelID)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	if group2model2channels == nil {
		return false
	}

	if isChannelIDInList(group2model2channels[group][modelName], channelID) {
		return true
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if normalized != "" && normalized != modelName {
		return isChannelIDInList(group2model2channels[group][normalized], channelID)
	}
	return false
}

// GetConfiguredChannel returns a specific enabled channel when it supports the
// requested group and model. Routing policies use this path when they are
// configured with an explicit channel chain instead of the legacy pool tiers.
func GetConfiguredChannel(group string, modelName string, channelID int, excludedChannels map[int]struct{}, excludedClusters map[int]struct{}) (*Channel, error) {
	if channelID <= 0 {
		return nil, nil
	}
	if _, excluded := excludedChannels[channelID]; excluded {
		return nil, nil
	}
	channel, err := CacheGetChannel(channelID)
	if err != nil {
		return nil, err
	}
	if channel.Status != common.ChannelStatusEnabled || !IsChannelEnabledForGroupModel(group, modelName, channelID) {
		return nil, nil
	}
	if channel.ClusterId > 0 {
		if _, excluded := excludedClusters[channel.ClusterId]; excluded {
			return nil, nil
		}
	}
	if channel.Id <= 0 {
		return nil, fmt.Errorf("configured channel %d is invalid", channelID)
	}
	return channel, nil
}

func IsChannelEnabledForAnyGroupModel(groups []string, modelName string, channelID int) bool {
	if len(groups) == 0 {
		return false
	}
	for _, g := range groups {
		if IsChannelEnabledForGroupModel(g, modelName, channelID) {
			return true
		}
	}
	return false
}

func isChannelEnabledForGroupModelDB(group string, modelName string, channelID int) bool {
	var count int64
	err := DB.Model(&Ability{}).
		Where(commonGroupCol+" = ? and model = ? and channel_id = ? and enabled = ?", group, modelName, channelID, true).
		Count(&count).Error
	if err == nil && count > 0 {
		return true
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if normalized == "" || normalized == modelName {
		return false
	}
	count = 0
	err = DB.Model(&Ability{}).
		Where(commonGroupCol+" = ? and model = ? and channel_id = ? and enabled = ?", group, normalized, channelID, true).
		Count(&count).Error
	return err == nil && count > 0
}

func isChannelIDInList(list []int, channelID int) bool {
	for _, id := range list {
		if id == channelID {
			return true
		}
	}
	return false
}
