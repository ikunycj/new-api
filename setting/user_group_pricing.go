package setting

import (
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// UserGroupPricingGroups stores the pricing groups available to each account
// group. A missing entry, or an entry containing "*", means all pricing
// groups. The account-group catalog and the pricing-group catalog are kept
// separate so adding an account group does not create a billing group.
const AllPricingGroups = "*"

var userGroupPricingGroups = map[string][]string{
	"default": {AllPricingGroups},
}
var userGroupPricingGroupsMutex sync.RWMutex

func normalizeUserGroupPricingGroups(groups []string) []string {
	if len(groups) == 0 {
		return []string{AllPricingGroups}
	}
	seen := make(map[string]struct{}, len(groups))
	result := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if group == AllPricingGroups {
			return []string{AllPricingGroups}
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		result = append(result, group)
	}
	if len(result) == 0 {
		return []string{AllPricingGroups}
	}
	return result
}

func GetUserGroupPricingGroups(userGroup string) []string {
	userGroupPricingGroupsMutex.RLock()
	groups, ok := userGroupPricingGroups[userGroup]
	if ok {
		groups = append([]string(nil), groups...)
	}
	userGroupPricingGroupsMutex.RUnlock()
	if !ok {
		return []string{AllPricingGroups}
	}
	return normalizeUserGroupPricingGroups(groups)
}

func UserGroupPricingGroupsAreAll(userGroup string) bool {
	groups := GetUserGroupPricingGroups(userGroup)
	return len(groups) == 1 && groups[0] == AllPricingGroups
}

func GetUserGroupPricingGroupsCopy() map[string][]string {
	userGroupPricingGroupsMutex.RLock()
	defer userGroupPricingGroupsMutex.RUnlock()

	result := make(map[string][]string, len(userGroupPricingGroups))
	for group, pricingGroups := range userGroupPricingGroups {
		result[group] = normalizeUserGroupPricingGroups(pricingGroups)
	}
	return result
}

func UserGroupPricingGroups2JSONString() string {
	userGroupPricingGroupsMutex.RLock()
	defer userGroupPricingGroupsMutex.RUnlock()

	jsonBytes, err := common.Marshal(userGroupPricingGroups)
	if err != nil {
		common.SysLog("error marshalling user group pricing groups: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func UpdateUserGroupPricingGroupsByJSONString(jsonStr string) error {
	updated := make(map[string][]string)
	if err := common.UnmarshalJsonStr(jsonStr, &updated); err != nil {
		return err
	}
	for group, pricingGroups := range updated {
		updated[group] = normalizeUserGroupPricingGroups(pricingGroups)
	}
	userGroupPricingGroupsMutex.Lock()
	defer userGroupPricingGroupsMutex.Unlock()
	userGroupPricingGroups = updated
	return nil
}
