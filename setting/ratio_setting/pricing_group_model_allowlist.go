package ratio_setting

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

func HasPricingGroupModelAllowlist(group string) bool {
	_, restricted := currentPricingGroupSnapshot().groupModelAllowlist[group]
	return restricted
}

// PricingGroupAllowsModel checks the public model ID before channel mappings.
func PricingGroupAllowsModel(group, modelName string) bool {
	models, restricted := currentPricingGroupSnapshot().groupModelAllowlist[group]
	if !restricted {
		return true
	}
	_, allowed := slices.BinarySearch(models, modelName)
	return allowed
}

func PricingGroupModelAllowlist2JSONString() string {
	data, err := common.Marshal(currentPricingGroupSnapshot().groupModelAllowlist)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func UpdatePricingGroupModelAllowlistByJSONString(jsonStr string) error {
	pricingGroupSnapshotUpdateMutex.Lock()
	defer pricingGroupSnapshotUpdateMutex.Unlock()
	snapshot := currentPricingGroupSnapshot()
	allowlist, err := ParsePricingGroupModelAllowlist(jsonStr, snapshot.groupRatios)
	if err != nil {
		return err
	}
	next := *snapshot
	next.groupModelAllowlist = allowlist
	pricingGroupSnapshotValue.Store(&next)
	return nil
}

// ParsePricingGroupModelAllowlist validates the stored model IDs. Pass ratios
// when validating a complete configuration and reject references to other groups.
func ParsePricingGroupModelAllowlist(jsonStr string, ratios map[string]float64) (map[string][]string, error) {
	var allowlist map[string][]string
	if err := common.UnmarshalJsonStr(jsonStr, &allowlist); err != nil {
		return nil, fmt.Errorf("定价分组模型白名单必须是 JSON 对象: %w", err)
	}
	if allowlist == nil {
		return nil, errors.New("定价分组模型白名单必须是 JSON 对象")
	}
	for group, models := range allowlist {
		if group == "" || group != strings.TrimSpace(group) || group == "auto" || group == "*" {
			return nil, errors.New("定价分组模型白名单包含无效的分组名")
		}
		if _, exists := ratios[group]; ratios != nil && !exists {
			return nil, fmt.Errorf("定价分组模型白名单包含不存在的分组: %s", group)
		}
		if len(models) == 0 {
			return nil, fmt.Errorf("分组 %s 启用模型白名单时必须至少选择一个模型", group)
		}
		sort.Strings(models)
		for index, name := range models {
			if name == "" || name != strings.TrimSpace(name) {
				return nil, fmt.Errorf("分组 %s 的模型白名单包含无效的模型名", group)
			}
			if index > 0 && models[index-1] == name {
				return nil, fmt.Errorf("分组 %s 的模型白名单包含重复的模型: %s", group, name)
			}
		}
	}
	return allowlist, nil
}
