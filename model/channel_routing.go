package model

import (
	"errors"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

// BillingGroupRoute owns the retry policy for one billing group.
// The billing group is the existing group used by tokens, abilities, and
// channels, and routes directly to channels.
type BillingGroupRoute struct {
	Id               int    `json:"id"`
	BillingGroup     string `json:"billing_group" gorm:"type:varchar(64);uniqueIndex"`
	Name             string `json:"name" gorm:"type:varchar(128)"`
	Enabled          bool   `json:"enabled" gorm:"index"`
	MaxTotalAttempts int    `json:"max_total_attempts"`
	TotalTimeoutMs   int    `json:"total_timeout_ms"`
	CreatedTime      int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64  `json:"updated_time" gorm:"bigint"`
}

// BillingGroupChannel stores the stable route order and the long-term weight
// used when dynamic scores are equivalent. Priority is a lower-is-earlier
// tie-breaker; dynamic pricing-group strategies decide the normal score tier.
type BillingGroupChannel struct {
	Id                  int     `json:"id"`
	BillingGroupRouteId int     `json:"billing_group_route_id" gorm:"index;uniqueIndex:idx_billing_route_channel"`
	ChannelId           int     `json:"channel_id" gorm:"index;uniqueIndex:idx_billing_route_channel"`
	Priority            int     `json:"priority"`
	Weight              int     `json:"weight"`
	MaxAttempts         int     `json:"max_attempts"`
	Enabled             bool    `json:"enabled" gorm:"index"`
	CostFactor          float64 `json:"cost_factor"`
}

// UpstreamErrorMapping translates a provider/channel error into the gateway's
// stable error taxonomy. ChannelId is an optional exact override; ChannelType
// is the provider adapter type and 0 means any type.
type UpstreamErrorMapping struct {
	Id           int    `json:"id"`
	ChannelId    int    `json:"channel_id" gorm:"index;uniqueIndex:idx_error_mapping_v2"`
	ChannelType  int    `json:"channel_type" gorm:"index;uniqueIndex:idx_error_mapping_v2"`
	RawCode      string `json:"raw_code" gorm:"type:varchar(128);uniqueIndex:idx_error_mapping_v2"`
	StatusCode   int    `json:"status_code" gorm:"uniqueIndex:idx_error_mapping_v2"`
	StableCode   int    `json:"stable_code" gorm:"index"`
	Category     string `json:"category" gorm:"type:varchar(64);index"`
	FailureScope string `json:"failure_scope" gorm:"type:varchar(32)"`
	Action       string `json:"action" gorm:"type:varchar(32)"`
	Retryable    bool   `json:"retryable"`
	Enabled      bool   `json:"enabled" gorm:"index"`
}

func (UpstreamErrorMapping) TableName() string {
	return "channel_error_mappings"
}

type ChannelRoutingConfig struct {
	Routes        []BillingGroupRoute    `json:"routes"`
	RouteChannels []BillingGroupChannel  `json:"route_channels"`
	ErrorMappings []UpstreamErrorMapping `json:"error_mappings"`
}

type RuntimeRoutingPolicy struct {
	MaxTotalAttempts int
	TotalTimeoutMs   int
	RoutingStrategy  ratio_setting.PricingGroupRoutingStrategy
}

func DefaultRuntimeRoutingPolicy() RuntimeRoutingPolicy {
	return RuntimeRoutingPolicy{
		MaxTotalAttempts: 4,
		TotalTimeoutMs:   30000,
		RoutingStrategy:  ratio_setting.DefaultPricingGroupRoutingStrategy(),
	}
}

type channelRoutingLookupCache struct {
	routes        map[string]BillingGroupRoute
	routeChannels map[int][]BillingGroupChannel
	mappings      []UpstreamErrorMapping
}

var channelRoutingLookup = struct {
	sync.RWMutex
	value channelRoutingLookupCache
}{value: channelRoutingLookupCache{
	routes:        make(map[string]BillingGroupRoute),
	routeChannels: make(map[int][]BillingGroupChannel),
}}

func InitChannelRoutingCache() {
	cache := channelRoutingLookupCache{
		routes:        make(map[string]BillingGroupRoute),
		routeChannels: make(map[int][]BillingGroupChannel),
	}
	if DB != nil && DB.Migrator().HasTable(&BillingGroupRoute{}) {
		var routes []BillingGroupRoute
		if err := DB.Where("enabled = ?", true).Order("id ASC").Find(&routes).Error; err == nil {
			for _, route := range routes {
				cache.routes[strings.TrimSpace(route.BillingGroup)] = route
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&BillingGroupChannel{}) {
		var channels []BillingGroupChannel
		if err := DB.Where("enabled = ?", true).Order("priority ASC, id ASC").Find(&channels).Error; err == nil {
			for _, channel := range channels {
				cache.routeChannels[channel.BillingGroupRouteId] = append(cache.routeChannels[channel.BillingGroupRouteId], channel)
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&UpstreamErrorMapping{}) {
		_ = DB.Where("enabled = ?", true).Order("id ASC").Find(&cache.mappings).Error
	}
	channelRoutingLookup.Lock()
	channelRoutingLookup.value = cache
	channelRoutingLookup.Unlock()
}

func ResolveBillingGroupRoute(billingGroup string) (RuntimeRoutingPolicy, []BillingGroupChannel, bool) {
	channelRoutingLookup.RLock()
	defer channelRoutingLookup.RUnlock()
	route, ok := channelRoutingLookup.value.routes[strings.TrimSpace(billingGroup)]
	if !ok {
		policy := DefaultRuntimeRoutingPolicy()
		if strategy, exists := ratio_setting.GetPricingGroupRoutingStrategy(billingGroup); exists {
			policy.RoutingStrategy = strategy
		}
		return policy, nil, false
	}
	policy := DefaultRuntimeRoutingPolicy()
	if route.MaxTotalAttempts > 0 {
		policy.MaxTotalAttempts = route.MaxTotalAttempts
	}
	if route.TotalTimeoutMs > 0 {
		policy.TotalTimeoutMs = route.TotalTimeoutMs
	}
	if strategy, exists := ratio_setting.GetPricingGroupRoutingStrategy(billingGroup); exists {
		policy.RoutingStrategy = strategy
	}
	channels := append([]BillingGroupChannel(nil), channelRoutingLookup.value.routeChannels[route.Id]...)
	return policy, channels, true
}

func MatchUpstreamErrorMapping(channelID int, channelType int, rawCode string, statusCode int) (UpstreamErrorMapping, bool) {
	channelRoutingLookup.RLock()
	defer channelRoutingLookup.RUnlock()
	rawCode = strings.ToLower(strings.TrimSpace(rawCode))
	bestScore := -1
	var best UpstreamErrorMapping
	for _, mapping := range channelRoutingLookup.value.mappings {
		mappingRawCode := strings.ToLower(strings.TrimSpace(mapping.RawCode))
		if mapping.ChannelId > 0 && mapping.ChannelId != channelID {
			continue
		}
		if mapping.ChannelType > 0 && mapping.ChannelType != channelType {
			continue
		}
		if mappingRawCode != "" && mappingRawCode != "*" && mappingRawCode != rawCode {
			continue
		}
		if mapping.StatusCode != 0 && mapping.StatusCode != statusCode {
			continue
		}
		score := 0
		if mapping.ChannelId > 0 {
			score += 8
		}
		if mapping.ChannelType > 0 {
			score += 4
		}
		if mappingRawCode != "" && mappingRawCode != "*" {
			score += 2
		}
		if mapping.StatusCode != 0 {
			score++
		}
		if score > bestScore {
			bestScore = score
			best = mapping
		}
	}
	return best, bestScore >= 0
}

func GetChannelRoutingConfig() (*ChannelRoutingConfig, error) {
	config := &ChannelRoutingConfig{}
	queries := []struct {
		order string
		value any
	}{
		{"billing_group ASC", &config.Routes},
		{"billing_group_route_id ASC, priority ASC, id ASC", &config.RouteChannels},
		{"channel_id DESC, channel_type DESC, raw_code ASC, status_code ASC", &config.ErrorMappings},
	}
	for _, query := range queries {
		if err := DB.Order(query.order).Find(query.value).Error; err != nil {
			return nil, err
		}
	}
	return config, nil
}

func SaveChannelRoutingConfig(config *ChannelRoutingConfig) error {
	if config == nil {
		return errors.New("channel routing config is required")
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		routeIDs := make([]int, 0, len(config.Routes))
		routeIDMap := make(map[int]int, len(config.Routes))
		routeGroupByID := make(map[int]string, len(config.Routes))
		seenGroups := make(map[string]struct{}, len(config.Routes))
		for i := range config.Routes {
			route := &config.Routes[i]
			clientRouteID := route.Id
			if route.Id < 0 {
				route.Id = 0
			}
			route.BillingGroup = strings.TrimSpace(route.BillingGroup)
			route.Name = strings.TrimSpace(route.Name)
			if route.BillingGroup == "" {
				return errors.New("billing_group is required")
			}
			if _, exists := seenGroups[route.BillingGroup]; exists {
				return errors.New("billing_group must be unique")
			}
			seenGroups[route.BillingGroup] = struct{}{}
			if route.Name == "" {
				route.Name = route.BillingGroup
			}
			applyRouteDefaults(route)
			if err := tx.Save(route).Error; err != nil {
				return err
			}
			routeIDs = append(routeIDs, route.Id)
			routeIDMap[clientRouteID] = route.Id
			routeGroupByID[route.Id] = route.BillingGroup
		}

		channelIDs := make([]int, 0, len(config.RouteChannels))
		seenRouteChannels := make(map[[2]int]struct{}, len(config.RouteChannels))
		enabledChannelsByRoute := make(map[int]int, len(config.Routes))
		for i := range config.RouteChannels {
			entry := &config.RouteChannels[i]
			if entry.Id < 0 {
				entry.Id = 0
			}
			if persistedRouteID, ok := routeIDMap[entry.BillingGroupRouteId]; ok {
				entry.BillingGroupRouteId = persistedRouteID
			}
			if entry.BillingGroupRouteId <= 0 || entry.ChannelId <= 0 || entry.MaxAttempts <= 0 || entry.Weight < 0 || entry.CostFactor <= 0 || math.IsNaN(entry.CostFactor) || math.IsInf(entry.CostFactor, 0) {
				return errors.New("route channel contains invalid values")
			}
			key := [2]int{entry.BillingGroupRouteId, entry.ChannelId}
			if _, exists := seenRouteChannels[key]; exists {
				return errors.New("a channel can appear only once in a billing group route")
			}
			seenRouteChannels[key] = struct{}{}
			billingGroup, ok := routeGroupByID[entry.BillingGroupRouteId]
			if !ok {
				return errors.New("route channel references an unknown billing group route")
			}
			var channel Channel
			if err := tx.First(&channel, "id = ?", entry.ChannelId).Error; err != nil {
				return err
			}
			belongsToGroup := false
			for _, group := range strings.Split(channel.Group, ",") {
				if strings.TrimSpace(group) == billingGroup {
					belongsToGroup = true
					break
				}
			}
			if !belongsToGroup {
				return errors.New("route channel does not belong to its billing group")
			}
			if entry.Enabled {
				enabledChannelsByRoute[entry.BillingGroupRouteId]++
			}
			if err := tx.Save(entry).Error; err != nil {
				return err
			}
			channelIDs = append(channelIDs, entry.Id)
		}
		for _, route := range config.Routes {
			if route.Enabled && enabledChannelsByRoute[route.Id] == 0 {
				return errors.New("enabled billing group route requires an enabled channel")
			}
		}

		mappingIDs := make([]int, 0, len(config.ErrorMappings))
		for i := range config.ErrorMappings {
			mapping := &config.ErrorMappings[i]
			if mapping.Id < 0 {
				mapping.Id = 0
			}
			mapping.RawCode = strings.ToLower(strings.TrimSpace(mapping.RawCode))
			mapping.Category = strings.TrimSpace(mapping.Category)
			mapping.FailureScope = strings.TrimSpace(mapping.FailureScope)
			mapping.Action = strings.TrimSpace(mapping.Action)
			if mapping.StableCode < 100000 || mapping.StableCode > 999999 {
				return errors.New("error mapping stable_code must be a six-digit number")
			}
			if mapping.StatusCode != 0 && (mapping.StatusCode < 100 || mapping.StatusCode > 599) {
				return errors.New("error mapping status_code must be 0 or a valid HTTP status")
			}
			if mapping.RawCode == "" && mapping.StatusCode == 0 {
				return errors.New("error mapping requires raw_code or status_code")
			}
			switch mapping.FailureScope {
			case "request", "credential", "channel", "provider":
			default:
				return errors.New("error mapping failure_scope is invalid")
			}
			switch mapping.Action {
			case "none", "retry_channel", "switch_channel", "retry_later", "abort", "manual":
			default:
				return errors.New("error mapping action is invalid")
			}
			if err := tx.Save(mapping).Error; err != nil {
				return err
			}
			mappingIDs = append(mappingIDs, mapping.Id)
		}

		if err := deleteMissingRows(tx, &BillingGroupChannel{}, "id", channelIDs); err != nil {
			return err
		}
		if err := deleteMissingRows(tx, &BillingGroupRoute{}, "id", routeIDs); err != nil {
			return err
		}
		return deleteMissingRows(tx, &UpstreamErrorMapping{}, "id", mappingIDs)
	}); err != nil {
		return err
	}
	// Routing is read from the process-local lookup cache on every request.
	// Publish the committed rows immediately so a successful admin update takes
	// effect without waiting for a process restart or unrelated cache rebuild.
	InitChannelRoutingCache()
	return nil
}

func deleteMissingRows(tx *gorm.DB, value any, column string, ids []int) error {
	query := tx.Model(value)
	if len(ids) > 0 {
		query = query.Where(column+" NOT IN ?", ids)
	} else {
		query = query.Where("1 = 1")
	}
	return query.Delete(value).Error
}

func applyRouteDefaults(route *BillingGroupRoute) {
	defaults := DefaultRuntimeRoutingPolicy()
	if route.MaxTotalAttempts <= 0 {
		route.MaxTotalAttempts = defaults.MaxTotalAttempts
	}
	if route.TotalTimeoutMs <= 0 {
		route.TotalTimeoutMs = defaults.TotalTimeoutMs
	}
}

func SortRouteChannels(channels []BillingGroupChannel) {
	sort.SliceStable(channels, func(i, j int) bool {
		if channels[i].Priority == channels[j].Priority {
			return channels[i].Id < channels[j].Id
		}
		return channels[i].Priority < channels[j].Priority
	})
}
