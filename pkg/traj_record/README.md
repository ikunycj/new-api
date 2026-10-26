# traj_record：按 API 协议记录调用

本分支基于 `ikunycj/new-api` 的 `d9af547`。HTTP 录制已接入现有 `/v1` 路由，位于 RelayTrace 之后、Distribute 之前；没有替换 RelayTrace。目标基线没有 Responses WebSocket relay，因此本分支提供 `BeginWebSocket` 录制接口及单测，但没有可用的 WebSocket 接线。下文 WebSocket 格式描述接口契约，不表示本分支已部署该传输。

## 当前行为（记录格式 v3）

默认采集 `/v1/messages`、`/v1/responses`、`/v1/responses/compact`、`/v1/chat/completions` 的 POST 请求，不要求已知 User-Agent。`TRAJ_RECORD_ENABLED=true` 仍是总开关。Pi、OpenCode 使用这些接口时均可采集。

- `api_family` 是 `messages`、`responses` 或 `chat_completions`；`agent` 是可选的 UA 识别结果，未知时不填写，完整 UA 仍保留。为旧读取工具兼容，`client` 暂时保留协议别名 `claude`、`codex`、`chat`，**不表示实际客户端身份**。
- `TRAJ_RECORD_API_FAMILIES=messages,responses,chat_completions` 控制协议范围。未设置时兼容旧 `TRAJ_RECORD_CLIENTS`，否则默认全选。旧 `TRAJ_RECORD_CLAUDE_UA_PREFIXES` / `TRAJ_RECORD_CODEX_UA_PREFIXES` 仍可显式限制 UA；未设置时不筛选。迁移时检查旧的显式配置，避免继续漏采。
- 每次 HTTP 调用一个 `call`。只对能够逐字节还原的请求拆分去重，否则写 `req_raw` / `req_raw_b64`。包括空请求、字符串 input、漂亮打印 JSON、重复 JSON 键。SHA-256 始终对应原始请求。原始响应仍受 `MAX_RESP_MB` 上限约束，超过时标记截断。
- 去重索引按协议、authenticated user/token 隔离，有 session 时再细分；`parent_rid` 只是存储引用。没有 session 的调用也能精确去重。**token、缓存键、前缀引用均不是会话身份**。`prompt_cache_key` 只留在请求里，不再回填 `thread_id`。
- 部分采样仅对已知 session 生效；没有 session 的 HTTP 调用保留全部。`SAMPLE_RATE=0` 仍关闭全部采集。WebSocket 按连接全量保留（rate 非零时），避免遗漏控制事件或拆散连接内上下文；因此设置 0.1 不意味着总体只有 10% 流量。
- Responses WebSocket 录制接口供 relay 在读取客户端消息及实际写回消息处调用（本分支尚未接线）。`ws_open` / `ws_message` / `ws_call` / `ws_close` 保留连接 ID、连接内序号、方向、时间、原始消息、截断/IO 失败，以及调用关联信息；包含控制消息、网关错误、单独发送的终止事件。保存的是 WebSocket 应用消息，不是 TCP 分片。每条消息上限使用 `MAX_RESP_MB`；所有记录复用有界队列、磁盘保护和丢弃计数。
- `ws_call` 另外保存该次原始 create 和 user/channel/request 等元数据，便于审计；当前 WebSocket 原始事件不做前缀去重。连接内序号的缺口可提示丢记录；连接 ID 不代表真实聊天 session。断连或轮转后仍须保留相关文件才能分析整个连接。

生产样本验证结果见 [2026-10-26 验证报告](validation-20261026.md)。独立采集仓库的 Python 导出器可以逐字节重建 v2/v3 HTTP 请求；该离线工具不属于本 PR。WebSocket 事件须按原生事件分析，不能当作完整 HTTP 调用。

### 单 key 回放（2026-10-23）

把现有 RelayTrace 样本中的可解析请求统一置于单一 user/token、无 session 的索引范围，使用现有算法。输入是历史稀疏样本，不代表长期生产负载上限；此处不模拟按文件大小轮转，测试的是该批请求的索引留存。

| 样本 | 调用数 | 历史原文 | 省下的历史字节 | 索引项峰值 | GC 后保留堆增量 |
| --- | ---: | ---: | ---: | ---: | ---: |
| GPT | 2,103 | 294.5 MB | 156.2 MB | 46,801 | 5.45 MB |
| Claude | 1,824 | 96.2 MB | 37.6 MB | 15,834 | 2.39 MB |

