package relayconvert

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupBuiltinResponseConverters(t *testing.T) {
	tests := []struct {
		lookupID       string
		id             string
		from           types.RelayFormat
		to             types.RelayFormat
		quality        ResponseConverterQuality
		stepConverters []string
	}{
		{lookupID: ResponseConverterOAIChatToOAIResponses, id: ConverterOpenAIChatToOpenAIResponses, from: types.RelayFormatOpenAI, to: types.RelayFormatOpenAIResponses, quality: ResponseConverterQualityGood},
		{lookupID: ResponseConverterOAIResponsesToOAIChat, id: ConverterOpenAIResponsesToOpenAIChat, from: types.RelayFormatOpenAIResponses, to: types.RelayFormatOpenAI, quality: ResponseConverterQualityGood},
		{lookupID: ResponseConverterOAIChatToClaudeMessages, id: ConverterOpenAIChatToClaudeMessages, from: types.RelayFormatOpenAI, to: types.RelayFormatClaude, quality: ResponseConverterQualityFair},
		{lookupID: ResponseConverterOAIChatToGeminiChat, id: ConverterOpenAIChatToGeminiContent, from: types.RelayFormatOpenAI, to: types.RelayFormatGemini, quality: ResponseConverterQualityFair},
		{lookupID: ResponseConverterClaudeMessagesToOAIChat, id: ConverterClaudeMessagesToOpenAIChat, from: types.RelayFormatClaude, to: types.RelayFormatOpenAI, quality: ResponseConverterQualityFair},
		{lookupID: ResponseConverterGeminiChatToOAIChat, id: ConverterGeminiContentToOpenAIChat, from: types.RelayFormatGemini, to: types.RelayFormatOpenAI, quality: ResponseConverterQualityFair},
		{
			lookupID: responseConverterClaudeToGemini,
			id:       requestConverterClaudeToGemini,
			from:     types.RelayFormatClaude,
			to:       types.RelayFormatGemini,
			quality:  ResponseConverterQualityDiscouraged,
			stepConverters: []string{
				ConverterClaudeMessagesToOpenAIChat,
				ConverterOpenAIChatToGeminiContent,
			},
		},
		{
			lookupID: responseConverterClaudeToResponses,
			id:       requestConverterClaudeToResponses,
			from:     types.RelayFormatClaude,
			to:       types.RelayFormatOpenAIResponses,
			quality:  ResponseConverterQualityFair,
			stepConverters: []string{
				ConverterClaudeMessagesToOpenAIChat,
				ConverterOpenAIChatToOpenAIResponses,
			},
		},
		{
			lookupID: responseConverterGeminiToClaude,
			id:       requestConverterGeminiToClaude,
			from:     types.RelayFormatGemini,
			to:       types.RelayFormatClaude,
			quality:  ResponseConverterQualityDiscouraged,
			stepConverters: []string{
				ConverterGeminiContentToOpenAIChat,
				ConverterOpenAIChatToClaudeMessages,
			},
		},
		{
			lookupID: responseConverterGeminiToResponses,
			id:       requestConverterGeminiToResponses,
			from:     types.RelayFormatGemini,
			to:       types.RelayFormatOpenAIResponses,
			quality:  ResponseConverterQualityFair,
			stepConverters: []string{
				ConverterGeminiContentToOpenAIChat,
				ConverterOpenAIChatToOpenAIResponses,
			},
		},
		{
			lookupID: responseConverterResponsesToClaude,
			id:       requestConverterResponsesToClaude,
			from:     types.RelayFormatOpenAIResponses,
			to:       types.RelayFormatClaude,
			quality:  ResponseConverterQualityFair,
			stepConverters: []string{
				ConverterOpenAIResponsesToOpenAIChat,
				ConverterOpenAIChatToClaudeMessages,
			},
		},
		{
			lookupID: responseConverterResponsesToGemini,
			id:       ConverterOpenAIResponsesToGemini,
			from:     types.RelayFormatOpenAIResponses,
			to:       types.RelayFormatGemini,
			quality:  ResponseConverterQualityFair,
			stepConverters: []string{
				ConverterOpenAIResponsesToOpenAIChat,
				ConverterOpenAIChatToGeminiContent,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.lookupID, func(t *testing.T) {
			spec, ok := LookupResponseConverter(tt.lookupID)
			require.True(t, ok)
			assert.Equal(t, tt.id, spec.ID)
			assert.Equal(t, tt.from, spec.From)
			assert.Equal(t, tt.to, spec.To)
			assert.Equal(t, tt.quality, spec.Quality)
			assert.Equal(t, tt.stepConverters, spec.StepConverters)
			if len(tt.stepConverters) == 0 {
				assert.NotNil(t, spec.Convert)
			} else {
				assert.Nil(t, spec.Convert)
			}
		})
	}

	_, ok := LookupResponseConverter("missing")
	assert.False(t, ok)
}

func TestConvertResponseRejectsNilAndUnsupportedRoute(t *testing.T) {
	_, err := ConvertResponse(nil, nil, types.RelayFormatOpenAI, (*dto.OpenAITextResponse)(nil))
	require.Error(t, err)

	_, err = ConvertResponse(nil, nil, types.RelayFormatEmbedding, &dto.OpenAITextResponse{})
	require.Error(t, err)
}

func TestConvertResponseDirectConverters(t *testing.T) {
	chat := textRegistryChatResponse()
	info := &convmeta.Values{ChannelMetaAttached: true, UpstreamModelName: "gemini-test"}

	toResponses, err := ConvertResponse(nil, info, types.RelayFormatOpenAIResponses, chat)
	require.NoError(t, err)
	assert.Equal(t, ConverterOpenAIChatToOpenAIResponses, toResponses.Converter)
	assert.Equal(t, ResponseConverterQualityGood, toResponses.Quality)
	assert.Equal(t, types.RelayFormatOpenAI, toResponses.From)
	assert.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), toResponses.To)
	assert.Equal(t, []ResponseStep{{Converter: ConverterOpenAIChatToOpenAIResponses, From: types.RelayFormatOpenAI, To: types.RelayFormatOpenAIResponses}}, toResponses.Steps)
	require.IsType(t, &dto.OpenAIResponsesResponse{}, toResponses.Value)
	assert.Equal(t, 9, toResponses.Usage.TotalTokens)
	require.NotNil(t, toResponses.Usage.BillingUsage)
	require.NotNil(t, toResponses.Usage.BillingUsage.OpenAIUsage)
	assert.Equal(t, dto.BillingUsageSourceOAIChat, toResponses.Usage.BillingUsage.Source)
	assert.Equal(t, 4, toResponses.Usage.BillingUsage.OpenAIUsage.PromptTokens)

	responses := &dto.OpenAIResponsesResponse{
		ID:        "resp_1",
		CreatedAt: 123,
		Model:     "gpt-test",
		Status:    []byte(`"completed"`),
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "hello"},
				},
			},
		},
		Usage: &dto.Usage{InputTokens: 4, OutputTokens: 6, TotalTokens: 10},
	}
	toChat, err := ConvertResponse(nil, info, types.RelayFormatOpenAI, responses)
	require.NoError(t, err)
	assert.Equal(t, ConverterOpenAIResponsesToOpenAIChat, toChat.Converter)
	assert.Equal(t, ResponseConverterQualityGood, toChat.Quality)
	require.IsType(t, &dto.OpenAITextResponse{}, toChat.Value)
	assert.Equal(t, 10, toChat.Usage.TotalTokens)
	require.NotNil(t, toChat.Usage.BillingUsage)
	require.NotNil(t, toChat.Usage.BillingUsage.OpenAIUsage)
	assert.Equal(t, dto.BillingUsageSourceOAIResponses, toChat.Usage.BillingUsage.Source)
	assert.Equal(t, 4, toChat.Usage.BillingUsage.OpenAIUsage.InputTokens)

	toClaude, err := ConvertResponse(nil, info, types.RelayFormatClaude, chat)
	require.NoError(t, err)
	assert.Equal(t, ConverterOpenAIChatToClaudeMessages, toClaude.Converter)
	assert.Equal(t, ResponseConverterQualityFair, toClaude.Quality)
	require.IsType(t, &dto.ClaudeResponse{}, toClaude.Value)
	assert.Equal(t, 9, toClaude.Usage.TotalTokens)
	require.NotNil(t, toClaude.Usage.BillingUsage)
	require.NotNil(t, toClaude.Usage.BillingUsage.OpenAIUsage)
	claudeValue := toClaude.Value.(*dto.ClaudeResponse)
	require.NotNil(t, claudeValue.Usage)
	require.NotNil(t, claudeValue.Usage.BillingUsage)
	require.NotNil(t, claudeValue.Usage.BillingUsage.OpenAIUsage)

	toGemini, err := ConvertResponse(nil, info, types.RelayFormatGemini, chat)
	require.NoError(t, err)
	assert.Equal(t, ConverterOpenAIChatToGeminiContent, toGemini.Converter)
	assert.Equal(t, ResponseConverterQualityFair, toGemini.Quality)
	require.IsType(t, &dto.GeminiChatResponse{}, toGemini.Value)
	assert.Equal(t, 9, toGemini.Usage.TotalTokens)
	require.NotNil(t, toGemini.Usage.BillingUsage)
	require.NotNil(t, toGemini.Usage.BillingUsage.OpenAIUsage)
	geminiValue := toGemini.Value.(*dto.GeminiChatResponse)
	require.NotNil(t, geminiValue.UsageMetadata.BillingUsage)
	require.NotNil(t, geminiValue.UsageMetadata.BillingUsage.OpenAIUsage)
}

