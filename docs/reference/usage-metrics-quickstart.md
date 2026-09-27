# 用量接口对接说明（外部对接用）

面向业务方（如邓坤团队）对接实时/历史用量数据的快速上手文档。完整字段说明见 [usage-metrics-api.md](./usage-metrics-api.md)，本文只讲怎么拿到凭证、怎么发第一个请求。

---

## 1. 认证怎么拿

登录 https://alltokenapi.com → 右上角个人设置 → 生成/复制「系统访问令牌」（Access Token）。

- 一个账号只有一个 access token，重新生成会让旧的失效
- 用户 ID 在同一个页面能看到，也可以登录后打开浏览器开发者工具，看任意一个 `/api/*` 请求的 header

---

## 2. 需要两个 header，缺一不可

```
Authorization: <access token>
New-Api-User: <你的用户 ID>
```

**这两个不是一人一套的两把钥匙。** `Authorization` 才是真正的凭据；`New-Api-User` 只是要求你把自己的用户 ID 也带上，服务端拿它和 token 解析出的身份核对，对不上就拒绝。单靠 `New-Api-User` 拿不到任何数据。

联调排错时可以对照这张表：

| Authorization | New-Api-User | 结果 |
|---|---|---|
| 正确 | 填自己的 ID | `200` |
| 正确 | 填别人的 ID | `401 New-Api-User does not match logged in user` |
| 正确 | 不填 | `401 New-Api-User header not provided` |
| 不填 | 只填 ID | `401 not logged in and no access token provided` |

---

## 3. 第一个请求

```bash
BASE=https://alltokenapi.com
AUTH='你的 access token'
USER_ID=你的用户ID

curl -sS "$BASE/api/data/realtime/self" \
  -H "Authorization: $AUTH" \
  -H "New-Api-User: $USER_ID"
```

返回：

```json
{
  "success": true,
  "data": {
    "user_id": 1,
    "node_name": "new-api-prod-1",
    "windows": [
      {"window_seconds": 60,   "requests": 0,  "rpm": 0,    "tpm": 0,      "cache_hit_rate": null, "avg_concurrency": 0},
      {"window_seconds": 300,  "requests": 7,  "rpm": 1.4,  "tpm": 1022.4, "cache_hit_rate": null, "avg_concurrency": 0.27},
      {"window_seconds": 3600, "requests": 77, "rpm": 1.28, "tpm": 902.67, "cache_hit_rate": null, "avg_concurrency": 0.073}
    ],
    "series": [ /* 最近 1 小时，一分钟一个点 */ ]
  }
}
```

只返回**调用者自己**的用量，`user_id` 从 token 解析而来，无法查询他人数据。

---

## 4. 按密钥 / 模型筛选

```bash
# 先看有哪些密钥/模型在跑（默认 60 秒窗口流量少时可能为空，建议查 3600）
curl -sS "$BASE/api/data/realtime/self/dimensions?window_seconds=3600" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"

# 按密钥筛
curl -sS "$BASE/api/data/realtime/self?token_id=1232" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"

# 按模型筛
curl -sS "$BASE/api/data/realtime/self?model=claude-opus-4-8" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

---

## 5. 历史用量（带花费，可查任意时间段）

```bash
curl -sS "$BASE/api/data/usage/self?start_timestamp=1790400000&end_timestamp=1790450000" \
  -H "Authorization: $AUTH" -H "New-Api-User: $USER_ID"
```

小时级精度，覆盖范围不受 `/realtime/self` 最多 6 小时的限制，但比不了分钟级细节。

---

## 6. 两个字段容易踩坑，接入时务必注意

### `cache_hit_rate`

`null` 表示**没有样本**，不是 0%。原因：只有上游明确回报了缓存元数据的请求才计入分母，窗口内没有这类请求时返回 `null`。

```js
// ✗ 错误：把无样本和真实的 0% 混为一谈
render(`${(rate || 0) * 100}%`);

// ✓ 正确
if (rate === null) render('—');
else render(`${(rate * 100).toFixed(1)}%`);
```

### `avg_concurrency`

滞后估算（按 Little's Law：`Σ耗时秒数 / 窗口秒数`），不是实时计数器。请求耗时只在结束时才计入，滞后幅度约等于请求耗时中位数（典型约 30 秒）。突发流量场景下这个值会明显偏低，**不要用它做实时限流判断**，要看当前负载优先用 300 秒窗口的 RPM。

---

## 7. 参数取值限制

- `window_seconds` 只接受 `60` / `300` / `3600`，其他值返回 `invalid window_seconds`
- 认证错误返回 HTTP 401，参数错误返回 HTTP 200 + `success: false` —— **不能只看状态码，要检查 `success` 字段**

---

## 更多

完整字段说明、管理员接口、安全边界、行为约定见 [usage-metrics-api.md](./usage-metrics-api.md)。