没有证据支持现在替换算法，因此本次保留现有索引及上限。活跃 token scope 不会按内部旧项独立过期、超限可能整块淘汰的问题仍在；轮转清空索引，缺失命中只影响文件大小。以上内存仅是索引留存的近似测量，不是进程峰值、队列或响应缓存。复现：在打好补丁的上游源码运行 `TRAJ_INDEX_SAMPLES=/path/samples.jsonl go test ./pkg/traj_record -run TestReplaySingleKeyIndex -v`。

下文的 Claude Code/Codex 行为分析说明原始设计动机；客户端假设不作为 v3 通用采集的准入条件。

traj_record 在网关上按 API 协议记录每一次选中的调用：完整请求、完整原始响应（包括 Claude 的 thinking 和 signature、Codex 的 reasoning 和 encrypted_content），以及尽量多的关联线索。存下来的数据用于离线重建 agent 轨迹，交给大模型打分，之后可能用于训练。

traj_record 使用独立配置和输出文件。RelayTrace 面向通用请求审计，traj_record 面向保留调用底稿与离线重建，两个中间件可以分别或同时启用。

默认关闭。设置 `TRAJ_RECORD_ENABLED=true` 才会生效。

## 0. 设计思路：先看这一节

这一节讲方案是怎么推出来的。后面各节是细节，可以按需查阅。依据是 2026-10 初对生产网关 RelayTrace 样本（`claude-trace-samples.jsonl`，2500 条 Claude 调用）的核对。

### 起点：理论上只要最后一个请求

Claude Code 是无状态客户端，每次调用 `/v1/messages` 都会带上完整的对话历史，包括之前每一轮的 thinking 和 signature。所以理论上：

```
整条轨迹 = 会话最后一次请求的 messages + 这次请求的响应
```

前面的请求都是它的前缀，可以不要。流式传输只影响响应怎么下发，和这一点无关。

### 核对样本后发现的特例

把第 N 次的响应和第 N+1 次请求里对应的 assistant 消息逐字段比较，原理基本成立，但有几类特例，每一类都推翻了「只存最后一个」的一部分：

| 特例 | 现象 | 对方案的影响 |
|---|---|---|
| 客户端改写历史 | 能比对的 381 轮里有 96 轮 `tool_use.input` 和模型原始输出不同（如去掉 `cd /app &&`，补上 `"replace_all": false`）。`cache_control` 断点每轮移到新的末尾，末尾的 `<total_tokens>` system 消息会变形或被替换 | 历史里的 assistant 消息不等于模型输出，**每一轮都要存原始响应**（第 5 节） |
| 没有下一轮 | 最后一轮、被中断的一轮，都没有后续请求来携带它 | 同上 |
| 压缩（compaction） | 压缩后的请求里，早期历史被替换成摘要 | 最后一个请求不再包含完整轨迹，**必须每次调用都记** |
| 一个会话多条线程 | 子 agent、标题生成等辅助请求、用户回退后的分支，共用同一个 `session_id` | 「最后一个请求」本身没有定义，不能按会话只留一条 |
| 采样、截断、丢弃 | RelayTrace 10% 采样、请求超过 256KB 截断、队列满就丢。长程轨迹最容易被截 | 对选中的会话逐次记录，不做请求级采样，不截断（第 8 节） |

所以结论反过来了：**选中会话的每一次调用都要完整记录请求和响应**。剩下的问题是体积。长会话里同一段历史会被重复发送几百次。

### 去重：按消息算原始字节的最长公共前缀

不去猜「这条请求接着哪条」，而是对 `messages` 的每个前缀算累积哈希。新请求在索引里找最长的已存前缀，只存后面新增的部分（`parent_rid` + `prefix_len` + `append`，第 3 节）。这样分支、重试、子 agent 都不需要特殊处理：任何存过的历史版本都可以当父记录。

上面的「客户端改写历史」在这里正好不碍事。改写只发生在历史末尾的一两条消息上。按原始字节比较时，被改写的那条自然落在公共前缀之外，被重新存一次。样本上验证，每轮最多多存一条消息。所以**不做任何规范化**：规范化会让存下来的和发出去的不一致，还要一直追着 Claude Code 的版本改规则。

压缩后的第一个请求前缀对不上，会存成 keyframe（`prefix_len = 0`）。这只多占空间，不影响正确性。压缩前后怎么接起来，留给离线处理。

`system` 和 `tools` 每次整块重复，单独存成按内容哈希去重的 blob。其余字节保持原样，可以逐字节还原请求并用 sha256 校验（第 4 节）。

### thinking 和 signature

signature 是这份数据里最需要保住的东西，有这几条约束：

