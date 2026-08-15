package types

import (
	"fmt"
	"strings"
)

type errorDefinition struct {
	Code         int
	Category     string
	FailureScope string
	Action       string
}

func classifyError(err *NewAPIError) errorDefinition {
	if err == nil {
		return errorDefinition{}
	}
	if err.classification != nil {
		return *err.classification
	}
	rawCode := strings.ToLower(strings.TrimSpace(string(err.errorCode)))
	source := err.GetErrorSource()
	switch source {
	case ErrorSourceOpenAI:
		return classifyOpenAIError(rawCode, err.StatusCode)
	case ErrorSourceCluster:
		return classifyClusterError(rawCode, err.StatusCode)
	default:
		return classifyAlltokenError(rawCode)
	}
}

func classifyOpenAIError(rawCode string, statusCode int) errorDefinition {
	switch {
	case strings.Contains(rawCode, "invalid_api_key"):
		return errorDefinition{102001, "auth", "credential", "failover"}
	case strings.Contains(rawCode, "unsupported_country") || strings.Contains(rawCode, "unsupported_region"):
		return errorDefinition{102002, "auth", "provider", "failover"}
	case strings.Contains(rawCode, "insufficient_quota") || strings.Contains(rawCode, "credit_balance"):
		return errorDefinition{103001, "quota", "credential", "failover"}
	case strings.Contains(rawCode, "rate_limit") || statusCode == 429:
		return errorDefinition{104001, "rate_limit", "cluster", "failover"}
	case statusCode == 503:
		return errorDefinition{105002, "upstream", "provider", "failover"}
	case statusCode >= 500:
		return errorDefinition{105001, "upstream", "provider", "failover"}
	default:
		return errorDefinition{100001, "unknown", "cluster", "failover"}
	}
}

func classifyClusterError(rawCode string, statusCode int) errorDefinition {
	switch {
	case strings.Contains(rawCode, "invalid_api_key") || strings.Contains(rawCode, "invalid_credential"):
		return errorDefinition{202001, "auth", "credential", "failover"}
	case strings.Contains(rawCode, "rate_limit") || statusCode == 429:
		return errorDefinition{204001, "rate_limit", "cluster", "failover"}
	case strings.Contains(rawCode, "all_pools_exhausted"):
		return errorDefinition{205003, "upstream", "cluster", "failover"}
	case strings.Contains(rawCode, "pool_exhausted"):
		return errorDefinition{205004, "upstream", "channel", "failover"}
	case strings.Contains(rawCode, "no_healthy_account"):
		return errorDefinition{205001, "upstream", "cluster", "failover"}
	case strings.Contains(rawCode, "timeout"):
		return errorDefinition{210001, "network", "cluster", "failover"}
	case statusCode >= 500:
		return errorDefinition{205002, "upstream", "cluster", "failover"}
	default:
		return errorDefinition{200001, "unknown", "cluster", "failover"}
	}
}

func classifyAlltokenError(rawCode string) errorDefinition {
	definitions := map[string]errorDefinition{
		"invalid_request":                 {301001, "request", "request", "none"},
		"bad_request_body":                {301002, "request", "request", "none"},
		"invalid_api_type":                {301003, "request", "request", "none"},
		"read_request_body_failed":        {301004, "request", "request", "none"},
		"access_denied":                   {302001, "auth", "request", "none"},
		"insufficient_user_quota":         {303001, "quota", "request", "none"},
		"pre_consume_token_quota_failed":  {303002, "quota", "request", "none"},
		"upstream_exhausted":              {305001, "upstream", "cluster", "retry_later"},
		"bad_response_status_code":        {305002, "upstream", "channel", "failover"},
		"bad_response":                    {305003, "upstream", "channel", "failover"},
		"empty_response":                  {305004, "upstream", "channel", "failover"},
		"aws_invoke_error":                {305005, "upstream", "channel", "failover"},
		"channel:no_available_key":        {306001, "channel", "channel", "failover"},
		"channel:param_override_invalid":  {306002, "channel", "channel", "failover"},
		"channel:header_override_invalid": {306003, "channel", "channel", "failover"},
		"channel:model_mapped_error":      {306004, "channel", "channel", "failover"},
		"channel:aws_client_error":        {306005, "channel", "channel", "failover"},
		"channel:invalid_key":             {306006, "channel", "channel", "failover"},
		"channel:response_time_exceeded":  {306007, "channel", "channel", "failover"},
		"sensitive_words_detected":        {307001, "policy", "request", "none"},
		"violation_fee.grok.csam":         {307002, "policy", "request", "none"},
		"prompt_blocked":                  {307003, "policy", "request", "manual"},
		"convert_request_failed":          {308001, "protocol", "request", "none"},
		"json_marshal_failed":             {308002, "protocol", "request", "none"},
		"bad_response_body":               {308003, "protocol", "channel", "failover"},
		"query_data_error":                {309001, "internal", "request", "none"},
		"update_data_error":               {309002, "internal", "request", "none"},
		"count_token_failed":              {309003, "internal", "request", "none"},
		"model_price_error":               {309004, "internal", "request", "none"},
		"get_channel_failed":              {309005, "internal", "request", "retry_later"},
		"gen_relay_info_failed":           {309006, "internal", "request", "none"},
		"do_request_failed":               {310001, "network", "channel", "failover"},
		"read_response_body_failed":       {310002, "network", "channel", "failover"},
		"model_not_found":                 {311001, "model", "channel", "failover"},
	}
	if definition, ok := definitions[rawCode]; ok {
		return definition
	}
	return errorDefinition{300001, "unknown", "request", "none"}
}

func buildErrorRef(code int, clusterCode int, poolTier int) string {
	if code <= 0 {
		return ""
	}
	ref := fmt.Sprintf("%06d", code)
	if clusterCode > 0 {
		ref += fmt.Sprintf("-C%d", clusterCode)
	}
	if poolTier >= 1 && poolTier <= 3 {
		ref += fmt.Sprintf("-P%d", poolTier)
	}
	return ref
}
