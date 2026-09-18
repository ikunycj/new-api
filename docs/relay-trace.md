# 中转请求/响应 JSONL 审计（relay trace）

在网关应用层记录**用户侧**原始 request / response，异步写入本地 JSONL 文件，用于审计留证与事后排障。

默认**关闭**，不影响现有部署。

## 为什么在应用层，而不是 nginx

nginx 拿不到审计真正需要的维度：

| 字段 | nginx | 应用层 |
|---|---|---|
| `user_id` / `token_id` | ❌ 只有密文 Authorization | ✅ |
| `channel_id`（实际命中哪个上游） | ❌ | ✅ |
| 重试链路（一次客户端请求可能打多个渠道） | ❌ 只有一行 access_log | ✅ |
| 转换后的真实 model 名 | ❌ | ✅ |

此外 nginx 的 `$request_body` 在超过 `client_body_buffer_size` 时会**静默为空**，响应体则需要 OpenResty + Lua 逐 chunk 拼接，对 SSE 是 $O(n^2)$ 的内存复制，且可能破坏流式实时性。

建议 nginx 保留轻量 access_log 做流量面可观测，body 级审计交给本功能，两边用 `request_id` 关联。

## 为什么不扩展 logs 表

1. `logs` 列表查询是 `SELECT *`（GORM `Find` 展开全部字段），加上 body 列后翻一页日志就会捞回几十 KB × 每页条数的无用数据
2. PostgreSQL 中超过约 2KB 的字段会进 TOAST 副表，`logs` 是系统最高频写入表，会引入写放大与 autovacuum 压力
3. `RecordConsumeLog` 是计费结算点，失败请求走的是 `RecordErrorLog`，而失败样本恰恰最有审计价值

因此 trace **不重复存元信息**，只存 `rid`，其余维度通过 `request_id` 关联回 `logs` 表。

## 为什么是 JSONL 而不是单个大 JSON

| | 单个大 JSON 数组 | JSONL |
|---|---|---|
| 追加写 | 要处理末尾 `]`，需 seek 回退 | 直接 append |
| 并发安全 | 多写者改结尾字节会损坏文件 | 单写者顺序追加 |
| 进程被 kill | 缺 `]`，**整个文件解析失败** | 只丢最后一行 |
| 查询 | 必须全量载入内存 | `grep` / `jq` 流式处理 |

JSONL 也是 ClickHouse / Loki / ELK 的标准导入格式，后续想接分析系统无需转换。

## 配置

全部通过环境变量控制，默认关闭。

| 变量 | 默认 | 说明 |
|---|---|---|
| `RELAY_TRACE_ENABLED` | `false` | 总开关 |
| `RELAY_TRACE_DIR` | `./logs/relay-trace` | 落盘目录 |
| `RELAY_TRACE_SAMPLE_RATE` | `0` | 成功请求采样率，整数百分比 0–100 |
| `RELAY_TRACE_ERROR_ALWAYS` | `true` | 非 2xx 一律记录，不受采样率限制 |
| `RELAY_TRACE_MAX_BODY_KB` | `256` | 单边 body 上限，超出截断并打标 |
| `RELAY_TRACE_QUEUE_SIZE` | `8192` | 投递队列容量，满则丢弃 |
| `RELAY_TRACE_MAX_FILE_MB` | `512` | 单文件轮转阈值 |
| `RELAY_TRACE_MAX_BACKUPS` | `30` | 保留的轮转文件数 |
| `RELAY_TRACE_USERS` | 空 | 用户白名单（逗号分隔），命中则 100% 记录 |
| `RELAY_TRACE_MODELS` | 空 | 模型白名单，命中则 100% 记录 |
| `RELAY_TRACE_DETECT_SIGNATURE` | `true` | 检测响应中的 Anthropic thinking signature |

### 推荐生产配置

```bash
RELAY_TRACE_ENABLED=true
RELAY_TRACE_SAMPLE_RATE=0      # 成功请求不记录
RELAY_TRACE_ERROR_ALWAYS=true  # 失败请求全量留证
```

只记错误：存储可控，而失败样本正是排障最需要的。定向排查某用户或某模型时用白名单，无需调高全局采样率。

## 记录格式

每行一个 JSON 对象：