- **只在流的末尾出现**：signature 通过 thinking 块最后的 `signature_delta` 下发。流被截断时，就算 thinking 正文已经收到了，也拿不到 signature。所以响应**不截断**，另记 `resp_complete`（是否收到 `message_stop`）。离线处理时丢掉不完整的响应。
- **存 SSE 原文，不在网关里解析**：`resp` 是逐字节的 SSE。`thinking_blocks` 和 `signatures` 只是方便检索的计数。离线工具从原文重新组装块，thinking、text、tool_use 的交错顺序，以及每个 signature 属于哪个块，都不会丢。
- **以响应为准，不以历史为准**：样本里 Claude Code 回传的 thinking 和 signature 和响应逐字节一致，但历史里同一条消息的其他部分（tool_use.input）可能被改过。所以 agent step 一律取当次调用的原始响应。
- **正文可能是空的**：正文是否下发由请求里的 `thinking.display` 决定。签名对应的是服务端的原始思维链，不是下发的正文。当前 Claude Code 对 Opus 发的是 `{"type":"adaptive","display":"updates"}`：大多数 thinking 块正文为空、只有 signature，少数块带一两句进度文字。即使有正文，也只是摘要或进度，不是原始推理。
- **转格式时的取舍**：ATIF 的 `extra.anthropic_content` 保留原始块，签名和块的对应关系完整。OpenAI 格式没有标准字段，用扩展字段 `reasoning_content` 和 `thinking_signatures`。一条消息里有多个 thinking 块时，这两个字段对不齐，需要精确对应时以 ATIF 为准（离线转换工具不在本 PR 中）。

### 其余取舍

- **在 agent 侧记录**：记的是 Claude Code 实际发出和收到的字节，不是网关转给上游的字节。网关可能改写的地方都记在 `upstream` 字段里，离线分析能判断两者是否等价（第 1 节）。
- **网关只留线索，关联放到离线做**：会话、分支、子 agent、压缩的拼接都不在网关里判断。网关只尽量多地留下 `session_id`、`parent_rid`、`resp_tool_uses`、时间戳等线索（第 6 节）。网关的逻辑保持简单，关联规则改了也只需离线重算。
- **每个文件自包含**：文件轮转时清空索引，每个文件可以单独重建、单独删除（第 7 节）。

### 会话采样

`TRAJ_RECORD_SAMPLE_RATE=0.1` 选择约 10% 的会话，默认 `1` 全量记录，`0` 不采集。非法值会打印错误并按 `0` 处理。

采样键为客户端类型、网关 user id 和 `session_id`，使用固定 SHA-256 哈希桶。因此同一会话的主 agent、共享 session 的子 agent、重试和 compaction 前后使用同一个结果；无须等待主请求完成，也无须维护会被 TTL 或轮转清空的采样缓存。未选中的请求在复制请求体、捕获响应和入队之前跳过，正常代理照常工作。

这里的单位是整个 session，而非单轮用户任务。Codex 从 `client_metadata.session_id` 或 `Session-Id` / `Session_id` 请求头取 ID；Claude 从 `metadata.user_id` 取 ID。compaction 改变历史、使去重回退为 keyframe，不影响采样，只要 session ID 未变。`/v1/responses/compact` 同样适用。

限制：子 agent 必须携带与主 agent 相同的 session ID。新 session ID 会作为新会话采样；仅有 thread / parent thread ID 无法无状态地确定根会话。部分采样（`0 < rate < 1`）时，缺失 session ID 的请求全量保留，避免独立抽样产生残缺轨迹。跨网关需保持相同 user id 和采样率；运行中修改采样率可能使已有会话改变判定，应在采集批次之间调整。采样不能消除队列溢出、磁盘不足和响应上限造成的缺失。

## 1. 记录位置：agent 侧，而非上游侧

中间件挂在 `router/relay-router.go` 的 `/v1` http 路由组上，位置在 TokenAuth 之后（此时已有 user id 和 token id）、Distribute 之前（此时还没有 handler 写过响应）。因此记录的是 **Claude Code 发给网关的原始字节**，以及 **网关返回给 Claude Code 的原始字节**。记录的不是网关发给 Anthropic 上游的字节。

选择 agent 侧的依据是对 new-api 转换链路的检查（Anthropic 渠道，默认配置，未开透传）：

- 请求经 `dto.ClaudeRequest` 反序列化再序列化后发往上游。变化包括：map 的 key 被排序、`<>&` 被转义成 `<` 等、数字变成 float64 表示、非法 UTF-8 被替换、未知顶层字段被丢弃（目前 Claude Code 没有发送未知字段）。这些变化不改变语义。
- 可能改变语义的改写都来自运营配置：模型映射、`-thinking`/effort 后缀、默认 max_tokens、渠道 SystemPrompt、参数覆盖、请求头覆盖、RemoveDisabledFields。Vertex 渠道会丢弃 `context_management` 和 `metadata`，Bedrock 渠道会丢弃 `metadata`。
- 开启透传（全局 PassThroughRequestEnabled 或渠道 PassThroughBodyEnabled）时，请求体原样发往上游。
- 响应方向：SSE 中 `data:` 行的内容原样转发，`event:` 行由网关重新生成，`message_delta` 中缺失 usage 时网关会补上。

