# 首页与文档页首屏性能优化方案

## 1. 结论

本项目继续采用以下边界：

```text
/             构建时 SSR/SSG
/docs         构建时 SSR/SSG
/docs/*       构建时 SSR/SSG
其它路由       保持现有 SPA
```

不迁移 Next.js，不增加生产 Node SSR 服务，不让 SSR 等待业务 API。构建阶段使用 React 生成 HTML，生产阶段仍由 Go 返回嵌入式静态产物。

但 SSR 不能单独解决当前的十多秒首屏。2026-08-02 的线上复测表明，当前主要问题是：

1. 空 HTML 必须等约 1.59 MB gzip 的同步 JavaScript 才能开始渲染；
2. 入口脚本之后还要经过 setup 检查、首页异步大块和首页 API；
3. 首页异步大块包含约 1.00 MB gzip 的全量 LobeHub 图标；
4. 七个语言包被静态打进入口，源文件合计约 977 KB gzip；
5. 缺失的 hash 静态文件被错误回退成 `200 text/html`；
6. Hero 动画和 Logo 分别约 717 KB、144 KB，且直到页面逻辑完成后才开始下载。

因此，最终方案不是“只加 SSR”，而是按收益与改动比执行四组改造：

```text
静态资源正确性
  -> 消除首屏 JS/API 瀑布
  -> 首页和文档页构建时 SSR/SSG
  -> 长缓存、发布兼容和真实用户监控
```

其中前两组改动很小，却决定 SSR 后能否同时获得较快的可见时间和可交互时间。

### 1.1 SSR 后的 API 边界

`/` 使用 SSR 后，目标是“首屏可见内容不依赖任何 API”，不是“页面生命周期内不再请求 API”。各请求的处理如下：

| 数据 | SSR/首屏处理 | hydration 后处理 |
| --- | --- | --- |
| setup | Go 路由直接读取进程状态，未完成时重定向 `/setup` | `/`、`/docs/*` 不再请求 setup API |
| 站点名、Logo、服务地址、导航模块 | Go 注入小型公开 bootstrap | `status` 最多后台刷新一次，不阻塞显示 |
| 默认/自定义首页模式 | Go 直接判断；默认首页输出 SSR，自定义首页保持 SPA | 默认首页不请求 `home_page_content` |
| pricing/catalog | 不进入 SSR，不影响 Hero | 价格区接近视口后才请求 |
| notice/announcements | 不进入 SSR | 浏览器空闲或用户打开通知时请求 |
| 登录态、用户、钱包 | 不注入公开 HTML | hydration 后按现有会话更新 |
| 文档正文 | 直达时已经包含在 SSR HTML | 写入 IndexedDB，空闲时只校验 manifest 文件名 |

因此，SSR 消除的是 `HTML -> JS -> setup -> route chunk -> API -> 首次绘制` 这条阻塞链。JavaScript 体积、hydration、语言包、图标、大图、缓存错误和发布期旧 chunk 仍必须分别治理。

## 2. 线上复测基线

### 2.1 测试说明

测试时间：2026-08-02。

测试方式：

- `curl` 冷请求，检查 TTFB、压缩、体积、缓存头和错误路由；
- Microsoft Edge 无头浏览器，禁用缓存；
- 移动端视口 `390 x 844`；
- 4 倍 CPU 降速；
- 网络延迟 150 ms，下载 1.6 Mbps，上传 750 Kbps；
- 使用 Performance API 记录 FCP、LCP、CLS、长任务和请求时序。

这是可复现的实验室基线，不等同于真实用户 p75。上线后必须再用 RUM 验证。

线上 `/api/status` 报告的部署提交为 `a906d4d3020ccba5ffeb919eb67716b404ebbbce`，当前本地 HEAD 为 `49b109f7`，两者的构建 hash 不同。线上与本地产物中对应的首页图标块解压体积均约 5.31 MB，说明主要结论仍适用，但每次发布后都必须重新记录指标，不能直接沿用本次数字。

### 2.2 alltokenapi.com 指标

| 项目 | 结果 |
| --- | ---: |
| 根 HTML 原始体积 | 1,178 B |
| 根 HTML gzip 传输 | 556 B |
| 五次冷请求 TTFB 中位数 | 约 0.74 s |
| HTML 可见正文 | 0，只有空 `#root` |
| 首批同步 JS | 约 1.59 MB gzip |
| 移动端首屏全部脚本 | 约 2.69 MB 传输 / 11.04 MB 解压 |
| 移动端全部资源 | 约 2.97 MB 传输 / 11.71 MB 解压 |
| 移动端 FCP | 15.75 s |
| 移动端 LCP | 16.74 s |
| 移动端 CLS | 0.0037 |
| 移动端长任务 | 4 个，共 1.37 s，最长 610 ms |
| 桌面冷缓存 FCP | 2.53 s |
| 桌面冷缓存 LCP | 3.13 s |

CLS 已经合格，当前不需要为了布局稳定性重写页面。优化重点应放在“内容何时出现”和“为出现内容下载了多少代码”。

### 2.3 实际请求瀑布

