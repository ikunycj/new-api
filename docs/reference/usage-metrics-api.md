# 用量与实时指标 API

面向终端用户的用量查询接口，覆盖两个场景：

- **实时** (`/realtime/*`) — 读进程内存，秒级新鲜度，最多回看 6 小时
- **历史** (`/usage/self`) — 读 PostgreSQL 小时级汇总，可查任意长时间范围，带花费

本文所有示例均在测试环境 `47.251.84.204` 实测通过。

---

## 认证

每个请求需要两个头：

```
Authorization: <access token>
New-Api-User: <用户 ID>
```

### 这两个头的分工

它们**不是两个凭据**。真正的凭据只有 `Authorization` 一个；`New-Api-User` 不提供任何认证能力。

| 头 | 作用 | 一个用户有几个 |
|---|---|---|
| `Authorization` | 唯一凭据，决定「你是谁」 | 1 个（`users.access_token`） |
| `New-Api-User` | 一致性断言，声明「我以为我是谁」 | 不是凭据 |

服务端会把 `New-Api-User` 的值和 access token 解析出的真实用户 ID 做比对，不一致直接拒绝。

实测四种组合：

| 组合 | 结果 |
|---|---|
| token 正确 + `New-Api-User: 1`（本人） | `200` |
| token 正确 + `New-Api-User: 5`（他人） | `401 New-Api-User does not match logged in user` |
| token 正确 + 不带 `New-Api-User` | `401 New-Api-User header not provided` |
| 不带 token + 只带 `New-Api-User: 1` | `401 not logged in and no access token provided` |

最后一行说明：光有用户 ID 完全没用。

**为什么需要这层校验**：防浏览器多标签页串号。假设 A 标签页登出换了账号，B 标签页还残留旧用户的前端状态 —— 此时 B 发请求，cookie 已是新用户，若没有这道断言，新用户的数据会被渲染进旧用户的界面。加上它，这类请求直接被拒，前端据此知道该刷新登录态。

它不参与权限判定，去掉不会让任何人获得额外权限，只会失去这层防串号保护。

### access token 从哪来

管理后台 → 个人设置 → 生成。一个用户只有一个，重新生成会使旧的失效。

### 术语澄清

这套系统里有三个不同的东西都叫 "token"，容易混：

| 概念 | 含义 | 在 API 里 |
|---|---|---|
| **access token** | 管理后台凭据，调 `/api/*` | `Authorization` 头 |
| **API 密钥** | 用户创建的中转密钥，调 `/v1/*` | `token_id` 参数 |
| **tokens** | 计费单位（prompt + completion） | `tokens` 字段、TPM |

本文中 `token_id` 一律指**用户的 API 密钥**，与 access token 无关。为避免歧义，示例里的 shell 变量命名为 `AUTH` 而非 `TOKEN`。

### 示例环境变量

```bash
BASE=http://47.251.84.204:3000
AUTH='ael9Q4R/x/6GU6HtiXTZfmB76wTwB1E='   # access token
USER_ID=1
```

---

## 1. 实时指标

```
GET /api/data/realtime/self
```

返回调用者自己的实时吞吐。用户 ID 取自会话，**不从任何参数读取**。

### 参数

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `token_id` | int | 否 | 按 API 密钥过滤 |
| `model` | string | 否 | 按模型过滤，最长 64 字符 |

两者可同时使用，取交集。

### 示例

```bash
curl -sS "$BASE/api/data/realtime/self" \
  -H "Authorization: $AUTH" \
  -H "New-Api-User: $USER_ID"
```

实际响应（`series` 已省略）：

```json
{
  "success": true,
  "message": "",
  "data": {
    "user_id": 1,
    "now": 1790443566,
    "node_name": "new-api-test-1",
    "windows": [
      {
        "window_seconds": 60,
        "requests": 1,
        "tokens": 41,
        "rpm": 1,
        "tpm": 41,
        "cache_read_tokens": 0,
        "input_tokens_total": 0,
        "cache_hit_rate": null,
        "avg_concurrency": 0.0667
      },
      {
        "window_seconds": 300,
        "requests": 11,
        "tokens": 4971,
        "rpm": 2.2,
        "tpm": 994.2,
        "cache_read_tokens": 11720,
        "input_tokens_total": 14705,
        "cache_hit_rate": 0.797,
        "avg_concurrency": 0.88
      },
      {
        "window_seconds": 3600,
        "requests": 49,
        "tokens": 31787,
        "rpm": 0.8167,
        "tpm": 529.78,
        "cache_read_tokens": 77042,
        "input_tokens_total": 101798,
        "cache_hit_rate": 0.7568,
        "avg_concurrency": 0.1975
      }
    ],
    "series": [ /* 61 个一分钟桶 */ ]
  }
}
```

