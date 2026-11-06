# 网关渠道路由、错误码与监控规范

## 1. 目标架构

网关只保留以下业务关系：

```text
用户 -> 用户分组 -> 计费分组 -> 渠道 -> 上游服务
```

- **用户**：登录账号与 API Key 的所有者。
- **用户分组**：复用 `users.group`，用于用户侧权限和账务规则。
- **计费分组**：API Key 授权的计费与模型能力边界，复用 Token、Ability 和 Channel 已有的 `group`。
- **渠道**：一个可独立请求、计费、重试和监控的上游入口。

上游系统内部是否存在账号池属于上游实现细节。网关收到的上游错误归因到实际请求渠道，并在计费分组内重新选择渠道。

## 2. 渠道选择

可选的路由成员关系保存在 `billing_group_routes` 和 `billing_group_channels` 两张表中。它们只负责确定计费分组是否有显式渠道集合，以及为动态选择提供稳定的优先级和权重。

### 2.1 路由成员字段

| 字段 | 含义 |
| --- | --- |
| `channel_id` | 渠道永久 ID，也是错误与监控定位键 |
| `priority` | 动态评分完全相同时用于稳定打破平局；强制优先渠道用它表达强制层级 |
| `weight` | 近似最高分候选之间的长期流量权重；配置为 0 表示不承接普通流量 |
| `enabled` | 是否参加该计费分组的路由 |

路由成员不再保存单渠道尝试上限、总尝试上限、总超时或成本系数。普通渠道不按 `priority` 形成硬层级；只有启用强制优先的渠道才会先于动态评分。

### 2.2 重试设置

重试设置属于当前渠道和当前计费分组的业务配置，不属于旧路由表字段：

- 渠道 `upstream_max_retries` 表示首个上游请求之后允许的额外重试次数，实际该渠道最多请求 `upstream_max_retries + 1` 次。
- 定价分组的固定重试模式中，`PricingGroupRetryPolicy.retry_times` 表示该分组首个请求之后允许的额外分组尝试次数，分组预算为 `retry_times + 1`。
- 跟随渠道模式（配置值 `follow_channels`，旧值 `active_channels` 会自动归一化）按当前可用渠道跟随计算，分组预算为每个渠道的 `1 + upstream_max_retries` 之和。

因此，固定模式的 `retry_times: 0` 或 `upstream_max_retries: 0` 都表示对应预算只允许一次请求，不表示“失败后再重试一次”；跟随渠道模式不使用 `retry_times`。API Key 不单独覆盖分组重试预算。

系统在完整能力候选集中先执行权限、能力、凭证和并发硬过滤，再按定价分组配置的价格、可用性、负载和 TTFT 策略评分。候选池内才应用权重做平滑分流。跨计费分组重试仅在 API Key 使用 `auto` 分组并明确开启 `cross_group_retry` 时发生。

## 3. 选择和切换过程

一次请求的处理顺序如下：

1. 根据用户和 API Key 确定允许使用的计费分组。
2. 根据计费分组、模型和请求路径过滤可用渠道。
3. 在跨组场景先选择定价分组，再在组内按动态策略建立候选池。
4. 在真正发送前原子预占并发名额。
5. 发起请求并记录渠道尝试结果。
6. 失败后先检查当前渠道是否仍有 `upstream_max_retries` 预算；没有预算时再选择其他渠道。
7. 分组预算耗尽后，只有允许跨组的 API Key 才能进入下一个计费分组。
8. 所有候选渠道耗尽后，对客户端返回最终错误。

每次切换都会重新计算剩余候选，并保留已确认失败渠道的排除状态。请求参数错误、内容策略错误和客户端主动取消不触发渠道切换。

## 4. 错误分层

`source` 表示错误最初来自哪一层：

| `source` | 含义 | 数字段 |
| --- | --- | --- |
| `openai` | 官方 OpenAI 或兼容官方协议的明确原始错误 | `1xxxxx` |
| `channel` | 当前渠道或其上游返回的错误 | `2xxxxx` |
| 空值 | 网关自身生成的错误 | `3xxxxx` |

