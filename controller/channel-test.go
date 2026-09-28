package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/samber/lo"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"

	"github.com/gin-gonic/gin"
)

type testResult struct {
	context          *gin.Context
	localErr         error
	newAPIError      *types.NewAPIError
	ttftMs           int64
	response         string
	responseBytes    int
	rawResponseBytes int
	reasoningTokens  int
	finishReason     string
}

const (
	modelTestTokenName    = model.ChannelTestTokenName
	channelProbeTokenName = model.ChannelProbeTokenName
	channelIQTestPrompt   = "Create an HTML file with an SVG 2D animation of a pelican riding a bicycle"
	// IQ output is a self-contained artifact rather than an open-ended coding
	// task. Keep the prompt compact, but enforce the artifact, transport, and
	// generation budgets independently: HTML size is measured after extracting
	// the visible answer, while the response limit also covers protocol fields
	// and provider reasoning.
	channelIQTestOutputContract   = "Return only one self-contained HTML document. Keep the source reasonably compact, with an inline SVG containing visible shapes and CSS or SVG animation. Do not add comments, libraries, external assets, network requests, Markdown fences, or explanations."
	channelIQTestMaxBytes         = 128 << 10
	channelIQTestMaxResponseBytes = 512 << 10
	channelIQTestMaxTokens        = uint(8192)
	channelIQTestTimeout          = 90 * time.Second
)

type testResponseBody struct {
	io.ReadCloser
	info *relaycommon.RelayInfo
}

func (body *testResponseBody) Read(p []byte) (int, error) {
	n, err := body.ReadCloser.Read(p)
	if n > 0 && body.info != nil {
		body.info.SetFirstResponseTime()
	}
	return n, err
}

func testTTFTMilliseconds(info *relaycommon.RelayInfo) int64 {
	if info == nil || !info.HasSendResponse() {
		return 0
	}
	ttftMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	if ttftMs <= 0 {
		return 0
	}
	return ttftMs
}

func supportsChannelTest(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeMidjourney,
		constant.ChannelTypeMidjourneyPlus,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu:
		return false
	default:
		return true
	}
}

func normalizeChannelTestEndpoint(channel *model.Channel, modelName, endpointType string) string {
	normalized := strings.TrimSpace(endpointType)
	if normalized != "" {
		return normalized
	}
	if strings.HasSuffix(modelName, ratio_setting.CompactModelSuffix) {
		return string(constant.EndpointTypeOpenAIResponseCompact)
	}
	if channel != nil && channel.Type == constant.ChannelTypeCodex {
		return string(constant.EndpointTypeOpenAIResponse)
	}
	return normalized
}

func resolveChannelTestUserID(c *gin.Context) (int, error) {
	if c != nil {
		if userID := c.GetInt("id"); userID > 0 {
			return userID, nil
		}
	}

	var rootUser model.User
	if err := model.DB.Select("id").Where("role = ?", common.RoleRootUser).First(&rootUser).Error; err != nil {
		return 0, fmt.Errorf("failed to resolve channel test user: %w", err)
	}
	if rootUser.Id == 0 {
		return 0, errors.New("failed to resolve channel test user")
	}
	return rootUser.Id, nil
}

func testChannel(ctx context.Context, channel *model.Channel, testUserID int, testModel string, endpointType string, isStream bool) testResult {
	return testChannelWithTokenName(ctx, channel, testUserID, testModel, endpointType, isStream, modelTestTokenName, "")
}

// ProbePricingGroupChannel tests the channel selected for a pricing-group
// monitor with the channel's own upstream credentials.
func ProbePricingGroupChannel(ctx context.Context, channel *model.Channel, monitor *model.ChannelMonitor) (int, int, error) {
	if channel == nil || monitor == nil {
		return 0, 0, errors.New("pricing group monitor has no channel")
	}
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return 0, 0, err
	}
	startedAt := time.Now()
	result := testChannelWithTokenName(
		ctx,
		channel,
		testUserID,
		monitor.TestModel,
		"",
		shouldUseStreamForAutomaticChannelTest(channel),
		channelProbeTokenName,
		monitor.PricingGroup,
	)
	latencyMs := int(time.Since(startedAt).Milliseconds())
	if result.localErr != nil {
		return 0, latencyMs, result.localErr
	}
	return http.StatusOK, latencyMs, nil
}

func testChannelWithTokenName(ctx context.Context, channel *model.Channel, testUserID int, testModel string, endpointType string, isStream bool, tokenName string, pricingGroup string) testResult {
	return testChannelWithPrompt(ctx, channel, testUserID, testModel, endpointType, isStream, tokenName, pricingGroup, "")
}

