# alltoken 错误码、错误记录与多 Cluster 切流规范

> 文档状态：本地实现稿
> 设计版本：v2
> 实施状态：代码、后台配置、监控和本地确定性压测已实现；尚未推送或发布

## 1. 设计目标

本规范解决四个问题：

1. 保留 OpenAI、cluster 的原始错误，兼容现有客户端。
2. alltoken 提供一套稳定、简短、可扩展的数字错误码。
3. 运维看到错误时能快速定位错误分类和失败 cluster。
4. 一个 cluster 失败后，alltoken 能自动切换到其他 cluster，并留下完整记录。

核心原则：

- HTTP 状态、上游原始 `code`、alltoken 数字码分别保存，不互相替代。
- 错误码描述“发生了什么”，cluster 编号描述“发生在哪里”。
- 尝试序号、错误次数、请求 ID 属于错误记录，不写进错误码。
- 程序判断数字码，人工排障使用简短的 `error_ref`。

## 2. 统一错误响应

```json
{
  "error": {
    "message": "Rate limit reached",
    "type": "rate_limit_error",
    "code": "rate_limit_exceeded",
    "source": "openai",
    "alltoken_code": 104001,
    "error_ref": "104001-C17-P1",
    "category": "rate_limit",
    "cluster_code": 17,
    "pool_tier": 1,
    "failure_scope": "cluster",
    "retryable": true,
    "action": "failover",
    "request_id": "req_01J..."
  }
}
```

| 字段 | 含义 | 程序是否应依赖 |
| --- | --- | --- |
| `message` | 脱敏后的错误说明 | 否 |
| `type` | OpenAI 兼容错误类型 | 兼容使用 |
| `code` | OpenAI/cluster 原始错误码 | 仅用于兼容和追查上游 |
| `source` | 错误所属层：`openai`、`cluster`、`alltoken` | 是 |
| `alltoken_code` | alltoken 稳定数字错误码 | 是，主要判断字段 |
| `error_ref` | 数字错误码、cluster 编号和号池层级的简短组合 | 仅用于检索和人工排障 |
| `category` | 稳定错误分类 | 是，用于统计和策略 |
| `cluster_code` | 失败 cluster 的短数字编号 | 是，用于定位和聚合 |
| `pool_tier` | 失败号池：`1` 免费、`2` Pro/Plus、`3` 兜底 | 是，用于定位和聚合 |
| `failure_scope` | 本次故障影响范围 | 是，用于决定排除范围 |
| `retryable` | 当前错误是否具备重试条件 | 是 |
| `action` | alltoken 对本次错误的实际处理动作 | 是 |
| `request_id` | 一次完整请求的唯一 ID | 用于查链路 |

建议响应头：

```text
X-Alltoken-Code: 104001
X-Alltoken-Error-Ref: 104001-C17-P1
X-Alltoken-Error-Source: openai
X-Alltoken-Retryable: true
X-Request-ID: req_01J...
```

`action` 只使用以下值：

| 值 | 含义 |
| --- | --- |
| `none` | 不自动切流或重试 |
| `failover` | 切换到其他渠道或 cluster |
| `retry_later` | 当前请求终止，客户端退避后可重试 |
| `abort` | 客户端取消或流已提交，禁止重放 |
| `manual` | 由策略配置或人工决定 |

## 3. 数字错误码格式

数字码固定为 6 位：

```text
LCCNNN
```

| 部分 | 位数 | 含义 |
| --- | ---: | --- |
| `L` | 1 | 错误来源层 |
| `CC` | 2 | 错误分类 |
| `NNN` | 3 | 分类内序号 |

计算规则：

```text
alltoken_code = L * 100000 + CC * 1000 + NNN
```

### 3.1 来源层

| `L` | `source` | 含义 |
| ---: | --- | --- |
| `1` | `openai` | OpenAI 官方产生的错误 |
| `2` | `cluster` | IKUN、sub2api 等中间 cluster 产生的错误 |
| `3` | `alltoken` | alltoken 本地产生的错误 |
| `9` | `unknown` | 暂时无法归属，保留使用 |

`source=cluster` 是逻辑层。具体使用 IKUN 还是 sub2api，通过 `cluster_type` 记录，不再把 `ikun` 作为错误来源枚举。

### 3.2 错误分类

