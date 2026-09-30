package controller

import (
	"io"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestExtractChannelTestText(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "chat completion string content",
			response: `{"choices":[{"message":{"content":"<html>chat</html>"}}]}`,
			want:     "<html>chat</html>",
		},
		{
			name:     "chat completion content blocks",
			response: `{"choices":[{"message":{"content":[{"type":"text","text":"<html>"},{"type":"text","text":"blocks</html>"}]}}]}`,
			want:     "<html>blocks</html>",
		},
		{
			name:     "responses output text",
			response: `{"output_text":"<html>responses</html>"}`,
			want:     "<html>responses</html>",
		},
		{
			name:     "responses output blocks",
			response: `{"output":[{"content":[{"type":"output_text","text":"<html>responses blocks</html>"}]}]}`,
			want:     "<html>responses blocks</html>",
		},
		{
			name:     "claude content blocks",
			response: `{"content":[{"type":"text","text":"<html>claude</html>"}]}`,
			want:     "<html>claude</html>",
		},
		{
			name:     "gemini parts",
			response: `{"candidates":[{"content":{"parts":[{"text":"<html>gemini</html>"}]}}]}`,
			want:     "<html>gemini</html>",
		},
		{
			name:     "plain text completion",
			response: `{"choices":[{"text":"<html>completion</html>"}]}`,
			want:     "<html>completion</html>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, extractChannelTestText([]byte(test.response)))
		})
	}
}

func TestNormalizeChannelIQTestHTML(t *testing.T) {
	assert.Equal(t, "<html>pelican</html>", normalizeChannelIQTestHTML("```html\n<html>pelican</html>\n```"))
	assert.Equal(t, "<html>pelican</html>", normalizeChannelIQTestHTML(" <html>pelican</html> "))
	assert.Equal(t, "<html><svg></svg></html>", normalizeChannelIQTestHTML("Here is the file:\n```html\n<html><svg></svg></html>\n```\nHope it helps."))
	assert.Contains(t, normalizeChannelIQTestHTML("```html\n<html><svg></svg></html>"), "```")
	assert.Equal(t, "<html><svg><circle /></svg></html>", normalizeChannelIQTestHTML("Explanation <svgx>ignored</svgx>\n<html><svg><circle /></svg></html>"))
	assert.Empty(t, extractChannelTestText([]byte(`{"choices":[{"message":{"content":" "}}]}`)))
	assert.Empty(t, extractChannelTestText([]byte(`{"choices":[{"message":{"content":"","reasoning_content":"thinking only"}}]}`)))
}

func TestExtractChannelIQTestFinishReason(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{name: "chat length", response: `{"choices":[{"finish_reason":"length"}]}`, want: "length"},
		{name: "claude max tokens", response: `{"stop_reason":"max_tokens"}`, want: "max_tokens"},
		{name: "responses incomplete", response: `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`, want: "max_output_tokens"},
		{name: "gemini max tokens", response: `{"candidates":[{"finishReason":"MAX_TOKENS"}]}`, want: "MAX_TOKENS"},
		{name: "responses completed", response: `{"status":"completed"}`, want: "completed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, extractChannelIQTestFinishReason([]byte(test.response)))
		})
	}
}

