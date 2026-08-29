package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

var group2model2channels map[string]map[string][]int // enabled channel
var channelsIDM map[int]*Channel                     // all channels include disabled

// channel2advancedCustomConfig caches parsed Advanced Custom (type 58) configs so
// path-aware selection avoids re-parsing JSON per request. Refreshed on full sync.
var channel2advancedCustomConfig map[int]*dto.AdvancedCustomConfig
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	InitChannelRoutingCache()
	if !common.MemoryCacheEnabled {
		InvalidatePricingCache()
		return
	}
	newChannelId2channel := make(map[int]*Channel)
	newChannel2advancedCustomConfig := make(map[int]*dto.AdvancedCustomConfig)
	var channels []*Channel
	DB.Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
		if channel.Type == constant.ChannelTypeAdvancedCustom {
			if config := channel.GetOtherSettings().AdvancedCustom; config != nil {
				newChannel2advancedCustomConfig[channel.Id] = config
			}
		}
	}
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool)
	for _, ability := range abilities {
		groups[ability.Group] = true
	}
	newGroup2model2channels := make(map[string]map[string][]int)
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]int)
	}
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue // skip disabled channels
		}
		groups := strings.Split(channel.Group, ",")
		for _, group := range groups {
			models := strings.Split(channel.Models, ",")
			for _, model := range models {
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]int, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel.Id)
			}
		}
	}

	// sort by priority
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return newChannelId2channel[channels[i]].GetPriority() > newChannelId2channel[channels[j]].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	//channelsIDM = newChannelId2channel
	for i, channel := range newChannelId2channel {
		if channel.ChannelInfo.IsMultiKey {
			channel.Keys = channel.GetKeys()
			if channel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
				if oldChannel, ok := channelsIDM[i]; ok {
					// 存在旧的渠道，如果是多key且轮询，保留轮询索引信息
					if oldChannel.ChannelInfo.IsMultiKey && oldChannel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
						channel.ChannelInfo.MultiKeyPollingIndex = oldChannel.ChannelInfo.MultiKeyPollingIndex
					}
				}
			}
		}
	}
	channelsIDM = newChannelId2channel
	channel2advancedCustomConfig = newChannel2advancedCustomConfig
	channelSyncLock.Unlock()
	// Lock ordering: InvalidatePricingCache acquires updatePricingLock, and
	// GetPricing (holding updatePricingLock) nests channelSyncLock.RLock via
	// loadPricingAdvancedCustomConfigs. channelSyncLock MUST be released before
	// invalidating the pricing cache, otherwise the reversed order deadlocks.
	InvalidatePricingCache()
	common.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func GetRandomSatisfiedChannel(group string, model string, retry int, requestPath string) (*Channel, error) {
	return GetRandomSatisfiedChannelExcluding(group, model, retry, requestPath, nil)
}

// GetRandomSatisfiedChannelExcluding selects a weighted channel while
// excluding channels already attempted for this request. This is used by the
// relay failover loop so a failed upstream is not selected again at the same
// priority.
func GetRandomSatisfiedChannelExcluding(group string, model string, retry int, requestPath string, excluded map[int]struct{}) (*Channel, error) {
	excludedChannels := excluded
	// if memory cache is disabled, get channel directly from database
	if !common.MemoryCacheEnabled {
		return GetChannelExcluding(group, model, retry, requestPath, excludedChannels)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	// First, try to find channels with the exact model name.
	channels := filterChannelsByRequestPathAndModel(group2model2channels[group][model], requestPath, model)

	// If no channels found, try to find channels with the normalized model name.
	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = filterChannelsByRequestPathAndModel(group2model2channels[group][normalizedModel], requestPath, model)
	}
	if len(excludedChannels) > 0 {
		filtered := make([]int, 0, len(channels))
		for _, channelID := range channels {
			if _, ok := excludedChannels[channelID]; ok {
				continue
			}
			filtered = append(filtered, channelID)
		}
		channels = filtered
	}

	if len(channels) == 0 {
		return nil, nil
	}

	if len(channels) == 1 {
		if channel, ok := channelsIDM[channels[0]]; ok {
			return channel, nil
		}
		return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channels[0])
	}

	uniquePriorities := make(map[int]bool)
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			uniquePriorities[int(channel.GetPriority())] = true
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}
	var sortedUniquePriorities []int
	for priority := range uniquePriorities {
		sortedUniquePriorities = append(sortedUniquePriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedUniquePriorities)))

	if retry >= len(uniquePriorities) {
		retry = len(uniquePriorities) - 1
	}
	targetPriority := int64(sortedUniquePriorities[retry])

	// get the priority for the given retry number
	var sumWeight = 0
	var targetChannels []*Channel
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			if channel.GetPriority() == targetPriority {
				sumWeight += channel.GetWeight()
				targetChannels = append(targetChannels, channel)
			}
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}

	if len(targetChannels) == 0 {
		return nil, errors.New(fmt.Sprintf("no channel found, group: %s, model: %s, priority: %d", group, model, targetPriority))
	}

	// smoothing factor and adjustment
	smoothingFactor := 1
	smoothingAdjustment := 0

	if sumWeight == 0 {
		// when all channels have weight 0, set sumWeight to the number of channels and set smoothing adjustment to 100
		// each channel's effective weight = 100
		sumWeight = len(targetChannels) * 100
		smoothingAdjustment = 100
	} else if sumWeight/len(targetChannels) < 10 {
		// when the average weight is less than 10, set smoothing factor to 100
		smoothingFactor = 100
	}

	// Calculate the total weight of all channels up to endIdx
	totalWeight := sumWeight * smoothingFactor

	// Generate a random value in the range [0, totalWeight)
	randomWeight := rand.Intn(totalWeight)

	// Find a channel based on its weight
	for _, channel := range targetChannels {
		randomWeight -= channel.GetWeight()*smoothingFactor + smoothingAdjustment
		if randomWeight < 0 {
			return channel, nil
		}
	}
	// return null if no channel is not found
	return nil, errors.New("channel not found")
}

