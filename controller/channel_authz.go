package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// channelPersistedUpdateFields is the allowlist passed to GORM for partial
// channel updates. Request-only fields and retired routing fields never reach
// the database update statement.
var channelPersistedUpdateFields = []struct {
	jsonName string
	column   string
}{
	{jsonName: "type", column: "type"},
	{jsonName: "key", column: "key"},
	{jsonName: "openai_organization", column: "openai_organization"},
	{jsonName: "test_model", column: "test_model"},
	{jsonName: "name", column: "name"},
	{jsonName: "base_url", column: "base_url"},
	{jsonName: "other", column: "other"},
	{jsonName: "models", column: "models"},
	{jsonName: "group", column: "group"},
	{jsonName: "model_mapping", column: "model_mapping"},
	{jsonName: "status_code_mapping", column: "status_code_mapping"},
	{jsonName: "auto_ban", column: "auto_ban"},
	{jsonName: "probe_interval_seconds", column: "probe_interval_seconds"},
	{jsonName: "auto_disabled_probe_interval_seconds", column: "auto_disabled_probe_interval_seconds"},
	{jsonName: "probe_failure_auto_ban", column: "probe_failure_auto_ban"},
	{jsonName: "probe_success_auto_enable", column: "probe_success_auto_enable"},
	{jsonName: "upstream_max_retries", column: "upstream_max_retries"},
	{jsonName: "price_multiplier", column: "price_multiplier"},
	{jsonName: "price_multiplier_mode", column: "price_multiplier_mode"},
	{jsonName: "force_priority", column: "force_priority"},
	{jsonName: "force_priority_scope", column: "force_priority_scope"},
	{jsonName: "other_info", column: "other_info"},
	{jsonName: "tag", column: "tag"},
	{jsonName: "setting", column: "setting"},
	{jsonName: "param_override", column: "param_override"},
	{jsonName: "header_override", column: "header_override"},
	{jsonName: "remark", column: "remark"},
	{jsonName: "weight", column: "weight"},
	{jsonName: "channel_info", column: "channel_info"},
	{jsonName: "settings", column: "settings"},
	{jsonName: "multi_key_mode", column: "channel_info"},
}

func channelUpdateColumns(requestData map[string]any) []string {
	columns := make([]string, 0, len(requestData)+1)
	seen := make(map[string]struct{}, len(requestData))
	keyProvided := false
	if value, ok := requestData["key"]; ok {
		key, isString := value.(string)
		keyProvided = isString && strings.TrimSpace(key) != ""
	}
	for _, field := range channelPersistedUpdateFields {
		if _, ok := requestData[field.jsonName]; !ok {
			continue
		}
		// The edit form deliberately sends no stored credential back to the
		// browser. Treat an empty key as "unchanged" so a partial update cannot
		// clear the credential or bypass ChannelSensitiveWrite.
		if field.jsonName == "key" && !keyProvided {
			continue
		}
		if _, ok := seen[field.column]; ok {
			continue
		}
		seen[field.column] = struct{}{}
		columns = append(columns, field.column)
	}
	if keyProvided {
		if _, exists := seen["channel_info"]; !exists {
			columns = append(columns, "channel_info")
		}
	}
	if _, ok := requestData["key_mode"]; ok {
		if _, exists := seen["channel_info"]; !exists {
			columns = append(columns, "channel_info")
		}
	}
	return columns
}

