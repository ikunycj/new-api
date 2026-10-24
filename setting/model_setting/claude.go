package model_setting

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

//var claudeHeadersSettings = map[string][]string{}
//
//var ClaudeThinkingAdapterEnabled = true
//var ClaudeThinkingAdapterMaxTokens = 8192
//var ClaudeThinkingAdapterBudgetTokensPercentage = 0.8

// ClaudeSettings 定义Claude模型的配置
type ClaudeSettings struct {
	HeadersSettings                       map[string]map[string][]string `json:"model_headers_settings"`
	DefaultMaxTokens                      map[string]int                 `json:"default_max_tokens"`
	ThinkingAdapterEnabled                bool                           `json:"thinking_adapter_enabled"`
	ThinkingAdapterBudgetTokensPercentage float64                        `json:"thinking_adapter_budget_tokens_percentage"`

	// ModelSpecValidationEnabled 开启后，Claude 原生请求在转发前按模型规格表
	// 预校验 max_tokens 与上下文占用（见 relay/claude_spec.go）。
	ModelSpecValidationEnabled bool `json:"model_spec_validation_enabled"`
	// ModelSpecLimits 按模型名覆写/补录规格；未登记的模型不做预校验。
	// 同名 key 优先于内置默认表，内置默认表无法通过后台删除。
	ModelSpecLimits map[string]ModelSpec `json:"model_spec_limits"`
}

// ModelSpec 描述单个模型的官方 API 规格。
type ModelSpec struct {
	// ContextWindow 上下文窗口（输入 token + 输出 token 的总数上限）。
	ContextWindow int `json:"context_window"`
	// MaxOutputTokens 单次输出（max_tokens）上限。
	MaxOutputTokens int `json:"max_output_tokens"`
}

// 内置默认规格，与官方 API 的参数校验语义对齐。
//
// 背景：部分订阅号池上游对超限请求不报错（照常计费）或参数校验口径与官方
// 不一致，客户端拿不到预期 4xx。这里的默认值只覆盖确认过的模型，更多模型
// 通过后台 model_spec_limits 补录，不存在兜底上限——未登记即不拦截。
var defaultClaudeModelSpecs = map[string]ModelSpec{
	"claude-sonnet-4-6": {ContextWindow: 200000, MaxOutputTokens: 64000},
	"claude-opus-5":     {ContextWindow: 200000, MaxOutputTokens: 128000},
}

// claudeModelDateSuffix 匹配模型名上的日期别名后缀，如 claude-opus-5-20251101。
var claudeModelDateSuffix = regexp.MustCompile(`-\d{8}$`)

// 默认配置
var defaultClaudeSettings = ClaudeSettings{
	HeadersSettings:        map[string]map[string][]string{},
	ThinkingAdapterEnabled: true,
	DefaultMaxTokens: map[string]int{
		"default": 8192,
	},
	ThinkingAdapterBudgetTokensPercentage: 0.8,
	ModelSpecValidationEnabled:            true,
	ModelSpecLimits:                       map[string]ModelSpec{},
}

// 全局实例
var claudeSettings = defaultClaudeSettings

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("claude", &claudeSettings)
}

// GetClaudeSettings 获取Claude配置
func GetClaudeSettings() *ClaudeSettings {
	// check default max tokens must have default key
	if _, ok := claudeSettings.DefaultMaxTokens["default"]; !ok {
		claudeSettings.DefaultMaxTokens["default"] = 8192
	}
	return &claudeSettings
}

func (c *ClaudeSettings) WriteHeaders(originModel string, httpHeader *http.Header) {
	if headers, ok := c.HeadersSettings[originModel]; ok {
		for headerKey, headerValues := range headers {
			mergedValues := normalizeHeaderListValues(
				append(append([]string(nil), httpHeader.Values(headerKey)...), headerValues...),
			)
			if len(mergedValues) == 0 {
				continue
			}
			httpHeader.Set(headerKey, strings.Join(mergedValues, ","))
		}
	}
}

func normalizeHeaderListValues(values []string) []string {
	normalizedValues := make([]string, 0, len(values))
	seenValues := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			normalizedItem := strings.TrimSpace(item)
			if normalizedItem == "" {
				continue
			}
			if _, exists := seenValues[normalizedItem]; exists {
				continue
			}
			seenValues[normalizedItem] = struct{}{}
			normalizedValues = append(normalizedValues, normalizedItem)
		}
	}
	return normalizedValues
}

func (c *ClaudeSettings) GetDefaultMaxTokens(model string) int {
	if maxTokens, ok := c.DefaultMaxTokens[model]; ok {
		return maxTokens
	}
	return c.DefaultMaxTokens["default"]
}

// GetModelSpec 返回模型的规格限制；未登记的模型返回 false，不做预校验。
//
// 匹配顺序：后台覆写精确名 → 内置默认精确名 → 剥掉 -YYYYMMDD 日期后缀再查
// 两张表。同名复写优先于内置默认。
func (c *ClaudeSettings) GetModelSpec(model string) (ModelSpec, bool) {
	if !c.ModelSpecValidationEnabled || model == "" {
		return ModelSpec{}, false
	}
	base := claudeModelDateSuffix.ReplaceAllString(model, "")
	for _, key := range []string{model, base} {
		if spec, ok := c.ModelSpecLimits[key]; ok {
			return spec, true
		}
		if spec, ok := defaultClaudeModelSpecs[key]; ok {
			return spec, true
		}
	}
	return ModelSpec{}, false
}
