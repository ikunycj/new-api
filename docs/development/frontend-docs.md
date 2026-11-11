# 前端文档维护

线上 `/docs` 的正文直接编写为 React TSX，并在构建阶段预渲染为 HTML 和正文 JSON。本目录的 Markdown 是开发说明，不参与线上正文构建。

## 源码与注册入口

| 入口 | 职责 |
| --- | --- |
| [features/docs](../../web/default/src/features/docs/) | 正文组件与 `DocsShell`、代码块、表格、步骤和图片组件 |
| [docs-config.ts](../../web/default/src/features/docs/docs-config.ts) | 页面 ID、路由、导航分组、顺序和 HTML 输出路径 |
| [docs-page.tsx](../../web/default/src/features/docs/docs-page.tsx) | 开发环境按路由加载组件，生产环境装载正文 |
| [docs-prerender-sources.ts](../../web/default/src/features/docs/docs-prerender-sources.ts) | 构建时组件映射及路由完整性类型约束 |
| [docs-loader.ts](../../web/default/src/features/docs/docs-loader.ts) | 正文 manifest、按需获取与缓存更新 |
| [build-prerender.ts](../../web/default/scripts/build-prerender.ts) | 首页与文档静态产物构建 |
| [web-router.go](../../router/web-router.go) | Go 侧公开路由、预渲染 HTML 和 bootstrap 输出 |

文档路由使用 `DOCS_LOCALE = 'zhCN'`，首页预渲染语言由构建脚本的 `LANGUAGES` 管理。文档组件中的文案维护方式遵守 [前端约定](../../web/default/AGENTS.md) 和当前组件实现。

设文档路由数为 `D = DOCS_ROUTES.length`，首页语言数为 `L = LANGUAGES.length`：构建生成 `L + D` 个预渲染 HTML、`D` 个中文正文 JSON 和一个 manifest。不要在说明或验收脚本中手工维护历史页数。

## 访问与缓存

1. 直接访问 `/docs` 或其子页时，Go 根据 manifest 返回中文预渲染 HTML，浏览器可以先显示正文。
2. React 绑定导航、复制等交互。站内跳转按需取得当前页面正文。
3. loader 使用文档语言、路由和内容哈希管理 IndexedDB 缓存；后台校验 manifest 后才更新变化的正文。

文档正文、JSON 和图片按需装载；它们不应作为首页首屏的整体依赖。

## 新增或修改文档

1. 在 `features/docs/` 新建或修改命名导出的正文组件，复用现有文档组件。
2. 新页在 `DOCS_ROUTES` 登记，并同步补齐 `docs-page.tsx` 的开发 loader 和 `docs-prerender-sources.ts` 的构建组件表。
3. 新增对应的 TanStack Router 路由文件；Go 的文档路由来自构建后 manifest。
4. 修改图片时同步更新 TSX 中的尺寸、替代文本和带哈希的资源文件名。
5. 按改动运行目标文件格式与 lint、类型检查、`bun run build`；涉及 Go 路由时运行对应路由测试。构建中的 `build:budget` 会核对首屏预算及文档产物一致性。

## 图片与源素材

- 发布图片位于 [public/static/image/docs](../../web/default/public/static/image/docs/)，正文引用响应式 WebP。不要把图片 Base64 写入正文 JSON。
- 图片应给出宽高，并按页面需要使用懒加载和异步解码。
- CC Switch 的 11 张原始截图与发布图映射保存在 [素材索引](../assets/README.md)，更新或重新压缩时从源图生成，源 PNG 不参与发布构建。
- 原始 Markdown 迁移稿已被 TSX 正文替代，不再双份维护。历史正文可从 Git 查询；独立的钱包与支付说明保留在 [billing](../billing/wallet-and-payments.md)。

首屏方案、旧测量基线和当时的验收结果见 [历史性能设计](../archive/design/frontend-ssr-first-screen-architecture-design.md)。这些记录不代表当前生产性能。