结论：对于没有上述配置改写的 Anthropic 渠道，agent 侧记录与上游实际收到的内容语义等价。为了让离线分析能识别出不等价的情况，每条记录都带 `upstream` 字段，记录透传、渠道 SystemPrompt、SystemPrompt 覆盖是否生效，以及参数覆盖、请求头覆盖、模型映射的具体内容，同时记录 `channel_id`/`channel_type`/`channel_tries`：

- `param_override_ops`：实际生效的参数覆盖（渠道配置合并渠道亲和性模板之后），按执行顺序每条操作一项，含 `mode`/`path`/`from`/`to`/`value`。旧格式的顶层键记为 `set`。带条件的操作只标 `conditional`，不记条件内容（是否命中要看请求本身）。`set_header` 不记 `value`，因为可能是凭据。
- `body_override`：上面的操作里有任何一条可能改请求体时为 true。只有 `pass_headers`、`set_header` 等请求头操作时为 false，例如内置的 `claude cli trace` 亲和性规则只做 `pass_headers`。
- `header_override_keys`：渠道请求头覆盖设置了哪些请求头，只记名字不记值。
- `model_mapping`：渠道的模型映射原文。

旧记录里的 `param_override`/`header_override` 是布尔值，只表示「非空」，分不清是否改了请求体。

后续计划：做一个上游侧的记录点，用来抽样验证"agent 侧 ≈ 上游侧"这一结论。

## 2. 协议适配及可选客户端线索

存储层不认识任何 API：一条请求被拆成"历史数组"（按前缀去重）、若干 blob（每个文件存一次）和其余顶层字段 `params`。每个客户端（`clients.go` 中的 `client`）只声明这几部分在哪里，以及怎样从请求和响应里提取关联线索：

| 兼容协议别名 | 路径 | 可选客户端识别前缀（非默认过滤） | 历史 | blob | 代码 |
| --- | --- | --- | --- | --- | --- |
| `claude` | `/v1/messages` | `claude-cli/` | `messages` | `system`、`tools` | `claude.go` |
| `codex` | `/v1/responses`、`/v1/responses/compact` | `codex_` | `input` | `instructions`、`tools` | `codex.go` |
| `chat` | `/v1/chat/completions` | 无 | `messages` | `tools`、`functions` | `chat.go` |

过滤条件（全部满足才记录）：

- `POST`，路径属于 `TRAJ_RECORD_CLIENTS` 中某个已启用的客户端（默认全部启用）
- 默认不筛选 `User-Agent`；显式设置 `TRAJ_RECORD_<NAME>_UA_PREFIXES` 时才按前缀筛选
- 设置了 `TRAJ_RECORD_USERS` 时，user id 必须在列表中

其他请求直接 `c.Next()`，不做任何分配。

### Claude Code

`metadata.user_id` 中的 session_id、device_id、account_uuid 会被解析出来作为线索。两种格式都支持：新版是 JSON 字符串 `{"device_id":…,"account_uuid":…,"session_id":…}`，旧版是 `user_<device>_account_<uuid>_session_<uuid>`。

### Codex

Codex 以 `store: false` 无状态模式调用 Responses API，每次请求都在 `input` 中带上整个线程，包括每个 reasoning 项的 `encrypted_content`（完整思维链的密文，只有 OpenAI 能解密）。所以它和 Claude Code 一样可以按前缀去重。生产样本（`gpt-trace-samples.jsonl`）中同一线程相邻的 768 对请求，全部是对上一条的严格前缀扩展。0.14x 版本把工具列表放在 `input[0]`，类型是 `additional_tools`，不是顶层的 `tools`，所以它同样参与前缀去重。

`client_metadata` 提供 session_id、thread_id、turn_id（没有时退回到 `Session-Id`/`Thread-Id` 请求头，旧版是 `Session_id`，`prompt_cache_key` 单独保留为候选线索，不回填线程 ID）、`x-codex-installation-id`（记作 device_id），子 agent 还有 `x-codex-parent-thread-id`（记作 parent_thread_id）。去重的范围是 session：fork 出来的子 agent 线程与父线程同属一个 session，复制过来的历史可以直接引用父线程的记录。`x-codex-turn-metadata` 里还有 request_kind、thread_source、subagent_kind、forked_from_thread_id 等字段，它们原样保存在 `params.client_metadata` 中。

