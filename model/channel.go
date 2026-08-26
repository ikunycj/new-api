package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Channel struct {
	Id                 int     `json:"id"`
	Type               int     `json:"type" gorm:"default:0"`
	Key                string  `json:"key" gorm:"not null"`
	OpenAIOrganization *string `json:"openai_organization"`
	TestModel          *string `json:"test_model"`
	Status             int     `json:"status" gorm:"default:1"`
	Name               string  `json:"name" gorm:"index"`
	Weight             *uint   `json:"weight" gorm:"default:0"`
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`
	TestTime           int64   `json:"test_time" gorm:"bigint"`
	LastTestTime       int64   `json:"last_test_time" gorm:"-"`
	LastTestIsAuto     bool    `json:"last_test_is_auto" gorm:"-"`
	ResponseTime       int     `json:"response_time"` // in milliseconds
	BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`
	Other              string  `json:"other"`
	Balance            float64 `json:"balance"` // in USD
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
	Models             string  `json:"models"`
	Group              string  `json:"group" gorm:"type:varchar(64)"`
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string `json:"model_mapping" gorm:"type:text"`
	//MaxInputTokens     *int    `json:"max_input_tokens" gorm:"default:0"`
	StatusCodeMapping                *string `json:"status_code_mapping" gorm:"type:varchar(1024);default:''"`
	AutoBan                          *int    `json:"auto_ban" gorm:"default:0"`
	ProbeIntervalSeconds             int     `json:"probe_interval_seconds"`
	AutoDisabledProbeIntervalSeconds int     `json:"auto_disabled_probe_interval_seconds"`
	ProbeFailureAutoBan              *bool   `json:"probe_failure_auto_ban"`
	ProbeSuccessAutoEnable           *bool   `json:"probe_success_auto_enable"`
	UpstreamMaxRetries               *int    `json:"upstream_max_retries"`
	MaxConcurrency                   *int    `json:"max_concurrency"`
	CurrentConcurrency               int     `json:"current_concurrency" gorm:"-"`
	PriceMultiplier                  float64 `json:"price_multiplier"`
	PriceMultiplierMode              string  `json:"price_multiplier_mode" gorm:"type:varchar(16)"`
	DailyTokens                      int64   `json:"daily_tokens" gorm:"-"`
	MonthlyTokens                    int64   `json:"monthly_tokens" gorm:"-"`
	DailyCostUSD                     float64 `json:"daily_cost_usd" gorm:"-"`
	MonthlyCostUSD                   float64 `json:"monthly_cost_usd" gorm:"-"`
	ForcePriority                    *bool   `json:"force_priority"`
	ForcePriorityScope               string  `json:"force_priority_scope" gorm:"type:varchar(16)"`
	PreviousDayProbeSuccessRate      float64 `json:"previous_day_probe_success_rate" gorm:"-"`
	PreviousDayProbeSampleCount      int     `json:"-" gorm:"-"`
	OtherInfo                        string  `json:"other_info"`
	Tag                              *string `json:"tag" gorm:"index"`
	Setting                          *string `json:"setting" gorm:"type:text"` // 渠道额外设置
	ParamOverride                    *string `json:"param_override" gorm:"type:text"`
	HeaderOverride                   *string `json:"header_override" gorm:"type:text"`
	Remark                           *string `json:"remark" gorm:"type:varchar(255)" validate:"max=255"`
	// add after v0.8.5
	ChannelInfo ChannelInfo `json:"channel_info" gorm:"type:json"`

	OtherSettings string `json:"settings" gorm:"column:settings"` // 其他设置，存储azure版本等不需要检索的信息，详见dto.ChannelOtherSettings

	// cache info
	Keys []string `json:"-" gorm:"-"`
}

const (
	DefaultChannelProbeIntervalSeconds      = 120
	DefaultAutoDisabledProbeIntervalSeconds = 10
	DefaultChannelUpstreamMaxRetries        = 1
	DefaultChannelMaxConcurrency            = 1000
	MaxChannelMaxConcurrency                = 10000
	MaxChannelProbeIntervalSeconds          = 7 * 24 * 60 * 60
	MaxChannelUpstreamRetries               = 100
	MaxChannelPriceMultiplier               = 1000
	ChannelPriceMultiplierModeUSD           = "usd"
	ChannelPriceMultiplierModeCNY           = "cny"
	ChannelForcePriorityScopeGroup          = "group"
	ChannelForcePriorityScopeCrossGroup     = "cross_group"
)

type ChannelInfo struct {
	IsMultiKey             bool                  `json:"is_multi_key"`                        // 是否多Key模式
	MultiKeySize           int                   `json:"multi_key_size"`                      // 多Key模式下的Key数量
	MultiKeyStatusList     map[int]int           `json:"multi_key_status_list"`               // key状态列表，key index -> status
	MultiKeyDisabledReason map[int]string        `json:"multi_key_disabled_reason,omitempty"` // key禁用原因列表，key index -> reason
	MultiKeyDisabledTime   map[int]int64         `json:"multi_key_disabled_time,omitempty"`   // key禁用时间列表，key index -> time
	MultiKeyPollingIndex   int                   `json:"multi_key_polling_index"`             // 多Key模式下轮询的key索引
	MultiKeyMode           constant.MultiKeyMode `json:"multi_key_mode"`
}

type ChannelSortOptions struct {
	SortBy    string
	SortOrder string
	IDSort    bool
}

var channelSortColumns = map[string]string{
	"id":            "id",
	"name":          "name",
	"balance":       "balance",
	"response_time": "response_time",
	"test_time":     "test_time",
}

