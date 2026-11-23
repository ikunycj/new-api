# 开发、构建与部署

本文说明当前仓库的开发和发布入口。环境和配置约束以 [AGENTS.md](../../AGENTS.md) 为准；既有服务器发布步骤以 [部署技能](../../.codex/skills/deploy-new-api/SKILL.md) 及其配套脚本为准。

## 本地开发

工具版本以 [go.mod](../../go.mod)、[前端 package.json](../../web/default/package.json) 和锁文件为准。前端使用 Bun，日常联调使用 PostgreSQL 和 Redis。

根目录 `.env` 是受保护的本地开发配置，启动前确认已配置正确的测试数据服务。普通开发、构建、启动和部署不得覆盖它。首次配置以及后续修改都需要环境和字段范围明确的授权；示例文件只用于了解变量，不应直接覆盖已有文件。

首次构建前端嵌入资源：

```bash
cd web
bun install --frozen-lockfile
cd default
bun run build
```

从仓库根目录启动后端：

```bash
go run main.go
```

从 `web/default/` 启动前端：

```bash
bun run dev
```

后端使用 `http://127.0.0.1:3000`；当前约定的前端端口为 `http://localhost:5173`，保持 `strictPort: true`。启动前检查分支约定和端口归属，健康的本工作区服务可复用。遇到其他进程占用时先报告，不自动终止进程或改端口。只做测试、类型检查、格式检查或构建时不启动前端服务。

### 测试数据服务

`aliyun` 测试实例与本地后端按当前测试约定都可以作为 master 连接同一套测试 PostgreSQL 和 Redis。两者都可能执行迁移和定时任务，部署后核对共享测试库的 schema 和测试数据；`CHANNEL_UPDATE_FREQUENCY` 默认保持未设置。

这套双 master 拓扑仅用于测试，测试 PostgreSQL 和 Redis 必须与生产服务物理隔离。测试主机的 new-api 和 OpenResty 默认保持运行。

### 日志聚合表检查

AI 协助启动或部署前，应检查应用实际使用的日志库：设置了 `LOG_SQL_DSN` 时使用独立日志库，否则使用 `SQL_DSN` 对应主库。

核对 `log_rollup_state`、`channel_usage_rollups` 的表、字段及查询索引，结构参考 [log_rollup.go](../../model/log_rollup.go)。缺失或不完整时，按 AGENTS.md 执行一次性修复，验证后立即删除临时 SQL 和执行器。只核对库名、schema、表和索引；不得输出连接串或凭证，不删除日志、不重置聚合状态、不手工插入状态行，也不把临时修复加入启动迁移。

## 配置边界

| 配置或入口 | 用途与约束 |
| --- | --- |
| [`.env.example`](../../.env.example) | 变量参考，不存放真实凭证 |
| 根目录 `.env` | 本地开发配置，不能作为部署制品 |
| `deploy/.env` | 原生部署环境配置，与本地 `.env` 独立 |
| 服务器 `/opt/new-api/.env` | 服务器独立维护，普通发布不得替换 |
| [docker-compose.dev.yml](../../docker-compose.dev.yml) | 独立容器开发栈，不应用于重建既有服务器数据服务 |
| [docker-compose.yml](../../docker-compose.yml) | 从当前分支构建 New API，同时启动 PostgreSQL 和 Redis |

任何 `.env` 修改都需要指定环境的逐字段授权；前后只记录文件元数据和 SHA-256 指纹，不输出秘密值或配置 diff。不要在本地、`deploy/` 和服务器配置之间复制、同步或互相覆盖。`SESSION_SECRET` 与 `CRYPTO_SECRET` 应按环境稳定维护。

## 全新部署

部署官方发行版时，使用上游仓库的 `docker-compose.yml`，配置 PostgreSQL 和 Redis 密码及 `SESSION_SECRET` 后执行 `docker compose up -d`。官方模板同时启动 New API、PostgreSQL 和 Redis，生产环境应从官方 Releases 选定镜像版本。[New API 官方 Docker Compose 指南](https://docs.newapi.ai/en/docs/installation/deployment-methods/docker-compose-installation)

运行当前分支的代码时，使用本仓库根目录的 `docker-compose.yml` 从源码构建镜像。根目录的 `docker-compose.yml` 和 `docker-compose.dev.yml` 均支持通过 `GENERATE_DEFAULT_TOKEN=true` 为账号密码注册的新用户创建初始 API 密钥。[README 的 Compose 步骤](../../README.md#docker-compose完整本地服务)

在「系统管理 → 计费与支付 → 用户初始值」中，可以通过「默认 API 分组」为自动创建的初始密钥添加多个定价分组，并调整优先顺序。默认选项为「跟随用户分组」。

## 既有服务器发布

先明确目标主机、目标分支及当前授权范围，再读取 [部署流程](../../.codex/skills/deploy-new-api/references/deployment-playbook.md) 和 [环境拓扑](../../.codex/skills/deploy-new-api/references/production-topology.md)。拓扑文件中的历史地址和容器名必须在发布前核实，不能直接作为当前状态。

1. 核实实时 DNS、SSH 目标、应用和数据服务归属；未经授权不得修改 `alltokenapi.com` DNS。
2. 检查工作区并保留无关修改，按部署技能核对指定目标分支的本地、已获取远端和实时远端 SHA。
3. 在工作站的干净工作树中安装锁定依赖、构建前端并交叉编译 Linux 二进制，生成摘要和归档；上传前核对提交和制品摘要。
4. 上传制品和部署脚本，使用应用当前的 Compose 文件和项目，指定已核实的 PostgreSQL、Redis 容器，复用已有 Docker 镜像及数据网络。保留回滚镜像，只重建应用服务。
5. 核对二进制摘要、镜像修订、容器健康和重启次数、数据库与 Redis 状态、本机及公网 `/api/status`，再验证受影响页面或最小业务请求。
6. 成功后记录版本和回滚信息，清理本次临时制品；失败时按部署脚本和流程执行回滚。

生产主机禁止执行前端、Go 或多阶段 Docker 源码构建，以及新增或重建 PostgreSQL、Redis。`aliyun` 内存有限，禁止运行 `docker system df`。

## 按改动选择验证

- 后端改动运行直接相关包或回归场景，跨公共契约或高风险数据变更时再扩大范围。
- 前端改动按需执行目标文件 lint、`bun run typecheck` 和 `bun run build`；公开文档构建还应通过随构建执行的预算检查。
- Markdown 整理检查本地链接、资源引用、文档路径和 `git diff --check`，不因此启动服务或运行全仓测试。
- 检查通过后审阅 diff，只提交任务相关文件；Pull Request 使用 [项目模板](../../.github/PULL_REQUEST_TEMPLATE.md)。

任何构建、静态检查或本地测试结果都不代表生产已部署或真实支付已验收。发布说明分别记录已验证内容和未验证内容。
