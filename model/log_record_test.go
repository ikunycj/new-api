package model

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLogRecordTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.RemoteAddr = "198.51.100.24:43123"
	c.Set("username", "ip-test-user")
	return c
}

func TestRecordConsumeAndErrorLogsAlwaysRecordClientIP(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &User{}, &Log{})

	previousLogConsumeEnabled := common.LogConsumeEnabled
	previousDataExportEnabled := common.DataExportEnabled
	common.LogConsumeEnabled = true
	common.DataExportEnabled = false
	t.Cleanup(func() {
		common.LogConsumeEnabled = previousLogConsumeEnabled
		common.DataExportEnabled = previousDataExportEnabled
	})
	require.NoError(t, DB.Create(&[]User{
		{Id: 42, Username: "consume-user"},
		{Id: 43, Username: "error-user"},
	}).Error)

	consumeContext := newLogRecordTestContext()
	RecordConsumeLog(consumeContext, 42, RecordConsumeLogParams{
		ModelName: "test-model",
		Content:   "consume",
	})

	errorContext := newLogRecordTestContext()
	RecordErrorLog(errorContext, 43, 1, "test-model", "test-token", "error", 2, 1, false, "default", nil)

	var logs []Log
	require.NoError(t, LOG_DB.Where("user_id IN ?", []int{42, 43}).Order("user_id asc").Find(&logs).Error)
	require.Len(t, logs, 2)
	assert.Equal(t, LogTypeConsume, logs[0].Type)
	assert.Equal(t, "198.51.100.24", logs[0].Ip)
	assert.Equal(t, LogTypeError, logs[1].Type)
	assert.Equal(t, "198.51.100.24", logs[1].Ip)

	var user User
	require.NoError(t, DB.First(&user, 42).Error)
	assert.Equal(t, "198.51.100.24", user.LastUsedIP)
}

func TestFillUsersRecentIPsUsesLatestAuditRecords(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &User{}, &Log{})

	require.NoError(t, DB.Create(&[]User{
		{Id: 101, Username: "login-user"},
		{Id: 102, Username: "usage-user", LastLoginIP: "203.0.113.9"},
		{Id: 103, Username: "empty-user"},
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 101, Type: LogTypeLogin, CreatedAt: 100, Ip: "198.51.100.1"},
		{UserId: 101, Type: LogTypeLogin, CreatedAt: 200, Ip: "198.51.100.2"},
		{UserId: 101, Type: LogTypeConsume, CreatedAt: 300, Ip: "198.51.100.3"},
		{UserId: 102, Type: LogTypeLogin, CreatedAt: 400, Ip: "198.51.100.4"},
		{UserId: 102, Type: LogTypeConsume, CreatedAt: 500},
	}).Error)

	users := []*User{
		{Id: 101},
		{Id: 102, LastLoginIP: "203.0.113.9"},
		{Id: 103},
	}
	require.NoError(t, FillUsersRecentIPs(users))

	assert.Equal(t, "198.51.100.2", users[0].LastLoginIP)
	assert.Equal(t, "198.51.100.3", users[0].LastUsedIP)
	assert.Equal(t, "203.0.113.9", users[1].LastLoginIP)
	assert.Empty(t, users[1].LastUsedIP)
	assert.Empty(t, users[2].LastLoginIP)
	assert.Empty(t, users[2].LastUsedIP)
}