func NewChannelSortOptions(sortBy string, sortOrder string, idSort bool) ChannelSortOptions {
	normalizedSortBy := strings.ToLower(strings.TrimSpace(sortBy))
	normalizedSortOrder := strings.ToLower(strings.TrimSpace(sortOrder))
	if _, ok := channelSortColumns[normalizedSortBy]; !ok {
		normalizedSortBy = ""
		normalizedSortOrder = ""
	} else if normalizedSortOrder != "asc" {
		normalizedSortOrder = "desc"
	}

	return ChannelSortOptions{
		SortBy:    normalizedSortBy,
		SortOrder: normalizedSortOrder,
		IDSort:    idSort,
	}
}

func (options ChannelSortOptions) Apply(query *gorm.DB) *gorm.DB {
	if columnName, ok := channelSortColumns[options.SortBy]; ok {
		return query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: columnName},
			Desc:   options.SortOrder != "asc",
		})
	}
	if options.IDSort {
		return query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   true,
		})
	}
	return query.Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true})
}

func resolveChannelSortOptions(idSort bool, sortOptions []ChannelSortOptions) ChannelSortOptions {
	if len(sortOptions) == 0 {
		return NewChannelSortOptions("", "", idSort)
	}
	options := sortOptions[0]
	options.IDSort = options.IDSort || idSort
	return options
}

func NormalizeChannelGroupFilter(group string) string {
	group = strings.TrimSpace(group)
	if group == "" || strings.EqualFold(group, "all") || strings.EqualFold(group, "null") {
		return ""
	}
	return group
}

func channelGroupFilterCondition() string {
	if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
		return `CONCAT(',', ` + commonGroupCol + `, ',') LIKE ? ESCAPE '!'`
	}
	return `(',' || ` + commonGroupCol + ` || ',') LIKE ? ESCAPE '!'`
}

func channelGroupFilterPattern(group string) string {
	group = strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	).Replace(group)
	return "%," + group + ",%"
}

func ApplyChannelGroupFilter(query *gorm.DB, group string) *gorm.DB {
	group = NormalizeChannelGroupFilter(group)
	if group == "" {
		return query
	}
	return query.Where(channelGroupFilterCondition(), channelGroupFilterPattern(group))
}

// Value implements driver.Valuer interface
func (c ChannelInfo) Value() (driver.Value, error) {
	return common.Marshal(&c)
}

// Scan implements sql.Scanner interface
func (c *ChannelInfo) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return common.Unmarshal(bytesValue, c)
}

func (channel *Channel) GetKeys() []string {
	if channel.Key == "" {
		return []string{}
	}
	if len(channel.Keys) > 0 {
		return channel.Keys
	}
	trimmed := strings.TrimSpace(channel.Key)
	// If the key starts with '[', try to parse it as a JSON array (e.g., for Vertex AI scenarios)
	if strings.HasPrefix(trimmed, "[") {
		var arr []json.RawMessage
		if err := common.Unmarshal([]byte(trimmed), &arr); err == nil {
			res := make([]string, len(arr))
			for i, v := range arr {
				res[i] = string(v)
			}
			return res
		}
	}
	// Otherwise, fall back to splitting by newline
	keys := strings.Split(strings.Trim(channel.Key, "\n"), "\n")
	return keys
}

