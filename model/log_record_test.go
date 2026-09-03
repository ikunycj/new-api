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
	setupPostgresAnalyticsTestDB(t, &Log{})

	previousLogConsumeEnabled := common.LogConsumeEnabled
	previousDataExportEnabled := common.DataExportEnabled
	common.LogConsumeEnabled = true
	common.DataExportEnabled = false
	t.Cleanup(func() {
		common.LogConsumeEnabled = previousLogConsumeEnabled
		common.DataExportEnabled = previousDataExportEnabled
	})

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
}
