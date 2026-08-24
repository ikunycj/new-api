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
	if err := DB.Find(&channels).Error; err != nil {
		common.SysLog("failed to load channels while syncing cache: " + err.Error())
	}
	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel == nil || channel.Id <= 0 {
			continue
		}
		channelIDs = append(channelIDs, channel.Id)
		newChannelId2channel[channel.Id] = channel
		if channel.Type == constant.ChannelTypeAdvancedCustom {
			if config := channel.GetOtherSettings().AdvancedCustom; config != nil {
				newChannel2advancedCustomConfig[channel.Id] = config
			}
		}
	}
	if rates, err := GetPreviousDayChannelProbeSuccessRates(channelIDs, time.Now()); err == nil {
		for _, channel := range channels {
			if channel != nil {
				channel.PreviousDayProbeSuccessRate = rates[channel.Id]
			}
		}
	} else {
		common.SysLog("failed to load channel probe rates while syncing cache: " + err.Error())
	}
	var abilities []*Ability
	if err := DB.Find(&abilities).Error; err != nil {
		common.SysLog("failed to load channel abilities while syncing cache: " + err.Error())
	}
	newGroup2model2channels := make(map[string]map[string][]int)
	seen := make(map[string]map[string]map[int]struct{})
	for _, ability := range abilities {
		if ability == nil || !ability.Enabled || ability.ChannelId <= 0 {
			continue
		}
		channel, ok := newChannelId2channel[ability.ChannelId]
		if !ok || channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		group := strings.TrimSpace(ability.Group)
		modelName := strings.TrimSpace(ability.Model)
		if group == "" || modelName == "" {
			continue
		}
		if _, ok := newGroup2model2channels[group]; !ok {
			newGroup2model2channels[group] = make(map[string][]int)
			seen[group] = make(map[string]map[int]struct{})
		}
		if _, ok := newGroup2model2channels[group][modelName]; !ok {
			newGroup2model2channels[group][modelName] = make([]int, 0)
			seen[group][modelName] = make(map[int]struct{})
		}
		if _, exists := seen[group][modelName][ability.ChannelId]; exists {
			continue
		}
		seen[group][modelName][ability.ChannelId] = struct{}{}
		newGroup2model2channels[group][modelName] = append(newGroup2model2channels[group][modelName], ability.ChannelId)
	}
	for _, model2channels := range newGroup2model2channels {
		for _, channelIDs := range model2channels {
			sort.Ints(channelIDs)
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

// GetEligibleChannels returns the complete enabled candidate set for a
// pricing-group/model/path combination. The order is stable by channel ID;
// dynamic routing policies apply their ranking after this capability lookup.
func GetEligibleChannels(group string, modelName string, requestPath string, excluded map[int]struct{}) ([]*Channel, error) {
	if !common.MemoryCacheEnabled {
		return getEligibleChannelsFromDB(group, modelName, requestPath, excluded)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	channelIDs := filterChannelsByRequestPathAndModel(group2model2channels[group][modelName], requestPath, modelName)
	if len(channelIDs) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(modelName)
		channelIDs = filterChannelsByRequestPathAndModel(group2model2channels[group][normalizedModel], requestPath, modelName)
	}
	channels := make([]*Channel, 0, len(channelIDs))
	seen := make(map[int]struct{}, len(channelIDs))
	for _, channelID := range channelIDs {
		if _, exists := seen[channelID]; exists {
			continue
		}
		seen[channelID] = struct{}{}
		if _, skip := excluded[channelID]; skip {
			continue
		}
		channel, ok := channelsIDM[channelID]
		if !ok || channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		channels = append(channels, channel)
	}
	sort.SliceStable(channels, func(i, j int) bool { return channels[i].Id < channels[j].Id })
	return channels, nil
}

func getEligibleChannelsFromDB(group string, modelName string, requestPath string, excluded map[int]struct{}) ([]*Channel, error) {
	var abilities []Ability
	groupColumn := commonGroupCol
	if groupColumn == "" {
		groupColumn = "`group`"
	}
	query := DB.Where(groupColumn+" = ? AND model = ? AND enabled = ?", group, modelName, true)
	if err := query.Find(&abilities).Error; err != nil {
		return nil, err
	}
	if len(abilities) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(modelName)
		if normalizedModel != modelName {
			if err := DB.Where(groupColumn+" = ? AND model = ? AND enabled = ?", group, normalizedModel, true).Find(&abilities).Error; err != nil {
				return nil, err
			}
		}
	}
	abilities = filterAbilitiesByRequestPathAndModel(abilities, requestPath, modelName)
	ids := make([]int, 0, len(abilities))
	seen := make(map[int]struct{}, len(abilities))
	for _, ability := range abilities {
		if _, exists := seen[ability.ChannelId]; exists {
			continue
		}
		seen[ability.ChannelId] = struct{}{}
		ids = append(ids, ability.ChannelId)
	}
	if len(ids) == 0 {
		return []*Channel{}, nil
	}
	var loaded []*Channel
	if err := DB.Where("id IN ? AND status = ?", ids, common.ChannelStatusEnabled).Find(&loaded).Error; err != nil {
		return nil, err
	}
	byID := make(map[int]*Channel, len(loaded))
	for _, channel := range loaded {
		byID[channel.Id] = channel
	}
	result := make([]*Channel, 0, len(ids))
	for _, id := range ids {
		if _, skip := excluded[id]; skip {
			continue
		}
		if channel := byID[id]; channel != nil {
			result = append(result, channel)
		}
	}
	rateIDs := make([]int, 0, len(result))
	for _, channel := range result {
		rateIDs = append(rateIDs, channel.Id)
	}
	if rates, err := GetPreviousDayChannelProbeSuccessRates(rateIDs, time.Now()); err == nil {
		for _, channel := range result {
			channel.PreviousDayProbeSuccessRate = rates[channel.Id]
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Id < result[j].Id })
	return result, nil
}

// GetRandomSatisfiedChannelExcluding selects a weighted channel while
// excluding channels already attempted for this request.
func GetRandomSatisfiedChannelExcluding(group string, model string, retry int, requestPath string, excluded map[int]struct{}) (*Channel, error) {
	channels, err := GetEligibleChannels(group, model, requestPath, excluded)
	if err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return nil, nil
	}
	if len(channels) == 1 {
		return channels[0], nil
	}

	// Dynamic routing consumes the full candidate set; this helper only applies
	// weighted selection.
	targetChannels := channels
	var sumWeight int
	for _, channel := range targetChannels {
		sumWeight += channel.GetWeight() + 10
	}
	if sumWeight <= 0 {
		return nil, errors.New("channel weight is invalid")
	}

	randomWeight := rand.Intn(sumWeight)

	// Find a channel based on its weight
	for _, channel := range targetChannels {
		randomWeight -= channel.GetWeight() + 10
		if randomWeight < 0 {
			return channel, nil
		}
	}
	// return null if no channel is not found
	return nil, errors.New("channel not found")
}

