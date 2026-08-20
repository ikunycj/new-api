# Sub2API -> new-api 用户迁移方案

本文只覆盖普通用户、余额、订阅剩余额度和 API Key。日志、请求记录、登录会话和
Sub2API 的内部 ID 不迁移。迁移执行前必须先做只读快照和 dry-run，dry-run 没有
阻断项时才允许写入目标库。

## 1. 源用户范围

只读取未软删除的普通用户：

```sql
deleted_at IS NULL AND role = 'user'
```

每个源用户以 `source_user_id` 建立映射，不能把源 ID 直接当作 new-api 的用户 ID。
迁移记录至少保存 `source_user_id`、目标用户 ID、源数据指纹和迁移批次号，重复执行
同一批次时必须更新/跳过原记录，不能再次增加余额或重复创建 API Key。

## 2. 用户名和邮箱规范

用户名是展示和会话校验所需的非空字段，邮箱是唯一账户标识。迁移预处理按以下顺序计算目标用户名：

同名用户名不再承担唯一登录定位；迁移验收和正式使用应使用邮箱登录。

1. `trim(source.username)`；
2. 结果为空时，使用 `lower(trim(source.email))`；
3. 用户名和邮箱都为空时，阻断该用户，不创建“空用户名”记录；
4. 不截断、不静默改写大小写。用户名是普通展示/登录字段，允许多个用户使用同名用户名；
   迁移幂等性必须依赖 `source_user_id` 映射，不能依赖用户名。

代码导入路径统一调用 `model.ResolveUsername(username, email)`；直接 SQL 或外部 ETL
也必须实现完全相同的顺序，不能只依赖前端表单校验。

应用层不再设置 20 个字符的用户名上限；仍需在 dry-run 中检查目标数据库实际字段容量，
超过数据库容量的记录必须阻断，不能截断。邮箱统一保存为 `lower(trim(email))`，并按目标
库的唯一性规则检查邮箱冲突；用户名不做唯一性检查。

new-api 启动迁移会清理历史 `users.username` 唯一约束/索引；邮箱唯一校验保留。这样已有
数据库升级后也能保存重复用户名，而不会只在新建数据库时生效。

目标库中历史遗留的空用户名也要在迁移前修复：有邮箱就回填规范化邮箱，没有邮箱就隔离，
不能让这类记录进入登录验收。

## 3. 字段和余额映射

| Sub2API | new-api | 规则 |
| --- | --- | --- |
| `username` | `username` | 按上面的非空回退规则处理 |
| `email` | `email` | 去首尾空格并转小写 |
| `password_hash` | `password` | bcrypt 哈希原样复制，不重新哈希 |
| `display_name`（如有） | `display_name` | 直接复制；为空时使用解析后的用户名 |
| `notes` | `remark` | 直接复制，按目标字段容量预检 |
| `status` | `status` | `active` 映射为启用，`disabled` 映射为禁用 |
| `balance` | `quota` | `max(balance, 0) / 5` CNY，再转换为 new-api quota 单位 |
| 有效订阅剩余额度 | `quota` | 按第 4 节计算后与余额相加 |

本方案的迁移范围已经限定为普通用户，因此目标 `role` 固定为普通用户，不迁移管理员权限。

余额换算固定为 `5 USD = 1 CNY`。负余额不向目标写入负额度；源端 `frozen_balance` 是
已从可用余额扣除、等待结算的冻结金额，不能重复加回。迁移前应等待其归零，或将该用户
隔离到人工结算队列。

## 4. 分组和订阅

按源分组名称映射到目标分组：

| Sub2API | new-api |
| --- | --- |
| OpenAI官方转发 | ChatGPT官转 |
| ChatGPT 生图专用 | 生图 |
| ChatGPT Plus | ChatGPT Plus |
| ChatGPT Pro | ChatGPT Pro |
| Claude | Claude Kiro |
| Claude Max 20x | Claude Max满血 |
| ChatGPT 羊毛福利 | 羊毛福利 |
| 其他分组或已删除分组 | ChatGPT Plus |

API Key 的分组必须通过分组名称映射，不能复用源分组数字 ID。没有源分组的 Key 使用
目标默认分组策略，并在 dry-run 报告中单独列出。

只有未软删除、状态为 `active`、未过期且目标分组仍为订阅型的订阅参与余额折算。订阅没有
单独的总剩余额度，按每个仍有效的日/周/月窗口计算：

