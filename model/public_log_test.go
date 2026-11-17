package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserLogProjectionKeepsInternalRecordAndAccounting(t *testing.T) {
	original := &Log{
		Id: 88, ChannelId: 23, ChannelName: "private-supplier", UpstreamRequestId: "upstream-private",
		Type: LogTypeConsume, Content: "正常消费", RequestId: "req_public", Quota: 250,
		Other: `{"admin_info":{"use_channel":[23,24]},"channel_name":"private-supplier","error_ref":"CH23","upstream_model_name":"secret-model","po":["replace auth"],"request_conversion":["Claude"],"model_price":0.1,"group_ratio":2,"cache_tokens":35,"subscription_consumed":250}`,
	}
	before := *original
	logs := []*Log{original}
	formatUserLogs(logs, 0)
	assert.Equal(t, before, *original, "internal/admin record must remain untouched")
	assert.Equal(t, 1, logs[0].ChannelId)
	assert.Equal(t, "auto", logs[0].ChannelName)
	assert.Empty(t, logs[0].UpstreamRequestId)
	assert.Equal(t, "req_public", logs[0].RequestId)
	assert.Equal(t, "正常消费", logs[0].Content)
	assert.Equal(t, 250, logs[0].Quota)
	assert.JSONEq(t, `{"model_price":0.1,"group_ratio":2,"cache_tokens":35,"subscription_consumed":250}`, logs[0].Other)
}

func TestUserLogProjectionHidesHistoricalFailureAndRefund(t *testing.T) {
	for _, kind := range []int{LogTypeError, LogTypeRefund} {
		original := &Log{ChannelId: 9, Type: kind, Content: "supplier pool exhausted", Other: `{"task_id":"task_public","reason":"supplier pool exhausted"}`}
		logs := []*Log{original}
		formatUserLogs(logs, 0)
		assert.Equal(t, common.DefaultPublicChannelErrorMessage(), logs[0].Content)
		assert.NotContains(t, logs[0].Other, "supplier pool exhausted")
		assert.Contains(t, logs[0].Other, "task_public")
		assert.Equal(t, "supplier pool exhausted", original.Content)
	}
}

func TestUserLogProjectionHonorsChannelErrorWhitelist(t *testing.T) {
	const rawMessage = "status_code=429, All available accounts are currently rate-limited. Please retry later."
	const errorMetadata = `{"status_code":429,"error_code":"unknown_error","error_type":"openai_error","request_path":"/v1/responses","admin_info":{"use_channel":["73"]}}`
	const passthrough429 = `[{"match_type":"status","match_value":"429","response_mode":"passthrough"}]`

	common.OptionMapRWMutex.Lock()
	previousOptions := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
	})

	for _, tc := range []struct {
		name        string
		whitelist   string
		metadata    string
		wantCode    string
		wantMessage string
	}{
		{
			name: "status passthrough", whitelist: passthrough429, metadata: errorMetadata,
			wantCode: "unknown_error", wantMessage: rawMessage,
		},
		{
			name:      "status custom response",
			whitelist: `[{"match_type":"status","match_value":"429","response_mode":"custom","custom_code":"rate_limited","custom_message":"Please retry later."}]`,
			metadata:  errorMetadata, wantCode: "rate_limited", wantMessage: "Please retry later.",
		},
		{
			name:      "error type passthrough",
			whitelist: `[{"match_type":"error","match_value":"openai_error","response_mode":"passthrough"}]`,
			metadata:  errorMetadata, wantCode: "unknown_error", wantMessage: rawMessage,
		},
		{
			name:      "unmatched status stays private",
			whitelist: `[{"match_type":"status","match_value":"401","response_mode":"passthrough"}]`,
			metadata:  errorMetadata, wantCode: common.PublicErrorCode, wantMessage: common.DefaultPublicChannelErrorMessage(),
		},
		{
			name: "missing status is not inferred from content", whitelist: passthrough429,
			metadata: `{"error_code":"unknown_error","request_path":"/v1/responses"}`,
			wantCode: common.PublicErrorCode, wantMessage: common.DefaultPublicChannelErrorMessage(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			common.OptionMapRWMutex.Lock()
			common.OptionMap[common.ChannelErrorWhitelistOptionKey] = tc.whitelist
			common.OptionMapRWMutex.Unlock()

			original := &Log{
				Id: 88, Type: LogTypeError, ChannelId: 73, ChannelName: "private-supplier",
				RequestId: "req_public_429", UpstreamRequestId: "upstream-private",
				Content: rawMessage, Other: tc.metadata,
			}
			before := *original
			logs := []*Log{original}
			formatUserLogs(logs, 0)

			assert.Equal(t, tc.wantMessage, logs[0].Content)
			var publicOther map[string]interface{}
			require.NoError(t, common.UnmarshalJsonStr(logs[0].Other, &publicOther))
			assert.Equal(t, map[string]interface{}{
				"error_code": tc.wantCode, "request_path": "/v1/responses",
			}, publicOther)
			assert.Equal(t, common.PublicChannelID, logs[0].ChannelId)
			assert.Equal(t, common.PublicChannelName, logs[0].ChannelName)
			assert.Empty(t, logs[0].UpstreamRequestId)
			assert.Equal(t, original.RequestId, logs[0].RequestId)
			assert.Equal(t, before, *original, "internal/admin record must remain untouched")
		})
	}
}

