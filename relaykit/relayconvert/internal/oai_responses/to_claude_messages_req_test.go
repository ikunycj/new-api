package oairesponses

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesRequestToClaudeMessagesCodexFields(t *testing.T) {
	var req dto.OpenAIResponsesRequest
	require.NoError(t, kitutil.UnmarshalJsonStr(`{
		"model": "claude-test",
		"instructions": "Use the available tools.",
		"input": [
			{"role": "developer", "content": [{"type": "input_text", "text": "Be concise."}]},
			{"role": "user", "content": [{"type": "input_text", "text": "Read the file."}]},
			{"type": "function_call", "call_id": "call_read", "name": "read_file", "arguments": "{\"path\":\"README.md\"}"},
			{"type": "function_call_output", "call_id": "call_read", "output": "File contents."}
		],
		"tools": [{"type": "function", "name": "read_file", "parameters": {
			"type": "object", "properties": {"path": {"type": "string"}}, "required": ["path"]
		}}],
		"tool_choice": "auto",
		"parallel_tool_calls": false,
		"max_output_tokens": 8192,
		"stream": true,
		"store": false,
		"prompt_cache_key": "test-session",
		"include": ["reasoning.encrypted_content", "file_search_call.results"],
		"client_metadata": ["unsupported metadata shape"],
		"previous_response_id": "resp_not_retrievable",
		"conversation": "conv_not_retrievable",
		"prompt": {"id": "pmpt_not_retrievable"},
		"reasoning": {"effort": "high", "summary": "auto", "mode": {}, "context": "all_turns"}
	}`, &req))
	before, err := kitutil.Marshal(req)
	require.NoError(t, err)

	got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, &req)
	require.NoError(t, err)
	body, err := kitutil.Marshal(got)
	require.NoError(t, err)
	// Assert the wire body, including the absence of Responses-only hints.
	assert.JSONEq(t, `{
		"model": "claude-test",
		"system": [{"type": "text", "text": "Use the available tools."}, {"type": "text", "text": "Be concise."}],
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "Read the file."}]},
			{"role": "assistant", "content": [{"type": "tool_use", "id": "call_read", "name": "read_file", "input": {"path": "README.md"}}]},
			{"role": "user", "content": [{"type": "tool_result", "tool_use_id": "call_read", "content": "File contents."}]}
		],
		"tools": [{"name": "read_file", "input_schema": {"type": "object", "properties": {"path": {"type": "string"}}, "required": ["path"]}}],
		"tool_choice": {"type": "auto", "disable_parallel_tool_use": true},
		"max_tokens": 8192,
		"stream": true,
		"thinking": {"type": "enabled", "budget_tokens": 4096}
	}`, string(body))
	after, err := kitutil.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after), "conversion must not strip fields from the caller's request")
}

func TestResponsesRequestToClaudeMessagesIgnoresUnsupportedHints(t *testing.T) {
	tests := []struct {
		name   string
		fields string
	}{
		{name: "absent", fields: `{}`},
		{name: "null", fields: `{"include":null,"client_metadata":null,"reasoning":{"mode":null,"context":null}}`},
		{name: "empty include", fields: `{"include":[]}`},
		{name: "encrypted content", fields: `{"include":["reasoning.encrypted_content"]}`},
		{name: "client metadata", fields: `{"client_metadata":{"originator":"codex"}}`},
		{name: "summary auto", fields: `{"reasoning":{"summary":"auto"}}`},
		{name: "summary concise", fields: `{"reasoning":{"summary":"concise"}}`},
		{name: "summary detailed", fields: `{"reasoning":{"summary":"detailed"}}`},
		{name: "mode", fields: `{"reasoning":{"mode":"standard"}}`},
		{name: "auto context", fields: `{"reasoning":{"context":"auto"}}`},
		{name: "current turn context", fields: `{"reasoning":{"context":"current_turn"}}`},
		{name: "cache key", fields: `{"prompt_cache_key":"test-session"}`},
		{name: "conversation", fields: `{"conversation":"conv_test"}`},
		{name: "previous response", fields: `{"previous_response_id":"resp_test"}`},
		{name: "prompt", fields: `{"prompt":{"id":"pmpt_test"}}`},
		{name: "context management", fields: `{"context_management":[{"type":"compaction","compact_threshold":1000}]}`},
		{name: "moderation", fields: `{"moderation":{}}`},
		{name: "cache options", fields: `{"prompt_cache_options":{"retention":"24h"}}`},
		{name: "truncation", fields: `{"truncation":"auto"}`},
		{name: "preset", fields: `{"preset":"test"}`},
		{name: "max tool calls", fields: `{"max_tool_calls":1}`},
		{name: "zero max tool calls", fields: `{"max_tool_calls":0}`},
		{name: "parallel calls wrong type", fields: `{"parallel_tool_calls":"false"}`},
		{name: "cache key wrong type", fields: `{"prompt_cache_key":123}`},
		{name: "metadata wrong type", fields: `{"client_metadata":[]}`},
		{name: "include wrong type", fields: `{"include":"reasoning.encrypted_content"}`},
		{name: "include wrong item type", fields: `{"include":[123]}`},
		{name: "include null item", fields: `{"include":[null]}`},
		{name: "unmapped include", fields: `{"include":["file_search_call.results"]}`},
		{name: "mixed include", fields: `{"include":["reasoning.encrypted_content","file_search_call.results"]}`},
		{name: "mode wrong type", fields: `{"reasoning":{"mode":{}}}`},
		{name: "all turns context", fields: `{"reasoning":{"context":"all_turns"}}`},
		{name: "unknown context", fields: `{"reasoning":{"context":"unknown"}}`},
		{name: "empty context", fields: `{"reasoning":{"context":""}}`},
		{name: "context wrong type", fields: `{"reasoning":{"context":true}}`},
		{name: "unknown effort", fields: `{"reasoning":{"effort":"unsupported"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := claudeResponsesRequestWithFields(t, tt.fields)
			got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, req)
			require.NoError(t, err)
			body, err := kitutil.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, `{"model":"claude-test","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"max_tokens":8192}`, string(body))
		})
	}
}