func TestConvertResponseMultiHopConverters(t *testing.T) {
	responses := textRegistryResponsesResponse()

	toClaude, err := ConvertResponse(nil, &convmeta.Values{}, types.RelayFormatClaude, responses)
	require.NoError(t, err)
	assert.Equal(t, requestConverterResponsesToClaude, toClaude.Converter)
	assert.Equal(t, ResponseConverterQualityFair, toClaude.Quality)
	assert.Equal(t, []ResponseStep{
		{Converter: ConverterOpenAIResponsesToOpenAIChat, From: types.RelayFormatOpenAIResponses, To: types.RelayFormatOpenAI},
		{Converter: ConverterOpenAIChatToClaudeMessages, From: types.RelayFormatOpenAI, To: types.RelayFormatClaude},
	}, toClaude.Steps)
	require.IsType(t, &dto.ClaudeResponse{}, toClaude.Value)
	claudeValue := toClaude.Value.(*dto.ClaudeResponse)
	require.Len(t, claudeValue.Content, 2)
	assert.Equal(t, "text", claudeValue.Content[0].Type)
	assert.Equal(t, "tool_use", claudeValue.Content[1].Type)
	assert.Equal(t, "lookup", claudeValue.Content[1].Name)
	assert.Equal(t, map[string]interface{}{"q": "x"}, claudeValue.Content[1].Input)
	assert.Equal(t, 11, toClaude.Usage.TotalTokens)

	toGemini, err := ConvertResponse(nil, &convmeta.Values{ChannelMetaAttached: true, UpstreamModelName: "gemini-test"}, types.RelayFormatGemini, responses)
	require.NoError(t, err)
	assert.Equal(t, ConverterOpenAIResponsesToGemini, toGemini.Converter)
	assert.Equal(t, ResponseConverterQualityFair, toGemini.Quality)
	assert.Equal(t, []ResponseStep{
		{Converter: ConverterOpenAIResponsesToOpenAIChat, From: types.RelayFormatOpenAIResponses, To: types.RelayFormatOpenAI},
		{Converter: ConverterOpenAIChatToGeminiContent, From: types.RelayFormatOpenAI, To: types.RelayFormatGemini},
	}, toGemini.Steps)
	require.IsType(t, &dto.GeminiChatResponse{}, toGemini.Value)
	geminiValue := toGemini.Value.(*dto.GeminiChatResponse)
	require.Len(t, geminiValue.Candidates, 1)
	require.Len(t, geminiValue.Candidates[0].Content.Parts, 2)
	assert.Equal(t, "hello", geminiValue.Candidates[0].Content.Parts[0].Text)
	require.NotNil(t, geminiValue.Candidates[0].Content.Parts[1].FunctionCall)
	assert.Equal(t, "lookup", geminiValue.Candidates[0].Content.Parts[1].FunctionCall.FunctionName)
	assert.Equal(t, map[string]interface{}{"q": "x"}, geminiValue.Candidates[0].Content.Parts[1].FunctionCall.Arguments)
	assert.Equal(t, 11, toGemini.Usage.TotalTokens)
}