func TestPublicLogSearchDoesNotProbeChannelOrUpstreamIDPostgres(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &User{}, &Log{})
	require.NoError(t, LOG_DB.Create(&Log{UserId: 7, ChannelId: 923, UpstreamRequestId: "upstream-secret", RequestId: "req_public", Type: LogTypeConsume}).Error)
	for _, keyword := range []string{"923", "upstream-secret"} {
		logs, total, err := GetUserLogs(7, LogTypeUnknown, 0, 0, "", keyword, "", 0, 10, "", "", "")
		require.NoError(t, err)
		assert.Empty(t, logs)
		assert.Zero(t, total)
	}
	logs, total, err := GetUserLogs(7, LogTypeUnknown, 0, 0, "", "req_public", "", 0, 10, "", "", "")
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, 1, logs[0].ChannelId)
	var stored Log
	require.NoError(t, LOG_DB.First(&stored).Error)
	assert.Equal(t, 923, stored.ChannelId)
	assert.Equal(t, "upstream-secret", stored.UpstreamRequestId)
	_, _, err = GetUserLogs(7, LogTypeUnknown, 0, 0, "", "", "", 0, 10, "", "", "upstream-secret")
	require.Error(t, err)
}

func TestPublicLogSearchDoesNotProbeChannelOrUpstreamIDSQLite(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&User{}, &Log{}))

	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	DB, LOG_DB = testDB, testDB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		initCol()
	})

	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 7, ChannelId: 923, ChannelName: "private-supplier",
		UpstreamRequestId: "upstream-secret", RequestId: "req_public", Type: LogTypeConsume,
	}).Error)

	for _, keyword := range []string{"923", "upstream-secret"} {
		logs, total, err := GetUserLogs(7, LogTypeUnknown, 0, 0, "", keyword, "", 0, 10, "", "", "")
		require.NoError(t, err)
		assert.Empty(t, logs, "public log keyword search must not match %q", keyword)
		assert.Zero(t, total)
	}

	logs, total, err := GetUserLogs(7, LogTypeUnknown, 0, 0, "", "req_public", "", 0, 10, "", "", "")
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, common.PublicChannelID, logs[0].ChannelId)
	assert.Equal(t, common.PublicChannelName, logs[0].ChannelName)

	var stored Log
	require.NoError(t, LOG_DB.First(&stored).Error)
	assert.Equal(t, 923, stored.ChannelId)
	assert.Equal(t, "upstream-secret", stored.UpstreamRequestId)

	_, _, err = GetUserLogs(7, LogTypeUnknown, 0, 0, "", "", "", 0, 10, "", "", "upstream-secret")
	require.Error(t, err)
}