// GetConfiguredRouteChannel selects only from the channels configured for a
// billing group. Route priority remains a pricing-group route property;
// weights apply among entries at the same route priority.
func GetConfiguredRouteChannel(group string, model string, requestPath string, entries []BillingGroupChannel, excluded map[int]struct{}) (*Channel, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	if !common.MemoryCacheEnabled {
		channelIDs := make([]int, 0, len(entries))
		entryByChannel := make(map[int]BillingGroupChannel, len(entries))
		for _, entry := range entries {
			if !entry.Enabled {
				continue
			}
			if _, skip := excluded[entry.ChannelId]; skip {
				continue
			}
			channelIDs = append(channelIDs, entry.ChannelId)
			entryByChannel[entry.ChannelId] = entry
		}
		if len(channelIDs) == 0 {
			return nil, nil
		}
		var abilities []Ability
		query := DB.Where(&Ability{Group: group, Model: model, Enabled: true}).Where("channel_id IN ?", channelIDs)
		if err := query.Find(&abilities).Error; err != nil {
			return nil, err
		}
		if len(abilities) == 0 {
			normalizedModel := ratio_setting.FormatMatchingModelName(model)
			if normalizedModel != model {
				query = DB.Where(&Ability{Group: group, Model: normalizedModel, Enabled: true}).Where("channel_id IN ?", channelIDs)
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
		bestPriority := 0
		hasPriority := false
		totalWeight := 0
		candidates := make([]BillingGroupChannel, 0, len(channels))
		for channelID := range channelByID {
			entry := entryByChannel[channelID]
			if !hasPriority || entry.Priority > bestPriority {
				bestPriority = entry.Priority
				hasPriority = true
				candidates = candidates[:0]
				totalWeight = 0
			}
			if entry.Priority != bestPriority {
				continue
			}
			weight := entry.Weight
			if weight <= 0 {
				weight = 100
			}
			totalWeight += weight
			candidates = append(candidates, entry)
		}
		if len(candidates) == 0 {
			return nil, nil
		}
		selected := rand.Intn(totalWeight)
		for _, entry := range candidates {
			weight := entry.Weight
			if weight <= 0 {
				weight = 100
			}
			selected -= weight
			if selected < 0 {
				return channelByID[entry.ChannelId], nil
			}
		}
		return channelByID[candidates[len(candidates)-1].ChannelId], nil
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

	bestPriority := 0
	hasPriority := false
	totalWeight := 0
	candidates := make([]BillingGroupChannel, 0)
	for _, entry := range entries {
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
		if !hasPriority || entry.Priority > bestPriority {
			bestPriority = entry.Priority
			hasPriority = true
			candidates = candidates[:0]
			totalWeight = 0
		}
		if entry.Priority != bestPriority {
			continue
		}
		weight := entry.Weight
		if weight <= 0 {
			weight = 100
		}
		totalWeight += weight
		candidates = append(candidates, entry)
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	selected := rand.Intn(totalWeight)
	for _, entry := range candidates {
		weight := entry.Weight
		if weight <= 0 {
			weight = 100
		}
		selected -= weight
		if selected < 0 {
			return channelsIDM[entry.ChannelId], nil
		}
	}
	return channelsIDM[candidates[len(candidates)-1].ChannelId], nil
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

// SyncChannelCacheEntry refreshes one channel and its enabled abilities after
// an atomic status update, without rebuilding the complete channel cache.
func SyncChannelCacheEntry(channel *Channel) {
	if !common.MemoryCacheEnabled || channel == nil || channel.Id <= 0 {
		return
	}
	var abilities []Ability
	if channel.Status == common.ChannelStatusEnabled {
		if err := DB.Where("channel_id = ? AND enabled = ?", channel.Id, true).Find(&abilities).Error; err != nil {
			common.SysLog(fmt.Sprintf("failed to sync channel abilities into cache: channel_id=%d, error=%v", channel.Id, err))
			abilities = nil
		}
	}

	channelSyncLock.Lock()
	if channelsIDM == nil {
		channelsIDM = make(map[int]*Channel)
	}
	oldChannel := channelsIDM[channel.Id]
	if oldChannel != nil {
		// PreviousDayProbeSuccessRate is computed outside the channels table.
		// Preserve it across status-only updates so replacing the cached model
		// does not temporarily make a channel look like it has a 0% success rate.
		channel.PreviousDayProbeSuccessRate = oldChannel.PreviousDayProbeSuccessRate
	}
	if channel.ChannelInfo.IsMultiKey {
		channel.Keys = channel.GetKeys()
		if oldChannel != nil && oldChannel.ChannelInfo.IsMultiKey &&
			oldChannel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling &&
			channel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
			channel.ChannelInfo.MultiKeyPollingIndex = oldChannel.ChannelInfo.MultiKeyPollingIndex
		}
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

	for group, model2channels := range group2model2channels {
		for modelName, channelIDs := range model2channels {
			filtered := channelIDs[:0]
			for _, channelID := range channelIDs {
				if channelID != channel.Id {
					filtered = append(filtered, channelID)
				}
			}
			if len(filtered) == 0 {
				delete(model2channels, modelName)
			} else {
				model2channels[modelName] = filtered
			}
		}
		if len(model2channels) == 0 {
			delete(group2model2channels, group)
		}
	}
	if channel.Status == common.ChannelStatusEnabled {
		if group2model2channels == nil {
			group2model2channels = make(map[string]map[string][]int)
		}
		for _, ability := range abilities {
			group := strings.TrimSpace(ability.Group)
			modelName := strings.TrimSpace(ability.Model)
			if group == "" || modelName == "" {
				continue
			}
			if group2model2channels[group] == nil {
				group2model2channels[group] = make(map[string][]int)
			}
			group2model2channels[group][modelName] = append(group2model2channels[group][modelName], channel.Id)
			sort.Ints(group2model2channels[group][modelName])
		}
	}
	channelSyncLock.Unlock()
	InvalidatePricingCache()
}
