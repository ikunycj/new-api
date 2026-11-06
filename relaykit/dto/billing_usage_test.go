package dto

import (
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeminiChatBillingUsageRequiresTokenContent(t *testing.T) {
	require.Nil(t, NewGeminiChatBillingUsage(nil))
	require.Nil(t, NewGeminiChatBillingUsage(&GeminiUsageMetadata{}))

	billingUsage := NewGeminiChatBillingUsage(&GeminiUsageMetadata{PromptTokenCount: 1})
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.GeminiUsageMetadata)
	assert.Equal(t, BillingUsageSourceGeminiChat, billingUsage.Source)
	assert.Equal(t, BillingUsageSemanticGemini, billingUsage.Semantic)
	assert.False(t, billingUsage.Estimated)
}

func TestNewClaudeMessagesBillingUsageRequiresTokenContent(t *testing.T) {
	require.Nil(t, NewClaudeMessagesBillingUsage(nil))
	require.Nil(t, NewClaudeMessagesBillingUsage(&ClaudeUsage{}))
	require.Nil(t, NewClaudeMessagesBillingUsage(&ClaudeUsage{CacheCreation: &ClaudeCacheCreationUsage{}}))

	billingUsage := NewClaudeMessagesBillingUsage(&ClaudeUsage{InputTokens: 1})
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.ClaudeUsage)
	assert.Equal(t, BillingUsageSourceClaudeMessages, billingUsage.Source)
	assert.Equal(t, BillingUsageSemanticAnthropic, billingUsage.Semantic)

	cacheOnly := NewClaudeMessagesBillingUsage(&ClaudeUsage{
		CacheCreation: &ClaudeCacheCreationUsage{Ephemeral5mInputTokens: 4},
	})
	require.NotNil(t, cacheOnly)
}

func TestNewOpenAIChatBillingUsageRequiresTokenContent(t *testing.T) {
	require.Nil(t, NewOpenAIChatBillingUsage(nil))
	require.Nil(t, NewOpenAIChatBillingUsage(&Usage{}))

	billingUsage := NewOpenAIChatBillingUsage(&Usage{PromptTokens: 1})
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.OpenAIUsage)
	assert.Equal(t, BillingUsageSourceOAIChat, billingUsage.Source)
	assert.Equal(t, BillingUsageSemanticOpenAI, billingUsage.Semantic)
	assert.Equal(t, 1, billingUsage.OpenAIUsage.PromptTokens)
}

func TestNewEstimatedGeminiChatBillingUsage(t *testing.T) {
	billingUsage := NewEstimatedGeminiChatBillingUsage(&Usage{
		PromptTokens:     11,
		CompletionTokens: 7,
	})

	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.GeminiUsageMetadata)
	assert.True(t, billingUsage.Estimated)
	assert.Equal(t, 11, billingUsage.GeminiUsageMetadata.PromptTokenCount)
	assert.Equal(t, 7, billingUsage.GeminiUsageMetadata.CandidatesTokenCount)
	assert.Equal(t, 18, billingUsage.GeminiUsageMetadata.TotalTokenCount)
}

