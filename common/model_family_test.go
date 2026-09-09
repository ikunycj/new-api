package common

import "testing"

func TestChannelModelFamilyClassification(t *testing.T) {
	cases := map[string]string{
		"claude-sonnet-4-6":              ModelFamilyClaude,
		"claude-sonnet-4-5-20250929":     ModelFamilyClaude,
		"claude-opus-4-8":                ModelFamilyClaude,
		"CLAUDE-FABLE-5":                 ModelFamilyClaude,
		"gpt-5.4-mini":                   ModelFamilyGPT,
		"gpt-5.6-sol-openai-compact":     ModelFamilyGPT,
		"codex-auto-review":              ModelFamilyGPT,
		"gemini-2.5-pro":                 ModelFamilyGemini,
		"some-unrecognized-model":        ModelFamilyUnknown,
		"":                               ModelFamilyUnknown,
		"  claude-haiku-4-5-20251001   ": ModelFamilyClaude,
	}
	for input, want := range cases {
		if got := ChannelModelFamily(input); got != want {
			t.Fatalf("ChannelModelFamily(%q) = %q, want %q", input, got, want)
		}
	}
}

// A channel that serves the family's cheap model must be probed with exactly
// that model, so every channel in the family is measured on the same axis.
func TestSelectProbeModelPrefersFamilyCheapModel(t *testing.T) {
	claudeModels := []string{"claude-opus-4-8", "claude-sonnet-4-6", "claude-fable-5"}
	model, family := SelectProbeModel("", claudeModels)
	if model != "claude-sonnet-4-6" {
		t.Fatalf("expected claude cheap probe model, got %q", model)
	}
	if family != ModelFamilyClaude {
		t.Fatalf("expected claude family, got %q", family)
	}

	gptModels := []string{"gpt-5.6-sol", "gpt-5.4-mini", "gpt-image-2"}
	model, family = SelectProbeModel("", gptModels)
	if model != "gpt-5.4-mini" {
		t.Fatalf("expected gpt cheap probe model, got %q", model)
	}
	if family != ModelFamilyGPT {
		t.Fatalf("expected gpt family, got %q", family)
	}
}

// The preferred model must win even when the channel configured a different,
// more expensive test model: consistency across the family is what makes the
// scores comparable.
func TestSelectProbeModelOverridesConfiguredTestModelWhenAvailable(t *testing.T) {
	model, family := SelectProbeModel("gpt-5.6-terra", []string{"gpt-5.6-terra", "gpt-5.4-mini"})
	if model != "gpt-5.4-mini" {
		t.Fatalf("expected preferred cheap model to win, got %q", model)
	}
	if family != ModelFamilyGPT {
		t.Fatalf("expected gpt family, got %q", family)
	}
}

// A distillation-only channel carries gpt models but not gpt-5.4-mini. Probing
// it with a model it does not serve would record a bogus upstream failure, so
// the channel's own configuration must win instead.
func TestSelectProbeModelFallsBackWhenCheapModelUnavailable(t *testing.T) {
	distillOnly := []string{"gpt-5.5", "gpt-5.6-sol", "gpt-5.6-terra-openai-compact"}

	model, family := SelectProbeModel("gpt-5.5", distillOnly)
	if model != "gpt-5.5" {
		t.Fatalf("expected configured test model fallback, got %q", model)
	}
	if family != ModelFamilyGPT {
		t.Fatalf("expected gpt family, got %q", family)
	}

	// With no configured test model, the first advertised model is used.
	model, family = SelectProbeModel("", distillOnly)
	if model != "gpt-5.5" {
		t.Fatalf("expected first advertised model, got %q", model)
	}
	if family != ModelFamilyGPT {
		t.Fatalf("expected gpt family, got %q", family)
	}
}

func TestSelectProbeModelEmptyChannel(t *testing.T) {
	model, family := SelectProbeModel("", nil)
	if model != "" {
		t.Fatalf("expected no probe model, got %q", model)
	}
	if family != ModelFamilyUnknown {
		t.Fatalf("expected unknown family, got %q", family)
	}
}

// Blank entries in a channel's model list must not be selected as probe models.
func TestSelectProbeModelIgnoresBlankEntries(t *testing.T) {
	model, family := SelectProbeModel("", []string{"", "   ", "claude-sonnet-4-6"})
	if model != "claude-sonnet-4-6" {
		t.Fatalf("expected blank entries skipped, got %q", model)
	}
	if family != ModelFamilyClaude {
		t.Fatalf("expected claude family, got %q", family)
	}
}