移动端实验中，关键请求顺序如下：

```text
0.00 s  请求 HTML
0.55 s  开始下载入口 JS/CSS
8.80 s  最大入口 JS 下载完成
9.42 s  DOMContentLoaded，开始 /api/status
9.49 s  开始 /api/setup
9.66 s  setup 完成，开始首页路由异步块
15.15 s 首页异步大块完成
15.69 s 同时发起 status、notice、home_page_content
15.75 s FCP
16.74 s LCP
17.31 s Logo 下载完成
```

最大的两个脚本是：

| 资源 | gzip/传输 | 解压 | 作用 |
| --- | ---: | ---: | --- |
| `static/js/index.*.js` | 约 1.13 MB | 约 3.93 MB | 入口、语言包和共享代码 |
| 首页异步大块 | 约 1.00 MB | 约 5.31 MB | 主要包含全量 `@lobehub/icons` |

API 响应本身体积很小，并不是带宽主因；问题在于它们位于渲染阻塞链上，并且 `/api/status` 被重复请求三次。

### 2.4 静态资源和缓存问题

当前线上行为：

```http
/static/js/index.<hash>.js
Cache-Control: max-age=604800

/static/js/not-exist.<hash>.js
HTTP/1.1 200 OK
Content-Type: text/html
```

这带来两个问题：

- 带 hash 的不可变资源只缓存一周，浪费重复访问收益；
- 旧 HTML 引用已经不存在的 chunk 时，浏览器收到 HTML 而不是真实 404，最终表现为脚本解析错误、ChunkLoadError 或白屏。

这不是 SSR 问题，必须独立修复。

## 3. 与 TokenHub 的对比

同日检查 `https://tokenhub.com/zh`：

- HTML 约 252 KB，收到 HTML 时已有完整 Hero 和正文；
- HTML 使用 `no-store`，带 hash 静态资源长期缓存；
- 关键字体、Logo 和 CSS 使用 preload；
- JavaScript 分为多个异步 chunk，非首屏图片延迟加载；
- 同一移动端实验的 FCP/LCP 约为 9.50/9.94 秒。

TokenHub 也不是一个轻量页面：其 HTML、CSS、字体和主线程开销仍然较大。因此本项目只采用它的有效原则，不复制它的具体体积和依赖结构：

| 采用 | 不采用 |
| --- | --- |
| HTML 先包含可读 Header、Hero、CTA | 250 KB 级别的大 HTML |
| JavaScript 后接管交互 | 为首屏预加载所有字体和脚本 |
| 非首屏图片和模块延迟加载 | 大量首屏 chunk 并发竞争带宽 |
| hash 资源 immutable 缓存 | 请求时 Node SSR |
| JS 未执行时页面仍可读和可导航 | 把动态价格和登录态放进 SSR |

当前站在相同实验下比 TokenHub 的 LCP 慢约 6.8 秒，差值基本对应当前的入口 JS、setup、首页图标块和 API 瀑布。

## 4. 哪些问题来自上游，哪些来自当前定制

对比当前分支、`upstream/main` 和共同基线后，可以按以下方式归因。

### 4.1 上游原有问题

- `index.html` 只有空 `#root`，所有可见内容依赖 React；
- `router/web-router.go` 对缺失静态文件使用 SPA fallback；
- `routes/__root.tsx` 首次访问会等待 `/api/setup`；
- `features/home/index.tsx` 等待 `/api/home_page_content` 后才显示默认首页；
- `i18n/config.ts` 静态导入七个完整语言包；
- `lib/lobe-icon.tsx` 使用 `import * as LobeIcons`；
- `main.tsx`、`useSystemConfig` 和 `useStatus` 存在重复 status 请求来源；
- 上游默认首页同样静态导入所有首页区块，只是区块比当前定制页轻。

### 4.2 当前定制放大的问题

- 默认首页由原来的五个较轻区块扩展为九个营销和预览区块，并全部静态导入；
- 新增 726 KB Ribbon 动画、16 KB CC Switch Logo 和约 76 KB 的界面图；
- 公共 Logo 从上游约 9.6 KB 增至 144 KB；
- Provider Marquee、Featured Models、Pricing Preview 都进入 `getLobeIcon` 路径，使首页加载全量图标包；
- `useHomeCatalog()` 先等待 status，再决定是否请求 pricing，形成 status -> pricing 的串行依赖；
- 文档正文新增为约 104 KB 的 TSX 源文件，仍跟随 React chunk 发布；
- 首页组件数量和定制翻译增加了语言包体积，但“七个语言包全部首屏加载”是上游结构问题。

### 4.3 不应错误归因的部分

- API 响应只有几十字节到约 1 KB，不能把十多秒主要归因于后端查询；
- 浏览器实际协商到了 HTTP/2，当前不是因为只使用 HTTP/1.1；
- CLS 已经很低，无需通过大规模布局重写解决首屏慢；
- SSR 不会自动删除 JavaScript、动画、语言包或 API；它只让 HTML 可以更早显示。

## 5. 优化原则

