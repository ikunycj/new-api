package model

import (
	"errors"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RoutingModeCostFirst          = "cost_first"
	RoutingModeBalanced           = "balanced"
	RoutingModeStabilityFirst     = "stability_first"
	BillingGroupTypeToB           = "toB"
	BillingGroupTypeToC           = "toC"
	RoutingStrategyPriority       = "priority"
	RoutingStrategyWeighted       = "weighted"
	ProfitGuardModeOff            = "off"
	ProfitGuardModeWarn           = "warn"
	ProfitGuardModeEnforce        = "enforce"
	ChannelCircuitConfigOptionKey = "ChannelCircuitConfig"
)

// BillingGroupRoute owns the retry and circuit policy for one billing group.
// The billing group is the existing group used by tokens, abilities, and
// channels; no extra cluster or account-pool layer is involved.
type BillingGroupRoute struct {
	Id           int    `json:"id"`
	BillingGroup string `json:"billing_group" gorm:"type:varchar(64);uniqueIndex"`
	Name         string `json:"name" gorm:"type:varchar(128)"`
	Mode         string `json:"mode" gorm:"type:varchar(32)"`
	// GroupType is the customer scope of this pricing/routing group. Empty
	// values are treated as ToB for backwards compatibility with legacy routes.
	GroupType string `json:"group_type" gorm:"type:varchar(8);index"`
	// StrategyConfig keeps the strategy and its tunables in one extensible JSON
	// document. The default is the legacy priority ordering.
	StrategyConfig          string  `json:"strategy_config" gorm:"type:text"`
	Enabled                 bool    `json:"enabled" gorm:"index"`
	MaxTotalAttempts        int     `json:"max_total_attempts"`
	TotalTimeoutMs          int     `json:"total_timeout_ms"`
	CircuitFailureThreshold int     `json:"circuit_failure_threshold"`
	CircuitWindowSeconds    int     `json:"circuit_window_seconds"`
	CircuitCooldownSeconds  int     `json:"circuit_cooldown_seconds"`
	CircuitHalfOpenRequests int     `json:"circuit_half_open_requests"`
	ProfitGuardMode         string  `json:"profit_guard_mode" gorm:"type:varchar(16)"`
	MinimumProfitMargin     float64 `json:"minimum_profit_margin"`
	CreatedTime             int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime             int64   `json:"updated_time" gorm:"bigint"`
}

// BillingGroupChannel defines the channel order and static distribution weight
// inside a billing group. Dynamic strategy weights are stored on the route.
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

// UpstreamErrorMapping translates a provider/channel error into AllToken's
// stable error taxonomy. ChannelId is an optional exact override; ChannelType
// is the provider adapter type and 0 means any type.
type UpstreamErrorMapping struct {
	Id           int    `json:"id"`
	ChannelId    int    `json:"channel_id" gorm:"index;uniqueIndex:idx_error_mapping_v2"`
	ChannelType  int    `json:"channel_type" gorm:"index;uniqueIndex:idx_error_mapping_v2"`
	RawCode      string `json:"raw_code" gorm:"type:varchar(128);uniqueIndex:idx_error_mapping_v2"`
	StatusCode   int    `json:"status_code" gorm:"uniqueIndex:idx_error_mapping_v2"`
	AlltokenCode int    `json:"alltoken_code" gorm:"index"`
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
	Routes          []BillingGroupRoute    `json:"routes"`
	RouteChannels   []BillingGroupChannel  `json:"route_channels"`
	ErrorMappings   []UpstreamErrorMapping `json:"error_mappings"`
	CircuitDefaults ChannelCircuitPolicy   `json:"circuit_defaults"`
	CircuitPresets  []ChannelCircuitPreset `json:"circuit_presets"`
}

type ChannelCircuitPreset struct {
	Key              string `json:"key"`
	Label            string `json:"label"`
	FailureThreshold int    `json:"failure_threshold"`
	WindowSeconds    int    `json:"window_seconds"`
	CooldownSeconds  int    `json:"cooldown_seconds"`
	HalfOpenRequests int    `json:"half_open_requests"`
}

// ChannelCircuitConfig is stored as one JSON option. It owns only circuit
// defaults; retry, timeout, scheduling, and profit protection remain separate.
type ChannelCircuitConfig struct {
	Default ChannelCircuitPolicy            `json:"default"`
	Modes   map[string]ChannelCircuitPolicy `json:"modes"`
	Presets []ChannelCircuitPreset          `json:"presets"`
}

