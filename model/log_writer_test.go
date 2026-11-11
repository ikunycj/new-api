package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRequestLogWriterTest(t *testing.T) {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	// Every connection to ":memory:" is a separate database; the writer's
	// workers must see the same table as the test.
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, testDB.AutoMigrate(&Log{}))

	previousLogDB, previousAsync := LOG_DB, common.RequestLogAsyncEnabled
	LOG_DB = testDB
	t.Cleanup(func() {
		StopRequestLogWriter()
		LOG_DB = previousLogDB
		common.RequestLogAsyncEnabled = previousAsync
	})
}

func countRequestLogs(t *testing.T, requestIds ...string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, LOG_DB.Model(&Log{}).Where("request_id IN ?", requestIds).Count(&count).Error)
	return count
}

func TestSubmitRequestLogWritesSynchronouslyWhenSwitchIsOff(t *testing.T) {
	setupRequestLogWriterTest(t)
	common.RequestLogAsyncEnabled = false
	StartRequestLogWriter()

	require.NoError(t, submitRequestLog(&Log{RequestId: "sync-1", Type: LogTypeConsume, Quota: 10}))

	// The row must be committed before submit returns, exactly like the old path.
	assert.Equal(t, int64(1), countRequestLogs(t, "sync-1"))
}

func TestSubmitRequestLogQueuedRowsAreFlushedOnStop(t *testing.T) {
	setupRequestLogWriterTest(t)
	common.RequestLogAsyncEnabled = true
	StartRequestLogWriter()

	ids := []string{"async-1", "async-2", "async-3"}
	for _, id := range ids {
		require.NoError(t, submitRequestLog(&Log{RequestId: id, Type: LogTypeConsume, Quota: 10}))
	}
	StopRequestLogWriter()

	assert.Equal(t, int64(len(ids)), countRequestLogs(t, ids...))
}

func TestSubmitRequestLogSwitchingOffKeepsQueuedRowsAndWritesNewOnesSynchronously(t *testing.T) {
	setupRequestLogWriterTest(t)
	common.RequestLogAsyncEnabled = true
	StartRequestLogWriter()
	require.NoError(t, submitRequestLog(&Log{RequestId: "queued-1", Type: LogTypeConsume}))

	common.RequestLogAsyncEnabled = false
	require.NoError(t, submitRequestLog(&Log{RequestId: "after-off-1", Type: LogTypeConsume}))
	assert.Equal(t, int64(1), countRequestLogs(t, "after-off-1"))

	StopRequestLogWriter()
	assert.Equal(t, int64(2), countRequestLogs(t, "queued-1", "after-off-1"))
}