1. 默认首页的 Header、Hero 和 CTA 不等待 JavaScript或 API。
2. 首页与文档的 SSR 只使用构建时稳定数据，不请求 pricing、status、notice 或用户接口。
3. 先删除瀑布和整包导入，再考虑微小的 React 重渲染优化。
4. SSR 首次客户端树必须与服务端 HTML 一致，登录态在 hydration 后更新。
5. 文档版本只比较文件名 hash，不在浏览器重新计算 hash，不依赖 ETag。
6. hash 资源永久不可变；HTML、manifest 和运行时 bootstrap 可重新验证。
7. 继续保留 Go 单体、TanStack Router、React Query 和现有 SPA 业务页面。

## 6. 目标架构

```mermaid
flowchart TD
    Request[浏览器请求] --> GoRouter[Go Web Router]
    GoRouter -->|/| HomeSSG[对应语言的 home.html]
    GoRouter -->|/docs 或 /docs/*| DocsSSG[对应语言和文档的 HTML]
    GoRouter -->|其它前端路由| SPA[现有 index.html]

    HomeSSG --> HomeHydration[Hydrate Header 和 Hero]
    HomeHydration --> DeferredHome[按可见性加载非首屏区块]
    DocsSSG --> DocsHydration[Hydrate DocsShell]
    DocsHydration --> IndexedDB[后台校验文件名并写入 IndexedDB]
    SPA --> CreateRoot[现有 createRoot]
```

推荐实现方式仍是构建时 SSR/SSG：

```text
构建阶段：Bun/Node + React renderToString 生成 HTML
生产阶段：Go embed.FS 读取并返回 HTML
浏览器：SSR 根节点 hydrateRoot，SPA 空根节点 createRoot
```

当前 `main.tsx` 在根节点非空时直接不执行任何渲染。落地 SSR 时必须改成显式 `hydrateRoot`，不能只把 HTML 填入 `#root`。

## 7. 第一优先级：小改动直接消除十秒链路

这些改动应先于或与 SSR 同一批完成。

### 7.1 缺失静态资源返回真实 404

调整 `router/web-router.go`：

```text
/static/*、/assets/* 请求未命中 embed.FS -> 404
/api/*、/v1/* 未命中 -> 现有 RelayNotFound
其它应用路由未命中 -> SPA index.html
```

验收：

```text
/static/js/not-exist.123.js -> 404，且不是 text/html
/dashboard/not-a-real-child -> 仍返回 SPA HTML
```

这是低风险修复，也能让客户端正确识别版本失配并执行一次受控刷新。

### 7.2 setup 不再阻塞公共 SSR 路由

`constant.Setup` 在 Go 进程中已经可用。对 `/` 和 `/docs/*`：

- 未初始化时由 Go 直接重定向到 `/setup`；
- 已初始化时直接返回 SSR HTML；
- 客户端 `beforeLoad` 对已有服务端 setup 标记的路由不再请求 `/api/setup`；
- 其它 SPA 路由暂时保留现有检查逻辑，控制改动范围。

这比 SSR 后继续发 setup API 更直接，也不会削弱未初始化实例的保护。

### 7.3 `/api/status` 收敛为一个数据源

当前网络记录出现三次 `/api/status`：

- `main.tsx` 为标题和 favicon 请求一次；
- `useSystemConfig({ autoLoad: true })` 请求一次；
- `useHomeCatalog()` 内的 `useStatus()` 再请求一次。

调整为：

- `main.tsx` 只应用本地缓存，不主动发网络请求；
- 根组件统一使用 React Query 的 `['status']`；
- `useSystemConfig` 和 `useHomeCatalog` 读取同一 query/store；
- SSR bootstrap 提供 system name、Logo、公开模块开关时，首次渲染直接使用 bootstrap；
- 后台最多刷新一次 status。

不得把整份 status JSON重复注入多个位置。

### 7.4 默认首页不等待 `home_page_content`

Go 已经持有 `common.OptionMap["HomePageContent"]`。返回首页时只注入小型公开 bootstrap：

```ts
interface HomeBootstrap {
  locale: string
  systemName: string
  logo: string
  setup: boolean
  homeMode: 'default' | 'custom-html' | 'custom-markdown' | 'custom-url'
  homeRevision?: string
  pricingEnabled: boolean
  pricingRequireAuth: boolean
}
```

其中 `homeRevision` 是服务端内容版本，不包含内容本身。

行为：

- `default`：直接使用 SSR 默认首页，不请求 `/api/home_page_content`；
- `custom-*` 且本地 revision 命中：立即显示本地缓存，后台校验；
- `custom-*` 且无缓存：显示稳定的自定义内容容器，再异步下载，不先闪现默认首页；
- 管理员从自定义首页切回默认首页时，bootstrap 立即决定默认模式，不受旧缓存影响。

现有 DOMPurify、Markdown 处理和 iframe sandbox 保持不变。

### 7.5 公共路由专用语言包与完整词典补载

`i18n/config.ts` 不再静态导入七个完整 JSON，而是维护公共词典和完整词典两组 loader：

