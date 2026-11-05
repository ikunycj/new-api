package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesHandlerCountsOutputCallsNotDeclarations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	operation_setting.SetToolPriceForTest("priced_fn", 5.0)
	t.Cleanup(func() {
		operation_setting.DeleteToolPriceForTest("priced_fn")
	})

	body, err := common.Marshal(dto.OpenAIResponsesResponse{
		Tools: []map[string]any{
			{"type": "web_search_preview"},
			{"type": "file_search"},
		},
		Output: []dto.ResponsesOutput{
			{Type: dto.BuildInCallWebSearchCall},
			{Type: dto.BuildInCallWebSearchCall},
			{Type: dto.BuildInCallFunctionCall, Name: "priced_fn"},
			{Type: dto.BuildInCallFunctionCall, Name: "unpriced_fn"},
		},
		Usage: &dto.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.1",
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {ToolName: dto.BuildInToolWebSearchPreview, CallCount: 0},
				dto.BuildInToolFileSearch:       {ToolName: dto.BuildInToolFileSearch, CallCount: 0},
			},
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	usage, apiErr := OaiResponsesHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 2, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview].CallCount)
	assert.Equal(t, 0, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch].CallCount)
	require.Contains(t, info.ResponsesUsageInfo.BuiltInTools, "priced_fn")
	assert.Equal(t, 1, info.ResponsesUsageInfo.BuiltInTools["priced_fn"].CallCount)
	assert.NotContains(t, info.ResponsesUsageInfo.BuiltInTools, "unpriced_fn")
}

func TestOaiResponsesHandlerDeclaredToolsWithoutOutputCountZero(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body, err := common.Marshal(dto.OpenAIResponsesResponse{
		Tools: []map[string]any{
			{"type": "web_search_preview"},
			{"type": "file_search"},
		},
		Output: []dto.ResponsesOutput{
			{Type: "message", Role: "assistant"},
		},
		Usage: &dto.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.1",
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {ToolName: dto.BuildInToolWebSearchPreview, CallCount: 0},
				dto.BuildInToolFileSearch:       {ToolName: dto.BuildInToolFileSearch, CallCount: 0},
			},
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	_, apiErr := OaiResponsesHandler(c, info, resp)
	require.Nil(t, apiErr)
	assert.Equal(t, 0, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview].CallCount)
	assert.Equal(t, 0, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch].CallCount)
}

func TestOaiResponsesHandlerCountsCompletedImageGenerationOutputs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body, err := common.Marshal(dto.OpenAIResponsesResponse{
		Status: []byte(`"completed"`),
		Output: []dto.ResponsesOutput{
			{
				Type:   dto.ResponsesOutputTypeImageGenerationCall,
				ID:     "img_1",
				Status: "completed",
				Result: "base64-a",
			},
			{
				Type:   dto.ResponsesOutputTypeImageGenerationCall,
				ID:     "img_2",
				Status: "completed",
				Result: "base64-b",
			},
			{
				Type:   dto.ResponsesOutputTypeImageGenerationCall,
				ID:     "img_empty",
				Status: "completed",
				Result: "",
			},
		},
		Usage: &dto.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := &relaycommon.RelayInfo{OriginModelName: "gpt-5.1"}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	_, apiErr := OaiResponsesHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.Contains(t, info.ResponsesUsageInfo.BuiltInTools, dto.BuildInToolImageGeneration)
	assert.Equal(t, 2, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolImageGeneration].CallCount)
	assert.False(t, c.GetBool("image_generation_call"))
}

func TestOaiResponsesHandlerIncompleteStatusCommitsZeroImageGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body, err := common.Marshal(dto.OpenAIResponsesResponse{
		Status: []byte(`"incomplete"`),
		Output: []dto.ResponsesOutput{
			{
				Type:   dto.ResponsesOutputTypeImageGenerationCall,
				ID:     "img_1",
				Status: "completed",
				Result: "base64-a",
			},
		},
		Usage: &dto.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.1",
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolImageGeneration: {ToolName: dto.BuildInToolImageGeneration, CallCount: 0},
			},
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	_, apiErr := OaiResponsesHandler(c, info, resp)
	require.Nil(t, apiErr)
	assert.Equal(t, 0, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolImageGeneration].CallCount)
}