// GetConfiguredRouteChannel selects the first eligible channel in the billing
// group's configured order. Legacy channel priority and weight do not affect
// configured ToB routes.
func GetConfiguredRouteChannel(group string, model string, requestPath string, entries []BillingGroupChannel, excluded map[int]struct{}) (*Channel, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	orderedEntries := append([]BillingGroupChannel(nil), entries...)
	SortRouteChannels(orderedEntries)
	if !common.MemoryCacheEnabled {
		channelIDs := make([]int, 0, len(orderedEntries))
		for _, entry := range orderedEntries {
			if !entry.Enabled {
				continue
			}
			if _, skip := excluded[entry.ChannelId]; skip {
				continue
			}
			channelIDs = append(channelIDs, entry.ChannelId)
		}
		if len(channelIDs) == 0 {
			return nil, nil
		}
		var abilities []Ability
		query := DB.Where(map[string]any{"group": group, "model": model, "enabled": true}).Where("channel_id IN ?", channelIDs)
		if err := query.Find(&abilities).Error; err != nil {
			return nil, err
		}
		if len(abilities) == 0 {
			normalizedModel := ratio_setting.FormatMatchingModelName(model)
			if normalizedModel != model {
				query = DB.Where(map[string]any{"group": group, "model": normalizedModel, "enabled": true}).Where("channel_id IN ?", channelIDs)
				if err := query.Find(&abilities).Error; err != nil {
					return nil, err
				}
			}
		}
		abilities = filterAbilitiesByRequestPathAndModel(abilities, requestPath, model)
		if len(abilities) == 0 {
			return nil, nil
		}
		eligibleIDs := make([]int, 0, len(abilities))
		for _, ability := range abilities {
			eligibleIDs = append(eligibleIDs, ability.ChannelId)
		}
		var channels []Channel
		if err := DB.Where("id IN ? AND status = ?", eligibleIDs, common.ChannelStatusEnabled).Find(&channels).Error; err != nil {
			return nil, err
		}
		channelByID := make(map[int]*Channel, len(channels))
		for i := range channels {
			channelByID[channels[i].Id] = &channels[i]
		}
		for _, entry := range orderedEntries {
			if channel := channelByID[entry.ChannelId]; channel != nil {
				return channel, nil
			}
		}
		return nil, nil
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	eligibleIDs := filterChannelsByRequestPathAndModel(group2model2channels[group][model], requestPath, model)
	if len(eligibleIDs) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		eligibleIDs = filterChannelsByRequestPathAndModel(group2model2channels[group][normalizedModel], requestPath, model)
	}
	eligible := make(map[int]struct{}, len(eligibleIDs))
	for _, channelID := range eligibleIDs {
		eligible[channelID] = struct{}{}
	}

	for _, entry := range orderedEntries {
		if !entry.Enabled {
			continue
		}
		if _, skip := excluded[entry.ChannelId]; skip {
			continue
		}
		if _, ok := eligible[entry.ChannelId]; !ok {
			continue
		}
		channel := channelsIDM[entry.ChannelId]
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		return channel, nil
	}
	return nil, nil
}

// FilterRealRouteChannels excludes channels reserved for managed mock tests
// from normal traffic. Mock requests never enter channel routing.
func FilterRealRouteChannels(entries []BillingGroupChannel) ([]BillingGroupChannel, error) {
	return filterRouteChannelsByMockSetting(entries, false)
}