```ts
const publicLocaleLoaders = {
  en: () => import('./public-locales/en.json'),
  zhCN: () => import('./public-locales/zhCN.json'),
  // 其余语言同理
}

const fullLocaleLoaders = {
  en: () => import('./locales/en.json'),
  zhCN: () => import('./locales/zh.json'),
  // 其余语言同理
}
```

`scripts/build-public-locales.ts` 在构建前扫描首页、文档、认证入口和公共布局源码，从完整词典的 6022 个键中生成 966 个公共键，并验证七种语言都存在对应翻译。`/` 和 `/docs*` 首次只加载当前语言的公共词典；进入其它路由前，根路由通过 `ensureFullLocale()` 补载并覆盖为当前语言的完整词典。

公共词典最大约 31.5 KB gzip，中文约 26.9 KB gzip；原中文完整词典约 140.3 KB gzip。由于翻译键本身就是英文源文案，并且构建时会校验多语言键完整性，`fallbackLng` 设为 `false`，避免非英文用户首次访问时再并发下载一份英文 fallback。首页跳转 `/sign-in` 的浏览器验收已确认会补载完整中文词典，页面文案正常。

SSR 语言选择顺序：

```text
语言 cookie -> Accept-Language -> 系统默认语言
```

客户端语言选择继续保留 localStorage，同时同步一个不含敏感信息的语言 cookie，保证 SSR 与 hydration 首次语言一致。

### 7.6 首页不再加载全量 LobeHub 图标

不需要立刻重写全站动态图标系统，只切断首页的导入链即可：

- Provider Marquee 的四个固定图标使用直接 import 或本地静态图；
- Featured Models 和 Pricing Preview 首屏使用供应商静态映射或文字 fallback；
- 用户滚动到动态模型区域后，才允许加载更完整的图标能力；
- `getLobeIcon` 继续服务 Dashboard、Models、Channels 等既有页面。

这项小范围修改预计可避免首页下载约 998 KB gzip / 5.31 MB 解压的图标块，是当前收益最高的单项前端改动。

### 7.7 首页非首屏区块按可见性加载

当前 `DefaultHome` 静态导入所有九个区块。调整为：

```text
首屏立即提供：PublicHeader、LandingHero、Provider 文本
接近视口加载：AI Clients、Featured Models、Capabilities
空闲或接近视口加载：Console、Gateway、Pricing、FAQ、CTA、Footer 增强逻辑
```

不能只使用 `React.lazy` 后立即渲染，因为这仍会马上请求所有 chunk。需要一个共享的 `DeferUntilVisible` 边界，在距离视口约 600-1000 px 时才挂载 lazy component，并为每个区块预留稳定高度。

SSR 只输出 Header、Hero、CTA 所需的稳定首屏 HTML和轻量的后续区块占位。搜索引擎需要的介绍文本可以保留为静态 HTML，但复杂预览和价格组件不进入首屏 hydration 树。

### 7.8 Hero 使用静态首帧

当前 Ribbon 动画约 726 KB。调整为：

- 从动画生成 30-80 KB 的 AVIF/WebP 静态首帧；
- SSR 和 LCP 只使用静态首帧，设置明确宽高；
- 动画在 hydration 后、浏览器空闲且 Hero 仍可见时加载；
- `prefers-reduced-motion`、Save-Data、慢网和移动端默认不升级动画；
- 不 preload 完整动画。

Logo 提供与实际展示尺寸匹配的小图，不让 662 x 662、144 KB 的 PNG 作为导航 Logo 下载。favicon 与页面 Logo 分开产物。

## 8. 首页 SSR/SSG 设计

### 8.1 SSR 输出范围

首页 HTML 必须包含：

- 当前语言的 `<html lang>`；
- title、description、canonical 和 Open Graph；
- 主题初始化短脚本；
- PublicHeader 的 Logo、站点名和主要导航；
- LandingHero 的标题、说明、两个 CTA；
- Ribbon 静态首帧；
- Provider 可读文本；
- 下一屏的内容提示或稳定占位。

不包含：

- 用户信息、钱包、token 或 session；
- 实时 pricing/catalog；
- notice 和后台通知；
- 请求时读取数据库的动态页面内容；
- 完整动画和非首屏图片。

### 8.2 组件复用

以最小改动拆成：

```text
HomeSsrView
  纯 props，服务端和首次客户端共同渲染

HomeClientContainer
  登录态、status 刷新、自定义首页和动态 catalog

HomeDeferredSections
  接近视口后按块加载
```

LandingHero 和 PublicHeader 优先改造成“纯视图 + 客户端增强”，不另建一套完全独立的 HTML 模板。

SSR 和首次 hydration 一律按未登录状态输出稳定 CTA。hydration 完成后，如果本地认证状态有效，再通过普通 state 更新成 Dashboard CTA。不得在首次 render 中直接读取 localStorage 改变 SSR 树。

客户端发现 `data-prerendered="true"` 时，必须先 `await router.load()`，等 TanStack Router 得到与当前 URL 对应的 route match 后再执行 `hydrateRoot`；否则客户端会用尚未加载的路由树核对服务端 HTML。构建时的预渲染 router 必须设置 `isServer: true`，根 `Outlet` 在服务端和客户端首次 render 中也必须保留一致的 `Suspense` 边界。

