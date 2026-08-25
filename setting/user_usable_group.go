package setting

import (
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var userUsableGroups = map[string]string{
	"default": "默认分组",
	"vip":     "vip分组",
}
var userUsableGroupsMutex sync.RWMutex

// UserGroupPricingGroups stores the pricing groups available to each account
// group. A missing entry, or an entry containing "*", means all pricing
// groups. The account-group catalog and the pricing-group catalog are kept
// separate so adding an account group does not create a billing group.
const AllPricingGroups = "*"

var userGroupPricingGroups = map[string][]string{
	"default": {AllPricingGroups},
}
var userGroupPricingGroupsMutex sync.RWMutex

func GetUserUsableGroupsCopy() map[string]string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	copyUserUsableGroups := make(map[string]string)
	for k, v := range userUsableGroups {
		copyUserUsableGroups[k] = v
	}
	return copyUserUsableGroups
}

func UserUsableGroups2JSONString() string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	jsonBytes, err := common.Marshal(userUsableGroups)
	if err != nil {
		common.SysLog("error marshalling user groups: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateUserUsableGroupsByJSONString(jsonStr string) error {
	updated := make(map[string]string)
	if err := common.UnmarshalJsonStr(jsonStr, &updated); err != nil {
		return err
	}
	userUsableGroupsMutex.Lock()
	defer userUsableGroupsMutex.Unlock()
	userUsableGroups = updated
	return nil
}

func GetUsableGroupDescription(groupName string) string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	if desc, ok := userUsableGroups[groupName]; ok {
		return desc
	}
	return groupName
}

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

func SetUserGroupPricingGroups(userGroup string, pricingGroups []string) {
	userGroupPricingGroupsMutex.Lock()
	defer userGroupPricingGroupsMutex.Unlock()
	if userGroupPricingGroups == nil {
		userGroupPricingGroups = make(map[string][]string)
	}
	userGroupPricingGroups[userGroup] = normalizeUserGroupPricingGroups(pricingGroups)
}

func DeleteUserGroupPricingGroups(userGroup string) {
	userGroupPricingGroupsMutex.Lock()
	defer userGroupPricingGroupsMutex.Unlock()
	delete(userGroupPricingGroups, userGroup)
}
