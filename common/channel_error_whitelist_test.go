package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicChannelErrorWhitelistMatchesStatusAndAnyErrorField(t *testing.T) {
	OptionMapRWMutex.Lock()
	previous := OptionMap
	OptionMap = map[string]string{
		ChannelErrorWhitelistOptionKey: `[{"match_type":"status","match_value":"429","response_mode":"custom","custom_code":"rate_limited","custom_type":"rate_limit_error","custom_message":"请求过于频繁"},{"match_type":"error","match_value":"quota","response_mode":"custom","custom_code":"quota_error","custom_type":"server_error","custom_message":"额度暂不可用"},{"match_type":"error","match_value":"invalid_request_error","response_mode":"passthrough"}]`,
	}
	OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		OptionMapRWMutex.Lock()
		OptionMap = previous
		OptionMapRWMutex.Unlock()
	})

	fields, matched := PublicChannelErrorFields(429, "provider_rate_limit", "upstream_error", "private supplier text")
	require.True(t, matched)
	assert.Equal(t, PublicErrorFields{Code: "rate_limited", Type: "rate_limit_error", Message: "请求过于频繁"}, fields)

	fields, matched = PublicChannelErrorFields(400, "insufficient_quota", "upstream_error", "private supplier text")
	require.True(t, matched)
	assert.Equal(t, PublicErrorFields{Code: "quota_error", Type: "server_error", Message: "额度暂不可用"}, fields)

	fields, matched = PublicChannelErrorFields(400, "other", "invalid_request_error", "private supplier text")
	require.True(t, matched)
	assert.Equal(t, "other", fields.Code)
	assert.Equal(t, "invalid_request_error", fields.Type)
	assert.Equal(t, "private supplier text", fields.Message)

	fields, matched = PublicChannelErrorFields(502, "other", "upstream_error", "private supplier text")
	require.False(t, matched)
	assert.Equal(t, PublicErrorCode, fields.Code)
	assert.Equal(t, "server_error", fields.Type)
	assert.Contains(t, fields.Message, "服务暂异常")
}

func TestPublicChannelErrorLegacyProjectionUsesFuzzyErrorMatching(t *testing.T) {
	OptionMapRWMutex.Lock()
	previous := OptionMap
	OptionMap = map[string]string{
		ChannelErrorWhitelistOptionKey: `[{"match_type":"error","match_value":"approved","response_mode":"passthrough"}]`,
	}
	OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		OptionMapRWMutex.Lock()
		OptionMap = previous
		OptionMapRWMutex.Unlock()
	})

	code, message := PublicChannelError("other", "approved error")
	assert.Equal(t, "other", code)
	assert.Equal(t, "approved error", message)
	code, message = PublicChannelError("other", "private supplier text")
	require.Equal(t, PublicErrorCode, code)
	assert.Contains(t, message, "服务暂异常")
}
