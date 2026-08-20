package model

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/pkg/observability"
	"gorm.io/gorm"
)

const (
	// UnifiedClusterBillingGroup is the single user-facing package for all
	// cluster-backed channels. Individual cluster IDs remain internal routing
	// details and are still used for failover isolation.
	UnifiedClusterBillingGroup = "通用套餐"

	ClusterStatusDisabled = 0
	ClusterStatusEnabled  = 1

	PoolTierFree      = 1
	PoolTierPremium   = 2
	PoolTierFallback  = 3
	PoolTierEmergency = 4

	FailoverModeConservative = "conservative"
	FailoverModeBalanced     = "balanced"
	FailoverModeAggressive   = "aggressive"

	RoutingStrategyCostFirst         = "cost_first"
	RoutingStrategyBalanced          = "balanced"
	RoutingStrategyStabilityFirst    = "stability_first"
	RoutingStrategyProCostFirst      = "pro_cost_first"
	RoutingStrategyProStabilityFirst = "pro_stability_first"
)

type Cluster struct {
	Id               int            `json:"id"`
	Name             string         `json:"name" gorm:"type:varchar(128);index"`
	Type             string         `json:"type" gorm:"type:varchar(64);index"`
	Status           int            `json:"status" gorm:"index"`
	BillingGroup     string         `json:"billing_group" gorm:"type:varchar(64);index"`
	PolicyId         int            `json:"policy_id" gorm:"index"`
	FailoverPriority int            `json:"failover_priority" gorm:"index"`
	Remark           string         `json:"remark" gorm:"type:varchar(255)"`
	Archived         bool           `json:"archived" gorm:"index"`
	CreatedTime      int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

type ClusterPool struct {
	Id          int     `json:"id"`
	ClusterId   int     `json:"cluster_id" gorm:"index;uniqueIndex:idx_cluster_pool_tier"`
	Tier        int     `json:"tier" gorm:"uniqueIndex:idx_cluster_pool_tier"`
	Name        string  `json:"name" gorm:"type:varchar(128)"`
	Status      int     `json:"status" gorm:"index"`
	CostFactor  float64 `json:"cost_factor"`
	Remark      string  `json:"remark" gorm:"type:varchar(255)"`
	CreatedTime int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime int64   `json:"updated_time" gorm:"bigint"`
}

type FailoverPolicy struct {
	Id                      int     `json:"id"`
	Name                    string  `json:"name" gorm:"type:varchar(128);index"`
	Mode                    string  `json:"mode" gorm:"type:varchar(32);index"`
	Strategy                string  `json:"strategy" gorm:"type:varchar(32);index"`
	Enabled                 bool    `json:"enabled" gorm:"index"`
	SamePoolRetries         int     `json:"same_pool_retries"`
	ConnectTimeoutMs        int     `json:"connect_timeout_ms"`
	FirstByteTimeoutMs      int     `json:"first_byte_timeout_ms"`
	MaxPoolAttempts         int     `json:"max_pool_attempts"`
	MaxClusterAttempts      int     `json:"max_cluster_attempts"`
	MaxTotalAttempts        int     `json:"max_total_attempts"`
	TotalFailoverBudgetMs   int     `json:"total_failover_budget_ms"`
	SwitchStatusCodes       string  `json:"switch_status_codes" gorm:"type:text"`
	SwitchErrorCodes        string  `json:"switch_error_codes" gorm:"type:text"`
	CircuitFailureThreshold int     `json:"circuit_failure_threshold"`
	CircuitWindowSeconds    int     `json:"circuit_window_seconds"`
	CircuitCooldownSeconds  int     `json:"circuit_cooldown_seconds"`
	CircuitHalfOpenRequests int     `json:"circuit_half_open_requests"`
	AllowPaidEscalation     bool    `json:"allow_paid_escalation"`
	AllowFallback           bool    `json:"allow_fallback"`
	MaxCostMultiplier       float64 `json:"max_cost_multiplier"`
	CreatedTime             int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime             int64   `json:"updated_time" gorm:"bigint"`
}

type FailoverPolicyStep struct {
	Id          int `json:"id"`
	PolicyId    int `json:"policy_id" gorm:"index;uniqueIndex:idx_policy_step_order"`
	StepOrder   int `json:"step_order" gorm:"uniqueIndex:idx_policy_step_order"`
	PoolTier    int `json:"pool_tier"`
	MaxAttempts int `json:"max_attempts"`
}

type FailoverGroup struct {
	Id          int    `json:"id"`
	Name        string `json:"name" gorm:"type:varchar(128);index"`
	PolicyId    int    `json:"policy_id" gorm:"index"`
	Enabled     bool   `json:"enabled" gorm:"index"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

type FailoverGroupMember struct {
	Id              int `json:"id"`
	FailoverGroupId int `json:"failover_group_id" gorm:"index;uniqueIndex:idx_failover_group_cluster"`
	ClusterId       int `json:"cluster_id" gorm:"index;uniqueIndex:idx_failover_group_cluster"`
	Priority        int `json:"priority"`
	Weight          int `json:"weight"`
}

type FailoverRule struct {
	Id              int    `json:"id"`
	FailoverGroupId int    `json:"failover_group_id" gorm:"index"`
	ModelPattern    string `json:"model_pattern" gorm:"type:varchar(255);index"`
	RoutePattern    string `json:"route_pattern" gorm:"type:varchar(255)"`
	UserGroup       string `json:"user_group" gorm:"type:varchar(64);index"`
	PolicyId        int    `json:"policy_id" gorm:"index"`
	Priority        int    `json:"priority"`
	Enabled         bool   `json:"enabled" gorm:"index"`
}

type UpstreamErrorMapping struct {
	Id           int    `json:"id"`
	ClusterType  string `json:"cluster_type" gorm:"type:varchar(64);index;uniqueIndex:idx_error_mapping"`
	RawCode      string `json:"raw_code" gorm:"type:varchar(128);uniqueIndex:idx_error_mapping"`
	StatusCode   int    `json:"status_code" gorm:"uniqueIndex:idx_error_mapping"`
	AlltokenCode int    `json:"alltoken_code" gorm:"index"`
	Category     string `json:"category" gorm:"type:varchar(64);index"`
	FailureScope string `json:"failure_scope" gorm:"type:varchar(32)"`
	Action       string `json:"action" gorm:"type:varchar(32)"`
	Retryable    bool   `json:"retryable"`
	Enabled      bool   `json:"enabled" gorm:"index"`
}

type FailoverConfig struct {
	Clusters      []Cluster              `json:"clusters"`
	Pools         []ClusterPool          `json:"pools"`
	Policies      []FailoverPolicy       `json:"policies"`
	PolicySteps   []FailoverPolicyStep   `json:"policy_steps"`
	Groups        []FailoverGroup        `json:"groups"`
	GroupMembers  []FailoverGroupMember  `json:"group_members"`
	Rules         []FailoverRule         `json:"rules"`
	ErrorMappings []UpstreamErrorMapping `json:"error_mappings"`
}

type RuntimeFailoverPolicy struct {
	Mode                    string
	Strategy                string
	SamePoolRetries         int
	MaxPoolAttempts         int
	MaxClusterAttempts      int
	MaxTotalAttempts        int
	TotalFailoverBudgetMs   int
	CircuitFailureThreshold int
	CircuitWindowSeconds    int
	CircuitCooldownSeconds  int
	CircuitHalfOpenRequests int
	AllowPaidEscalation     bool
	AllowFallback           bool
	MaxCostMultiplier       float64
	PoolTiers               []int
	PoolAttemptsByTier      map[int]int
}

func DefaultRuntimeFailoverPolicy(mode string) RuntimeFailoverPolicy {
	switch normalizeFailoverMode(mode) {
	case FailoverModeConservative:
		return withDefaultPoolOrder(RuntimeFailoverPolicy{Mode: FailoverModeConservative, Strategy: RoutingStrategyStabilityFirst, SamePoolRetries: 1, MaxPoolAttempts: 4, MaxClusterAttempts: 2, MaxTotalAttempts: 4, TotalFailoverBudgetMs: 12000, CircuitFailureThreshold: 8, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60, CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2})
	case FailoverModeAggressive:
		return withDefaultPoolOrder(RuntimeFailoverPolicy{Mode: FailoverModeAggressive, Strategy: RoutingStrategyCostFirst, MaxPoolAttempts: 4, MaxClusterAttempts: 4, MaxTotalAttempts: 8, TotalFailoverBudgetMs: 6000, CircuitFailureThreshold: 3, CircuitWindowSeconds: 30, CircuitCooldownSeconds: 90, CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2})
	default:
		return withDefaultPoolOrder(RuntimeFailoverPolicy{Mode: FailoverModeBalanced, Strategy: RoutingStrategyBalanced, MaxPoolAttempts: 4, MaxClusterAttempts: 3, MaxTotalAttempts: 6, TotalFailoverBudgetMs: 10000, CircuitFailureThreshold: 5, CircuitWindowSeconds: 60, CircuitCooldownSeconds: 60, CircuitHalfOpenRequests: 1, AllowPaidEscalation: true, AllowFallback: true, MaxCostMultiplier: 2})
	}
}

func withDefaultPoolOrder(policy RuntimeFailoverPolicy) RuntimeFailoverPolicy {
	policy.PoolAttemptsByTier = make(map[int]int)
	switch policy.Strategy {
	case RoutingStrategyProCostFirst, RoutingStrategyProStabilityFirst:
		policy.PoolTiers = []int{PoolTierPremium, PoolTierFallback}
	case RoutingStrategyStabilityFirst:
		policy.PoolTiers = []int{PoolTierFallback, PoolTierEmergency, PoolTierPremium, PoolTierFree}
	case RoutingStrategyBalanced:
		policy.PoolTiers = []int{PoolTierPremium, PoolTierFree, PoolTierFallback, PoolTierEmergency}
	default:
		policy.PoolTiers = []int{PoolTierFree, PoolTierPremium, PoolTierFallback, PoolTierEmergency}
	}
	return policy
}

func GetRuntimeFailoverPolicy(mode string) RuntimeFailoverPolicy {
	fallback := DefaultRuntimeFailoverPolicy(mode)
	failoverLookup.RLock()
	policy, ok := failoverLookup.value.policies[fallback.Mode]
	failoverLookup.RUnlock()
	if !ok {
		return fallback
	}
	return policy
}

type failoverLookupCache struct {
	clusterTypes         map[int]string
	clusterPolicyIDs     map[int]int
	billingGroupClusters map[string][]int
	billingGroupPolicies map[string]int
	mappings             []UpstreamErrorMapping
	policies             map[string]RuntimeFailoverPolicy
	policiesByID         map[int]RuntimeFailoverPolicy
	policiesByStrategy   map[string]RuntimeFailoverPolicy
	policyIDsByStrategy  map[string]int
	groups               map[int]FailoverGroup
	groupMembers         map[int][]FailoverGroupMember
	groupsByName         map[string]FailoverGroup
	rules                []FailoverRule
}

var failoverLookup = struct {
	sync.RWMutex
	value failoverLookupCache
}{value: failoverLookupCache{
	clusterTypes:         make(map[int]string),
	clusterPolicyIDs:     make(map[int]int),
	billingGroupClusters: make(map[string][]int),
	billingGroupPolicies: make(map[string]int),
	policies:             make(map[string]RuntimeFailoverPolicy),
	policiesByID:         make(map[int]RuntimeFailoverPolicy),
	policiesByStrategy:   make(map[string]RuntimeFailoverPolicy),
	policyIDsByStrategy:  make(map[string]int),
	groups:               make(map[int]FailoverGroup),
	groupMembers:         make(map[int][]FailoverGroupMember),
	groupsByName:         make(map[string]FailoverGroup),
}}

func InitFailoverCache() {
	cache := failoverLookupCache{
		clusterTypes:         make(map[int]string),
		clusterPolicyIDs:     make(map[int]int),
		billingGroupClusters: make(map[string][]int),
		billingGroupPolicies: make(map[string]int),
		policies:             make(map[string]RuntimeFailoverPolicy),
		policiesByID:         make(map[int]RuntimeFailoverPolicy),
		policiesByStrategy:   make(map[string]RuntimeFailoverPolicy),
		policyIDsByStrategy:  make(map[string]int),
		groups:               make(map[int]FailoverGroup),
		groupMembers:         make(map[int][]FailoverGroupMember),
		groupsByName:         make(map[string]FailoverGroup),
	}
	activeClusterCodes := make([]int, 0)
	if DB != nil && DB.Migrator().HasTable(&Cluster{}) {
		var clusters []Cluster
		if err := DB.Select("id", "type", "billing_group", "policy_id", "failover_priority").Where("status = ? AND archived = ?", ClusterStatusEnabled, false).Order("failover_priority DESC, id ASC").Find(&clusters).Error; err == nil {
			for _, cluster := range clusters {
				activeClusterCodes = append(activeClusterCodes, cluster.Id)
				cache.clusterTypes[cluster.Id] = strings.ToLower(strings.TrimSpace(cluster.Type))
				if cluster.PolicyId > 0 {
					cache.clusterPolicyIDs[cluster.Id] = cluster.PolicyId
				}
				billingGroup := strings.TrimSpace(cluster.BillingGroup)
				if billingGroup == "" {
					continue
				}
				cache.billingGroupClusters[billingGroup] = append(cache.billingGroupClusters[billingGroup], cluster.Id)
				if _, exists := cache.billingGroupPolicies[billingGroup]; !exists && cluster.PolicyId > 0 {
					cache.billingGroupPolicies[billingGroup] = cluster.PolicyId
				}
			}
		}
	}
	observability.ReplaceClusterInfo(activeClusterCodes)
	if DB != nil && DB.Migrator().HasTable(&UpstreamErrorMapping{}) {
		_ = DB.Where("enabled = ?", true).Find(&cache.mappings).Error
		sort.SliceStable(cache.mappings, func(i, j int) bool {
			return cache.mappings[i].Id < cache.mappings[j].Id
		})
	}
	if DB != nil && DB.Migrator().HasTable(&FailoverPolicy{}) {
		var policies []FailoverPolicy
		if err := DB.Where("enabled = ?", true).Order("id ASC").Find(&policies).Error; err == nil {
			for _, stored := range policies {
				mode := normalizeFailoverMode(stored.Mode)
				runtime := DefaultRuntimeFailoverPolicy(mode)
				runtime.Strategy = normalizeRoutingStrategy(stored.Strategy, mode)
				runtime = withDefaultPoolOrder(runtime)
				if stored.SamePoolRetries >= 0 {
					runtime.SamePoolRetries = stored.SamePoolRetries
				}
				if stored.MaxPoolAttempts > 0 {
					runtime.MaxPoolAttempts = stored.MaxPoolAttempts
				}
				if stored.MaxClusterAttempts > 0 {
					runtime.MaxClusterAttempts = stored.MaxClusterAttempts
				}
				if stored.MaxTotalAttempts > 0 {
					runtime.MaxTotalAttempts = stored.MaxTotalAttempts
				}
				if stored.TotalFailoverBudgetMs > 0 {
					runtime.TotalFailoverBudgetMs = stored.TotalFailoverBudgetMs
				}
				if stored.CircuitFailureThreshold > 0 {
					runtime.CircuitFailureThreshold = stored.CircuitFailureThreshold
				}
				if stored.CircuitWindowSeconds > 0 {
					runtime.CircuitWindowSeconds = stored.CircuitWindowSeconds
				}
				if stored.CircuitCooldownSeconds > 0 {
					runtime.CircuitCooldownSeconds = stored.CircuitCooldownSeconds
				}
				if stored.CircuitHalfOpenRequests > 0 {
					runtime.CircuitHalfOpenRequests = stored.CircuitHalfOpenRequests
				}
				runtime.AllowPaidEscalation = stored.AllowPaidEscalation
				runtime.AllowFallback = stored.AllowFallback
				if stored.MaxCostMultiplier > 0 {
					runtime.MaxCostMultiplier = stored.MaxCostMultiplier
				}
				if runtime.Strategy == DefaultRuntimeFailoverPolicy(mode).Strategy {
					cache.policies[mode] = runtime
				}
				cache.policiesByID[stored.Id] = runtime
				if _, exists := cache.policiesByStrategy[runtime.Strategy]; !exists {
					cache.policiesByStrategy[runtime.Strategy] = runtime
					cache.policyIDsByStrategy[runtime.Strategy] = stored.Id
				}
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&FailoverPolicyStep{}) {
		var steps []FailoverPolicyStep
		if err := DB.Order("policy_id ASC, step_order ASC").Find(&steps).Error; err == nil {
			configuredPolicies := make(map[int]struct{})
			for _, step := range steps {
				runtime, ok := cache.policiesByID[step.PolicyId]
				if !ok || step.PoolTier < PoolTierFree || step.PoolTier > PoolTierEmergency {
					continue
				}
				if _, configured := configuredPolicies[step.PolicyId]; !configured {
					runtime.PoolTiers = nil
					runtime.PoolAttemptsByTier = make(map[int]int)
					configuredPolicies[step.PolicyId] = struct{}{}
				}
				if step.MaxAttempts > 0 {
					runtime.PoolAttemptsByTier[step.PoolTier] = step.MaxAttempts
				}
				runtime.PoolTiers = append(runtime.PoolTiers, step.PoolTier)
				cache.policiesByID[step.PolicyId] = runtime
				if runtime.Strategy == DefaultRuntimeFailoverPolicy(runtime.Mode).Strategy {
					cache.policies[runtime.Mode] = runtime
				}
				if cache.policyIDsByStrategy[runtime.Strategy] == step.PolicyId {
					cache.policiesByStrategy[runtime.Strategy] = runtime
				}
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&FailoverGroup{}) {
		var groups []FailoverGroup
		if err := DB.Where("enabled = ?", true).Find(&groups).Error; err == nil {
			for _, group := range groups {
				cache.groups[group.Id] = group
				if name := strings.ToLower(strings.TrimSpace(group.Name)); name != "" {
					cache.groupsByName[name] = group
				}
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&FailoverGroupMember{}) {
		var members []FailoverGroupMember
		if err := DB.Order("priority DESC, id ASC").Find(&members).Error; err == nil {
			for _, member := range members {
				if _, enabled := cache.groups[member.FailoverGroupId]; !enabled {
					continue
				}
				cache.groupMembers[member.FailoverGroupId] = append(cache.groupMembers[member.FailoverGroupId], member)
			}
		}
	}
	if DB != nil && DB.Migrator().HasTable(&FailoverRule{}) {
		_ = DB.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&cache.rules).Error
	}
	failoverLookup.Lock()
	failoverLookup.value = cache
	failoverLookup.Unlock()
}

func ResolveRuntimeFailover(mode string, modelName string, route string, userGroup string, billingGroup string) (RuntimeFailoverPolicy, []int, bool) {
	return ResolveRuntimeFailoverWithStrategy(mode, "", modelName, route, userGroup, billingGroup)
}

func ResolveRuntimeFailoverWithStrategy(mode string, strategy string, modelName string, route string, userGroup string, billingGroup string) (RuntimeFailoverPolicy, []int, bool) {
	policy := GetRuntimeFailoverPolicy(mode)
	failoverLookup.RLock()
	defer failoverLookup.RUnlock()
	clusterOrder := append([]int(nil), failoverLookup.value.billingGroupClusters[billingGroup]...)
	if strings.TrimSpace(mode) == "" {
		if policyID := failoverLookup.value.billingGroupPolicies[billingGroup]; policyID > 0 {
			if configured, exists := failoverLookup.value.policiesByID[policyID]; exists {
				policy = configured
			}
		}
	}
	rulePolicy := false
	// A failover group whose name matches the concrete billing group acts as a
	// mixed channel. Its members become the ordered cluster candidates, so a
	// token using that group gets the same cross-cluster failover behavior as a
	// wildcard routing rule without requiring a second mapping layer.
	if mixedGroup, ok := failoverLookup.value.groupsByName[strings.ToLower(strings.TrimSpace(billingGroup))]; ok {
		if strings.TrimSpace(mode) == "" {
			if configured, exists := failoverLookup.value.policiesByID[mixedGroup.PolicyId]; exists {
				policy = configured
			}
		}
		members := failoverLookup.value.groupMembers[mixedGroup.Id]
		clusterOrder = make([]int, 0, len(members))
		seen := make(map[int]struct{}, len(members))
		for _, member := range members {
			if _, exists := seen[member.ClusterId]; exists {
				continue
			}
			seen[member.ClusterId] = struct{}{}
			clusterOrder = append(clusterOrder, member.ClusterId)
		}
		rulePolicy = len(clusterOrder) > 0
	}
	for _, rule := range failoverLookup.value.rules {
		group, ok := failoverLookup.value.groups[rule.FailoverGroupId]
		if !ok {
			continue
		}
		if rule.UserGroup != "" && rule.UserGroup != "*" && rule.UserGroup != userGroup {
			continue
		}
		if !matchesFailoverPattern(rule.ModelPattern, modelName) || !matchesFailoverPattern(rule.RoutePattern, route) {
			continue
		}
		if strings.TrimSpace(mode) == "" {
			policyID := rule.PolicyId
			if policyID <= 0 {
				policyID = group.PolicyId
			}
			if configured, exists := failoverLookup.value.policiesByID[policyID]; exists {
				policy = configured
			}
		}
		rulePolicy = true
		members := failoverLookup.value.groupMembers[group.Id]
		clusterOrder = make([]int, 0, len(members))
		seen := make(map[int]struct{}, len(members))
		for _, member := range members {
			if _, exists := seen[member.ClusterId]; exists {
				continue
			}
			seen[member.ClusterId] = struct{}{}
			clusterOrder = append(clusterOrder, member.ClusterId)
		}
		break
	}
	if strings.TrimSpace(mode) == "" && IsRoutingStrategy(strategy) {
		normalizedStrategy := strings.ToLower(strings.TrimSpace(strategy))
		if configured, exists := failoverLookup.value.policiesByStrategy[normalizedStrategy]; exists {
			policy = configured
		} else {
			policy.Strategy = normalizedStrategy
			policy = withDefaultPoolOrder(policy)
		}
	}
	return policy, clusterOrder, rulePolicy
}

func GetRuntimeFailoverPolicyForCluster(clusterID int, fallback RuntimeFailoverPolicy) RuntimeFailoverPolicy {
	if clusterID <= 0 {
		return fallback
	}
	failoverLookup.RLock()
	policyID := failoverLookup.value.clusterPolicyIDs[clusterID]
	policy, exists := failoverLookup.value.policiesByID[policyID]
	failoverLookup.RUnlock()
	if !exists {
		return fallback
	}
	return policy
}

func matchesFailoverPattern(pattern string, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "*" {
		return true
	}
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func MatchUpstreamErrorMapping(clusterID int, rawCode string, statusCode int) (UpstreamErrorMapping, bool) {
	failoverLookup.RLock()
	defer failoverLookup.RUnlock()
	clusterType := failoverLookup.value.clusterTypes[clusterID]

	rawCode = strings.ToLower(strings.TrimSpace(rawCode))
	bestScore := -1
	var best UpstreamErrorMapping
	for _, mapping := range failoverLookup.value.mappings {
		mappingClusterType := strings.ToLower(strings.TrimSpace(mapping.ClusterType))
		mappingRawCode := strings.ToLower(strings.TrimSpace(mapping.RawCode))
		if mappingClusterType != "" && mappingClusterType != "*" && mappingClusterType != clusterType {
			continue
		}
		if mappingRawCode != "" && mappingRawCode != "*" && mappingRawCode != rawCode {
			continue
		}
		if mapping.StatusCode != 0 && mapping.StatusCode != statusCode {
			continue
		}
		score := 0
		if mappingClusterType != "" && mappingClusterType != "*" {
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

func GetFailoverConfig() (*FailoverConfig, error) {
	config := &FailoverConfig{}
	queries := []struct {
		order string
		value any
	}{
		{"id ASC", &config.Clusters},
		{"cluster_id ASC, tier ASC", &config.Pools},
		{"id ASC", &config.Policies},
		{"policy_id ASC, step_order ASC", &config.PolicySteps},
		{"id ASC", &config.Groups},
		{"failover_group_id ASC, priority DESC", &config.GroupMembers},
		{"priority DESC, id ASC", &config.Rules},
		{"cluster_type ASC, raw_code ASC, status_code ASC", &config.ErrorMappings},
	}
	for _, query := range queries {
		if err := DB.Order(query.order).Find(query.value).Error; err != nil {
			return nil, err
		}
	}
	for i := range config.Policies {
		config.Policies[i].Strategy = normalizeRoutingStrategy(config.Policies[i].Strategy, config.Policies[i].Mode)
	}
	return config, nil
}

func SaveFailoverConfig(config *FailoverConfig) error {
	if config == nil {
		return errors.New("failover config is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for i := range config.Clusters {
			cluster := &config.Clusters[i]
			cluster.Name = strings.TrimSpace(cluster.Name)
			cluster.Type = strings.TrimSpace(cluster.Type)
			if cluster.Name == "" || cluster.Type == "" {
				return errors.New("cluster name and type are required")
			}
			if err := tx.Save(cluster).Error; err != nil {
				return err
			}
		}
		for i := range config.Pools {
			pool := &config.Pools[i]
			if pool.ClusterId <= 0 || pool.Tier < PoolTierFree || pool.Tier > PoolTierEmergency {
				return errors.New("pool cluster_id and tier are invalid")
			}
			if pool.CostFactor < 0 {
				return errors.New("pool cost_factor cannot be negative")
			}
			if err := tx.Save(pool).Error; err != nil {
				return err
			}
		}
		for i := range config.Policies {
			policy := &config.Policies[i]
			policy.Mode = normalizeFailoverMode(policy.Mode)
			policy.Strategy = normalizeRoutingStrategy(policy.Strategy, policy.Mode)
			if policy.Name == "" {
				policy.Name = policy.Mode
			}
			if policy.SamePoolRetries < 0 || policy.MaxPoolAttempts <= 0 || policy.MaxClusterAttempts <= 0 || policy.MaxTotalAttempts <= 0 || policy.TotalFailoverBudgetMs <= 0 {
				return errors.New("policy attempts and failover budget are invalid")
			}
			if policy.CircuitFailureThreshold <= 0 || policy.CircuitWindowSeconds <= 0 || policy.CircuitCooldownSeconds <= 0 || policy.CircuitHalfOpenRequests <= 0 {
				return errors.New("policy circuit settings must be positive")
			}
			if policy.MaxTotalAttempts < policy.MaxClusterAttempts {
				policy.MaxTotalAttempts = policy.MaxClusterAttempts
			}
			if policy.MaxCostMultiplier <= 0 {
				policy.MaxCostMultiplier = 1
			}
			if err := tx.Save(policy).Error; err != nil {
				return err
			}
		}
		stepOrders := make(map[[2]int]struct{}, len(config.PolicySteps))
		stepTiers := make(map[[2]int]struct{}, len(config.PolicySteps))
		for i := range config.PolicySteps {
			step := &config.PolicySteps[i]
			if step.PolicyId <= 0 || step.StepOrder <= 0 || step.PoolTier < PoolTierFree || step.PoolTier > PoolTierEmergency || step.MaxAttempts <= 0 {
				return errors.New("policy step is invalid")
			}
			orderKey := [2]int{step.PolicyId, step.StepOrder}
			tierKey := [2]int{step.PolicyId, step.PoolTier}
			if _, duplicate := stepOrders[orderKey]; duplicate {
				return errors.New("policy step order must be unique within a policy")
			}
			if _, duplicate := stepTiers[tierKey]; duplicate {
				return errors.New("pool tier must be unique within a policy")
			}
			stepOrders[orderKey] = struct{}{}
			stepTiers[tierKey] = struct{}{}
		}
		policyIDsForSteps := make([]int, 0, len(config.Policies))
		for _, policy := range config.Policies {
			if policy.Id > 0 {
				policyIDsForSteps = append(policyIDsForSteps, policy.Id)
			}
		}
		if len(policyIDsForSteps) > 0 {
			stepQuery := tx.Where("policy_id IN ?", policyIDsForSteps)
			stepIDs := make([]int, 0, len(config.PolicySteps))
			for _, step := range config.PolicySteps {
				if step.Id > 0 {
					stepIDs = append(stepIDs, step.Id)
				}
			}
			if len(stepIDs) > 0 {
				stepQuery = stepQuery.Where("id NOT IN ?", stepIDs)
			}
			if err := stepQuery.Delete(&FailoverPolicyStep{}).Error; err != nil {
				return err
			}
		}
		// Temporarily move retained rows out of the unique key space as well.
		// A simple swap (P1 <-> P2) keeps both IDs in the payload, so deleting
		// only removed rows is not enough to avoid a transient unique-index hit.
		for _, step := range config.PolicySteps {
			if step.Id <= 0 {
				continue
			}
			if err := tx.Model(&FailoverPolicyStep{}).Where("id = ?", step.Id).Updates(map[string]any{
				"policy_id": -step.Id,
				"step_order": -step.Id,
			}).Error; err != nil {
				return err
			}
		}
		// Remove rows that are no longer part of the submitted configuration
		// before upserting the new steps. This matters when an administrator
		// reorders or replaces a step: saving first would hit the unique
		// (policy_id, step_order) index against the old row and roll back the
		// entire transaction.
		for i := range config.PolicySteps {
			if err := tx.Save(&config.PolicySteps[i]).Error; err != nil {
				return err
			}
		}
		groupNames := make(map[string]struct{}, len(config.Groups))
		for i := range config.Groups {
			group := &config.Groups[i]
			group.Name = strings.TrimSpace(group.Name)
			if group.Name == "" || group.PolicyId <= 0 {
				return errors.New("failover group name and policy_id are required")
			}
			groupName := strings.ToLower(group.Name)
			if _, exists := groupNames[groupName]; exists {
				return fmt.Errorf("failover group name %q is duplicated", group.Name)
			}
			groupNames[groupName] = struct{}{}
			if err := tx.Save(group).Error; err != nil {
				return err
			}
		}
		for i := range config.GroupMembers {
			member := &config.GroupMembers[i]
			if member.FailoverGroupId <= 0 || member.ClusterId <= 0 || member.Weight < 0 {
				return errors.New("failover group member is invalid")
			}
			if err := tx.Save(member).Error; err != nil {
				return err
			}
		}
		for i := range config.Rules {
			rule := &config.Rules[i]
			rule.ModelPattern = strings.TrimSpace(rule.ModelPattern)
			rule.RoutePattern = strings.TrimSpace(rule.RoutePattern)
			rule.UserGroup = strings.TrimSpace(rule.UserGroup)
			if rule.FailoverGroupId <= 0 {
				return errors.New("failover rule group is required")
			}
			if rule.ModelPattern == "" {
				rule.ModelPattern = "*"
			}
			if rule.RoutePattern == "" {
				rule.RoutePattern = "*"
			}
			if rule.UserGroup == "" {
				rule.UserGroup = "*"
			}
			if err := tx.Save(rule).Error; err != nil {
				return err
			}
		}
		if len(config.ErrorMappings) > 0 {
			for i := range config.ErrorMappings {
				mapping := &config.ErrorMappings[i]
				mapping.ClusterType = strings.ToLower(strings.TrimSpace(mapping.ClusterType))
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
				if mapping.Category == "" || mapping.FailureScope == "" || mapping.Action == "" {
					return errors.New("error mapping category, failure_scope, and action are required")
				}
				switch mapping.FailureScope {
				case "request", "credential", "channel", "cluster", "provider":
				default:
					return errors.New("error mapping failure_scope is invalid")
				}
				switch mapping.Action {
				case "none", "failover", "retry_later", "abort", "manual":
				default:
					return errors.New("error mapping action is invalid")
				}
			}
			if err := tx.Save(&config.ErrorMappings).Error; err != nil {
				return err
			}
		}
		clusterIDs := make([]int, 0, len(config.Clusters))
		for _, cluster := range config.Clusters {
			if cluster.Id > 0 {
				clusterIDs = append(clusterIDs, cluster.Id)
			}
		}
		clusterQuery := tx.Model(&Cluster{}).Where("archived = ?", false)
		if len(clusterIDs) > 0 {
			clusterQuery = clusterQuery.Where("id NOT IN ?", clusterIDs)
		}
		if err := clusterQuery.Updates(map[string]any{"archived": true, "status": ClusterStatusDisabled}).Error; err != nil {
			return err
		}

		poolIDs := make([]int, 0, len(config.Pools))
		for _, pool := range config.Pools {
			if pool.Id > 0 {
				poolIDs = append(poolIDs, pool.Id)
			}
		}
		poolQuery := tx.Model(&ClusterPool{}).Where("status = ?", ClusterStatusEnabled)
		if len(poolIDs) > 0 {
			poolQuery = poolQuery.Where("id NOT IN ?", poolIDs)
		}
		if err := poolQuery.Update("status", ClusterStatusDisabled).Error; err != nil {
			return err
		}

		policyIDs := make([]int, 0, len(config.Policies))
		for _, policy := range config.Policies {
			if policy.Id > 0 {
				policyIDs = append(policyIDs, policy.Id)
			}
		}
		policyQuery := tx.Model(&FailoverPolicy{}).Where("enabled = ?", true)
		if len(policyIDs) > 0 {
			policyQuery = policyQuery.Where("id NOT IN ?", policyIDs)
		}
		if err := policyQuery.Update("enabled", false).Error; err != nil {
			return err
		}

		groupIDs := make([]int, 0, len(config.Groups))
		for _, group := range config.Groups {
			if group.Id > 0 {
				groupIDs = append(groupIDs, group.Id)
			}
		}
		groupQuery := tx.Model(&FailoverGroup{}).Where("enabled = ?", true)
		if len(groupIDs) > 0 {
			groupQuery = groupQuery.Where("id NOT IN ?", groupIDs)
		}
		if err := groupQuery.Update("enabled", false).Error; err != nil {
			return err
		}

		ruleIDs := make([]int, 0, len(config.Rules))
		for _, rule := range config.Rules {
			if rule.Id > 0 {
				ruleIDs = append(ruleIDs, rule.Id)
			}
		}
		ruleQuery := tx.Model(&FailoverRule{}).Where("enabled = ?", true)
		if len(ruleIDs) > 0 {
			ruleQuery = ruleQuery.Where("id NOT IN ?", ruleIDs)
		}
		if err := ruleQuery.Update("enabled", false).Error; err != nil {
			return err
		}

		mappingIDs := make([]int, 0, len(config.ErrorMappings))
		for _, mapping := range config.ErrorMappings {
			if mapping.Id > 0 {
				mappingIDs = append(mappingIDs, mapping.Id)
			}
		}
		mappingQuery := tx.Model(&UpstreamErrorMapping{}).Where("enabled = ?", true)
		if len(mappingIDs) > 0 {
			mappingQuery = mappingQuery.Where("id NOT IN ?", mappingIDs)
		}
		if err := mappingQuery.Update("enabled", false).Error; err != nil {
			return err
		}
		return nil
	})
}

func normalizeFailoverMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case FailoverModeConservative:
		return FailoverModeConservative
	case FailoverModeAggressive:
		return FailoverModeAggressive
	default:
		return FailoverModeBalanced
	}
}

func normalizeRoutingStrategy(strategy string, mode string) string {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case RoutingStrategyCostFirst, RoutingStrategyBalanced, RoutingStrategyStabilityFirst, RoutingStrategyProCostFirst, RoutingStrategyProStabilityFirst:
		return strings.ToLower(strings.TrimSpace(strategy))
	default:
		switch normalizeFailoverMode(mode) {
		case FailoverModeConservative:
			return RoutingStrategyStabilityFirst
		case FailoverModeAggressive:
			return RoutingStrategyCostFirst
		default:
			return RoutingStrategyBalanced
		}
	}
}

func IsRoutingStrategy(strategy string) bool {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case RoutingStrategyCostFirst, RoutingStrategyBalanced, RoutingStrategyStabilityFirst, RoutingStrategyProCostFirst, RoutingStrategyProStabilityFirst:
		return true
	default:
		return false
	}
}
