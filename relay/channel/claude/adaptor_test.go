package claude

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestToClaudeMessages(t *testing.T) {
	maxOutputTokens := uint(512)
	stream := false
	request := dto.OpenAIResponsesRequest{
		Model:           "claude-sonnet-test",
		Instructions:    mustClaudeRawMessage(t, "Be concise."),
		Input:           mustClaudeRawMessage(t, []map[string]interface{}{{"role": "user", "content": "hello"}}),
		MaxOutputTokens: &maxOutputTokens,
		Stream:          &stream,
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)
	claudeRequest, ok := converted.(*dto.ClaudeRequest)
	require.True(t, ok)
	assert.Equal(t, "claude-sonnet-test", claudeRequest.Model)
	require.NotNil(t, claudeRequest.MaxTokens)
	assert.Equal(t, uint(512), *claudeRequest.MaxTokens)
	assert.Equal(t, "Be concise.", claudeRequest.ParseSystem()[0].GetText())
	require.Len(t, claudeRequest.Messages, 1)
	parts, parseErr := claudeRequest.Messages[0].ParseContent()
	require.NoError(t, parseErr)
	require.Len(t, parts, 1)
	assert.Equal(t, "hello", parts[0].GetText())
}

func TestConvertCodexResponsesRequestUsesClaudeDefaults(t *testing.T) {
	settings := model_setting.GetClaudeSettings()
	previousMaxTokens := settings.DefaultMaxTokens
	settings.DefaultMaxTokens = map[string]int{"default": 8192}
	t.Cleanup(func() { settings.DefaultMaxTokens = previousMaxTokens })

	// Codex can omit max_output_tokens; the host's configured Messages default
	// must still apply after accepting Responses-side client/output hints.
	var request dto.OpenAIResponsesRequest
	require.NoError(t, common.UnmarshalJsonStr(`{
		"model": "claude-opus-5-5",
		"input": [{"role":"user","content":[{"type":"input_text","text":"hello"}]}],
		"include": ["reasoning.encrypted_content"],
		"client_metadata": {"originator":"codex"},
		"reasoning": {"effort":"high","summary":"auto","mode":"standard","context":"current_turn"},
		"stream": true,
		"store": false
	}`, &request))

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model": "claude-opus-5-5",
		"messages": [{"role":"user","content":[{"type":"text","text":"hello"}]}],
		"max_tokens": 8192,
		"stream": true,
		"thinking": {"type":"adaptive","display":"summarized"},
		"output_config": {"effort":"high"}
	}`, string(body))
}

func mustClaudeRawMessage(t *testing.T, value interface{}) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}

func TestConvertResponsesUnsupportedHintsWithClaudeDefaults(t *testing.T) {
	settings := model_setting.GetClaudeSettings()
	previousMaxTokens := settings.DefaultMaxTokens
	settings.DefaultMaxTokens = map[string]int{"default": 8192, "claude-custom": 4096}
	t.Cleanup(func() { settings.DefaultMaxTokens = previousMaxTokens })

	tests := []struct {
		name      string
		model     string
		fields    string
		maxTokens uint
	}{
		{name: "omitted limit", model: "claude-test", maxTokens: 8192},
		{name: "null limit", model: "claude-test", fields: `,"max_output_tokens":null`, maxTokens: 8192},
		{name: "model default", model: "claude-custom", maxTokens: 4096},
		{name: "explicit limit", model: "claude-test", fields: `,"max_output_tokens":256`, maxTokens: 256},
		{name: "explicit zero", model: "claude-test", fields: `,"max_output_tokens":0`, maxTokens: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := fmt.Sprintf(`{
				"model": %q,
				"input": "hello",
				"previous_response_id": "resp_not_retrievable",
				"include": ["reasoning.encrypted_content", "file_search_call.results"],
				"client_metadata": [],
				"parallel_tool_calls": "false",
				"tool_choice": {},
				"stream": false%s
			}`, tt.model, tt.fields)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(payload))
			c.Request.Header.Set("Content-Type", "application/json")
			request, err := helper.GetAndValidateResponsesRequest(c)
			require.NoError(t, err)
			converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, &relaycommon.RelayInfo{}, *request)
			require.NoError(t, err)
			body, err := common.Marshal(converted)
			require.NoError(t, err)
			assert.JSONEq(t, fmt.Sprintf(`{
				"model": %q,
				"messages": [{"role":"user","content":[{"type":"text","text":"hello"}]}],
				"max_tokens": %d,
				"stream": false
			}`, tt.model, tt.maxTokens), string(body))
		})
	}
}

func TestConvertResponsesUsesMappedClaudeThinkingCapabilities(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("model_mapping", `{"custom-reasoning-model":"claude-opus-5-5"}`)
	request := dto.OpenAIResponsesRequest{
		Model:           "custom-reasoning-model",
		Input:           mustClaudeRawMessage(t, "hello"),
		MaxOutputTokens: common.GetPointer(uint(8192)),
		Reasoning:       &dto.Reasoning{Effort: "xhigh"},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: request.Model,
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: request.Model},
	}
	require.NoError(t, helper.ModelMappedHelper(c, info, &request))
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, request)
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.JSONEq(t, `{
  "model":"claude-opus-5-5",
  "messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],
  "max_tokens":8192,
  "thinking":{"type":"adaptive"},
  "output_config":{"effort":"xhigh"}
 }`, string(body))
}
