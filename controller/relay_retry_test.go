package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withOrderedRoutingBillingSettings(t *testing.T) {
	t.Helper()
	savedGroupRatios := ratio_setting.GroupRatio2JSONString()
	savedModelPrices := ratio_setting.ModelPrice2JSONString()
	savedFreePreConsume := operation_setting.GetQuotaSetting().EnableFreeModelPreConsume
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(savedGroupRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedModelPrices))
		operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = savedFreePreConsume
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"account":1,"free":0,"cheap":0.1,"paid":0.2,"expensive":0.4}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"ordered-routing-price":0.001}`))
	operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = false
}

func newOrderedRoutingBillingInfo(userID int) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:          userID,
		UserGroup:       "account",
		OriginModelName: "ordered-routing-price",
		IsPlayground:    true,
		ForcePreConsume: true,
		UserSetting:     dto.UserSetting{BillingPreference: "wallet_only"},
	}
}

func TestCancellationPhase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	before, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(t, "before_upstream", cancellationPhase(before, false))
	assert.Equal(t, "upstream", cancellationPhase(before, true))

	response, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, err := response.Writer.Write([]byte("partial response"))
	require.NoError(t, err)
	assert.Equal(t, "response", cancellationPhase(response, true))
}

func TestRelayRetryCommittedStopsAfterStreamingOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{IsStream: true}

	assert.False(t, relayRetryCommitted(ctx, info))
	_, err := ctx.Writer.Write([]byte("data: first\n\n"))
	require.NoError(t, err)
	assert.True(t, relayRetryCommitted(ctx, info))
}

func TestRelayRetryCommittedStopsOnClientCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestContext, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest("POST", "/v1/chat/completions", nil).WithContext(requestContext)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = request
	cancel()

	assert.True(t, relayRetryCommitted(ctx, &relaycommon.RelayInfo{}))
}

func TestShouldRetryUpstreamSourcesAcrossHTTP4xxAnd5xx(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		source     types.ErrorSource
		statusCode int
	}{
		{name: "openai bad request", source: types.ErrorSourceOpenAI, statusCode: http.StatusBadRequest},
		{name: "openai unauthorized", source: types.ErrorSourceOpenAI, statusCode: http.StatusUnauthorized},
		{name: "ikun server error", source: types.ErrorSourceIkun, statusCode: http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			err := types.WithOpenAIError(types.OpenAIError{
				Message: "upstream failure",
				Code:    "upstream_code",
				Source:  tt.source,
			}, tt.statusCode)
			require.True(t, shouldRetry(ctx, err, 1))
		})
	}
}

func TestShouldRetryHonorsConfiguredNonRetryableMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "vendor rejected request",
		Code:    "vendor_policy_error",
		Source:  types.ErrorSourceCluster,
	}, http.StatusBadGateway)
	err.SetClassification(207001, "policy", "request", "none", false)

	assert.False(t, shouldRetry(ctx, err, 1))
	assert.False(t, isFailoverEligible(ctx, err))
}

func TestShouldRetryDoesNotFailoverAllTokenClientErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.NewErrorWithStatusCode(errors.New("invalid request"), types.ErrorCodeInvalidRequest, http.StatusBadRequest)

	assert.False(t, shouldRetry(ctx, err, 1))
	assert.False(t, isFailoverEligible(ctx, err))
}

func TestShouldRetryRespectsSpecificChannelSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("specific_channel_id", 42)
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "upstream failure",
		Code:    "server_error",
		Source:  types.ErrorSourceIkun,
	}, http.StatusBadGateway)

	assert.False(t, shouldRetry(ctx, err, 1))
	assert.False(t, isFailoverEligible(ctx, err))
}

func TestReserveRelayGroupBillingCreatesSessionForFreeToPaidFallback(t *testing.T) {
	withOrderedRoutingBillingSettings(t)
	savedBillingRate := operation_setting.BillingUSDToCNYRate
	t.Cleanup(func() { operation_setting.BillingUSDToCNYRate = savedBillingRate })
	operation_setting.BillingUSDToCNYRate = 7.3
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       40001,
		Username: "routing-free-paid",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
	}).Error)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := newOrderedRoutingBillingInfo(40001)
	info.UsingGroup = "free"
	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "free")
	priceData, err := helper.ModelPriceHelper(ctx, info, 0, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.True(t, priceData.FreeModel)
	require.Equal(t, 7.3, priceData.BillingUSDToCNYRate)
	require.Nil(t, info.Billing)

	operation_setting.BillingUSDToCNYRate = 99
	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "paid")
	info.UsingGroup = "paid"
	apiErr := reserveRelayGroupBilling(ctx, info, 0, &types.TokenCountMeta{})

	require.Nil(t, apiErr)
	require.NotNil(t, info.Billing)
	assert.Equal(t, 730, info.Billing.GetPreConsumedQuota())
	assert.Equal(t, 7.3, info.PriceData.BillingUSDToCNYRate)
	userQuota, err := model.GetUserQuota(40001, true)
	require.NoError(t, err)
	assert.Equal(t, 270, userQuota)
}

func TestReserveRelayGroupBillingRejectsUnaffordableHigherFallback(t *testing.T) {
	withOrderedRoutingBillingSettings(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       40002,
		Username: "routing-insufficient",
		Status:   common.UserStatusEnabled,
		Quota:    150,
	}).Error)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := newOrderedRoutingBillingInfo(40002)
	info.UsingGroup = "cheap"
	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "cheap")
	require.Nil(t, reserveRelayGroupBilling(ctx, info, 0, &types.TokenCountMeta{}))
	require.NotNil(t, info.Billing)
	assert.Equal(t, 50, info.Billing.GetPreConsumedQuota())

	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "expensive")
	info.UsingGroup = "expensive"
	apiErr := reserveRelayGroupBilling(ctx, info, 0, &types.TokenCountMeta{})

	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())
	assert.Equal(t, 50, info.Billing.GetPreConsumedQuota())
	userQuota, err := model.GetUserQuota(40002, true)
	require.NoError(t, err)
	assert.Equal(t, 100, userQuota)
}

func TestReserveRelayGroupBillingRefreshesTieredSnapshotOnFallback(t *testing.T) {
	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "billing_setting.") {
			savedConfig[key] = value
		}
		return nil
	}))
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig)) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"ordered-routing-tiered":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"ordered-routing-tiered":"tier(\"base\", p * 2 + c * 4)"}`,
	}))
	withOrderedRoutingBillingSettings(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       40003,
		Username: "routing-tiered",
		Status:   common.UserStatusEnabled,
		Quota:    100000,
	}).Error)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := newOrderedRoutingBillingInfo(40003)
	info.OriginModelName = "ordered-routing-tiered"
	info.UsingGroup = "cheap"
	info.BillingRequestInput = &billingexpr.RequestInput{Body: []byte(`{}`)}
	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "cheap")
	require.Nil(t, reserveRelayGroupBilling(ctx, info, 1000, &types.TokenCountMeta{}))
	require.NotNil(t, info.TieredBillingSnapshot)
	assert.InDelta(t, 0.1, info.TieredBillingSnapshot.GroupRatio, 0.000001)
	cheapQuota := info.Billing.GetPreConsumedQuota()

	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "expensive")
	info.UsingGroup = "expensive"
	require.Nil(t, reserveRelayGroupBilling(ctx, info, 1000, &types.TokenCountMeta{}))
	require.NotNil(t, info.TieredBillingSnapshot)
	assert.InDelta(t, 0.4, info.TieredBillingSnapshot.GroupRatio, 0.000001)
	assert.Greater(t, info.Billing.GetPreConsumedQuota(), cheapQuota)
}