`/v1/responses/compact` 是 Codex 的远程压缩，返回的压缩结果同样是密文（`compaction` 项的 `encrypted_content`），也会被记录。

## 3. 去重：按消息粒度的原始字节最长公共前缀

Claude Code 每次请求都会携带完整对话历史，长会话中同一段历史会被重复发送几百次。逐条全量存储不可接受，所以需要去重。

### 做法

- 把请求体的 `messages` 数组拆成一条条原始 JSON 字节（不做任何规范化）。
- 对每个前缀计算累积哈希：`h[0] = sha256(scope)`，`h[i+1] = sha256(h[i] ‖ sha256(msg_i))`。scope 包含协议、user id、token id；有明确 session_id 时再细分，不把这个范围解释为会话身份。
- 在内存索引中查找最长的已存前缀，得到 `parent_rid` 和 `prefix_len`。记录只存 `append = messages[prefix_len:]`。
- 写盘成功之后，才把本条记录新存的前缀插入索引，保证索引永远不会指向文件里不存在的记录。

重建任意一条请求的 messages：`messages(rid) = messages(parent_rid)[:prefix_len] + append`，沿 parent_rid 链递归即可。

### 为什么不做规范化

Claude Code 每轮会改写历史末尾的一两条消息：`cache_control` 断点会移动到新的末尾消息上；末尾 `role:system` 的 `<total_tokens>` 消息会从 block 形式变成纯字符串，或者被替换。Claude Code 还可能改写 `tool_use.input`。

如果先规范化（去掉 cache_control 等）再比较，会让"存下来的"和"发出去的"不一致，而且需要一直追着 Claude Code 的版本变化维护规则。使用原始字节时，被改写的那条消息自然会落在公共前缀之外，作为 append 的一部分被重新存储，代价只是多存一两条消息。

在样本（`claude-trace-samples.jsonl`）上验证：凡是找到父记录的请求，父记录长度与公共前缀长度之差为 1 的有 406 次，为 0 的有 63 次，没有更大的差值。也就是说，原始字节前缀匹配每轮最多多存一条消息。

### 为什么不用"每个会话只存最近两版"的环形方案

那种方案需要猜测哪条请求是哪条请求的延续。分支（用户回退后重新生成）、压缩、子 agent 都会让猜测失败。前缀索引按哈希精确匹配，不需要猜，任何历史版本都可以作为父记录，分支自然支持。

### system 与 tools

`system` 和 `tools` 每次都整块重复，而且一个会话内基本不变。它们存成单独的 `blob` 行，key 为内容 sha256 的前 16 字节（hex）。调用行通过 `refs`（如 `{"system":…,"tools":…}`）引用。Codex 的 `instructions` 同样处理。每个文件内同一 blob 只写一次。

### 其余顶层字段

`model`、`max_tokens`、`thinking`、`metadata`、`stream`（Codex 还有 `reasoning`、`include`、`client_metadata` 等）其余顶层字段原样存在 `params` 中，不去重（体积很小）。

## 4. 字节精确，可校验

- 原始 JSON 片段（messages、system、tools、params 的各个值）以字节拼接的方式写入行内，不经过 `json.Marshal`。`json.Marshal` 会把 RawMessage 压缩，并把 `<>&` 转义，导致字节不一致。
- `req_sha256` 是客户端发来的原始请求体的 sha256。
- `key_order` 记录顶层 key 在原始请求体中的顺序。只有当"按 key_order 把各部分拼回 `{"k":v,...}`"能逐字节还原请求体时才写入这个字段（写入前已经实际校验过）。因此离线工具重建后计算 sha256，与 `req_sha256` 比对，就能证明重建无误。
- v3 不再 compact 请求；旧 v2 的 `req_compacted: true` 记录只能重建压缩后的字节，导出工具会明确标记其不是原始字节。
- 无法按 `key_order` 精确重建（例如空白、重复 key）时，v3 保存原始 `req_raw` / `req_raw_b64`，不拆分。
- 请求体不是带 messages 数组的 JSON 对象时，整体存进 `req_raw`（非 UTF-8 时存 `req_raw_b64`）。

验证结果（`TestReplaySamples`）：用样本中的 995 条 Claude Code 请求回放，995 条全部重建成功，sha256 全部一致。

## 5. 每轮都存完整原始响应

响应原样存在 `resp` 中（SSE 原文，包括 thinking_delta 和 signature_delta；非 UTF-8 时存 `resp_b64`）。不尝试用"下一轮请求里的 assistant 消息"代替响应，原因有两个：