func TestBillingUsageJSONUsesProtocolNamedFields(t *testing.T) {
	billingUsage := &BillingUsage{
		OpenAIUsage:         &Usage{PromptTokens: 1, BillingUsage: NewClaudeMessagesBillingUsage(&ClaudeUsage{InputTokens: 9})},
		ClaudeUsage:         &ClaudeUsage{InputTokens: 2, BillingUsage: NewOpenAIChatBillingUsage(&Usage{PromptTokens: 8})},
		GeminiUsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 3, BillingUsage: NewOpenAIChatBillingUsage(&Usage{PromptTokens: 7})},
	}

	data, err := kitutil.Marshal(billingUsage)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"openai_usage"`)
	assert.Contains(t, string(data), `"claude_usage"`)
	assert.Contains(t, string(data), `"gemini_usage_metadata"`)
	assert.NotContains(t, string(data), `"usage":`)
	assert.NotContains(t, string(data), `"usage_metadata"`)

	clone := CloneBillingUsage(billingUsage)
	require.NotNil(t, clone.OpenAIUsage)
	require.NotNil(t, clone.ClaudeUsage)
	require.NotNil(t, clone.GeminiUsageMetadata)
	assert.Nil(t, clone.OpenAIUsage.BillingUsage)
	assert.Nil(t, clone.ClaudeUsage.BillingUsage)
	assert.Nil(t, clone.GeminiUsageMetadata.BillingUsage)
}

func TestInputTokenDetailsCacheCreationAliasesAnd12hBounds(t *testing.T) {
	tests := []struct {
		name    string
		details InputTokenDetails
		total   int
		long    int
	}{
		{name: "empty"},
		{name: "native", details: InputTokenDetails{CacheWriteTokens: 10}, total: 10},
		{name: "claude alias", details: InputTokenDetails{CachedCreationTokens: 12}, total: 12},
		{name: "explicit cache alias", details: InputTokenDetails{CacheCreationInputTokens: 14}, total: 14},
		{name: "equal aliases are not added", details: InputTokenDetails{CacheWriteTokens: 10, CachedCreationTokens: 10, CacheCreationInputTokens: 10}, total: 10},
		{name: "largest alias wins", details: InputTokenDetails{CacheWriteTokens: 10, CachedCreationTokens: 12, CacheCreationInputTokens: 14}, total: 14},
		{name: "negative aliases", details: InputTokenDetails{CacheWriteTokens: -10, CachedCreationTokens: -12, CacheCreationInputTokens: -14}},
		{name: "split is not a total", details: InputTokenDetails{CacheWriteTokensDetails: &CacheWriteTokenDetails{TTL30mTokens: 5, TTL12hTokens: 7}}},
		{name: "12h subset", details: InputTokenDetails{CacheWriteTokens: 10, CacheWriteTokensDetails: &CacheWriteTokenDetails{TTL30mTokens: 3, TTL12hTokens: 7}}, total: 10, long: 7},
		{name: "12h clamped to alias total", details: InputTokenDetails{CacheCreationInputTokens: 10, CacheWriteTokensDetails: &CacheWriteTokenDetails{TTL12hTokens: 15}}, total: 10, long: 10},
		{name: "negative 12h", details: InputTokenDetails{CacheWriteTokens: 10, CacheWriteTokensDetails: &CacheWriteTokenDetails{TTL12hTokens: -1}}, total: 10},
		{name: "negative total with positive 12h", details: InputTokenDetails{CacheWriteTokens: -10, CacheWriteTokensDetails: &CacheWriteTokenDetails{TTL12hTokens: 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.total, tt.details.CacheCreationTokensTotal())
			assert.Equal(t, tt.long, tt.details.CacheWrite12hTokens())
		})
	}
}

func TestOpenAICacheUsageJSONPreservesChatAndResponsesFields(t *testing.T) {
	const detailsJSON = `{"cached_tokens":3,"cached_creation_tokens":8,"cache_write_tokens":10,"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":7},"text_tokens":20,"audio_tokens":2,"image_tokens":1}`
	tests := []struct {
		name string
		body string
	}{
		{name: "chat", body: `{"usage":{"prompt_tokens":30,"completion_tokens":2,"total_tokens":32,"prompt_tokens_details":` + detailsJSON + `}}`},
		{name: "responses", body: `{"usage":{"input_tokens":30,"output_tokens":2,"total_tokens":32,"input_tokens_details":` + detailsJSON + `}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload SimpleResponse
			require.NoError(t, kitutil.Unmarshal([]byte(tt.body), &payload))
			details := payload.PromptTokensDetails
			field := "prompt_tokens_details"
			if tt.name == "responses" {
				require.NotNil(t, payload.InputTokensDetails)
				details = *payload.InputTokensDetails
				field = "input_tokens_details"
			}
			assert.Equal(t, 12, details.CacheCreationInputTokens)
			assert.Equal(t, 12, details.CacheCreationTokensTotal())
			require.NotNil(t, details.CacheWriteTokensDetails)
			assert.Equal(t, 5, details.CacheWriteTokensDetails.TTL30mTokens)
			assert.Equal(t, 7, details.CacheWrite12hTokens())

			encoded, err := kitutil.Marshal(payload)
			require.NoError(t, err)
			var wire struct {
				Usage map[string]any `json:"usage"`
			}
			require.NoError(t, kitutil.Unmarshal(encoded, &wire))
			encodedDetails, err := kitutil.Marshal(wire.Usage[field])
			require.NoError(t, err)
			assert.JSONEq(t, detailsJSON, string(encodedDetails))
		})
	}
	encoded, err := kitutil.Marshal(InputTokenDetails{})
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), `"cache_creation_input_tokens"`)
	assert.NotContains(t, string(encoded), `"cache_write_tokens_details"`)
	encoded, err = kitutil.Marshal(CacheWriteTokenDetails{})
	require.NoError(t, err)
	assert.JSONEq(t, `{"ttl_30m_tokens":0,"ttl_12h_tokens":0}`, string(encoded))
}