```text
window_remaining = max(limit - effective_usage, 0)
subscription_remaining = min(all configured window_remaining)
```

窗口使用量按源端重置时间归一化；没有任何数值窗口上限的有效期订阅不能推导为可转余额，
必须人工处理。每条订阅的 `subscription_remaining / 5` 计入用户余额，同一订阅的日、周、
月窗口不能相加。

## 5. API Key

只迁移未软删除的 Key。源端 Sub2API 默认 Key 是 `sk-` 加 64 位十六进制字符串，
而 new-api 的 `tokens.key` 内部值不保存 `sk-` 前缀，前端展示、复制和请求时只补一个
`sk-`。因此迁移时必须先规范化存储值：

```text
target_token.key = trim_prefix(source_api_key.key, "sk-")
```

外部用户仍使用原始的完整 Key；只有目标数据库的内部存储值去掉前缀。必须拒绝或隔离
前缀后仍含额外连字符、空值、重复前缀或超过目标字段长度的自定义 Key，不能把完整源 Key
直接写入 `tokens.key`，也不能让前端再次产生 `sk-sk-...`。迁移验收必须同时检查目标库
存储值不带 `sk-`、界面复制值恰好一个 `sk-`，并使用该值请求 `/v1/models` 返回 200。

目标方案需要保留名称、状态、过期时间、绑定分组、独立限额、已用量和访问限制；
Key 的独立限额不能再次计入用户钱包，避免重复入账。源端已过期或已用尽的 Key 可以迁移
为禁用状态，不能迁移后重新获得可用额度。

当前 `deploy/ikun.love/migrate-sub2api-users.ps1` 的安全范围更窄：它只接受独立额度、
已用额度、IP 白/黑名单和 5 小时/1 天/7 天限流均为零的标准 Key；任何一项不为零都会在
dry-run 阶段阻断。全服迁移前必须确认这些字段全为零，或先扩展并单独验证目标字段映射，
不能把“脚本跳过了该字段”当成“字段已迁移”。

## 6. 执行和验收顺序

1. 备份源库和目标库，导出不含密码明文的 dry-run 报告。
2. 校验目标分组、邮箱冲突、数据库字段容量、冻结余额、订阅窗口和 Key 格式。
3. 按用户事务写入用户、余额和 API Key；每个用户成功后写入源 ID 映射和数据指纹。
4. 验证每个目标用户的用户名非空、邮箱登录、bcrypt 密码登录、原 API Key 认证、余额
   换算和分组映射。
5. 由管理员使用用户管理界面修改一个长用户名，并再改成另一用户已有的用户名，确认两次
   都能保存；邮箱重复时仍应被拒绝。修改用户名后清理用户缓存并建议重新登录，以刷新旧
   会话中的用户名。

任一步骤出现阻断项都停止批次，不跳过冲突、不截断用户名、不重复入账。回滚使用源 ID
映射删除本批次创建的数据或恢复目标库备份；迁移完成后用户已经产生业务数据时，不执行
无条件删除，改为按映射人工回滚。

## 7. 灰度实战复盘（2026-08-20）

### 7.1 已验证的清洁导入

以下两户在目标端没有同邮箱用户、没有目标活动记录，适合走当前脚本的清洁导入路径：

| 邮箱 | 源 ID -> 目标 ID | 导入 quota | Key | mapping 批次 |
| --- | --- | ---: | ---: | --- |
| `1941456753@qq.com` | `6 -> 12` | `19,572,622` | 2 | `20260820-120451` |
| `2674155201@qq.com` | `65 -> 13` | `1,851,954` | 3 | `20260820-121352` |

两户均保留了源端 bcrypt 密码哈希；目标 Token 内部值没有 `sk-`，补回一个前缀后逐个请求
`/v1/models` 均返回 HTTP 200。迁移后目标 quota、Token 数和 mapping 对齐，目标应用、
PostgreSQL、Redis 均保持 healthy，旧 `sub2api.service` 也没有重启。

### 7.2 目标冲突不能直接覆盖

以下两户被明确拦截，不能使用当前 `-Apply` 路径覆盖：

