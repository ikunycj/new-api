package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	MaxTokenGroupCandidates = 8
	MaxTokenGroupNameLength = 64
)

func GetUserUsableGroups(_ string) map[string]string {
	// This catalog contains pricing/routing groups that a user may assign to
	// tokens. The account group stored in users.group is a separate namespace
	// and must never be implicitly promoted into this catalog.
	return setting.GetUserUsableGroupsCopy()
}

func GroupInUserUsableGroups(userGroup, groupName string) bool {
	_, ok := GetUserUsableGroups(userGroup)[groupName]
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
		if !GroupInUserUsableGroups(userGroup, group) {
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

func validateConcreteTokenGroup(userGroup, group string) error {
	if !GroupInUserUsableGroups(userGroup, group) {
		return fmt.Errorf("无权访问 %s 分组", group)
	}
	if !ratio_setting.ContainsGroupRatio(group) {
		return fmt.Errorf("分组 %s 已被弃用", group)
	}
	return nil
}
