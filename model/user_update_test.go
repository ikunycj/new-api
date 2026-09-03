package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/glebarez/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserUpdateTestState(t *testing.T) {
	t.Helper()
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)

	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
	})
}

func TestUserUpdateDoesNotOverwriteAccountingFields(t *testing.T) {
	setupUserUpdateTestState(t)

	user := User{
		Id:           1,
		Username:     "quota-race-user",
		Password:     "password",
		DisplayName:  "before",
		Status:       common.UserStatusEnabled,
		Quota:        1000,
		UsedQuota:    20,
		RequestCount: 3,
		CreatedIP:    "198.51.100.1",
		LastLoginAt:  100,
		LastLoginIP:  "198.51.100.2",
		LastUsedIP:   "198.51.100.3",
	}
	require.NoError(t, DB.Create(&user).Error)

	staleUser, err := GetUserById(user.Id, true)
	require.NoError(t, err)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"quota":         gorm.Expr("quota - ?", 400),
		"used_quota":    gorm.Expr("used_quota + ?", 400),
		"request_count": gorm.Expr("request_count + ?", 1),
		"last_used_at":  1234567890,
	}).Error)

	staleUser.DisplayName = "after"
	staleUser.CreatedIP = "203.0.113.10"
	staleUser.LastLoginAt = 200
	staleUser.LastLoginIP = "203.0.113.11"
	staleUser.LastUsedIP = "203.0.113.12"
	require.NoError(t, staleUser.Update(false))

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "after", got.DisplayName)
	assert.Equal(t, 600, got.Quota)
	assert.Equal(t, 420, got.UsedQuota)
	assert.Equal(t, 4, got.RequestCount)
	assert.Equal(t, int64(1234567890), got.LastUsedAt)
	assert.Equal(t, "198.51.100.1", got.CreatedIP)
	assert.Equal(t, int64(100), got.LastLoginAt)
	assert.Equal(t, "198.51.100.2", got.LastLoginIP)
	assert.Equal(t, "198.51.100.3", got.LastUsedIP)
}

func TestUpdateUserUsedQuotaAndRequestCountRecordsLastUsedAt(t *testing.T) {
	setupUserUpdateTestState(t)

	user := User{
		Id:       3,
		Username: "api-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(&user).Error)

	beforeUpdate := common.GetTimestamp()
	UpdateUserUsedQuotaAndRequestCount(user.Id, 100)
	afterUpdate := common.GetTimestamp()

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, 100, got.UsedQuota)
	assert.Equal(t, 1, got.RequestCount)
	assert.GreaterOrEqual(t, got.LastUsedAt, beforeUpdate)
	assert.LessOrEqual(t, got.LastUsedAt, afterUpdate)
}

func TestUpdateUserLastLoginAtWithIPRecordsTimestampAndIP(t *testing.T) {
	setupUserUpdateTestState(t)

	user := User{
		Id:       4,
		Username: "login-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(&user).Error)

	beforeUpdate := common.GetTimestamp()
	UpdateUserLastLoginAtWithIP(user.Id, "198.51.100.42")
	afterUpdate := common.GetTimestamp()

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.GreaterOrEqual(t, got.LastLoginAt, beforeUpdate)
	assert.LessOrEqual(t, got.LastLoginAt, afterUpdate)
	assert.Equal(t, "198.51.100.42", got.LastLoginIP)
}

func TestFillUsersLastUsedAtUsesLatestConsumeLogAsFallback(t *testing.T) {
	setupUserUpdateTestState(t)

	previousLogDB := LOG_DB
	LOG_DB = DB
	t.Cleanup(func() {
		LOG_DB = previousLogDB
	})
	require.NoError(t, DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, DB.Create(&[]Log{
		{UserId: 10, Type: LogTypeConsume, CreatedAt: 100},
		{UserId: 10, Type: LogTypeConsume, CreatedAt: 300},
		{UserId: 10, Type: LogTypeError, CreatedAt: 400},
		{UserId: 11, Type: LogTypeConsume, CreatedAt: 500},
	}).Error)

	users := []*User{
		{Id: 10},
		{Id: 11, LastUsedAt: 450},
		{Id: 12},
	}
	require.NoError(t, FillUsersLastUsedAt(users))

	assert.Equal(t, int64(300), users[0].LastUsedAt)
	assert.Equal(t, int64(450), users[1].LastUsedAt)
	assert.Zero(t, users[2].LastUsedAt)
}

func TestUpdateUserSettingOnlyUpdatesSetting(t *testing.T) {
	setupUserUpdateTestState(t)

	user := User{
		Id:           2,
		Username:     "setting-user",
		Password:     "password",
		Status:       common.UserStatusEnabled,
		Quota:        1000,
		UsedQuota:    20,
		RequestCount: 3,
	}
	require.NoError(t, DB.Create(&user).Error)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"quota":         gorm.Expr("quota - ?", 250),
		"used_quota":    gorm.Expr("used_quota + ?", 250),
		"request_count": gorm.Expr("request_count + ?", 1),
	}).Error)

	require.NoError(t, UpdateUserSetting(user.Id, dto.UserSetting{Language: "zh"}))

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, 750, got.Quota)
	assert.Equal(t, 270, got.UsedQuota)
	assert.Equal(t, 4, got.RequestCount)
	assert.Equal(t, "zh", got.GetSetting().Language)
}

