# 前端文档 TSX 与预渲染方案

## 目标

- 文档正文只维护简体中文，不生成英文、繁体中文、法语、日语、俄语或越南语版本。
- 任意界面语言访问 `/docs` 时都返回简体中文文档。
- 文档继续直接编写为 React TSX，并在前端构建阶段转换为静态 HTML。
- 首页继续按现有七种界面语言预渲染，文档不进入首页正文或首屏资源。

## 源码边界

文档组件位于 `web/default/src/features/docs/`。每篇文档是一个命名导出的 React 组件，并使用现有的 `DocsShell`、`CodeBlock`、`NumberedSteps`、`DocsTable` 等组件组织内容。

文档内容以 TSX 为发布源。`docs/reference/relay-station/` 保存迁移前的 24 篇 Markdown 和 11 张原始截图，只作为内容核对与图片再处理的参考，不参与生产构建，避免参考稿被误认为线上内容。源文件中空白的推荐奖励页，以当前系统返佣功能为准单独编写 TSX。

`web/default/src/features/docs/docs-config.ts` 是文档路由、导航分组、显示顺序和预渲染输出路径的统一注册表。开发环境 loader 与预渲染组件表分别负责按需加载和静态渲染，不通过统一 barrel 导出正文。

## 构建产物

`web/default/scripts/build-prerender.ts` 保留七种首页语言，但只用 `zhCN` 渲染文档路由。当前二十四篇文档对应：

- 七个首页 HTML，每种界面语言一个；
- 二十四个中文文档 HTML，每条文档路由一个；
- 二十四个带内容哈希的中文文档 JSON；
- 一个文档 manifest。
-

因此当前预渲染 HTML 是 `7 + 24 = 31` 个，而不是按七种语言复制整套文档的 `7 × 25 = 175` 个。新增文档只增加一个中文 HTML 和一个中文 JSON。

## 运行流程

### 直接访问文档

1. Go 从文档 manifest 获取公开文档路由，并将所有 `/docs` 请求固定映射到 `zhCN` 预渲染文件。
2. 浏览器收到已经包含正文的 HTML，可以先显示内容。
3. React 使用 `hydrateRoot` 绑定导航、复制按钮等交互。
4. 客户端空闲时读取 manifest；只有正文哈希变化才获取新的中文 JSON。

### 站内跳转文档

1. TanStack Router 按需加载目标文档路由 chunk。
2. `DocsPage` 使用固定的 `zhCN:<route>` IndexedDB 缓存键。
3. 有缓存时直接显示；没有缓存时只请求 `zhCN` manifest 条目和当前正文 JSON。

### 访问首页或其他路由

文档 TSX 由文档路由按需加载，文档 HTML、JSON 和正文图片不会作为首页请求。预渲染文件数量只影响构建时间、发布包大小和 Go 启动时读取的静态 HTML，不会使浏览器一次加载全部文档。

## 新增文档步骤

1. 在 `web/default/src/features/docs/` 新建命名导出的文档组件并复用现有文档组件。
2. 在 `docs-config.ts` 的 `DOCS_ROUTES` 中登记 ID、分组、名称、URL 和输出文件；侧栏、前后页和构建路由会从这里派生。
3. 在 `docs-page.tsx` 的开发环境动态 loader 中登记组件，保持每篇正文按路由拆包。
4. 在 `docs-prerender-sources.ts` 中登记组件；`satisfies Record<DocsRoutePath, ComponentType>` 会在编译期检查是否缺页。
5. 新增 TanStack Router 路由文件。Go 路由会从构建后的 manifest 获取文档 URL，不再逐条维护后端路由清单。
6. 运行格式化、目标文件 lint、前端类型检查、生产构建和 Go 路由测试，确认只生成一套中文文档。

## 图片约定

- 原始截图放在 `docs/reference/relay-station/assets/<article>/`，不要放入 `web/default/public` 后原样发布。
- 生产页面只引用带内容哈希的响应式 WebP。当前 CC Switch 的 11 张原始 PNG 分别生成 760px 和 1520px 两种宽度，共 22 个 WebP，位于 `web/default/public/static/image/docs/cc-switch/`。
- 正文图片提供明确的宽高，并使用 `loading="lazy"` 和 `decoding="async"`。
- 不把截图编码成 Base64，不把大图放进文档 JSON 或首页 JavaScript。
- 原始 PNG 不进入 `dist`；更新截图时必须重新生成两个尺寸并同步更新 TSX 中的内容哈希文件名。
- 图片需要独立构建或上传 OSS/CDN 时，使用稳定的自有资源域名和长期 immutable 缓存。

## 验证标准

- `dist/prerender` 只包含七个首页 HTML 和一套 `zhCN` 文档 HTML。
- `dist/static/docs` 只包含 `zhCN` 正文 JSON 和 manifest。
- manifest 只登记 `zhCN` 文档路由。
- 当前构建应包含七个首页 HTML、二十四个文档 HTML、二十四个文档 JSON 和二十二张 CC Switch WebP，且不包含参考目录中的原始 PNG。
- 首屏预算脚本会拒绝非 `zhCN` 文档产物，以及 HTML、JSON、manifest 路由数量不一致的构建。
- 使用任意语言 Cookie 或 `Accept-Language` 直接访问 `/docs`，响应正文和 bootstrap locale 都是 `zhCN`。
- `/` 的生产首屏预算继续通过。
