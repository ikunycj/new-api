# storie 轨迹验证（2026-10-26）

结论：该文件的 HTTP 记录可精确重建，没有发现损坏、跨 scope 引用或完成标记错误。此结论针对已落盘记录，不证明入口流量全部被采集，也不代表每次模型调用成功。

## 样本与范围

- 文件：`storie-traj.jsonl`，508,342,262 字节；原始内容不提交到 PR。
- SHA-256：`ec695c6140ed8a5e5399e12eb8ca7a0e37e4cc4095ab3801eebc545ec6460a9f`。
- 时间（UTC）：2026-10-22 15:10:10.853 至 2026-10-26 14:09:59.856。
- 4 个 segment、123 条 blob、4,706 次 HTTP 调用；v2 为 3,018 次，v3 为 1,688 次。
- v3 从 2026-10-25 05:48:03.198 UTC 开始，覆盖约 32 小时。
- Messages 3,811 次，Responses 895 次。没有 Chat Completions、compaction 路径或 WebSocket 样本，不能据此宣称这些路径已通过生产验证。

## 数据核对

- 全部 4,706 个请求逐字节重建，SHA-256 与 `req_size` 全部一致；Python 和 Go 两个实现独立通过。
- 4,376 个前缀引用均可解析；未发现跨协议、user、token 或 session 的引用。全部 blob 内容哈希正确。
- 每个 segment 内 HTTP seq 连续、rid 无重复。首段 seq 从 3,897 开始，符合文件轮转清空索引但保留进程序号的行为，不视为文件内部丢记录。
- user、token、channel、model 均有值；结束时间不早于开始时间。
- 响应字节长度与 `resp_size` 一致，截断数为 0。原始响应没有独立的外部对照或响应哈希，因此这里只验证内部一致性。
- 原始 SSE 结束事件与 `resp_complete` 全部一致：4,452 次完成、254 次未完成。
- HTTP 200 为 4,473 次；其余 233 次为 400/401/403/500/503，均标记未完成。
- 21 次 HTTP 200 缺少结束事件，其中 20 次标记客户端取消；另 1 次是 v2 Messages，仅有 message_start/content_block_start，无法仅凭轨迹判断中断原因。
- v3 的 1,688 次中，1,553 次完成，130 次非 200，5 次 HTTP 200 未完成且全部标记客户端取消。
- 全文件 `client_canceled` 为 536 次，可能与完成重叠；取消标记本身不等于响应缺失。

未取得部署日志/Stats 或入口请求总数，无法排除入队丢弃、采样或磁盘保护导致的文件外缺失。没有用存储引用或 token 身份推断完整聊天会话。

## PR 适配边界

目标 `ikunycj/new-api` main 为 `d9af547`，不同于部署插件所用官方 rc.41。保留目标已有 HTTP 路由与 RelayTrace 顺序，将 ChannelSettings import 适配回目标的 `dto`。不升级目标仓库或依赖。

同步协议选择、Chat Completions、精确原文 fallback、按认证身份隔离去重、未知 session 保留策略及 WebSocket 录制 API。目标没有 Responses WebSocket relay，API 及单测保留但没有接线；不能将部署仓库的 WebSocket 支持表述为目标 PR 已支持。

## 复现

目标 PR 使用 Go 1.26.10：

```sh
go test ./pkg/traj_record ./common/relaytrace ./middleware ./router
go test -race ./pkg/traj_record ./common/relaytrace ./middleware
TRAJ_CAPTURE_FILE=/path/storie-traj.jsonl go test ./pkg/traj_record -run TestAuditDeployedCapture -v
git diff --check
```

采集仓库另提供仅输出聚合统计的审计命令（不属于网关运行依赖）：

```sh
python3 scripts/audit_traj_capture.py instances/storie-traj.jsonl
PYTHONPATH=scripts python3 -m unittest scripts.test_audit_traj_capture scripts.test_traj_capture
```