type ChannelCircuitPolicy struct {
	FailureThreshold int `json:"failure_threshold"`
	WindowSeconds    int `json:"window_seconds"`
	CooldownSeconds  int `json:"cooldown_seconds"`
	HalfOpenRequests int `json:"half_open_requests"`
}

func fallbackChannelCircuitConfig() ChannelCircuitConfig {
	return ChannelCircuitConfig{
		Default: ChannelCircuitPolicy{
			FailureThreshold: 5, WindowSeconds: 60, CooldownSeconds: 60, HalfOpenRequests: 1,
		},
		Modes: map[string]ChannelCircuitPolicy{
			RoutingModeCostFirst: {
				FailureThreshold: 8, WindowSeconds: 60, CooldownSeconds: 60, HalfOpenRequests: 1,
			},
			RoutingModeStabilityFirst: {
				FailureThreshold: 3, WindowSeconds: 60, CooldownSeconds: 90, HalfOpenRequests: 1,
			},
		},
		Presets: []ChannelCircuitPreset{
			{Key: "sensitive", Label: "Sensitive", FailureThreshold: 3, WindowSeconds: 30, CooldownSeconds: 60, HalfOpenRequests: 1},
			{Key: "standard", Label: "Standard", FailureThreshold: 20, WindowSeconds: 60, CooldownSeconds: 30, HalfOpenRequests: 1},
			{Key: "relaxed", Label: "Relaxed", FailureThreshold: 50, WindowSeconds: 60, CooldownSeconds: 30, HalfOpenRequests: 2},
		},
	}
}

func DefaultChannelCircuitConfigJSONString() string {
	data, err := common.Marshal(fallbackChannelCircuitConfig())
	if err != nil {
		return "{}"
	}
	return string(data)
}

func validChannelCircuitPolicy(policy ChannelCircuitPolicy) bool {
	return policy.FailureThreshold >= 1 && policy.FailureThreshold <= 10000 &&
		policy.WindowSeconds >= 1 && policy.WindowSeconds <= 86400 &&
		policy.CooldownSeconds >= 1 && policy.CooldownSeconds <= 86400 &&
		policy.HalfOpenRequests >= 1 && policy.HalfOpenRequests <= 100
}

func normalizeChannelCircuitConfig(config ChannelCircuitConfig) (ChannelCircuitConfig, bool) {
	fallback := fallbackChannelCircuitConfig()
	if !validChannelCircuitPolicy(config.Default) {
		return fallback, false
	}
	for _, mode := range []string{RoutingModeCostFirst, RoutingModeStabilityFirst} {
		policy, ok := config.Modes[mode]
		if !ok || !validChannelCircuitPolicy(policy) {
			return fallback, false
		}
	}
	if len(config.Presets) == 0 || len(config.Presets) > 20 {
		return fallback, false
	}
	seenPresetKeys := make(map[string]struct{}, len(config.Presets))
	for _, preset := range config.Presets {
		key := strings.TrimSpace(preset.Key)
		if key == "" || strings.TrimSpace(preset.Label) == "" ||
			preset.FailureThreshold < 1 || preset.FailureThreshold > 10000 ||
			preset.WindowSeconds < 1 || preset.WindowSeconds > 86400 ||
			preset.CooldownSeconds < 1 || preset.CooldownSeconds > 86400 ||
			preset.HalfOpenRequests < 1 || preset.HalfOpenRequests > 100 {
			return fallback, false
		}
		if _, exists := seenPresetKeys[key]; exists {
			return fallback, false
		}
		seenPresetKeys[key] = struct{}{}
	}
	return config, true
}

