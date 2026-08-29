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
	require.NoError(t, testDB.Create(&Log{
		UserId:    7,
		TokenId:   9,
		ChannelId: 11,
		Group:     "toC",
		Type:      LogTypeConsume,
		CreatedAt: 1800,
		Quota:     125000,
		Other:     `{"cost_reconciliation":{"status":"estimated","user_charge_usd_micros":250000,"estimated_cost_usd_micros":200000,"successful_cost_usd_micros":200000}}`,
	}).Error)

	count, err := RebuildCostReconciliationRollup(0, 7200)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	count, err = RebuildCostReconciliationRollup(0, 7200)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	rows, total, totals, err := ListCostReconciliationRollups(CostReconciliationQuery{StartTimestamp: 0, EndTimestamp: 7200})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, int64(2), total)
	require.Equal(t, int64(3600), rows[0].BucketStart)
	require.Equal(t, int64(0), rows[1].BucketStart)
	require.Equal(t, int64(1250000), totals.UserChargeUSDMicros)
	require.Equal(t, int64(250000), totals.DiffUSDMicros)

	rows, total, _, err = ListCostReconciliationRollups(CostReconciliationQuery{StartTimestamp: 0, EndTimestamp: 7200, Keyword: "billing", TokenName: "cost-key"})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
}

func TestParseCostSnapshotReadsFailedPartialUsageCost(t *testing.T) {
	snapshot := parseCostSnapshot(`{"cost_reconciliation":{"status":"estimated","failed_partial_usage_cost_usd_micros":321}}`)

	require.Equal(t, int64(321), snapshot.FailedPartialCostUSDMicros)
}

func TestCostReconciliationGroupAliasIsCanonicalized(t *testing.T) {
	require.Equal(t, "通用套餐", canonicalCostReconciliationGroup("ChatGPT Plus"))
	require.Equal(t, "通用套餐", canonicalCostReconciliationGroup("通用套餐"))
	require.Equal(t, "成本套餐", canonicalCostReconciliationGroup("成本套餐"))
	aliases := costReconciliationGroupAliases("通用套餐")
	require.Contains(t, aliases, "通用套餐")
	require.Contains(t, aliases, "ChatGPT Plus")
	require.Contains(t, aliases, "GPT-PLUS")
	require.Contains(t, aliases, "gpt-image-2")
}