func testChannelWithPrompt(ctx context.Context, channel *model.Channel, testUserID int, testModel string, endpointType string, isStream bool, tokenName string, pricingGroup string, prompt string) testResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if prompt != "" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, channelIQTestTimeout)
		defer cancel()
	}
	tokenName = strings.TrimSpace(tokenName)
	if tokenName == "" {
		tokenName = modelTestTokenName
	}
	tik := time.Now()
	if !supportsChannelTest(channel.Type) {
		channelTypeName := constant.GetChannelTypeName(channel.Type)
		return testResult{
			localErr: fmt.Errorf("%s channel test is not supported", channelTypeName),
		}
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testModel = strings.TrimSpace(testModel)
	if testModel == "" {
		testModel = channel.GetTestModel()
	}
	if testModel == "" {
		return testResult{localErr: errors.New("test model is required")}
	}

	endpointType = normalizeChannelTestEndpoint(channel, testModel, endpointType)

	requestPath := "/v1/chat/completions"

	// 如果指定了端点类型，使用指定的端点类型
	if endpointType != "" {
		if endpointInfo, ok := common.GetDefaultEndpointInfo(constant.EndpointType(endpointType)); ok {
			requestPath = endpointInfo.Path
		}
	} else {
		// 如果没有指定端点类型，使用原有的自动检测逻辑

		if strings.Contains(strings.ToLower(testModel), "rerank") {
			requestPath = "/v1/rerank"
		}

		// 先判断是否为 Embedding 模型
		if strings.Contains(strings.ToLower(testModel), "embedding") ||
			strings.HasPrefix(testModel, "m3e") || // m3e 系列模型
			strings.Contains(testModel, "bge-") || // bge 系列模型
			strings.Contains(testModel, "embed") ||
			channel.Type == constant.ChannelTypeMokaAI { // 其他 embedding 模型
			requestPath = "/v1/embeddings" // 修改请求路径
		}

		// VolcEngine 图像生成模型
		if channel.Type == constant.ChannelTypeVolcEngine && strings.Contains(testModel, "seedream") {
			requestPath = "/v1/images/generations"
		}

		// responses-only models
		if strings.Contains(strings.ToLower(testModel), "codex") {
			requestPath = "/v1/responses"
		}

		// responses compaction models (must use /v1/responses/compact)
		if strings.HasSuffix(testModel, ratio_setting.CompactModelSuffix) {
			requestPath = "/v1/responses/compact"
		}
	}
	if strings.HasPrefix(requestPath, "/v1/responses/compact") {
		testModel = ratio_setting.WithCompactModelSuffix(testModel)
	}
	if prompt != "" && strings.HasPrefix(requestPath, "/v1/responses/compact") {
		return testResult{localErr: errors.New("智商测试不支持 Responses 压缩端点")}
	}
	if prompt != "" && !isChannelIQTestPath(requestPath) {
		return testResult{localErr: fmt.Errorf("智商测试仅支持文本生成模型")}
	}

	c.Request = httptest.NewRequestWithContext(ctx, http.MethodPost, requestPath, nil)

	cache, err := model.GetUserCache(testUserID)
	if err != nil {
		return testResult{
			localErr:    err,
			newAPIError: nil,
		}
	}
	cache.WriteContext(c)
	c.Set("id", testUserID)

	//c.Request.Header.Set("Authorization", "Bearer "+channel.Key)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("channel", channel.Type)
	c.Set("base_url", channel.GetBaseURL())
	group, _ := model.GetUserGroup(testUserID, false)
	c.Set("group", group)
	if pricingGroup != "" {
		common.SetContextKey(c, constant.ContextKeyUsingGroup, pricingGroup)
	}

	newAPIError := middleware.SetupContextForSelectedChannel(c, channel, testModel)
	if newAPIError != nil {
		return testResult{
			context:     c,
			localErr:    newAPIError,
			newAPIError: newAPIError,
		}
	}

	// Determine relay format based on endpoint type or request path
	var relayFormat types.RelayFormat
	if endpointType != "" {
		// 根据指定的端点类型设置 relayFormat
		switch constant.EndpointType(endpointType) {
		case constant.EndpointTypeOpenAI:
			relayFormat = types.RelayFormatOpenAI
		case constant.EndpointTypeOpenAIResponse:
			relayFormat = types.RelayFormatOpenAIResponses
		case constant.EndpointTypeOpenAIResponseCompact:
			relayFormat = types.RelayFormatOpenAIResponsesCompaction
		case constant.EndpointTypeOpenAIAlphaSearch:
			relayFormat = types.RelayFormatOpenAIAlphaSearch
		case constant.EndpointTypeAnthropic:
			relayFormat = types.RelayFormatClaude
		case constant.EndpointTypeGemini:
			relayFormat = types.RelayFormatGemini
		case constant.EndpointTypeJinaRerank:
			relayFormat = types.RelayFormatRerank
		case constant.EndpointTypeImageGeneration:
			relayFormat = types.RelayFormatOpenAIImage
		case constant.EndpointTypeEmbeddings:
			relayFormat = types.RelayFormatEmbedding
		default:
			relayFormat = types.RelayFormatOpenAI
		}
	} else {
		// 根据请求路径自动检测
		relayFormat = types.RelayFormatOpenAI
		if c.Request.URL.Path == "/v1/embeddings" {
			relayFormat = types.RelayFormatEmbedding
		}
		if c.Request.URL.Path == "/v1/images/generations" {
			relayFormat = types.RelayFormatOpenAIImage
		}
		if c.Request.URL.Path == "/v1/messages" {
			relayFormat = types.RelayFormatClaude
		}
		if strings.Contains(c.Request.URL.Path, "/v1beta/models") {
			relayFormat = types.RelayFormatGemini
		}
		if c.Request.URL.Path == "/v1/rerank" || c.Request.URL.Path == "/rerank" {
			relayFormat = types.RelayFormatRerank
		}
		if c.Request.URL.Path == "/v1/responses" {
			relayFormat = types.RelayFormatOpenAIResponses
		}
		if strings.HasPrefix(c.Request.URL.Path, "/v1/responses/compact") {
			relayFormat = types.RelayFormatOpenAIResponsesCompaction
		}
		if c.Request.URL.Path == "/v1/alpha/search" {
			relayFormat = types.RelayFormatOpenAIAlphaSearch
		}
	}

	request := buildTestRequest(testModel, endpointType, channel, isStream, prompt)

	info, err := relaycommon.GenRelayInfo(c, relayFormat, request, nil)

	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeGenRelayInfoFailed),
		}
	}

	info.IsChannelTest = true
	info.InitChannelMeta(c)

	err = attachTestBillingRequestInput(info, request)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeJsonMarshalFailed),
		}
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeChannelModelMappedError),
		}
	}

	testModel = info.UpstreamModelName
	// 更新请求中的模型名称
	request.SetModelName(testModel)

	apiType, _ := common.ChannelType2APIType(channel.Type)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact &&
		apiType != constant.APITypeOpenAI &&
		apiType != constant.APITypeCodex {
		return testResult{
			context:     c,
			localErr:    fmt.Errorf("responses compaction test only supports openai/codex channels, got api type %d", apiType),
			newAPIError: types.NewError(fmt.Errorf("unsupported api type: %d", apiType), types.ErrorCodeInvalidApiType),
		}
	}
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return testResult{
			context:     c,
			localErr:    fmt.Errorf("invalid api type: %d, adaptor is nil", apiType),
			newAPIError: types.NewError(fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), types.ErrorCodeInvalidApiType),
		}
	}

	//// 创建一个用于日志的 info 副本，移除 ApiKey
	//logInfo := info
	//logInfo.ApiKey = ""
	common.SysLog(fmt.Sprintf("testing channel %d with model %s , info %+v ", channel.Id, testModel, info.ToString()))

	priceData, err := helper.ModelPriceHelper(c, info, 0, request.GetTokenCountMeta())
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithStatusCode(http.StatusBadRequest)),
		}
	}

	adaptor.Init(info)

	var convertedRequest any
	// 根据 RelayMode 选择正确的转换函数
	switch info.RelayMode {
	case relayconstant.RelayModeEmbeddings:
		// Embedding 请求 - request 已经是正确的类型
		if embeddingReq, ok := request.(*dto.EmbeddingRequest); ok {
			convertedRequest, err = adaptor.ConvertEmbeddingRequest(c, info, *embeddingReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid embedding request type"),
				newAPIError: types.NewError(errors.New("invalid embedding request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeImagesGenerations:
		// 图像生成请求 - request 已经是正确的类型
		if imageReq, ok := request.(*dto.ImageRequest); ok {
			convertedRequest, err = adaptor.ConvertImageRequest(c, info, *imageReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid image request type"),
				newAPIError: types.NewError(errors.New("invalid image request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeRerank:
		// Rerank 请求 - request 已经是正确的类型
		if rerankReq, ok := request.(*dto.RerankRequest); ok {
			convertedRequest, err = adaptor.ConvertRerankRequest(c, info.RelayMode, *rerankReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid rerank request type"),
				newAPIError: types.NewError(errors.New("invalid rerank request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeResponses:
		// Response 请求 - request 已经是正确的类型
		if responseReq, ok := request.(*dto.OpenAIResponsesRequest); ok {
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, *responseReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid response request type"),
				newAPIError: types.NewError(errors.New("invalid response request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeResponsesCompact:
		// Response compaction request - convert to OpenAIResponsesRequest before adapting
		switch req := request.(type) {
		case *dto.OpenAIResponsesCompactionRequest:
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, dto.OpenAIResponsesRequest{
				Model:              req.Model,
				Input:              req.Input,
				Instructions:       req.Instructions,
				PreviousResponseID: req.PreviousResponseID,
			})
		case *dto.OpenAIResponsesRequest:
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(c, info, *req)
		default:
			return testResult{
				context:     c,
				localErr:    errors.New("invalid response compaction request type"),
				newAPIError: types.NewError(errors.New("invalid response compaction request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	case relayconstant.RelayModeAlphaSearch:
		if alphaReq, ok := request.(*dto.AlphaSearchRequest); ok {
			convertedRequest = alphaReq
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid alpha search request type"),
				newAPIError: types.NewError(errors.New("invalid alpha search request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	default:
		// Chat/Completion 等其他请求类型
		if generalReq, ok := request.(*dto.GeneralOpenAIRequest); ok {
			convertedRequest, err = adaptor.ConvertOpenAIRequest(c, info, generalReq)
		} else {
			return testResult{
				context:     c,
				localErr:    errors.New("invalid general request type"),
				newAPIError: types.NewError(errors.New("invalid general request type"), types.ErrorCodeConvertRequestFailed),
			}
		}
	}

	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
		}
	}
	if prompt != "" {
		disableChannelIQTestThinking(convertedRequest)
	}
	var jsonData []byte
	if info.RelayMode == relayconstant.RelayModeAlphaSearch {
		alphaReq, ok := request.(*dto.AlphaSearchRequest)
		if !ok {
			err = errors.New("invalid alpha search request type")
		} else {
			jsonData, err = buildChannelAlphaSearchRequestBody(alphaReq.RawBody, info.OriginModelName, info.UpstreamModelName)
		}
	} else {
		jsonData, err = common.Marshal(convertedRequest)
	}
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewError(err, types.ErrorCodeJsonMarshalFailed),
		}
	}

	//jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings)
	//if err != nil {
	//	return testResult{
	//		context:     c,
	//		localErr:    err,
	//		newAPIError: types.NewError(err, types.ErrorCodeConvertRequestFailed),
	//	}
	//}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
				return testResult{
					context:     c,
					localErr:    fixedErr,
					newAPIError: relaycommon.NewAPIErrorFromParamOverride(fixedErr),
				}
			}
			return testResult{
				context:     c,
				localErr:    err,
				newAPIError: types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid),
			}
		}
	}
	if prompt != "" {
		jsonData, err = capChannelIQTestTokenFields(jsonData)
		if err != nil {
			return testResult{
				context:     c,
				localErr:    err,
				newAPIError: types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid),
			}
		}
		jsonData, err = finalizeChannelIQTestRequest(jsonData, convertedRequest, channel, testModel)
		if err != nil {
			return testResult{
				context:     c,
				localErr:    err,
				newAPIError: types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid),
			}
		}
	}

	requestBody := bytes.NewBuffer(jsonData)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError),
		}
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if prompt != "" && !isStream && httpResp.Body != nil {
			httpResp.Body = newLimitedTestResponseBody(httpResp.Body, channelIQTestMaxResponseBytes)
		}
		if httpResp.StatusCode != http.StatusOK {
			err := service.RelayErrorHandler(c.Request.Context(), httpResp, true)
			common.SysError(fmt.Sprintf(
				"channel test bad response: channel_id=%d name=%s type=%d model=%s endpoint_type=%s status=%d err=%v",
				channel.Id,
				channel.Name,
				channel.Type,
				testModel,
				endpointType,
				httpResp.StatusCode,
				err,
			))
			return testResult{
				context:     c,
				localErr:    err,
				newAPIError: types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError),
			}
		}
	}
	if !isStream && httpResp != nil && httpResp.Body != nil {
		httpResp.Body = &testResponseBody{ReadCloser: httpResp.Body, info: info}
	}
	usageA, respErr := adaptor.DoResponse(c, httpResp, info)
	if respErr != nil {
		return testResult{
			context:     c,
			localErr:    respErr,
			newAPIError: respErr,
		}
	}
	usage, usageErr := coerceTestUsage(usageA, isStream, info.GetEstimatePromptTokens())
	if usageErr != nil {
		return testResult{
			context:     c,
			localErr:    usageErr,
			newAPIError: types.NewOpenAIError(usageErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	result := w.Result()
	var respBody []byte
	if prompt != "" {
		respBody, err = readLimitedTestResponseBody(result.Body, channelIQTestMaxResponseBytes)
	} else {
		respBody, err = readTestResponseBody(result.Body, isStream)
	}
	if err != nil {
		return testResult{
			context:     c,
			localErr:    err,
			newAPIError: types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError),
		}
	}
	if bodyErr := validateTestResponseBody(respBody, isStream); bodyErr != nil {
		return testResult{
			context:     c,
			localErr:    bodyErr,
			newAPIError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
		}
	}
	var generatedResponse string
	var finishReason string
	if prompt != "" {
		finishReason = extractChannelIQTestFinishReason(respBody)
		generatedResponse = normalizeChannelIQTestHTML(extractChannelTestText(respBody))
		if err := validateChannelIQTestHTMLSize(generatedResponse); err != nil {
			return testResult{
				context:          c,
				localErr:         err,
				newAPIError:      types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
				rawResponseBytes: len(respBody),
				responseBytes:    len(generatedResponse),
				finishReason:     finishReason,
			}
		}
		if err := validateChannelIQTestHTML(generatedResponse, finishReason); err != nil {
			return testResult{
				context:          c,
				localErr:         err,
				newAPIError:      types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
				rawResponseBytes: len(respBody),
				responseBytes:    len(generatedResponse),
				finishReason:     finishReason,
			}
		}
	}
	info.SetEstimatePromptTokens(usage.PromptTokens)

	quota, tieredResult := settleTestQuota(info, priceData, usage)
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	consumedTime := float64(milliseconds) / 1000.0
	other := buildTestLogOther(c, info, priceData, usage, tieredResult)
	cacheInputTokens, cacheStatsAvailable := service.CacheStatsInputTokens(info, usage)
	ttftMs := testTTFTMilliseconds(info)
	model.RecordConsumeLog(c, testUserID, model.RecordConsumeLogParams{
		ChannelId:           channel.Id,
		PromptTokens:        usage.PromptTokens,
		CompletionTokens:    usage.CompletionTokens,
		InputTokensTotal:    cacheInputTokens,
		CacheReadTokens:     usage.PromptTokensDetails.CachedTokens,
		CacheWriteTokens:    usage.PromptTokensDetails.CacheCreationTokensTotal(),
		CacheStatsAvailable: cacheStatsAvailable,
		ModelName:           info.OriginModelName,
		TokenName:           tokenName,
		Quota:               quota,
		Content:             tokenName,
		UseTimeSeconds:      int(consumedTime),
		IsStream:            info.IsStream,
		Group:               info.UsingGroup,
		Other:               other,
	})
	model.UpdateCachedChannelTestTTFT(channel.Id, float64(ttftMs))
	if prompt == "" {
		common.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))
	} else {
		common.SysLog(fmt.Sprintf(
			"channel IQ test completed: channel_id=%d finish_reason=%q generated_bytes=%d response_bytes=%d reasoning_tokens=%d",
			channel.Id,
			finishReason,
			len(generatedResponse),
			len(respBody),
			usage.CompletionTokenDetails.ReasoningTokens,
		))
	}
	return testResult{
		context:          c,
		localErr:         nil,
		newAPIError:      nil,
		ttftMs:           ttftMs,
		response:         generatedResponse,
		responseBytes:    len(generatedResponse),
		rawResponseBytes: len(respBody),
		reasoningTokens:  usage.CompletionTokenDetails.ReasoningTokens,
		finishReason:     finishReason,
	}
}

func validateChannelIQTestHTMLSize(html string) error {
	if len(html) > channelIQTestMaxBytes {
		return fmt.Errorf("智商测试结果超过 %d 字节上限", channelIQTestMaxBytes)
	}
	return nil
}

func disableChannelIQTestThinking(request any) {
	switch testRequest := request.(type) {
	case *dto.ClaudeRequest:
		// Remove any thinking mode produced by a protocol converter. Providers
		// with an explicit off switch receive it again in the final request step.
		testRequest.Thinking = nil
		testRequest.OutputConfig = nil
		testRequest.MaxTokensToSample = nil
	case *dto.GeneralOpenAIRequest:
		// Custom routes can inject a reasoning field during protocol conversion.
		// These explicit disabled values are normalized again after channel
		// parameter overrides, so a channel override cannot turn reasoning back
		// on for the IQ request.
		testRequest.ReasoningEffort = ""
		testRequest.Reasoning = nil
		testRequest.THINKING = nil
		testRequest.EnableThinking = nil
		testRequest.Think = nil
	case *dto.OpenAIResponsesRequest:
		testRequest.Reasoning = nil
		testRequest.EnableThinking = nil
	case *dto.GeminiChatRequest:
		testRequest.GenerationConfig.ThinkingConfig = nil
	}
}

// finalizeChannelIQTestRequest reapplies the IQ-specific controls after
// channel parameter overrides. Overrides are intentionally allowed to shape
// normal requests, but they must not be able to re-enable thinking, streaming,
// or an unbounded output budget for this diagnostic request.
func finalizeChannelIQTestRequest(jsonData []byte, request any, channel *model.Channel, modelName string) ([]byte, error) {
	var payload map[string]any
	if err := common.Unmarshal(jsonData, &payload); err != nil {
		return nil, fmt.Errorf("invalid IQ test request after final normalization: %w", err)
	}

	switch request.(type) {
	case *dto.ClaudeRequest:
		payload["stream"] = false
		payload["max_tokens"] = channelIQTestMaxTokens
		delete(payload, "max_tokens_to_sample")
		delete(payload, "max_output_tokens")
		delete(payload, "max_completion_tokens")
		delete(payload, "maxCompletionTokens")
		delete(payload, "maxOutputTokens")
		if isDeepSeekIQClaudeRoute(channel, modelName) {
			payload["thinking"] = map[string]any{"type": "disabled"}
		} else {
			delete(payload, "thinking")
		}
		delete(payload, "output_config")
	case *dto.GeminiChatRequest:
		generationConfig, ok := payload["generationConfig"].(map[string]any)
		if !ok {
			generationConfig = map[string]any{}
		}
		if isGeminiThinkingModel(modelName) {
			generationConfig["thinkingConfig"] = map[string]any{
				"includeThoughts": false,
				"thinkingBudget":  0,
			}
		} else {
			delete(generationConfig, "thinkingConfig")
			delete(generationConfig, "thinking_config")
		}
		generationConfig["maxOutputTokens"] = channelIQTestMaxTokens
		payload["generationConfig"] = generationConfig
		delete(payload, "stream")
		delete(payload, "max_tokens")
		delete(payload, "max_completion_tokens")
		delete(payload, "max_output_tokens")
		delete(payload, "maxCompletionTokens")
		delete(payload, "maxOutputTokens")
	case *dto.GeneralOpenAIRequest:
		payload["stream"] = false
		if _, ok := payload["max_completion_tokens"]; ok {
			payload["max_completion_tokens"] = channelIQTestMaxTokens
			delete(payload, "max_tokens")
		} else {
			payload["max_tokens"] = channelIQTestMaxTokens
			delete(payload, "max_completion_tokens")
		}
		delete(payload, "max_output_tokens")
		delete(payload, "maxCompletionTokens")
		delete(payload, "maxOutputTokens")
		if channel != nil && channel.Type == constant.ChannelTypeDeepSeek {
			payload["thinking"] = map[string]any{"type": "disabled"}
		} else if dto.IsOpenAIReasoningOModel(modelName) {
			payload["reasoning_effort"] = "none"
		}
	case *dto.OpenAIResponsesRequest:
		payload["stream"] = false
		payload["max_output_tokens"] = channelIQTestMaxTokens
		delete(payload, "max_tokens")
		delete(payload, "max_completion_tokens")
		delete(payload, "maxCompletionTokens")
		delete(payload, "maxOutputTokens")
		if dto.IsOpenAIReasoningOModel(modelName) {
			payload["reasoning"] = map[string]any{"effort": "none"}
		}
	}

	return common.Marshal(payload)
}

func isDeepSeekIQClaudeRoute(channel *model.Channel, modelName string) bool {
	if channel == nil || !strings.Contains(strings.ToLower(modelName), "deepseek") {
		return false
	}
	if channel.Type == constant.ChannelTypeDeepSeek {
		return true
	}
	return channel.Type == constant.ChannelTypeAdvancedCustom &&
		strings.Contains(strings.ToLower(channel.GetBaseURL()), "/anthropic")
}

func isGeminiThinkingModel(modelName string) bool {
	normalized := strings.ToLower(modelName)
	return strings.Contains(normalized, "gemini-2.5") ||
		strings.Contains(normalized, "gemini-3") ||
		strings.Contains(normalized, "thinking")
}

func attachTestBillingRequestInput(info *relaycommon.RelayInfo, request dto.Request) error {
	if info == nil {
		return nil
	}

	input, err := helper.BuildBillingExprRequestInputFromRequest(request, info.RequestHeaders)
	if err != nil {
		return err
	}
	info.BillingRequestInput = &input
	return nil
}

func settleTestQuota(info *relaycommon.RelayInfo, priceData types.PriceData, usage *dto.Usage) (int, *billingexpr.TieredResult) {
	if usage != nil && info != nil && info.TieredBillingSnapshot != nil {
		isClaudeUsageSemantic := usage.UsageSemantic == "anthropic" || info.GetFinalRequestRelayFormat() == types.RelayFormatClaude
		usedVars := billingexpr.UsedVars(info.TieredBillingSnapshot.ExprString)
		if ok, quota, result := service.TryTieredSettle(info, service.BuildTieredTokenParams(usage, isClaudeUsageSemantic, usedVars)); ok {
			return quota, result
		}
	}

	billingUSDToCNYRate := priceData.EffectiveBillingUSDToCNYRate()
	groupRatio := priceData.GroupRatioInfo.GroupRatio
	recordClamp := func(clamp *common.QuotaClamp) {
		if clamp != nil && info != nil && info.QuotaClamp == nil {
			info.QuotaClamp = clamp
		}
	}

	if !priceData.UsePrice {
		tokens := float64(usage.PromptTokens) + float64(usage.CompletionTokens)*priceData.CompletionRatio
		multiplier := priceData.ModelRatio * billingUSDToCNYRate * groupRatio
		quota, clamp := common.QuotaRoundChecked(tokens * multiplier)
		recordClamp(clamp)
		if multiplier != 0 && quota <= 0 {
			quota = 1
		}
		return quota, nil
	}

	quota, clamp := common.QuotaFromFloatChecked(priceData.ModelPrice * common.QuotaPerUnit * billingUSDToCNYRate * groupRatio)
	recordClamp(clamp)
	if priceData.ModelPrice != 0 && billingUSDToCNYRate != 0 && groupRatio != 0 && quota <= 0 {
		quota = 1
	}
	return quota, nil
}

func buildTestLogOther(c *gin.Context, info *relaycommon.RelayInfo, priceData types.PriceData, usage *dto.Usage, tieredResult *billingexpr.TieredResult) map[string]interface{} {
	other := service.GenerateTextOtherInfo(c, info, priceData.ModelRatio, priceData.GroupRatioInfo.GroupRatio, priceData.CompletionRatio,
		usage.PromptTokensDetails.CachedTokens, priceData.CacheRatio, priceData.ModelPrice, priceData.GroupRatioInfo.GroupSpecialRatio)
	if tieredResult != nil {
		service.InjectTieredBillingInfo(other, info, tieredResult)
	}
	if info != nil && info.QuotaClamp != nil {
		adminInfo, ok := other["admin_info"].(map[string]interface{})
		if !ok || adminInfo == nil {
			adminInfo = map[string]interface{}{}
			other["admin_info"] = adminInfo
		}
		adminInfo["quota_saturation"] = info.QuotaClamp.AuditMap()
	}
	return other
}

func coerceTestUsage(usageAny any, isStream bool, estimatePromptTokens int) (*dto.Usage, error) {
	switch u := usageAny.(type) {
	case *dto.Usage:
		return u, nil
	case dto.Usage:
		return &u, nil
	case nil:
		if !isStream {
			return nil, errors.New("usage is nil")
		}
		usage := &dto.Usage{
			PromptTokens: estimatePromptTokens,
		}
		usage.TotalTokens = usage.PromptTokens
		return usage, nil
	default:
		if !isStream {
			return nil, fmt.Errorf("invalid usage type: %T", usageAny)
		}
		usage := &dto.Usage{
			PromptTokens: estimatePromptTokens,
		}
		usage.TotalTokens = usage.PromptTokens
		return usage, nil
	}
}

func readTestResponseBody(body io.ReadCloser, isStream bool) ([]byte, error) {
	defer func() { _ = body.Close() }()
	const maxStreamLogBytes = 8 << 10
	if isStream {
		return io.ReadAll(io.LimitReader(body, maxStreamLogBytes))
	}
	return io.ReadAll(body)
}

func readLimitedTestResponseBody(body io.ReadCloser, maxBytes int64) ([]byte, error) {
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("channel test response exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func capChannelIQTestTokenFields(jsonData []byte) ([]byte, error) {
	if len(jsonData) == 0 {
		return jsonData, nil
	}
	var payload map[string]any
	if err := common.Unmarshal(jsonData, &payload); err != nil {
		return nil, fmt.Errorf("invalid IQ test request after parameter override: %w", err)
	}
	var normalizeFields func(any) bool
	normalizeFields = func(value any) bool {
		foundTokenField := false
		switch object := value.(type) {
		case map[string]any:
			for key, field := range object {
				switch strings.ToLower(key) {
				case "max_tokens", "maxcompletiontokens", "max_completion_tokens", "maxoutputtokens", "max_output_tokens", "maxtokens":
					foundTokenField = true
					// IQ tests own their output budget. A channel override must not
					// lower it enough to make a valid artifact impossible.
					object[key] = channelIQTestMaxTokens
				case "stream":
					object[key] = false
				case "reasoning_effort", "reasoningeffort", "reasoning", "thinking", "enable_thinking", "enablethinking", "thinkingconfig", "thinking_config", "thinkingbudget", "thinking_budget", "includethoughts", "include_thoughts", "output_config", "outputconfig", "max_tokens_to_sample", "maxtokenstosample", "budget_tokens", "budgettokens":
					delete(object, key)
				default:
					if normalizeFields(field) {
						foundTokenField = true
					}
				}
			}
		case []any:
			for _, item := range object {
				if normalizeFields(item) {
					foundTokenField = true
				}
			}
		}
		return foundTokenField
	}
	if !normalizeFields(payload) {
		if contents, ok := payload["contents"]; ok && contents != nil {
			payload["generationConfig"] = map[string]any{"maxOutputTokens": channelIQTestMaxTokens}
		} else if _, ok := payload["input"]; ok {
			payload["max_output_tokens"] = channelIQTestMaxTokens
		} else {
			payload["max_tokens"] = channelIQTestMaxTokens
		}
	}
	return common.Marshal(payload)
}

func buildChannelAlphaSearchRequestBody(rawBody []byte, originModel, upstreamModel string) ([]byte, error) {
	if len(rawBody) == 0 {
		return nil, errors.New("empty alpha search request body")
	}
	if upstreamModel == "" || upstreamModel == originModel {
		return rawBody, nil
	}
	var payload map[string]any
	if err := common.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}
	payload["model"] = upstreamModel
	return common.Marshal(payload)
}

type limitedTestResponseBody struct {
	io.ReadCloser
	reader io.Reader
	max    int64
	total  int64
}

func newLimitedTestResponseBody(body io.ReadCloser, maxBytes int64) io.ReadCloser {
	return &limitedTestResponseBody{
		ReadCloser: body,
		reader:     io.LimitReader(body, maxBytes+1),
		max:        maxBytes,
	}
}

func (body *limitedTestResponseBody) Read(p []byte) (int, error) {
	n, err := body.reader.Read(p)
	body.total += int64(n)
	if body.total > body.max {
		return n, fmt.Errorf("channel test response exceeds %d bytes", body.max)
	}
	return n, err
}

func detectErrorFromTestResponseBody(respBody []byte) error {
	b := bytes.TrimSpace(respBody)
	if len(b) == 0 {
		return nil
	}
	if message := detectErrorMessageFromJSONBytes(b); message != "" {
		return fmt.Errorf("upstream error: %s", message)
	}

	for _, line := range bytes.Split(b, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}
		if message := detectErrorMessageFromJSONBytes(payload); message != "" {
			return fmt.Errorf("upstream error: %s", message)
		}
	}

	return nil
}

func validateStreamTestResponseBody(respBody []byte) error {
	b := bytes.TrimSpace(respBody)
	if len(b) == 0 {
		return errors.New("stream response body is empty")
	}

	for _, line := range bytes.Split(b, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}

		return nil
	}

	return errors.New("stream response body does not contain a valid stream event")
}

func validateTestResponseBody(respBody []byte, isStream bool) error {
	if bodyErr := detectErrorFromTestResponseBody(respBody); bodyErr != nil {
		return bodyErr
	}
	if isStream {
		return validateStreamTestResponseBody(respBody)
	}
	return nil
}

func shouldUseStreamForAutomaticChannelTest(channel *model.Channel) bool {
	return channel != nil && (channel.ProbeStreamEnabled || channel.Type == constant.ChannelTypeCodex)
}

func detectErrorMessageFromJSONBytes(jsonBytes []byte) string {
	if len(jsonBytes) == 0 {
		return ""
	}
	if jsonBytes[0] != '{' && jsonBytes[0] != '[' {
		return ""
	}
	errVal := gjson.GetBytes(jsonBytes, "error")
	if !errVal.Exists() || errVal.Type == gjson.Null {
		return ""
	}

	message := gjson.GetBytes(jsonBytes, "error.message").String()
	if message == "" {
		message = gjson.GetBytes(jsonBytes, "error.error.message").String()
	}
	if message == "" && errVal.Type == gjson.String {
		message = errVal.String()
	}
	if message == "" {
		message = errVal.Raw
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return "upstream returned error payload"
	}
	return message
}

func isChannelIQTestPath(requestPath string) bool {
	return strings.Contains(requestPath, "/chat/completions") ||
		strings.Contains(requestPath, "/messages") ||
		strings.Contains(requestPath, "/responses") ||
		strings.Contains(requestPath, "/generateContent")
}

func testPromptOrDefault(prompt string) string {
	if prompt != "" {
		return prompt
	}
	return "hi"
}

func channelIQTestInstructions(prompt string) json.RawMessage {
	if prompt == "" {
		return nil
	}
	data, err := common.Marshal(channelIQTestOutputContract)
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}

func channelIQTestMaxOutputTokens(prompt string) *uint {
	if prompt == "" {
		return nil
	}
	return lo.ToPtr(channelIQTestMaxTokens)
}

func extractChannelTestText(response []byte) string {
	for _, path := range []string{
		"choices.0.message.content",
		"choices.0.text",
		"output_text",
	} {
		value := gjson.GetBytes(response, path)
		if value.Type == gjson.String && strings.TrimSpace(value.String()) != "" {
			return strings.TrimSpace(value.String())
		}
		if value.IsArray() {
			var text strings.Builder
			value.ForEach(func(_, part gjson.Result) bool {
				if part.Type == gjson.String {
					text.WriteString(part.String())
				} else if part.Get("text").Type == gjson.String {
					text.WriteString(part.Get("text").String())
				}
				return true
			})
			if text.Len() > 0 {
				return strings.TrimSpace(text.String())
			}
		}
	}

	for _, path := range []string{"content.#.text"} {
		value := gjson.GetBytes(response, path)
		var text strings.Builder
		value.ForEach(func(_, part gjson.Result) bool {
			if part.Type == gjson.String {
				text.WriteString(part.String())
			}
			return true
		})
		if text.Len() > 0 {
			return strings.TrimSpace(text.String())
		}
	}

	output := gjson.GetBytes(response, "output")
	var responsesText strings.Builder
	output.ForEach(func(_, item gjson.Result) bool {
		item.Get("content").ForEach(func(_, part gjson.Result) bool {
			if part.Get("text").Type == gjson.String {
				responsesText.WriteString(part.Get("text").String())
			}
			return true
		})
		return true
	})
	if responsesText.Len() > 0 {
		return strings.TrimSpace(responsesText.String())
	}

	geminiParts := gjson.GetBytes(response, "candidates.0.content.parts")
	var geminiText strings.Builder
	geminiParts.ForEach(func(_, part gjson.Result) bool {
		if part.Get("text").Type == gjson.String {
			geminiText.WriteString(part.Get("text").String())
		}
		return true
	})
	if geminiText.Len() > 0 {
		return strings.TrimSpace(geminiText.String())
	}
	return ""
}

func extractChannelIQTestFinishReason(response []byte) string {
	for _, path := range []string{
		"choices.0.finish_reason",
		"stop_reason",
		"message.stop_reason",
		"delta.stop_reason",
		"candidates.0.finishReason",
		"candidates.0.finish_reason",
		"finish_reason",
	} {
		value := gjson.GetBytes(response, path)
		if value.Type == gjson.String && strings.TrimSpace(value.String()) != "" {
			return strings.TrimSpace(value.String())
		}
	}

	status := strings.TrimSpace(gjson.GetBytes(response, "status").String())
	if strings.EqualFold(status, "incomplete") {
		if reason := strings.TrimSpace(gjson.GetBytes(response, "incomplete_details.reason").String()); reason != "" {
			return reason
		}
		return status
	}
	if reason := strings.TrimSpace(gjson.GetBytes(response, "incomplete_details.reason").String()); reason != "" {
		return reason
	}
	return status
}

func normalizeChannelIQTestHTML(response string) string {
	response = strings.TrimSpace(response)
	lines := strings.Split(response, "\n")
	fenceStart := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && trimmed != "```" {
			fenceStart = index
			break
		}
	}
	if fenceStart >= 0 {
		fenceEnd := -1
		for index := fenceStart + 1; index < len(lines); index++ {
			if strings.TrimSpace(lines[index]) == "```" {
				fenceEnd = index
				break
			}
		}
		if fenceEnd < 0 {
			return response
		}
		response = strings.Join(lines[fenceStart+1:fenceEnd], "\n")
	}

	return trimChannelIQTestHTMLDocument(response)
}

func trimChannelIQTestHTMLDocument(response string) string {
	response = strings.TrimSpace(response)
	lower := strings.ToLower(response)
	start := -1
	for _, marker := range []string{"<!doctype", "<html", "<svg"} {
		if index := findChannelIQTagStart(lower, marker); index >= 0 && (start < 0 || index < start) {
			start = index
		}
	}
	if start >= 0 {
		response = response[start:]
		lower = strings.ToLower(response)
	}
	if end := strings.LastIndex(lower, "</html>"); end >= 0 {
		response = response[:end+len("</html>")]
	} else if end := strings.LastIndex(lower, "</svg>"); end >= 0 {
		response = response[:end+len("</svg>")]
	}
	return strings.TrimSpace(response)
}

func findChannelIQTagStart(value, marker string) int {
	for offset := 0; offset < len(value); {
		index := strings.Index(value[offset:], marker)
		if index < 0 {
			return -1
		}
		index += offset
		boundary := index + len(marker)
		if boundary >= len(value) || value[boundary] == '>' || value[boundary] == '/' || value[boundary] == ' ' || value[boundary] == '\t' || value[boundary] == '\n' || value[boundary] == '\r' {
			return index
		}
		offset = boundary
	}
	return -1
}

func validateChannelIQTestHTML(document, finishReason string) error {
	value := strings.TrimSpace(document)
	reason := strings.ToLower(strings.TrimSpace(finishReason))
	switch reason {
	case "length", "max_tokens", "max_output_tokens", "max_completion_tokens", "incomplete", "truncated", "max_tokens_exceeded":
		return fmt.Errorf("智商测试生成被截断（完成原因：%s）", finishReason)
	case "content_filter", "safety", "refusal", "blocked", "failed", "cancelled", "canceled", "in_progress", "other":
		return fmt.Errorf("智商测试未正常完成（完成原因：%s）", finishReason)
	case "tool_calls", "tool_use", "function_call":
		return fmt.Errorf("智商测试未生成 HTML（完成原因：%s）", finishReason)
	case "stop", "completed", "end_turn", "stop_sequence", "finished", "success", "":
	default:
		return fmt.Errorf("智商测试无法确认生成完成（完成原因：%s）", finishReason)
	}
	if value == "" {
		return errors.New("智商测试结果为空")
	}
	if strings.Contains(value, "```") {
		return errors.New("智商测试生成被截断：Markdown 代码围栏未闭合")
	}
	tokenizer := html.NewTokenizer(strings.NewReader(value))
	hasSVG := false
	hasSVGClose := false
	svgDepth := 0
	hasRenderableSVGChild := false
	hasHTML := false
	hasHTMLClose := false
	hasAnimation := false
	styleDepth := 0
	scriptDepth := 0
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				goto parsed
			}
			return fmt.Errorf("智商测试结果不完整：HTML 解析失败（%v）", tokenizer.Err())
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokenizer.TagName()
			tagName := strings.ToLower(string(name))
			switch tagName {
			case "base", "embed", "frame", "iframe", "object", "portal":
				return errors.New("智商测试结果包含不允许的嵌入内容")
			case "svg":
				hasSVG = true
				if tokenType == html.StartTagToken {
					svgDepth++
				}
			case "html":
				hasHTML = true
				if tokenType == html.SelfClosingTagToken {
					hasHTMLClose = true
				}
			case "animate", "animatetransform", "animatemotion":
				hasAnimation = true
			case "style":
				styleDepth++
			case "script":
				scriptDepth++
			default:
				if svgDepth > 0 && tagName != "style" && tagName != "script" && tagName != "title" && tagName != "desc" {
					hasRenderableSVGChild = true
				}
			}
			for {
				key, value, more := tokenizer.TagAttr()
				keyName := strings.ToLower(string(key))
				valueText := strings.ToLower(string(value))
				if strings.HasPrefix(keyName, "on") && len(keyName) > 2 {
					return errors.New("智商测试结果包含脚本事件处理器")
				}
				if keyName == "style" && strings.Contains(valueText, "animation") {
					hasAnimation = true
				}
				if keyName == "style" && (strings.Contains(valueText, "url(http") || strings.Contains(valueText, "url(https")) {
					return errors.New("智商测试结果包含外部资源或网络请求")
				}
				if keyName == "http-equiv" && valueText == "refresh" {
					return errors.New("智商测试结果包含外部资源或网络请求")
				}
				if keyName == "src" || keyName == "poster" || keyName == "action" || keyName == "xlink:href" || keyName == "href" {
					if valueText != "" && !strings.HasPrefix(valueText, "#") && !strings.HasPrefix(valueText, "data:") {
						return errors.New("智商测试结果包含外部资源")
					}
				}
				if !more {
					break
				}
			}
		case html.TextToken:
			if styleDepth > 0 || scriptDepth > 0 {
				text := strings.ToLower(string(tokenizer.Raw()))
				if strings.Contains(text, "fetch(") || strings.Contains(text, "xmlhttprequest") || strings.Contains(text, "websocket") || strings.Contains(text, "eventsource") || strings.Contains(text, "http://") || strings.Contains(text, "https://") || strings.Contains(text, "@import") || strings.Contains(text, "url(http") {
					return errors.New("智商测试结果包含外部资源或网络请求")
				}
				if strings.Contains(text, "@keyframes") || strings.Contains(text, "animation:") || strings.Contains(text, "animation-") ||
					strings.Contains(text, "requestanimationframe") || strings.Contains(text, "setinterval(") || strings.Contains(text, "settimeout(") {
					hasAnimation = true
				}
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			switch strings.ToLower(string(name)) {
			case "svg":
				if svgDepth == 0 {
					return errors.New("智商测试结果不完整：SVG 标签未闭合")
				}
				svgDepth--
				hasSVGClose = true
			case "html":
				hasHTMLClose = true
			case "style":
				if styleDepth > 0 {
					styleDepth--
				}
			case "script":
				if scriptDepth > 0 {
					scriptDepth--
				}
			}
		}
	}

parsed:
	if styleDepth != 0 || scriptDepth != 0 {
		return errors.New("智商测试结果不完整：style 或 script 标签未闭合")
	}
	if !hasSVG {
		return errors.New("智商测试结果不完整：缺少 <svg> 元素")
	}
	if !hasHTML {
		return errors.New("智商测试结果不完整：缺少 <html> 文档")
	}
	if !hasSVGClose || svgDepth != 0 {
		return errors.New("智商测试结果不完整：缺少 </svg>")
	}
	if hasHTML && !hasHTMLClose {
		return errors.New("智商测试结果不完整：缺少 </html>")
	}
	if !hasRenderableSVGChild {
		return errors.New("智商测试结果不完整：SVG 中没有可展示内容")
	}
	if !hasAnimation {
		return errors.New("智商测试结果不完整：未检测到 SVG 动画")
	}
	return nil
}

func buildTestRequest(model string, endpointType string, channel *model.Channel, isStream bool, prompt string) dto.Request {
	testResponsesInput := json.RawMessage(`[{"role":"user","content":"hi"}]`)
	if prompt != "" {
		promptData, err := common.Marshal([]map[string]string{{"role": "user", "content": testPromptOrDefault(prompt)}})
		if err == nil {
			testResponsesInput = json.RawMessage(promptData)
		}
	}

	// 根据端点类型构建不同的测试请求
	if endpointType != "" {
		switch constant.EndpointType(endpointType) {
		case constant.EndpointTypeEmbeddings:
			// 返回 EmbeddingRequest
			return &dto.EmbeddingRequest{
				Model: model,
				Input: []any{"hello world"},
			}
		case constant.EndpointTypeImageGeneration:
			// 返回 ImageRequest
			return &dto.ImageRequest{
				Model:  model,
				Prompt: "a cute cat",
				N:      lo.ToPtr(uint(1)),
				Size:   "1024x1024",
			}
		case constant.EndpointTypeJinaRerank:
			// 返回 RerankRequest
			return &dto.RerankRequest{
				Model:     model,
				Query:     "What is Deep Learning?",
				Documents: []any{"Deep Learning is a subset of machine learning.", "Machine learning is a field of artificial intelligence."},
				TopN:      lo.ToPtr(2),
			}
		case constant.EndpointTypeOpenAIResponse:
			// 返回 OpenAIResponsesRequest
			return &dto.OpenAIResponsesRequest{
				Model:           model,
				Input:           testResponsesInput,
				Instructions:    channelIQTestInstructions(prompt),
				MaxOutputTokens: channelIQTestMaxOutputTokens(prompt),
				Stream:          lo.ToPtr(isStream),
			}
		case constant.EndpointTypeOpenAIResponseCompact:
			// 返回 OpenAIResponsesCompactionRequest
			return &dto.OpenAIResponsesCompactionRequest{
				Model: model,
				Input: testResponsesInput,
			}
		case constant.EndpointTypeOpenAIAlphaSearch:
			return &dto.AlphaSearchRequest{
				Model: model,
				RawBody: func() json.RawMessage {
					body, err := common.Marshal(map[string]any{
						"model": model,
						"input": []map[string]string{{"role": "user", "content": testPromptOrDefault(prompt)}},
					})
					if err != nil {
						return nil
					}
					return body
				}(),
			}
		case constant.EndpointTypeAnthropic, constant.EndpointTypeGemini, constant.EndpointTypeOpenAI:
			// 返回 GeneralOpenAIRequest
			maxTokens := lo.ToPtr(uint(16))
			if prompt != "" {
				maxTokens = lo.ToPtr(channelIQTestMaxTokens)
			}
			if constant.EndpointType(endpointType) == constant.EndpointTypeGemini {
				if prompt == "" {
					maxTokens = lo.ToPtr(uint(3000))
				}
			}
			messages := []dto.Message{
				{Role: "user", Content: testPromptOrDefault(prompt)},
			}
			if prompt != "" {
				messages = append([]dto.Message{{Role: "system", Content: channelIQTestOutputContract}}, messages...)
			}
			req := &dto.GeneralOpenAIRequest{
				Model:     model,
				Stream:    lo.ToPtr(isStream),
				Messages:  messages,
				MaxTokens: maxTokens,
			}
			if isStream {
				req.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
			}
			return req
		}
	}

	// 自动检测逻辑（保持原有行为）
	if strings.Contains(strings.ToLower(model), "rerank") {
		return &dto.RerankRequest{
			Model:     model,
			Query:     "What is Deep Learning?",
			Documents: []any{"Deep Learning is a subset of machine learning.", "Machine learning is a field of artificial intelligence."},
			TopN:      lo.ToPtr(2),
		}
	}

	// 先判断是否为 Embedding 模型
	if strings.Contains(strings.ToLower(model), "embedding") ||
		strings.HasPrefix(model, "m3e") ||
		strings.Contains(model, "bge-") {
		// 返回 EmbeddingRequest
		return &dto.EmbeddingRequest{
			Model: model,
			Input: []any{"hello world"},
		}
	}

	// Responses compaction models (must use /v1/responses/compact)
	if strings.HasSuffix(model, ratio_setting.CompactModelSuffix) {
		return &dto.OpenAIResponsesCompactionRequest{
			Model: model,
			Input: testResponsesInput,
		}
	}

	// Responses-only models (e.g. codex series)
	if strings.Contains(strings.ToLower(model), "codex") {
		return &dto.OpenAIResponsesRequest{
			Model:           model,
			Input:           testResponsesInput,
			Instructions:    channelIQTestInstructions(prompt),
			MaxOutputTokens: channelIQTestMaxOutputTokens(prompt),
			Stream:          lo.ToPtr(isStream),
		}
	}

	// Chat/Completion 请求 - 返回 GeneralOpenAIRequest
	messages := []dto.Message{
		{Role: "user", Content: testPromptOrDefault(prompt)},
	}
	if prompt != "" {
		messages = append([]dto.Message{{Role: "system", Content: channelIQTestOutputContract}}, messages...)
	}
	testRequest := &dto.GeneralOpenAIRequest{
		Model:    model,
		Stream:   lo.ToPtr(isStream),
		Messages: messages,
	}
	if isStream {
		testRequest.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}

	if dto.IsOpenAIReasoningOModel(model) {
		if prompt != "" {
			testRequest.MaxCompletionTokens = lo.ToPtr(channelIQTestMaxTokens)
		} else {
			testRequest.MaxCompletionTokens = lo.ToPtr(uint(16))
		}
	} else if strings.Contains(model, "thinking") {
		if !strings.Contains(model, "claude") || prompt != "" {
			if prompt != "" {
				testRequest.MaxTokens = lo.ToPtr(channelIQTestMaxTokens)
			} else {
				testRequest.MaxTokens = lo.ToPtr(uint(50))
			}
		}
	} else if strings.Contains(model, "gemini") {
		maxTokens := uint(3000)
		if prompt != "" {
			maxTokens = channelIQTestMaxTokens
		}
		testRequest.MaxTokens = lo.ToPtr(maxTokens)
	} else {
		if prompt != "" {
			testRequest.MaxTokens = lo.ToPtr(channelIQTestMaxTokens)
		} else {
			testRequest.MaxTokens = lo.ToPtr(uint(16))
		}
	}

	return testRequest
}

func TestChannel(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		channel, err = model.GetChannelById(channelId, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	testModel := c.Query("model")
	endpointType := c.Query("endpoint_type")
	isStream, _ := strconv.ParseBool(c.Query("stream"))
	intelligenceTest, _ := strconv.ParseBool(c.Query("iq_test"))
	testUserID, err := resolveChannelTestUserID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tik := time.Now()
	requestCtx := context.Background()
	if c.Request != nil {
		requestCtx = c.Request.Context()
	}
	var result testResult
	if intelligenceTest {
		if isStream {
			common.ApiError(c, errors.New("智商测试暂不支持流式输出"))
			return
		}
		result = testChannelWithPrompt(requestCtx, channel, testUserID, testModel, endpointType, false, modelTestTokenName, "", channelIQTestPrompt)
	} else {
		result = testChannel(requestCtx, channel, testUserID, testModel, endpointType, isStream)
	}
	if result.localErr != nil {
		if intelligenceTest {
			common.SysError(fmt.Sprintf(
				"channel IQ test failed: channel_id=%d model=%s finish_reason=%q generated_bytes=%d response_bytes=%d reasoning_tokens=%d err=%v",
				channel.Id,
				testModel,
				result.finishReason,
				result.responseBytes,
				result.rawResponseBytes,
				result.reasoningTokens,
				result.localErr,
			))
		}
		resp := gin.H{
			"success": false,
			"message": result.localErr.Error(),
			"time":    0.0,
		}
		if result.newAPIError != nil {
			resp["error_code"] = result.newAPIError.GetErrorCode()
		}
		c.JSON(http.StatusOK, resp)
		return
	}
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	go channel.UpdateResponseTime(milliseconds)
	consumedTime := float64(milliseconds) / 1000.0
	if result.newAPIError != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    result.newAPIError.Error(),
			"time":       consumedTime,
			"error_code": result.newAPIError.GetErrorCode(),
		})
		return
	}
	recoverChannelAfterSuccessfulTest(requestCtx, channel.Id, result.context)
	data := gin.H{
		"response_time": milliseconds,
		"ttft_ms":       result.ttftMs,
	}
	if intelligenceTest {
		data["response"] = result.response
		data["response_bytes"] = result.responseBytes
		data["finish_reason"] = result.finishReason
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"time":    consumedTime,
		"data":    data,
	})
}

