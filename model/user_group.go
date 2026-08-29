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

// DefaultUserGroup is the stable account-group name used by legacy users.
const DefaultUserGroup = "default"

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
	ActiveToday      int64    `json:"active_today" gorm:"-"`
	ActiveMonth      int64    `json:"active_month" gorm:"-"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
	TopupRatio       float64  `json:"topup_ratio"`
	PricingGroups    []string `json:"pricing_groups"`
	PricingGroupsAll bool     `json:"pricing_groups_all"`
}

type userGroupActivityRow struct {
	UserId       int   `gorm:"column:user_id"`
	LastActiveAt int64 `gorm:"column:last_active_at"`
}

type userGroupMembershipRow struct {
	Id        int    `gorm:"column:id"`
	UserGroup string `gorm:"column:user_group"`
}

// UserGroupUpdate contains the mutable identity and configuration of an
// account group. The built-in default group keeps its stable name.
type UserGroupUpdate struct {
	Name             *string
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

func listUserGroupSummaries() ([]UserGroupSummary, error) {
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

func listUserGroupSummariesAt(now time.Time) ([]UserGroupSummary, error) {
	groups, err := listUserGroupSummaries()
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return groups, nil
	}
	if LOG_DB == nil {
		return nil, errors.New("log database is not initialized")
	}

	todayStart, tomorrowStart, _, rollingMonthStart, _ := adminConsoleTimeBounds(now)
	var activityRows []userGroupActivityRow
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, MAX(created_at) AS last_active_at").
		Where("type = ? AND user_id <> 0 AND created_at >= ? AND created_at < ?", LogTypeConsume, rollingMonthStart, tomorrowStart).
		Group("user_id").
		Scan(&activityRows).Error; err != nil {
		return nil, err
	}

	groupByName := make(map[string]*UserGroupSummary, len(groups))
	lastActiveByUser := make(map[int]int64, len(activityRows))
	activeUserIds := make([]int, 0, len(activityRows))
	for index := range groups {
		groupByName[groups[index].Name] = &groups[index]
	}
	for _, activity := range activityRows {
		lastActiveByUser[activity.UserId] = activity.LastActiveAt
		activeUserIds = append(activeUserIds, activity.UserId)
	}

	const membershipBatchSize = 1000
	for start := 0; start < len(activeUserIds); start += membershipBatchSize {
		end := min(start+membershipBatchSize, len(activeUserIds))
		var memberships []userGroupMembershipRow
		if err := DB.Model(&User{}).
			Select("id, "+commonGroupCol+" AS user_group").
			Where("id IN ?", activeUserIds[start:end]).
			Scan(&memberships).Error; err != nil {
			return nil, err
		}
		for _, membership := range memberships {
			summary := groupByName[membership.UserGroup]
			if summary == nil {
				continue
			}
			summary.ActiveMonth++
			if lastActiveByUser[membership.Id] >= todayStart {
				summary.ActiveToday++
			}
		}
	}
	return groups, nil
}

func ListUserGroupSummaries() ([]UserGroupSummary, error) {
	return listUserGroupSummariesAt(time.Now())
}

func GetUserGroupSummary(name string) (*UserGroupSummary, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("用户分组名称不能为空")
	}
	groups, err := listUserGroupSummaries()
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

func validateUserGroupTopupRatios(ratios map[string]float64) error {
	for name, ratio := range ratios {
		if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 {
			return fmt.Errorf("充值倍率必须是大于等于 0 的有限数值: %s", name)
		}
	}
	return nil
}

func buildUserGroupConfigurationOptionsFromMaps(currentName, nextName string, update UserGroupUpdate, remove bool, ratioMap map[string]float64, pricingMap map[string][]string) (map[string]string, error) {
	if remove {
		delete(ratioMap, currentName)
		delete(pricingMap, currentName)
	} else {
		if nextName != currentName {
			ratio, ok := ratioMap[currentName]
			if !ok {
				ratio = 1
			}
			pricingGroups, ok := pricingMap[currentName]
			if !ok {
				pricingGroups = []string{setting.AllPricingGroups}
			}
			delete(ratioMap, currentName)
			delete(pricingMap, currentName)
			ratioMap[nextName] = ratio
			pricingMap[nextName] = pricingGroups
		}
		if update.TopupRatio != nil {
			ratioMap[nextName] = *update.TopupRatio
		}
		if update.PricingGroups != nil {
			pricingMap[nextName] = *update.PricingGroups
		}
	}
	if err := validateUserGroupTopupRatios(ratioMap); err != nil {
		return nil, err
	}
	ratioJSON, err := common.Marshal(ratioMap)
	if err != nil {
		return nil, err
	}
	pricingJSON, err := common.Marshal(pricingMap)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		topupGroupRatioOption:        string(ratioJSON),
		userGroupPricingGroupsOption: string(pricingJSON),
	}, nil
}

func buildUserGroupConfigurationOptions(currentName, nextName string, update UserGroupUpdate, remove bool) (map[string]string, error) {
	ratioMap := make(map[string]float64)
	if err := common.UnmarshalJsonStr(common.TopupGroupRatio2JSONString(), &ratioMap); err != nil {
		return nil, err
	}
	pricingMap := setting.GetUserGroupPricingGroupsCopy()
	return buildUserGroupConfigurationOptionsFromMaps(currentName, nextName, update, remove, ratioMap, pricingMap)
}

const (
	topupGroupRatioOption        = "TopupGroupRatio"
	userGroupPricingGroupsOption = "UserGroupPricingGroups"
)

func loadLockedUserGroupConfiguration(tx *gorm.DB) (map[string]float64, map[string][]string, error) {
	ratioFallback := common.TopupGroupRatio2JSONString()
	pricingFallback := setting.UserGroupPricingGroups2JSONString()
	optionValues := map[string]string{
		topupGroupRatioOption:        ratioFallback,
		userGroupPricingGroupsOption: pricingFallback,
	}

	// Always lock these shared option rows in the same order. User-group rows
	// are locked before this helper is called, so concurrent group updates
	// serialize before replacing the full JSON maps.
	for _, key := range []string{topupGroupRatioOption, userGroupPricingGroupsOption} {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Option{
			Key:   key,
			Value: optionValues[key],
		}).Error; err != nil {
			return nil, nil, err
		}
		var option Option
		if err := lockForUpdate(tx).Where(commonKeyCol+" = ?", key).First(&option).Error; err != nil {
			return nil, nil, err
		}
		optionValues[key] = option.Value
	}

	ratioMap := make(map[string]float64)
	if err := common.UnmarshalJsonStr(optionValues[topupGroupRatioOption], &ratioMap); err != nil {
		return nil, nil, err
	}
	pricingMap := make(map[string][]string)
	if err := common.UnmarshalJsonStr(optionValues[userGroupPricingGroupsOption], &pricingMap); err != nil {
		return nil, nil, err
	}
	return ratioMap, pricingMap, nil
}

func persistUserGroupConfiguration(name string, update UserGroupUpdate, remove bool) error {
	values, err := buildUserGroupConfigurationOptions(name, name, update, remove)
	if err != nil {
		return err
	}
	return UpdateOptionsBulk(values)
}

func UpdateUserGroupConfiguration(name string, update UserGroupUpdate) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("用户分组名称不能为空")
	}
	if update.Name == nil && update.TopupRatio == nil && update.PricingGroups == nil && update.PricingGroupsAll == nil {
		return "", errors.New("没有可更新的用户分组配置")
	}
	nextName := name
	if update.Name != nil {
		normalized, err := normalizeUserGroupName(*update.Name)
		if err != nil {
			return "", err
		}
		if name == DefaultUserGroup && normalized != DefaultUserGroup {
			return "", errors.New("default 分组不能重命名")
		}
		nextName = normalized
	}
	if update.PricingGroupsAll != nil && !*update.PricingGroupsAll && update.PricingGroups == nil {
		return "", errors.New("关闭全部定价分组时必须选择至少一个定价分组")
	}
	if update.PricingGroupsAll != nil && !*update.PricingGroupsAll && update.PricingGroups != nil && len(*update.PricingGroups) == 0 {
		return "", errors.New("请选择至少一个定价分组")
	}
	if update.TopupRatio != nil {
		if math.IsNaN(*update.TopupRatio) || math.IsInf(*update.TopupRatio, 0) || *update.TopupRatio < 0 {
			return "", errors.New("充值倍率必须是大于等于 0 的有限数值")
		}
	}
	if update.PricingGroupsAll != nil && *update.PricingGroupsAll {
		all := []string{setting.AllPricingGroups}
		update.PricingGroups = &all
	} else if update.PricingGroups != nil {
		normalized, err := normalizeUserGroupPricingSelection(*update.PricingGroups)
		if err != nil {
			return "", err
		}
		update.PricingGroups = &normalized
	}
	if DB == nil {
		return "", errors.New("database is not initialized")
	}
	userIDs := make([]int, 0)
	var optionValues map[string]string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var group UserGroup
		if err := lockForUpdate(tx).Where("name = ?", name).First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("用户分组不存在")
			}
			return err
		}
		if nextName != name {
			var duplicateCount int64
			if err := tx.Model(&UserGroup{}).Where("name = ?", nextName).Count(&duplicateCount).Error; err != nil {
				return err
			}
			if duplicateCount > 0 {
				return errors.New("用户分组已存在")
			}
			if err := tx.Unscoped().Model(&User{}).
				Where(commonGroupCol+" = ?", name).
				Pluck("id", &userIDs).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Model(&User{}).
				Where(commonGroupCol+" = ?", name).
				UpdateColumn("group", nextName).Error; err != nil {
				return err
			}
			group.Name = nextName
		}
		ratioMap, pricingMap, err := loadLockedUserGroupConfiguration(tx)
		if err != nil {
			return err
		}
		optionValues, err = buildUserGroupConfigurationOptionsFromMaps(name, nextName, update, false, ratioMap, pricingMap)
		if err != nil {
			return err
		}
		group.UpdatedAt = time.Now().Unix()
		if err := tx.Save(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.New("用户分组已存在")
			}
			return err
		}
		for _, key := range []string{topupGroupRatioOption, userGroupPricingGroupsOption} {
			value := optionValues[key]
			option := Option{Key: key}
			if err := tx.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
				return err
			}
			option.Value = value
			if err := tx.Save(&option).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	for _, key := range []string{topupGroupRatioOption, userGroupPricingGroupsOption} {
		value := optionValues[key]
		if err := updateOptionMap(key, value); err != nil {
			common.SysLog(fmt.Sprintf("failed to refresh user-group option %s after commit: %v", key, err))
			loadOptionsFromDatabase()
			break
		}
	}
	for _, userID := range userIDs {
		if err := invalidateUserCache(userID); err != nil {
			common.SysLog(fmt.Sprintf("failed to invalidate renamed user group cache for user %d: %v", userID, err))
		}
	}
	return nextName, nil
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