func TestConvertResponseByIDExecutesMultiHopAndChecksSource(t *testing.T) {
	responses := textRegistryResponsesResponse()

	result, err := ConvertResponseByID(nil, nil, responseConverterResponsesToGemini, responses)
	require.NoError(t, err)
	assert.Equal(t, ConverterOpenAIResponsesToGemini, result.Converter)
	assert.Equal(t, []ResponseStep{
		{Converter: ConverterOpenAIResponsesToOpenAIChat, From: types.RelayFormatOpenAIResponses, To: types.RelayFormatOpenAI},
		{Converter: ConverterOpenAIChatToGeminiContent, From: types.RelayFormatOpenAI, To: types.RelayFormatGemini},
	}, result.Steps)

	_, err = ConvertResponseByID(nil, nil, responseConverterResponsesToGemini, textRegistryChatResponse())
	require.Error(t, err)
}

func TestConvertResponseProviderToOAIChatUsage(t *testing.T) {
	claude := &dto.ClaudeResponse{
		Id:         "msg_1",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-test",
		StopReason: "end_turn",
		Content: []dto.ClaudeMediaMessage{
			{Type: "tool_use", Id: "toolu_1", Name: "lookup", Input: map[string]interface{}{"q": "x"}},
		},
		Usage: &dto.ClaudeUsage{
			InputTokens:              10,
			CacheReadInputTokens:     3,
			CacheCreationInputTokens: 4,
			OutputTokens:             5,
			CacheCreation: &dto.ClaudeCacheCreationUsage{
				Ephemeral5mInputTokens: 1,
				Ephemeral1hInputTokens: 3,
			},
		},
	}
	toChat, err := ConvertResponse(nil, nil, types.RelayFormatOpenAI, claude)
	require.NoError(t, err)
	assert.Equal(t, ConverterClaudeMessagesToOpenAIChat, toChat.Converter)
	require.IsType(t, &dto.OpenAITextResponse{}, toChat.Value)
	assert.Equal(t, 17, toChat.Usage.PromptTokens)
	assert.Equal(t, 5, toChat.Usage.CompletionTokens)
	assert.Equal(t, 22, toChat.Usage.TotalTokens)
	assert.Equal(t, 3, toChat.Usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 4, toChat.Usage.PromptTokensDetails.CachedCreationTokens)
	assert.Equal(t, 4, toChat.Usage.PromptTokensDetails.CacheWriteTokens)
	require.NotNil(t, toChat.Usage.BillingUsage)
	require.NotNil(t, toChat.Usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, dto.BillingUsageSourceClaudeMessages, toChat.Usage.BillingUsage.Source)
	assert.Equal(t, dto.BillingUsageSemanticAnthropic, toChat.Usage.BillingUsage.Semantic)
	assert.Equal(t, 10, toChat.Usage.BillingUsage.ClaudeUsage.InputTokens)
	assert.Equal(t, 3, toChat.Usage.BillingUsage.ClaudeUsage.CacheReadInputTokens)
	assert.Equal(t, 4, toChat.Usage.BillingUsage.ClaudeUsage.CacheCreationInputTokens)
	assert.Equal(t, 5, toChat.Usage.BillingUsage.ClaudeUsage.OutputTokens)
	chatValue := toChat.Value.(*dto.OpenAITextResponse)
	require.Len(t, chatValue.Choices, 1)
	require.Len(t, chatValue.Choices[0].Message.ParseToolCalls(), 1)
	assert.JSONEq(t, `{"q":"x"}`, chatValue.Choices[0].Message.ParseToolCalls()[0].Function.Arguments)

	gemini := &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{
			{
				Content: dto.GeminiChatContent{
					Parts: []dto.GeminiPart{
						{Text: "hello"},
						{FunctionCall: &dto.FunctionCall{FunctionName: "lookup", Arguments: map[string]interface{}{"q": "x"}}},
					},
				},
			},
		},
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:        7,
			ToolUsePromptTokenCount: 2,
			CandidatesTokenCount:    5,
			ThoughtsTokenCount:      3,
			TotalTokenCount:         17,
			CachedContentTokenCount: 4,
			PromptTokensDetails: []dto.GeminiPromptTokensDetails{
				{Modality: "TEXT", TokenCount: 5},
				{Modality: "IMAGE", TokenCount: 1},
			},
			ToolUsePromptTokensDetails: []dto.GeminiPromptTokensDetails{
				{Modality: "AUDIO", TokenCount: 3},
			},
			CandidatesTokensDetails: []dto.GeminiPromptTokensDetails{
				{Modality: "TEXT", TokenCount: 4},
				{Modality: "IMAGE", TokenCount: 1},
			},
		},
	}
	toChat, err = ConvertResponse(nil, &convmeta.Values{ChannelMetaAttached: true, UpstreamModelName: "gemini-test"}, types.RelayFormatOpenAI, gemini)
	require.NoError(t, err)
	assert.Equal(t, ConverterGeminiContentToOpenAIChat, toChat.Converter)
	require.IsType(t, &dto.OpenAITextResponse{}, toChat.Value)
	assert.Equal(t, 9, toChat.Usage.PromptTokens)
	assert.Equal(t, 8, toChat.Usage.CompletionTokens)
	assert.Equal(t, 17, toChat.Usage.TotalTokens)
	assert.Equal(t, 3, toChat.Usage.CompletionTokenDetails.ReasoningTokens)
	assert.Equal(t, 4, toChat.Usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 5, toChat.Usage.PromptTokensDetails.TextTokens)
	assert.Equal(t, 3, toChat.Usage.PromptTokensDetails.AudioTokens)
	assert.Equal(t, 1, toChat.Usage.PromptTokensDetails.ImageTokens)
	assert.Equal(t, 4, toChat.Usage.CompletionTokenDetails.TextTokens)
	assert.Equal(t, 1, toChat.Usage.CompletionTokenDetails.ImageTokens)
	require.NotNil(t, toChat.Usage.BillingUsage)
	require.NotNil(t, toChat.Usage.BillingUsage.GeminiUsageMetadata)
	assert.Equal(t, dto.BillingUsageSourceGeminiChat, toChat.Usage.BillingUsage.Source)
	assert.Equal(t, dto.BillingUsageSemanticGemini, toChat.Usage.BillingUsage.Semantic)
	assert.Equal(t, 7, toChat.Usage.BillingUsage.GeminiUsageMetadata.PromptTokenCount)
	assert.Equal(t, 2, toChat.Usage.BillingUsage.GeminiUsageMetadata.ToolUsePromptTokenCount)
	assert.Equal(t, 17, toChat.Usage.BillingUsage.GeminiUsageMetadata.TotalTokenCount)
}