| 邮箱 | 目标现状 | 源端候选导入 | 结论 |
| --- | --- | ---: | --- |
| `1542960201@qq.com` | ID 10，quota `5,040,922`，已用 `209,078`，1 个 Token，36 条日志 | `5,100,000` | 保留目标现状，等待合并授权 |
| `3479224465@qq.com` | ID 3，quota `749,823`，已用 `177`，6 条日志，无 Token | `121,586,758` | 保留目标现状，等待合并授权 |

邮箱冲突不等于“已经迁移”：必须同时检查 `sub2api_user_migration_mappings`。目标已有
余额、密码、Token、登录或消费记录时，默认策略是保留目标用户名/密码/余额/历史和已有
Token；只有在管理员明确批准后，才可追加一次源余额并补导入缺失 Key，而且要先保存目标
行快照、写入幂等 mapping，不能用清洁导入 SQL。

### 7.3 灰度中暴露的并发风险

`2674155201@qq.com` 在第一次 dry-run 时显示目标冲突，随后目标记录消失并被另一执行流程
成功导入。全服迁移必须停止并发迁移进程、暂停注册/余额变更，或至少为每个批次设置唯一
操作窗口；dry-run 与 apply 之间还要重新检查邮箱、mapping 和源 `frozen_balance`，避免
把变化中的目标状态误判为可导入。

## 8. 晚上全服迁移 Runbook

### 8.1 迁移前冻结和备份

1. 确认源 `ssh ikun.love-sub2api` 与目标 `ssh ikun.love` 的 SSH 别名、DNS 和服务健康；
   旧 Sub2API 必须保持 active，不能为迁移停止它。
2. 暂停新注册、充值、余额消费和 API Key 管理，至少记录冻结开始时间；不能在源数据持续
   变化时做全量快照。
3. 在源库和目标库分别做数据库级备份，并保存备份文件的权限、时间和 SHA-256。目标端每户
   写入前仍会生成不含密码明文和完整 Key 的审计快照；审计文件在服务器的
   `/opt/new-api/migration-backups/` 下，不能提交 Git。
4. 先只读盘点用户数、普通用户数、邮箱重复数、冻结余额数、有效订阅数、Key 状态和特殊
   Key 策略数；任何异常先进入人工队列。

### 8.2 Dry-run 阻断规则

迁移脚本有三种输入范围，默认都是 dry-run，只有显式加入 `-Apply` 才可能写目标库：

```powershell
# 灰度：只处理明确列出的账户。
& .\deploy\ikun.love\migrate-sub2api-users.ps1 `
  -Email 'user-a@qq.com','user-b@qq.com'

# 全量：源端普通用户在一个 PostgreSQL 查询快照中读取，随后批量预检目标。
& .\deploy\ikun.love\migrate-sub2api-users.ps1 -All

# 需要人工复核快照时，显式落盘。该文件包含 bcrypt 哈希和 API Key，必须只保存在
# deploy/ikun.love/migration-snapshots/ 这类受限且被 Git 忽略的目录。
& .\deploy\ikun.love\migrate-sub2api-users.ps1 `
  -All -SnapshotOutputPath .\deploy\ikun.love\migration-snapshots\full.json

# 审核后的同一快照可以反复 dry-run；传入文件前由操作者确认 ACL 仅限管理员。
& .\deploy\ikun.love\migrate-sub2api-users.ps1 `
  -SnapshotPath .\deploy\ikun.love\migration-snapshots\full.json
