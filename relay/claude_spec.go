package relay

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"
)

// Claude 模型规格预校验。
//
// 部分订阅号池上游对超限请求不报 4xx（max_tokens 超输出上限、上下文爆窗都会
// 照单全收并计费），客户端拿不到官方语义的错误。这里在建立上游连接之前按
// 模型规格表拦截两类请求：
//
//   - max_tokens 超过模型输出上限；
//   - 估算输入 token + max_tokens 超过模型上下文窗口。
//
// 输入 token 复用计费预扣的同一套估算器（CountTextToken → EstimateTokenByModel，
// CJK 按字计、英文按词计）。估算器与上游真实 tokenizer 存在小幅偏差，因此
// 只在明显超过上限时拒绝；图片类媒体不参与输入估算（见 GetTokenCountMeta）。
// 拦截不发上游、不计费，错误码与官方一致（invalid_request_error / HTTP 400）。
func validateClaudeModelSpec(request *dto.ClaudeRequest) *types.NewAPIError {
	if request == nil || request.Model == "" {
		return nil
	}
	spec, ok := model_setting.GetClaudeSettings().GetModelSpec(request.Model)
	if !ok {
		return nil
	}

	maxTokens := 0
	if request.MaxTokens != nil {
		maxTokens = int(*request.MaxTokens)
	}

	if maxTokens > spec.MaxOutputTokens {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("max_tokens: %d > %d, which is the maximum allowed number of output tokens for %s",
				maxTokens, spec.MaxOutputTokens, request.Model),
			types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	meta := request.GetTokenCountMeta()
	if meta == nil {
		return nil
	}
	inputEst := service.CountTextToken(meta.CombineText, request.Model)
	if inputEst+maxTokens > spec.ContextWindow {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("prompt (estimated at %d tokens) plus max_tokens (%d) exceeds the context window of %d tokens for this model: %s",
				inputEst, maxTokens, spec.ContextWindow, request.Model),
			types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	return nil
}
