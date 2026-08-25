package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRebuildCostReconciliationRollupIsIdempotent(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
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
	require.NoError(t, testDB.AutoMigrate(&User{}, &Token{}, &Log{}, &CostReconciliationRollup{}))
	require.NoError(t, testDB.Create(&User{Id: 7, Username: "billing-user", Password: "password123"}).Error)
	require.NoError(t, testDB.Create(&Token{Id: 9, UserId: 7, Name: "cost-key", Key: "cost-key-secret"}).Error)
	require.NoError(t, testDB.Create(&Log{
		UserId:    7,
		TokenId:   9,
		ChannelId: 11,
		Group:     "toC",
		Type:      LogTypeConsume,
		CreatedAt: 3600,
		Quota:     500000,
		Other:     `{"cost_reconciliation":{"status":"estimated","user_charge_usd_micros":1000000,"estimated_cost_usd_micros":800000,"successful_cost_usd_micros":800000}}`,
	}).Error)

	count, err := RebuildCostReconciliationRollup(0, 7200)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	count, err = RebuildCostReconciliationRollup(0, 7200)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	rows, total, totals, err := ListCostReconciliationRollups(CostReconciliationQuery{StartTimestamp: 0, EndTimestamp: 7200})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, int64(1000000), totals.UserChargeUSDMicros)
	require.Equal(t, int64(200000), totals.DiffUSDMicros)

	rows, total, _, err = ListCostReconciliationRollups(CostReconciliationQuery{StartTimestamp: 0, EndTimestamp: 7200, Keyword: "billing", TokenName: "cost-key"})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
}