func TestClaudeResponsesHintsDoNotRelaxChatValidation(t *testing.T) {
	tests := []struct {
		name   string
		fields string
		want   string
	}{
		{name: "include", fields: `{"include":["reasoning.encrypted_content"]}`, want: "include"},
		{name: "client metadata", fields: `{"client_metadata":{"originator":"codex"}}`, want: "client_metadata"},
		{name: "summary", fields: `{"reasoning":{"summary":"auto"}}`, want: "reasoning.summary/mode/context"},
		{name: "mode", fields: `{"reasoning":{"mode":"standard"}}`, want: "reasoning.summary/mode/context"},
		{name: "context", fields: `{"reasoning":{"context":"current_turn"}}`, want: "reasoning.summary/mode/context"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResponsesRequestToChatCompletionsRequest(claudeResponsesRequestWithFields(t, tt.fields))
			require.EqualError(t, err, "responses to chat conversion cannot preserve fields: "+tt.want)
		})
	}
}

func claudeResponsesRequestWithFields(t *testing.T, fields string) *dto.OpenAIResponsesRequest {
	t.Helper()
	req := &dto.OpenAIResponsesRequest{
		Model:           "claude-test",
		Input:           []byte(`"hello"`),
		MaxOutputTokens: kitutil.GetPointer(uint(8192)),
	}
	require.NoError(t, kitutil.UnmarshalJsonStr(fields, req))
	return req
}