func NormalizeChannelCircuitConfigJSONString(raw string) (string, error) {
	var config ChannelCircuitConfig
	if err := common.Unmarshal([]byte(raw), &config); err != nil {
		return "", errors.New("ChannelCircuitConfig must be valid JSON")
	}
	normalized, ok := normalizeChannelCircuitConfig(config)
	if !ok {
		return "", errors.New("ChannelCircuitConfig contains missing or out-of-range values")
	}
	data, err := common.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetChannelCircuitConfig() ChannelCircuitConfig {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[ChannelCircuitConfigOptionKey]
	common.OptionMapRWMutex.RUnlock()
	normalizedRaw, err := NormalizeChannelCircuitConfigJSONString(raw)
	if err != nil {
		return fallbackChannelCircuitConfig()
	}
	var config ChannelCircuitConfig
	if err := common.Unmarshal([]byte(normalizedRaw), &config); err != nil {
		return fallbackChannelCircuitConfig()
	}
	return config
}

type RuntimeRoutingPolicy struct {
	Mode                    string
	MaxTotalAttempts        int
	TotalTimeoutMs          int
	CircuitFailureThreshold int
	CircuitWindowSeconds    int
	CircuitCooldownSeconds  int
	CircuitHalfOpenRequests int
	ProfitGuardMode         string
	MinimumProfitMargin     float64
	Strategy                string
	StrategyConfig          RoutingStrategyConfig
}

// RoutingStrategyConfig is intentionally a single JSON object so future
// strategy knobs do not require another database column or API shape.
type RoutingStrategyConfig struct {
	Type               string  `json:"type"`
	PriceWeight        float64 `json:"price_weight,omitempty"`
	AvailabilityWeight float64 `json:"availability_weight,omitempty"`
	LoadWeight         float64 `json:"load_weight,omitempty"`
}

func DefaultRuntimeRoutingPolicy(mode string) RuntimeRoutingPolicy {
	config := GetChannelCircuitConfig()
	defaults := config.Default
	normalizedMode := normalizeRoutingMode(mode)
	if modeDefaults, ok := config.Modes[normalizedMode]; ok {
		defaults = modeDefaults
	}
	policy := RuntimeRoutingPolicy{
		Mode:                    normalizedMode,
		MaxTotalAttempts:        4,
		TotalTimeoutMs:          30000,
		CircuitFailureThreshold: defaults.FailureThreshold,
		CircuitWindowSeconds:    defaults.WindowSeconds,
		CircuitCooldownSeconds:  defaults.CooldownSeconds,
		CircuitHalfOpenRequests: defaults.HalfOpenRequests,
		ProfitGuardMode:         ProfitGuardModeOff,
		Strategy:                RoutingStrategyPriority,
	}
	switch policy.Mode {
	case RoutingModeCostFirst:
		policy.MaxTotalAttempts = 6
		policy.TotalTimeoutMs = 45000
	case RoutingModeStabilityFirst:
		policy.MaxTotalAttempts = 3
		policy.TotalTimeoutMs = 20000
	}
	return policy
}

type channelRoutingLookupCache struct {
	routes           map[string]BillingGroupRoute
	toBBillingGroups map[string]struct{}
	routeChannels    map[int][]BillingGroupChannel
	mappings         []UpstreamErrorMapping
}

var channelRoutingLookup = struct {
	sync.RWMutex
	value channelRoutingLookupCache
}{value: channelRoutingLookupCache{
	routes:           make(map[string]BillingGroupRoute),
	toBBillingGroups: make(map[string]struct{}),
	routeChannels:    make(map[int][]BillingGroupChannel),
}}

func InitChannelRoutingCache() {
	cache := channelRoutingLookupCache{
		routes:           make(map[string]BillingGroupRoute),
		toBBillingGroups: make(map[string]struct{}),
		routeChannels:    make(map[int][]BillingGroupChannel),
	}
	if DB != nil && DB.Migrator().HasTable(&BillingGroupRoute{}) {
		var routes []BillingGroupRoute
		if err := DB.Order("id ASC").Find(&routes).Error; err == nil {
			for _, route := range routes {
				billingGroup := strings.TrimSpace(route.BillingGroup)
				if billingGroup == "" {
					continue
				}
				if normalizeBillingGroupType(route.GroupType) == BillingGroupTypeToB {
					cache.toBBillingGroups[billingGroup] = struct{}{}
				}
				if route.Enabled {
					cache.routes[billingGroup] = route
				}
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&BillingGroupChannel{}) {
		var channels []BillingGroupChannel
		if err := DB.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&channels).Error; err == nil {
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

// GetBillingGroupTypes classifies groups from the explicit route type. Legacy
// routes without group_type remain ToB so existing deployments do not change.
func GetBillingGroupTypes(groups map[string]float64) map[string]string {
	groupTypes := make(map[string]string, len(groups))
	channelRoutingLookup.RLock()
	defer channelRoutingLookup.RUnlock()
	for group := range groups {
		groupTypes[group] = BillingGroupTypeToC
		if _, ok := channelRoutingLookup.value.toBBillingGroups[group]; ok {
			groupTypes[group] = BillingGroupTypeToB
		}
	}
	return groupTypes
}

// IsBillingGroupToB reports whether the configured billing group is routed as
// a ToB group. Group names remain data-driven through BillingGroupRoute.
func IsBillingGroupToB(group string) bool {
	group = strings.TrimSpace(group)
	if group == "" {
		return false
	}
	// `toB` is the existing reserved user billing group.
	if strings.EqualFold(group, BillingGroupTypeToB) {
		return true
	}
	return GetBillingGroupTypes(map[string]float64{group: 1})[group] == BillingGroupTypeToB
}

func ResolveBillingGroupRoute(billingGroup string) (RuntimeRoutingPolicy, []BillingGroupChannel, bool) {
	channelRoutingLookup.RLock()
	defer channelRoutingLookup.RUnlock()
	route, ok := channelRoutingLookup.value.routes[strings.TrimSpace(billingGroup)]
	if !ok {
		return DefaultRuntimeRoutingPolicy(RoutingModeBalanced), nil, false
	}
	policy := DefaultRuntimeRoutingPolicy(route.Mode)
	policy.StrategyConfig = parseRoutingStrategyConfig(route.StrategyConfig)
	policy.Strategy = policy.StrategyConfig.Type
	if route.MaxTotalAttempts > 0 {
		policy.MaxTotalAttempts = route.MaxTotalAttempts
	}
	if route.TotalTimeoutMs > 0 {
		policy.TotalTimeoutMs = route.TotalTimeoutMs
	}
	if route.CircuitFailureThreshold > 0 {
		policy.CircuitFailureThreshold = route.CircuitFailureThreshold
	}
	if route.CircuitWindowSeconds > 0 {
		policy.CircuitWindowSeconds = route.CircuitWindowSeconds
	}
	if route.CircuitCooldownSeconds > 0 {
		policy.CircuitCooldownSeconds = route.CircuitCooldownSeconds
	}
	if route.CircuitHalfOpenRequests > 0 {
		policy.CircuitHalfOpenRequests = route.CircuitHalfOpenRequests
	}
	policy.ProfitGuardMode = normalizeProfitGuardMode(route.ProfitGuardMode)
	if route.MinimumProfitMargin >= 0 && route.MinimumProfitMargin < 100 &&
		!math.IsNaN(route.MinimumProfitMargin) && !math.IsInf(route.MinimumProfitMargin, 0) {
		policy.MinimumProfitMargin = route.MinimumProfitMargin
	}
	channels := append([]BillingGroupChannel(nil), channelRoutingLookup.value.routeChannels[route.Id]...)
	return policy, channels, true
}

func ResolveChannelCostFactor(billingGroup string, channelID int) float64 {
	_, channels, ok := ResolveBillingGroupRoute(billingGroup)
	if !ok {
		return 1
	}
	for _, channel := range channels {
		if channel.ChannelId == channelID && channel.CostFactor > 0 {
			return channel.CostFactor
		}
	}
	return 1
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
		{"billing_group_route_id ASC, priority DESC, id ASC", &config.RouteChannels},
		{"channel_id DESC, channel_type DESC, raw_code ASC, status_code ASC", &config.ErrorMappings},
	}
	for _, query := range queries {
		if err := DB.Order(query.order).Find(query.value).Error; err != nil {
			return nil, err
		}
	}
	for i := range config.Routes {
		config.Routes[i].ProfitGuardMode = normalizeProfitGuardMode(config.Routes[i].ProfitGuardMode)
		config.Routes[i].GroupType = normalizeBillingGroupType(config.Routes[i].GroupType)
		config.Routes[i].StrategyConfig = normalizeStrategyConfigJSON(config.Routes[i].StrategyConfig)
	}
	circuitConfig := GetChannelCircuitConfig()
	config.CircuitDefaults = circuitConfig.Default
	config.CircuitPresets = circuitConfig.Presets
	return config, nil
}

func SaveChannelRoutingConfig(config *ChannelRoutingConfig) error {
	if config == nil {
		return errors.New("channel routing config is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
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
			route.Mode = normalizeRoutingMode(route.Mode)
			route.GroupType = normalizeBillingGroupType(route.GroupType)
			if route.StrategyConfig == "" {
				route.StrategyConfig = marshalRoutingStrategyConfig(RoutingStrategyConfig{Type: RoutingStrategyPriority})
			} else {
				route.StrategyConfig = normalizeStrategyConfigJSON(route.StrategyConfig)
			}
			route.ProfitGuardMode = normalizeProfitGuardMode(route.ProfitGuardMode)
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
			if route.MinimumProfitMargin < 0 || route.MinimumProfitMargin >= 100 ||
				math.IsNaN(route.MinimumProfitMargin) || math.IsInf(route.MinimumProfitMargin, 0) {
				return errors.New("minimum_profit_margin must be between 0 and 100")
			}
			applyRouteDefaults(route)
			if err := tx.Save(route).Error; err != nil {
				return err
			}
			routeIDs = append(routeIDs, route.Id)
			routeIDMap[clientRouteID] = route.Id
			routeGroupByID[route.Id] = route.BillingGroup
		}

		routeChannelIndexes := make(map[int][]int, len(config.Routes))
		for i := range config.RouteChannels {
			entry := &config.RouteChannels[i]
			if entry.Id < 0 {
				entry.Id = 0
			}
			if persistedRouteID, ok := routeIDMap[entry.BillingGroupRouteId]; ok {
				entry.BillingGroupRouteId = persistedRouteID
			}
			routeChannelIndexes[entry.BillingGroupRouteId] = append(
				routeChannelIndexes[entry.BillingGroupRouteId],
				i,
			)
		}
		for _, indexes := range routeChannelIndexes {
			sort.SliceStable(indexes, func(i, j int) bool {
				left := config.RouteChannels[indexes[i]]
				right := config.RouteChannels[indexes[j]]
				if left.Priority == right.Priority {
					return left.Id < right.Id
				}
				return left.Priority > right.Priority
			})
			for position, index := range indexes {
				config.RouteChannels[index].Priority = len(indexes) - position
			}
		}

		channelIDs := make([]int, 0, len(config.RouteChannels))
		configuredChannelIDs := make([]int, 0, len(config.RouteChannels))
		seenChannelIDs := make(map[int]struct{}, len(config.RouteChannels))
		for _, entry := range config.RouteChannels {
			if entry.ChannelId <= 0 {
				continue
			}
			if _, exists := seenChannelIDs[entry.ChannelId]; exists {
				continue
			}
			seenChannelIDs[entry.ChannelId] = struct{}{}
			configuredChannelIDs = append(configuredChannelIDs, entry.ChannelId)
		}
		channelsByID := make(map[int]Channel, len(configuredChannelIDs))
		if len(configuredChannelIDs) > 0 {
			var configuredChannels []Channel
			if err := tx.Where("id IN ?", configuredChannelIDs).Find(&configuredChannels).Error; err != nil {
				return err
			}
			for _, channel := range configuredChannels {
				channelsByID[channel.Id] = channel
			}
		}
		seenRouteChannels := make(map[[2]int]struct{}, len(config.RouteChannels))
		enabledChannelsByRoute := make(map[int]int, len(config.Routes))
		totalAttemptsByRoute := make(map[int]int, len(config.Routes))
		for i := range config.RouteChannels {
			entry := &config.RouteChannels[i]
			if entry.BillingGroupRouteId <= 0 || entry.ChannelId <= 0 || entry.MaxAttempts <= 0 || entry.CostFactor <= 0 ||
				math.IsNaN(entry.CostFactor) || math.IsInf(entry.CostFactor, 0) {
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
			channel, ok := channelsByID[entry.ChannelId]
			if !ok {
				return gorm.ErrRecordNotFound
			}
			strategy := parseRoutingStrategyConfig(routeStrategyConfig(config.Routes, entry.BillingGroupRouteId))
			if strategy.Type == RoutingStrategyPriority {
				entry.Weight = 0
			} else if entry.Weight < 0 {
				return errors.New("route channel weight must be non-negative")
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
				totalAttemptsByRoute[entry.BillingGroupRouteId] += entry.MaxAttempts
			}
			if err := tx.Save(entry).Error; err != nil {
				return err
			}
			channelIDs = append(channelIDs, entry.Id)
		}
		for i := range config.Routes {
			route := &config.Routes[i]
			if attempts := totalAttemptsByRoute[route.Id]; attempts > 0 {
				route.MaxTotalAttempts = attempts
				if err := tx.Model(route).Update("max_total_attempts", attempts).Error; err != nil {
					return err
				}
			}
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
			if mapping.AlltokenCode < 100000 || mapping.AlltokenCode > 999999 {
				return errors.New("error mapping alltoken_code must be a six-digit number")
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
	})
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
	defaults := DefaultRuntimeRoutingPolicy(route.Mode)
	if route.MaxTotalAttempts <= 0 {
		route.MaxTotalAttempts = defaults.MaxTotalAttempts
	}
	if route.TotalTimeoutMs <= 0 {
		route.TotalTimeoutMs = defaults.TotalTimeoutMs
	}
	if route.CircuitFailureThreshold <= 0 {
		route.CircuitFailureThreshold = defaults.CircuitFailureThreshold
	}
	if route.CircuitWindowSeconds <= 0 {
		route.CircuitWindowSeconds = defaults.CircuitWindowSeconds
	}
	if route.CircuitCooldownSeconds <= 0 {
		route.CircuitCooldownSeconds = defaults.CircuitCooldownSeconds
	}
	if route.CircuitHalfOpenRequests <= 0 {
		route.CircuitHalfOpenRequests = defaults.CircuitHalfOpenRequests
	}
}

func normalizeRoutingMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case RoutingModeCostFirst:
		return RoutingModeCostFirst
	case RoutingModeStabilityFirst:
		return RoutingModeStabilityFirst
	default:
		return RoutingModeBalanced
	}
}

func normalizeBillingGroupType(groupType string) string {
	if strings.EqualFold(strings.TrimSpace(groupType), BillingGroupTypeToC) {
		return BillingGroupTypeToC
	}
	return BillingGroupTypeToB
}

func parseRoutingStrategyConfig(raw string) RoutingStrategyConfig {
	config := RoutingStrategyConfig{Type: RoutingStrategyPriority}
	if strings.TrimSpace(raw) != "" {
		var payload struct {
			Type               string  `json:"type"`
			Strategy           string  `json:"strategy"`
			PriceWeight        float64 `json:"price_weight"`
			AvailabilityWeight float64 `json:"availability_weight"`
			LoadWeight         float64 `json:"load_weight"`
		}
		if err := common.Unmarshal([]byte(raw), &payload); err != nil {
			return config
		}
		config.Type = payload.Type
		if config.Type == "" {
			config.Type = payload.Strategy
		}
		config.PriceWeight = payload.PriceWeight
		config.AvailabilityWeight = payload.AvailabilityWeight
		config.LoadWeight = payload.LoadWeight
	}
	if strings.EqualFold(config.Type, RoutingStrategyWeighted) {
		config.Type = RoutingStrategyWeighted
	} else {
		config.Type = RoutingStrategyPriority
	}
	return normalizeRoutingStrategyConfig(config)
}

func normalizeRoutingStrategyConfig(config RoutingStrategyConfig) RoutingStrategyConfig {
	if config.Type != RoutingStrategyWeighted {
		return RoutingStrategyConfig{Type: RoutingStrategyPriority}
	}
	weights := []float64{config.PriceWeight, config.AvailabilityWeight, config.LoadWeight}
	total := 0.0
	valid := true
	for _, weight := range weights {
		if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			valid = false
			break
		}
		total += weight
	}
	if !valid || total <= 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		return RoutingStrategyConfig{
			Type:               RoutingStrategyWeighted,
			PriceWeight:        40,
			AvailabilityWeight: 40,
			LoadWeight:         20,
		}
	}
	if math.Abs(total-100) > 0.0001 {
		config.PriceWeight *= 100 / total
		config.AvailabilityWeight *= 100 / total
		config.LoadWeight *= 100 / total
	}
	return config
}

func marshalRoutingStrategyConfig(config RoutingStrategyConfig) string {
	data, err := common.Marshal(config)
	if err != nil {
		return `{"type":"priority"}`
	}
	return string(data)
}

func normalizeStrategyConfigJSON(raw string) string {
	return marshalRoutingStrategyConfig(parseRoutingStrategyConfig(raw))
}

func routeStrategyConfig(routes []BillingGroupRoute, routeID int) string {
	for _, route := range routes {
		if route.Id == routeID {
			return route.StrategyConfig
		}
	}
	return ""
}

func normalizeProfitGuardMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ProfitGuardModeWarn:
		return ProfitGuardModeWarn
	case ProfitGuardModeEnforce:
		return ProfitGuardModeEnforce
	default:
		return ProfitGuardModeOff
	}
}

func SortRouteChannels(channels []BillingGroupChannel) {
	sort.SliceStable(channels, func(i, j int) bool {
		if channels[i].Priority == channels[j].Priority {
			return channels[i].Id < channels[j].Id
		}
		return channels[i].Priority > channels[j].Priority
	})
}