func runResponsesImageBillingStream(t *testing.T, events ...string) *relaycommon.RelayInfo {
	t.Helper()
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	var body strings.Builder
	for _, event := range events {
		body.WriteString("data: ")
		body.WriteString(event)
		body.WriteString("\n\n")
	}
	body.WriteString("data: [DONE]\n\n")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "responses-image-billing-test")
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.1",
		DisablePing:     true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.1",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body.String())),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}

	_, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, info.ResponsesUsageInfo)
	require.Contains(t, info.ResponsesUsageInfo.BuiltInTools, dto.BuildInToolImageGeneration)
	return info
}

func TestOaiResponsesStreamHandlerDeduplicatesCompletedImageOutput(t *testing.T) {
	item := `{"type":"image_generation_call","id":"img_1","call_id":"call_1","status":"completed","result":"base64-a"}`
	info := runResponsesImageBillingStream(
		t,
		`{"type":"response.output_item.done","output_index":0,"item":`+item+`}`,
		`{"type":"response.completed","response":{"status":"completed","output":[`+item+`],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}`,
	)

	assert.Equal(t, 1, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolImageGeneration].CallCount)
}

func TestOaiResponsesStreamHandlerDiscardsImageOutputOnIncomplete(t *testing.T) {
	info := runResponsesImageBillingStream(
		t,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"image_generation_call","id":"img_1","status":"completed","result":"base64-a"}}`,
		`{"type":"response.incomplete","response":{"status":"incomplete"}}`,
	)

	assert.Equal(t, 0, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolImageGeneration].CallCount)
}

func TestOaiResponsesStreamHandlerDoesNotCountPartialImageEvent(t *testing.T) {
	info := runResponsesImageBillingStream(
		t,
		`{"type":"response.image_generation_call.partial_image","output_index":0,"partial_image_b64":"partial-bytes"}`,
		`{"type":"response.completed","response":{"status":"completed","output":[]}}`,
	)

	assert.Equal(t, 0, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolImageGeneration].CallCount)
}

func TestOpenAIHandlersPreserveCacheUsageJSONAndMergedSnapshots(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	const detailsJSON = `{"cached_tokens":3,"cached_creation_tokens":8,"cache_write_tokens":10,"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7},"text_tokens":20,"audio_tokens":2,"image_tokens":1}`
	responsesUsage := `{"input_tokens":30,"output_tokens":2,"total_tokens":32,"input_tokens_details":` + detailsJSON + `}`
	chatUsage := `{"prompt_tokens":30,"completion_tokens":2,"total_tokens":32,"prompt_tokens_details":` + detailsJSON + `}`
	responsesBody := `{"id":"resp_cache","model":"gpt-test","status":"completed","output":[],"usage":` + responsesUsage + `}`
	chatBody := `{"id":"chatcmpl_cache","model":"gpt-test","choices":[],"usage":` + chatUsage + `}`
	imageBody := `{"created":1710000000,"data":[{"b64_json":"image"}],"usage":` + responsesUsage + `}`
	responsesStream := "data: " + `{"type":"response.completed","response":` + responsesBody + `}` + "\n\ndata: " + `{"type":"response.done","response":{"usage":{"input_tokens_details":{"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":0}}}}}` + "\n\ndata: [DONE]\n\n"
	chatStream := "data: " + `{"id":"chatcmpl_cache","model":"gpt-test","choices":[],"usage":` + chatUsage + `}` + "\n\ndata: " + `{"id":"chatcmpl_cache","model":"gpt-test","choices":[],"usage":{"completion_tokens":2,"total_tokens":32,"prompt_tokens_details":{"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":0}}}}` + "\n\ndata: [DONE]\n\n"
	imageStream := "data: " + `{"type":"image_generation.completed","b64_json":"image","usage":` + responsesUsage + `}` + "\n\ndata: " + `{"usage":{"output_tokens":2,"total_tokens":32,"input_tokens_details":{"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":0}}}}` + "\n\ndata: [DONE]\n\n"
	tests := []struct {
		name    string
		body    string
		stream  bool
		handler func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError)
	}{
		{name: "responses JSON", body: responsesBody, handler: OaiResponsesHandler},
		{name: "responses SSE", body: responsesStream, stream: true, handler: OaiResponsesStreamHandler},
		{name: "responses compaction", body: responsesBody, handler: func(c *gin.Context, _ *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
			return OaiResponsesCompactionHandler(c, resp)
		}},
		{name: "chat JSON forced format", body: chatBody, handler: OpenaiHandler},
		{name: "chat SSE forced format", body: chatStream, stream: true, handler: OaiStreamHandler},
		{name: "chat to responses JSON", body: chatBody, handler: OaiChatToResponsesHandler},
		{name: "chat to responses SSE", body: chatStream, stream: true, handler: OaiChatToResponsesStreamHandler},
		{name: "responses to chat JSON", body: responsesBody, handler: OaiResponsesToChatHandler},
		{name: "responses to chat SSE", body: responsesStream, stream: true, handler: OaiResponsesToChatStreamHandler},
		{name: "responses to buffered chat", body: responsesStream, handler: OaiResponsesToChatBufferedStreamHandler},
		{name: "image JSON", body: imageBody, handler: OpenaiImageHandler},
		{name: "image SSE", body: imageStream, stream: true, handler: OpenaiImageStreamHandler},
		{name: "image JSON converted to SSE", body: imageBody, handler: OpenaiImageStreamHandler},
	}
	var expected dto.InputTokenDetails
	require.NoError(t, common.UnmarshalJsonStr(detailsJSON, &expected))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			c.Set(common.RequestIdKey, "cache-usage-adaptation-test")
			info := &relaycommon.RelayInfo{
				ChannelMeta:        &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: "gpt-test"},
				IsStream:           tt.stream,
				RelayFormat:        types.RelayFormatOpenAI,
				ShouldIncludeUsage: true,
				DisablePing:        true,
			}
			info.ChannelSetting.ForceFormat = true
			contentType := "application/json"
			if tt.stream || tt.name == "responses to buffered chat" {
				contentType = "text/event-stream"
			}
			resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(tt.body)), Header: http.Header{"Content-Type": []string{contentType}}}
			usage, apiErr := tt.handler(c, info, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 30, usage.PromptTokens)
			assert.Equal(t, 2, usage.CompletionTokens)
			assert.Equal(t, 32, usage.TotalTokens)
			// Chat-to-Responses exposes the Responses details shape; other
			// handlers return canonical prompt details.
			details := &usage.PromptTokensDetails
			if strings.HasPrefix(tt.name, "chat to responses") {
				details = usage.InputTokensDetails
			}
			require.NotNil(t, details)
			assert.Equal(t, expected, *details)
			assert.Equal(t, 12, details.CacheCreationTokensTotal())
			assert.Equal(t, 7, details.CacheWrite12hTokens())
			assert.Contains(t, w.Body.String(), `"cache_creation_input_tokens":12`)
			assert.Contains(t, w.Body.String(), `"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7}`)
			if usage.BillingUsage != nil {
				require.NotNil(t, usage.BillingUsage.OpenAIUsage)
				assert.Equal(t, dto.BillingUsageSemanticOpenAI, usage.BillingUsage.Semantic)
				snapshot := usage.BillingUsage.OpenAIUsage.PromptTokensDetails
				if usage.BillingUsage.OpenAIUsage.InputTokensDetails != nil {
					snapshot = *usage.BillingUsage.OpenAIUsage.InputTokensDetails
				}
				assert.Equal(t, expected, snapshot)
				require.NotSame(t, details.CacheWriteTokensDetails, snapshot.CacheWriteTokensDetails)
				details.CacheWriteTokensDetails.TTL12hTokens = 99
				assert.Equal(t, 7, snapshot.CacheWrite12hTokens())
			} else if usage.InputTokensDetails != nil {
				require.NotSame(t, details.CacheWriteTokensDetails, usage.InputTokensDetails.CacheWriteTokensDetails)
				details.CacheWriteTokensDetails.TTL12hTokens = 99
				assert.Equal(t, 7, usage.InputTokensDetails.CacheWrite12hTokens())
			}
		})
	}
}

