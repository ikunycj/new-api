package controller

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSelfChannelDisplayURLOnlyChangesToBView(t *testing.T) {
	baseURL := "https://upstream.example.com/v1"
	displayURL := "https://customer.example.com/api"

	t.Run("configured display URL", func(t *testing.T) {
		settings, err := json.Marshal(map[string]string{"tob_display_url": displayURL})
		require.NoError(t, err)
		view := selfChannelView{
			BaseURL:  &baseURL,
			Settings: string(settings),
		}

		got := selfChannelDisplayURL(view)
		require.NotNil(t, got)
		require.Equal(t, displayURL, *got)
		require.Equal(t, baseURL, *view.BaseURL, "the view helper must not mutate the real channel URL")
	})

	t.Run("empty or invalid display URL falls back to upstream URL", func(t *testing.T) {
		for _, settings := range []string{
			`{"tob_display_url":""}`,
			`{"tob_display_url":"   "}`,
			`not-json`,
		} {
			view := selfChannelView{BaseURL: &baseURL, Settings: settings}
			got := selfChannelDisplayURL(view)
			require.NotNil(t, got)
			require.Equal(t, baseURL, *got)
		}
	})
}

func TestRespondChannelTestIncludesSameErrorCodeForAllCallers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	errorCode := types.ErrorCodeBadResponse
	result := testResult{
		localErr:    types.NewError(errors.New("upstream failed"), errorCode),
		newAPIError: types.NewError(errors.New("upstream failed"), errorCode),
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	respondChannelTest(ctx, result, time.Now())

	var response struct {
		Success   bool            `json:"success"`
		ErrorCode types.ErrorCode `json:"error_code"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, errorCode, response.ErrorCode)
}

func TestSettleTestQuotaUsesTieredBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:   "tiered_expr",
			ExprString:    `param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`,
			ExprHash:      billingexpr.ExprHashString(`param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`),
			GroupRatio:    1,
			EstimatedTier: "stream",
			QuotaPerUnit:  common.QuotaPerUnit,
			ExprVersion:   1,
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"stream":true}`),
		},
	}

	quota, result := settleTestQuota(info, types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 2,
	}, &dto.Usage{
		PromptTokens: 1000,
	})

	require.Equal(t, 1500, quota)
	require.NotNil(t, result)
	require.Equal(t, "stream", result.MatchedTier)
}

func TestSettleTestQuotaAppliesBillingRateAndGroupRatio(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	usage := &dto.Usage{PromptTokens: 1000, TotalTokens: 1000}

	ratioQuota, result := settleTestQuota(info, types.PriceData{
		ModelRatio:          2,
		CompletionRatio:     1,
		BillingUSDToCNYRate: 7.3,
		GroupRatioInfo:      types.GroupRatioInfo{GroupRatio: 0.05},
	}, usage)
	require.Nil(t, result)
	require.Equal(t, 730, ratioQuota)

	fixedQuota, result := settleTestQuota(info, types.PriceData{
		UsePrice:            true,
		ModelPrice:          1,
		BillingUSDToCNYRate: 7.3,
		GroupRatioInfo:      types.GroupRatioInfo{GroupRatio: 0.05},
	}, usage)
	require.Nil(t, result)
	require.Equal(t, 182500, fixedQuota)
}

func TestSettleTestQuotaSaturatesOverflow(t *testing.T) {
	priceData := types.PriceData{
		UsePrice:            true,
		ModelPrice:          math.MaxFloat64,
		BillingUSDToCNYRate: 7.3,
		GroupRatioInfo:      types.GroupRatioInfo{GroupRatio: 1},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
		PriceData:   priceData,
	}
	usage := &dto.Usage{PromptTokens: 1, TotalTokens: 1}

	quota, result := settleTestQuota(info, priceData, usage)

	require.Nil(t, result)
	require.Equal(t, common.MaxQuota, quota)
	require.NotNil(t, info.QuotaClamp)
	require.Equal(t, common.QuotaClampOverflow, info.QuotaClamp.Kind)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	other := buildTestLogOther(ctx, info, priceData, usage, nil)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	require.NotNil(t, adminInfo["quota_saturation"])
}

func TestBuildTestLogOtherInjectsTieredInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr",
			ExprString:  `tier("base", p * 2)`,
		},
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 12,
		},
	}

	other := buildTestLogOther(ctx, info, priceData, usage, &billingexpr.TieredResult{
		MatchedTier: "base",
	})

	require.Equal(t, "tiered_expr", other["billing_mode"])
	require.Equal(t, "base", other["matched_tier"])
	require.NotEmpty(t, other["expr_b64"])
}

func TestResolveChannelTestUserIDUsesRequestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("id", 2)

	userID, err := resolveChannelTestUserID(ctx)

	require.NoError(t, err)
	require.Equal(t, 2, userID)
}

func TestSelectChannelsForAutomaticTestPassiveRecoveryOnlyUsesAutoDisabled(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusAutoDisabled},
		{Id: 3, Status: common.ChannelStatusManuallyDisabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModePassiveRecovery)

	require.Len(t, selected, 1)
	require.Equal(t, 2, selected[0].Id)
}

func TestSelectChannelsForAutomaticTestScheduledSkipsManualDisabled(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusAutoDisabled},
		{Id: 3, Status: common.ChannelStatusManuallyDisabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModeScheduledAll)

	require.Len(t, selected, 2)
	require.Equal(t, 1, selected[0].Id)
	require.Equal(t, 2, selected[1].Id)
}

func TestTestAllChannelsRejectsExistingActiveTask(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemTask{}, &model.SystemTaskLock{}))

	existing, err := model.CreateSystemTask(model.SystemTaskTypeChannelTest, nil, nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/test", nil)

	TestAllChannels(ctx)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), existing.TaskID)
	require.Contains(t, recorder.Body.String(), "已有通道测试任务正在运行或等待中")
}