// channelTestSummary records the outcome of one channel test cycle so the
// system task can persist a per-run result for history.
type channelTestSummary struct {
	Tested    int `json:"tested"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Disabled  int `json:"disabled"`
}

// 仅由显式单渠道/批量测试成功路径调用，分组监控不参与渠道恢复。
func recoverChannelAfterSuccessfulTest(ctx context.Context, channelID int, testContext *gin.Context) {
	usingKey := ""
	if testContext != nil {
		usingKey = common.GetContextKeyString(testContext, constant.ContextKeyChannelKey)
	}
	if err := service.RecoverChannelAfterTest(ctx, channelID, usingKey); err != nil {
		common.SysError(fmt.Sprintf("recover channel after successful test failed: channel=%d error=%v", channelID, err))
	}
}

// performChannelTests runs the channel test loop synchronously, honoring ctx
// cancellation so a system-task runner that loses its lease stops promptly. When
// report is non-nil it is called after each channel with (processed, total) so
// the system task can surface progress.
func performChannelTests(ctx context.Context, channels []*model.Channel, testUserID int, report func(processed, total int)) channelTestSummary {
	summary := channelTestSummary{}
	var disableThreshold = int64(common.ChannelDisableThreshold * 1000)
	if disableThreshold == 0 {
		disableThreshold = 10000000 // a impossible value
	}

	total := len(channels)
	for index, channel := range channels {
		if ctx != nil && ctx.Err() != nil {
			break
		}
		if report != nil {
			report(index, total) // channels completed before this one
		}
		if channel.Status == common.ChannelStatusManuallyDisabled {
			continue
		}
		if channel.GetTestModel() == "" {
			continue
		}
		isChannelEnabled := channel.Status == common.ChannelStatusEnabled
		tik := time.Now()
		result := testChannel(ctx, channel, testUserID, "", "", shouldUseStreamForAutomaticChannelTest(channel))
		tok := time.Now()
		milliseconds := tok.Sub(tik).Milliseconds()
		if ctx != nil && ctx.Err() != nil {
			break
		}

		summary.Tested++

		shouldBanChannel := false
		newAPIError := result.newAPIError
		// request error disables the channel
		if newAPIError != nil {
			shouldBanChannel = service.ShouldDisableChannel(result.newAPIError)
		}

		// 当错误检查通过，才检查响应时间
		if common.AutomaticDisableChannelEnabled && !shouldBanChannel {
			if milliseconds > disableThreshold {
				err := fmt.Errorf("响应时间 %.2fs 超过阈值 %.2fs", float64(milliseconds)/1000.0, float64(disableThreshold)/1000.0)
				newAPIError = types.NewOpenAIError(err, types.ErrorCodeChannelResponseTimeExceeded, http.StatusRequestTimeout)
				shouldBanChannel = true
			}
		}

		// 恢复只依据原始测试结果，不受业务禁用阈值合成错误影响。
		if result.localErr == nil && result.newAPIError == nil {
			recoverChannelAfterSuccessfulTest(ctx, channel.Id, result.context)
		}
		if newAPIError == nil && result.localErr == nil {
			summary.Succeeded++
		} else {
			summary.Failed++
		}

		// disable channel
		if isChannelEnabled && shouldBanChannel && channel.GetAutoBan() {
			processChannelError(result.context, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(result.context, constant.ContextKeyChannelKey), channel.GetAutoBan()), newAPIError)
			summary.Disabled++
		}

		channel.UpdateResponseTime(milliseconds)
		if common.RequestInterval > 0 {
			if ctx == nil {
				time.Sleep(common.RequestInterval)
			} else {
				select {
				case <-ctx.Done():
					return summary
				case <-time.After(common.RequestInterval):
				}
			}
		}
	}
	if report != nil && (ctx == nil || ctx.Err() == nil) {
		report(total, total) // mark complete only when the full set was tested
	}
	return summary
}

// runChannelTestTask runs one on-demand channel test cycle. Cross-instance
// execution is guarded by the system task per-type lock.
func runChannelTestTask(ctx context.Context, report func(processed, total int)) (channelTestSummary, error) {
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return channelTestSummary{}, err
	}
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		return channelTestSummary{}, err
	}
	summary := performChannelTests(ctx, channels, testUserID, report)
	if ctx == nil || ctx.Err() == nil {
		service.NotifyRootUser(dto.NotifyTypeChannelTest, "通道测试完成", "所有通道测试已完成")
	}
	return summary, nil
}

// TestAllChannels enqueues a channel_test system task instead of running the
// test loop inline. If any channel_test task is already active, the manual run is
// rejected.
func TestAllChannels(c *gin.Context) {
	task, created, err := service.EnqueueSystemTask(model.SystemTaskTypeChannelTest, channelTestTaskPayload{
		Manual: true,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !created {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "已有通道测试任务正在运行或等待中，不能启动本次手动任务",
			"data": gin.H{
				"task_id": task.TaskID,
				"status":  task.Status,
				"type":    task.Type,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"task_id": task.TaskID,
			"status":  task.Status,
		},
	})
}