| `CC` | `category` | 含义 |
| ---: | --- | --- |
| `00` | `unknown` | 未知错误 |
| `01` | `request` | 请求格式、参数、API 类型 |
| `02` | `auth` | API Key、权限、地区限制 |
| `03` | `quota` | 余额、额度、计费、预扣 |
| `04` | `rate_limit` | 请求、Token、并发限流 |
| `05` | `upstream` | 上游不可用、过载、响应异常 |
| `06` | `channel` | 渠道 Key、映射、渠道配置 |
| `07` | `policy` | 内容安全、敏感词、策略拦截 |
| `08` | `protocol` | 序列化、协议转换、响应格式 |
| `09` | `internal` | 数据库、内部服务、业务处理 |
| `10` | `network` | DNS、连接、TLS、超时、断流 |
| `11` | `model` | 模型不存在、模型配置错误 |
| `99` | `reserved` | 保留，不直接使用 |

### 3.3 示例

| 数字码 | 拆解 | 含义 |
| ---: | --- | --- |
| `104001` | `1-04-001` | OpenAI 限流 |
| `204001` | `2-04-001` | cluster 内当前渠道/号池限流 |
| `301001` | `3-01-001` | alltoken 参数错误 |
| `305001` | `3-05-001` | 所有上游耗尽 |
| `306001` | `3-06-001` | 渠道没有可用 Key |

数字码不是 HTTP 状态码。`429`、`500`、`503` 继续作为独立的 HTTP 状态保存。

## 4. 快速定位失败 Cluster

每个物理 cluster 在 alltoken 注册时获得一个永久短编号：

```text
cluster_code = 17
```

人工排障使用：

```text
error_ref = <alltoken_code>-C<cluster_code>-P<pool_tier>
```

示例：

| `error_ref` | 含义 |
| --- | --- |
| `301001` | alltoken 参数错误，没有经过 cluster |
| `205004-C17-P1` | 第 17 号 cluster 的免费号池耗尽 |
| `104001-C23-P2` | 请求经过第 23 号 cluster 的 Pro/Plus 号池，收到 OpenAI 限流 |
| `305001-C23-P3` | 所有上游耗尽，最后失败的是第 23 号 cluster 的兜底链路 |

设计规则：

- `cluster_code` 范围为 `1-999999`，单调分配，删除后不得复用。
- 同一个物理 cluster 的多个渠道配置必须共享同一个 `cluster_code`。
- `cluster_code` 不从 IP、域名或渠道名称生成。
- 普通客户端可看到 `cluster_code`，但看不到真实 IP、域名、Key 和内部 `cluster_id`。
- 程序不要解析 `error_ref`；程序直接读取 `alltoken_code` 和 `cluster_code`。
- 没有关联号池时省略 `-P<n>`，例如 alltoken 请求参数错误只返回 `301001`。

不再把尝试序号放进 `error_ref`。第几次尝试属于错误记录，不属于错误码。

## 5. 错误码与错误记录

### 5.1 错误码

错误码是一条稳定定义：

```text
104001 = OpenAI rate limit
```

它可以在不同时间、不同请求、不同 cluster 上重复出现。

### 5.2 错误记录

每次实际发生错误，都创建一条唯一记录：

```json
{
  "error_event_id": "err_01J...",
  "event_kind": "upstream_attempt",
  "request_id": "req_01J...",
  "alltoken_code": 104001,
  "error_ref": "104001-C23",
  "source": "openai",
  "category": "rate_limit",
  "cluster_code": 23,
  "cluster_id": "clu_01J...",
  "cluster_type": "ikun",
  "channel_id": 128,
  "attempt_no": 2,
  "status_code": 429,
  "failure_scope": "cluster",
  "action": "failover",
  "occurred_at": "2026-08-15T12:30:01.123Z"
}
```

字段含义：

| 字段 | 含义 |
| --- | --- |
| `error_event_id` | 一条错误记录的唯一 ID |
| `event_kind` | `upstream_attempt` 或 `final_response` |
| `attempt_no` | 本次请求第几次上游尝试，例如 `2` |
| `request_id` | 关联同一次请求的所有记录 |
| `alltoken_code` | 错误分类定义 |
| `cluster_code` | 哪个 cluster 失败 |

`attempt_no=2` 不是第二条历史错误，也不是错误总量，只表示当前请求的第二次上游尝试。