func channelHasSensitiveChanges(channel *PatchChannel, origin *model.Channel, requestData map[string]any) bool {
	if _, ok := requestData["type"]; ok && channel.Type != origin.Type {
		return true
	}
	if _, ok := requestData["key"]; ok && strings.TrimSpace(channel.Key) != "" && channel.Key != origin.Key {
		return true
	}
	if _, ok := requestData["base_url"]; ok && !equalStringPtr(channel.BaseURL, origin.BaseURL) {
		return true
	}
	if _, ok := requestData["openai_organization"]; ok && !equalStringPtr(channel.OpenAIOrganization, origin.OpenAIOrganization) {
		return true
	}
	if _, ok := requestData["header_override"]; ok && !equalStringPtr(channel.HeaderOverride, origin.HeaderOverride) {
		return true
	}
	if _, ok := requestData["param_override"]; ok && !equalStringPtr(channel.ParamOverride, origin.ParamOverride) {
		return true
	}
	if _, ok := requestData["setting"]; ok && !equalStringPtr(channel.Setting, origin.Setting) {
		return true
	}
	if _, ok := requestData["other"]; ok && channel.Other != origin.Other {
		return true
	}
	if _, ok := requestData["settings"]; ok && channel.OtherSettings != origin.OtherSettings {
		return true
	}
	if _, ok := requestData["key_mode"]; ok && channel.KeyMode != nil {
		return true
	}
	// Fail closed: any field present in the request that is neither a known
	// sensitive field (gated above) nor an explicitly classified non-sensitive
	// field must be treated as sensitive. This keeps a newly added channel field
	// from silently becoming editable by ChannelWrite-only admins until it is
	// consciously classified in channelNonSensitiveFields.
	for field := range requestData {
		if _, ok := channelSensitiveFields[field]; ok {
			continue
		}
		if _, ok := channelNonSensitiveFields[field]; ok {
			continue
		}
		if _, ok := channelOperationalFields[field]; ok {
			continue
		}
		if _, ok := channelReadOnlyFields[field]; ok {
			continue
		}
		return true
	}
	return false
}

// channelSensitiveFields lists the channel fields whose modification requires
// ChannelSensitiveWrite. They are each checked individually in
// channelHasSensitiveChanges with a precise old-vs-new comparison; this set is
// used to exclude them from the fail-closed scan for unknown fields.
var channelSensitiveFields = map[string]struct{}{
	"type":                {},
	"key":                 {},
	"base_url":            {},
	"openai_organization": {},
	"header_override":     {},
	"param_override":      {},
	"setting":             {},
	"other":               {},
	"settings":            {},
	"key_mode":            {},
}

// channelOperationalFields lists fields managed by operation endpoints instead
// of the general channel edit endpoint.
var channelOperationalFields = map[string]struct{}{
	"status": {},
}

// channelReadOnlyFields lists server-managed/accounting fields that the general
// channel edit endpoint must ignore even if a client sends them.
var channelReadOnlyFields = map[string]struct{}{
	"created_time":                    {},
	"test_time":                       {},
	"response_time":                   {},
	"balance":                         {},
	"balance_updated_time":            {},
	"used_quota":                      {},
	"previous_day_probe_success_rate": {},
}

func clearChannelReadOnlyFields(channel *PatchChannel, requestData map[string]any) {
	if _, ok := requestData["created_time"]; ok {
		channel.CreatedTime = 0
	}
	if _, ok := requestData["test_time"]; ok {
		channel.TestTime = 0
	}
	if _, ok := requestData["response_time"]; ok {
		channel.ResponseTime = 0
	}
	if _, ok := requestData["balance"]; ok {
		channel.Balance = 0
	}
	if _, ok := requestData["balance_updated_time"]; ok {
		channel.BalanceUpdatedTime = 0
	}
	if _, ok := requestData["used_quota"]; ok {
		channel.UsedQuota = 0
	}
}

// channelNonSensitiveFields lists routing / server-managed channel
// fields a ChannelWrite admin may edit without ChannelSensitiveWrite. When a new
// field is added to model.Channel it must be added to either this set or
// channelSensitiveFields or channelOperationalFields; otherwise it falls through
// to the fail-closed branch and is treated as sensitive. The
// TestChannelFieldsAreClassified guard test enforces this.
var channelNonSensitiveFields = map[string]struct{}{
	"id":                                   {},
	"test_model":                           {},
	"probe_interval_seconds":               {},
	"auto_disabled_probe_interval_seconds": {},
	"probe_failure_auto_ban":               {},
	"probe_success_auto_enable":            {},
	"upstream_max_retries":                 {},
	"price_multiplier":                     {},
	"price_multiplier_mode":                {},
	"force_priority":                       {},
	"force_priority_scope":                 {},
	"name":                                 {},
	"weight":                               {},
	"models":                               {},
	"group":                                {},
	"model_mapping":                        {},
	"status_code_mapping":                  {},
	"auto_ban":                             {},
	"other_info":                           {},
	"tag":                                  {},
	"remark":                               {},
	"channel_info":                         {},
	"multi_key_mode":                       {},
}