func TestEnsureEmailAvailableRejectsExistingEmailCaseInsensitive(t *testing.T) {
	setupUserUpdateTestState(t)

	require.NoError(t, DB.Create(&User{
		Username: "existing",
		Password: "old-password",
		Email:    "Taken@Example.com",
		Status:   common.UserStatusEnabled,
	}).Error)

	err := EnsureEmailAvailable(" taken@example.COM ", 0)
	require.ErrorIs(t, err, ErrEmailAlreadyTaken)

	user, err := GetUniqueUserByEmail("TAKEN@example.com")
	require.NoError(t, err)
	assert.Equal(t, "existing", user.Username)

	require.NoError(t, EnsureEmailAvailable("taken@example.com", user.Id))
}

func TestUsernameValidationAllowsValuesLongerThanLegacyLimit(t *testing.T) {
	username := strings.Repeat("u", 64)
	user := User{Username: username, Password: "password123"}
	require.NoError(t, common.Validate.Struct(&user))
}

func TestEditPersistsLongUsername(t *testing.T) {
	setupUserUpdateTestState(t)

	user := &User{Username: "short-name", Password: "password123", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)

	longUsername := strings.Repeat("long-username-", 8)
	user.Username = longUsername
	require.NoError(t, user.EditWithTx(DB, false))

	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, longUsername, stored.Username)
}

func TestInsertUsesEmailWhenUsernameIsBlank(t *testing.T) {
	setupUserUpdateTestState(t)

	user := &User{
		Email:  "fallback@example.com",
		Status: common.UserStatusEnabled,
	}
	require.NoError(t, user.Insert(0))
	assert.Equal(t, "fallback@example.com", user.Username)

	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, "fallback@example.com", stored.Username)
}

