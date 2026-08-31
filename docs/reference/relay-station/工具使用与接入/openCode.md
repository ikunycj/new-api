# OpenCode

在 OpenCode 中保存 All Token API 凭据，并将其注册为自定义 OpenAI 兼容服务商。

> [!info] 配置核验
> 本文主流程将 API Key 直接写入用户级 `opencode.json`，因此关闭终端后仍然有效。也可以使用 OpenCode 的 `/connect` 保存凭据，但不要同时依赖两种来源。

## 1. 选择配置范围

OpenCode 支持 JSON 和 JSONC，多个配置源会合并，后加载的配置覆盖冲突字段。

| 范围 | 配置文件 |
| --- | --- |
| 全局 | `~/.config/opencode/opencode.json` |
| 项目 | 项目根目录的 `opencode.json` |
| 自定义文件 | `OPENCODE_CONFIG` 指向的文件 |

服务商通常适合放在全局配置。项目配置可以提交到 Git，因此不要在其中写明文密钥。

## 2. 准备模型和接口类型

1. 在 [API Key 页面](https://alltokenapi.com/keys)创建密钥。
2. 在[模型定价页面](https://alltokenapi.com/pricing)复制准确模型 ID。
3. 确认模型使用的接口：

| 模型接口 | `npm` 适配器 | Base URL |
| --- | --- | --- |
| `/v1/chat/completions` | `@ai-sdk/openai-compatible` | `https://alltokenapi.com/v1` |
| `/v1/responses` | `@ai-sdk/openai` | `https://alltokenapi.com/v1` |

以下主教程按 Chat Completions 编写。Responses 模型必须切换适配器，不能只修改模型 ID。

## 3. 在 opencode.json 中直接保存密钥

在全局配置文件 `~/.config/opencode/opencode.json`（Windows 对应用户配置目录）中合并以下内容。不要把含有明文密钥的项目级 `opencode.json` 提交到 Git。

将以下配置合并到全局配置文件：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "alltokenapi/此处替换为准确的模型 ID",
  "small_model": "alltokenapi/此处替换为准确的模型 ID",
  "provider": {
    "alltokenapi": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "All Token API",
      "options": {
        "baseURL": "https://alltokenapi.com/v1",
        "apiKey": "此处替换为 API Key"
      },
      "models": {
        "此处替换为准确的模型 ID": {
          "name": "此处替换为准确的模型 ID"
        }
      }
    }
  }
}
```

需要替换四处模型 ID。`small_model` 用同一模型可避免标题生成等轻量任务落到其他服务商；有更便宜且兼容的模型时，可以单独替换。

### Responses 模型

如果模型明确使用 `/v1/responses`，只修改这一项：

```json
"npm": "@ai-sdk/openai"
```

其余 Provider ID、Base URL 和模型引用保持不变。

## 4. 选择并验证模型

1. 启动 `opencode`。
2. 在 TUI 中输入 `/models`，选择 `alltokenapi/模型ID`。
3. 发送一个简短提示，或执行一次非交互测试：

```bash
opencode run "只回复 OK"
```

4. 在[使用日志](https://alltokenapi.com/usage-logs)确认请求模型、状态和费用。

## 5. 常见问题

### /models 中没有 alltokenapi

- 检查 `opencode.json` 的生效范围和 JSON/JSONC 语法。
- Provider ID 必须在 `model`、`small_model` 和 `provider` 配置中都写成 `alltokenapi`。
- 模型必须定义在 `provider.alltokenapi.models` 中。

### 认证失败

- 确认 `provider.alltokenapi.options.apiKey` 已替换为有效密钥。
- 如果曾使用 `/connect`，不要同时保留错误的 `auth.json` 凭据；排障时只保留一种来源。

### 404、流式输出或工具调用失败

- Chat Completions 使用 `@ai-sdk/openai-compatible`。
- Responses 使用 `@ai-sdk/openai`。
- Base URL 对这两种 OpenAI 兼容接口都应以 `/v1` 结尾。
- 普通对话成功但工具调用失败，通常表示模型或接口不完整支持 Tool Calling。

### 上下文长度显示不准确

确认准确数值后，可在模型下添加：

```json
"limit": {
  "context": 128000,
  "output": 32000
}
```

不要从模型名称猜测限制；错误值会影响 OpenCode 的上下文压缩判断。

## 6. 官方参考

- [OpenCode 服务商配置](https://opencode.ai/docs/providers/)
- [OpenCode 配置文件](https://opencode.ai/docs/config/)
- [OpenCode 官方仓库](https://github.com/anomalyco/opencode)
