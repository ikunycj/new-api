# AllToken 渠道路由、错误码与监控规范

## 1. 目标架构

AllToken 保留商业权益和请求计费两类独立维度：

```text
用户 -> 用户组（users.group）-> 可用计费组 -> Token 计费组（tokens.group）-> 渠道 -> 上游服务
```

- **用户**：登录账号与 API Key 的所有者。
- **用户组**：复用 `users.group`，表示账号的商业权益层级，并决定该用户可选择哪些计费组。
- **计费分组**：API Key 授权的计费与模型能力边界，复用 Token、Ability 和 Channel 已有的 `group`。
- **渠道**：一个可独立请求、计费、重试、熔断和监控的上游入口。

AllToken 不再维护 Cluster 和号池。上游系统内部是否存在账号池属于上游实现细节；AllToken 收到的上游错误统一归因到实际请求渠道，并在计费分组内直接切换渠道。

## 2. 路由配置

每个计费分组最多配置一条启用的路由。路由由 `billing_group_routes` 和 `billing_group_channels` 两张表组成。

### 2.1 路由策略

| 策略 | 用途 | 默认总尝试次数 | 默认总超时 | 默认熔断阈值 |
| --- | --- | ---: | ---: | ---: |
| `cost_first` | 允许在低成本渠道上多尝试后再切换 | 6 | 45 秒 | 8 |
| `balanced` | 在成本与稳定性之间折中 | 4 | 30 秒 | 5 |
| `stability_first` | 更快离开异常渠道 | 3 | 20 秒 | 3 |

策略默认值可以在管理后台逐项覆盖：

- `max_total_attempts`：一次客户端请求允许的全部上游尝试数。
- `total_timeout_ms`：整个渠道切换过程的时间预算，不是单个模型响应超时。
- `circuit_failure_threshold`：窗口内触发渠道熔断的失败数。
- `circuit_window_seconds`：熔断统计窗口。
- `circuit_cooldown_seconds`：渠道熔断后的冷却时间。
- `circuit_half_open_requests`：半开状态允许的探测请求数。

### 2.2 渠道顺序

每条渠道配置包含：

| 字段 | 含义 |
| --- | --- |
| `channel_id` | 渠道永久 ID，也是错误与监控定位键 |
| `priority` | 仅用于定价分组显式选择“优先级策略”时的兼容顺序；普通渠道调度不读取该字段 |
| `weight` | 渠道长期流量权重；0 按默认权重 1 处理 |
| `max_attempts` | 切换前在该渠道最多尝试多少次 |
| `cost_factor` | 压测成本估算使用的渠道成本系数 |
| `enabled` | 是否参加路由 |

渠道必须属于同一个计费分组并支持请求模型。普通渠道统一按 Weight 分流；只有定价分组明确选择兼容的“优先级策略”时，才使用该路由内部顺序。

未配置或停用计费分组路由时，系统使用 Channel/Ability 的健康状态与 Weight 逻辑；遗留 `priority` 仅作为兼容数据保留。跨计费分组重试仅在 API Key 使用 `auto` 分组并明确开启 `cross_group_retry` 时发生。

## 3. 切流过程

一次请求的处理顺序如下：

1. 根据用户和 API Key 确定允许使用的计费分组。
2. 根据计费分组、模型和请求路径过滤可用渠道。
3. 按渠道健康状态、并发和 Weight 选择渠道；兼容优先级策略仅在定价分组显式启用时生效。
4. 检查该渠道的熔断状态。
5. 发起请求并记录渠道尝试结果。
6. 对可重试错误，在 `max_attempts` 和全局预算内重试当前渠道。
7. 当前渠道达到尝试上限、被错误规则要求切换或被熔断时，选择下一渠道。
8. 所有候选渠道或预算耗尽后，对客户端返回最终错误。

渠道熔断按 `channel_id + route` 统计。Redis 可用时多个 AllToken 实例共享状态；Redis 不可用时退化为进程内状态。请求参数错误、内容策略错误和客户端主动取消不应计入渠道熔断。

## 4. 错误分层

`source` 表示错误最初来自哪一层：

| `source` | 含义 | 数字段 |
| --- | --- | --- |
| `openai` | 官方 OpenAI 或兼容官方协议的明确原始错误 | `1xxxxx` |
| `channel` | 当前渠道或其上游返回的错误 | `2xxxxx` |
| `alltoken` | AllToken 网关自身生成的错误 | `3xxxxx` |

历史输入中的 `cluster` 和 `ikun` 只作为兼容别名读取，输出一律规范化为 `channel`。不要使用字符串包含匹配来决定切流，应优先使用六位 `alltoken_code`、`failure_scope`、`action` 和 `retryable`。

### 4.1 六位数字格式

```text
SCCDDD
```

- `S`：来源层，1=OpenAI，2=渠道，3=AllToken。
- `CC`：错误类别。
- `DDD`：该类别下可扩展的具体编号。

`alltoken_code` 是稳定分类，不编码某个具体渠道。具体失败渠道通过独立的 `channel_id` 和 `channel_name` 表达。

`error_ref` 是一条可检索的错误记录引用：

```text
204001-CH38
```

其中 `204001` 是错误格式中的稳定分类，`CH38` 表示这次错误实际发生在渠道 38。它不是“第 38 条错误”。

### 4.2 主要错误码

