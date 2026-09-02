package ratio_setting

import (
	"errors"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

const (
	PricingGroupRetryModeFixed          = "fixed"
	PricingGroupRetryModeActiveChannels = "active_channels"
	MaxPricingGroupRetryTimes           = 100
	// The values below are the initial strategy IDs. They are not an enum:
	// administrators may create arbitrary strategy IDs and names.
	PricingGroupRoutingStrategyPriceFirst = "price_first"
	PricingGroupRoutingStrategyBalanced   = "balanced"
	PricingGroupRoutingStrategyStable     = "stable"
)

type PricingGroupRetryPolicy struct {
	Mode       string `json:"mode"`
	RetryTimes int    `json:"retry_times"`
}

// PricingGroupRoutingStrategy is an independent strategy definition. Strategy
// contains the stable catalog ID for runtime use and is intentionally omitted
// from JSON; the ID is the key in PricingGroupRoutingConfiguration.Strategies.
// Weights are percentages and must add up to 100.
type PricingGroupRoutingStrategy struct {
	Strategy           string  `json:"-"`
	Name               string  `json:"name"`
	PriceWeight        float64 `json:"price_weight"`
	AvailabilityWeight float64 `json:"availability_weight"`
	LoadWeight         float64 `json:"load_weight"`
	TTFTWeight         float64 `json:"ttft_weight"`
}

// PricingGroupRoutingConfiguration stores the strategy catalog separately
// from per-group references. A strategy can therefore be reused by many
// pricing groups and can exist without being assigned to a group.
type PricingGroupRoutingConfiguration struct {
	Strategies    map[string]PricingGroupRoutingStrategy `json:"strategies"`
	GroupBindings map[string]string                      `json:"group_bindings"`
}

func DefaultPricingGroupRoutingStrategy() PricingGroupRoutingStrategy {
	return PricingGroupRoutingStrategy{
		Strategy:           PricingGroupRoutingStrategyBalanced,
		PriceWeight:        40,
		AvailabilityWeight: 40,
		LoadWeight:         20,
		TTFTWeight:         0,
	}
}

func PricingGroupRoutingStrategyPreset(strategy string) PricingGroupRoutingStrategy {
	result := DefaultPricingGroupRoutingStrategy()
	switch strings.TrimSpace(strategy) {
	case PricingGroupRoutingStrategyPriceFirst:
		result.Strategy = PricingGroupRoutingStrategyPriceFirst
		result.PriceWeight, result.AvailabilityWeight, result.LoadWeight, result.TTFTWeight = 65, 20, 15, 0
	case PricingGroupRoutingStrategyStable:
		result.Strategy = PricingGroupRoutingStrategyStable
		result.PriceWeight, result.AvailabilityWeight, result.LoadWeight, result.TTFTWeight = 20, 60, 20, 0
	}
	return result
}

// DefaultPricingGroupRoutingStrategies returns the initial catalog. These are
// ordinary CRUD records; the IDs are merely stable defaults and are not
// restricted during validation.
func DefaultPricingGroupRoutingStrategies() map[string]PricingGroupRoutingStrategy {
	result := make(map[string]PricingGroupRoutingStrategy, 3)
	for _, id := range []string{
		PricingGroupRoutingStrategyPriceFirst,
		PricingGroupRoutingStrategyBalanced,
		PricingGroupRoutingStrategyStable,
	} {
		strategy := PricingGroupRoutingStrategyPreset(id)
		switch id {
		case PricingGroupRoutingStrategyPriceFirst:
			strategy.Name = "价格优先"
		case PricingGroupRoutingStrategyBalanced:
			strategy.Name = "均衡"
		case PricingGroupRoutingStrategyStable:
			strategy.Name = "稳定"
		}
		result[id] = strategy
	}
	return result
}

type PricingGroupConfiguration struct {
	GroupRatios             map[string]float64
	GroupEnabled            map[string]bool
	GroupOrder              []string
	RetryPolicies           map[string]PricingGroupRetryPolicy
	RoutingStrategies       map[string]PricingGroupRoutingStrategy
	RoutingStrategyBindings map[string]string
}

type pricingGroupSnapshot struct {
	groupRatios             map[string]float64
	groupEnabled            map[string]bool
	groupOrder              []string
	retryPolicies           map[string]PricingGroupRetryPolicy
	routingStrategies       map[string]PricingGroupRoutingStrategy
	routingStrategyBindings map[string]string
}

var pricingGroupSnapshotValue atomic.Pointer[pricingGroupSnapshot]
var pricingGroupSnapshotUpdateMutex sync.Mutex

type groupRatioConfigValue struct{}
type pricingGroupRetryPolicyConfigValue struct{}
type pricingGroupRoutingStrategyConfigValue struct{}

type GroupRatioSetting struct {
	GroupRatio                  *groupRatioConfigValue                  `json:"group_ratio"`
	PricingGroupRetryPolicy     *pricingGroupRetryPolicyConfigValue     `json:"pricing_group_retry_policy"`
	PricingGroupRoutingStrategy *pricingGroupRoutingStrategyConfigValue `json:"pricing_group_routing_strategy"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		defaultGroupRatio,
		defaultPricingGroupEnabled(defaultGroupRatio),
		[]string{"default", "vip", "svip"},
		nil,
		defaultPricingGroupRoutingStrategies(),
		map[string]string{
			"default": PricingGroupRoutingStrategyBalanced,
			"vip":     PricingGroupRoutingStrategyBalanced,
			"svip":    PricingGroupRoutingStrategyBalanced,
		},
	))
	groupRatioSetting = GroupRatioSetting{
		GroupRatio:                  &groupRatioConfigValue{},
		PricingGroupRetryPolicy:     &pricingGroupRetryPolicyConfigValue{},
		PricingGroupRoutingStrategy: &pricingGroupRoutingStrategyConfigValue{},
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func newPricingGroupSnapshot(
	ratios map[string]float64,
	enabled map[string]bool,
	order []string,
	retryPolicies map[string]PricingGroupRetryPolicy,
	routingStrategies map[string]PricingGroupRoutingStrategy,
	routingStrategyBindings map[string]string,
) *pricingGroupSnapshot {
	ratioCopy := make(map[string]float64, len(ratios))
	for group, ratio := range ratios {
		ratioCopy[group] = ratio
	}
	enabledCopy := make(map[string]bool, len(enabled))
	for group, groupEnabled := range enabled {
		enabledCopy[group] = groupEnabled
	}
	policyCopy := make(map[string]PricingGroupRetryPolicy, len(retryPolicies))
	for group, policy := range retryPolicies {
		policyCopy[group] = policy
	}
	strategyCopy := make(map[string]PricingGroupRoutingStrategy, len(routingStrategies))
	for group, strategy := range routingStrategies {
		// The map key is the only persisted identity. Ignore any stale value
		// supplied in the in-memory struct and keep runtime IDs coherent.
		strategy.Strategy = group
		strategyCopy[group] = strategy
	}
	bindingCopy := make(map[string]string)
	for group, strategyID := range routingStrategyBindings {
		bindingCopy[group] = strategyID
	}

	configuredOrder := append([]string(nil), order...)
	ordered := make([]string, 0, len(ratioCopy))
	seen := make(map[string]struct{}, len(ratioCopy))
	for _, group := range configuredOrder {
		if _, exists := ratioCopy[group]; !exists {
			continue
		}
		if _, exists := seen[group]; exists {
			continue
		}
		seen[group] = struct{}{}
		ordered = append(ordered, group)
	}
	remaining := make([]string, 0, len(ratioCopy)-len(ordered))
	for group := range ratioCopy {
		if _, exists := seen[group]; !exists {
			remaining = append(remaining, group)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)

	return &pricingGroupSnapshot{
		groupRatios:             ratioCopy,
		groupEnabled:            enabledCopy,
		groupOrder:              ordered,
		retryPolicies:           policyCopy,
		routingStrategies:       strategyCopy,
		routingStrategyBindings: bindingCopy,
	}
}

func currentPricingGroupSnapshot() *pricingGroupSnapshot {
	return pricingGroupSnapshotValue.Load()
}

func (groupRatioConfigValue) MarshalJSON() ([]byte, error) {
	return common.Marshal(currentPricingGroupSnapshot().groupRatios)
}

func (*groupRatioConfigValue) UnmarshalJSON(data []byte) error {
	return UpdateGroupRatioByJSONString(string(data))
}

func (pricingGroupRetryPolicyConfigValue) MarshalJSON() ([]byte, error) {
	return common.Marshal(currentPricingGroupSnapshot().retryPolicies)
}

func (*pricingGroupRetryPolicyConfigValue) UnmarshalJSON(data []byte) error {
	return UpdatePricingGroupRetryPolicyByJSONString(string(data))
}

func (pricingGroupRoutingStrategyConfigValue) MarshalJSON() ([]byte, error) {
	snapshot := currentPricingGroupSnapshot()
	return common.Marshal(PricingGroupRoutingConfiguration{
		Strategies:    snapshot.routingStrategies,
		GroupBindings: snapshot.routingStrategyBindings,
	})
}

func (*pricingGroupRoutingStrategyConfigValue) UnmarshalJSON(data []byte) error {
	return UpdatePricingGroupRoutingStrategyByJSONString(string(data))
}

func GetGroupRatioCopy() map[string]float64 {
	snapshot := currentPricingGroupSnapshot()
	result := make(map[string]float64, len(snapshot.groupRatios))
	for group, ratio := range snapshot.groupRatios {
		result[group] = ratio
	}
	return result
}

func ContainsGroupRatio(name string) bool {
	_, ok := currentPricingGroupSnapshot().groupRatios[name]
	return ok
}

func GetPricingGroupEnabledCopy() map[string]bool {
	snapshot := currentPricingGroupSnapshot()
	result := make(map[string]bool, len(snapshot.groupEnabled))
	for group, enabled := range snapshot.groupEnabled {
		result[group] = enabled
	}
	return result
}

func IsPricingGroupEnabled(name string) bool {
	return currentPricingGroupSnapshot().groupEnabled[strings.TrimSpace(name)]
}

func PricingGroupEnabled2JSONString() string {
	data, err := common.Marshal(currentPricingGroupSnapshot().groupEnabled)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func UpdatePricingGroupEnabledByJSONString(jsonStr string) error {
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	snapshot := currentPricingGroupSnapshot()
	enabled, err := parsePricingGroupEnabled(jsonStr, snapshot.groupRatios)
	if err != nil {
		return err
	}
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		snapshot.groupRatios,
		enabled,
		snapshot.groupOrder,
		snapshot.retryPolicies,
		snapshot.routingStrategies,
		snapshot.routingStrategyBindings,
	))
	return nil
}

func GroupRatio2JSONString() string {
	data, err := common.Marshal(currentPricingGroupSnapshot().groupRatios)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	ratios, err := parseGroupRatios(jsonStr)
	if err != nil {
		return err
	}
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	snapshot := currentPricingGroupSnapshot()
	enabled := make(map[string]bool, len(ratios))
	for group := range ratios {
		groupEnabled, exists := snapshot.groupEnabled[group]
		if !exists {
			groupEnabled = true
		}
		enabled[group] = groupEnabled
	}
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		ratios,
		enabled,
		snapshot.groupOrder,
		snapshot.retryPolicies,
		snapshot.routingStrategies,
		snapshot.routingStrategyBindings,
	))
	return nil
}

func GetPricingGroupRetryPolicy(group string) (PricingGroupRetryPolicy, bool) {
	group = strings.TrimSpace(group)
	snapshot := currentPricingGroupSnapshot()
	if policy, exists := snapshot.retryPolicies[group]; exists {
		return policy, true
	}
	if _, exists := snapshot.groupRatios[group]; !exists {
		return PricingGroupRetryPolicy{}, false
	}
	return PricingGroupRetryPolicy{Mode: PricingGroupRetryModeActiveChannels}, true
}

func GetPricingGroupRetryPolicyCopy() map[string]PricingGroupRetryPolicy {
	snapshot := currentPricingGroupSnapshot()
	result := make(map[string]PricingGroupRetryPolicy, len(snapshot.retryPolicies))
	for group, policy := range snapshot.retryPolicies {
		result[group] = policy
	}
	return result
}

func GetPricingGroupRoutingStrategy(group string) (PricingGroupRoutingStrategy, bool) {
	group = strings.TrimSpace(group)
	snapshot := currentPricingGroupSnapshot()
	if strategyID, bound := snapshot.routingStrategyBindings[group]; bound {
		if strategy, exists := snapshot.routingStrategies[strategyID]; exists {
			if strategy.Strategy == "" {
				strategy.Strategy = strategyID
			}
			// Name is catalog metadata for the admin UI. Runtime scoring only
			// consumes the ID and weights.
			strategy.Name = ""
			return strategy, true
		}
	}
	if _, exists := snapshot.groupRatios[group]; !exists {
		return PricingGroupRoutingStrategy{}, false
	}
	// A malformed or partially loaded configuration must not take routing
	// down. The persisted configuration validator rejects this state; runtime
	// callers still receive the documented balanced default while it is being
	// repaired.
	return DefaultPricingGroupRoutingStrategy(), true
}

func GetPricingGroupRoutingStrategyCopy() map[string]PricingGroupRoutingStrategy {
	snapshot := currentPricingGroupSnapshot()
	result := make(map[string]PricingGroupRoutingStrategy, len(snapshot.routingStrategies))
	for strategyID, strategy := range snapshot.routingStrategies {
		if strategy.Strategy == "" {
			strategy.Strategy = strategyID
		}
		result[strategyID] = strategy
	}
	return result
}

// GetPricingGroupRoutingStrategyBindingsCopy returns the independent
// pricing-group -> strategy ID references.
func GetPricingGroupRoutingStrategyBindingsCopy() map[string]string {
	snapshot := currentPricingGroupSnapshot()
	result := make(map[string]string, len(snapshot.routingStrategyBindings))
	for group, strategyID := range snapshot.routingStrategyBindings {
		result[group] = strategyID
	}
	return result
}

// GetPricingGroupRoutingConfiguration returns a deep copy suitable for an
// admin editor or API response.
func GetPricingGroupRoutingConfiguration() PricingGroupRoutingConfiguration {
	snapshot := currentPricingGroupSnapshot()
	strategies := make(map[string]PricingGroupRoutingStrategy, len(snapshot.routingStrategies))
	for strategyID, strategy := range snapshot.routingStrategies {
		if strategy.Strategy == "" {
			strategy.Strategy = strategyID
		}
		strategies[strategyID] = strategy
	}
	bindings := make(map[string]string, len(snapshot.routingStrategyBindings))
	for group, strategyID := range snapshot.routingStrategyBindings {
		bindings[group] = strategyID
	}
	return PricingGroupRoutingConfiguration{Strategies: strategies, GroupBindings: bindings}
}

func PricingGroupRoutingStrategy2JSONString() string {
	data, err := common.Marshal(GetPricingGroupRoutingConfiguration())
	if err != nil {
		return `{"strategies":{},"group_bindings":{}}`
	}
	return string(data)
}

func UpdatePricingGroupRoutingStrategyByJSONString(jsonStr string) error {
	configuration, err := ParsePricingGroupRoutingConfiguration(
		jsonStr,
		currentPricingGroupSnapshot().groupRatios,
	)
	if err != nil {
		return err
	}
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	snapshot := currentPricingGroupSnapshot()
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		snapshot.groupRatios,
		snapshot.groupEnabled,
		snapshot.groupOrder,
		snapshot.retryPolicies,
		configuration.Strategies,
		configuration.GroupBindings,
	))
	return nil
}

func PricingGroupRetryPolicy2JSONString() string {
	data, err := common.Marshal(currentPricingGroupSnapshot().retryPolicies)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func UpdatePricingGroupRetryPolicyByJSONString(jsonStr string) error {
	policies, err := parsePricingGroupRetryPolicies(jsonStr)
	if err != nil {
		return err
	}
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	snapshot := currentPricingGroupSnapshot()
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		snapshot.groupRatios,
		snapshot.groupEnabled,
		snapshot.groupOrder,
		policies,
		snapshot.routingStrategies,
		snapshot.routingStrategyBindings,
	))
	return nil
}

func ApplyPricingGroupConfiguration(configuration *PricingGroupConfiguration) {
	if configuration == nil {
		return
	}
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		configuration.GroupRatios,
		configuration.GroupEnabled,
		configuration.GroupOrder,
		configuration.RetryPolicies,
		configuration.RoutingStrategies,
		configuration.RoutingStrategyBindings,
	))
}

func ParsePricingGroupConfiguration(groupRatioJSON, groupEnabledJSON, groupOrderJSON, retryPolicyJSON, routingStrategyJSON string) (*PricingGroupConfiguration, error) {
	ratioMap, err := parseGroupRatios(groupRatioJSON)
	if err != nil {
		return nil, err
	}
	enabledMap, err := parsePricingGroupEnabled(groupEnabledJSON, ratioMap)
	if err != nil {
		return nil, err
	}
	order, err := parsePricingGroupOrder(groupOrderJSON)
	if err != nil {
		return nil, err
	}
	if len(order) != len(ratioMap) {
		return nil, errors.New("定价分组顺序必须包含全部定价分组")
	}
	for _, group := range order {
		if _, exists := ratioMap[group]; !exists {
			return nil, errors.New("定价分组顺序包含不存在的分组: " + group)
		}
	}

	retryPolicies, err := parsePricingGroupRetryPolicies(retryPolicyJSON)
	if err != nil {
		return nil, err
	}
	if len(retryPolicies) != len(ratioMap) {
		return nil, errors.New("重试策略必须覆盖全部定价分组")
	}
	for group := range retryPolicies {
		if _, exists := ratioMap[group]; !exists {
			return nil, errors.New("重试策略包含不存在的定价分组: " + group)
		}
	}

	routingConfiguration, err := ParsePricingGroupRoutingConfiguration(routingStrategyJSON, ratioMap)
	if err != nil {
		return nil, err
	}

	return &PricingGroupConfiguration{
		GroupRatios:             ratioMap,
		GroupEnabled:            enabledMap,
		GroupOrder:              order,
		RetryPolicies:           retryPolicies,
		RoutingStrategies:       routingConfiguration.Strategies,
		RoutingStrategyBindings: routingConfiguration.GroupBindings,
	}, nil
}

// ParsePersistedPricingGroupConfiguration loads independently stored
// pricing-group options. An absent enabled map initializes every configured
// group to enabled. An absent routing configuration is initialized to the
// built-in catalog; a non-empty value must use the current catalog/bindings
// shape and is never interpreted as the retired per-group strategy map.
func ParsePersistedPricingGroupConfiguration(groupRatioJSON, groupEnabledJSON, groupOrderJSON, retryPolicyJSON, routingStrategyJSON string) (*PricingGroupConfiguration, error) {
	ratioMap, err := parseGroupRatios(groupRatioJSON)
	if err != nil {
		return nil, err
	}
	var enabledMap map[string]bool
	if strings.TrimSpace(groupEnabledJSON) == "" {
		enabledMap = defaultPricingGroupEnabled(ratioMap)
	} else {
		enabledMap, err = parsePricingGroupEnabled(groupEnabledJSON, ratioMap)
		if err != nil {
			return nil, err
		}
	}
	order, err := parsePricingGroupOrder(groupOrderJSON)
	if err != nil {
		return nil, err
	}
	retryPolicies, err := parsePricingGroupRetryPolicies(retryPolicyJSON)
	if err != nil {
		return nil, err
	}
	if len(retryPolicies) == 0 {
		retryPolicies = defaultPricingGroupRetryPolicies(ratioMap)
	} else if err := validatePricingGroupMapCoverage("重试策略", ratioMap, retryPolicies); err != nil {
		return nil, err
	}

	var routingConfiguration PricingGroupRoutingConfiguration
	trimmedRoutingJSON := strings.TrimSpace(routingStrategyJSON)
	if trimmedRoutingJSON == "" || trimmedRoutingJSON == "{}" || trimmedRoutingJSON == "null" {
		routingConfiguration = defaultPricingGroupRoutingConfiguration(ratioMap)
	} else {
		routingConfiguration, err = ParsePricingGroupRoutingConfiguration(routingStrategyJSON, ratioMap)
	}
	if err != nil {
		return nil, err
	}

	snapshot := newPricingGroupSnapshot(
		ratioMap,
		enabledMap,
		order,
		retryPolicies,
		routingConfiguration.Strategies,
		routingConfiguration.GroupBindings,
	)

	return &PricingGroupConfiguration{
		GroupRatios:             snapshot.groupRatios,
		GroupEnabled:            snapshot.groupEnabled,
		GroupOrder:              snapshot.groupOrder,
		RetryPolicies:           snapshot.retryPolicies,
		RoutingStrategies:       snapshot.routingStrategies,
		RoutingStrategyBindings: snapshot.routingStrategyBindings,
	}, nil
}

func defaultPricingGroupEnabled(ratios map[string]float64) map[string]bool {
	result := make(map[string]bool, len(ratios))
	for group := range ratios {
		result[group] = true
	}
	return result
}

func parsePricingGroupEnabled(jsonStr string, ratios map[string]float64) (map[string]bool, error) {
	var enabled map[string]bool
	if err := common.UnmarshalJsonStr(jsonStr, &enabled); err != nil {
		return nil, err
	}
	if enabled == nil {
		return nil, errors.New("定价分组启用状态必须是 JSON 对象")
	}
	if err := validatePricingGroupMapCoverage("启用状态", ratios, enabled); err != nil {
		return nil, err
	}
	return enabled, nil
}

func defaultPricingGroupRetryPolicies(ratios map[string]float64) map[string]PricingGroupRetryPolicy {
	result := make(map[string]PricingGroupRetryPolicy, len(ratios))
	for group := range ratios {
		result[group] = PricingGroupRetryPolicy{Mode: PricingGroupRetryModeActiveChannels}
	}
	return result
}

func validatePricingGroupMapCoverage[T any](label string, ratios map[string]float64, values map[string]T) error {
	if len(values) != len(ratios) {
		return errors.New(label + "必须覆盖全部定价分组")
	}
	for group := range values {
		if _, exists := ratios[group]; !exists {
			return errors.New(label + "包含不存在的定价分组: " + group)
		}
	}
	return nil
}

func parsePricingGroupRetryPolicies(jsonStr string) (map[string]PricingGroupRetryPolicy, error) {
	var policies map[string]PricingGroupRetryPolicy
	if err := common.UnmarshalJsonStr(jsonStr, &policies); err != nil {
		return nil, err
	}
	if policies == nil {
		return nil, errors.New("定价分组重试策略必须是 JSON 对象")
	}
	for group, policy := range policies {
		if group == "" || group != strings.TrimSpace(group) {
			return nil, errors.New("定价分组重试策略包含无效的分组名")
		}
		switch policy.Mode {
		case PricingGroupRetryModeFixed:
			if policy.RetryTimes < 0 || policy.RetryTimes > MaxPricingGroupRetryTimes {
				return nil, errors.New("分组 " + group + " 的固定重试次数必须在 0 到 100 之间")
			}
		case PricingGroupRetryModeActiveChannels:
			policy.RetryTimes = 0
			policies[group] = policy
		default:
			return nil, errors.New("分组 " + group + " 的重试模式无效")
		}
	}
	return policies, nil
}

// ParsePricingGroupRoutingConfiguration validates the independently managed
// strategy catalog and its group references. Passing nil groupRatios validates
// only the catalog shape; otherwise every pricing group must have exactly one
// binding and every binding must reference an existing strategy.
func ParsePricingGroupRoutingConfiguration(
	jsonStr string,
	groupRatios map[string]float64,
) (PricingGroupRoutingConfiguration, error) {
	type rawRoutingStrategy struct {
		Name               *string  `json:"name"`
		PriceWeight        *float64 `json:"price_weight"`
		AvailabilityWeight *float64 `json:"availability_weight"`
		LoadWeight         *float64 `json:"load_weight"`
		TTFTWeight         *float64 `json:"ttft_weight"`
	}
	type rawConfiguration struct {
		Strategies    map[string]rawRoutingStrategy `json:"strategies"`
		GroupBindings map[string]string             `json:"group_bindings"`
	}

	var raw rawConfiguration
	trimmedJSON := strings.TrimSpace(jsonStr)
	if trimmedJSON == "" || trimmedJSON == "{}" || trimmedJSON == "null" {
		if groupRatios != nil {
			return defaultPricingGroupRoutingConfiguration(groupRatios), nil
		}
		return PricingGroupRoutingConfiguration{}, errors.New("调度策略配置不能为空")
	}
	if err := common.UnmarshalJsonStr(jsonStr, &raw); err != nil {
		return PricingGroupRoutingConfiguration{}, err
	}
	if raw.Strategies == nil || raw.GroupBindings == nil {
		return PricingGroupRoutingConfiguration{}, errors.New("调度策略配置必须包含 strategies 和 group_bindings")
	}
	if len(raw.Strategies) == 0 {
		return PricingGroupRoutingConfiguration{}, errors.New("至少需要一个调度策略")
	}

	strategies := make(map[string]PricingGroupRoutingStrategy, len(raw.Strategies))
	strategyNames := make(map[string]string, len(raw.Strategies))
	for strategyID, definition := range raw.Strategies {
		if strategyID == "" || strategyID != strings.TrimSpace(strategyID) {
			return PricingGroupRoutingConfiguration{}, errors.New("调度策略包含无效的策略 ID")
		}
		if definition.Name == nil || strings.TrimSpace(*definition.Name) == "" {
			return PricingGroupRoutingConfiguration{}, errors.New("策略 " + strategyID + " 的名称不能为空")
		}
		name := strings.TrimSpace(*definition.Name)
		if existingID, exists := strategyNames[name]; exists {
			return PricingGroupRoutingConfiguration{}, errors.New("策略名称不能重复: " + existingID + " 和 " + strategyID)
		}
		strategyNames[name] = strategyID
		if definition.PriceWeight == nil || definition.AvailabilityWeight == nil || definition.LoadWeight == nil {
			return PricingGroupRoutingConfiguration{}, errors.New("策略 " + strategyID + " 必须同时设置价格、可用性、负载和首Token延迟权重")
		}
		ttftWeight := float64(0)
		if definition.TTFTWeight != nil {
			ttftWeight = *definition.TTFTWeight
		}
		strategy := PricingGroupRoutingStrategy{
			Strategy:           strategyID,
			Name:               name,
			PriceWeight:        *definition.PriceWeight,
			AvailabilityWeight: *definition.AvailabilityWeight,
			LoadWeight:         *definition.LoadWeight,
			TTFTWeight:         ttftWeight,
		}
		if strategy.PriceWeight < 0 || strategy.AvailabilityWeight < 0 || strategy.LoadWeight < 0 || strategy.TTFTWeight < 0 ||
			math.IsNaN(strategy.PriceWeight) || math.IsNaN(strategy.AvailabilityWeight) || math.IsNaN(strategy.LoadWeight) || math.IsNaN(strategy.TTFTWeight) ||
			math.IsInf(strategy.PriceWeight, 0) || math.IsInf(strategy.AvailabilityWeight, 0) || math.IsInf(strategy.LoadWeight, 0) || math.IsInf(strategy.TTFTWeight, 0) ||
			math.Abs(strategy.PriceWeight+strategy.AvailabilityWeight+strategy.LoadWeight+strategy.TTFTWeight-100) > 0.0001 {
			return PricingGroupRoutingConfiguration{}, errors.New("策略 " + strategyID + " 的调度权重总和必须为 100")
		}
		strategies[strategyID] = strategy
	}

	bindings := make(map[string]string, len(raw.GroupBindings))
	for group, strategyID := range raw.GroupBindings {
		if group == "" || group != strings.TrimSpace(group) {
			return PricingGroupRoutingConfiguration{}, errors.New("调度策略绑定包含无效的定价分组名")
		}
		strategyID = strings.TrimSpace(strategyID)
		if _, exists := strategies[strategyID]; !exists {
			return PricingGroupRoutingConfiguration{}, errors.New("定价分组 " + group + " 引用了不存在的策略: " + strategyID)
		}
		if groupRatios != nil {
			if _, exists := groupRatios[group]; !exists {
				return PricingGroupRoutingConfiguration{}, errors.New("调度策略绑定包含不存在的定价分组: " + group)
			}
		}
		bindings[group] = strategyID
	}
	if groupRatios != nil && len(bindings) != len(groupRatios) {
		return PricingGroupRoutingConfiguration{}, errors.New("调度策略绑定必须覆盖全部定价分组")
	}
	return PricingGroupRoutingConfiguration{Strategies: strategies, GroupBindings: bindings}, nil
}

func defaultPricingGroupRoutingStrategies() map[string]PricingGroupRoutingStrategy {
	return DefaultPricingGroupRoutingStrategies()
}

func defaultPricingGroupRoutingConfiguration(ratios map[string]float64) PricingGroupRoutingConfiguration {
	bindings := make(map[string]string, len(ratios))
	for group := range ratios {
		bindings[group] = PricingGroupRoutingStrategyBalanced
	}
	return PricingGroupRoutingConfiguration{
		Strategies:    defaultPricingGroupRoutingStrategies(),
		GroupBindings: bindings,
	}
}

func GetPricingGroupOrder() []string {
	return append([]string(nil), currentPricingGroupSnapshot().groupOrder...)
}

func PricingGroupOrder2JSONString() string {
	data, err := common.Marshal(GetPricingGroupOrder())
	if err != nil {
		return "[]"
	}
	return string(data)
}

func UpdatePricingGroupOrderByJSONString(jsonStr string) error {
	groups, err := parsePricingGroupOrder(jsonStr)
	if err != nil {
		return err
	}
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	snapshot := currentPricingGroupSnapshot()
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		snapshot.groupRatios,
		snapshot.groupEnabled,
		groups,
		snapshot.retryPolicies,
		snapshot.routingStrategies,
		snapshot.routingStrategyBindings,
	))
	return nil
}

func parsePricingGroupOrder(jsonStr string) ([]string, error) {
	var groups []string
	if err := common.UnmarshalJsonStr(jsonStr, &groups); err != nil {
		return nil, err
	}
	if groups == nil {
		return nil, errors.New("pricing group order must be a JSON array")
	}
	seen := make(map[string]struct{}, len(groups))
	for index, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			return nil, errors.New("pricing group order contains an empty group")
		}
		if _, exists := seen[group]; exists {
			return nil, errors.New("pricing group order contains duplicate group: " + group)
		}
		seen[group] = struct{}{}
		groups[index] = group
	}
	return groups, nil
}

func GetGroupRatio(name string) float64 {
	ratio, ok := currentPricingGroupSnapshot().groupRatios[name]
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func parseGroupRatios(jsonStr string) (map[string]float64, error) {
	ratios := make(map[string]float64)
	if err := common.UnmarshalJsonStr(jsonStr, &ratios); err != nil {
		return nil, err
	}
	if ratios == nil {
		return nil, errors.New("定价分组倍率必须是 JSON 对象")
	}
	for name, ratio := range ratios {
		if name == "" || name != strings.TrimSpace(name) {
			return nil, errors.New("定价分组包含无效的分组名")
		}
		if ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return nil, errors.New("定价分组倍率必须是大于等于 0 的有限数值: " + name)
		}
	}
	return ratios, nil
}