func TestOpenAIConversionEstimatesRetainCacheOnlyUsage(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	const detailsJSON = `{"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7}}`
	chat := `{"choices":[],"usage":{"prompt_tokens_details":` + detailsJSON + `}}`
	responses := `{"status":"completed","output":[],"usage":{"input_tokens_details":` + detailsJSON + `}}`
	chatStream := "data: " + chat + "\n\ndata: [DONE]\n\n"
	responsesStream := "data: " + `{"type":"response.completed","response":` + responses + `}` + "\n\ndata: [DONE]\n\n"
	tests := []struct {
		name    string
		body    string
		stream  bool
		handler func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError)
	}{
		{name: "chat to responses JSON", body: chat, handler: OaiChatToResponsesHandler},
		{name: "chat to responses SSE", body: chatStream, stream: true, handler: OaiChatToResponsesStreamHandler},
		{name: "responses to chat JSON", body: responses, handler: OaiResponsesToChatHandler},
		{name: "responses to chat SSE", body: responsesStream, stream: true, handler: OaiResponsesToChatStreamHandler},
		{name: "responses to buffered chat", body: responsesStream, handler: OaiResponsesToChatBufferedStreamHandler},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w, resp, info := newResponsesChatTestContext(t, tt.body, tt.stream)
			usage, apiErr := tt.handler(c, info, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			details := &usage.PromptTokensDetails
			if strings.HasPrefix(tt.name, "chat to responses") {
				details = usage.InputTokensDetails
			}
			require.NotNil(t, details)
			assert.Equal(t, 12, details.CacheCreationInputTokens)
			assert.Equal(t, 7, details.CacheWrite12hTokens())
			assert.Contains(t, w.Body.String(), `"cache_creation_input_tokens":12`)
			assert.Contains(t, w.Body.String(), `"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7}`)
			require.NotNil(t, usage.BillingUsage)
			require.NotNil(t, usage.BillingUsage.OpenAIUsage)
		})
	}
}