- Claude Code 会改写历史中的 assistant 消息（例如 `tool_use.input`），所以下一轮请求里的内容不等于模型的实际输出。
- 最后一轮（以及被中断的那一轮）根本没有"下一轮"。

为方便检索，额外从响应中提取摘要字段：`resp_id`、`resp_model`（实际作答的模型）、`stop_reason`、`usage_start`、`usage`、`resp_tool_uses`（tool_use 的 id 和 name）、`thinking_blocks`、`signatures`、`resp_complete`（是否收到 message_stop）。

响应不截断。`TRAJ_RECORD_MAX_RESP_MB`（默认 64MB）只用来防止失控的流耗尽内存，超出时标记 `resp_truncated`，`resp_size` 仍是实际总字节数。

## 6. 离线关联（压缩、子 agent、分支）

关联不在网关里做，全部留给离线处理。网关只负责尽量多地留下线索。

- **同一会话的轮次**：明确的 `session_id` 及请求/响应关联证据；`parent_rid`/`prefix_len` 仅用于存储重建。
- **分支**：多条记录的 parent_rid 指向同一个父记录。
- **压缩（compaction）**：压缩后的请求历史被替换成摘要，前缀不匹配，会作为 keyframe（`prefix_len = 0`）存储。关联方法：同一 `session_id`；时间上紧跟压缩前的最后一轮；压缩请求（Claude Code 发出的"请总结"请求）的响应文本通常就是新历史开头的摘要，可以做文本匹配。
- **子 agent**：子 agent 的请求通常带不同的 system/tools 组合，首条 user 消息就是父 agent 中 Agent/Task 工具 `tool_use.input.prompt` 的内容。用这段文本去匹配父 agent 记录中的 tool_use，再用 `resp_tool_uses` 中的 id 找到对应的 tool_result。
- **tool_use id → 工具名**：直接看 `resp_tool_uses`，不需要解析 SSE。

调用行上的线索字段：

| 类别 | 字段 |
| --- | --- |
| 时间 | `ts_start`、`ts_first_byte`、`ts_end`（毫秒） |
| 身份 | `user_id`、`username`、`user_group`、`token_id`、`token_name`、`group`、`client_ip` |
| Claude Code | `session_id`、`device_id`、`account_uuid`、`headers`（含 User-Agent、anthropic-beta 等） |
| 路由 | `channel_id`、`channel_name`、`channel_type`、`channel_tries`、`upstream`、`model` |
| 请求 | `req_size`、`req_sha256`、`n_messages`、`parent_rid`、`prefix_len`、`system_ref`、`tools_ref`、`key_order` |
| 响应 | 见第 5 节 |
| 顺序 | `rid`（网关请求 id）、`instance`、`seq` |

请求头按名称过滤：名称中含 `auth`、`key`、`token`、`secret`、`cookie`、`password`（不区分大小写）的请求头一律不记录，所以 Authorization、X-Api-Key 不会落盘。

## 7. 文件格式与自包含

目录 `TRAJ_RECORD_DIR`（默认 `./logs/traj-record`），当前文件为 `traj.jsonl`，文件及轮转压缩文件的创建权限为 0640，目录为 0750，允许同组用户读取（实际创建权限受进程 umask 影响）。已有文件和目录的权限不会自动修改，升级时需按需执行 `chmod g+rx "$TRAJ_RECORD_DIR"` 和 `chmod g+r "$TRAJ_RECORD_DIR"/traj*.jsonl*`。每行一个 JSON 对象，`type` 字段区分三种行：

```
{"instance":"host-a","pid":123,"ts":1790000000000,"type":"segment_start","v":2}
{"v":2,"type":"blob","kind":"system","hash":"<32 hex>","data":<原始 system JSON>}
{"v":2,"type":"blob","kind":"tools","hash":"<32 hex>","data":<原始 tools JSON>}
{"v":2,"type":"call","client":"claude","rid":"...","seq":1,...,"history_key":"messages","refs":{"system":"<hash>","tools":"<hash>"},"params":{<原始其余顶层字段>},"append":[<原始历史项>...]}
```

v2 相对 v1 的变化：新增 `client`、`history_key`，`n_messages` 改名为 `n_history`，`system_ref`/`tools_ref` 合并为 `refs`。v1 文件里只有 Claude 记录，离线脚本两种格式都能读。

响应摘要字段按客户端填写：Claude 填 `thinking_blocks`、`signatures`，Codex 填 `reasoning_items`、`encrypted_items`（带 encrypted_content 的输出项数）。两者都填 `resp_id`、`resp_model`、`stop_reason`（Codex 为 response 的 status，`incomplete` 时附带原因）、`usage`、`resp_tool_uses`（Claude 的 tool_use id，Codex 的 call_id）。