登录态、运行时导航模块、`NavigationProgress` 和 `Toaster` 都属于 hydration 后增强：服务端和客户端首次 render 不输出依赖本地会话的差异内容，hydration 完成后再挂载或更新。开发环境 Devtools 不作为生产 hydration 树的一部分。

路由生成器会让所有 route 文件参与客户端入口图，因此 route 顶层依赖也属于首屏预算。`beforeLoad` 所需的 section ID、默认 section 和白名单必须放在不含 JSX 的轻量配置文件中，不能从后台设置页的 `section-registry.tsx` 读取。公共首页也必须直接导入 `public-layout`，不能通过同时导出认证后台布局的 layout barrel。否则即使页面组件本身是 lazy，KaTeX、React Hook Form、Zod 和后台设置 UI 仍可能被共享 chunk 拉回首页。

### 8.3 API 时序

默认首页目标时序：

```text
HTML + CSS + Hero 静态图 -> 首屏可见
hydration -> 最多一次 status 后台刷新
接近价格区块 -> 按模块权限请求 pricing
打开通知或浏览器空闲 -> 请求 notice
```

默认首页不请求 `home_page_content`，不请求 setup，也不因 pricing 失败隐藏 Hero。

`useHomeCatalog()` 不再等待 status 网络请求后才开始所有工作。它先读取 SSR/bootstrap 或本地缓存中的公开模块开关；只有价格区块接近视口且允许访问时才请求 pricing。

## 9. 文档页 SSR 与长期本地缓存

### 9.1 内容产物

为减少迁移范围，现阶段保留已有文档 TSX 作为构建时内容源，不先迁移 Markdown。构建脚本在 Bun 中使用现有组件和 i18n 渲染每个语言版本，提取 `DocsShell` 内部 HTML，再生成 SSR 页面和 hash payload。浏览器路由只打包通用 `DocsPage`，不再打包每篇 TSX 正文。

构建时生成：

```text
dist/prerender/{locale}/docs/{route}.html
dist/static/docs/manifest.json
dist/static/docs/{locale}/{page-id}.{hash}.json
```

payload 示例：

```json
{
  "html": "<div>导航、标题、正文、目录和翻页内容</div>"
}
```

payload 只来自仓库内可信 TSX 和翻译文件，不接受运行时用户 HTML。复制代码、文档内 SPA 导航和移动端文档选择器由通用客户端壳通过事件委托增强。

### 9.2 文件名 hash

```text
/static/docs/zh/codex.90ef12ab34cd.json
/static/docs/en/introduction.33cd55ef77ab.json
```

hash 输入包含：

- 当前语言最终渲染出的完整文档内容；
- 标题、说明、导航、目录和代码示例；
- 会改变 payload JSON 的构建逻辑。

使用 SHA-256 前 12 或 16 个十六进制字符。浏览器只比较文件名，不下载后重新计算内容 hash，不比较更新时间和 ETag。

### 9.3 manifest

```json
{
  "version": 1,
  "locales": {
    "zhCN": {
      "/docs": "introduction.12ab34cd56ef.json",
      "/docs/tools/codex": "codex.90ef12ab34cd.json"
    }
  }
}
```

缓存规则：

```http
/static/docs/manifest.json
Cache-Control: no-cache, must-revalidate

/static/docs/{locale}/{page-id}.{hash}.json
Cache-Control: public, max-age=31536000, immutable
```

### 9.4 IndexedDB

正文长期存放在 IndexedDB，不使用 localStorage：

```ts
interface CachedDoc {
  key: `${locale}:${route}`
  fileName: string
  html: string
  savedAt: number
}
```

不设置时间过期。只有 manifest 或 SSR 标记中的文件名变化时才替换。

### 9.5 首次访问和 SPA 导航

直接访问文档：

```text
Go 返回包含完整正文的 SSR HTML
  -> 页面立即可读
  -> hydration 从现有 DOM 读取正文和 bootstrap fileName
  -> 将正文模板写入 IndexedDB
  -> 空闲时重新验证 manifest
```

如果 IndexedDB 的 `fileName` 与 SSR 标记相同，不再次下载正文 JSON。

SPA 文档导航：

```text
先读取 IndexedDB 中该语言和路由的旧缓存并立即显示
  -> 后台读取 manifest 当前 fileName
  -> IndexedDB 命中同名文件：不下载正文
  -> fileName 不同：后台请求新 hash 文件
  -> 成功：原子替换页面和缓存
  -> 失败：保留旧文档，显示轻量重试状态
```

这里的“文档后台异步加载”不意味着直接访问文档先显示空白：直接访问有 SSR 正文；只有 SPA 导航到未缓存文档时才显示稳定 skeleton。

## 10. Go 路由和构建产物

### 10.1 构建产物

```text
dist/index.html                              现有 SPA
dist/prerender/{locale}/home.html           首页
dist/prerender/{locale}/docs/{route}.html   文档
dist/static/docs/manifest.json
dist/static/docs/{locale}/{page-id}.{hash}.json
```

