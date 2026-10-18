# 真实模型压测：Claude Sonnet 4.6 / Claude Opus 5 / GPT-6 Sol

本文记录 2026-09-28 对线上网关 `https://alltokenapi.com` 做的三个模型压测：用的脚本、每个模型的测试方法、结果，以及过程中踩到的坑。

> **注意**：这些压测直接打线上网关和真实上游，会产生真实费用（1000 RPM × 2 分钟约 2000 个请求、1000 万以上输入 token）。跑之前确认 key 的额度和上游限流。本地 mock 压测见 [README.md](README.md)。

## 1. 脚本

压测脚本是 [`loadtest.py`](loadtest.py)，它是 k6 脚本的 Python 版本，依赖 `httpx`（`pip install httpx`）。真实模型压测用它的 `capacity` profile 加 `--rpm`：按固定到达速率发请求（open model），不会因为响应变慢而自动降速，所以能测出网关和上游在目标 RPM 下的真实表现。

### 常用参数

| 参数 | 说明 |
|---|---|
| `capacity --rpm N --duration 2m` | 以 N RPM 的固定到达速率持续 2 分钟 |
| `--stream` | 发流式请求，同时统计首字节时间 |
| `--api-format openai\|claude` | `openai` 走 `/v1/chat/completions`；`claude` 走 `/v1/messages`，带 `x-api-key` 和 `anthropic-version`，不发 `stream_options` |
| `--context-tokens N` | 在问题前面拼一段约 N token 的生成文本（一个温和的小镇故事），用来模拟长上下文、触发 prompt caching |
| `--context-cache` / `--no-context-cache` | 默认开启：上下文作为共享前缀、唯一编号放在后面。OpenAI/Responses 格式把上下文放进独立的 system 消息、编号和问题放进 user 消息（部分 Azure 上游按整条消息匹配缓存，拼在同一条消息里会每次整段重写）；Claude 格式给上下文块加 `cache_control: ephemeral`，并设置 `metadata.user_id` 让请求命中渠道亲和。关闭后每个请求前缀都不同，用于测无缓存的情况 |
| `--api-format responses` | 走 `/v1/responses`，输出上限用 `max_output_tokens`，流里的 `response.failed` / `error` 事件计为应用错误 |
| `--prompt-cache-key --sessions N` | OpenAI/Responses 格式带 `prompt_cache_key`，在 N 个会话间轮换，用于让上游按会话固定渠道或部署 |
| `--temperature X` | 默认**不发送** temperature（原因见第 4 节），只有显式传入时才发送 |
| `--max-tokens N` | 输出上限，默认取 `loadtest.config.json` 的 `max_output_tokens`（256） |
| `--max-tokens-field max_completion_tokens` | OpenAI 推理模型要求这个字段时使用；gpt-6-sol 两种都接受 |
| `--api-key` / 环境变量 `LOADTEST_TOKENS` | API key，可重复或逗号分隔，VU 轮流使用 |
| `--summary-out path.json` | 把汇总结果写成 JSON |

### 输出怎么看

- `success rate` / `status ...`：按状态码统计。HTTP 200 但流里带 `error` 的会计入 `application errors`。
- `error sample`：前几个错误的状态码、request id 和响应体。拿 request id 可以去后台用量日志查实际渠道和上游原始错误。
- `duration` / `ttfb` / `first chunk`：总耗时、首字节、首个内容块的 p50/p95/p99。
- `prompt cache`：缓存读写 token 数和命中率（有 `cached_tokens` 或 `cache_read_input_tokens` > 0 的请求占比）。
- `refusals`：Claude 返回 `stop_reason: refusal` 的次数和分类（`stop_details.category`）。HTTP 是 200，但模型没有给出回答。
- `thresholds`：沿用 `loadtest.config.json` 的阈值。其中延迟阈值（p95 < 2s 等）是给 mock 上游定的，对真实大模型必然 FAIL，只看错误率那几项即可。

## 2. 准备

```sh
cd deploy/loadtest
export BASE_URL=https://alltokenapi.com
export LOADTEST_TOKENS=sk-xxxx   # 换成压测专用 key，不要写进仓库
```

先用单个请求确认模型可用，再上压测：