`client_canceled` 表示 handler 返回时客户端已经断开。agent 中途放弃一次生成（用户打断，或者 Codex 收到子 agent 的消息后重开这一轮）时，响应停在半截，`resp_complete` 为 false，同时 `client_canceled` 为 true。上游自己中断时，`client_canceled` 为 false。注意 Codex 常在收到 `response.completed` 后立即断开，而 new-api 还要等上游流结束、做计费，所以完整的响应也经常带 `client_canceled: true`。只有和 `resp_complete: false` 同时出现时，它才表示中途放弃。

- blob 行总是写在引用它的 call 行之前。
- **每个文件自包含**：文件超过 `TRAJ_RECORD_MAX_FILE_MB` 时轮转，轮转同时清空前缀索引和 blob 集合，并在新文件开头写 `segment_start`。新文件中每个会话的第一条记录都是 keyframe，所有 parent_rid 和 blob 引用都能在同一个文件内找到。任意一个文件可以单独处理、单独删除。
- 进程重启也会写 `segment_start`，并且内存索引从空开始。因此离线工具在遇到 `segment_start` 时应清空自己的状态。重建器只需要按行顺序读取一个文件。
- 轮转后的文件改名为 `traj-<YYYYMMDD-HHMMSS.mmm>.jsonl`，并在后台压缩成 `.jsonl.zst`。`TRAJ_RECORD_MAX_BACKUPS` 默认为 0，即全部保留（这些文件是训练数据，不是日志）。

重建算法（`traj_record_test.go` 中的 `reconstruct` 是参考实现）：

```
for line in file:
    segment_start: 清空 blobs 和 history
    blob:          blobs[hash] = data
    call:          msgs = history[parent_rid][:prefix_len] + append   (无 parent_rid 时只有 append)
                   history[rid] = msgs
                   若有 key_order: body = "{" + join(f'"{k}":' + part(k) for k in key_order, ",") + "}"
                     其中 part(history_key) = "[" + join(msgs, ",") + "]"
                          k 在 refs 中时 part(k) = blobs[refs[k]]
                          其他 part(k) = params[k]
                   校验 sha256(body) == req_sha256（req_compacted 时除外）
```

### zstd 窗口

网关使用 klauspost/compress 的 zstd 编码器，窗口为 `TRAJ_RECORD_ZSTD_WINDOW_MB`（默认 128MB）。去重之后文件里仍然有大量跨记录的重复内容（例如每轮重新存储的末尾消息、相似的 keyframe），大窗口能利用这些重复。该编码器没有 long-distance matching 模式，如果需要最高压缩率，可以离线用 `zstd -d` 解压后，再用 `zstd --long=31 -19` 重新压缩。解压 `--long` 压缩的文件时同样需要传 `--long=31`。

## 8. 可靠性

- 请求 goroutine 只复制请求体和响应字节，所有解析、哈希、写盘都在单个后台 writer goroutine 中完成，所以索引不需要加锁。
- 入队先尝试非阻塞写入；队列满时最多等待 `TRAJ_RECORD_ENQUEUE_TIMEOUT_MS`（默认 200ms），仍然满则丢弃该条记录。丢弃会计数（`Stats()`），并输出限频日志（最多每 10 秒一条）。丢失一条记录不会破坏后续记录：下一条记录只会匹配到更短的前缀，或者成为 keyframe。
- `seq` 在每个进程内单调递增，writer 取出的每条记录都占用一个序号。文件中 seq 的跳号表示该记录写盘失败，或者因磁盘空间不足被跳过。入队阶段被丢弃的记录不占用 seq，需要看日志和 `Stats()` 中的 dropped 计数。
- 磁盘剩余空间低于 `TRAJ_RECORD_MIN_FREE_DISK_MB`（默认 2048MB）时暂停写入（每 10 秒检查一次），恢复后自动继续。暂停期间跳过的记录计入 diskPaused。
- 中间件用 defer 提交记录，handler panic 时记录仍会提交。
- 响应包装器通过嵌入 `gin.ResponseWriter` 保留 Flush，并实现 `Unwrap`，所以 SSE 照常逐块推送，`http.ResponseController` 也能找到底层 writer。
- 优雅退出时 `Close()` 会排空队列、刷盘，并等待后台压缩完成。
- 多实例部署时每个实例写自己的目录（或者用 `instance` 字段区分）。同一会话的请求可能落在不同实例上，此时每个实例各自形成自包含的链，离线可以按 `session_id` + 时间合并。