当前实现的核心文件：

```text
web/default/src/features/docs/docs-cache.ts
web/default/src/features/docs/docs-page.tsx
web/default/src/entry-prerender.tsx
web/default/scripts/build-prerender.ts
```

### 10.2 Go 返回逻辑

`ThemeAssets` 保持现有 `fs.FS` 接口。`SetWebRouter` 启动时从嵌入文件系统读取预渲染模板和文档 manifest，不增加生产 Node 服务，也不在每个请求中读取磁盘：

```go
type ThemeAssets struct {
    DefaultBuildFS   fs.FS
    DefaultIndexPage []byte
}
```

路由顺序：

1. API 和 Relay 保持现有处理；
2. `/static/*` 命中 embed.FS，否则真实 404；
3. 未 setup 的 `/` 和 `/docs/*` 重定向 `/setup`；
4. `/` 按语言返回 home HTML；
5. `/docs/*` 按语言和 page ID 返回 docs HTML；
6. 其它前端路由返回现有 SPA index；
7. 未纳入预渲染映射的其它应用路由继续使用 SPA fallback。

Go 只替换经过 JSON 安全编码的小型公开 bootstrap 占位符，不把用户数据写入 HTML。

## 11. 缓存、压缩与发布兼容

### 11.1 缓存头

```http
/ 和 /docs/* HTML:
  Cache-Control: no-cache, must-revalidate

/static/docs/manifest.json:
  Cache-Control: no-cache, must-revalidate

/static/**/*.<hash>.*:
  Cache-Control: public, max-age=31536000, immutable

/logo.png、/favicon.ico 等非 hash 资源:
  Cache-Control: no-cache, must-revalidate
```

不要继续对所有非根路径统一设置 `max-age=604800`。缓存策略必须根据“HTML、manifest、hash 资源、非 hash 资源”分类。

### 11.2 发布期旧 chunk

真实 404 能让错误可诊断，但不能代替发布兼容。当前单二进制替换后，旧 hash 资源会立刻消失。

推荐发布包额外包含 `dist/static`，在服务器维护一个内容寻址的静态资源目录：

```text
/opt/new-api/web-static/
  当前发布 hash 文件
  上一发布 hash 文件
```

发布顺序：

```text
先合并上传新 static 文件
  -> 验证新 hash 文件可访问
  -> 再切换返回新 HTML 的 Go 二进制
  -> 至少保留上一版本或保留 7 天
```

hash 文件名不会冲突，可以增量合并。OpenResty 优先从该目录返回 `/static/`，未命中再返回真实 404，不回退 SPA。

客户端已通过 `src/lib/chunk-load-recovery.ts` 增加受控恢复：识别 ChunkLoadError、动态 import 失败和静态脚本加载失败，并按当前入口 build hash 在 `sessionStorage` 中限制为每个版本最多刷新一次。它只是兜底，不能替代上一版本静态资源保留，也不能通过无限 reload 掩盖缺失资源。

### 11.3 压缩和网络

当前资源已有 gzip。可在构建时额外生成 `.br` 和 `.gz`，在确认 OpenResty 模块支持后使用 `brotli_static`/`gzip_static`，减少 Go 动态压缩 CPU 和大脚本传输。

本次从当前工作站到源站的冷请求 TTFB 中位数约 0.74 秒，TokenHub 约 0.18 秒。应用优化完成后，如果主要用户仍跨境访问洛杉矶源站，静态资源 CDN、对象存储或更近的区域节点会成为进一步降低 p75 LCP 的主要手段。CDN 只缓存 `/static` 和公开图片；`/api`、`/v1`、HTML、鉴权和用户数据必须绕过缓存。

## 12. 实施顺序与收益

同条件完整方案目标为 FCP <=4s、LCP <=5s、首屏传输 <=500KB。当前分支已经完成 P0-P3 的本地实现与生产构建，但尚未部署线上，因此线上 FCP/LCP 仍必须在明确授权发布后重新测量，不能把构建体积直接当作生产指标。

### 12.1 当前分支本地验收结果

2026-08-03 使用与线上基线相同的 1.6 Mbps 下载、750 Kbps 上传、150 ms 延迟和 4 倍 CPU 降速，对 production build 的本地 gzip HTTP fixture 进行了五次冷缓存测试，以下为中位数：

| 项目 | 桌面 | 移动端 |
| --- | ---: | ---: |
| FCP | 2.272 s | 2.028 s |
| LCP | 2.272 s | 2.028 s |
| CLS | 0.00210 | 0.00048 |
| 最大长任务 | 241 ms | 250 ms |
| LCP 前传输 | 326 KB | 298 KB |
| 10 秒总传输 | 471 KB | 471 KB |
| hydration 必需 JS gzip | 235.5 KiB | 235.5 KiB |
| Console/Page error | 0/5 | 0/5 |

LCP 元素稳定为 19.4 KB 的 Ribbon WebP 静态首帧，726 KB 动画请求为 0。默认首页与文档直达在验收窗口内的 `status`、`notice`、`pricing` 和 `home_page_content` 请求均为 0；`status` 只会在首次交互或 5 秒延迟后后台刷新。首页未滚动时只挂载首屏 section，滚动后再加载下半页。