func TestValidateChannelIQTestHTML(t *testing.T) {
	complete := "<!doctype html><html><head><style>@keyframes ride{to{transform:rotate(360deg)}}</style></head><body><svg><circle /><animate attributeName=\"x\" values=\"0;10\" dur=\"1s\" repeatCount=\"indefinite\" /></svg></body></html>"
	assert.NoError(t, validateChannelIQTestHTML(complete, "stop"))

	largePrefix := `<html><body><svg data-padding="`
	largeSuffix := `"><circle /><animate attributeName="x" values="0;10" dur="1s" repeatCount="indefinite" /></svg></body></html>`
	largeDocument := largePrefix + strings.Repeat("x", (256<<10)-len(largePrefix)-len(largeSuffix)) + largeSuffix
	assert.Greater(t, len(largeDocument), 128<<10)
	assert.NoError(t, validateChannelIQTestHTML(largeDocument, "stop"))

	assert.ErrorContains(t, validateChannelIQTestHTML(complete, "length"), "生成被截断")
	assert.ErrorContains(t, validateChannelIQTestHTML("<html><svg>", "stop"), "缺少 </svg>")
	assert.ErrorContains(t, validateChannelIQTestHTML("<html><svg></svg>", "stop"), "缺少 </html>")
	assert.ErrorContains(t, validateChannelIQTestHTML("```html\n"+complete, "stop"), "代码围栏未闭合")
	assert.ErrorContains(t, validateChannelIQTestHTML("", "incomplete"), "生成被截断")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<html><body><svg><circle /></svg></body></html>`, "stop"), "未检测到 SVG 动画")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<html><body><svg><circle /><animate /></svg></body></html><script>fetch('https://example.com')</script>`, "stop"), "外部资源")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<html><body><svg onclick="fetch('https://example.com')"><circle /><animate /></svg></body></html>`, "stop"), "事件处理器")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<html><body><svg><circle /><animate /></svg><iframe srcdoc="<p>embedded</p>"></iframe></body></html>`, "stop"), "嵌入内容")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<html><head><meta http-equiv="refresh" content="0;url=https://example.com"></head><body><svg><circle /><animate /></svg></body></html>`, "stop"), "外部资源")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<svg><circle /><animate /></svg>`, "stop"), "缺少 <html> 文档")
	assert.ErrorContains(t, validateChannelIQTestHTML(`<html><body><svgx><circle /></svgx></body></html>`, "stop"), "缺少 <svg> 元素")
}

func TestBuildChannelIQTestRequest(t *testing.T) {
	request := buildTestRequest("gpt-4o", "", nil, false, channelIQTestPrompt)
	chatRequest, ok := request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatRequest.Messages, 2)
	assert.Equal(t, "system", chatRequest.Messages[0].Role)
	assert.Equal(t, channelIQTestOutputContract, chatRequest.Messages[0].Content)
	assert.Equal(t, "user", chatRequest.Messages[1].Role)
	assert.Equal(t, channelIQTestPrompt, chatRequest.Messages[1].Content)
	assert.Nil(t, chatRequest.MaxTokens)

	responsesRequest, ok := buildTestRequest("codex-mini", "", nil, false, channelIQTestPrompt).(*dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Contains(t, string(responsesRequest.Input), channelIQTestPrompt)
	assert.Nil(t, responsesRequest.MaxOutputTokens)
}

func TestBuildChannelAlphaSearchTestRequestBodyPreservesInput(t *testing.T) {
	request := buildTestRequest("gpt-5.1", string(constant.EndpointTypeOpenAIAlphaSearch), nil, false, "")
	alphaRequest, ok := request.(*dto.AlphaSearchRequest)
	require.True(t, ok)

	body, err := buildChannelAlphaSearchRequestBody(alphaRequest.RawBody, "gpt-5.1", "mapped-search-model")
	require.NoError(t, err)
	assert.Contains(t, string(body), `"input"`)
	assert.Contains(t, string(body), `"mapped-search-model"`)
}

func TestDisableChannelIQTestThinking(t *testing.T) {
	maxTokens := uint(4096)
	request := &dto.ClaudeRequest{MaxTokens: &maxTokens}

	disableChannelIQTestThinking(request)
	assert.Nil(t, request.Thinking)
	assert.Equal(t, uint(4096), *request.MaxTokens)
	assert.Nil(t, request.OutputConfig)
}

func TestFinalizeChannelIQTestRequestForClaudeRoutes(t *testing.T) {
	deepseekBaseURL := "https://api.deepseek.com/anthropic"
	tests := []struct {
		name    string
		channel *model.Channel
	}{
		{
			name: "advanced custom deepseek anthropic route",
			channel: &model.Channel{
				Type:    constant.ChannelTypeAdvancedCustom,
				BaseURL: &deepseekBaseURL,
			},
		},
		{
			name: "native deepseek channel",
			channel: &model.Channel{
				Type: constant.ChannelTypeDeepSeek,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(`{"model":"deepseek-flash","max_tokens":2048,"max_completion_tokens":99,"stream":true,"thinking":{"type":"enabled","budget_tokens":4096},"output_config":{"effort":"high"},"reasoning_effort":"high"}`)
			normalized, err := normalizeChannelIQTestControls(input)
			require.NoError(t, err)
			output, err := finalizeChannelIQTestRequest(normalized, &dto.ClaudeRequest{}, test.channel, "deepseek-flash")
			require.NoError(t, err)

			assert.Equal(t, uint64(2048), gjson.GetBytes(output, "max_tokens").Uint())
			assert.True(t, gjson.GetBytes(output, "stream").Exists())
			assert.False(t, gjson.GetBytes(output, "stream").Bool())
			assert.Equal(t, "disabled", gjson.GetBytes(output, "thinking.type").String())
			assert.False(t, gjson.GetBytes(output, "thinking.budget_tokens").Exists())
			assert.False(t, gjson.GetBytes(output, "output_config").Exists())
			assert.False(t, gjson.GetBytes(output, "max_completion_tokens").Exists())
			assert.False(t, gjson.GetBytes(output, "reasoning_effort").Exists())
		})
	}
}

