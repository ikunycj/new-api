package console_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateChannelErrorWhitelist(t *testing.T) {
	require.NoError(t, ValidateConsoleSettings(`[{"match_type":"status","match_value":"429","response_mode":"custom","custom_code":"rate_limited","custom_type":"rate_limit_error","custom_message":"请稍后重试"},{"match_type":"error","match_value":"quota","response_mode":"passthrough"}]`, "ChannelErrorWhitelist"))
	require.Error(t, ValidateConsoleSettings(`[{"match_type":"status","match_value":"200","response_mode":"passthrough"}]`, "ChannelErrorWhitelist"))
	require.Error(t, ValidateConsoleSettings(`[{"match_type":"error","match_value":"quota","response_mode":"unknown"}]`, "ChannelErrorWhitelist"))
	require.Error(t, ValidateConsoleSettings(`[{"match_type":"error","match_value":"quota","response_mode":"passthrough","custom_code":"bad"}]`, "ChannelErrorWhitelist"))
	require.Error(t, ValidateConsoleSettings(`[{"match_type":"error","match_value":"quota","response_mode":"passthrough"},{"match_type":"error","match_value":"quota","response_mode":"custom","custom_code":"duplicate","custom_type":"server_error","custom_message":"duplicate"}]`, "ChannelErrorWhitelist"))
}
