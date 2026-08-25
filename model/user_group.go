package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxUserGroupNameLength = 64

// UserGroup is the administrator-managed account-group catalog. It is
// intentionally separate from pricing and routing groups.
type UserGroup struct {
	Id        int    `json:"id"`
	Name      string `json:"name" gorm:"type:varchar(64);uniqueIndex;not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (UserGroup) TableName() string {
	return "user_groups"
}

// UserGroupSummary is the read model used by the administrator group page.
type UserGroupSummary struct {
	Id               int      `json:"id"`
	Name             string   `json:"name"`
	UserCount        int64    `json:"user_count"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
	TopupRatio       float64  `json:"topup_ratio"`
	PricingGroups    []string `json:"pricing_groups"`
	PricingGroupsAll bool     `json:"pricing_groups_all"`
}

// UserGroupUpdate contains the mutable configuration of an account group.
// The group name is intentionally excluded: it is a stable identity used by
// users, billing, and authorization checks.
type UserGroupUpdate struct {
	TopupRatio       *float64
	PricingGroups    *[]string
	PricingGroupsAll *bool
}

func normalizeUserGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("用户分组名称不能为空")
	}
	if utf8.RuneCountInString(name) > maxUserGroupNameLength {
		return "", fmt.Errorf("用户分组名称不能超过 %d 个字符", maxUserGroupNameLength)
	}
	if strings.ContainsAny(name, "/\\") {
		return "", errors.New("用户分组名称不能包含路径分隔符")
	}
	if name == "auto" {
		return "", errors.New("auto 是系统保留分组名称")
	}
	return name, nil
}

func (group *UserGroup) BeforeCreate(_ *gorm.DB) error {
	name, err := normalizeUserGroupName(group.Name)
	if err != nil {
		return err
	}
	group.Name = name
	return nil
}

func (group *UserGroup) BeforeUpdate(_ *gorm.DB) error {
	name, err := normalizeUserGroupName(group.Name)
	if err != nil {
		return err
	}
	group.Name = name
	return nil
}

// EnsureDefaultUserGroup makes the built-in default group available after
// migration and is safe to call repeatedly. Its configuration is mutable,
// but its identity cannot be deleted.
func EnsureDefaultUserGroup() error {
	if DB == nil {
		return errors.New("database is not initialized")
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&UserGroup{Name: DefaultUserGroup}).Error
}

func ListUserGroupNames() ([]string, error) {
	if DB == nil {
		return nil, errors.New("database is not initialized")
	}
	var names []string
	err := DB.Model(&UserGroup{}).
		Order(clause.OrderBy{Expression: clause.Expr{
			SQL:  "CASE WHEN name = ? THEN 0 ELSE 1 END",
			Vars: []any{DefaultUserGroup},
		}}).
		Order("name ASC").
		Pluck("name", &names).Error
	return names, err
}

func IsUserGroupName(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	if DB == nil {
		return false, errors.New("database is not initialized")
	}
	var count int64
	err := DB.Model(&UserGroup{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

func ListUserGroupSummaries() ([]UserGroupSummary, error) {
	if DB == nil {
		return nil, errors.New("database is not initialized")
	}
	var groups []UserGroupSummary
	err := DB.Model(&UserGroup{}).
		Select("user_groups.id, user_groups.name, COUNT(users.id) AS user_count, user_groups.created_at, user_groups.updated_at").
		Joins("LEFT JOIN users ON users." + commonGroupCol + " = user_groups.name AND users.deleted_at IS NULL").
		Group("user_groups.id, user_groups.name, user_groups.created_at, user_groups.updated_at").
		Order(clause.OrderBy{Expression: clause.Expr{
			SQL:  "CASE WHEN user_groups.name = ? THEN 0 ELSE 1 END",
			Vars: []any{DefaultUserGroup},
		}}).
		Order("user_groups.name ASC").
		Scan(&groups).Error
	if err != nil {
		return nil, err
	}
	for index := range groups {
		applyUserGroupConfiguration(&groups[index])
	}
	return groups, nil
}

func GetUserGroupSummary(name string) (*UserGroupSummary, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("用户分组名称不能为空")
	}
	groups, err := ListUserGroupSummaries()
	if err != nil {
		return nil, err
	}
	for index := range groups {
		if groups[index].Name == name {
			return &groups[index], nil
		}
	}
	return nil, errors.New("用户分组不存在")
}

func applyUserGroupConfiguration(summary *UserGroupSummary) {
	summary.TopupRatio = common.GetTopupGroupRatio(summary.Name)
	pricingGroups := setting.GetUserGroupPricingGroups(summary.Name)
	summary.PricingGroups = pricingGroups
	summary.PricingGroupsAll = setting.UserGroupPricingGroupsAreAll(summary.Name)
}

func normalizeUserGroupPricingSelection(groups []string) ([]string, error) {
	if len(groups) == 0 {
		return []string{setting.AllPricingGroups}, nil
	}
	catalog, err := GetPricingGroupNames()
	if err != nil {
		return nil, err
	}
	catalogSet := make(map[string]struct{}, len(catalog))
	for _, group := range catalog {
		catalogSet[group] = struct{}{}
	}
	seen := make(map[string]struct{}, len(groups))
	result := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			return nil, errors.New("定价分组名称不能为空")
		}
		if group == setting.AllPricingGroups {
			if len(groups) != 1 {
				return nil, errors.New("全部不能与其他定价分组同时选择")
			}
			return []string{setting.AllPricingGroups}, nil
		}
		if _, ok := catalogSet[group]; !ok {
			return nil, fmt.Errorf("定价分组不存在: %s", group)
		}
		if _, ok := seen[group]; ok {
			return nil, fmt.Errorf("定价分组不能重复: %s", group)
		}
		seen[group] = struct{}{}
		result = append(result, group)
	}
	return result, nil
}

