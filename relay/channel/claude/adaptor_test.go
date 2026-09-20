package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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

func mustClaudeRawMessage(t *testing.T, value interface{}) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}