一个请求尝试 3 个 cluster 且全部失败时：

```text
3 条 event_kind=upstream_attempt
1 条 event_kind=final_response
```

## 6. 初始错误码注册表

### 6.1 OpenAI 常见错误

| 数字码 | 名称 | 常见 HTTP | 原始 `code` 示例 |
| ---: | --- | ---: | --- |
| `102001` | `INVALID_API_KEY` | 401 | `invalid_api_key` |
| `102002` | `UNSUPPORTED_REGION` | 403 | `unsupported_country_region_territory` |
| `103001` | `CREDIT_EXHAUSTED` | 429 | `credit_balance_exhausted`、`insufficient_quota` |
| `103002` | `ORG_SPEND_LIMIT` | 429 | `organization_spend_limit_exceeded` |
| `103003` | `PROJECT_SPEND_LIMIT` | 429 | `project_spend_limit_exceeded` |
| `103004` | `ORG_USAGE_LIMIT` | 429 | `organization_usage_limit_exceeded` |
| `104001` | `RATE_LIMIT_EXCEEDED` | 429 | `rate_limit_exceeded` |
| `105001` | `SERVER_ERROR` | 500 | `server_error` |
| `105002` | `OVERLOADED` | 503 | `overloaded`、`service_unavailable` |

OpenAI 官方参考：<https://developers.openai.com/api/docs/guides/error-codes>

### 6.2 Cluster 常见错误

| 数字码 | 名称 | 原始 `code` 示例 | 默认动作 |
| ---: | --- | --- | --- |
| `200001` | `CLUSTER_UNKNOWN` | 未识别错误 | `failover` |
| `202001` | `CLUSTER_INVALID_CREDENTIAL` | `invalid_api_key` | `failover` |
| `204001` | `CLUSTER_POOL_RATE_LIMIT` | `rate_limit_exceeded` | `failover` |
| `205001` | `NO_HEALTHY_ACCOUNT` | `no_healthy_account` | `failover` |
| `205002` | `CLUSTER_UNAVAILABLE` | `service_unavailable` | `failover` |
| `205003` | `ALL_POOLS_EXHAUSTED` | `all_pools_exhausted` | `failover` |
| `205004` | `POOL_EXHAUSTED` | `pool_exhausted` | `failover` |
| `210001` | `CLUSTER_TIMEOUT` | `timeout` | `failover` |

IKUN、sub2api 或其他实现可以返回不同原始字符串，alltoken 负责映射到稳定数字码。无法识别时使用 `200001`，同时保留原始 `code`。

### 6.3 alltoken 本地错误

