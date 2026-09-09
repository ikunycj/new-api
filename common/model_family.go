package common

import "strings"

// Model families group models whose health signals are comparable. Channel
// health is bucketed per family because a claude channel and a gpt channel have
// different latency envelopes and different upstream failure modes: mixing them
// into one score would make the "availability" input meaningless for both.
//
// The families are deliberately coarse. The goal is not taxonomy, it is to
// answer one question: may these two observations be averaged together?
const (
	ModelFamilyClaude  = "claude"
	ModelFamilyGPT     = "gpt"
	ModelFamilyGemini  = "gemini"
	ModelFamilyUnknown = "other"
)

// preferredProbeModelByFamily is the model a synthetic probe should use for each
// family. Probing every channel in a family with the SAME model is what makes
// the resulting scores fair: if channel A were probed with an expensive
// reasoning model and channel B with a small one, B would always look faster.
//
// The chosen models are the cheap workhorses of each family, because probes are
// pure cost: they buy information, not user value.
var preferredProbeModelByFamily = map[string]string{
	ModelFamilyClaude: "claude-sonnet-4-6",
	ModelFamilyGPT:    "gpt-5.4-mini",
}

// ChannelModelFamily classifies one model name into a family. Matching is by
// prefix on the normalized name so dated and suffixed variants
// (claude-sonnet-4-5-20250929, gpt-5.6-sol-openai-compact) land in the same
// bucket as their base model.
func ChannelModelFamily(modelName string) string {
	name := strings.ToLower(strings.TrimSpace(modelName))
	if name == "" {
		return ModelFamilyUnknown
	}
	switch {
	case strings.HasPrefix(name, "claude-"):
		return ModelFamilyClaude
	case strings.HasPrefix(name, "gpt-"), strings.HasPrefix(name, "o1"),
		strings.HasPrefix(name, "o3"), strings.HasPrefix(name, "codex"):
		return ModelFamilyGPT
	case strings.HasPrefix(name, "gemini-"):
		return ModelFamilyGemini
	default:
		return ModelFamilyUnknown
	}
}

// PreferredProbeModelForFamily returns the cheap probe model for a family and
// whether one is defined. Families without a designated probe model fall back to
// the channel's own configuration.
func PreferredProbeModelForFamily(family string) (string, bool) {
	preferred, ok := preferredProbeModelByFamily[family]
	return preferred, ok
}

// SelectProbeModel picks the model a probe should send to one channel.
//
// It prefers the family's shared cheap model so that scores within a family stay
// comparable, but only when the channel actually serves it. A channel that does
// not (for example a distillation-only channel that carries gpt-5.5 but not
// gpt-5.4-mini) must not be probed with a model it would reject: that would
// record an upstream failure that says nothing about the channel's health.
//
// Resolution order:
//  1. the family's preferred cheap model, if the channel serves it
//  2. the channel's configured test model, if any
//  3. the channel's first advertised model
//
// The returned family is derived from the model actually chosen, so the health
// bucket always matches what was measured.
func SelectProbeModel(configuredTestModel string, availableModels []string) (model string, family string) {
	normalized := make([]string, 0, len(availableModels))
	for _, candidate := range availableModels {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			normalized = append(normalized, candidate)
		}
	}

	// Family is inferred from the channel's own model list: a channel is
	// effectively single-family in practice, so the first recognized model is a
	// reliable signal of which family's probe model to aim for.
	inferredFamily := ModelFamilyUnknown
	for _, candidate := range normalized {
		if resolved := ChannelModelFamily(candidate); resolved != ModelFamilyUnknown {
			inferredFamily = resolved
			break
		}
	}

	if preferred, ok := PreferredProbeModelForFamily(inferredFamily); ok {
		for _, candidate := range normalized {
			if strings.EqualFold(candidate, preferred) {
				return preferred, inferredFamily
			}
		}
	}

	if configured := strings.TrimSpace(configuredTestModel); configured != "" {
		return configured, ChannelModelFamily(configured)
	}

	if len(normalized) > 0 {
		return normalized[0], ChannelModelFamily(normalized[0])
	}

	return "", ModelFamilyUnknown
}