前缀索引的内存：每存一条消息占一个索引项（32 字节哈希 + rid）。超过 `TRAJ_RECORD_INDEX_TTL_MIN`（默认 6 小时）未活动的会话会被清出索引；索引项超过 `TRAJ_RECORD_INDEX_MAX_ENTRIES`（默认 200 万）时，按最近使用时间淘汰会话，直到降到上限的 90%。被淘汰的会话下次请求会作为 keyframe 存储，只增加文件体积，不影响正确性。

## 9. 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TRAJ_RECORD_ENABLED` | `false` | 总开关 |
| `TRAJ_RECORD_DIR` | `./logs/traj-record` | 输出目录 |
| `TRAJ_RECORD_INSTANCE` | 主机名 | 写入每条记录的实例名 |
| `TRAJ_RECORD_API_FAMILIES` | `messages,responses,chat_completions` | 逗号分隔的启用协议 |
| `TRAJ_RECORD_CLIENTS` | 全部 | 旧协议别名配置，API_FAMILIES 优先 |
| `TRAJ_RECORD_CLAUDE_UA_PREFIXES` | 未设置（不筛选） | Messages 的可选 User-Agent 前缀限制 |
| `TRAJ_RECORD_CODEX_UA_PREFIXES` | 未设置（不筛选） | Responses 的可选 User-Agent 前缀限制 |
| `TRAJ_RECORD_CHAT_UA_PREFIXES` | 未设置（不筛选） | Chat Completions 的可选 User-Agent 前缀限制 |
| `TRAJ_RECORD_SAMPLE_RATE` | `1` | 会话采样比例，0–1；缺失会话 ID 的请求在非零采样率时全量保留 |
| `TRAJ_RECORD_USERS` | 空（全部） | 逗号分隔的 user id 白名单 |
| `TRAJ_RECORD_QUEUE_SIZE` | `256` | 队列长度；每项包含完整请求和响应，同时也是内存上限 |
| `TRAJ_RECORD_ENQUEUE_TIMEOUT_MS` | `200` | 队列满时的最长等待时间 |
| `TRAJ_RECORD_MAX_RESP_MB` | `64` | 单个响应的捕获上限 |
| `TRAJ_RECORD_MAX_FILE_MB` | `1024` | 轮转阈值 |
| `TRAJ_RECORD_MAX_BACKUPS` | `0`（全部保留） | 保留的轮转文件数 |
| `TRAJ_RECORD_ZSTD_WINDOW_MB` | `128` | 轮转文件的 zstd 窗口；0 表示不压缩 |
| `TRAJ_RECORD_MIN_FREE_DISK_MB` | `2048` | 剩余空间低于此值时暂停；0 表示不检查 |
| `TRAJ_RECORD_INDEX_TTL_MIN` | `360` | 会话在索引中的空闲过期时间 |
| `TRAJ_RECORD_INDEX_MAX_ENTRIES` | `2000000` | 索引项上限 |

## 10. 测试

```
go test ./pkg/traj_record/
# 用真实样本回放并校验每条请求的 sha256（样本含真实用户数据，不在仓库中）
TRAJ_RECORD_SAMPLES=/path/claude-trace-samples.jsonl go test -run TestReplaySamples -v ./pkg/traj_record/
TRAJ_RECORD_SAMPLES=/path/gpt-trace-samples.jsonl go test -run TestReplaySamples -v ./pkg/traj_record/
```

样本回放结果：

- Claude：995 条调用，526 条 keyframe；请求 138.1MB + 响应 47.8MB，写入文件 110.1MB（未压缩）。
- Codex：1922 条调用，1152 条 keyframe；请求 294.6MB + 响应 337.7MB，写入文件 560.1MB（未压缩）。响应体积大，是因为 `response.created`、`response.in_progress` 事件会回显完整的 instructions 和工具定义，`response.completed` 又会重复一遍全部输出项。这些重复留给 zstd 压缩处理。这份样本是稀疏抽样，一半以上的会话只有一条请求，所以 keyframe 比例很高。连续采集时，同一会话的大部分请求都能找到父记录，去重效果会明显更好。

## 11. 已知限制与待验证

- 未验证：压缩请求本身（Claude Code 发起的"请总结"请求）是否是旧历史的前缀延伸；`claude --resume` 恢复会话时 session_id 和历史字节是否保持一致。这两点都只影响去重率，不影响正确性。
- 记录的是 agent 侧字节，不是上游实际收到的字节（见第 1 节）。上游侧记录点留待后续实现。new-api 的 Codex 渠道（ChatGPT 订阅）在转发前会补空的 `instructions`、强制 `store=false`，并删掉 `max_output_tokens`、`temperature` 等字段。
- Responses WebSocket 录制 API 已包含单测，但目标基线没有相应 relay，尚未接线。
- 离线格式转换和会话构图不属于此插件的运行路径。