func (channel *Channel) GetNextEnabledKey() (string, int, *types.NewAPIError) {
	// If not in multi-key mode, return the original key string directly.
	if !channel.ChannelInfo.IsMultiKey {
		return channel.Key, 0, nil
	}

	// Obtain all keys (split by \n)
	keys := channel.GetKeys()
	if len(keys) == 0 {
		// No keys available, return error, should disable the channel
		return "", 0, types.NewError(errors.New("no keys available"), types.ErrorCodeChannelNoAvailableKey)
	}

	lock := GetChannelPollingLock(channel.Id)
	lock.Lock()
	defer lock.Unlock()

	statusList := channel.ChannelInfo.MultiKeyStatusList
	// helper to get key status, default to enabled when missing
	getStatus := func(idx int) int {
		if statusList == nil {
			return common.ChannelStatusEnabled
		}
		if status, ok := statusList[idx]; ok {
			return status
		}
		return common.ChannelStatusEnabled
	}

	// Collect indexes of enabled keys
	enabledIdx := make([]int, 0, len(keys))
	for i := range keys {
		if getStatus(i) == common.ChannelStatusEnabled {
			enabledIdx = append(enabledIdx, i)
		}
	}
	// If no specific status list or none enabled, return an explicit error so caller can
	// properly handle a channel with no available keys (e.g. mark channel disabled).
	// Returning the first key here caused requests to keep using an already-disabled key.
	if len(enabledIdx) == 0 {
		return "", 0, types.NewError(errors.New("no enabled keys"), types.ErrorCodeChannelNoAvailableKey)
	}

	switch channel.ChannelInfo.MultiKeyMode {
	case constant.MultiKeyModeRandom:
		// Randomly pick one enabled key
		selectedIdx := enabledIdx[rand.Intn(len(enabledIdx))]
		return keys[selectedIdx], selectedIdx, nil
	case constant.MultiKeyModePolling:
		// Use channel-specific lock to ensure thread-safe polling

		channelInfo, err := CacheGetChannelInfo(channel.Id)
		if err != nil {
			return "", 0, types.NewError(err, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
		}
		defer func() {
			if common.DebugEnabled {
				logger.LogDebug(nil, "channel %d polling index: %d", channel.Id, channel.ChannelInfo.MultiKeyPollingIndex)
			}
			if !common.MemoryCacheEnabled {
				_ = channel.SaveChannelInfo()
			} else {
				// CacheUpdateChannel(channel)
			}
		}()
		// Start from the saved polling index and look for the next enabled key
		start := channelInfo.MultiKeyPollingIndex
		if start < 0 || start >= len(keys) {
			start = 0
		}
		for i := 0; i < len(keys); i++ {
			idx := (start + i) % len(keys)
			if getStatus(idx) == common.ChannelStatusEnabled {
				// update polling index for next call (point to the next position)
				channel.ChannelInfo.MultiKeyPollingIndex = (idx + 1) % len(keys)
				return keys[idx], idx, nil
			}
		}
		// Fallback – should not happen, but return first enabled key
		return keys[enabledIdx[0]], enabledIdx[0], nil
	default:
		// Unknown mode, default to first enabled key (or original key string)
		return keys[enabledIdx[0]], enabledIdx[0], nil
	}
}

// HasEnabledKey is a read-only availability check used by routing before a
// candidate is scored. It avoids consuming polling state or returning a
// channel that cannot provide credentials for this request.
func (channel *Channel) HasEnabledKey() bool {
	if channel == nil {
		return false
	}
	if !channel.ChannelInfo.IsMultiKey {
		return strings.TrimSpace(channel.Key) != ""
	}
	keys := channel.GetKeys()
	if len(keys) == 0 {
		return false
	}
	for index := range keys {
		status, exists := channel.ChannelInfo.MultiKeyStatusList[index]
		// MultiKeyStatusList stores only explicit disabled states. A missing
		// entry therefore has the same enabled-by-default meaning as
		// GetNextEnabledKey's status lookup.
		if !exists || status == common.ChannelStatusEnabled {
			return true
		}
	}
	return false
}

func (channel *Channel) SaveChannelInfo() error {
	return DB.Model(channel).Update("channel_info", channel.ChannelInfo).Error
}

func (channel *Channel) GetModels() []string {
	if channel.Models == "" {
		return []string{}
	}
	return strings.Split(strings.Trim(channel.Models, ","), ",")
}

func (channel *Channel) GetGroups() []string {
	if channel.Group == "" {
		return []string{}
	}
	groups := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawGroup := range strings.Split(channel.Group, ",") {
		group := strings.TrimSpace(rawGroup)
		if group == "" {
			continue
		}
		if _, exists := seen[group]; exists {
			continue
		}
		seen[group] = struct{}{}
		groups = append(groups, group)
	}
	return groups
}

func (channel *Channel) NormalizeGroups() {
	if channel == nil {
		return
	}
	channel.Group = strings.Join(channel.GetGroups(), ",")
}

func (channel *Channel) GetOtherInfo() map[string]interface{} {
	otherInfo := make(map[string]interface{})
	if channel.OtherInfo != "" {
		err := common.Unmarshal([]byte(channel.OtherInfo), &otherInfo)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal other info: channel_id=%d, tag=%s, name=%s, error=%v", channel.Id, channel.GetTag(), channel.Name, err))
		}
	}
	return otherInfo
}

func (channel *Channel) SetOtherInfo(otherInfo map[string]interface{}) {
	otherInfoBytes, err := json.Marshal(otherInfo)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal other info: channel_id=%d, tag=%s, name=%s, error=%v", channel.Id, channel.GetTag(), channel.Name, err))
		return
	}
	channel.OtherInfo = string(otherInfoBytes)
}

func (channel *Channel) GetTag() string {
	if channel.Tag == nil {
		return ""
	}
	return *channel.Tag
}

func (channel *Channel) SetTag(tag string) {
	channel.Tag = &tag
}

func (channel *Channel) GetAutoBan() bool {
	if channel.AutoBan == nil {
		return false
	}
	return *channel.AutoBan == 1
}

func (channel *Channel) GetProbeIntervalSeconds() int {
	if channel == nil || channel.ProbeIntervalSeconds <= 0 {
		return DefaultChannelProbeIntervalSeconds
	}
	return channel.ProbeIntervalSeconds
}

func (channel *Channel) GetAutoDisabledProbeIntervalSeconds() int {
	if channel == nil || channel.AutoDisabledProbeIntervalSeconds <= 0 {
		return DefaultAutoDisabledProbeIntervalSeconds
	}
	return channel.AutoDisabledProbeIntervalSeconds
}

func (channel *Channel) ShouldProbeFailureAutoBan() bool {
	if channel == nil {
		return false
	}
	if channel.ProbeFailureAutoBan != nil {
		return *channel.ProbeFailureAutoBan
	}
	return channel.GetAutoBan()
}

func (channel *Channel) ShouldProbeSuccessAutoEnable() bool {
	if channel == nil {
		return false
	}
	if channel.ProbeSuccessAutoEnable != nil {
		return *channel.ProbeSuccessAutoEnable
	}
	return true
}

func (channel *Channel) GetTestModel() string {
	if channel == nil {
		return ""
	}
	if channel.TestModel != nil {
		return strings.TrimSpace(*channel.TestModel)
	}
	return ""
}

// GetUpstreamMaxRetries returns retries after the first upstream attempt. A
// nil value uses the product default; an explicit zero means one attempt.
func (channel *Channel) GetUpstreamMaxRetries() int {
	if channel == nil || channel.UpstreamMaxRetries == nil {
		return DefaultChannelUpstreamMaxRetries
	}
	if *channel.UpstreamMaxRetries < 0 {
		return 0
	}
	return *channel.UpstreamMaxRetries
}

// GetMaxConcurrency returns the maximum number of requests that may be
// active on this channel. A missing or non-positive value uses the product
// default of 1000.
func (channel *Channel) GetMaxConcurrency() int {
	if channel == nil || channel.MaxConcurrency == nil || *channel.MaxConcurrency <= 0 {
		return DefaultChannelMaxConcurrency
	}
	return *channel.MaxConcurrency
}