```sh
# Claude 格式
curl -s $BASE_URL/v1/messages -H "x-api-key: $LOADTEST_TOKENS" -H 'anthropic-version: 2023-06-01' \
  -H 'content-type: application/json' \
  -d '{"model":"claude-opus-5","max_tokens":32,"messages":[{"role":"user","content":"hi"}]}'

# OpenAI 格式
curl -s $BASE_URL/v1/chat/completions -H "Authorization: Bearer $LOADTEST_TOKENS" \
  -H 'content-type: application/json' \
  -d '{"model":"gpt-6-sol","max_tokens":32,"messages":[{"role":"user","content":"hi"}]}'
```

建议顺序：先 100 RPM 验证链路，再 1000 RPM。

## 3. 三个模型的测试方法

三个模型都用同样的负载形态：流式、4k 上下文、共享前缀以命中缓存、输出上限 256 token、每轮 2 分钟。

### 3.1 claude-sonnet-4-6

Claude 渠道开了请求体透传，必须用 Claude 原生格式（`--api-format claude`）。用 OpenAI 格式会把 `stream_options` 原样透传给上游，导致 400 `stream_options: Extra inputs are not permitted`。

```sh
python3 loadtest.py capacity --rpm 1000 --duration 2m --stream \
  --api-format claude --context-tokens 4000 \
  --model claude-sonnet-4-6 --summary-out results/rpm1000-sonnet-4-6-ctx4k-cache.json
```

### 3.2 claude-opus-5

方式同 Sonnet 4.6，有两点必须注意：

1. **不要发 temperature/top_p/top_k**。脚本默认已经不发；不要加 `--temperature`。
2. **上下文要用中性文本**。脚本默认的故事文本拒答率约 0；运维报告、日志类文本会触发 Opus 5 输出分类器的 cyber 拒答（见第 4 节）。

```sh
python3 loadtest.py capacity --rpm 100 --duration 2m --stream \
  --api-format claude --context-tokens 4000 \
  --model claude-opus-5 --summary-out results/rpm100-opus-5-notemp.json

python3 loadtest.py capacity --rpm 1000 --duration 2m --stream \
  --api-format claude --context-tokens 4000 \
  --model claude-opus-5 --summary-out results/rpm1000-opus-5-notemp.json
```

### 3.3 gpt-6-sol

走 OpenAI 格式（默认 `--api-format openai`），流式请求会带 `stream_options.include_usage`。OpenAI 的缓存是按前缀自动命中的，不需要 `cache_control`。

```sh
python3 loadtest.py capacity --rpm 100 --duration 2m --stream \
  --api-format openai --context-tokens 4000 \
  --model gpt-6-sol --summary-out results/rpm100-gpt-6-sol.json

python3 loadtest.py capacity --rpm 1000 --duration 2m --stream \
  --api-format openai --context-tokens 4000 \
  --model gpt-6-sol --summary-out results/rpm1000-gpt-6-sol.json
```

## 4. 结果（2026-09-28）

### 汇总

| 模型 | RPM | 成功率 | 失败 | 总耗时 p50 / p95 / p99 | 首字节 p50 / p95 | 缓存命中 | 输出 tok/s |
|---|---|---|---|---|---|---|---|
| claude-sonnet-4-6 | 1000 | 99.95% | 1 × 超时 | 6.4s / 9.2s / 14.7s | 1.7s / 3.5s | 84.6% | 3057 |
| claude-opus-5 | 100 | 100% | 0 | 8.1s / 11.9s / 15.6s | 5.3s / 8.6s | 43.2% | 397 |
| claude-opus-5 | 1000 | **100%** | 0 | 7.4s / 10.6s / 13.0s | 4.1s / 7.1s | 80.5% | 4010 |
| gpt-6-sol | 100 | 100% | 0 | 5.0s / 15.2s / 19.9s | 3.6s / 12.9s | 100% | 112 |
| gpt-6-sol | 1000 | 99.10% | 17 × 503，1 × 流中断 | 6.2s / 14.9s / 23.2s | 4.4s / 12.1s | 99.95% | 957 |

结果文件都在 `results/` 下，文件名与第 3 节命令中的 `--summary-out` 一致。