func TestConvertResponsePreservesBillingUsageAcrossChatResponsesBridge(t *testing.T) {
	chat := textRegistryChatResponse()
	chat.Usage.BillingUsage = dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
		InputTokens:              10,
		CacheReadInputTokens:     3,
		CacheCreationInputTokens: 4,
		OutputTokens:             5,
	})

	toResponses, err := ConvertResponse(nil, nil, types.RelayFormatOpenAIResponses, chat)
	require.NoError(t, err)
	require.NotNil(t, toResponses.Usage.BillingUsage)
	require.NotNil(t, toResponses.Usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, 10, toResponses.Usage.BillingUsage.ClaudeUsage.InputTokens)

	responsesValue := toResponses.Value.(*dto.OpenAIResponsesResponse)
	toChat, err := ConvertResponse(nil, nil, types.RelayFormatOpenAI, responsesValue)
	require.NoError(t, err)
	require.NotNil(t, toChat.Usage.BillingUsage)
	require.NotNil(t, toChat.Usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, 4, toChat.Usage.BillingUsage.ClaudeUsage.CacheCreationInputTokens)
}

func TestConvertResponseUsesBillingUsageWhenRestoringNativeTargets(t *testing.T) {
	chat := textRegistryChatResponse()
	chat.Usage.BillingUsage = dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
		InputTokens:              10,
		CacheReadInputTokens:     3,
		CacheCreationInputTokens: 4,
		OutputTokens:             5,
	})

	toClaude, err := ConvertResponse(nil, nil, types.RelayFormatClaude, chat)
	require.NoError(t, err)
	claudeValue := toClaude.Value.(*dto.ClaudeResponse)
	require.NotNil(t, claudeValue.Usage)
	assert.Equal(t, 10, claudeValue.Usage.InputTokens)
	assert.Equal(t, 3, claudeValue.Usage.CacheReadInputTokens)
	assert.Equal(t, 4, claudeValue.Usage.CacheCreationInputTokens)
	assert.Equal(t, 5, claudeValue.Usage.OutputTokens)

	chat.Usage.BillingUsage = dto.NewGeminiChatBillingUsage(&dto.GeminiUsageMetadata{
		PromptTokenCount:        7,
		ToolUsePromptTokenCount: 2,
		CandidatesTokenCount:    5,
		ThoughtsTokenCount:      3,
		TotalTokenCount:         17,
	})

	toGemini, err := ConvertResponse(nil, nil, types.RelayFormatGemini, chat)
	require.NoError(t, err)
	geminiValue := toGemini.Value.(*dto.GeminiChatResponse)
	assert.Equal(t, 7, geminiValue.UsageMetadata.PromptTokenCount)
	assert.Equal(t, 2, geminiValue.UsageMetadata.ToolUsePromptTokenCount)
	assert.Equal(t, 5, geminiValue.UsageMetadata.CandidatesTokenCount)
	assert.Equal(t, 3, geminiValue.UsageMetadata.ThoughtsTokenCount)
	assert.Equal(t, 17, geminiValue.UsageMetadata.TotalTokenCount)
}