func (channel *Channel) GetPriceMultiplier() float64 {
	if channel == nil || channel.PriceMultiplier <= 0 || math.IsNaN(channel.PriceMultiplier) || math.IsInf(channel.PriceMultiplier, 0) {
		return 1
	}
	return channel.PriceMultiplier
}

func (channel *Channel) GetPriceMultiplierMode() string {
	if channel == nil {
		return ChannelPriceMultiplierModeUSD
	}
	switch strings.ToLower(strings.TrimSpace(channel.PriceMultiplierMode)) {
	case ChannelPriceMultiplierModeCNY:
		return ChannelPriceMultiplierModeCNY
	default:
		return ChannelPriceMultiplierModeUSD
	}
}

func (channel *Channel) CalculateTokenCostUSD(tokens int64, billingUSDToCNYRate float64) float64 {
	if channel == nil || tokens <= 0 {
		return 0
	}
	multiplier := channel.GetPriceMultiplier()
	if channel.GetPriceMultiplierMode() == ChannelPriceMultiplierModeCNY &&
		billingUSDToCNYRate > 0 && !math.IsNaN(billingUSDToCNYRate) && !math.IsInf(billingUSDToCNYRate, 0) {
		multiplier /= billingUSDToCNYRate
	}
	return float64(tokens) / 1_000_000 * multiplier
}

func (channel *Channel) IsForcePriority() bool {
	return channel != nil && channel.ForcePriority != nil && *channel.ForcePriority
}

func (channel *Channel) GetForcePriorityScope() string {
	if channel == nil {
		return ChannelForcePriorityScopeGroup
	}
	if strings.EqualFold(strings.TrimSpace(channel.ForcePriorityScope), ChannelForcePriorityScopeCrossGroup) {
		return ChannelForcePriorityScopeCrossGroup
	}
	return ChannelForcePriorityScopeGroup
}

func (channel *Channel) Save() error {
	return DB.Save(channel).Error
}

func (channel *Channel) SaveWithoutKey() error {
	if channel.Id == 0 {
		return errors.New("channel ID is 0")
	}
	return DB.Omit("key").Save(channel).Error
}

func GetAllChannels(startIdx int, num int, selectAll bool, idSort bool, sortOptions ...ChannelSortOptions) ([]*Channel, error) {
	var channels []*Channel
	var err error
	order := resolveChannelSortOptions(idSort, sortOptions)
	if selectAll {
		err = order.Apply(DB).Find(&channels).Error
	} else {
		err = order.Apply(DB).Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	}
	return channels, err
}

func GetChannelsByTag(tag string, idSort bool, selectAll bool, sortOptions ...ChannelSortOptions) ([]*Channel, error) {
	var channels []*Channel
	order := resolveChannelSortOptions(idSort, sortOptions)
	query := order.Apply(DB.Where("tag = ?", tag))
	if !selectAll {
		query = query.Omit("key")
	}
	err := query.Find(&channels).Error
	return channels, err
}

func SearchChannels(keyword string, group string, model string, idSort bool, sortOptions ...ChannelSortOptions) ([]*Channel, error) {
	var channels []*Channel
	modelsCol := "`models`"

	// 如果是 PostgreSQL，使用双引号
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		modelsCol = `"models"`
	}

	baseURLCol := "`base_url`"
	// 如果是 PostgreSQL，使用双引号
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		baseURLCol = `"base_url"`
	}

	order := resolveChannelSortOptions(idSort, sortOptions)

	// 构造基础查询
	baseQuery := DB.Model(&Channel{}).Omit("key")

	// 构造WHERE子句
	whereClause := "(id = ? OR name LIKE ? OR " + commonKeyCol + " = ? OR " + baseURLCol + " LIKE ?) AND " + modelsCol + " LIKE ?"
	args := []any{common.String2Int(keyword), "%" + keyword + "%", keyword, "%" + keyword + "%", "%" + model + "%"}
	baseQuery = ApplyChannelGroupFilter(baseQuery.Where(whereClause, args...), group)

	// 执行查询
	err := order.Apply(baseQuery).Find(&channels).Error
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func GetChannelById(id int, selectAll bool) (*Channel, error) {
	channel := &Channel{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(channel, "id = ?", id).Error
	} else {
		err = DB.Omit("key").First(channel, "id = ?", id).Error
	}
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func BatchInsertChannels(channels []Channel) error {
	if len(channels) == 0 {
		return nil
	}
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for index := range channels {
		channels[index].NormalizeGroups()
	}
	for _, chunk := range lo.Chunk(channels, 50) {
		if err := tx.Create(&chunk).Error; err != nil {
			tx.Rollback()
			return err
		}
		for _, channel_ := range chunk {
			if err := channel_.AddAbilities(tx); err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit().Error
}

func BatchDeleteChannels(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureChannelsCanBeDeleted(tx, ids); err != nil {
			return err
		}
		return deleteChannelRecords(tx, ids)
	})
}

func (channel *Channel) GetWeight() int {
	if channel.Weight == nil {
		return 0
	}
	return int(*channel.Weight)
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	url := *channel.BaseURL
	if url == "" {
		url = constant.ChannelBaseURLs[channel.Type]
	}
	return url
}

func (channel *Channel) GetModelMapping() string {
	if channel.ModelMapping == nil {
		return ""
	}
	return *channel.ModelMapping
}

func (channel *Channel) GetStatusCodeMapping() string {
	if channel.StatusCodeMapping == nil {
		return ""
	}
	return *channel.StatusCodeMapping
}

func (channel *Channel) Insert() error {
	channel.NormalizeGroups()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(channel).Error; err != nil {
			return err
		}
		return channel.AddAbilities(tx)
	})
}