func persistUserGroupConfiguration(name string, update UserGroupUpdate, remove bool) error {
	ratioMap := make(map[string]float64)
	if err := common.UnmarshalJsonStr(common.TopupGroupRatio2JSONString(), &ratioMap); err != nil {
		return err
	}
	pricingMap := setting.GetUserGroupPricingGroupsCopy()
	if remove {
		delete(ratioMap, name)
		delete(pricingMap, name)
	} else {
		if update.TopupRatio != nil {
			ratioMap[name] = *update.TopupRatio
		}
		if update.PricingGroups != nil {
			pricingMap[name] = *update.PricingGroups
		}
	}
	ratioJSON, err := common.Marshal(ratioMap)
	if err != nil {
		return err
	}
	pricingJSON, err := common.Marshal(pricingMap)
	if err != nil {
		return err
	}
	if err := UpdateOptionsBulk(map[string]string{
		"TopupGroupRatio":        string(ratioJSON),
		"UserGroupPricingGroups": string(pricingJSON),
	}); err != nil {
		return err
	}
	return nil
}

func UpdateUserGroupConfiguration(name string, update UserGroupUpdate) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("用户分组名称不能为空")
	}
	if update.TopupRatio == nil && update.PricingGroups == nil && update.PricingGroupsAll == nil {
		return errors.New("没有可更新的用户分组配置")
	}
	if update.PricingGroupsAll != nil && !*update.PricingGroupsAll && update.PricingGroups == nil {
		return errors.New("关闭全部定价分组时必须选择至少一个定价分组")
	}
	if update.PricingGroupsAll != nil && !*update.PricingGroupsAll && update.PricingGroups != nil && len(*update.PricingGroups) == 0 {
		return errors.New("请选择至少一个定价分组")
	}
	if update.TopupRatio != nil {
		if math.IsNaN(*update.TopupRatio) || math.IsInf(*update.TopupRatio, 0) || *update.TopupRatio < 0 {
			return errors.New("充值倍率必须是大于等于 0 的有限数值")
		}
	}
	if update.PricingGroupsAll != nil && *update.PricingGroupsAll {
		all := []string{setting.AllPricingGroups}
		update.PricingGroups = &all
	} else if update.PricingGroups != nil {
		normalized, err := normalizeUserGroupPricingSelection(*update.PricingGroups)
		if err != nil {
			return err
		}
		update.PricingGroups = &normalized
	}
	if DB == nil {
		return errors.New("database is not initialized")
	}
	var group UserGroup
	if err := DB.Where("name = ?", name).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户分组不存在")
		}
		return err
	}
	if err := persistUserGroupConfiguration(name, update, false); err != nil {
		return err
	}
	return DB.Model(&group).Update("updated_at", time.Now().Unix()).Error
}

func CreateUserGroup(name string) (*UserGroup, error) {
	name, err := normalizeUserGroupName(name)
	if err != nil {
		return nil, err
	}
	if DB == nil {
		return nil, errors.New("database is not initialized")
	}
	var existing UserGroup
	if err := DB.Where("name = ?", name).First(&existing).Error; err == nil {
		return nil, errors.New("用户分组已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	group := &UserGroup{Name: name}
	if err := DB.Create(group).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("用户分组已存在")
		}
		return nil, err
	}
	if err := persistUserGroupConfiguration(name, UserGroupUpdate{
		TopupRatio:    float64Ptr(1),
		PricingGroups: stringSlicePtr([]string{setting.AllPricingGroups}),
	}, false); err != nil {
		_ = DB.Delete(group).Error
		return nil, err
	}
	return group, nil
}

func DeleteUserGroup(name string) error {
	name, err := normalizeUserGroupName(name)
	if err != nil {
		return err
	}
	if name == DefaultUserGroup {
		return errors.New("default 分组不能删除")
	}
	if DB == nil {
		return errors.New("database is not initialized")
	}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		var group UserGroup
		if err := lockForUpdate(tx).Where("name = ?", name).First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("用户分组不存在")
			}
			return err
		}
		var userCount int64
		if err := tx.Model(&User{}).
			Where(commonGroupCol+" = ? AND deleted_at IS NULL", name).
			Count(&userCount).Error; err != nil {
			return err
		}
		if userCount > 0 {
			return fmt.Errorf("分组仍有 %d 名用户，不能删除", userCount)
		}
		return tx.Delete(&group).Error
	}); err != nil {
		return err
	}
	return persistUserGroupConfiguration(name, UserGroupUpdate{}, true)
}

func float64Ptr(value float64) *float64 {
	return &value
}

func stringSlicePtr(value []string) *[]string {
	return &value
}
