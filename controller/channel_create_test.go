package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAddChannelRandomDelayDefaultPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL channel creation tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	schema := fmt.Sprintf("channel_create_%d", time.Now().UnixNano())
	require.NoError(t, db.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	t.Cleanup(func() { assert.NoError(t, db.Exec(`DROP SCHEMA "`+schema+`" CASCADE`).Error) })
	require.NoError(t, db.Exec(`SET search_path TO "`+schema+`"`).Error)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.User{}, &model.Log{}))

	previousDB, previousLogDB, previousRedis := model.DB, model.LOG_DB, common.RedisEnabled
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	t.Cleanup(func() { model.DB, model.LOG_DB, common.RedisEnabled = previousDB, previousLogDB, previousRedis })

	for _, tc := range []struct {
		name    string
		field   string
		enabled bool
	}{
		{name: "omitted", enabled: true},
		{name: "disabled", field: `,"probe_random_delay_enabled":false`, enabled: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"mode":"single","channel":{"name":%q,"type":1,"key":"test-key","models":"test-model","test_model":"test-model","group":"default"%s}}`, tc.name, tc.field)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/channel/", strings.NewReader(body))
			AddChannel(c)

			require.Equal(t, http.StatusOK, w.Code)
			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			require.True(t, response.Success, response.Message)
			var stored model.Channel
			require.NoError(t, db.Where("name = ?", tc.name).First(&stored).Error)
			assert.Equal(t, tc.enabled, stored.ProbeRandomDelayEnabled)
		})
	}
}