// Update persists a channel. When fields are supplied, explicitly selected
// zero values are written as well; callers without a field list retain the
// legacy full-model update behavior.
func (channel *Channel) Update(fields ...string) error {
	channel.NormalizeGroups()
	// If this is a multi-key channel, recalculate MultiKeySize based on the current key list to avoid inconsistency after editing keys
	if channel.ChannelInfo.IsMultiKey {
		var keyStr string
		if channel.Key != "" {
			keyStr = channel.Key
		} else {
			// If key is not provided, read the existing key from the database
			if existing, err := GetChannelById(channel.Id, true); err == nil {
				keyStr = existing.Key
			}
		}
		// Parse the key list (supports newline separation or JSON array)
		keys := []string{}
		if keyStr != "" {
			trimmed := strings.TrimSpace(keyStr)
			if strings.HasPrefix(trimmed, "[") {
				var arr []json.RawMessage
				if err := common.Unmarshal([]byte(trimmed), &arr); err == nil {
					keys = make([]string, len(arr))
					for i, v := range arr {
						keys[i] = string(v)
					}
				}
			}
			if len(keys) == 0 { // fallback to newline split
				keys = strings.Split(strings.Trim(keyStr, "\n"), "\n")
			}
		}
		channel.ChannelInfo.MultiKeySize = len(keys)
		// Clean up status data that exceeds the new key count to prevent index out of range
		if channel.ChannelInfo.MultiKeyStatusList != nil {
			for idx := range channel.ChannelInfo.MultiKeyStatusList {
				if idx >= channel.ChannelInfo.MultiKeySize {
					delete(channel.ChannelInfo.MultiKeyStatusList, idx)
				}
			}
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		updateQuery := tx.Model(channel)
		if len(fields) > 0 {
			updateQuery = updateQuery.Select(fields)
		}
		if err := updateQuery.Updates(channel).Error; err != nil {
			return err
		}
		if err := tx.Model(channel).First(channel, "id = ?", channel.Id).Error; err != nil {
			return err
		}
		return channel.UpdateAbilities(tx)
	})
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("response_time", "test_time").Updates(Channel{
		TestTime:     common.GetTimestamp(),
		ResponseTime: int(responseTime),
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update response time: channel_id=%d, error=%v", channel.Id, err))
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_time", "balance").Updates(Channel{
		BalanceUpdatedTime: common.GetTimestamp(),
		Balance:            balance,
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update balance: channel_id=%d, error=%v", channel.Id, err))
	}
}

func (channel *Channel) Delete() error {
	if channel == nil || channel.Id <= 0 {
		return errors.New("channel id must be positive")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		ids := []int{channel.Id}
		if err := ensureChannelsCanBeDeleted(tx, ids); err != nil {
			return err
		}
		return deleteChannelRecords(tx, ids)
	})
}