| 数字码 | 名称 | 原始 `code` | 默认动作 |
| ---: | --- | --- | --- |
| `301001` | `INVALID_REQUEST` | `invalid_request` | `none` |
| `301002` | `BAD_REQUEST_BODY` | `bad_request_body` | `none` |
| `301003` | `INVALID_API_TYPE` | `invalid_api_type` | `none` |
| `301004` | `READ_REQUEST_BODY_FAILED` | `read_request_body_failed` | `none` |
| `302001` | `ACCESS_DENIED` | `access_denied` | `none` |
| `303001` | `INSUFFICIENT_USER_QUOTA` | `insufficient_user_quota` | `none` |
| `303002` | `PRE_CONSUME_FAILED` | `pre_consume_token_quota_failed` | `none` |
| `304001` | `ALLTOKEN_RATE_LIMIT` | `rate_limit_exceeded` | `none` |
| `305001` | `UPSTREAM_EXHAUSTED` | `upstream_exhausted` | `retry_later` |
| `305002` | `BAD_RESPONSE_STATUS` | `bad_response_status_code` | `failover` |
| `305003` | `BAD_RESPONSE` | `bad_response` | `failover` |
| `305004` | `EMPTY_RESPONSE` | `empty_response` | `failover` |
| `305005` | `AWS_INVOKE_ERROR` | `aws_invoke_error` | `failover` |
| `306001` | `CHANNEL_NO_KEY` | `channel:no_available_key` | `failover` |
| `306002` | `CHANNEL_PARAM_INVALID` | `channel:param_override_invalid` | `failover` |
| `306003` | `CHANNEL_HEADER_INVALID` | `channel:header_override_invalid` | `failover` |
| `306004` | `CHANNEL_MODEL_MAP_ERROR` | `channel:model_mapped_error` | `failover` |
| `306005` | `CHANNEL_AWS_CLIENT_ERROR` | `channel:aws_client_error` | `failover` |
| `306006` | `CHANNEL_INVALID_KEY` | `channel:invalid_key` | `failover` |
| `306007` | `CHANNEL_TIMEOUT` | `channel:response_time_exceeded` | `failover` |
| `307001` | `SENSITIVE_WORDS` | `sensitive_words_detected` | `none` |
| `307002` | `CONTENT_VIOLATION` | `violation_fee.grok.csam` | `none` |
| `307003` | `PROMPT_BLOCKED` | `prompt_blocked` | `manual` |
| `308001` | `CONVERT_REQUEST_FAILED` | `convert_request_failed` | `none` |
| `308002` | `JSON_MARSHAL_FAILED` | `json_marshal_failed` | `none` |
| `308003` | `BAD_RESPONSE_BODY` | `bad_response_body` | `failover` |
| `309001` | `QUERY_DATA_ERROR` | `query_data_error` | `none` |
| `309002` | `UPDATE_DATA_ERROR` | `update_data_error` | `none` |
| `309003` | `COUNT_TOKEN_FAILED` | `count_token_failed` | `none` |
| `309004` | `MODEL_PRICE_ERROR` | `model_price_error` | `none` |
| `309005` | `GET_CHANNEL_FAILED` | `get_channel_failed` | `retry_later` |
| `309006` | `GEN_RELAY_INFO_FAILED` | `gen_relay_info_failed` | `none` |
| `310001` | `DO_REQUEST_FAILED` | `do_request_failed` | `failover` |
| `310002` | `READ_RESPONSE_FAILED` | `read_response_body_failed` | `failover` |
| `311001` | `MODEL_NOT_FOUND` | `model_not_found` | `failover` |

## 7. 多 Cluster 标识

alltoken 建立独立的 cluster 注册表：

| 字段 | 作用 |
| --- | --- |
| `cluster_id` | 内部永久 ID，不对普通客户端公开 |
| `cluster_code` | 简短数字编号，用于错误响应、Grafana 和告警 |
| `cluster_type` | `ikun`、`sub2api` 等实现类型 |
| `name` | 管理后台显示名称 |
| `failover_group` | 允许互相切流的 cluster 集合 |

渠道配置只引用 `cluster_id`。`cluster_code` 由注册表分配，不能由渠道自行填写。

同一个物理 cluster 即使配置多个模型、Key 或渠道记录，也必须引用相同 `cluster_id` 和 `cluster_code`。

## 8. 故障范围与切流

### 8.1 故障范围

| `failure_scope` | 典型场景 | 排除对象 |
| --- | --- | --- |
| `request` | 参数错误、用户权限、内容策略 | 不切流，终止请求 |
| `credential` | alltoken 直接管理的单个 Key 失效 | 当前凭证或渠道 |
| `channel` | 渠道映射、Header 配置错误 | 当前 `channel_id` |
| `cluster` | cluster 5xx、超时、账号池耗尽 | 当前 `cluster_id` 下所有渠道 |
| `provider` | 官方供应商区域性或全局故障 | 该 provider 的候选集合 |

cluster 内部凭证由 cluster 自己重试。如果 cluster 的内部账号已经全部失败，返回给 alltoken 时必须使用 `failure_scope=cluster`。

### 8.2 请求内切流

每次请求维护：

```text
attempted_channel_ids
attempted_cluster_ids
```

流程：

1. 按模型、用户组和 `failover_group` 生成候选 cluster。
2. 排除请求内已经失败的渠道、cluster 和全局熔断的 cluster。
3. 按优先级选择 cluster，再在 cluster 内按权重选择渠道。
4. 上游失败后写入一条 `upstream_attempt` 错误记录。
5. `channel/credential` 只排除当前渠道，允许同 Cluster 从 P1 切到 P2/P3；`cluster/provider` 排除整个 Cluster。
6. 响应尚未提交且仍有候选时，切到下一个 cluster。
7. 没有候选或达到预算时返回 `305001 / UPSTREAM_EXHAUSTED`。

三套默认模式：