| 范围/错误码 | 含义 | 默认动作 |
| --- | --- | --- |
| `102xxx` | 官方认证或地区限制 | `switch_channel` |
| `103xxx` | 官方额度不足 | `switch_channel` |
| `104001` | 官方 429 | `switch_channel` |
| `105xxx` | 官方 5xx/不可用 | `switch_channel` |
| `202xxx` | 渠道凭证错误 | `switch_channel` |
| `204001` | 渠道 429 | `switch_channel` |
| `205xxx` | 渠道上游不可用或无健康账号 | `switch_channel` |
| `210001` | 渠道超时 | `switch_channel` |
| `301xxx` | 客户端请求格式错误 | `none` |
| `302xxx` | AllToken 鉴权错误 | `none` |
| `303xxx` | 用户额度或预扣费失败 | `none` |
| `305001` | 所有候选渠道已耗尽 | `retry_later` |
| `306xxx` | AllToken 检测到的渠道配置/Key 错误 | `switch_channel` |
| `307xxx` | 内容策略错误 | `none` 或 `manual` |
| `308xxx` | 协议转换或响应解析错误 | 按作用域处理 |
| `309xxx` | AllToken 内部错误 | 通常不切流 |
| `310xxx` | AllToken 到渠道的网络错误 | `switch_channel` |
| `311001` | 渠道不支持模型 | `switch_channel` |

## 5. 错误记录

客户端最终错误和内部上游尝试使用同一结构。示例：

```json
{
  "source": "channel",
  "source_code": "channel.rate_limit_error",
  "alltoken_code": 204001,
  "error_ref": "204001-CH38",
  "category": "rate_limit",
  "channel_id": 38,
  "channel_name": "Claude Pro",
  "failure_scope": "channel",
  "action": "switch_channel",
  "retryable": true,
  "request_id": "req_01J...",
  "attempt_count": 2
}
```

响应头同步提供：

- `X-Alltoken-Error-Source`
- `X-Alltoken-Error-Code`
- `X-Alltoken-Code`
- `X-Alltoken-Error-Ref`
- `X-Alltoken-Retryable`

### 5.1 作用域

| `failure_scope` | 含义 |
| --- | --- |
| `request` | 当前请求本身有问题，不应切流 |
| `credential` | 当前渠道凭证失败，可按配置重试/切换 |
| `channel` | 当前渠道失败，切换到下一渠道 |
| `provider` | 上游服务级故障，当前渠道记失败并切换 |

### 5.2 动作

| `action` | 含义 |
| --- | --- |
| `none` | 不执行额外动作 |
| `retry_channel` | 在渠道尝试预算内重试当前渠道 |
| `switch_channel` | 切换到下一候选渠道 |
| `retry_later` | 已耗尽当前可用路径，建议客户端稍后重试 |
| `abort` | 立即终止 |
| `manual` | 需要人工检查 |

### 5.3 自定义映射

`channel_error_mappings` 可按以下维度把不同上游格式映射到稳定错误码：

1. 精确 `channel_id`
2. `channel_type`
3. 原始 `raw_code`
4. HTTP `status_code`

匹配越具体优先级越高。`channel_id=0`、`channel_type=0`、`status_code=0` 和 `raw_code=*` 表示通配。映射必须指定六位数字错误码、作用域、动作和是否可重试。

## 6. 监控与告警

渠道路由只暴露有界标签，避免把 URL、Key、请求正文或用户内容写入指标。

```text
alltoken_channel_requests_total{channel_id,outcome}
alltoken_channel_switch_total{from_channel,to_channel,mode}
alltoken_channel_circuit_state{channel_id,route,state}
alltoken_error_events_total{event_kind,alltoken_code,category,channel_id}
alltoken_final_errors_total{alltoken_code,category,channel_id}
alltoken_failover_duration_seconds{outcome,mode}
```

Grafana 以渠道为筛选和归因维度，至少展示：

- 各渠道请求量与失败率。
- 渠道之间的切换次数。
- 打开/半开的渠道熔断器。
- 按渠道和六位错误码聚合的错误量。
- 最终对外成功率、P95 延迟和耗尽错误。
- 数据库、Redis、ECS 和应用资源指标。

核心告警包括渠道熔断、渠道切换激增、最终错误率过高和所有候选渠道耗尽。Alertmanager 按 `channel_id` 聚合告警。

## 7. 数据迁移与回滚

新运行时只迁移和读写：

- `billing_group_routes`
- `billing_group_channels`
- `channel_error_mappings`

旧 Cluster、号池和旧切流策略表不再迁移、不再读取，但本次改造不会主动删除这些物理表或历史数据。需要回滚旧版本时，原始数据仍可供旧代码读取；确认新架构稳定并完成数据备份后，才能单独安排物理清理。

## 8. 验收清单

1. 只有管理员可以看到并调用 Load-test Demo 专用统计接口；访问结果不受 `users.group` 取值影响。
2. 一个计费分组只能保存一条路由，且只能选择属于该分组的渠道。
3. Weight 较高的健康渠道获得更高长期流量，不会因遗留 Channel.priority 形成硬分层。
4. 当前渠道 429、5xx 或超时后，按 `max_attempts` 重试并切到下一渠道。
5. 错误记录包含正确的 `alltoken_code`、`channel_id` 和 `error_ref`。
6. 渠道熔断后不再接收新请求，冷却后半开探测成功可恢复。
7. Grafana 能按 `channel_id` 看到请求、失败、切换和熔断；不再依赖 Cluster/号池指标。
8. 所有候选渠道耗尽时返回 `305001`，并触发对应告警。