// ensureChannelsCanBeDeleted prevents a raw channel deletion from leaving a
// billing-group route pointing at a missing channel. The routing editor must
// remove the channel from its route first, preserving a valid saved config.
func ensureChannelsCanBeDeleted(db *gorm.DB, ids []int) error {
	if db == nil || len(ids) == 0 {
		return nil
	}
	var routeChannel BillingGroupChannel
	if err := db.Select("channel_id").Where("channel_id IN ?", ids).First(&routeChannel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return fmt.Errorf("channel %d is referenced by billing-group routing; remove it from the routing configuration before deleting", routeChannel.ChannelId)
}

// deleteChannelRecords removes a channel and all records owned by it. Routing
// references are intentionally not deleted here; callers must reject those
// channels first so a saved billing-group route cannot be silently changed.
func deleteChannelRecords(tx *gorm.DB, ids []int) error {
	deleteProbeHistory := tx.Migrator().HasTable(&ChannelProbeHistory{})
	deleteProbeState := tx.Migrator().HasTable(&ChannelProbeState{})
	deleteErrorMappings := tx.Migrator().HasTable(&UpstreamErrorMapping{})
	deleteCostEntries := tx.Migrator().HasTable(&ChannelCostEntry{})
	for _, chunk := range lo.Chunk(ids, 200) {
		if deleteErrorMappings {
			if err := tx.Where("channel_id IN ?", chunk).Delete(&UpstreamErrorMapping{}).Error; err != nil {
				return err
			}
		}
		if deleteCostEntries {
			if err := tx.Where("channel_id IN ?", chunk).Delete(&ChannelCostEntry{}).Error; err != nil {
				return err
			}
		}
		if deleteProbeHistory {
			if err := tx.Where("channel_id IN ?", chunk).Delete(&ChannelProbeHistory{}).Error; err != nil {
				return err
			}
		}
		if deleteProbeState {
			if err := tx.Where("channel_id IN ?", chunk).Delete(&ChannelProbeState{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("channel_id IN ?", chunk).Delete(&Ability{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", chunk).Delete(&Channel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

var channelStatusLock sync.Mutex

// channelPollingLocks stores locks for each channel.id to ensure thread-safe polling
var channelPollingLocks sync.Map

// GetChannelPollingLock returns or creates a mutex for the given channel ID
func GetChannelPollingLock(channelId int) *sync.Mutex {
	if lock, exists := channelPollingLocks.Load(channelId); exists {
		return lock.(*sync.Mutex)
	}
	// Create new lock for this channel
	newLock := &sync.Mutex{}
	actual, _ := channelPollingLocks.LoadOrStore(channelId, newLock)
	return actual.(*sync.Mutex)
}

// CleanupChannelPollingLocks removes locks for channels that no longer exist
// This is optional and can be called periodically to prevent memory leaks
func CleanupChannelPollingLocks() {
	var activeChannelIds []int
	DB.Model(&Channel{}).Pluck("id", &activeChannelIds)

	activeChannelSet := make(map[int]bool)
	for _, id := range activeChannelIds {
		activeChannelSet[id] = true
	}

	channelPollingLocks.Range(func(key, value interface{}) bool {
		channelId := key.(int)
		if !activeChannelSet[channelId] {
			channelPollingLocks.Delete(channelId)
		}
		return true
	})
}

func handlerMultiKeyUpdate(channel *Channel, usingKey string, status int, reason string) {
	keys := channel.GetKeys()
	if len(keys) == 0 {
		channel.Status = status
	} else {
		keyIndex := -1
		for i, key := range keys {
			if key == usingKey {
				keyIndex = i
				break
			}
		}
		if keyIndex < 0 {
			if usingKey != "" {
				common.SysLog(fmt.Sprintf("failed to update multi-key status: channel_id=%d, using key not found", channel.Id))
				return
			}
			if status == common.ChannelStatusEnabled && !hasEnabledMultiKey(keys, channel.ChannelInfo.MultiKeyStatusList) {
				return
			}
			channel.Status = status
			info := channel.GetOtherInfo()
			info["status_reason"] = reason
			info["status_time"] = common.GetTimestamp()
			channel.SetOtherInfo(info)
			return
		}
		if channel.ChannelInfo.MultiKeyStatusList == nil {
			channel.ChannelInfo.MultiKeyStatusList = make(map[int]int)
		}
		if status == common.ChannelStatusEnabled {
			delete(channel.ChannelInfo.MultiKeyStatusList, keyIndex)
			delete(channel.ChannelInfo.MultiKeyDisabledReason, keyIndex)
			delete(channel.ChannelInfo.MultiKeyDisabledTime, keyIndex)
		} else {
			channel.ChannelInfo.MultiKeyStatusList[keyIndex] = status
			if channel.ChannelInfo.MultiKeyDisabledReason == nil {
				channel.ChannelInfo.MultiKeyDisabledReason = make(map[int]string)
			}
			if channel.ChannelInfo.MultiKeyDisabledTime == nil {
				channel.ChannelInfo.MultiKeyDisabledTime = make(map[int]int64)
			}
			channel.ChannelInfo.MultiKeyDisabledReason[keyIndex] = reason
			channel.ChannelInfo.MultiKeyDisabledTime[keyIndex] = common.GetTimestamp()
		}
		channel.ReconcileMultiKeyAvailability(status == common.ChannelStatusEnabled)
	}
}

// ReconcileMultiKeyAvailability keeps the channel status aligned with its key
// availability. Explicit key enable operations may recover an auto-disabled
// channel, while a manually disabled channel always remains disabled.
func (channel *Channel) ReconcileMultiKeyAvailability(allowAutoRecovery bool) {
	if channel == nil || !channel.ChannelInfo.IsMultiKey {
		return
	}
	if hasEnabledMultiKey(channel.GetKeys(), channel.ChannelInfo.MultiKeyStatusList) {
		if allowAutoRecovery && channel.Status == common.ChannelStatusAutoDisabled {
			channel.Status = common.ChannelStatusEnabled
			info := channel.GetOtherInfo()
			delete(info, "status_reason")
			delete(info, "status_time")
			channel.SetOtherInfo(info)
		}
		return
	}
	if channel.Status == common.ChannelStatusManuallyDisabled {
		return
	}
	if channel.Status != common.ChannelStatusAutoDisabled {
		info := channel.GetOtherInfo()
		info["status_reason"] = "All keys are disabled"
		info["status_time"] = common.GetTimestamp()
		channel.SetOtherInfo(info)
	}
	channel.Status = common.ChannelStatusAutoDisabled
}

func hasEnabledMultiKey(keys []string, statusList map[int]int) bool {
	for i := range keys {
		if statusList == nil {
			return true
		}
		status, ok := statusList[i]
		if !ok || status == common.ChannelStatusEnabled {
			return true
		}
	}
	return false
}

func UpdateChannelStatus(channelId int, usingKey string, status int, reason string) bool {
	if channelId <= 0 {
		return false
	}

	channelStatusLock.Lock()
	defer channelStatusLock.Unlock()
	pollingLock := GetChannelPollingLock(channelId)
	pollingLock.Lock()

	var updatedChannel *Channel
	statusChanged := false
	changed := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var channel Channel
		if err := lockForUpdate(tx).Where("id = ?", channelId).First(&channel).Error; err != nil {
			return err
		}
		beforeStatus := channel.Status
		beforeOtherInfo := channel.OtherInfo
		beforeChannelInfo, err := common.Marshal(channel.ChannelInfo)
		if err != nil {
			return err
		}
		if channel.ChannelInfo.IsMultiKey {
			handlerMultiKeyUpdate(&channel, usingKey, status, reason)
		} else {
			if channel.Status == status {
				updatedChannel = &channel
				return nil
			}
			info := channel.GetOtherInfo()
			info["status_reason"] = reason
			info["status_time"] = common.GetTimestamp()
			channel.SetOtherInfo(info)
			channel.Status = status
		}

		afterChannelInfo, err := common.Marshal(channel.ChannelInfo)
		if err != nil {
			return err
		}
		statusChanged = beforeStatus != channel.Status
		changed = statusChanged || beforeOtherInfo != channel.OtherInfo || string(beforeChannelInfo) != string(afterChannelInfo)
		updatedChannel = &channel
		if !changed {
			return nil
		}
		if err := tx.Model(&Channel{}).Where("id = ?", channelId).Updates(map[string]any{
			"status":       channel.Status,
			"other_info":   channel.OtherInfo,
			"channel_info": channel.ChannelInfo,
		}).Error; err != nil {
			return err
		}
		if statusChanged {
			return tx.Model(&Ability{}).Where("channel_id = ?", channelId).
				Update("enabled", channel.Status == common.ChannelStatusEnabled).Error
		}
		return nil
	})
	pollingLock.Unlock()
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update channel status: channel_id=%d, status=%d, error=%v", channelId, status, err))
		return false
	}
	if !changed || updatedChannel == nil || !common.MemoryCacheEnabled {
		return changed
	}
	SyncChannelCacheEntry(updatedChannel)
	return changed
}

func EnableChannelByTag(tag string) error {
	err := DB.Model(&Channel{}).Where("tag = ?", tag).Update("status", common.ChannelStatusEnabled).Error
	if err != nil {
		return err
	}
	err = UpdateAbilityStatusByTag(tag, true)
	return err
}

func DisableChannelByTag(tag string) error {
	err := DB.Model(&Channel{}).Where("tag = ?", tag).Update("status", common.ChannelStatusManuallyDisabled).Error
	if err != nil {
		return err
	}
	err = UpdateAbilityStatusByTag(tag, false)
	return err
}

func EditChannelByTag(tag string, newTag *string, modelMapping *string, models *string, group *string, weight *uint, paramOverride *string, headerOverride *string) error {
	updateData := Channel{}
	shouldReCreateAbilities := false
	// 如果 newTag 不为空且不等于 tag，则更新 tag
	if newTag != nil && *newTag != tag {
		updateData.Tag = newTag
	}
	if modelMapping != nil {
		updateData.ModelMapping = modelMapping
	}
	if models != nil && *models != "" {
		shouldReCreateAbilities = true
		updateData.Models = *models
	}
	if group != nil && *group != "" {
		shouldReCreateAbilities = true
		updateData.Group = *group
		updateData.NormalizeGroups()
	}
	if weight != nil {
		updateData.Weight = weight
	}
	if paramOverride != nil {
		updateData.ParamOverride = paramOverride
	}
	if headerOverride != nil {
		updateData.HeaderOverride = headerOverride
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		var channelIds []int
		if err := tx.Model(&Channel{}).Where("tag = ?", tag).Pluck("id", &channelIds).Error; err != nil {
			return err
		}
		if len(channelIds) == 0 {
			return nil
		}
		if err := tx.Model(&Channel{}).Where("id IN ?", channelIds).Updates(updateData).Error; err != nil {
			return err
		}
		if shouldReCreateAbilities {
			var channels []*Channel
			if err := tx.Where("id IN ?", channelIds).Find(&channels).Error; err != nil {
				return err
			}
			for _, channel := range channels {
				if err := channel.UpdateAbilities(tx); err != nil {
					return fmt.Errorf("update abilities for channel %d: %w", channel.Id, err)
				}
			}
			return nil
		}

		abilityUpdates := make(map[string]any, 2)
		if newTag != nil {
			abilityUpdates["tag"] = *newTag
		}
		if weight != nil {
			abilityUpdates["weight"] = *weight
		}
		if len(abilityUpdates) == 0 {
			return nil
		}
		return tx.Model(&Ability{}).Where("channel_id IN ?", channelIds).Updates(abilityUpdates).Error
	})
}

func UpdateChannelUsedQuota(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeChannelUsedQuota, id, quota)
		return
	}
	updateChannelUsedQuota(id, quota)
}

func updateChannelUsedQuota(id int, quota int) {
	err := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to update channel used quota: channel_id=%d, delta_quota=%d, error=%v", id, quota, err))
	}
}

func DeleteChannelByStatus(status int64) (int64, error) {
	return deleteChannelsByStatuses([]int64{status})
}

func DeleteDisabledChannel() (int64, error) {
	return deleteChannelsByStatuses([]int64{common.ChannelStatusAutoDisabled, common.ChannelStatusManuallyDisabled})
}

func deleteChannelsByStatuses(statuses []int64) (int64, error) {
	var deleted int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var ids []int
		if err := tx.Model(&Channel{}).Where("status IN ?", statuses).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := ensureChannelsCanBeDeleted(tx, ids); err != nil {
			return err
		}
		if err := deleteChannelRecords(tx, ids); err != nil {
			return err
		}
		deleted = int64(len(ids))
		return nil
	})
	return deleted, err
}