### 字段说明

| 字段 | 说明 |
|---|---|
| `node_name` | 产生该快照的节点名 |
| `windows` | 固定三档：60 / 300 / 3600 秒 |
| `rpm` / `tpm` | 窗口内均值，按分钟折算 |
| `cache_hit_rate` | `0`–`1`，**或 `null`** |
| `avg_concurrency` | 平均并发估算，见下 |
| `series` | 一分钟粒度序列，覆盖最长窗口 |

#### `cache_hit_rate` 为什么可能是 null

**`null` 不等于「没有请求」。** 它表示这个窗口**没有可用的样本**，有两种完全不同的成因：

| 情况 | `requests` | `input_tokens_total` | `cache_hit_rate` |
|---|---|---|---|
| 窗口内没有任何请求 | `0` | `0` | `null` |
| 有请求，但上游均未回报缓存元数据 | `>0` | `0` | `null` |
| 有请求且上游回报了 | `>0` | `>0` | `0`–`1` |
| 有样本，但一次都没命中 | `>0` | `>0` | `0` |

判断规则只看分母：`input_tokens_total <= 0` 时返回 `null`，否则返回 `cache_read_tokens / input_tokens_total`。

第 2 行是最容易误判的一种 —— 实测数据里出现过 3600 秒窗口有 83 个请求、`cache_hit_rate` 仍为 `null` 的情况。要区分「没请求」和「有请求但无样本」，看 `requests` 字段。

**为什么要把缓存静默的请求排除，而不是算作未命中**：假设一个窗口有 100 个请求，其中 10 个来自支持缓存的上游且全部命中，另外 90 个上游根本不谈缓存。把这 90 个算作未命中，命中率会显示 10%；但真实答案是 100%。因为分母一旦混入无样本的请求，这个比率既不叫命中率，也无法和历史卡片对比。

分母用独立的 `input_tokens_total` 而不是 `tokens`，也是同一个原因：`tokens` 包含补全 token 和全部缓存静默请求。

这条规则与小时级汇总表 `quota_data` 一致，所以实时卡片和历史卡片对同一批数据不会给出互相矛盾的答案。

**前端必须区分 `null` 和 `0`**：

```js
const rate = window.cache_hit_rate;
if (rate === null) {
  render('—');                    // 无样本
} else {
  render(`${(rate * 100).toFixed(1)}%`);   // 真实比率，含真实的 0%
}
```

**不能写 `rate || 0`** —— 那会把 `null` 和真实的 `0` 一起压成 "0%"。用户看到 0% 会判断缓存失效并去排查，而实际情况可能只是没有样本，两个结论对应的运维动作完全不同。

注意 `avg_concurrency` 的处理方式**不同**：没有请求时它是实打实的 `0`，不是 `null`。因为并发可以确定地回答 —— 没有请求就是没有并发。只有缓存命中率存在「无法回答」这个状态。

#### `avg_concurrency` 的含义与局限

按 Little's Law 估算：

```
avg_concurrency = Σ(请求耗时秒数) / 窗口秒数
```

**这是统计估算，不是实时计数器。** 请求的耗时只在它**结束时**才被计入，所以这个值天然滞后。

滞后幅度约等于请求耗时的中位数。以生产实测数据为例：p50 = 15 秒，p90 = 86 秒，**典型滞后约 30 秒**。

看上面的实测响应就能直观看到这个效应：60 秒窗口是 `0.0667`，300 秒窗口是 `0.88`。差了一个数量级，因为长请求还没结束，尚未计入短窗口。

实用建议：

- **UI 上必须标注「约 30 秒滞后」**，否则用户会拿它判断「我现在是不是被限并发了」而被误导
- 判断当前负载用 **300 秒档**最稳，误差约 ±31%，70% 的情况落在真实值 50% 以内
- 突发流量场景下这个值会严重偏低，此时应参考 RPM
- 已知局限：约 21% 的 busy_seconds 来自耗时超过 5 分钟的请求，这部分在结束前完全不可见

