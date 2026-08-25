package ratio_setting

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

var groupRatioMap = types.NewRWMap[string, float64]()

const (
	PricingGroupRetryModeFixed          = "fixed"
	PricingGroupRetryModeActiveChannels = "active_channels"
	MaxPricingGroupRetryTimes           = 100
)

type PricingGroupRetryPolicy struct {
	Mode       string `json:"mode"`
	RetryTimes int    `json:"retry_times"`
}

var pricingGroupRetryPolicyMap = types.NewRWMap[string, PricingGroupRetryPolicy]()

var pricingGroupOrder = struct {
	sync.RWMutex
	groups []string
}{
	groups: []string{"default", "vip", "svip"},
}

type GroupRatioSetting struct {
	GroupRatio              *types.RWMap[string, float64]                 `json:"group_ratio"`
	PricingGroupRetryPolicy *types.RWMap[string, PricingGroupRetryPolicy] `json:"pricing_group_retry_policy"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	groupRatioMap.AddAll(defaultGroupRatio)
	groupRatioSetting = GroupRatioSetting{
		GroupRatio:              groupRatioMap,
		PricingGroupRetryPolicy: pricingGroupRetryPolicyMap,
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func GetGroupRatioCopy() map[string]float64 {
	return groupRatioMap.ReadAll()
}

func ContainsGroupRatio(name string) bool {
	_, ok := groupRatioMap.Get(name)
	return ok
}

func GroupRatio2JSONString() string {
	return groupRatioMap.MarshalJSONString()
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupRatioMap, jsonStr)
}

func GetPricingGroupRetryPolicy(group string) (PricingGroupRetryPolicy, bool) {
	group = strings.TrimSpace(group)
	if policy, exists := pricingGroupRetryPolicyMap.Get(group); exists {
		return policy, true
	}
	if !ContainsGroupRatio(group) {
		return PricingGroupRetryPolicy{}, false
	}
	return PricingGroupRetryPolicy{Mode: PricingGroupRetryModeActiveChannels}, true
}

func GetPricingGroupRetryPolicyCopy() map[string]PricingGroupRetryPolicy {
	return pricingGroupRetryPolicyMap.ReadAll()
}

func PricingGroupRetryPolicy2JSONString() string {
	return pricingGroupRetryPolicyMap.MarshalJSONString()
}

func UpdatePricingGroupRetryPolicyByJSONString(jsonStr string) error {
	policies, err := parsePricingGroupRetryPolicies(jsonStr)
	if err != nil {
		return err
	}
	normalized, err := common.Marshal(policies)
	if err != nil {
		return err
	}
	return types.LoadFromJsonString(pricingGroupRetryPolicyMap, string(normalized))
}

func CheckPricingGroupRetryPolicy(jsonStr string) error {
	policies, err := parsePricingGroupRetryPolicies(jsonStr)
	if err != nil {
		return err
	}
	ratios := GetGroupRatioCopy()
	for group := range policies {
		if _, exists := ratios[group]; !exists {
			return errors.New("重试策略包含不存在的定价分组: " + group)
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

func GetPricingGroupOrder() []string {
	pricingGroupOrder.RLock()
	configured := append([]string(nil), pricingGroupOrder.groups...)
	pricingGroupOrder.RUnlock()

	ratios := GetGroupRatioCopy()
	ordered := make([]string, 0, len(ratios))
	seen := make(map[string]struct{}, len(ratios))
	for _, group := range configured {
		if _, exists := ratios[group]; !exists {
			continue
		}
		if _, exists := seen[group]; exists {
			continue
		}
		seen[group] = struct{}{}
		ordered = append(ordered, group)
	}

	remaining := make([]string, 0, len(ratios)-len(ordered))
	for group := range ratios {
		if _, exists := seen[group]; !exists {
			remaining = append(remaining, group)
		}
	}
	sort.Strings(remaining)
	return append(ordered, remaining...)
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
	pricingGroupOrder.Lock()
	pricingGroupOrder.groups = groups
	pricingGroupOrder.Unlock()
	return nil
}

func CheckPricingGroupOrder(jsonStr string) error {
	groups, err := parsePricingGroupOrder(jsonStr)
	if err != nil {
		return err
	}
	ratios := GetGroupRatioCopy()
	for _, group := range groups {
		if _, exists := ratios[group]; !exists {
			return errors.New("pricing group order contains unknown group: " + group)
		}
	}
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
	ratio, ok := groupRatioMap.Get(name)
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func CheckGroupRatio(jsonStr string) error {
	checkGroupRatio := make(map[string]float64)
	err := common.UnmarshalJsonStr(jsonStr, &checkGroupRatio)
	if err != nil {
		return err
	}
	for name, ratio := range checkGroupRatio {
		if ratio < 0 {
			return errors.New("group ratio must be not less than 0: " + name)
		}
	}
	return nil
}
