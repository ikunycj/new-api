package relay

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func claudeSpecTestRequest(model string, maxTokens uint, text string) *dto.ClaudeRequest {
	return &dto.ClaudeRequest{
		Model:     model,
		MaxTokens: common.GetPointer[uint](maxTokens),
		Messages:  []dto.ClaudeMessage{{Role: "user", Content: text}},
	}
}

func TestValidateClaudeModelSpec(t *testing.T) {
	settings := model_setting.GetClaudeSettings()
	prevEnabled := settings.ModelSpecValidationEnabled
	defer func() { settings.ModelSpecValidationEnabled = prevEnabled }()
	settings.ModelSpecValidationEnabled = true

	// 1. max_tokens 超过模型输出上限 → 400
	err := validateClaudeModelSpec(claudeSpecTestRequest("claude-sonnet-4-6", 66536, "hi"))
	if err == nil || !strings.Contains(err.Error(), "maximum allowed number of output tokens") {
		t.Fatalf("expected max_tokens rejection, got %v", err)
	}

	// 2. 正常请求不拦
	if err := validateClaudeModelSpec(claudeSpecTestRequest("claude-sonnet-4-6", 8192, "hi")); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}

	// 3. CJK 大载荷撞上下文窗口 → 400（估算约 31.4 万 > 20 万）
	huge := strings.Repeat("数据分析。", 60000)
	err = validateClaudeModelSpec(claudeSpecTestRequest("claude-opus-5", 4096, huge))
	if err == nil || !strings.Contains(err.Error(), "context window") {
		t.Fatalf("expected context window rejection, got %v", err)
	}

	// 4. 日期后缀别名同样命中规格表
	err = validateClaudeModelSpec(claudeSpecTestRequest("claude-sonnet-4-6-20260101", 66536, "hi"))
	if err == nil || !strings.Contains(err.Error(), "maximum allowed number of output tokens") {
		t.Fatalf("expected date-suffix alias rejection, got %v", err)
	}

	// 5. 未登记的模型不校验
	if err := validateClaudeModelSpec(claudeSpecTestRequest("unknown-model-x", 999999, "hi")); err != nil {
		t.Fatalf("expected pass for unregistered model, got %v", err)
	}

	// 6. 总开关关闭时不校验
	settings.ModelSpecValidationEnabled = false
	if err := validateClaudeModelSpec(claudeSpecTestRequest("claude-sonnet-4-6", 66536, "hi")); err != nil {
		t.Fatalf("expected pass when validation disabled, got %v", err)
	}
}