导航 Logo 使用 56 x 56、1.67 KB 的 WebP，favicon 使用 28 x 28、688 B 的 WebP。Rsbuild 的 favicon 配置已显式指向小图，避免模板图标之外又自动注入并下载原 49.7 KB 的 `favicon.ico`。

本轮还发现并移除了多条隐藏的首屏依赖：系统设置 route 的 `beforeLoad` 原本从 JSX registry 读取常量，使初始 JS 约 640 KB gzip；拆出轻量 route 配置后降到约 312 KB。随后将首页的 `PublicLayout` 改为直接文件导入，避免 hydration 后再请求约 162 KB gzip 的认证后台共享块。公共路由词典把首次词典降到最多约 31.5 KB gzip；Base UI/Radix 的合并块改为只处理异步 chunk，约 106 KB gzip 的后台 UI primitives 不再成为首页同步入口依赖，初始同步 JS 进一步降到约 206 KB gzip。

4 倍 CPU 下的最大长任务仍为 241-250 ms，发生在约 2 秒的初始 hydration 阶段，已经满足本阶段不超过 300 ms 的验收线，但尚未达到低于 200 ms 的进阶目标。继续拆分公共 Header 的访客静态视图和语言、主题、通知、登录菜单增强，可以进一步减少主线程工作，但会扩大 hydration、首个交互和 CLS 的回归面；当前 LCP 已约 2.03 秒，因此不把这项高风险重构作为本轮上线前置条件。上线后若 RUM 的 INP 或长任务仍不合格，再根据真实设备证据实施。

构建流程已接入 `scripts/check-first-screen-budget.ts`。脚本读取 prerender HTML 和实际产物，计算 gzip 体积与首屏资源集合，任何预算超限都会令 `bun run build` 失败。


### P0：正确性和可观测性

| 改动 | 主要文件 | 收益 | 风险 |
| --- | --- | --- | --- |
| 缺失 `/static` 返回 404 | `router/web-router.go` | 消除伪 200 和白屏误诊 | 低 |
| 分类缓存头 | `middleware/cache.go` | hash 资源长期复用 | 低 |
| 构建输出 gzip 体积报告和预算 | `scripts/check-first-screen-budget.ts` | 超限直接阻断构建 | 低 |
| 记录 FCP/LCP/CLS/INP/build version | 前端性能采样 | 获得真实 p75 | 低至中 |

### P1：小改动高收益

| 改动 | 预期直接影响 |
| --- | --- |
| 公共路由 setup 改为 Go 判定 | 删除一段首屏网络屏障 |
| status 请求去重 | 3 次降为最多 1 次 |
| 默认首页不等 home content | 删除全屏 Loading 和最后一段 API 屏障 |
| 公共路由专用词典，非公共路由补载完整词典 | 公共首屏词典最大约 31.5 KB gzip，并取消英文 fallback 首次下载 |
| 首页绕开全量 Lobe icons | 避免约 1.00 MB gzip / 5.31 MB 解压块 |
| Hero 静态首帧和小 Logo | 首屏图片从约 861 KB 降到目标 100 KB 内 |
| 非首屏区块按可见性加载 | 避免首屏并发下载和执行全部营销区块 |

### P2：首页构建时 SSR/SSG

- 生成各语言 home HTML；
- Go 返回 SSR HTML 和 bootstrap；
- `main.tsx` 使用 `hydrateRoot`；
- 保证 JS 禁用时 Header、Hero 和 CTA 可读可用；
- 保持其它路由 createRoot SPA。

### P3：文档构建时 SSR/SSG 和 IndexedDB

- 保留 TSX 作为构建源，客户端正文改为 hash payload；
- 生成 docs HTML、manifest 和 hash JSON；
- 实现文件名校验、IndexedDB 和失败回退；
- 保持 DocsShell 和 SPA 文档导航。

### P4：发布与链路优化

- 当前和上一版本静态资源并存；
- 构建时 Brotli/gzip；
- 结合 RUM 决定是否给 `/static` 接 CDN或迁移更近节点；
- 不在没有数据时先进行全站基础设施重构。

P4 涉及生产发布目录、OpenResty 能力和真实用户数据，本分支尚未部署，因此以上三项均是发布阶段待办，不能标记为已经完成。当前代码只完成真实静态 404、分类缓存头和客户端一次性 ChunkLoadError 恢复。

## 13. 性能预算

### 13.1 构建预算

| 项目 | 当前产物 | 预算 |
| --- | ---: | ---: |
| 首页 SSR HTML gzip | 6.0 KiB | <= 50 KiB |
| 最大文档 SSR HTML gzip | 8.9 KiB | <= 100 KiB |
| 首屏关键 CSS gzip | 60.4 KiB | <= 80 KiB |
| 首屏 LCP 图片 | 18.9 KiB | <= 80 KiB |
| 首页渲染前必须下载的 JS | 0 B | 0 B |
| 首页 hydration 必需 JS gzip（含最大公共词典） | 235.5 KiB | <= 450 KiB |
| 最大初始 JS chunk gzip | 70.9 KiB | <= 250 KiB |
| 公共首屏预算传输 | 322.5 KiB | <= 500 KiB |
| 首页首屏同步 API | 0 | 0 |
| hydration 后 status 请求 | 最多 1 | <= 1 |
| 默认首页 home content 请求 | 0 | 0 |
| 未到价格区块时 pricing 请求 | 0 | 0 |