```json
{
  "rid": "20260918101530-abc123",
  "ts": 1758160530123,
  "status": 200,
  "duration_ms": 3250,
  "method": "POST",
  "path": "/v1/messages",
  "model": "claude-opus-4-8",
  "stream": true,
  "req_headers": {"Content-Type": "application/json", "Anthropic-Version": "2023-06-01"},
  "req": {"model": "claude-opus-4-8", "messages": [...]},
  "resp": "event: message_start\ndata: {...}\n\n...",
  "req_size": 2048,
  "resp_size": 15360,
  "has_signature": true
}
```

字段说明：

- `rid` — 关联 `logs` 表与 nginx access_log 的主键
- `req` — 请求体为合法 JSON 时**原样内联**（`json.RawMessage`，无转义）；非 JSON（如 multipart 音频）落到 `req_raw`
- `resp` — 非流式为 JSON 文本，流式为**原始 SSE 事件流**
- `req_size` / `resp_size` — 真实字节数，即使 body 被截断也保持准确
- `req_truncated` / `resp_truncated` — 标记 body 被 `MAX_BODY_KB` 截断，避免把截断误判为空
- `has_signature` — 见下节

## has_signature：渠道签名纯度审计

派生字段，三态：

| 取值 | 含义 |
|---|---|
| 字段不存在 | 响应中没有 `signature` 键。非 thinking 模型的正常情况 |
| `true` | 存在非空签名 |
| `false` | **签名键存在但值为空** — 上游把签名剥掉了 |

区分「键不存在」和「键存在但为空」很关键：前者是正常的，后者才是脏渠道的证据。

找出剥离签名的渠道：

```bash
# 列出所有空签名请求的 rid
jq -r 'select(.has_signature == false) | .rid' logs/relay-trace/trace.jsonl

# 关联 logs 表定位具体渠道
psql -c "SELECT channel_id, count(*) FROM logs WHERE request_id IN (...) GROUP BY channel_id ORDER BY 2 DESC"
```

相比主动发探测请求，这种方式零 token 成本、样本来自真实流量、可持续观察趋势。

## 常用查询

```bash
# 按 request_id 定位单条
grep '"rid":"20260918101530-abc123"' logs/relay-trace/trace.jsonl | jq .

# 所有 5xx 请求的模型分布
jq -r 'select(.status >= 500) | .model' trace.jsonl | sort | uniq -c | sort -rn

# 耗时超过 30s 的请求
jq 'select(.duration_ms > 30000) | {rid, model, duration_ms}' trace.jsonl

# 查看压缩归档
zcat logs/relay-trace/trace-*.jsonl.gz | jq 'select(.status != 200)'
```

## 容量估算

单条约 20KB（长上下文 Claude 请求可达 30KB+）：

$$\text{日增} \approx QPS \times 86400 \times \text{采样率} \times 20\text{KB}$$

- 100 QPS、1% 采样 → 约 1.7 GB/天，轮转 gzip 后约 300 MB
- 100 QPS、**全量** → 约 170 GB/天

这就是必须保留采样开关的原因。

## 安全

- 文件权限 `0600`，目录 `0700`
- 请求头采用**白名单**机制，`Authorization` / `X-Api-Key` / `X-Goog-Api-Key` / `Cookie` / `Proxy-Authorization` 永不落盘
- 请求体含用户完整 prompt 明文，**目录访问权限需严格控制**

## 可靠性设计

- **非阻塞投递**：队列满时直接丢弃并计数，绝不阻塞中转链路。磁盘写满或 writer 卡死时，用户请求照常返回
- **单 writer goroutine**：channel 即队列即串行化点，无需锁，不会出现交错的半行
- **不 fsync**：一次 fsync 耗时 1–10ms，会抹平本地追加的微秒级优势；代价是断电丢失最后 1 秒数据，审计场景可接受
- **优雅退出**：`main.go` 在 shutdown 时调用 `relaytrace.Close()`，drain 队列并 flush 缓冲区

## 实现要点

挂载位置在 `TokenAuth()` 之后（才有 user_id）、`Distribute()` 之前（响应包装需早于任何写入）。

包装 `gin.ResponseWriter` 时有三个必须保留的行为，否则会破坏中转链路：

1. **`Flush()` 透传** — 否则 SSE 不再逐字输出，用户侧变成最后一次性喷出
2. **`Unwrap()`** — `relay/helper/stream_scanner.go` 的 `ExtendWriteDeadline` 依赖 `http.NewResponseController` 找到底层 writer，断链会导致写超时静默失效
3. **`Hijack()`** — `/v1/realtime` websocket 升级需要

此外中间件读取请求体后会**重新填充 `c.Request.Body`**，保证直接读 body 的 handler 不受影响。以上均有单测覆盖。
