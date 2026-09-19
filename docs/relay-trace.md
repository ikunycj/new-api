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
| `RELAY_TRACE_MIN_FREE_DISK_MB` | `2048` | 剩余空间低于此值暂停写入，`0` 关闭该保护 |
| `RELAY_TRACE_DISK_CHECK_SEC` | `10` | 剩余空间采样间隔（秒） |

### 磁盘水位熔断

审计日志绝不能成为网关丢掉文件系统的原因。轮转 + gzip 不足以保证安全：生产流量波动极大（实测 6.7k–1.6M 请求/天，相差 235 倍），峰值日可能跑赢压缩速度。

低于阈值时暂停写入并打一条日志，恢复后再打一条 —— 只在状态翻转时记录，避免持续低盘时日志本身把磁盘写满。被熔断丢弃的记录计入独立计数器，不混进 `failed`，这样「磁盘满了」和「写坏了」可以区分。

statfs 采样带缓存（默认 10 秒），不会给每条记录都加一次系统调用。采样失败时**选择放行而非暂停** —— 挂载点读不到不应该静默关掉审计。

用 `Bavail` 而非 `Bfree`：文件系统给 root 预留的块计在 `Bfree` 里，信任它会导致普通写入已经 ENOSPC、而保护还以为有空间。

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

下列数字来自生产 `logs` 表近 7 天 126 万条真实元数据：按上下文长度分档、取各档真实权重与平均输出长度，生成结构等价的 body 后实测 JSONL 体积。与测试机实采记录（86.2 KB/条，gzip 5.1%）校准一致。

| 上下文区间（token） | 权重 | 单条均值 | gzip 后 |
|---|---|---|---|
| 0–999 | 86.5% | 32.8 KB | 1.8 KB |
| 1000–4999 | 7.2% | 65.2 KB | 2.7 KB |
| 5000–19985 | 1.0% | 557 KB | 15.6 KB |
| 20012–49998 | 0.9% | 360 KB | 12.3 KB |
| 50000–99998 | 1.8% | 526 KB | 18.5 KB |
| 100001–149997 | 1.4% | 637 KB | 23.8 KB |
| 150001–199975 | 0.6% | 845 KB | 30.6 KB |
| 200014+ | 0.6% | 1372 KB | 50.7 KB |

加权单条均值 **72.9 KB**。

**日常量**（30529 条/天）：原始 2.5 GB/天，gzip 后 0.13–0.75 GB/天，7 天留存 0.9–5.3 GB。

**峰值日**（实测 09-12 达 159 万条，其中 91% 是短请求）：原始 **54.3 GB/天**。即便按乐观的 5% 压缩率，7 天留存也要 19 GB；30% 保守估计下要 114 GB。

两个结论：

1. **原始 54 GB/天足以打满中小磁盘**，这正是磁盘水位熔断必须存在的原因 —— 从写完到压完的时间窗里就可能写满。
2. **上生产必须降采样率。** 按 10% 采样，峰值日降到 5.4 GB/天原始，安全得多。

> 测算方法提醒：峰值日短请求占比高，若拿全局加权均值 86.2 KB 去套算会得到 130.9 GB/天，**高估 2.4 倍**。必须按当日实际构成计算。

## 安全

- 文件权限 `0600`，目录 `0700`
- 请求头采用**白名单**机制，`Authorization` / `X-Api-Key` / `X-Goog-Api-Key` / `Cookie` / `Proxy-Authorization` 永不落盘
- 请求体含用户完整 prompt 明文，**目录访问权限需严格控制**

## 可靠性设计

- **非阻塞投递**：队列满时直接丢弃并计数，绝不阻塞中转链路。磁盘写满或 writer 卡死时，用户请求照常返回
- **单 writer goroutine**：channel 即队列即串行化点，无需锁，不会出现交错的半行
- **不 fsync**：一次 fsync 耗时 1–10ms，会抹平本地追加的微秒级优势；代价是断电丢失最后 1 秒数据，审计场景可接受
- **优雅退出**：`main.go` 在 shutdown 时调用 `relaytrace.Close()`，drain 队列并 flush 缓冲区
- **磁盘水位熔断**：剩余空间低于阈值时暂停写入，见上文「磁盘水位熔断」

### 无需外部定时任务

压缩与清理是**轮转内置**的，不依赖 cron 或任何外部调度：

- 活跃文件超过 `MAX_FILE_MB` → 立即轮转为带时间戳的文件
- 轮转后**后台 goroutine 立刻 gzip**，完成后删除未压缩的原件
- 随即 `prune` 掉超出 `MAX_BACKUPS` 的最旧文件

这样设计是有意的：按大小触发比按时间触发更贴合真实风险。峰值日可能几分钟就写满一个文件，等 cron 到点已经太迟；而空闲日一天都不到阈值，定时压缩只是空跑。压缩在后台执行，不阻塞写入；`close()` 会 `bg.Wait()` 等待在途压缩完成，避免进程退出后留下半截文件。

唯一需要外部调度的是**异地归档**（7 天外传到存储服务器），因为那涉及本机之外的目标。

## 实现要点

挂载位置在 `TokenAuth()` 之后（才有 user_id）、`Distribute()` 之前（响应包装需早于任何写入）。

包装 `gin.ResponseWriter` 时有三个必须保留的行为，否则会破坏中转链路：

1. **`Flush()` 透传** — 否则 SSE 不再逐字输出，用户侧变成最后一次性喷出
2. **`Unwrap()`** — `relay/helper/stream_scanner.go` 的 `ExtendWriteDeadline` 依赖 `http.NewResponseController` 找到底层 writer，断链会导致写超时静默失效
3. **`Hijack()`** — `/v1/realtime` websocket 升级需要

此外中间件读取请求体后会**重新填充 `c.Request.Body`**，保证直接读 body 的 handler 不受影响。以上均有单测覆盖。