前八项由 `scripts/check-first-screen-budget.ts` 读取实际构建产物并失败退出；请求项由浏览器验收覆盖，不能只依赖文档约定。

### 13.2 实验室目标

使用本次同一移动端配置和冷缓存，取至少五次中位数：

| 指标 | 线上旧基线 | 当前分支本地 | 本轮验收线 |
| --- | ---: | ---: | ---: |
| FCP | 15.75 s | 2.028 s | <= 4 s |
| LCP | 16.74 s | 2.028 s | <= 5 s |
| CLS | 0.0037 | 0.00048 | <= 0.1 |
| 最长长任务 | 610 ms | 250 ms | <= 300 ms；进阶目标 <= 200 ms |
| 传输 | 2.97 MB | LCP 前 298 KB；10 秒共 471 KB | 公共首屏预算 <= 500 KB |

P0-P3 的功能和本地性能验收已经完成。绝对时间可能受本地 fixture 和源站网络差异影响，因此上线后仍需用同一脚本重新记录至少五次中位数；250 ms 长任务通过本轮验收，但不能表述为已经达到 200 ms 进阶目标。

### 13.3 真实用户目标

上线后按国家/地区、网络类型、设备和 build version 分组：

- LCP p75 <= 2.5 s；
- INP p75 <= 200 ms；
- CLS p75 <= 0.1；
- 静态资源失败率和 ChunkLoadError 可按 build version 追踪；
- 首次访问与重复访问分别统计。

如果应用层预算已达标但跨境用户 LCP 仍高，才由 RUM 证据推动 CDN或节点调整。

## 14. 验收清单

### 路由和 SSR

- `/` 在 JavaScript 禁用时可见 Header、Hero、CTA；
- `/docs/*` 在 JavaScript 禁用时可见导航、标题和正文；
- `/dashboard`、`/pricing`、`/sign-in` 仍使用现有 SPA；
- 未 setup 实例访问首页和文档会进入 setup；
- SSR 与 hydration 没有 mismatch 警告；
- 已登录和未登录 CTA 在 hydration 后正确更新。

### 请求和资源

- 默认首页首屏不请求 setup、home content、pricing；
- status 最多一次且不阻止 SSR 内容显示；
- 首屏不加载全量 `@lobehub/icons`；
- 首屏只加载当前语言的公共词典，不加载七个完整语言包或英文 fallback；
- 从公共页进入非公共路由时，先补载当前语言完整词典；
- Base UI/Radix 后台 primitives 不进入首页同步入口；
- Ribbon 动画不会成为 LCP 请求；
- 非首屏区块未接近视口时不下载其脚本和图片；
- 缺失 hash 静态文件返回真实 404；
- 带 hash 资源返回一年 immutable。

### 文档缓存

- 第一次直接访问由 SSR 显示正文；
- 相同 fileName 命中 IndexedDB 时不下载正文；
- manifest fileName 更新后下载新正文并原子替换；
- 新正文失败时保留旧缓存；
- 不同语言使用独立缓存；
- 浏览器不计算正文 hash，不使用 ETag 判断版本。

### 发布

以下条目是发布阶段验收清单，本轮未部署：

- 新静态资源先于新 HTML 可访问；
- 上一版本 HTML 引用的 chunk 仍可访问；
- 受控 ChunkLoadError 刷新最多执行一次；
- 发布后用同一脚本重新记录五次冷缓存和热缓存；
- 指标带 build version，可区分新旧版本。

## 15. 明确不做

- 不做全站 SSR；
- 不迁移 Next.js 或增加生产 Node 服务；
- 不让 SSR 串行等待 status、pricing、notice、home content；
- 不把登录态、钱包或 token 注入公开 HTML；
- 不使用 Service Worker 作为首轮优化，避免增加版本和缓存失效复杂度；
- 不为了分包而制造几十个首屏并发 chunk；
- 不 preload 全部字体、动画、语言包和非首屏脚本；
- 不用无限刷新掩盖发布期缺失 chunk。

## 16. 最终决策

保持现有 Go + React 架构，只为首页和文档页增加构建时 SSR/SSG。

最优实施路径是先用小改动删除当前明确测得的四个阻塞源：setup、重复 status、全量语言包、首页全量图标；同时修复静态资源 404 和缓存。随后 SSR 让 Header、Hero 和文档正文直接随 HTML 到达，文档正文通过文件名 hash 和 IndexedDB 长期缓存，非首屏区块按可见性加载。

这样既保留当前首页视觉、自定义首页能力、DocsShell、TanStack Router 和 Go 单体部署，又能把优化重点放在已测得的真实瓶颈上，而不是用一次大框架迁移替代性能治理。