func GetPaginatedTags(offset int, limit int) ([]*string, error) {
	return GetPaginatedChannelTags(DB.Model(&Channel{}), offset, limit)
}

func GetPaginatedChannelTags(query *gorm.DB, offset int, limit int) ([]*string, error) {
	var tags []*string
	err := query.
		Select("DISTINCT tag").
		Where("tag is not null AND tag != ''").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "tag"}}).
		Offset(offset).
		Limit(limit).
		Find(&tags).Error
	return tags, err
}

func SearchTags(keyword string, group string, model string, idSort bool) ([]*string, error) {
	var tags []*string
	modelsCol := "`models`"

	// 如果是 PostgreSQL，使用双引号
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		modelsCol = `"models"`
	}

	baseURLCol := "`base_url`"
	// 如果是 PostgreSQL，使用双引号
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		baseURLCol = `"base_url"`
	}

	order := "id desc"

	// 构造基础查询
	baseQuery := DB.Model(&Channel{}).Omit("key")

	// 构造WHERE子句
	whereClause := "(id = ? OR name LIKE ? OR " + commonKeyCol + " = ? OR " + baseURLCol + " LIKE ?) AND " + modelsCol + " LIKE ?"
	args := []any{common.String2Int(keyword), "%" + keyword + "%", keyword, "%" + keyword + "%", "%" + model + "%"}
	baseQuery = ApplyChannelGroupFilter(baseQuery.Where(whereClause, args...), group)

	subQuery := baseQuery.
		Select("tag").
		Where("tag != ''").
		Order(order)

	err := DB.Table("(?) as sub", subQuery).
		Select("DISTINCT tag").
		Find(&tags).Error

	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (channel *Channel) ValidateSettings() error {
	channelParams := &dto.ChannelSettings{}
	if channel.Setting != nil && *channel.Setting != "" {
		err := common.Unmarshal([]byte(*channel.Setting), channelParams)
		if err != nil {
			return err
		}
	}
	channelOtherSettings := &dto.ChannelOtherSettings{}
	if channel.OtherSettings != "" {
		err := common.UnmarshalJsonStr(channel.OtherSettings, channelOtherSettings)
		if err != nil {
			return err
		}
	}
	if channel.Type == constant.ChannelTypeAdvancedCustom {
		if channelOtherSettings.AdvancedCustom == nil {
			return fmt.Errorf("advanced_custom is required")
		}
	}
	if channelOtherSettings.AdvancedCustom != nil {
		if err := channelOtherSettings.AdvancedCustom.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) GetSetting() dto.ChannelSettings {
	setting := dto.ChannelSettings{}
	if channel.Setting != nil && *channel.Setting != "" {
		err := common.Unmarshal([]byte(*channel.Setting), &setting)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal setting: channel_id=%d, error=%v", channel.Id, err))
			channel.Setting = nil // 清空设置以避免后续错误
			_ = channel.Save()    // 保存修改
		}
	}
	return setting
}

