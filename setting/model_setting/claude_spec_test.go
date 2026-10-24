package model_setting

import "testing"

func TestGetModelSpec(t *testing.T) {
	s := GetClaudeSettings()
	prevEnabled := s.ModelSpecValidationEnabled
	prevLimits := s.ModelSpecLimits
	prevEnabledDef := defaultClaudeModelSpecs["claude-opus-5"]
	defer func() {
		s.ModelSpecValidationEnabled = prevEnabled
		s.ModelSpecLimits = prevLimits
		defaultClaudeModelSpecs["claude-opus-5"] = prevEnabledDef
	}()
	s.ModelSpecLimits = map[string]ModelSpec{}
	s.ModelSpecValidationEnabled = true

	// 内置默认表生效
	spec, ok := s.GetModelSpec("claude-sonnet-4-6")
	if !ok || spec.MaxOutputTokens != 64000 || spec.ContextWindow != 200000 {
		t.Fatalf("unexpected sonnet spec: %+v ok=%v", spec, ok)
	}

	// 后台覆写优先于内置默认
	s.ModelSpecLimits["claude-sonnet-4-6"] = ModelSpec{ContextWindow: 100000, MaxOutputTokens: 32000}
	spec, ok = s.GetModelSpec("claude-sonnet-4-6")
	if !ok || spec.MaxOutputTokens != 32000 {
		t.Fatalf("expected override to win, got %+v", spec)
	}

	// 日期后缀别名命中
	spec, ok = s.GetModelSpec("claude-opus-5-20251101")
	if !ok || spec.MaxOutputTokens != 128000 {
		t.Fatalf("unexpected opus spec: %+v ok=%v", spec, ok)
	}

	// 未登记模型不拦截
	if _, ok := s.GetModelSpec("some-other-model"); ok {
		t.Fatalf("unexpected spec for unregistered model")
	}

	// 后台补录未登记模型
	s.ModelSpecLimits["some-other-model"] = ModelSpec{ContextWindow: 128000, MaxOutputTokens: 8192}
	if _, ok := s.GetModelSpec("some-other-model"); !ok {
		t.Fatalf("expected added spec to be found")
	}

	// 总开关关闭
	s.ModelSpecValidationEnabled = false
	if _, ok := s.GetModelSpec("claude-sonnet-4-6"); ok {
		t.Fatalf("spec should be unavailable when disabled")
	}
}
