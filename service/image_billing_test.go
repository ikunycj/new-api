package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupImageBillingPostgres(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL image billing tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	previousDB, previousLogs := model.DB, model.LOG_DB
	mainType, logType := common.MainDatabaseType(), common.LogDatabaseType()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogs
		common.SetDatabaseTypes(mainType, logType)
		require.NoError(t, tx.Rollback().Error)
		require.NoError(t, sqlDB.Close())
	})
	schema := fmt.Sprintf("image_billing_test_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	require.NoError(t, tx.AutoMigrate(&model.User{}))
	model.DB, model.LOG_DB = tx, tx
	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
}

func TestImageBillingReservesOverridesAndSettlesActualCountPostgres(t *testing.T) {
	for index, test := range []struct {
		name   string
		budget int
		allow  bool
	}{
		{"sufficient quota", 10000, true},
		{"insufficient quota", 3000, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupImageBillingPostgres(t)
			userID := 730001 + index
			require.NoError(t, model.DB.Create(&model.User{Id: userID, Username: "image-billing-user", Quota: test.budget, Status: common.UserStatusEnabled}).Error)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			ctx.Set("id", userID)
			ctx.Set("role", common.RoleCommonUser)
			const expression = `tier("image", fixed(0.0035)) * image_count`
			info := &relaycommon.RelayInfo{
				UserId: userID, IsPlayground: true, ForcePreConsume: true,
				UserSetting: dto.UserSetting{BillingPreference: "wallet_only"},
				TieredBillingSnapshot: &billingexpr.BillingSnapshot{
					BillingMode: "tiered_expr", ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
					GroupRatio: 1, QuotaPerUnit: 500000, BillingUSDToCNYRate: 1,
					EstimatedImageCount: common.GetPointer(1), EstimatedQuotaAfterGroup: 1750,
					EstimatedBillingUnit: billingexpr.BillingUnitRequest, EstimatedFixedPrice: common.GetPointer(0.0035),
				},
				PriceData: types.PriceData{GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}},
			}
			require.Nil(t, PreConsumeBilling(ctx, 1750, info))
			apiErr := PrepareImageBillingForRequest(ctx, info, 3)
			if !test.allow {
				require.NotNil(t, apiErr)
				assert.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())
				assert.Equal(t, 1750, info.Billing.GetPreConsumedQuota())
				quota, err := model.GetUserQuota(userID, true)
				require.NoError(t, err)
				assert.Equal(t, 1250, quota)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, 5250, info.Billing.GetPreConsumedQuota())
			info.SetBillingImageCount(2)
			ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{})
			require.True(t, ok)
			require.NotNil(t, result)
			assert.Equal(t, 3500, quota)
			require.NoError(t, info.Billing.Settle(quota))
			remaining, err := model.GetUserQuota(userID, true)
			require.NoError(t, err)
			assert.Equal(t, 6500, remaining)
		})
	}
}

func TestImageBillingRejectsInvalidQuantityBeforeReservation(t *testing.T) {
	for _, count := range []int{0, -1, dto.MaxImageN + 1} {
		info := &relaycommon.RelayInfo{}
		err := PrepareImageBillingForRequest(nil, info, count)
		require.NotNil(t, err)
		assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
		assert.Nil(t, info.Billing)
	}
}
