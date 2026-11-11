# OpenAPI 描述

| 文件 | 内容 |
| --- | --- |
| [api.json](api.json) | 管理与用户控制台 API |
| [relay.json](relay.json) | 模型中转 API |

这些文件是仓库中的静态描述，不保证覆盖当前分支的所有路由和字段。排查接口行为时，核对 [路由](../../router/)、对应 controller 和 DTO；不要仅凭描述文件判断接口已实现或权限已生效。

公开 API 的页面正文见 [前端 API 文档](../../web/default/src/features/docs/ai-model-api.tsx)，页面注册见 [docs-config.ts](../../web/default/src/features/docs/docs-config.ts)。更新接口时应同时核对相关描述与用户文档，避免继续维护指向不存在页面的 Markdown 接口索引。
