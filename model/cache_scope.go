package model

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/common"
)

// cacheDatabaseScope prevents snapshots from one database instance being
// served to another instance when they share a Redis server. The raw DSNs are
// hashed so credentials never become part of a Redis key.
func cacheDatabaseScope() string {
	if !common.RedisEnabled || common.RDB == nil {
		return fmt.Sprintf("db-%p", LOG_DB)
	}
	if configured := os.Getenv("CACHE_NAMESPACE_SCOPE"); configured != "" {
		digest := sha256.Sum256([]byte(configured))
		return fmt.Sprintf("%x", digest[:8])
	}
	mainDSN := os.Getenv("SQL_DSN")
	if mainDSN == "" {
		mainDSN = fmt.Sprintf("sqlite=%s;db=%p", common.SQLitePath, DB)
	}
	logDSN := os.Getenv("LOG_SQL_DSN")
	if logDSN == "" {
		logDSN = fmt.Sprintf("log-db=%p;%s", LOG_DB, mainDSN)
	}
	input := fmt.Sprintf("main=%s\x00log=%s\x00main-type=%s\x00log-type=%s", mainDSN, logDSN, common.MainDatabaseType(), common.LogDatabaseType())
	digest := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", digest[:8])
}