func (channel *Channel) SetSetting(setting dto.ChannelSettings) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal setting: channel_id=%d, error=%v", channel.Id, err))
		return
	}
	channel.Setting = common.GetPointer[string](string(settingBytes))
}

func (channel *Channel) GetOtherSettings() dto.ChannelOtherSettings {
	setting := dto.ChannelOtherSettings{}
	if channel.OtherSettings != "" {
		err := common.UnmarshalJsonStr(channel.OtherSettings, &setting)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal setting: channel_id=%d, error=%v", channel.Id, err))
			channel.OtherSettings = "{}" // 清空设置以避免后续错误
			_ = channel.Save()           // 保存修改
		}
	}
	return setting
}

func (channel *Channel) SetOtherSettings(setting dto.ChannelOtherSettings) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal setting: channel_id=%d, error=%v", channel.Id, err))
		return
	}
	channel.OtherSettings = string(settingBytes)
}

func (channel *Channel) GetParamOverride() map[string]interface{} {
	paramOverride := make(map[string]interface{})
	if channel.ParamOverride != nil && *channel.ParamOverride != "" {
		err := common.Unmarshal([]byte(*channel.ParamOverride), &paramOverride)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal param override: channel_id=%d, error=%v", channel.Id, err))
		}
	}
	return paramOverride
}

func (channel *Channel) GetHeaderOverride() map[string]interface{} {
	headerOverride := make(map[string]interface{})
	if channel.HeaderOverride != nil && *channel.HeaderOverride != "" {
		err := common.Unmarshal([]byte(*channel.HeaderOverride), &headerOverride)
		if err != nil {
			common.SysLog(fmt.Sprintf("failed to unmarshal header override: channel_id=%d, error=%v", channel.Id, err))
		}
	}
	return headerOverride
}

func GetChannelsByIds(ids []int) ([]*Channel, error) {
	var channels []*Channel
	err := DB.Where("id in (?)", ids).Find(&channels).Error
	return channels, err
}

func BatchSetChannelTag(ids []int, tag *string) error {
	if len(ids) == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Channel{}).Where("id IN ?", ids).Update("tag", tag).Error; err != nil {
			return err
		}
		return tx.Model(&Ability{}).Where("channel_id IN ?", ids).Update("tag", tag).Error
	})
}

// CountAllChannels returns total channels in DB
func CountAllChannels() (int64, error) {
	var total int64
	err := DB.Model(&Channel{}).Count(&total).Error
	return total, err
}

// CountAllTags returns number of non-empty distinct tags
func CountAllTags() (int64, error) {
	return CountChannelTags(DB.Model(&Channel{}))
}

func CountChannelTags(query *gorm.DB) (int64, error) {
	var total int64
	err := query.Where("tag is not null AND tag != ''").Distinct("tag").Count(&total).Error
	return total, err
}

// Get channels of specified type with pagination
func GetChannelsByType(startIdx int, num int, idSort bool, channelType int) ([]*Channel, error) {
	var channels []*Channel
	order := "id desc"
	err := DB.Where("type = ?", channelType).Order(order).Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	return channels, err
}

// Count channels of specific type
func CountChannelsByType(channelType int) (int64, error) {
	var count int64
	err := DB.Model(&Channel{}).Where("type = ?", channelType).Count(&count).Error
	return count, err
}

// Return map[type]count for all channels
func CountChannelsGroupByType() (map[int64]int64, error) {
	type result struct {
		Type  int64 `gorm:"column:type"`
		Count int64 `gorm:"column:count"`
	}
	var results []result
	err := DB.Model(&Channel{}).Select("type, count(*) as count").Group("type").Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int64)
	for _, r := range results {
		counts[r.Type] = r.Count
	}
	return counts, nil
}
