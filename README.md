<div align="center">

<img src="./web/default/public/logo.png" alt="New API" width="120" />

# New API

**面向组织和团队的统一大模型网关与 AI 资产管理平台**

[![License](https://img.shields.io/github/license/QuantumNous/new-api?color=brightgreen)](./LICENSE)
[![Latest release](https://img.shields.io/github/v/release/QuantumNous/new-api?color=brightgreen&include_prereleases)](https://github.com/QuantumNous/new-api/releases)
[![Docker image](https://img.shields.io/badge/docker-dockerHub-blue)](https://hub.docker.com/r/CalciumIon/new-api)

</div>

New API 把多个模型厂商和异构接口收敛到一个可管理的入口。团队可以在同一套用户、令牌、渠道、模型、权限、用量和成本规则下接入 OpenAI、Claude、Gemini、Azure、AWS Bedrock 以及其他兼容或自定义上游服务；调用方则可以继续使用熟悉的 OpenAI、Claude 或 Gemini API 格式。

它适合私有化部署、企业内部 AI 平台、研发团队共享网关，以及其他已经取得上游服务授权的场景。项目不提供或代办任何上游账号、API Key 或模型服务，使用者需要自行遵守上游服务条款和所在地区的法律法规。

## 你可以用它做什么

- **统一接入**：用一个 API 入口管理多家模型服务，按渠道和模型路由请求。
- **兼容现有客户端**：支持 OpenAI Chat Completions、Responses、Realtime，以及 Claude Messages、Google Gemini 等接口格式，并提供部分格式转换。
- **可靠路由**：支持渠道权重、失败重试、模型限流、缓存和多节点配置同步。
- **组织管理**：管理用户、令牌、分组、模型权限、渠道凭据和操作审计。
- **用量与成本**：记录请求用量，按模型和组织进行额度扣减、成本核算和统计分析。
- **多媒体与异步任务**：覆盖文本、图片、音频、视频、Embedding、Rerank 及部分异步任务渠道。
- **可观测性**：提供运行状态、请求日志、Prometheus 指标和后台数据看板。

## 项目结构

| 层次 | 主要技术 |
| --- | --- |
| 后端 | Go、Gin、GORM |
| 前端 | React 19、TypeScript、Rsbuild、Base UI、Tailwind CSS |
| 数据 | SQLite、MySQL、PostgreSQL、Redis |
| 接口 | REST、SSE、WebSocket、OpenAI/Claude/Gemini 兼容协议 |

```text
.
├── main.go                 # Go 服务入口
├── router/                 # 路由与中间件装配
├── controller/             # HTTP 控制器
├── service/                # 业务服务与任务调度
├── model/                  # GORM 模型、查询与迁移
├── relay/                  # 上游适配器、请求转换与流式处理
├── common/                 # 配置、缓存、JSON、日志等公共能力
├── dto/ types/ constant/   # 请求 DTO、协议类型与常量
├── web/default/            # React 管理前端
├── deploy/                 # 部署与运维配置
└── docs/                   # 开发、部署和专项设计文档
```

后端采用 `Router -> Controller -> Service -> Model` 分层；模型厂商相关逻辑集中在 `relay/`，前端按功能模块组织在 `web/default/src/features/`。

## 快速运行

### Docker Compose（完整本地服务）

需要 Docker Engine 和 Compose 插件。先准备配置：

```bash
git clone https://github.com/QuantumNous/new-api.git
cd new-api
cp .env.example .env
```

编辑 `.env`，取消注释或补充以下必需配置并替换示例值：

```dotenv
POSTGRES_USER=new_api
POSTGRES_PASSWORD=replace-with-a-database-password
POSTGRES_DB=new_api
REDIS_PASSWORD=replace-with-a-redis-password
SESSION_SECRET=replace-with-a-stable-random-secret
CRYPTO_SECRET=replace-with-another-stable-random-secret
```

不要把真实密钥提交到 Git。随后构建并启动：

```bash
docker compose up -d --build
```

打开 <http://localhost:3000>。查看状态和日志：

```bash
docker compose ps
docker compose logs -f --tail=200 new-api
```

停止应用时使用 `docker compose down`。只有确认要删除本地开发数据库卷时才使用 `docker compose down -v`。

### 使用已发布镜像

只需本地试用时，可以让服务使用 SQLite：

```bash
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/new-api:latest
```

未设置 `SQL_DSN` 时，服务会使用 `/data`（或 `SQLITE_PATH` 指定位置）下的 SQLite 数据库。生产环境建议使用 PostgreSQL 或 MySQL，设置稳定的 `SESSION_SECRET` 和 `CRYPTO_SECRET`，并固定镜像版本而不是使用 `latest`。

更多生产部署方式见 [部署文档](./docs/development-deployment.md) 和 [官方文档](https://docs.newapi.pro/zh/docs/installation)。

## 本地开发

### 环境要求

| 工具 | 建议版本或说明 |
| --- | --- |
| Go | 1.25+（以 `go.mod` 为准） |
| Bun | 1.x；前端统一使用 Bun，不使用 npm/yarn/pnpm |
| PostgreSQL / MySQL / SQLite | 三者均受支持；日常联调推荐 PostgreSQL |
| Redis | 可选；启用后用于缓存和多节点同步 |
| Docker Compose | 运行完整开发栈时需要 |

### 方式一：本机运行 Go 和前端

复制配置模板并按实际环境填写。只做轻量试用时可以省略 `SQL_DSN`，程序会回退到 SQLite；团队联调时请明确配置测试数据库和 Redis。

```bash
cp .env.example .env
```

首次拉取代码时，先安装依赖并生成 Go 后端需要嵌入的前端资源：

```bash
cd web
bun install --frozen-lockfile
cd default
bun run build
cd ../..
```

启动后端（默认 `http://127.0.0.1:3000`）：

```bash
go run main.go
```

另开终端启动前端（默认 `http://localhost:5173`）：

```bash
cd web/default
bun run dev
```

Rsbuild 会把 `/api`、`/mj` 和 `/pg` 请求代理到本地 Go 服务。前端端口固定为 `5173`，端口冲突时请先处理占用进程，不要静默改端口。

### 方式二：Docker 提供开发依赖

该方式让 Docker 运行 PostgreSQL、Redis 和后端，前端仍在本机运行，适合前端开发和接口联调：

```bash
docker compose -f docker-compose.dev.yml up -d
cd web
bun install --frozen-lockfile
cd default
bun run dev
```

修改 Go 代码后重新构建后端容器：

```bash
docker compose -f docker-compose.dev.yml up -d --build new-api
```

停止开发栈：

```bash
docker compose -f docker-compose.dev.yml down
```

仓库提供了常用 Make 入口：

| 命令 | 作用 |
| --- | --- |
| `make dev-api` | 启动 Docker 开发后端、PostgreSQL 和 Redis |
| `make dev-api-rebuild` | 重建并启动开发后端 |
| `make start-api` | 本机启动 Go 后端 |
| `make dev-web` | 安装依赖并启动前端 |
| `make dev` | 启动开发后端和前端 |
| `make build-web` | 构建前端并写入 Go 的嵌入资源 |

## 开发约定

### 后端

- 新功能按 `router -> controller -> service -> model` 分层；厂商适配器放在 `relay/channel/`。
- JSON 的序列化和反序列化使用 `common/json.go` 提供的包装函数，不在业务代码中直接调用 `encoding/json` 的操作函数。
- 数据库代码必须同时兼容 SQLite、MySQL（>= 5.7.8）和 PostgreSQL（>= 9.6）；优先使用 GORM 查询方法，确有必要使用原生 SQL 时提供各数据库分支。
- 请求中的可选数值字段使用指针类型保留显式的 `0`、`false` 等值；涉及额度、计费和任务时遵守 `pkg/billingexpr/expr.md` 及项目中的额度饱和规则。
- 不在代码、提交记录、日志或截图中写入 API Key、数据库密码、会话密钥和其他敏感配置。

### 前端

- 技术栈为 React 19、TypeScript、TanStack Router、TanStack Query、Zustand、Base UI 和 Tailwind CSS。
- 所有用户可见文案都通过 `react-i18next` 的 `t()` 提供翻译；新增或修改文案时同步维护 `web/default/src/i18n/locales/`。
- 功能页面放在 `web/default/src/features/<feature>/`，通用组件和工具分别放在 `src/components/`、`src/lib/`。
- 使用项目已有的 UI 组件和图标，保持键盘操作、表单校验、错误提示和响应式布局的一致性。

### 分支与提交

为每个功能或修复创建独立分支，提交前只加入本次任务涉及的文件：

```bash
git switch -c feature/<short-name>
git status --short
git diff --check
git add <files>
git commit -m "feat: <short description>"
```

提交和 Pull Request 应说明变更目的、验证方式和可能影响；请使用仓库提供的 [PR 模板](./.github/PULL_REQUEST_TEMPLATE.md)。

## 验证清单

后端改动至少运行：

```bash
go test ./...
```

前端改动至少运行：

```bash
cd web/default
bun run typecheck
bun run lint
bun run build
```

前后端一起改动时，再运行：

```bash
bun run build:check
```

提交前还应检查前端格式：

```bash
bun run format:check
```

完成后检查 `git diff --check`，确认 `.env`、数据库文件、日志和构建产物没有进入提交。涉及数据库迁移、计费、认证、渠道协议或多节点行为时，应补充针对真实业务契约的回归测试。

## 构建

构建包含前端资源的本机二进制：

```bash
make build-web
go build -o new-api .
```

构建完整容器镜像：

```bash
docker build -t new-api:local .
```

生产发布应使用经过审查的 commit SHA、版本标签或镜像 digest，并在上线前完成数据库备份、健康检查和最小业务请求验证。详细流程见 [开发、构建与部署](./docs/development-deployment.md)。

## 文档导航

- [开发、构建与部署](./docs/development-deployment.md)：本地开发、Compose、服务器上线、验证和回滚。
- [环境变量参考](./.env.example)：配置项名称和用途；真实环境配置不应提交。
- [官方文档](https://docs.newapi.pro/zh/docs)：安装、API、功能和常见问题。
- [管理 API OpenAPI 描述](./docs/openapi/api.json)。
- [模型中转接口 OpenAPI 描述](./docs/openapi/relay.json)。
- [计费表达式说明](./pkg/billingexpr/expr.md)：动态计费和额度换算规则。
- `deploy/` 与 `docs/` 下的 README 和专题文档：对应部署、压测、监控等局部场景。

## 贡献与许可证

欢迎提交 Bug 修复、功能改进、测试和文档。提交前请先搜索已有 Issue/PR，并确保变更范围清晰、没有敏感信息。

项目以 [AGPLv3](./LICENSE) 许可证发布，并基于 [One API](https://github.com/songquanpeng/one-api) 进行开发。使用本项目对外提供服务时，请自行完成上游授权、内容安全、实名、日志留存、税务和其他适用的合规义务。

问题反馈请使用 [GitHub Issues](https://github.com/QuantumNous/new-api/issues)，版本信息见 [Releases](https://github.com/QuantumNous/new-api/releases)。