func TestEditAllowsDuplicateUsername(t *testing.T) {
	setupUserUpdateTestState(t)

	first := &User{Username: "duplicate-target", Password: "password123", Status: common.UserStatusEnabled}
	second := &User{Username: "editable-user", Password: "password123", Status: common.UserStatusEnabled}
	require.NoError(t, first.Insert(0))
	require.NoError(t, second.Insert(0))

	second.Username = first.Username
	err := second.EditWithTx(DB, false)
	require.NoError(t, err)

	var count int64
	require.NoError(t, DB.Model(&User{}).Where("username = ?", first.Username).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestMigrateUsernameToNonUniqueSQLite(t *testing.T) {
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

	require.NoError(t, DB.Exec("CREATE TABLE `users` (\n"+
		"`id` integer primary key,\n"+
		"`username` varchar(255) UNIQUE,\n"+
		"`email` varchar(255)\n)").Error)
	require.NoError(t, migrateUsernameToNonUnique())
	require.NoError(t, DB.Exec("INSERT INTO users (username) VALUES (?)", "same-name").Error)
	require.NoError(t, DB.Exec("INSERT INTO users (username) VALUES (?)", "same-name").Error)
}

func TestRemoveLegacyUserClassificationColumnSQLite(t *testing.T) {
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

	require.NoError(t, DB.Exec("CREATE TABLE `users` (`id` integer primary key, `email` varchar(255), `user_type` varchar(8))").Error)
	require.NoError(t, DB.Exec("CREATE INDEX `idx_users_user_type` ON `users` (`user_type`)").Error)
	require.True(t, DB.Migrator().HasColumn(&legacyUserClassificationColumn{}, "user_type"))
	require.NoError(t, removeLegacyUserClassificationColumn())
	assert.False(t, DB.Migrator().HasColumn(&legacyUserClassificationColumn{}, "user_type"))
	assert.False(t, DB.Migrator().HasIndex(&legacyUserClassificationColumn{}, "idx_users_user_type"))
}

func TestInsertRejectsDuplicateEmailWithoutUniqueIndex(t *testing.T) {
	setupUserUpdateTestState(t)

	require.NoError(t, DB.Create(&User{
		Username: "existing",
		Password: "old-password",
		Email:    "taken@example.com",
		Status:   common.UserStatusEnabled,
	}).Error)

	user := &User{
		Username: "oauth-user",
		Email:    "TAKEN@example.com",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}

	err := user.Insert(0)
	require.ErrorIs(t, err, ErrEmailAlreadyTaken)

	var count int64
	require.NoError(t, DB.Model(&User{}).Where("username = ?", "oauth-user").Count(&count).Error)
	assert.Zero(t, count)
}

func TestInsertKeepsBlankPasswordForPasswordlessUser(t *testing.T) {
	setupUserUpdateTestState(t)

	user := &User{
		Username: "passwordless-user",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}

	require.NoError(t, user.Insert(0))

	var stored User
	require.NoError(t, DB.Where("username = ?", user.Username).First(&stored).Error)
	assert.Empty(t, stored.Password)
}

func TestValidateAndFillRejectsPasswordlessUser(t *testing.T) {
	setupUserUpdateTestState(t)

	require.NoError(t, DB.Create(&User{
		Username: "passwordless-user",
		Password: "",
		Status:   common.UserStatusEnabled,
	}).Error)

	loginUser := User{
		Username: "passwordless-user",
		Password: "NewPassword123",
	}
	err := loginUser.ValidateAndFill()
	require.ErrorIs(t, err, ErrInvalidCredentials)

	var stored User
	require.NoError(t, DB.Where("username = ?", "passwordless-user").First(&stored).Error)
	assert.Empty(t, stored.Password)
}

func TestResetUserPasswordByEmailRequiresSingleActiveMatch(t *testing.T) {
	setupUserUpdateTestState(t)

	require.NoError(t, DB.Create(&User{
		Username: "duplicate-1",
		Password: "old-1",
		Email:    "legacy@example.com",
		AffCode:  "dupe1",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, DB.Create(&User{
		Username: "duplicate-2",
		Password: "old-2",
		Email:    "LEGACY@example.com",
		AffCode:  "dupe2",
		Status:   common.UserStatusEnabled,
	}).Error)

	err := ResetUserPasswordByEmail("legacy@example.com", "NewPassword123")
	require.ErrorIs(t, err, ErrEmailAmbiguous)

	var duplicates []User
	require.NoError(t, DB.Where("LOWER(email) = ?", "legacy@example.com").Order("username asc").Find(&duplicates).Error)
	require.Len(t, duplicates, 2)
	assert.Equal(t, "old-1", duplicates[0].Password)
	assert.Equal(t, "old-2", duplicates[1].Password)

	require.NoError(t, DB.Create(&User{
		Username: "unique",
		Password: "old",
		Email:    "unique@example.com",
		AffCode:  "unique",
		Status:   common.UserStatusEnabled,
	}).Error)

	require.NoError(t, ResetUserPasswordByEmail("UNIQUE@example.com", "NewPassword123"))

	var unique User
	require.NoError(t, DB.Where("username = ?", "unique").First(&unique).Error)
	assert.True(t, common.ValidatePasswordAndHash("NewPassword123", unique.Password))

	err = ResetUserPasswordByEmail("missing@example.com", "NewPassword123")
	require.True(t, errors.Is(err, ErrEmailNotFound))
}
