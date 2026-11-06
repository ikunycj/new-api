package model

import (
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// BillingGroupRoute stores the optional route membership for one billing group.
// The billing group is the existing group used by tokens, abilities, and
// channels, and routes directly to channels.
type BillingGroupRoute struct {
	Id           int    `json:"id"`
	BillingGroup string `json:"billing_group" gorm:"type:varchar(64);uniqueIndex"`
	Name         string `json:"name" gorm:"type:varchar(128)"`
	Enabled      bool   `json:"enabled" gorm:"index"`
	CreatedTime  int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64  `json:"updated_time" gorm:"bigint"`
}

// BillingGroupChannel stores the stable route order and the long-term weight
// used when dynamic scores are equivalent. Priority is a lower-is-earlier
// tie-breaker; dynamic pricing-group strategies decide the normal score tier.
type BillingGroupChannel struct {
	Id                  int  `json:"id"`
	BillingGroupRouteId int  `json:"billing_group_route_id" gorm:"index;uniqueIndex:idx_billing_route_channel"`
	ChannelId           int  `json:"channel_id" gorm:"index;uniqueIndex:idx_billing_route_channel"`
	Priority            int  `json:"priority"`
	Weight              int  `json:"weight"`
	Enabled             bool `json:"enabled" gorm:"index"`
}

type RuntimeRoutingPolicy struct {
	RoutingStrategy ratio_setting.PricingGroupRoutingStrategy
}

func DefaultRuntimeRoutingPolicy() RuntimeRoutingPolicy {
	return RuntimeRoutingPolicy{
		RoutingStrategy: ratio_setting.DefaultPricingGroupRoutingStrategy(),
	}
}

type channelRoutingLookupCache struct {
	routes        map[string]BillingGroupRoute
	routeChannels map[int][]BillingGroupChannel
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
	channelRoutingLookup.Lock()
	channelRoutingLookup.value = cache
	channelRoutingLookup.Unlock()
}

func ResolveBillingGroupRoute(billingGroup string) (RuntimeRoutingPolicy, []BillingGroupChannel, bool) {
	channelRoutingLookup.RLock()
	defer channelRoutingLookup.RUnlock()
	route, ok := channelRoutingLookup.value.routes[strings.TrimSpace(billingGroup)]
	policy := DefaultRuntimeRoutingPolicy()
	if strategy, exists := ratio_setting.GetPricingGroupRoutingStrategy(billingGroup); exists {
		policy.RoutingStrategy = strategy
	}
	if !ok {
		return policy, nil, false
	}
	channels := append([]BillingGroupChannel(nil), channelRoutingLookup.value.routeChannels[route.Id]...)
	return policy, channels, true
}

func SortRouteChannels(channels []BillingGroupChannel) {
	sort.SliceStable(channels, func(i, j int) bool {
		if channels[i].Priority == channels[j].Priority {
			return channels[i].Id < channels[j].Id
		}
		return channels[i].Priority < channels[j].Priority
	})
}
