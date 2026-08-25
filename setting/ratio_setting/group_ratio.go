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
)

type PricingGroupRetryPolicy struct {
	Mode       string `json:"mode"`
	RetryTimes int    `json:"retry_times"`
}

type PricingGroupConfiguration struct {
	GroupRatios   map[string]float64
	GroupOrder    []string
	RetryPolicies map[string]PricingGroupRetryPolicy
}

type pricingGroupSnapshot struct {
	groupRatios   map[string]float64
	groupOrder    []string
	retryPolicies map[string]PricingGroupRetryPolicy
}

var pricingGroupSnapshotValue atomic.Pointer[pricingGroupSnapshot]
var pricingGroupSnapshotUpdateMutex sync.Mutex

type groupRatioConfigValue struct{}
type pricingGroupRetryPolicyConfigValue struct{}

type GroupRatioSetting struct {
	GroupRatio              *groupRatioConfigValue              `json:"group_ratio"`
	PricingGroupRetryPolicy *pricingGroupRetryPolicyConfigValue `json:"pricing_group_retry_policy"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		defaultGroupRatio,
		[]string{"default", "vip", "svip"},
		nil,
	))
	groupRatioSetting = GroupRatioSetting{
		GroupRatio:              &groupRatioConfigValue{},
		PricingGroupRetryPolicy: &pricingGroupRetryPolicyConfigValue{},
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func newPricingGroupSnapshot(
	ratios map[string]float64,
	order []string,
	retryPolicies map[string]PricingGroupRetryPolicy,
) *pricingGroupSnapshot {
	ratioCopy := make(map[string]float64, len(ratios))
	for group, ratio := range ratios {
		ratioCopy[group] = ratio
	}
	policyCopy := make(map[string]PricingGroupRetryPolicy, len(retryPolicies))
	for group, policy := range retryPolicies {
		policyCopy[group] = policy
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
		groupRatios:   ratioCopy,
		groupOrder:    ordered,
		retryPolicies: policyCopy,
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
	pricingGroupSnapshotValue.Store(newPricingGroupSnapshot(
		ratios,
		snapshot.groupOrder,
		snapshot.retryPolicies,
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
		snapshot.groupOrder,
		policies,
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
		configuration.GroupOrder,
		configuration.RetryPolicies,
	))
}

func ParsePricingGroupConfiguration(groupRatioJSON, groupOrderJSON, retryPolicyJSON string) (*PricingGroupConfiguration, error) {
	ratioMap, err := parseGroupRatios(groupRatioJSON)
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

	return &PricingGroupConfiguration{
		GroupRatios:   ratioMap,
		GroupOrder:    order,
		RetryPolicies: retryPolicies,
	}, nil
}

// ParsePersistedPricingGroupConfiguration normalizes independently stored
// pricing-group options into one complete runtime configuration.
func ParsePersistedPricingGroupConfiguration(groupRatioJSON, groupOrderJSON, retryPolicyJSON string) (*PricingGroupConfiguration, error) {
	ratioMap, err := parseGroupRatios(groupRatioJSON)
	if err != nil {
		return nil, err
	}
	order, err := parsePricingGroupOrder(groupOrderJSON)
	if err != nil {
		return nil, err
	}
	retryPolicies, err := parsePricingGroupRetryPolicies(retryPolicyJSON)
	if err != nil {
		return nil, err
	}

	snapshot := newPricingGroupSnapshot(ratioMap, order, retryPolicies)
	normalizedPolicies := make(map[string]PricingGroupRetryPolicy, len(snapshot.groupRatios))
	for group := range snapshot.groupRatios {
		policy, exists := snapshot.retryPolicies[group]
		if !exists {
			policy = PricingGroupRetryPolicy{Mode: PricingGroupRetryModeActiveChannels}
		}
		normalizedPolicies[group] = policy
	}

	return &PricingGroupConfiguration{
		GroupRatios:   snapshot.groupRatios,
		GroupOrder:    snapshot.groupOrder,
		RetryPolicies: normalizedPolicies,
	}, nil
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
		groups,
		snapshot.retryPolicies,
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
