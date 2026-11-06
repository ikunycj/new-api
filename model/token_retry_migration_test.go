package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRemoveLegacyTokenGroupRetryTimesColumnSQLite(t *testing.T) {
	legacyDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	previousDB := DB
	previousDatabaseType := common.MainDatabaseType()
	DB = legacyDB
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousDatabaseType)
	})

	require.NoError(t, DB.Exec("CREATE TABLE tokens (id integer primary key, group_retry_times text, group_candidates text)").Error)
	require.True(t, DB.Migrator().HasColumn(&Token{}, "group_retry_times"))

	require.NoError(t, removeLegacyTokenGroupRetryTimesColumn())
	assert.False(t, DB.Migrator().HasColumn(&Token{}, "group_retry_times"))

	// The migration is safe to run more than once after the column is gone.
	require.NoError(t, removeLegacyTokenGroupRetryTimesColumn())
}