func TestConvertStreamResponseDirectConverters(t *testing.T) {
	info := &convmeta.Values{
		ClaudeConvertInfo: &convmeta.ClaudeConvertInfo{
			LastMessagesType: convmeta.LastMessageTypeNone,
		},
	}
	info.SendResponseCount = 1
	finishReason := "stop"
	result, err := ConvertStreamResponse(nil, info, types.RelayFormatClaude, &dto.ChatCompletionsStreamResponse{
		Id:    "chatcmpl_1",
		Model: "gpt-test",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				FinishReason: &finishReason,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Content: respPtr("hello"),
				},
			},
		},
		Usage: &dto.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
	})
	require.NoError(t, err)
	assert.True(t, result.Stream)
	assert.Equal(t, ConverterOpenAIChatToClaudeMessages, result.Converter)
	require.IsType(t, []*dto.ClaudeResponse{}, result.Value)
	assert.Equal(t, 5, result.Usage.TotalTokens)

	result, err = ConvertStreamResponse(nil, &convmeta.Values{ChannelMetaAttached: true, UpstreamModelName: "gemini-test"}, types.RelayFormatOpenAI, &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{Content: dto.GeminiChatContent{Parts: []dto.GeminiPart{{Text: "hello"}}}}},
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     1,
			CandidatesTokenCount: 2,
			TotalTokenCount:      3,
		},
	})
	require.NoError(t, err)
	assert.True(t, result.Stream)
	assert.Equal(t, ConverterGeminiContentToOpenAIChat, result.Converter)
	require.IsType(t, &dto.ChatCompletionsStreamResponse{}, result.Value)
	assert.Equal(t, 3, result.Usage.TotalTokens)
}

func TestConvertStreamResponseStatefulDirectConverters(t *testing.T) {
	chatState, err := NewResponseStreamState(types.RelayFormatOpenAI, types.RelayFormatOpenAIResponses, ResponseStreamOptions{
		ID:    "resp_1",
		Model: "gpt-test",
	})
	require.NoError(t, err)
	chatResults, err := ConvertStreamResponseChunk(nil, nil, chatState, &dto.ChatCompletionsStreamResponse{
		Id:    "chatcmpl_1",
		Model: "gpt-test",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: respPtr("hello")}},
		},
		Usage: &dto.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
	})
	require.NoError(t, err)
	require.NotEmpty(t, chatResults)
	assert.Equal(t, ConverterOpenAIChatToOpenAIResponses, chatResults[0].Converter)
	assert.Equal(t, []ResponseStep{{Converter: ConverterOpenAIChatToOpenAIResponses, From: types.RelayFormatOpenAI, To: types.RelayFormatOpenAIResponses}}, chatResults[0].Steps)
	assert.Equal(t, 5, chatState.Usage().TotalTokens)

	finalResults, err := FinalizeStreamResponse(nil, nil, chatState)
	require.NoError(t, err)
	require.NotEmpty(t, finalResults)
	lastEvent, ok := finalResults[len(finalResults)-1].Value.(ChatToResponsesStreamEvent)
	require.True(t, ok)
	assert.Equal(t, "response.completed", lastEvent.Type)

	responsesState, err := NewResponseStreamState(types.RelayFormatOpenAIResponses, types.RelayFormatOpenAI, ResponseStreamOptions{
		ID:    "chatcmpl_1",
		Model: "gpt-test",
	})
	require.NoError(t, err)
	responsesResults, err := ConvertStreamResponseChunk(nil, nil, responsesState, &dto.ResponsesStreamResponse{
		Type:  "response.output_text.delta",
		Delta: "hello",
	})
	require.NoError(t, err)
	require.NotEmpty(t, responsesResults)
	assert.Equal(t, ConverterOpenAIResponsesToOpenAIChat, responsesResults[0].Converter)
	assert.Equal(t, []ResponseStep{{Converter: ConverterOpenAIResponsesToOpenAIChat, From: types.RelayFormatOpenAIResponses, To: types.RelayFormatOpenAI}}, responsesResults[0].Steps)
	require.IsType(t, dto.ChatCompletionsStreamResponse{}, responsesResults[len(responsesResults)-1].Value)
}