func TestFinalizeChannelIQTestRequestDisablesDeepSeekOpenAIThinking(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeDeepSeek}
	input := []byte(`{"max_tokens":2048,"stream":true,"thinking":{"type":"enabled"},"reasoning_effort":"high"}`)
	normalized, err := normalizeChannelIQTestControls(input)
	require.NoError(t, err)
	output, err := finalizeChannelIQTestRequest(normalized, &dto.GeneralOpenAIRequest{}, channel, "deepseek-flash")
	require.NoError(t, err)

	assert.Equal(t, uint64(2048), gjson.GetBytes(output, "max_tokens").Uint())
	assert.True(t, gjson.GetBytes(output, "stream").Exists())
	assert.False(t, gjson.GetBytes(output, "stream").Bool())
	assert.Equal(t, "disabled", gjson.GetBytes(output, "thinking.type").String())
	assert.False(t, gjson.GetBytes(output, "reasoning_effort").Exists())
}

func TestChannelIQTestRejectsReasoningOnlyTruncatedResponse(t *testing.T) {
	response := []byte(`{"choices":[{"message":{"content":"","reasoning_content":"thinking only"},"finish_reason":"length"}]}`)
	finishReason := extractChannelIQTestFinishReason(response)
	generatedHTML := normalizeChannelIQTestHTML(extractChannelTestText(response))

	assert.Empty(t, generatedHTML)
	assert.Equal(t, "length", finishReason)
	assert.ErrorContains(t, validateChannelIQTestHTML(generatedHTML, finishReason), "生成被截断")
}

func TestReadLimitedChannelIQTestResponseBody(t *testing.T) {
	completeResponse := strings.Repeat("x", int(channelIQTestMaxResponseBytes))
	body, err := readLimitedTestResponseBody(io.NopCloser(strings.NewReader(completeResponse)), int64(channelIQTestMaxResponseBytes))
	require.NoError(t, err)
	assert.Len(t, body, channelIQTestMaxResponseBytes)

	tooLargeResponse := completeResponse + "x"
	_, err = readLimitedTestResponseBody(io.NopCloser(strings.NewReader(tooLargeResponse)), int64(channelIQTestMaxResponseBytes))
	assert.ErrorContains(t, err, "exceeds")
}

func TestNormalizeChannelIQTestControls(t *testing.T) {
	input := `{"max_tokens":100000,"stream":true,"generation_config":{"max_output_tokens":9000,"thinking_config":{"thinking_budget":1000}},"extra":{"maxCompletionTokens":100000,"reasoning_effort":"high"}}`
	output, err := normalizeChannelIQTestControls([]byte(input))
	require.NoError(t, err)
	assert.NotContains(t, string(output), "thinking_config")
	assert.NotContains(t, string(output), "reasoning_effort")
	assert.Equal(t, uint64(100000), gjson.GetBytes(output, "max_tokens").Uint())
	assert.Equal(t, uint64(9000), gjson.GetBytes(output, "generation_config.max_output_tokens").Uint())
	assert.Equal(t, uint64(100000), gjson.GetBytes(output, "extra.maxCompletionTokens").Uint())
	assert.Contains(t, string(output), `"stream":false`)
}

func TestDisableChannelIQTestGeminiThinking(t *testing.T) {
	budget := 1024
	request := &dto.GeminiChatRequest{
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ThinkingConfig: &dto.GeminiThinkingConfig{ThinkingBudget: &budget},
		},
	}

	disableChannelIQTestThinking(request)
	assert.Nil(t, request.GenerationConfig.ThinkingConfig)
}