func filterRouteChannelsByMockSetting(entries []BillingGroupChannel, mock bool) ([]BillingGroupChannel, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	channelsByID := make(map[int]*Channel, len(entries))
	if !common.MemoryCacheEnabled {
		channelIDs := make([]int, 0, len(entries))
		seen := make(map[int]struct{}, len(entries))
		for _, entry := range entries {
			if !entry.Enabled {
				continue
			}
			if _, ok := seen[entry.ChannelId]; ok {
				continue
			}
			seen[entry.ChannelId] = struct{}{}
			channelIDs = append(channelIDs, entry.ChannelId)
		}
		channels, err := GetChannelsByIds(channelIDs)
		if err != nil {
			return nil, err
		}
		for _, channel := range channels {
			channelsByID[channel.Id] = channel
		}
	}
	filtered := make([]BillingGroupChannel, 0, len(entries))
	for _, entry := range entries {
		if !entry.Enabled {
			continue
		}
		channel := channelsByID[entry.ChannelId]
		if common.MemoryCacheEnabled {
			var err error
			channel, err = CacheGetChannel(entry.ChannelId)
			if err != nil {
				return nil, err
			}
		}
		if channel != nil && channel.Status == common.ChannelStatusEnabled && channel.GetSetting().MockLoadTest == mock {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

// HasSatisfiedChannelExcluding reports whether at least one eligible channel
// remains after excluding the channels already attempted by a request.
func HasSatisfiedChannelExcluding(group string, model string, requestPath string, excluded map[int]struct{}) (bool, error) {
	excludedChannels := excluded
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelExcluding(group, model, 0, requestPath, excludedChannels)
		return channel != nil, err
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	channels := filterChannelsByRequestPathAndModel(group2model2channels[group][model], requestPath, model)
	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = filterChannelsByRequestPathAndModel(group2model2channels[group][normalizedModel], requestPath, model)
	}
	for _, channelID := range channels {
		if _, excluded := excludedChannels[channelID]; excluded {
			continue
		}
		if _, ok := channelsIDM[channelID]; ok {
			return true, nil
		}
	}
	return false, nil
}

// filterChannelsByRequestPathAndModel restricts candidates by request path and
// model. Only Advanced Custom (type 58) channels are path-checked: they are kept
// only when one of their configured routes matches requestPath and model. All
// other channel types always pass. When requestPath is empty, filtering is skipped.
// Caller must hold channelSyncLock (read lock). The cached slice is never mutated.
func filterChannelsByRequestPathAndModel(channels []int, requestPath string, model string) []int {
	if requestPath == "" || len(channels) == 0 {
		return channels
	}
	filtered := make([]int, 0, len(channels))
	for _, channelId := range channels {
		channel, ok := channelsIDM[channelId]
		if !ok {
			// keep it so the downstream consistency error is raised as before
			filtered = append(filtered, channelId)
			continue
		}
		if channel.Type != constant.ChannelTypeAdvancedCustom {
			filtered = append(filtered, channelId)
			continue
		}
		if config := channel2advancedCustomConfig[channelId]; config != nil && config.SupportsPathForModel(requestPath, model) {
			filtered = append(filtered, channelId)
		}
	}
	return filtered
}

func CacheGetChannel(id int) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetChannelById(id, true)
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return c, nil
}

func CacheGetChannelInfo(id int) (*ChannelInfo, error) {
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelById(id, true)
		if err != nil {
			return nil, err
		}
		return &channel.ChannelInfo, nil
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return &c.ChannelInfo, nil
}

func CacheUpdateChannelStatus(id int, status int) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel, ok := channelsIDM[id]; ok {
		channel.Status = status
	}
	if status != common.ChannelStatusEnabled {
		// delete the channel from group2model2channels
		for group, model2channels := range group2model2channels {
			for model, channels := range model2channels {
				for i, channelId := range channels {
					if channelId == id {
						// remove the channel from the slice
						group2model2channels[group][model] = append(channels[:i], channels[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func CacheUpdateChannel(channel *Channel) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	if channel == nil {
		channelSyncLock.Unlock()
		return
	}

	if channelsIDM == nil {
		channelsIDM = make(map[int]*Channel)
	}
	if oldChannel, ok := channelsIDM[channel.Id]; ok {
		logger.LogDebug(nil, "CacheUpdateChannel before: id=%d, name=%s, status=%d, polling_index=%d", channel.Id, channel.Name, channel.Status, oldChannel.ChannelInfo.MultiKeyPollingIndex)
	}
	channelsIDM[channel.Id] = channel
	if channel2advancedCustomConfig == nil {
		channel2advancedCustomConfig = make(map[int]*dto.AdvancedCustomConfig)
	}
	delete(channel2advancedCustomConfig, channel.Id)
	if channel.Type == constant.ChannelTypeAdvancedCustom {
		if config := channel.GetOtherSettings().AdvancedCustom; config != nil {
			channel2advancedCustomConfig[channel.Id] = config
		}
	}
	logger.LogDebug(nil, "CacheUpdateChannel after: id=%d, name=%s, status=%d, polling_index=%d", channel.Id, channel.Name, channel.Status, channel.ChannelInfo.MultiKeyPollingIndex)
	// Lock ordering: do NOT hold channelSyncLock while calling
	// InvalidatePricingCache. GetPricing acquires updatePricingLock first and then
	// channelSyncLock.RLock (via loadPricingAdvancedCustomConfigs); acquiring
	// updatePricingLock while holding channelSyncLock would be an AB-BA deadlock.
	channelSyncLock.Unlock()
	InvalidatePricingCache()
}
