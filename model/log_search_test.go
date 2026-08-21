package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLogKeywordSearchMatchesUserEmailAndRemark(t *testing.T) {
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

	now := time.Now().Unix()
	require.NoError(t, DB.Create(&User{
		Id:       7,
		Username: "alice",
		Password: "password123",
		Email:    "alice@example.com",
		Remark:   "Finance owner",
	}).Error)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 7, Username: "alice", Type: LogTypeConsume, CreatedAt: now, Quota: 42},
		{UserId: 8, Username: "bob", Type: LogTypeConsume, CreatedAt: now, Quota: 100},
	}).Error)

	logs, total, err := GetAllLogs(LogTypeUnknown, now-1, now+1, "", "finance", "", 0, 20, 0, "", "", "")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, logs, 1)
	assert.Equal(t, 7, logs[0].UserId)

	stat, err := SumUsedQuota(LogTypeConsume, now-1, now+1, "", "example.com", "", 0, "", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 42, stat.Quota)
}
