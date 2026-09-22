package model

import (
	"errors"
	"fmt"
	"math"
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

// Channel-test TTFT is published independently from the persisted channel
// snapshot. Serializing full refreshes keeps an older, slower refresh from
// replacing a newer snapshot, while the per-channel version protects a
// just-completed test from being overwritten by a log query that started
// before that test finished.
var channelCacheRefreshLock sync.Mutex
var channelTestTTFTVersions = make(map[int]uint64)

func InitChannelCache() {
	channelCacheRefreshLock.Lock()
	defer channelCacheRefreshLock.Unlock()

	InitChannelRoutingCache()
	if !common.MemoryCacheEnabled {
		InvalidatePricingCache()
		return
	}

	channelSyncLock.RLock()
	refreshTTFTVersions := make(map[int]uint64, len(channelTestTTFTVersions))
	for channelID, version := range channelTestTTFTVersions {
		refreshTTFTVersions[channelID] = version
	}
	channelSyncLock.RUnlock()

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
	enrichPreviousDayChannelRuntimeMetrics(channels, time.Now())
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
		// LastTestTTFTMs is a runtime-only value. If the log query did not
		// return a usable sample (for example, consume logging is disabled or
		// a test completed concurrently with this refresh), retain the latest
		// value already published in the process instead of briefly resetting
		// the scheduler's metric to zero.
		if oldChannel, ok := channelsIDM[i]; ok && oldChannel != nil {
			// If a test completed after this refresh began, the in-memory value
			// is newer than the query snapshot and must win even when the query
			// returned a positive (but stale) sample.
			if channelTestTTFTVersions[i] != refreshTTFTVersions[i] {
				channel.LastTestTTFTMs = oldChannel.LastTestTTFTMs
			} else if channel.LastTestTTFTMs <= 0 && oldChannel.LastTestTTFTMs > 0 {
				channel.LastTestTTFTMs = oldChannel.LastTestTTFTMs
			}
		}
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
	for channelID := range channelTestTTFTVersions {
		if _, ok := newChannelId2channel[channelID]; !ok {
			delete(channelTestTTFTVersions, channelID)
		}
	}
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
	enrichPreviousDayChannelRuntimeMetrics(result, time.Now())
	sort.SliceStable(result, func(i, j int) bool { return result[i].Id < result[j].Id })
	return result, nil
}

// enrichPreviousDayChannelRuntimeMetrics loads the non-persisted metrics used
// by dynamic routing. Keeping this enrichment beside cache loading ensures the
// selector sees the same probe and TTFT data as the channel management API.
func enrichPreviousDayChannelRuntimeMetrics(channels []*Channel, now time.Time) {
	if len(channels) == 0 {
		return
	}
	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel != nil && channel.Id > 0 {
			channelIDs = append(channelIDs, channel.Id)
		}
	}
	if len(channelIDs) == 0 {
		return
	}
	// Routing cache refreshes also run in SQLite-backed tests and on the hot
	// request path. Keep this enrichment limited to probe and TTFT tables;
	// historical usage is PostgreSQL-specific and belongs to the admin API.
	if rates, samples, err := GetPreviousDayChannelProbeStats(channelIDs, now); err == nil {
		for _, channel := range channels {
			if channel != nil {
				channel.PreviousDayProbeSuccessRate = rates[channel.Id]
				channel.PreviousDayProbeSampleCount = samples[channel.Id]
			}
		}
	} else {
		common.SysLog("failed to load channel probe rates while syncing cache: " + err.Error())
	}
	if averages, err := GetPreviousDayChannelAverageTTFTs(channelIDs, now); err == nil {
		for _, channel := range channels {
			if channel != nil {
				channel.PreviousDayAverageTTFTMs = averages[channel.Id]
			}
		}
	} else {
		common.SysLog("failed to load channel TTFT metrics while syncing cache: " + err.Error())
	}
	if ttfts, err := GetLatestChannelTestTTFTs(channelIDs); err == nil {
		for _, channel := range channels {
			if channel != nil {
				channel.LastTestTTFTMs = ttfts[channel.Id]
			}
		}
	} else {
		common.SysLog("failed to load latest channel test TTFTs while syncing cache: " + err.Error())
	}
}

// UpdateCachedChannelTestTTFT publishes a newly completed channel-test TTFT
// to the in-memory channel cache. Test TTFT is a runtime metric rather than a
// persisted channel field, so replacing the cached model would risk losing
// concurrent runtime state; update only this metric instead.
func UpdateCachedChannelTestTTFT(channelID int, ttftMs float64) {
	if channelID <= 0 || ttftMs <= 0 || math.IsNaN(ttftMs) || math.IsInf(ttftMs, 0) {
		return
	}
	if !common.MemoryCacheEnabled {
		return
	}

	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	cached, ok := channelsIDM[channelID]
	if !ok || cached == nil {
		return
	}
	updated := *cached
	updated.LastTestTTFTMs = ttftMs
	channelsIDM[channelID] = &updated
	if channelTestTTFTVersions == nil {
		channelTestTTFTVersions = make(map[int]uint64)
	}
	channelTestTTFTVersions[channelID]++
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
		// These metrics are computed outside the channels table. Preserve them
		// when a caller replaces the cached channel with a persisted copy.
		channel.PreviousDayProbeSuccessRate = oldChannel.PreviousDayProbeSuccessRate
		channel.PreviousDayProbeSampleCount = oldChannel.PreviousDayProbeSampleCount
		channel.PreviousDayAverageTTFTMs = oldChannel.PreviousDayAverageTTFTMs
		channel.LastTestTTFTMs = oldChannel.LastTestTTFTMs
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
		channel.PreviousDayProbeSampleCount = oldChannel.PreviousDayProbeSampleCount
		channel.PreviousDayAverageTTFTMs = oldChannel.PreviousDayAverageTTFTMs
		channel.LastTestTTFTMs = oldChannel.LastTestTTFTMs
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
