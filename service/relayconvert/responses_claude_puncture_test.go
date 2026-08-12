package relayconvert

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPunctureResponsesClaudeThroughMockMessagesSSE(t *testing.T) {
	stream := true
	maxOutputTokens := uint(512)
	req := &dto.OpenAIResponsesRequest{
		Model:           "claude-sonnet-test",
		Instructions:    mustPunctureRawMessage(t, "Answer with the available tools."),
		Input:           mustPunctureRawMessage(t, []map[string]any{{"role": "user", "content": "look up the weather"}}),
		MaxOutputTokens: &maxOutputTokens,
		Reasoning:       &dto.Reasoning{Effort: "medium"},
		Stream:          &stream,
		Tools: mustPunctureRawMessage(t, []map[string]any{{
			"type":        "function",
			"name":        "lookup_weather",
			"description": "Look up current weather.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{"city": map[string]any{"type": "string"}},
				"required":   []string{"city"},
			},
		}}),
	}

	claudeReq, err := OpenAIResponsesRequestToClaudeMessages(nil, req)
	require.NoError(t, err)
	require.Len(t, claudeReq.Messages, 1)
	assert.Equal(t, "user", claudeReq.Messages[0].Role)
	assert.Equal(t, "claude-sonnet-test", claudeReq.Model)
	require.NotNil(t, claudeReq.MaxTokens)
	assert.Equal(t, uint(512), *claudeReq.MaxTokens)
	require.NotNil(t, claudeReq.Thinking)
	assert.Equal(t, 2048, claudeReq.Thinking.GetBudgetTokens())

	var received dto.ClaudeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.NoError(t, common.DecodeJson(r.Body, &received))
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{
			Type: "message_start",
			Message: &dto.ClaudeMediaMessage{
				Id:    "msg_puncture_1",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-sonnet-test",
				Usage: &dto.ClaudeUsage{InputTokens: 7},
			},
		})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{
			Type: "content_block_start",
			Index: intPointer(0),
			ContentBlock: &dto.ClaudeMediaMessage{Type: "text", Text: stringPointer("")},
		})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{
			Type: "content_block_delta",
			Index: intPointer(0),
			Delta: &dto.ClaudeMediaMessage{Type: "text_delta", Text: stringPointer("I will check that.")},
		})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{Type: "content_block_stop", Index: intPointer(0)})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{
			Type: "content_block_start",
			Index: intPointer(1),
			ContentBlock: &dto.ClaudeMediaMessage{Type: "tool_use", Id: "toolu_puncture_1", Name: "lookup_weather", Input: map[string]any{}},
		})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{
			Type: "content_block_delta",
			Index: intPointer(1),
			Delta: &dto.ClaudeMediaMessage{Type: "input_json_delta", PartialJson: stringPointer(`{"city":"Shanghai"}`)},
		})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{Type: "content_block_stop", Index: intPointer(1)})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{
			Type:       "message_delta",
			Delta:      &dto.ClaudeMediaMessage{StopReason: stringPointer("tool_use")},
			Usage:      &dto.ClaudeUsage{OutputTokens: 5},
		})
		writePunctureClaudeEvent(t, w, dto.ClaudeResponse{Type: "message_stop"})
	}))
	defer server.Close()

	body, err := common.Marshal(claudeReq)
	require.NoError(t, err)
	resp, err := http.Post(server.URL+"/v1/messages", "application/json", strings.NewReader(string(body)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	state, err := NewResponseStreamState(types.RelayFormatClaude, types.RelayFormatOpenAIResponses, ResponseStreamOptions{
		ID:    "msg_puncture_1",
		Model: "claude-sonnet-test",
	})
	require.NoError(t, err)

	var events []ChatToResponsesStreamEvent
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var claudeEvent dto.ClaudeResponse
		require.NoError(t, common.UnmarshalJsonStr(strings.TrimSpace(strings.TrimPrefix(line, "data:")), &claudeEvent))
		results, convertErr := ConvertStreamResponseChunk(nil, nil, state, &claudeEvent)
		require.NoError(t, convertErr)
		for _, result := range results {
			converted, ok := result.Value.(ChatToResponsesStreamEvent)
			require.True(t, ok, "unexpected converted stream type %T", result.Value)
			events = append(events, converted)
		}
	}
	require.NoError(t, scanner.Err())

	finalResults, err := FinalizeStreamResponse(nil, nil, state)
	require.NoError(t, err)
	for _, result := range finalResults {
		converted, ok := result.Value.(ChatToResponsesStreamEvent)
		require.True(t, ok, "unexpected final stream type %T", result.Value)
		events = append(events, converted)
	}

	assert.Equal(t, "Answer with the available tools.", claudeReq.ParseSystem()[0].GetText())
	assert.Equal(t, "claude-sonnet-test", received.Model)
	require.NotNil(t, received.Stream)
	assert.True(t, *received.Stream)
	require.Len(t, received.Messages, 1)
	assert.Equal(t, "look up the weather", received.Messages[0].GetStringContent())
	receivedTools, ok := received.Tools.([]any)
	require.True(t, ok)
	require.Len(t, receivedTools, 1)
	receivedTool, ok := receivedTools[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "lookup_weather", receivedTool["name"])
	tools, ok := claudeReq.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	assert.Equal(t, "lookup_weather", tool.Name)
	assert.Equal(t, "response.created", events[0].Type)
	assert.Contains(t, responseEventTypes(events), "response.output_text.delta")
	assert.Contains(t, responseEventTypes(events), "response.function_call_arguments.delta")
	assert.Contains(t, responseEventTypes(events), "response.function_call_arguments.done")
	assert.Equal(t, "response.completed", events[len(events)-1].Type)

	var sawTool bool
	for _, event := range events {
		if event.Type == "response.output_item.done" && event.Payload.Item != nil && event.Payload.Item.Type == "function_call" {
			sawTool = true
			assert.Equal(t, "lookup_weather", event.Payload.Item.Name)
			assert.Equal(t, `{"city":"Shanghai"}`, event.Payload.Item.ArgumentsString())
		}
	}
	assert.True(t, sawTool)
	require.NotNil(t, state.Usage())
	t.Logf("converted usage: prompt=%d completion=%d total=%d", state.Usage().PromptTokens, state.Usage().CompletionTokens, state.Usage().TotalTokens)
}

func writePunctureClaudeEvent(t *testing.T, w http.ResponseWriter, event dto.ClaudeResponse) {
	t.Helper()
	data, err := common.Marshal(event)
	require.NoError(t, err)
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
	require.NoError(t, err)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func mustPunctureRawMessage(t *testing.T, value any) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}

func responseEventTypes(events []ChatToResponsesStreamEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}
	return types
}

func intPointer(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