func TestResponsesRequestToClaudeMessagesToolChoiceFallback(t *testing.T) {
	const functionTools = `[{"type":"web_search"},{"type":"function","name":"read_file","parameters":{"type":"object","properties":{}}}]`
	tests := []struct {
		name       string
		tools      string
		fields     string
		wantChoice string
	}{
		{name: "automatic", tools: functionTools, fields: `{"tool_choice":"auto"}`, wantChoice: `{"type":"auto"}`},
		{name: "required", tools: functionTools, fields: `{"tool_choice":"required"}`, wantChoice: `{"type":"any"}`},
		{name: "none", tools: functionTools, fields: `{"tool_choice":"none","parallel_tool_calls":false}`, wantChoice: `{"type":"none"}`},
		{name: "named function", tools: functionTools, fields: `{"tool_choice":{"type":"function","name":"read_file"}}`, wantChoice: `{"type":"tool","name":"read_file"}`},
		{name: "disable parallel", tools: functionTools, fields: `{"parallel_tool_calls":false}`, wantChoice: `{"type":"auto","disable_parallel_tool_use":true}`},
		{name: "enable parallel", tools: functionTools, fields: `{"parallel_tool_calls":true}`, wantChoice: `{"type":"auto"}`},
		{name: "missing hints", tools: functionTools, fields: `{}`},
		{name: "malformed choice", tools: functionTools, fields: `{"tool_choice":[]}`},
		{name: "unknown choice", tools: functionTools, fields: `{"tool_choice":"unknown"}`},
		{name: "unsupported builtin choice", tools: functionTools, fields: `{"tool_choice":{"type":"web_search"}}`},
		{name: "missing named function", tools: functionTools, fields: `{"tool_choice":{"type":"function","name":"not_declared"}}`},
		{name: "malformed parallel hint", tools: functionTools, fields: `{"parallel_tool_calls":"false"}`},
		{name: "valid choice with malformed parallel hint", tools: functionTools, fields: `{"tool_choice":"required","parallel_tool_calls":{}}`, wantChoice: `{"type":"any"}`},
		{name: "valid parallel with malformed choice", tools: functionTools, fields: `{"tool_choice":123,"parallel_tool_calls":false}`, wantChoice: `{"type":"auto","disable_parallel_tool_use":true}`},
		{name: "valid parallel with missing named function", tools: functionTools, fields: `{"tool_choice":{"type":"function","name":"not_declared"},"parallel_tool_calls":false}`, wantChoice: `{"type":"auto","disable_parallel_tool_use":true}`},
		{name: "no tools", fields: `{"tool_choice":"required","parallel_tool_calls":false}`},
		{name: "only unsupported tools", tools: `[{"type":"web_search"}]`, fields: `{"tool_choice":"required","parallel_tool_calls":false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := claudeResponsesRequestWithFields(t, tt.fields)
			req.Tools = []byte(tt.tools)
			got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, req)
			require.NoError(t, err)
			body, err := kitutil.Marshal(got)
			require.NoError(t, err)
			var wire map[string]any
			require.NoError(t, kitutil.Unmarshal(body, &wire))
			if tt.tools == functionTools {
				tools, err := kitutil.Marshal(wire["tools"])
				require.NoError(t, err)
				assert.JSONEq(t, `[{"name":"read_file","input_schema":{"type":"object","properties":{}}}]`, string(tools))
			} else {
				assert.NotContains(t, wire, "tools")
			}
			if tt.wantChoice == "" {
				assert.NotContains(t, wire, "tool_choice", "omit unmappable hints instead of emitting null or an invalid choice")
				return
			}
			choice, err := kitutil.Marshal(wire["tool_choice"])
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantChoice, string(choice))
		})
	}
}

func TestResponsesRequestToClaudeMessagesPreservesExplicitZeroValues(t *testing.T) {
	req := claudeResponsesRequestWithFields(t, `{"max_output_tokens":0,"temperature":0,"top_p":0,"stream":false,"include":["unsupported"]}`)
	got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, req)
	require.NoError(t, err)
	body, err := kitutil.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"claude-test","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"max_tokens":0,"temperature":0,"top_p":0,"stream":false}`, string(body))
}

func TestResponsesRequestToClaudeMessagesRejectsMalformedPayload(t *testing.T) {
	tests := []struct {
		name   string
		fields string
		want   string
	}{
		{name: "missing model", fields: `{"model":""}`, want: "model is required"},
		{name: "input object", fields: `{"input":{}}`, want: "unsupported responses input type"},
		{name: "input array", fields: `{"input":[1]}`, want: "invalid input array"},
		{name: "tools", fields: `{"tools":{}}`, want: "invalid tools"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := claudeResponsesRequestWithFields(t, tt.fields)
			got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, req)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestResponsesRequestToClaudeAdaptiveThinking(t *testing.T) {
	tests := []struct {
		name, model, effort, summary, thinking, output string
	}{
		{name: "opus 5.5 low", model: "claude-opus-5-5", effort: "low", thinking: `{"type":"adaptive"}`, output: `{"effort":"low"}`},
		{name: "opus 5.5 medium", model: "claude-opus-5-5", effort: "medium", thinking: `{"type":"adaptive"}`, output: `{"effort":"medium"}`},
		{name: "opus 5.5 high summary", model: "claude-opus-5-5", effort: "high", summary: "auto", thinking: `{"type":"adaptive","display":"summarized"}`, output: `{"effort":"high"}`},
		{name: "opus 5.5 xhigh", model: "claude-opus-5-5", effort: "xhigh", thinking: `{"type":"adaptive"}`, output: `{"effort":"xhigh"}`},
		{name: "opus 5.5 max", model: "claude-opus-5-5", effort: "max", thinking: `{"type":"adaptive"}`, output: `{"effort":"max"}`},
		{name: "minimal maps to low", model: "claude-opus-5-5", effort: "minimal", thinking: `{"type":"adaptive"}`, output: `{"effort":"low"}`},
		{name: "cannot disable always on thinking", model: "claude-opus-5-5", effort: "none", thinking: `{"type":"adaptive"}`, output: `{"effort":"low"}`},
		{name: "omitted effort uses model default", model: "claude-opus-5-5"},
		{name: "unknown effort uses model default", model: "claude-opus-5-5", effort: "future-effort"},
		{name: "opus 5", model: "claude-opus-5", effort: "high", thinking: `{"type":"adaptive"}`, output: `{"effort":"high"}`},
		{name: "opus 4.8", model: "claude-opus-4-8", effort: "high", thinking: `{"type":"adaptive"}`, output: `{"effort":"high"}`},
		{name: "dated opus 4.7", model: "claude-opus-4-7-20260416", effort: "xhigh", summary: "detailed", thinking: `{"type":"adaptive","display":"summarized"}`, output: `{"effort":"xhigh"}`},
		{name: "opus 4.6 cannot use xhigh", model: "claude-opus-4-6", effort: "xhigh", thinking: `{"type":"adaptive"}`, output: `{"effort":"high"}`},
		{name: "opus 4.6 supports max", model: "claude-opus-4-6", effort: "max", thinking: `{"type":"adaptive"}`, output: `{"effort":"max"}`},
		{name: "opus 4.7 can disable thinking", model: "claude-opus-4-7", effort: "none", thinking: `{"type":"disabled"}`},
		{name: "sonnet 4.6", model: "claude-sonnet-4-6", effort: "high", thinking: `{"type":"adaptive"}`, output: `{"effort":"high"}`},
		{name: "sonnet 4.6 supports max", model: "claude-sonnet-4-6", effort: "max", thinking: `{"type":"adaptive"}`, output: `{"effort":"max"}`},
		{name: "sonnet 5 supports max", model: "claude-sonnet-5", effort: "max", thinking: `{"type":"adaptive"}`, output: `{"effort":"max"}`},
		{name: "sonnet 5.5 cannot fully disable thinking", model: "claude-sonnet-5-5", effort: "none", thinking: `{"type":"between_tools"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := claudeResponsesRequestWithFields(t, `{}`)
			req.Model = tt.model
			if tt.effort != "" || tt.summary != "" {
				req.Reasoning = &dto.Reasoning{Effort: tt.effort, Summary: tt.summary}
			}
			req.Temperature = kitutil.GetPointer(0.0)
			req.TopP = kitutil.GetPointer(0.0)
			before, err := kitutil.Marshal(req)
			require.NoError(t, err)
			got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, req)
			require.NoError(t, err)
			body, err := kitutil.Marshal(got)
			require.NoError(t, err)
			var wire map[string]any
			require.NoError(t, kitutil.Unmarshal(body, &wire))
			if tt.thinking == "" {
				assert.NotContains(t, wire, "thinking")
			} else {
				raw, err := kitutil.Marshal(wire["thinking"])
				require.NoError(t, err)
				assert.JSONEq(t, tt.thinking, string(raw))
			}
			if tt.output == "" {
				assert.NotContains(t, wire, "output_config")
			} else {
				assert.JSONEq(t, tt.output, string(got.OutputConfig))
			}
			assert.NotContains(t, wire, "temperature")
			assert.NotContains(t, wire, "top_p")
			assert.Equal(t, tt.model, got.Model)
			after, err := kitutil.Marshal(req)
			require.NoError(t, err)
			assert.JSONEq(t, string(before), string(after))
		})
	}
}

func TestResponsesRequestToClaudeLegacyThinkingBudget(t *testing.T) {
	tests := []struct {
		name, effort string
		limit        uint
		budget       int
	}{
		{name: "low", effort: "low", limit: 8192, budget: 1280},
		{name: "medium", effort: "medium", limit: 8192, budget: 2048},
		{name: "high", effort: "high", limit: 8192, budget: 4096},
		{name: "minimal", effort: "minimal", limit: 8192, budget: 1280},
		{name: "xhigh", effort: "xhigh", limit: 8192, budget: 4096},
		{name: "max", effort: "max", limit: 8192, budget: 4096},
		{name: "budget below output limit", effort: "high", limit: 2048, budget: 2047},
		{name: "minimum budget fits", effort: "high", limit: 1025, budget: 1024},
		{name: "minimum budget does not fit", effort: "high", limit: 1024},
		{name: "small output limit", effort: "high", limit: 64},
		{name: "explicit zero is preserved", effort: "high", limit: 0},
		{name: "unknown effort", effort: "future-effort", limit: 8192},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := claudeResponsesRequestWithFields(t, `{}`)
			req.Model = "claude-sonnet-4-5"
			req.Reasoning = &dto.Reasoning{Effort: tt.effort}
			req.MaxOutputTokens = &tt.limit
			got, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), nil, req)
			require.NoError(t, err)
			require.NotNil(t, got.MaxTokens)
			assert.Equal(t, tt.limit, *got.MaxTokens)
			if tt.budget == 0 {
				assert.Nil(t, got.Thinking)
			} else {
				require.NotNil(t, got.Thinking)
				assert.Equal(t, "enabled", got.Thinking.Type)
				require.NotNil(t, got.Thinking.BudgetTokens)
				assert.Equal(t, tt.budget, *got.Thinking.BudgetTokens)
			}
			assert.Empty(t, got.OutputConfig)
		})
	}
}