### 按密钥 / 模型过滤

```bash
# 按 API 密钥
curl -sS "$BASE/api/data/realtime/self?token_id=1232" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"

# 按模型
curl -sS "$BASE/api/data/realtime/self?model=claude-opus-4-8" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"

# 两者交集
curl -sS "$BASE/api/data/realtime/self?token_id=1232&model=claude-opus-4-8" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

过滤后响应会回显 `token_id` / `model` 字段，客户端据此判断一个迟到的响应属于哪次筛选，避免用户快速切换筛选条件时渲染错数据。

---

## 2. 可选的筛选维度

```
GET /api/data/realtime/self/dimensions
```

列出当前有流量的密钥和模型，用于填充筛选下拉框 —— 不必让用户从全部密钥里盲猜哪些是活跃的。

### 参数

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| `window_seconds` | int | `60` | 只接受 `60` / `300` / `3600` |

### 示例

```bash
curl -sS "$BASE/api/data/realtime/self/dimensions?window_seconds=3600" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

```json
{
  "success": true,
  "data": {
    "tokens": [
      {"token_id": 1232, "token_name": "think 测试", "requests": 16},
      {"token_id": 1244, "token_name": "key111",     "requests": 7}
    ],
    "models": [
      {"model": "claude-opus-4-8", "requests": 16},
      {"model": "claude-opus-4-6", "requests": 7}
    ],
    "truncated": false
  }
}
```

两个列表都按请求数降序。

`truncated: true` 表示维度环已达上限（2000 组合），列表不完整。账户总计不受影响 —— 被拒的组合仍然计入账户级数字，只是无法单独拆出来。

注意默认窗口是 60 秒，流量稀疏时会返回空列表。想看全貌用 `window_seconds=3600`。

---

## 3. 历史用量

```
GET /api/data/usage/self
```

`/realtime/self` 的长周期对应接口。读 PostgreSQL 的小时级汇总表，可查任意时间跨度并带花费，但**精度最细到小时**。

### 参数

| 参数 | 类型 | 说明 |
|---|---|---|
| `start_timestamp` | int64 | Unix 秒 |
| `end_timestamp` | int64 | Unix 秒 |
| `bucket_seconds` | int64 | 聚合粒度，默认 3600 |
| `token_id` | int | 按密钥过滤 |
| `model` | string | 按模型过滤 |

### 示例

```bash
curl -sS "$BASE/api/data/usage/self?start_timestamp=1790400000&end_timestamp=1790450000" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

```json
{
  "success": true,
  "data": {
    "start_time": 1790400000,
    "end_time": 1790450000,
    "bucket_seconds": 3600,
    "totals": {
      "requests": 1129,
      "tokens": 716458,
      "quota": 3792831,
      "cache_read_tokens": 77042,
      "input_tokens_total": 101798,
      "rpm": 1.3548,
      "tpm": 859.75,
      "cache_hit_rate": 0.7568
    },
    "series": [
      {
        "timestamp": 1790400000,
        "requests": 74,
        "tokens": 22123,
        "quota": 114436,
        "cache_read_tokens": 0,
        "input_tokens_total": 0,
        "rpm": 1.2333,
        "tpm": 368.72
      }
    ]
  }
}
```

`quota` 是花费，单位是系统内部额度。**这个接口没有 `avg_concurrency`** —— 并发估算依赖实时环里的 `busy_seconds`，历史汇总表没有这个维度。

### 与实时接口的取舍

| | `/realtime/self` | `/usage/self` |
|---|---|---|
| 数据源 | 进程内存环 | PostgreSQL |
| 时间跨度 | 最多 6 小时 | 任意 |
| 精度 | 10 秒槽，1 分钟序列 | 1 小时 |
| 花费 | 无 | 有（`quota`） |
| 并发 | 有 | 无 |
| 重启后 | 清零 | 保留 |
| 多节点 | 账户级已合并 | 天然全局 |

---

## 4. 管理员接口

需要管理员角色（`AdminAuth`）。普通用户调用返回 403。

### 所有用户概览

```bash
curl -sS "$BASE/api/data/realtime/users?window_seconds=3600" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