| 参数 | 保守 | 均衡 | 激进 | 含义 |
| --- | ---: | ---: | ---: | --- |
| `max_cluster_attempts` | 2 | 3 | 4 | 一次请求最多访问多少个 Cluster |
| `max_total_attempts` | 4 | 6 | 8 | 一次请求的总上游尝试上限 |
| `total_failover_budget_ms` | 12000 | 10000 | 6000 | 整条切流链路的时间预算 |
| `circuit_failure_threshold` | 8 | 5 | 3 | 窗口内多少次 Cluster 故障后熔断 |
| `circuit_window_seconds` | 60 | 60 | 30 | 熔断统计窗口 |
| `circuit_cooldown_seconds` | 60 | 60 | 90 | 熔断后的冷却时间 |

成本与号池参数：

- `allow_paid_escalation`：是否允许从 P1 升级到 P2。
- `allow_fallback`：是否允许进入 P3 兜底链路。
- `max_cost_multiplier`：允许选择的最大号池成本倍率。
- `same_pool_retries`、`max_pool_attempts`：为 Cluster 内部实现预留的重试预算。
- `X-Alltoken-Failover-Mode`：请求级选择 `conservative`、`balanced` 或 `aggressive`。

流式响应一旦向客户端输出内容，不再自动重放。

### 8.3 Cluster 熔断

```text
closed -> open -> half_open -> closed
```

熔断按 `cluster_id + route` 统计。连接失败、超时、cluster 5xx 和全部账号池耗尽计入熔断；参数错误、内容策略和客户端取消不计入。启用 Redis 时熔断状态由 Redis Lua 脚本原子维护，多个 alltoken 实例共享；Redis 不可用时自动回退到进程内熔断。

## 9. 数据库设计与后台配置

| 表 | 用途 |
| --- | --- |
| `clusters` | Cluster 永久编号、类型、启停和归档状态 |
| `cluster_pools` | 每个 Cluster 的 P1/P2/P3、成本倍率和启停状态 |
| `failover_policies` | 三种模式的尝试预算、时间预算、成本和熔断参数 |
| `failover_policy_steps` | 可扩展的号池执行顺序和单步尝试上限 |
| `failover_groups` | 可互相切流的 Cluster 集合 |
| `failover_group_members` | Cluster 在切流组内的优先级和权重 |
| `failover_rules` | 按模型、路由、用户组选择切流组和策略 |
| `upstream_error_mappings` | 将不同 Cluster 实现的原始错误映射为稳定错误码和动作 |

管理员入口：`/failover`。接口：

```text
GET /api/channel/failover/config
PUT /api/channel/failover/config
```

删除 Cluster 时执行软归档，删除号池、策略和错误映射时执行禁用，避免历史错误记录失去解释依据。错误映射按“Cluster 类型、原始 code、HTTP 状态”匹配，精确规则优先于 `*` 通配规则。

## 10. 所有上游耗尽

```json
{
  "error": {
    "code": "upstream_exhausted",
    "source": "alltoken",
    "alltoken_code": 305001,
    "error_ref": "305001-C23",
    "category": "upstream",
    "cluster_code": 23,
    "retryable": true,
    "action": "retry_later",
    "request_id": "req_01J...",
    "attempt_count": 3,
    "cluster_attempt_count": 3,
    "cause": {
      "source": "openai",
      "code": "rate_limit_exceeded",
      "alltoken_code": 104001,
      "error_ref": "104001-C23",
      "status_code": 429
    }
  }
}
```

最外层 `305001-C23` 表示所有候选耗尽，最后失败的是第 23 号 cluster。完整尝试顺序通过 `request_id` 查询错误记录。

## 11. 错误量与监控

错误量通过错误记录统计，不写进错误码。

推荐指标：

```text
alltoken_error_events_total{event_kind, alltoken_code, category, cluster_code}
alltoken_final_errors_total{alltoken_code, category}
alltoken_pool_requests_total{cluster_code, pool_tier, result}
alltoken_pool_failover_total{cluster_code, from_pool, to_pool, mode}
alltoken_cluster_failover_total{from_cluster, to_cluster, mode}
alltoken_cluster_circuit_state{cluster_code, state}
alltoken_failover_duration_seconds{result, mode}
```

统计口径：

- `event_kind=upstream_attempt`：cluster 实际失败量。
- `event_kind=final_response`：用户最终收到的错误量。
- 按 `alltoken_code` 统计具体错误。
- 按 `category` 统计错误分类。
- 按 `cluster_code` 统计各 cluster 稳定性。