func TestOpenAIBillingUsageRequiresNonZeroCacheContent(t *testing.T) {
	tests := []struct {
		name    string
		details *CacheWriteTokenDetails
		alias   int
		want    bool
	}{
		{name: "nil"},
		{name: "zero", details: &CacheWriteTokenDetails{}},
		{name: "30m", details: &CacheWriteTokenDetails{TTL30mTokens: 4}, want: true},
		{name: "12h", details: &CacheWriteTokenDetails{TTL12hTokens: 6}, want: true},
		{name: "negative is still reported", details: &CacheWriteTokenDetails{TTL12hTokens: -6}, want: true},
		{name: "explicit cache alias", alias: 8, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want && tt.alias == 0, tt.details.HasTokens())
			details := InputTokenDetails{CacheWriteTokensDetails: tt.details, CacheCreationInputTokens: tt.alias}
			for _, usage := range []*Usage{{PromptTokensDetails: details}, {InputTokensDetails: &details}} {
				assert.Equal(t, tt.want, HasOpenAIUsageTokens(usage))
				assert.Equal(t, tt.want, NewOpenAIChatBillingUsage(usage) != nil)
				assert.Equal(t, tt.want, NewOpenAIResponsesBillingUsage(usage) != nil)
			}
		})
	}
}

func TestOpenAIBillingUsageAndDetailsCloneDetachCachePointers(t *testing.T) {
	details := InputTokenDetails{
		CacheCreationInputTokens: 10,
		CacheWriteTokensDetails:  &CacheWriteTokenDetails{TTL30mTokens: 3, TTL12hTokens: 7},
	}
	clonedDetails := details.Clone()
	require.NotSame(t, details.CacheWriteTokensDetails, clonedDetails.CacheWriteTokensDetails)
	clonedDetails.CacheWriteTokensDetails.TTL12hTokens = 1
	assert.Equal(t, 7, details.CacheWriteTokensDetails.TTL12hTokens)
	assert.Nil(t, (InputTokenDetails{}).Clone().CacheWriteTokensDetails)

	usage := &Usage{PromptTokensDetails: details, InputTokensDetails: &details, BillingUsage: &BillingUsage{Source: "nested"}}
	billing := NewOpenAIChatBillingUsage(usage)
	require.NotNil(t, billing)
	require.NotNil(t, billing.OpenAIUsage)
	assert.Nil(t, billing.OpenAIUsage.BillingUsage)
	require.NotSame(t, usage.InputTokensDetails, billing.OpenAIUsage.InputTokensDetails)
	require.NotSame(t, details.CacheWriteTokensDetails, billing.OpenAIUsage.PromptTokensDetails.CacheWriteTokensDetails)
	require.NotSame(t, details.CacheWriteTokensDetails, billing.OpenAIUsage.InputTokensDetails.CacheWriteTokensDetails)
	require.NotSame(t, billing.OpenAIUsage.PromptTokensDetails.CacheWriteTokensDetails, billing.OpenAIUsage.InputTokensDetails.CacheWriteTokensDetails)

	clone := CloneBillingUsage(billing)
	require.NotNil(t, clone.OpenAIUsage)
	clone.OpenAIUsage.PromptTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 2
	clone.OpenAIUsage.InputTokensDetails.CacheWriteTokensDetails.TTL30mTokens = 1
	usage.PromptTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 9
	assert.Equal(t, 7, billing.OpenAIUsage.PromptTokensDetails.CacheWriteTokensDetails.TTL12hTokens)
	assert.Equal(t, 3, billing.OpenAIUsage.InputTokensDetails.CacheWriteTokensDetails.TTL30mTokens)
}