```json
{
  "success": true,
  "data": {
    "node_name": "new-api-test-1",
    "now": 1790443681,
    "window_seconds": 3600,
    "users": [
      {
        "user_id": 1,
        "username": "ikunycj",
        "requests": 52,
        "tokens": 36610,
        "rpm": 0.8667,
        "tpm": 610.17,
        "last_seen": 1790443592
      }
    ]
  }
}
```

按请求数降序，窗口内无流量的用户不出现。

### 指定用户详情

```bash
curl -sS "$BASE/api/data/realtime/users/1" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

返回结构与 `/realtime/self` 相同，用于管理员排查具体用户的问题。

---

## 5. 安全边界

### `/self` 接口无法越权

用户 ID 来自 `c.GetInt("id")`（由 access token 解析），不从任何参数读取。实测：

```bash
curl -sS "$BASE/api/data/realtime/self?user_id=5" \
  -H "Authorization: $AUTH" -H "New-Api-User: 1"
# 返回 user_id = 1，user_id 参数被完全忽略
```

### `token_id` 无法用于横向越权

过滤是在**调用者自己的环内部**进行的。传入他人的密钥 ID 只会匹配到空结果，不会触及那个账户的数据。

这也是为什么这里不需要额外的归属校验 —— 用户 ID 是外层约束，且不可由调用方指定。

### 管理员是唯一的跨用户路径

`/realtime/users` 和 `/realtime/users/:id` 走 `AdminAuth`，是读取他人数据的唯一入口。

---

## 6. 错误响应

统一形态：

```json
{"success": false, "message": "错误描述"}
```

实测到的错误：

| 场景 | HTTP | message |
|---|---|---|
| 缺 `New-Api-User` | 401 | `Unauthorized, New-Api-User header not provided` |
| `New-Api-User` 与登录用户不符 | 401 | `Unauthorized, New-Api-User does not match logged in user` |
| 缺 access token | 401 | `Unauthorized, not logged in and no access token provided` |
| `window_seconds` 非 60/300/3600 | 200 | `invalid window_seconds` |
| `token_id` 非正整数 | 200 | `invalid token_id` |
| `model` 超过 64 字符 | 200 | `invalid model` |

注意认证类错误返回 HTTP 401，参数类错误返回 HTTP 200 + `success: false`。客户端不能只看状态码，**必须检查 `success` 字段**。

---

## 7. 行为约定

### 窗口取值被限制为三档

`window_seconds` 只接受 `60` / `300` / `3600`。环只保留 6 小时，更长的窗口会在不足的数据上打长标签，产生误导。

### 空用户返回零值而非错误

从未产生流量的用户会得到三个全零窗口和一条 61 点的零值序列，`cache_hit_rate` 为 `null`，`avg_concurrency` 为 `0`。这样前端不需要为空状态写特殊分支。

注意这两个字段对「无数据」的表达方式不同，原因见上文 `cache_hit_rate` 一节。

### 窗口边界会对齐槽

窗口起点向下对齐到 10 秒边界，因此实际覆盖范围最多比标称多 9 秒。这样做是为了让同一个窗口始终跨越固定数量的槽 —— 否则同一个「1 分钟」在不同时刻会包含 6 或 7 个槽，导致数字无故抖动。

### 数据不持久化

实时环在进程内存中，重启后清零。持久记录是小时级汇总表，由 `/usage/self` 读取。

### 多节点部署下的合并行为

账户级（不带筛选）的数字会跨节点合并：各节点每 5 秒把自己的环摘要写入共享 Redis，读取时合并所有节点。

**带 `token_id` 或 `model` 筛选的请求目前仍是单节点数据。** 按维度发布会让 Redis 键空间乘以（用户 × 密钥 × 模型）的组合数，在有明确需求前不值得这个复杂度。

Redis 不可用时自动降级为单节点数据，不报错。

---

## 附：轮询建议

面板默认 5 秒轮询一次 `/realtime/self`。这个间隔与后台向 Redis 发布摘要的周期一致，再快也拿不到更新的跨节点数据。

需要同时展示多个筛选维度时，优先用一次不带筛选的请求拿账户总计，再按需请求具体维度 —— 不带筛选的路径不触碰维度环，开销更低。