### 各模型结论

- **claude-sonnet-4-6**：1000 RPM 下稳定，首字节最快。加上 `metadata.user_id` 渠道亲和后，缓存命中率从 65.5% 提升到 84.6%。
- **claude-opus-5**：去掉 temperature 后，1000 RPM 全部成功，没有 503、429，也没有拒答。之前的 503 全部是 temperature 引起的，不是上游限流。
- **gpt-6-sol**：中位数延迟和 Opus 5 相当，但长尾更长（首字节 p95 约 12s）。1000 RPM 的 17 个 503 集中在第 109–120 秒，前 100 秒没有错误，更像上游短时波动，不是持续的容量上限。它平均每个请求只输出约 70 token（Opus 5 约 255），所以输出 tok/s 偏低，不代表吞吐差。

## 5. 踩过的坑

| 现象 | 原因 | 处理 |
|---|---|---|
| 400 `stream_options: Extra inputs are not permitted` | 用 OpenAI 格式打开了透传的 Claude 渠道，`stream_options` 被原样透传 | Claude 模型用 `--api-format claude` 走 `/v1/messages` |
| 有长上下文，但缓存 read/write 一直是 0 | Claude 需要显式 `cache_control`，且可缓存前缀必须达到最小长度（Sonnet 4.6 为 1024 token，Opus 5 为 512 token）；唯一编号放在前面也会破坏前缀 | 脚本把上下文放在前面并加 `cache_control`，唯一编号放在后面 |
| 有缓存写入，但命中率低 | 缓存按上游账号隔离，请求被分到不同渠道或账号就命中不了 | 设置 `metadata.user_id`，命中网关内置的 "claude cli trace" 渠道亲和规则 |
| Opus 5 返回 200，但 `stop_reason: refusal`，分类为 cyber | 输出分类器对运维报告、日志类上下文敏感（运维报告约 100%，书店日志约 9–19%） | 默认上下文改成中性的小镇故事，拒答降到约 0 |
| Opus 5 大量快速 503，Sonnet 4.6 正常 | Opus 4.7+/Opus 5/Sonnet 5/Fable 5 拒绝非默认的 temperature/top_p/top_k，上游返回 400，网关对外统一显示成 503 并换渠道重试（p95 被拉到约 55s） | 脚本默认不发 temperature。网关已在 `relay/claude_handler.go` 和两个 OpenAI→Claude 转换器中自动去掉这三个参数（透传模式也会处理），需部署后对真实用户生效 |
| 429 `您已达到请求数限制：1分钟内最多请求1000次` | new-api 内置限流器的文案；出现在某个上游中转渠道上，是该上游（同样基于 new-api）的限制，不是我们的 | 调整该渠道的权重或联系上游提额 |
| gpt-6-sol 发一句 "hi"，`prompt_tokens` 约 4.4k | 渠道或上游在每个请求前注入了约 4.4k token 的系统提示（几乎全部命中缓存） | 压测结果里的输入 token 要扣掉这部分；同时确认这部分是否计入用户账单 |

### 哪些模型拒绝采样参数

2026-09-28 在线上逐个测试（带 `temperature=0.5`、`top_p=0.9`、`top_k=5`）的结果：

| 拒绝（返回 503，即上游 400） | 接受 |
|---|---|
| claude-opus-4-7、claude-opus-4-8、claude-opus-5、claude-opus-5-5、claude-sonnet-5、claude-fable-5、claude-fable-5-1 | claude-opus-4-5、claude-opus-4-6、claude-sonnet-4-5、claude-sonnet-4-6、claude-haiku-4-5 |

网关里对应的模型列表在 `setting/model_setting/claude.go` 的 `claudeFixedSamplingModels`，新模型上线时需要同步更新。

## 6. 排查失败请求

1. 从 `error sample` 或 `refusal_samples`（JSON 里）拿到 request id，格式如 `202609280658128024129108268d9d6XqC1oEq6`。
2. 在后台用量日志中按 request id 查询，看实际走的渠道、重试次数和上游原始错误。
3. 网关会把上游错误统一显示成 503 "Service temporarily unavailable"，只有 429 会原样保留。所以客户端看到 503 时，一定要去日志看真实原因。
