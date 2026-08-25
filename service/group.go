package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	MaxTokenGroupCandidates     = 8
	MaxTokenGroupNameLength     = 64
	DefaultTokenGroupRetryTimes = 3
	MaxTokenGroupRetryTimes     = 100
)

func GetUserGroupPricingGroups(userGroup string) map[string]string {
	// This catalog contains pricing/routing groups that a user may assign to
	// tokens. The account group stored in users.group is a separate namespace
	// and must never be implicitly promoted into this catalog.
	groups := make(map[string]string)
	for group := range ratio_setting.GetGroupRatioCopy() {
		if group == "auto" {
			continue
		}
		groups[group] = group
	}
	if setting.UserGroupPricingGroupsAreAll(userGroup) {
		return groups
	}
	allowed := setting.GetUserGroupPricingGroups(userGroup)
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, group := range allowed {
		allowedSet[group] = struct{}{}
	}
	for group := range groups {
		if _, ok := allowedSet[group]; !ok {
			delete(groups, group)
		}
	}
	return groups
}

func UserGroupCanUsePricingGroup(userGroup, pricingGroup string) bool {
	_, ok := GetUserGroupPricingGroups(userGroup)[pricingGroup]
	return ok
}

func ValidateTokenGroup(userGroup, group string) error {
	if group == "" {
		return nil
	}
	if strings.TrimSpace(group) == "" {
		return fmt.Errorf("分组不能为空")
	}
	if utf8.RuneCountInString(group) > MaxTokenGroupNameLength {
		return fmt.Errorf("分组名称长度不能超过 %d 个字符", MaxTokenGroupNameLength)
	}
	if group == "auto" {
		if !UserGroupCanUsePricingGroup(userGroup, group) {
			return fmt.Errorf("无权访问 %s 分组", group)
		}
		return nil
	}
	return validateConcreteTokenGroup(userGroup, group)
}

// ValidateTokenGroupCandidates validates concrete groups independently of the
// internal "auto" marker used for ordered candidates. The marker itself is
// never accepted as a selectable pricing group.
func ValidateTokenGroupCandidates(userGroup string, groups []string) error {
	if len(groups) > MaxTokenGroupCandidates {
		return fmt.Errorf("候选分组不能超过 %d 个", MaxTokenGroupCandidates)
	}
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("候选分组不能为空")
		}
		if utf8.RuneCountInString(group) > MaxTokenGroupNameLength {
			return fmt.Errorf("候选分组名称长度不能超过 %d 个字符", MaxTokenGroupNameLength)
		}
		if group == "auto" {
			return fmt.Errorf("候选分组不能包含 auto")
		}
		if _, ok := seen[group]; ok {
			return fmt.Errorf("候选分组不能重复: %s", group)
		}
		seen[group] = struct{}{}
		if err := validateConcreteTokenGroup(userGroup, group); err != nil {
			return err
		}
	}
	return nil
}

// NormalizeTokenGroupRetryTimes keeps retry counts only for concrete groups
// selected by the token. Missing counts receive the product default.
func NormalizeTokenGroupRetryTimes(groups []string, values map[string]int) (map[string]int, error) {
	result := make(map[string]int, len(groups))
	for _, group := range groups {
		if strings.TrimSpace(group) == "" || group == "auto" {
			continue
		}
		retryTimes := DefaultTokenGroupRetryTimes
		if configured, ok := values[group]; ok {
			retryTimes = configured
		}
		if retryTimes < 0 || retryTimes > MaxTokenGroupRetryTimes {
			return nil, fmt.Errorf("分组 %s 的重试次数必须在 0 到 %d 之间", group, MaxTokenGroupRetryTimes)
		}
		result[group] = retryTimes
	}
	return result, nil
}

func validateConcreteTokenGroup(userGroup, group string) error {
	if !UserGroupCanUsePricingGroup(userGroup, group) {
		return fmt.Errorf("无权访问 %s 分组", group)
	}
	if !ratio_setting.ContainsGroupRatio(group) {
		return fmt.Errorf("分组 %s 已被弃用", group)
	}
	return nil
}