```

`-All` 不是数据库备份，也不替代源端冻结。它将一次 SQL 语句看到的普通用户、订阅和
Key 固定为当前运行的输入，避免逐户读取时跨越多个源端状态；源端的注册、充值、消费和
Key 变更仍必须在全量 apply 前冻结。全量模式不会在终端或 JSON 报告写出完整邮箱、密码
哈希或完整 Key：每行只使用不可逆的邮箱短指纹，Key 仅以数量和策略类别计数。

全量 dry-run 对目标用户、既有 mapping 和所有 Token（包括软删除 Token，以符合唯一索引）
做批量预检。`-Apply` 前再逐户检查一次邮箱、mapping 和 Key 冲突，以防 dry-run 与写入
之间出现新注册或其他并发导入。

本批次新增导入候选必须全部显示 `ready` 才能进入 apply。`already_migrated` 是已完成的
幂等结果，不应再次 apply；`-Apply` 对任何其他结果都会整体拒绝写入，而不是悄悄跳过后
报告“成功”。常见阻断结果及报告字段如下：

| 结果 | 含义 | 复核字段 |
| --- | --- | --- |
| `source_key_policy_blocked` | Key 格式或未支持的额度、已用量、IP、限流策略 | `SpecialKeys`、`Detail` |
| `source_blocked` | 非 active、冻结余额、订阅窗口或 quota 范围异常 | `BalanceUsd`、`SubscriptionUsd`、`Detail` |
| `target_conflict_skipped` | 目标有同一规范化邮箱 | `TargetId` |
| `target_key_conflict_skipped` | 目标任意 Token 行已使用该内部 Key | `Tokens`、`Detail` |
| `batch_key_conflict_blocked` | 同一源快照内两个用户声明相同 Key | `Detail` |
| `source_duplicate_email` | 源端规范化邮箱不是一对一 | `Detail` |

`GroupAliases` 会报告未知/空源组回退到 `ChatGPT Plus` 的次数，便于在 apply 前确认别名规则；
它不会包含 API Key 值。

其余下列结果也必须停止该用户，不得“跳过后算成功”：

- `source_not_found`、非普通用户、非 active、`frozen_balance != 0`；
- 邮箱为空、邮箱冲突或目标已有真实业务记录；
- 用户名和邮箱无法解析，或备注/用户名超过目标字段容量；
- Key 不是 `sk-` 加 64 位十六进制、重复、已软删除，或包含当前脚本不支持的独立额度、
  已用额度、IP 白/黑名单、限流窗口；
- 源分组映射后的目标组不存在，且 `ChatGPT Plus` 也不可用；
- 有效订阅没有可计算的数值窗口，或订阅状态/过期时间无法确认。

### 8.3 分批写入和验收

1. 先迁移 1-5 户，使用显式 `-Apply`；每户事务必须同时写用户、Token 和 mapping，事务
   返回的用户数、Token 数、mapping 数必须分别与预期一致。

```powershell
& .\deploy\ikun.love\migrate-sub2api-users.ps1 `
  -Email 'user-a@qq.com','user-b@qq.com' -Apply
```

全服 apply 仅在同一份已审核快照的 dry-run 没有任何阻断项、两端备份完成且源端已冻结时执行：

```powershell
& .\deploy\ikun.love\migrate-sub2api-users.ps1 `
  -SnapshotPath .\deploy\ikun.love\migration-snapshots\full.json -Apply
```

不要先执行 `-All` dry-run、等待源数据继续变化、再执行另一次 `-All -Apply` 并把两次当作同一
快照；这种情况要么重新审核最新 dry-run，要么使用受限 ACL 的 `-SnapshotPath`。快照文件、
数据库备份和线上 JSON 报告均不得提交 Git 或发送到非受限位置。

2. 每批完成后核对：目标邮箱/用户名非空、quota 公式、密码哈希、Token 数/状态/过期时间/
   分组、mapping 指纹和审计快照路径。数据库内部 Token 必须是 64 字符且不含 `sk-`。
3. 对每个 active Token 在不打印 Key 的前提下请求目标本机 `/v1/models`，必须返回 200；失败
   的 Key 单独阻断，不继续扩大批次。
4. 观察目标应用、PostgreSQL、Redis、错误日志和消费记录；确认旧 Sub2API 服务、8080 端口
   和原公网健康检查没有重启或异常。
5. 批次验收通过后再处理下一批。重复运行 dry-run 只能得到 `already_migrated`，不得再次
   增加 quota 或创建 Token。

### 8.4 冲突账户和回滚

冲突账户不使用清洁导入脚本。管理员若批准合并，必须另行执行并记录：目标迁移前快照、
源数据指纹、追加 quota、实际新增的 Key、目标密码保留结果和新的 mapping；追加动作必须
可重放且只能成功一次。目标用户已经产生新业务数据后，不删除用户作为回滚；优先恢复备份
或做可审计的反向调账/禁用新增 Token。清洁导入失败时可按 mapping 删除本批次新建行，
但不能删除已经被用户使用过的目标数据。

### 8.5 迁移结束

全量验收通过后，再解除注册/充值/消费冻结，并保留源 Sub2API 只读运行一段观察期。迁移
报告、审计快照和数据库备份不进入 Git；仓库只保存脚本、规则和脱敏经验，不保存密码、
完整 API Key、`.env`、构建产物或线上报告。
