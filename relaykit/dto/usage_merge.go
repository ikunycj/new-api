package dto

import "strings"

// MergeUsageNonZero overlays cumulative OpenAI usage snapshots. Later positive
// counters replace earlier values, rather than adding aliases or losing counts
// omitted in a later frame. Mutable input details and billing snapshots are copied.
func MergeUsageNonZero(current *Usage, incoming *Usage) *Usage {
	if current == nil {
		current = &Usage{}
	}
	if incoming == nil {
		return current
	}
	// Provider-native non-OpenAI snapshots retain their existing authoritative
	// overwrite semantics, including zeros. This merger only aggregates OpenAI.
	if incoming.BillingUsage != nil && incoming.BillingUsage.OpenAIUsage == nil {
		replacement := cloneOpenAIUsage(incoming)
		replacement.BillingUsage = CloneBillingUsage(incoming.BillingUsage)
		return replacement
	}
	if incoming.PromptTokens > 0 {
		current.PromptTokens = incoming.PromptTokens
	}
	if incoming.CompletionTokens > 0 {
		current.CompletionTokens = incoming.CompletionTokens
	}
	if incoming.TotalTokens > 0 {
		current.TotalTokens = incoming.TotalTokens
	}
	if incoming.PromptCacheHitTokens > 0 {
		current.PromptCacheHitTokens = incoming.PromptCacheHitTokens
	}
	if incoming.InputTokens > 0 {
		current.InputTokens = incoming.InputTokens
	}
	if incoming.OutputTokens > 0 {
		current.OutputTokens = incoming.OutputTokens
	}
	if incoming.ClaudeCacheCreation5mTokens > 0 {
		current.ClaudeCacheCreation5mTokens = incoming.ClaudeCacheCreation5mTokens
	}
	if incoming.ClaudeCacheCreation1hTokens > 0 {
		current.ClaudeCacheCreation1hTokens = incoming.ClaudeCacheCreation1hTokens
	}

	mergeInputTokenDetails(&current.PromptTokensDetails, incoming.PromptTokensDetails)
	if incoming.InputTokensDetails != nil {
		if current.InputTokensDetails == nil {
			current.InputTokensDetails = &InputTokenDetails{}
		}
		mergeInputTokenDetails(current.InputTokensDetails, *incoming.InputTokensDetails)
	}
	if incoming.CompletionTokenDetails.TextTokens > 0 {
		current.CompletionTokenDetails.TextTokens = incoming.CompletionTokenDetails.TextTokens
	}
	if incoming.CompletionTokenDetails.AudioTokens > 0 {
		current.CompletionTokenDetails.AudioTokens = incoming.CompletionTokenDetails.AudioTokens
	}
	if incoming.CompletionTokenDetails.ImageTokens > 0 {
		current.CompletionTokenDetails.ImageTokens = incoming.CompletionTokenDetails.ImageTokens
	}
	if incoming.CompletionTokenDetails.ReasoningTokens > 0 {
		current.CompletionTokenDetails.ReasoningTokens = incoming.CompletionTokenDetails.ReasoningTokens
	}
	if incoming.UsageSemantic != "" {
		current.UsageSemantic = incoming.UsageSemantic
	}
	if incoming.UsageSource != "" {
		current.UsageSource = incoming.UsageSource
	}
	if incoming.BillingUsage != nil {
		previous := current.BillingUsage
		merged := CloneBillingUsage(incoming.BillingUsage)
		if previous != nil && previous.OpenAIUsage != nil && merged.OpenAIUsage != nil &&
			(previous.Source == "" || merged.Source == "" || strings.EqualFold(previous.Source, merged.Source)) &&
			(previous.Semantic == "" || merged.Semantic == "" || strings.EqualFold(previous.Semantic, merged.Semantic)) {
			merged.OpenAIUsage = MergeUsageNonZero(cloneOpenAIUsage(previous.OpenAIUsage), merged.OpenAIUsage)
			if merged.Source == "" {
				merged.Source = previous.Source
			}
			if merged.Semantic == "" {
				merged.Semantic = previous.Semantic
			}
			merged.Estimated = previous.Estimated || merged.Estimated
		}
		current.BillingUsage = merged
	}
	if incoming.Cost != nil {
		current.Cost = incoming.Cost
	}
	if total := current.PromptTokens + current.CompletionTokens; total > current.TotalTokens {
		current.TotalTokens = total
	}
	if total := current.InputTokens + current.OutputTokens; total > current.TotalTokens {
		current.TotalTokens = total
	}
	return current
}

func mergeInputTokenDetails(current *InputTokenDetails, incoming InputTokenDetails) {
	if incoming.CachedTokens > 0 {
		current.CachedTokens = incoming.CachedTokens
	}
	if incoming.CachedCreationTokens > 0 {
		current.CachedCreationTokens = incoming.CachedCreationTokens
	}
	if incoming.CacheWriteTokens > 0 {
		current.CacheWriteTokens = incoming.CacheWriteTokens
	}
	if incoming.CacheCreationInputTokens > 0 {
		current.CacheCreationInputTokens = incoming.CacheCreationInputTokens
	}
	if incoming.CacheWriteTokensDetails != nil {
		var merged CacheWriteTokenDetails
		if current.CacheWriteTokensDetails != nil {
			merged = *current.CacheWriteTokensDetails
		}
		if incoming.CacheWriteTokensDetails.TTL30mTokens > 0 {
			merged.TTL30mTokens = incoming.CacheWriteTokensDetails.TTL30mTokens
		}
		if incoming.CacheWriteTokensDetails.TTL12hTokens > 0 {
			merged.TTL12hTokens = incoming.CacheWriteTokensDetails.TTL12hTokens
		}
		current.CacheWriteTokensDetails = &merged
	}
	if incoming.TextTokens > 0 {
		current.TextTokens = incoming.TextTokens
	}
	if incoming.AudioTokens > 0 {
		current.AudioTokens = incoming.AudioTokens
	}
	if incoming.ImageTokens > 0 {
		current.ImageTokens = incoming.ImageTokens
	}
}