func TestSparseCacheUsageEstimatesRemainBillable(t *testing.T) {
	for _, responses := range []bool{false, true} {
		native := &dto.Usage{PromptTokensDetails: dto.InputTokenDetails{CacheCreationInputTokens: 100}}
		var current *dto.Usage
		if responses {
			native.InputTokensDetails = &dto.InputTokenDetails{CacheCreationInputTokens: 100}
			current = relayconvert.UsageFromResponsesUsage(native)
		} else {
			current = relayconvert.UsageFromChatUsage(native)
		}
		require.NotNil(t, current.BillingUsage)
		merged := mergeEstimatedOpenAIUsage(current, &dto.Usage{PromptTokens: 1000, CompletionTokens: 100, TotalTokens: 1100})
		require.True(t, merged.BillingUsage.Estimated)
		var canonical *dto.Usage
		if responses {
			canonical = relayconvert.UsageFromResponsesUsage(merged.BillingUsage.OpenAIUsage)
		} else {
			canonical = merged.BillingUsage.OpenAIUsage
		}
		assert.Equal(t, 100, canonical.PromptTokensDetails.CacheCreationTokensTotal())
		assert.Zero(t, native.InputTokens)
		const expression = `tier("base", p * 2 + c * 12)`
		info := &relaycommon.RelayInfo{TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr", ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), GroupRatio: 1, QuotaPerUnit: 500000, BillingUSDToCNYRate: 1,
		}}
		ok, quota, result := service.TryTieredSettle(info, service.BuildTieredTokenParams(canonical, false, billingexpr.UsedVars(expression)))
		require.True(t, ok)
		require.NotNil(t, result)
		assert.Equal(t, 1600, quota)
	}
}

func TestOpenaiHandlerEstimatedPromptRetainsCacheFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"choices":[],"usage":{"completion_tokens":2,"prompt_tokens_details":{"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7}}}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, RelayFormat: types.RelayFormatOpenAI}
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": []string{"application/json"}}}
	usage, apiErr := OpenaiHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 2, usage.CompletionTokens)
	assert.Equal(t, 12, usage.PromptTokensDetails.CacheCreationTokensTotal())
	assert.Equal(t, 7, usage.PromptTokensDetails.CacheWrite12hTokens())
	var wire dto.SimpleResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &wire))
	assert.Equal(t, usage.PromptTokensDetails, wire.PromptTokensDetails)
}