func TestMergeUsagePreservesCacheSnapshotsWithoutAddingAliases(t *testing.T) {
	var first, second Usage
	require.NoError(t, kitutil.Unmarshal([]byte(`{"input_tokens":30,"total_tokens":30,"input_tokens_details":{"cache_write_tokens":10,"cached_creation_tokens":10,"cache_creation_input_tokens":10,"cache_write_tokens_details":{"ttl_30m_tokens":3,"ttl_12h_tokens":7}}}`), &first))
	require.NoError(t, kitutil.Unmarshal([]byte(`{"output_tokens":2,"total_tokens":32,"input_tokens_details":{"cache_creation_input_tokens":12,"cache_write_tokens_details":{"ttl_30m_tokens":5,"ttl_12h_tokens":0}}}`), &second))
	first.PromptTokensDetails = first.InputTokensDetails.Clone()
	second.PromptTokensDetails = second.InputTokensDetails.Clone()
	first.BillingUsage = NewOpenAIResponsesBillingUsage(&first)
	second.BillingUsage = NewOpenAIResponsesBillingUsage(&second)

	merged := MergeUsageNonZero(nil, &first)
	merged = MergeUsageNonZero(merged, &second)
	require.NotNil(t, merged.BillingUsage)
	require.NotNil(t, merged.BillingUsage.OpenAIUsage)
	assert.Equal(t, 30, merged.InputTokens)
	assert.Equal(t, 2, merged.OutputTokens)
	assert.Equal(t, 32, merged.TotalTokens)
	for _, details := range []*InputTokenDetails{&merged.PromptTokensDetails, merged.InputTokensDetails, merged.BillingUsage.OpenAIUsage.InputTokensDetails} {
		require.NotNil(t, details)
		assert.Equal(t, 10, details.CacheWriteTokens)
		assert.Equal(t, 10, details.CachedCreationTokens)
		assert.Equal(t, 12, details.CacheCreationTokensTotal())
		require.NotNil(t, details.CacheWriteTokensDetails)
		assert.Equal(t, 5, details.CacheWriteTokensDetails.TTL30mTokens)
		assert.Equal(t, 7, details.CacheWrite12hTokens())
	}
	assert.Equal(t, 30, merged.BillingUsage.OpenAIUsage.InputTokens)
	assert.Equal(t, 2, merged.BillingUsage.OpenAIUsage.OutputTokens)
	first.InputTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 1
	second.InputTokensDetails.CacheWriteTokensDetails.TTL30mTokens = 1
	merged.PromptTokensDetails.CacheWriteTokensDetails.TTL12hTokens = 2
	assert.Equal(t, 7, merged.InputTokensDetails.CacheWrite12hTokens())
	assert.Equal(t, 7, merged.BillingUsage.OpenAIUsage.InputTokensDetails.CacheWrite12hTokens())
	assert.Equal(t, 5, merged.InputTokensDetails.CacheWriteTokensDetails.TTL30mTokens)
}

func TestMergeUsageLeavesClaudeSnapshotAuthoritative(t *testing.T) {
	current := &Usage{PromptTokens: 4, InputTokens: 4, TotalTokens: 4}
	incoming := &Usage{
		CompletionTokens: 2,
		OutputTokens:     2,
		TotalTokens:      2,
		UsageSemantic:    BillingUsageSemanticOpenAI,
		UsageSource:      BillingUsageSemanticAnthropic,
		BillingUsage:     NewClaudeMessagesBillingUsage(&ClaudeUsage{OutputTokens: 2}),
	}
	merged := MergeUsageNonZero(current, incoming)
	assert.Equal(t, 0, merged.PromptTokens)
	assert.Equal(t, 0, merged.InputTokens)
	assert.Equal(t, 2, merged.TotalTokens)
	require.NotNil(t, merged.BillingUsage.ClaudeUsage)
	assert.Equal(t, 0, merged.BillingUsage.ClaudeUsage.InputTokens)
	assert.Equal(t, 2, merged.BillingUsage.ClaudeUsage.OutputTokens)
	require.NotSame(t, incoming.BillingUsage.ClaudeUsage, merged.BillingUsage.ClaudeUsage)
}