func TestConvertStreamResponseStatefulMultiHopResponsesToClaude(t *testing.T) {
	info := &convmeta.Values{
		ClaudeConvertInfo: &convmeta.ClaudeConvertInfo{
			LastMessagesType: convmeta.LastMessageTypeNone,
		},
	}
	state, err := NewResponseStreamState(types.RelayFormatOpenAIResponses, types.RelayFormatClaude, ResponseStreamOptions{
		ID:    "chatcmpl_1",
		Model: "gpt-test",
	})
	require.NoError(t, err)

	results, err := ConvertStreamResponseChunk(nil, info, state, &dto.ResponsesStreamResponse{
		Type:  "response.output_text.delta",
		Delta: "hello",
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, requestConverterResponsesToClaude, results[0].Converter)
	assert.Equal(t, []ResponseStep{
		{Converter: ConverterOpenAIResponsesToOpenAIChat, From: types.RelayFormatOpenAIResponses, To: types.RelayFormatOpenAI},
		{Converter: ConverterOpenAIChatToClaudeMessages, From: types.RelayFormatOpenAI, To: types.RelayFormatClaude},
	}, results[0].Steps)

	var sawTextDelta bool
	for _, result := range results {
		claudeResponse, ok := result.Value.(*dto.ClaudeResponse)
		if !ok || claudeResponse == nil {
			continue
		}
		if claudeResponse.Type == "content_block_delta" && claudeResponse.Delta != nil && claudeResponse.Delta.Text != nil && *claudeResponse.Delta.Text == "hello" {
			sawTextDelta = true
		}
	}
	assert.True(t, sawTextDelta)

	state.SetUsage(&dto.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5})
	_, err = FinalizeStreamResponse(nil, info, state)
	require.NoError(t, err)
	assert.Equal(t, 5, state.Usage().TotalTokens)
}

func TestResponseUsageMatrixChatAndResponsesDetails(t *testing.T) {
	chat := textRegistryChatResponse()
	chat.Usage = dto.Usage{
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         3,
			CachedCreationTokens: 2,
			CacheWriteTokens:     6,
			TextTokens:           4,
			AudioTokens:          1,
			ImageTokens:          5,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			ReasoningTokens: 2,
			TextTokens:      2,
			AudioTokens:     1,
			ImageTokens:     2,
		},
	}
	result, err := ConvertResponse(nil, nil, types.RelayFormatOpenAIResponses, chat)
	require.NoError(t, err)
	assert.Equal(t, 10, result.Usage.InputTokens)
	assert.Equal(t, 5, result.Usage.OutputTokens)
	assert.Equal(t, 20, result.Usage.TotalTokens)
	require.NotNil(t, result.Usage.InputTokensDetails)
	assert.Equal(t, 3, result.Usage.InputTokensDetails.CachedTokens)
	assert.Equal(t, 2, result.Usage.InputTokensDetails.CachedCreationTokens)
	assert.Equal(t, 6, result.Usage.InputTokensDetails.CacheWriteTokens)
	assert.Equal(t, 4, result.Usage.InputTokensDetails.TextTokens)
	assert.Equal(t, 1, result.Usage.InputTokensDetails.AudioTokens)
	assert.Equal(t, 5, result.Usage.InputTokensDetails.ImageTokens)
	assert.Equal(t, 2, result.Usage.CompletionTokenDetails.ReasoningTokens)
	assert.Equal(t, 2, result.Usage.CompletionTokenDetails.TextTokens)
	assert.Equal(t, 1, result.Usage.CompletionTokenDetails.AudioTokens)
	assert.Equal(t, 2, result.Usage.CompletionTokenDetails.ImageTokens)

	responses := &dto.OpenAIResponsesResponse{
		ID:        "resp_1",
		Status:    []byte(`"completed"`),
		Model:     "gpt-test",
		Output:    []dto.ResponsesOutput{},
		CreatedAt: 123,
		Usage: &dto.Usage{
			InputTokens:  12,
			OutputTokens: 8,
			TotalTokens:  21,
			InputTokensDetails: &dto.InputTokenDetails{
				CachedTokens:         4,
				CachedCreationTokens: 1,
				CacheWriteTokens:     7,
				TextTokens:           5,
				AudioTokens:          2,
				ImageTokens:          1,
			},
			CompletionTokenDetails: dto.OutputTokenDetails{
				ReasoningTokens: 3,
				TextTokens:      4,
				AudioTokens:     1,
				ImageTokens:     3,
			},
		},
	}
	result, err = ConvertResponse(nil, nil, types.RelayFormatOpenAI, responses)
	require.NoError(t, err)
	assert.Equal(t, 12, result.Usage.PromptTokens)
	assert.Equal(t, 8, result.Usage.CompletionTokens)
	assert.Equal(t, 21, result.Usage.TotalTokens)
	assert.Equal(t, 4, result.Usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 1, result.Usage.PromptTokensDetails.CachedCreationTokens)
	assert.Equal(t, 7, result.Usage.PromptTokensDetails.CacheWriteTokens)
	assert.Equal(t, 5, result.Usage.PromptTokensDetails.TextTokens)
	assert.Equal(t, 2, result.Usage.PromptTokensDetails.AudioTokens)
	assert.Equal(t, 1, result.Usage.PromptTokensDetails.ImageTokens)
	assert.Equal(t, 3, result.Usage.CompletionTokenDetails.ReasoningTokens)
	assert.Equal(t, 4, result.Usage.CompletionTokenDetails.TextTokens)
	assert.Equal(t, 1, result.Usage.CompletionTokenDetails.AudioTokens)
	assert.Equal(t, 3, result.Usage.CompletionTokenDetails.ImageTokens)
}

func TestOpenAICacheUsageConversionRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		details string
	}{
		{name: "explicit cache alias only", details: `{"cache_creation_input_tokens":12}`},
		{name: "cache lifetime only", details: `{"cache_write_tokens_details":{"ttl_30m_tokens":3,"ttl_12h_tokens":7}}`},
		{name: "explicit zero lifetimes", details: `{"cache_write_tokens_details":{"ttl_30m_tokens":0,"ttl_12h_tokens":0}}`},
		{name: "all aliases", details: `{"cached_tokens":3,"cached_creation_tokens":8,"cache_write_tokens":10,"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":3,"ttl_12h_tokens":7}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chat dto.OpenAITextResponse
			require.NoError(t, kitutil.Unmarshal([]byte(`{"id":"chatcmpl_cache","model":"gpt-test","usage":{"prompt_tokens":30,"completion_tokens":2,"total_tokens":32,"usage_semantic":"openai","usage_source":"oai_chat","claude_cache_creation_5_m_tokens":2,"claude_cache_creation_1_h_tokens":6,"prompt_tokens_details":`+tt.details+`}}`), &chat))
			expected := chat.Usage.PromptTokensDetails.Clone()

			toResponses, err := ConvertResponse(nil, nil, types.RelayFormatOpenAIResponses, &chat)
			require.NoError(t, err)
			require.IsType(t, &dto.OpenAIResponsesResponse{}, toResponses.Value)
			responses := toResponses.Value.(*dto.OpenAIResponsesResponse)
			require.NotNil(t, responses.Usage.InputTokensDetails)
			assert.Equal(t, expected, *responses.Usage.InputTokensDetails)
			require.NotNil(t, responses.Usage.BillingUsage)
			require.NotNil(t, responses.Usage.BillingUsage.OpenAIUsage)
			assert.Equal(t, expected, responses.Usage.BillingUsage.OpenAIUsage.PromptTokensDetails)

			wire, err := kitutil.Marshal(responses)
			require.NoError(t, err)
			var decoded dto.OpenAIResponsesResponse
			require.NoError(t, kitutil.Unmarshal(wire, &decoded))
			toChat, err := ConvertResponse(nil, nil, types.RelayFormatOpenAI, &decoded)
			require.NoError(t, err)
			assert.Equal(t, expected, toChat.Usage.PromptTokensDetails)
			assert.Equal(t, "openai", toChat.Usage.UsageSemantic)
			assert.Equal(t, "oai_chat", toChat.Usage.UsageSource)
			assert.Equal(t, 30, toChat.Usage.PromptTokens)
			assert.Equal(t, 32, toChat.Usage.TotalTokens)
			assert.Equal(t, 2, toChat.Usage.ClaudeCacheCreation5mTokens)
			assert.Equal(t, 6, toChat.Usage.ClaudeCacheCreation1hTokens)
			require.NotNil(t, toChat.Usage.BillingUsage.OpenAIUsage)
			assert.Equal(t, dto.BillingUsageSourceOAIChat, toChat.Usage.BillingUsage.Source)
			assert.Equal(t, expected, toChat.Usage.BillingUsage.OpenAIUsage.PromptTokensDetails)

			back, err := ConvertResponse(nil, nil, types.RelayFormatOpenAIResponses, toChat.Value)
			require.NoError(t, err)
			assert.Equal(t, expected, *back.Usage.InputTokensDetails)
			if expected.CacheWriteTokensDetails != nil {
				require.NotSame(t, chat.Usage.PromptTokensDetails.CacheWriteTokensDetails, responses.Usage.InputTokensDetails.CacheWriteTokensDetails)
				require.NotSame(t, decoded.Usage.InputTokensDetails.CacheWriteTokensDetails, toChat.Usage.PromptTokensDetails.CacheWriteTokensDetails)
				require.NotSame(t, toChat.Usage.PromptTokensDetails.CacheWriteTokensDetails, back.Usage.InputTokensDetails.CacheWriteTokensDetails)
				responses.Usage.InputTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 99
				back.Usage.InputTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 98
				assert.Equal(t, expected, chat.Usage.PromptTokensDetails)
				assert.Equal(t, expected, toChat.Usage.PromptTokensDetails)
				assert.Equal(t, expected, responses.Usage.BillingUsage.OpenAIUsage.PromptTokensDetails)
				assert.Equal(t, expected, back.Usage.BillingUsage.OpenAIUsage.PromptTokensDetails)
			}
		})
	}
}

func TestOpenAICacheBillingSnapshotSurvivesClaudeAndGeminiRoundTrip(t *testing.T) {
	for _, target := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatGemini} {
		t.Run(string(target), func(t *testing.T) {
			chat := &dto.OpenAITextResponse{
				Id:    "chatcmpl_cache",
				Model: "gpt-test",
				Usage: dto.Usage{
					PromptTokens:     30,
					CompletionTokens: 2,
					TotalTokens:      32,
					PromptTokensDetails: dto.InputTokenDetails{
						CacheCreationInputTokens: 12,
						CacheWriteTokensDetails:  &dto.CacheWriteTokenDetails{TTL30mTokens: 5, TTL12hTokens: 7},
					},
				},
			}
			info := &convmeta.Values{ChannelMetaAttached: true, UpstreamModelName: "gpt-test"}
			converted, err := ConvertResponse(nil, info, target, chat)
			require.NoError(t, err)
			wire, err := kitutil.Marshal(converted.Value)
			require.NoError(t, err)
			var decoded any
			if target == types.RelayFormatClaude {
				decoded = &dto.ClaudeResponse{}
			} else {
				decoded = &dto.GeminiChatResponse{}
			}
			require.NoError(t, kitutil.Unmarshal(wire, decoded))
			back, err := ConvertResponse(nil, info, types.RelayFormatOpenAI, decoded)
			require.NoError(t, err)
			require.NotNil(t, back.Usage.BillingUsage)
			require.NotNil(t, back.Usage.BillingUsage.OpenAIUsage)
			snapshot := back.Usage.BillingUsage.OpenAIUsage
			assert.Equal(t, dto.BillingUsageSemanticOpenAI, back.Usage.BillingUsage.Semantic)
			assert.Equal(t, 30, snapshot.PromptTokens)
			assert.Equal(t, 32, snapshot.TotalTokens)
			assert.Equal(t, chat.Usage.PromptTokensDetails, snapshot.PromptTokensDetails)
			chat.Usage.PromptTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 99
			assert.Equal(t, 7, snapshot.PromptTokensDetails.CacheWrite12hTokens())
		})
	}
}

func TestOpenAIStreamCacheUsageMergeAndRoundTrip(t *testing.T) {
	state, err := NewResponseStreamState(types.RelayFormatOpenAI, types.RelayFormatOpenAIResponses, ResponseStreamOptions{ID: "resp_cache", Model: "gpt-test"})
	require.NoError(t, err)
	for _, raw := range []string{
		`{"choices":[],"usage":{"prompt_tokens":30,"total_tokens":30,"prompt_tokens_details":{"cache_write_tokens":10,"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":3,"ttl_12h_tokens":7}}}}`,
		`{"choices":[],"usage":{"completion_tokens":2,"prompt_tokens_details":{"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":0}}}}`,
	} {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, kitutil.Unmarshal([]byte(raw), &chunk))
		_, err = ConvertStreamResponseChunk(nil, nil, state, &chunk)
		require.NoError(t, err)
	}
	results, err := FinalizeStreamResponse(nil, nil, state)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	event, ok := results[len(results)-1].Value.(ChatToResponsesStreamEvent)
	require.True(t, ok)
	assert.Equal(t, "response.completed", event.Type)
	require.NotNil(t, event.Payload.Response.Usage.InputTokensDetails)
	expected := dto.InputTokenDetails{CacheWriteTokens: 10, CacheCreationInputTokens: 12, CacheWriteTokensDetails: &dto.CacheWriteTokenDetails{TTL30mTokens: 5, TTL12hTokens: 7}}
	assert.Equal(t, expected, *event.Payload.Response.Usage.InputTokensDetails)
	assert.Equal(t, 12, event.Payload.Response.Usage.InputTokensDetails.CacheCreationTokensTotal())

	back, err := NewResponseStreamState(types.RelayFormatOpenAIResponses, types.RelayFormatOpenAI, ResponseStreamOptions{IncludeUsage: true})
	require.NoError(t, err)
	// An earlier Responses snapshot reports the prompt and cache counters.
	var created dto.ResponsesStreamResponse
	require.NoError(t, kitutil.Unmarshal([]byte(`{"type":"response.created","response":{"usage":{"input_tokens":30,"input_tokens_details":{"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7}}}}}`), &created))
	_, err = ConvertStreamResponseChunk(nil, nil, back, &created)
	require.NoError(t, err)
	wire, err := kitutil.Marshal(event.Payload)
	require.NoError(t, err)
	var terminal dto.ResponsesStreamResponse
	require.NoError(t, kitutil.Unmarshal(wire, &terminal))
	chunks, err := ConvertStreamResponseChunk(nil, nil, back, &terminal)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)
	chunk, ok := chunks[len(chunks)-1].Value.(dto.ChatCompletionsStreamResponse)
	require.True(t, ok)
	require.NotNil(t, chunk.Usage)
	assert.Equal(t, 30, chunk.Usage.PromptTokens)
	assert.Equal(t, 2, chunk.Usage.CompletionTokens)
	assert.Equal(t, 32, chunk.Usage.TotalTokens)
	assert.Equal(t, expected, chunk.Usage.PromptTokensDetails)
	require.NotNil(t, chunk.Usage.BillingUsage.OpenAIUsage)
	assert.Equal(t, expected, chunk.Usage.BillingUsage.OpenAIUsage.PromptTokensDetails)
	terminal.Response.Usage.InputTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 1
	assert.Equal(t, 7, chunk.Usage.PromptTokensDetails.CacheWrite12hTokens())
}

func textRegistryChatResponse() *dto.OpenAITextResponse {
	msg := dto.Message{
		Role:    "assistant",
		Content: "hello",
	}
	msg.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   "call_1",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "lookup",
				Arguments: `{"q":"x"}`,
			},
		},
	})
	return &dto.OpenAITextResponse{
		Id:      "chatcmpl_1",
		Model:   "gpt-test",
		Created: 123,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: "tool_calls",
			},
		},
		Usage: dto.Usage{PromptTokens: 4, CompletionTokens: 5, TotalTokens: 9},
	}
}

func textRegistryResponsesResponse() *dto.OpenAIResponsesResponse {
	return &dto.OpenAIResponsesResponse{
		ID:        "resp_1",
		CreatedAt: 123,
		Model:     "gpt-test",
		Status:    []byte(`"completed"`),
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "hello"},
				},
			},
			{
				Type:      "function_call",
				ID:        "call_1",
				CallId:    "call_1",
				Name:      "lookup",
				Arguments: []byte(`{"q":"x"}`),
			},
		},
		Usage: &dto.Usage{InputTokens: 4, OutputTokens: 7, TotalTokens: 11},
	}
}

func respPtr[T any](value T) *T {
	return &value
}