禁止把 `request_id`、`error_event_id` 或 `error_ref` 作为 Prometheus 标签，避免高基数。

建议告警：

- `305001` 出现或持续增长。
- 单个 `cluster_code` 的 5xx、超时或熔断率超过阈值。
- `104001`、`204001` 限流持续增长。
- 平均 `cluster_attempt_count` 持续升高。
- `309001`、`309002` 数据库错误出现。
- P1 剩余 Token 低于阈值、P2/P3 消耗异常增长。
- Cluster 熔断、跨 Cluster 切流频率过高。

Grafana 的 `new-api-loadtest` 看板已包含号池余量、号池消耗、池间切流、Cluster 间切流、熔断状态、稳定错误码和告警面板。Alertmanager 对 Cluster 熔断告警抑制同 Cluster 下的低级号池告警，减少告警风暴。

## 12. 确定性切流压测

本地压测栈包含两个 Mock Cluster。每次成功固定消费 30 Token，支持重置、耗尽和禁用：

```sh
curl -X POST 'http://localhost:18080/control/reset?free=300&premium=600&fallback=900'
curl -X POST 'http://localhost:18080/control/exhaust?pool=free'
curl -X POST 'http://localhost:18080/control/exhaust?pool=all'
curl -X POST 'http://localhost:18080/control/disable'
curl -X POST 'http://localhost:18080/control/enable'
curl 'http://localhost:18080/control/state'
```

执行：

```sh
cd deploy/loadtest
./run.sh pool-failover
```

验收顺序：P1 余额下降 -> P1 耗尽后使用 P2 -> P2 耗尽后使用 P3 -> Cluster A 全部耗尽后切换 Cluster B -> Cluster A 熔断告警触发 -> 冷却后半开探测 -> 成功后恢复关闭。

## 13. 扩展与注册表管理

错误码使用机器可读注册表作为唯一数据源：

```yaml
version: 1
errors:
  - code: 104001
    name: RATE_LIMIT_EXCEEDED
    source: openai
    category: rate_limit
    raw_codes:
      - rate_limit_exceeded
    default_action: failover
    default_retryable: true
```

规则：

- 数字码发布后不得改义、删除或复用。
- 新错误只申请当前分类中的新 `NNN`。
- 未知 OpenAI 错误使用 `100001`，未知 cluster 错误使用 `200001`，未知 alltoken 错误使用 `300001`。
- 原始 `code` 可以新增映射，但不能改变已发布数字码语义。
- Go、TypeScript、文档和 Grafana 配置从同一注册表生成。
- CI 校验数字码唯一、分类正确、历史码未被修改。

## 14. 兼容与上线

- 保留原始 `error.code`，旧客户端不受影响。
- 新客户端优先使用 `alltoken_code`，不要匹配 `message`。
- 旧的 `source_code` 可以保留一个兼容周期，但不再作为主判断字段。
- `source=ikun` 作为输入兼容别名，规范化为 `source=cluster`。
- 上线顺序：注册表与常量 -> 错误响应 -> cluster 注册表 -> 跨 cluster 切流 -> 日志指标 -> Grafana 与告警 -> 灰度发布。

## 15. 验收清单

- [ ] 原始 `code` 和 alltoken 数字码同时返回。
- [ ] 数字码符合 `LCCNNN` 格式。
- [ ] `error_ref` 能直接定位错误和失败 cluster。
- [ ] 同一物理 cluster 的多个渠道共享 `cluster_code`。
- [ ] `attempt_no` 只存在于错误记录，不进入错误码。
- [ ] cluster 级错误会排除同一 cluster 下所有渠道。
- [ ] channel 级错误只排除当前渠道。
- [ ] 达到尝试次数或时间预算后返回 `305001`。
- [ ] `allow_paid_escalation=false` 时不会进入 P2。
- [ ] `allow_fallback=false` 时不会进入 P3。
- [ ] 自定义错误映射会覆盖内置分类并影响排除范围。
- [ ] 多实例共享 Redis 熔断状态，Redis 故障时仍可本机降级。
- [ ] 已开始输出的流式请求不会自动重放。
- [ ] 错误量可以按错误码、分类、cluster 和最终响应分别统计。
- [ ] Grafana 和告警展示 `error_ref`、`request_id` 和 cluster 名称。