上游错误来源只接受 `openai` 和 `channel`。网关自身生成的错误不设置来源。错误分类由内置错误目录依据错误来源、原始错误码和 HTTP 状态码完成。

### 4.1 六位数字格式

```text
SCCDDD
```

- `S`：来源层，1=OpenAI，2=渠道，3=网关。
- `CC`：错误类别。
- `DDD`：该类别下可扩展的具体编号。

`stable_code` 是稳定分类，不编码具体渠道。具体失败渠道通过 `channel_id` 和 `channel_name` 表达。

`error_ref` 是一条可检索的错误记录引用，例如 `204001-CH38`，其中 `CH38` 表示实际失败渠道。

### 4.2 主要错误码

| 范围/错误码 | 含义 | 默认动作 |
| --- | --- | --- |
| `102xxx` | 官方认证或地区限制 | `switch_channel` |
| `103xxx` | 官方额度不足 | `switch_channel` |
| `104001` | 官方 429 | `switch_channel` |
| `105xxx` | 官方 5xx/不可用 | `switch_channel` |
| `202xxx` | 渠道凭证错误 | `switch_channel` |
| `204001` | 渠道 429 | `switch_channel` |
| `205xxx` | 渠道上游不可用 | `switch_channel` |
| `210001` | 渠道超时 | `switch_channel` |
| `301xxx` | 客户端请求格式错误 | `none` |
| `302xxx` | 网关鉴权错误 | `none` |
| `303xxx` | 用户额度或预扣费失败 | `none` |
| `305001` | 所有候选渠道已耗尽 | `retry_later` |
| `306xxx` | 网关检测到的渠道配置/Key 错误 | `switch_channel` |
| `307xxx` | 内容策略错误 | `none` 或 `manual` |
| `308xxx` | 协议转换或响应解析错误 | 按作用域处理 |
| `309xxx` | 网关内部错误 | 通常不切流 |
| `310xxx` | 网关到渠道的网络错误 | `switch_channel` |
| `311001` | 渠道不支持模型 | `switch_channel` |

## 5. 错误记录

客户端最终错误和内部上游尝试使用同一结构。示例：

```json
{
  "source": "channel",
  "source_code": "channel.rate_limit_error",
  "stable_code": 204001,
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

- `X-Error-Source`
- `X-Error-Source-Code`
- `X-Error-Stable-Code`
- `X-Error-Ref`
- `X-Error-Retryable`

具体失败渠道通过 `channel_id` 和 `channel_name` 表达；旧版本的产品前缀字段和错误映射配置不再生成或解析。

### 5.1 作用域和动作

| `failure_scope` | 含义 |
| --- | --- |
| `request` | 当前请求本身有问题，不应切流 |
| `credential` | 当前渠道凭证失败，可重试或切换 |
| `channel` | 当前渠道失败，切换到下一渠道 |
| `provider` | 上游服务级故障，当前渠道记失败并切换 |

| `action` | 含义 |
| --- | --- |
| `none` | 不执行额外动作 |
| `retry_channel` | 在渠道尝试预算内重试当前渠道 |
| `switch_channel` | 切换到下一候选渠道 |
| `retry_later` | 已耗尽当前可用路径，建议客户端稍后重试 |
| `abort` | 立即终止 |
| `manual` | 需要人工检查 |

## 6. 验收清单

1. 动态候选只能来自用户和 API Key 有权使用的计费分组及渠道。
2. 高评分渠道成功时不会访问后续渠道。
3. `upstream_max_retries` 控制单渠道首请求后的额外请求次数。
4. `PricingGroupRetryPolicy.retry_times` 控制定价分组级预算。
5. 错误记录包含正确的 `stable_code`、`channel_id` 和 `error_ref`。
6. 所有候选渠道耗尽时返回 `305001`。
