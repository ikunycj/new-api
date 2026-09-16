package controller

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type channelRecoveryTransport struct {
	beforeResponse func(*http.Request) error
	status         int
	notified       chan struct{}
}

type channelRecoveryNotifyBody struct {
	io.ReadCloser
	done chan struct{}
}

func (body channelRecoveryNotifyBody) Close() error {
	err := body.ReadCloser.Close()
	body.done <- struct{}{}
	return err
}

func (transport *channelRecoveryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != "127.0.0.1" {
		return nil, fmt.Errorf("测试禁止外部请求: %s", request.URL.Host)
	}
	if request.URL.Path == "/notify" {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: channelRecoveryNotifyBody{
			ReadCloser: io.NopCloser(strings.NewReader(`{}`)), done: transport.notified,
		}}, nil
	}
	if transport.beforeResponse != nil {
		if err := transport.beforeResponse(request); err != nil {
			return nil, err
		}
	}
	body := `{"id":"test","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
	if transport.status != http.StatusOK {
		body = `{"error":{"message":"upstream unavailable","type":"server_error"}}`
	}
	return &http.Response{StatusCode: transport.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
}

var channelRecoveryFixtureID = 70000

// 使用独立 schema，不运行应用迁移；单连接池串行处理恢复事务及异步响应时间更新。
func setupChannelRecoveryControllerDB(t *testing.T) (*gorm.DB, *model.Channel, int, *channelRecoveryTransport, chan struct{}) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run isolated PostgreSQL channel recovery tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	schema := fmt.Sprintf("channel_recovery_%d", time.Now().UnixNano())
	require.NoError(t, db.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	t.Cleanup(func() { assert.NoError(t, db.Exec(`DROP SCHEMA "`+schema+`" CASCADE`).Error) })
	require.NoError(t, db.Exec(`SET search_path TO "`+schema+`"`).Error)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	previousCache, previousRedis, previousLogConsume := common.MemoryCacheEnabled, common.RedisEnabled, common.LogConsumeEnabled
	previousDisable, previousInterval := common.AutomaticDisableChannelEnabled, common.RequestInterval
	previousThreshold := common.ChannelDisableThreshold
	previousNotifyLimit := constant.NotifyLimitCount
	constant.NotifyLimitCount = 10
	model.DB = db
	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	common.MemoryCacheEnabled, common.RedisEnabled, common.LogConsumeEnabled = false, false, false
	common.AutomaticDisableChannelEnabled, common.RequestInterval = false, 0
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetDatabaseTypes(previousMain, previousLog)
		_ = model.InitLogDB()
		common.SetDatabaseTypes(previousMain, previousLog)
		model.LOG_DB = previousLogDB
		common.MemoryCacheEnabled, common.RedisEnabled, common.LogConsumeEnabled = previousCache, previousRedis, previousLogConsume
		common.AutomaticDisableChannelEnabled, common.RequestInterval = previousDisable, previousInterval
		common.ChannelDisableThreshold = previousThreshold
		constant.NotifyLimitCount = previousNotifyLimit
	})
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.User{}, &model.ChannelProbeState{}, &model.ChannelProbeHistory{}))
	if service.GetHttpClient() == nil {
		service.InitHttpClient()
	}
	client := service.GetHttpClient()
	previousTransport := client.Transport
	transport := &channelRecoveryTransport{status: http.StatusOK, notified: make(chan struct{}, 8)}
	client.Transport = transport
	fetchSetting := system_setting.GetFetchSetting()
	previousFetch := *fetchSetting
	fetchSetting.EnableSSRFProtection = false
	t.Cleanup(func() { client.Transport = previousTransport; *fetchSetting = previousFetch })
	channelRecoveryFixtureID++
	userID := channelRecoveryFixtureID
	settings, err := common.Marshal(dto.UserSetting{NotifyType: dto.NotifyTypeWebhook, WebhookUrl: "http://127.0.0.1/notify", AcceptUnsetRatioModel: true})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.User{Id: userID, Username: "probe-test", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default", Setting: string(settings)}).Error)
	channel := &model.Channel{Id: userID, Type: constant.ChannelTypeOpenAI, Name: "probe-test", Key: "first", BaseURL: common.GetPointer("http://127.0.0.1"),
		Status: common.ChannelStatusAutoDisabled, AutoProbeEnabled: common.GetPointer(true), AutoBan: common.GetPointer(0),
		Models: "gpt-4o", TestModel: common.GetPointer("gpt-4o"), Group: "default"}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.Ability{ChannelId: channel.Id, Model: "gpt-4o", Group: "default"}).Error)
	responseTimeUpdated := make(chan struct{}, 8)
	require.NoError(t, db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("test:response_time", func(update *gorm.DB) {
		for _, field := range update.Statement.Selects {
			if field == "response_time" {
				responseTimeUpdated <- struct{}{}
				break
			}
		}
	}))
	return db, channel, userID, transport, responseTimeUpdated
}

func waitChannelRecoveryEffect(t *testing.T, effect chan struct{}) {
	t.Helper()
	select {
	case <-effect:
	case <-time.After(5 * time.Second):
		t.Fatal("等待测试异步副作用完成超时")
	}
}

func TestManualChannelTestsRecoverOnlyWithCurrentPolicy(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, scenario := range []string{"recover", "slow success", "manual disable in flight", "switch off in flight", "probe off", "multi key recover", "key disabled in flight", "all keys disabled", "upstream failure", "failed recovery"} {
			t.Run(fmt.Sprintf("batch=%t/%s", batch, scenario), func(t *testing.T) {
				db, channel, userID, transport, responseTimeUpdated := setupChannelRecoveryControllerDB(t)
				wantStatus := common.ChannelStatusAutoDisabled
				wantSuccess := true
				if scenario == "recover" || scenario == "slow success" || scenario == "multi key recover" {
					wantStatus = common.ChannelStatusEnabled
				}
				if scenario == "slow success" {
					common.AutomaticDisableChannelEnabled = true
					common.ChannelDisableThreshold = 0.001
				}
				if scenario == "probe off" {
					channel.AutoProbeEnabled = common.GetPointer(false)
				}
				if scenario == "multi key recover" || scenario == "key disabled in flight" || scenario == "all keys disabled" {
					channel.Key = "first\nsecond"
					channel.ChannelInfo = model.ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyStatusList: map[int]int{1: common.ChannelStatusManuallyDisabled}}
				}
				if scenario == "all keys disabled" {
					channel.ChannelInfo.MultiKeyStatusList[0] = common.ChannelStatusAutoDisabled
					wantSuccess = false
				}
				if scenario == "upstream failure" {
					channel.Status = common.ChannelStatusEnabled
					wantStatus = common.ChannelStatusEnabled
				}
				if scenario == "upstream failure" || scenario == "failed recovery" {
					wantSuccess, transport.status = false, http.StatusServiceUnavailable
				}
				require.NoError(t, db.Save(channel).Error)
				require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", channel.Status == common.ChannelStatusEnabled).Error)
				requests := 0
				transport.beforeResponse = func(request *http.Request) error {
					requests++
					assert.Equal(t, "Bearer first", request.Header.Get("Authorization"))
					switch scenario {
					case "slow success":
						// 模拟成功但超过 1ms 禁用阈值的上游响应。
						time.Sleep(5 * time.Millisecond)
					case "manual disable in flight":
						assert.True(t, model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusManuallyDisabled, "manual"))
						wantStatus = common.ChannelStatusManuallyDisabled
					case "switch off in flight":
						return db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("auto_probe_enabled", false).Error
					case "key disabled in flight":
						assert.True(t, model.UpdateChannelStatus(channel.Id, "first", common.ChannelStatusAutoDisabled, "isolated"))
					}
					return nil
				}
				if batch {
					summary := performChannelTests(context.Background(), []*model.Channel{channel}, userID, nil)
					assert.Equal(t, 1, summary.Tested)
					// 批量统计保留阈值失败语义，但不能阻止真实成功后的恢复。
					wantBatchSuccess := wantSuccess && scenario != "slow success"
					assert.Equal(t, wantBatchSuccess, summary.Succeeded == 1)
					assert.Equal(t, !wantBatchSuccess, summary.Failed == 1)
					assert.Zero(t, summary.Disabled, "手动测试失败不能扩大业务自动禁用规则")
					waitChannelRecoveryEffect(t, responseTimeUpdated)
				} else {
					recorder := httptest.NewRecorder()
					ctx, _ := gin.CreateTestContext(recorder)
					ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/test", nil)
					ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(channel.Id)}}
					ctx.Set("id", userID)
					TestChannel(ctx)
					var response struct {
						Success bool   `json:"success"`
						Message string `json:"message"`
					}
					require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
					assert.Equal(t, wantSuccess, response.Success, response.Message)
					if wantSuccess {
						waitChannelRecoveryEffect(t, responseTimeUpdated)
					}
				}
				if wantStatus == common.ChannelStatusEnabled && wantSuccess {
					waitChannelRecoveryEffect(t, transport.notified)
				}
				if scenario == "all keys disabled" {
					assert.Zero(t, requests)
				} else {
					assert.Equal(t, 1, requests)
				}
				var stored model.Channel
				require.NoError(t, db.First(&stored, channel.Id).Error)
				assert.Equal(t, wantStatus, stored.Status)
				if scenario == "slow success" {
					assert.Greater(t, stored.ResponseTime, int(common.ChannelDisableThreshold*1000))
				}
				if channel.ChannelInfo.IsMultiKey {
					assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.ChannelInfo.MultiKeyStatusList[1])
					if scenario == "key disabled in flight" {
						assert.Equal(t, common.ChannelStatusAutoDisabled, stored.ChannelInfo.MultiKeyStatusList[0])
					}
				}
				var ability model.Ability
				require.NoError(t, db.First(&ability, "channel_id = ?", channel.Id).Error)
				assert.Equal(t, wantStatus == common.ChannelStatusEnabled, ability.Enabled)
			})
		}
	}
}

func TestManualBatchLocalFailureDoesNotRecoverChannel(t *testing.T) {
	db, channel, userID, _, responseTimeUpdated := setupChannelRecoveryControllerDB(t)
	channel.Type = constant.ChannelTypeKling
	summary := performChannelTests(context.Background(), []*model.Channel{channel}, userID, nil)
	assert.Equal(t, channelTestSummary{Tested: 1, Failed: 1}, summary)
	waitChannelRecoveryEffect(t, responseTimeUpdated)
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
}

func TestProbePolicyDoesNotExpandBusinessErrorAutoBan(t *testing.T) {
	db, channel, _, _, _ := setupChannelRecoveryControllerDB(t)
	channel.Status = common.ChannelStatusEnabled
	require.NoError(t, db.Save(channel).Error)
	upstreamError := types.NewError(fmt.Errorf("no enabled key"), types.ErrorCodeChannelNoAvailableKey)
	assert.False(t, service.ShouldDisableChannel(upstreamError))
	common.AutomaticDisableChannelEnabled = true
	assert.True(t, service.ShouldDisableChannel(upstreamError))
	service.DisableChannel(*types.NewChannelError(channel.Id, channel.Type, channel.Name, false, channel.Key, channel.GetAutoBan()), "business error")
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, channel.Status, "开启探测不能绕过业务 auto_ban")
}

func TestManualBatchFailureRespectsBusinessAutoBan(t *testing.T) {
	for _, slowSuccess := range []bool{false, true} {
		for _, tc := range []struct {
			name          string
			globalDisable bool
			autoBan       int
			wantDisabled  int
		}{
			{"global disabled", false, 1, 0},
			{"channel auto ban disabled", true, 0, 0},
			{"business auto ban enabled", true, 1, 1},
		} {
			t.Run(fmt.Sprintf("slow_success=%t/%s", slowSuccess, tc.name), func(t *testing.T) {
				db, channel, userID, transport, responseTimeUpdated := setupChannelRecoveryControllerDB(t)
				previousRanges, previousErrorLog := operation_setting.AutomaticDisableStatusCodeRanges, constant.ErrorLogEnabled
				operation_setting.AutomaticDisableStatusCodeRanges = []operation_setting.StatusCodeRange{{Start: http.StatusUnauthorized, End: http.StatusUnauthorized}}
				constant.ErrorLogEnabled = false
				t.Cleanup(func() {
					operation_setting.AutomaticDisableStatusCodeRanges = previousRanges
					constant.ErrorLogEnabled = previousErrorLog
				})
				common.AutomaticDisableChannelEnabled = tc.globalDisable
				common.ChannelDisableThreshold = 0
				channel.Status, channel.AutoBan = common.ChannelStatusEnabled, common.GetPointer(tc.autoBan)
				require.NoError(t, db.Save(channel).Error)
				require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", true).Error)
				transport.status = http.StatusUnauthorized
				if slowSuccess {
					transport.status = http.StatusOK
					common.ChannelDisableThreshold = 0.001
					transport.beforeResponse = func(*http.Request) error {
						time.Sleep(5 * time.Millisecond)
						return nil
					}
				}

				summary := performChannelTests(context.Background(), []*model.Channel{channel}, userID, nil)
				wantSummary := channelTestSummary{Tested: 1, Failed: 1, Disabled: tc.wantDisabled}
				if slowSuccess && !tc.globalDisable {
					wantSummary.Succeeded, wantSummary.Failed = 1, 0
				}
				assert.Equal(t, wantSummary, summary)
				waitChannelRecoveryEffect(t, responseTimeUpdated)
				wantStatus := common.ChannelStatusEnabled
				if tc.wantDisabled == 1 {
					waitChannelRecoveryEffect(t, transport.notified)
					wantStatus = common.ChannelStatusAutoDisabled
				}
				var stored model.Channel
				require.NoError(t, db.First(&stored, channel.Id).Error)
				assert.Equal(t, wantStatus, stored.Status)
				var ability model.Ability
				require.NoError(t, db.First(&ability, "channel_id = ?", channel.Id).Error)
				assert.Equal(t, wantStatus == common.ChannelStatusEnabled, ability.Enabled)
			})
		}
	}
}

func TestChannelProbeExecutorRecoversMultiKeyWithoutChangingIsolation(t *testing.T) {
	db, channel, userID, transport, _ := setupChannelRecoveryControllerDB(t)
	channel.Key = "first\nsecond"
	channel.ChannelInfo = model.ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyStatusList: map[int]int{1: common.ChannelStatusAutoDisabled}}
	require.NoError(t, db.Save(channel).Error)
	executor := channelProbeExecutor{testUserID: userID, request: func(ctx context.Context, channel *model.Channel, userID int) testResult {
		return testChannelWithTokenName(ctx, channel, userID, "", "", false, channelProbeTokenName, "")
	}}
	result := executor.run(context.Background(), channel.Id)
	require.NoError(t, result.err)
	assert.Equal(t, channelProbeSummary{Checked: 1, Succeeded: 1, Enabled: 1}, result.summary)
	waitChannelRecoveryEffect(t, transport.notified)
	var stored model.Channel
	require.NoError(t, db.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	assert.Equal(t, channel.ChannelInfo, stored.ChannelInfo)
}

func TestPricingGroupMonitorSuccessDoesNotRecoverChannel(t *testing.T) {
	db, channel, _, _, _ := setupChannelRecoveryControllerDB(t)
	status, _, err := ProbePricingGroupChannel(context.Background(), channel, &model.ChannelMonitor{TestModel: "gpt-4o", PricingGroup: "default"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
}

func TestChannelProbeExecutorCommitsOnlyFinalResult(t *testing.T) {
	for _, recoverOnRetry := range []bool{false, true} {
		t.Run(fmt.Sprint(recoverOnRetry), func(t *testing.T) {
			db, channel, userID, transport, _ := setupChannelRecoveryControllerDB(t)
			channel.Status = common.ChannelStatusEnabled
			require.NoError(t, db.Save(channel).Error)
			require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", true).Error)
			attempts := 0
			transport.beforeResponse = func(*http.Request) error {
				attempts++
				transport.status = http.StatusServiceUnavailable
				if recoverOnRetry && attempts == 2 {
					transport.status = http.StatusOK
				}
				var stored model.Channel
				if err := db.First(&stored, channel.Id).Error; err != nil {
					return err
				}
				assert.Equal(t, common.ChannelStatusEnabled, stored.Status, "重试结束前不能提前禁用")
				return nil
			}
			executor := channelProbeExecutor{testUserID: userID, request: func(ctx context.Context, channel *model.Channel, userID int) testResult {
				return testChannelWithTokenName(ctx, channel, userID, "", "", false, channelProbeTokenName, "")
			}}
			result := executor.run(context.Background(), channel.Id)
			require.NoError(t, result.err)
			assert.Equal(t, 2, attempts)
			assert.Equal(t, 1, result.summary.Checked)
			assert.Equal(t, recoverOnRetry, result.summary.Succeeded == 1)
			assert.Equal(t, !recoverOnRetry, result.summary.Disabled == 1)
			if !recoverOnRetry {
				waitChannelRecoveryEffect(t, transport.notified)
			}
			var history []model.ChannelProbeHistory
			require.NoError(t, db.Find(&history).Error)
			require.Len(t, history, 1)
			assert.Equal(t, recoverOnRetry, history[0].Success)
			var state model.ChannelProbeState
			require.NoError(t, db.First(&state, "channel_id = ?", channel.Id).Error)
			assert.Zero(t, state.LeaseUntil)
			wantInterval := int64(channel.GetProbeIntervalSeconds())
			if !recoverOnRetry {
				wantInterval = int64(channel.GetAutoDisabledProbeIntervalSeconds())
			}
			assert.Equal(t, wantInterval, state.NextProbeAt-state.LastProbeAt)
		})
	}
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
