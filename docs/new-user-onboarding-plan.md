# 新注册用户控制台新手指引方案

> 文档状态：Phase 0 后端契约和 Phase 1 首页引导均已实现并通过定向验证
>
> 目标分支：`codex/new-user-onboarding-plan`
>
> 适用范围：`web/default` 控制台、用户注册/登录链路、用户自助 API

## 1. 背景与目标

用户完成注册并第一次进入控制台时，需要知道三件事：

1. 创建 API Key。
2. 确认可用额度或充值方式。
3. 用 Playground 或兼容客户端发起第一条请求。

当前默认前端首页已经有一个“设置引导”，按 API Key、余额/已用额度和请求数计算这三个步骤的进度，但它只是首页的长期 checklist：展开状态写在浏览器 `localStorage`，没有用户隔离，也没有服务端完成标记。因此它不能可靠表达“新注册用户只展示一次”的产品语义。

项目同时已有 `/docs/quick-start`，内容覆盖 API Key、模型 ID、Playground/客户端接入和验证排错，可作为新手指引的详细教程出口。匿名 `/setup` 是站点管理员初始化向导，与用户 onboarding 无关，不应复用或耦合。

本方案的目标是：

- 新注册用户第一次进入控制台时自动看到一次低打扰的新手指引；明确开始或跳过后不再自动弹出。
- 覆盖当前会创建用户的密码注册、OAuth、微信路径；Passkey 目前只用于已登录用户绑定或已有账户登录，未来若增加 Passkey 直接注册，必须复用公开注册规则。
- 完成或跳过后跨设备、跨浏览器保持状态，并支持未来发布新版指引。
- 复用现有首页设置引导的三步内容，不重复建设一套业务 checklist。
- 老用户、管理员后台代创建的用户默认不被意外打扰。

## 2. 现状调查

### 2.1 认证和注册入口

- 用户模型已有 `CreatedAt`、`LastLoginAt` 和 JSON `Setting`，但没有 onboarding 字段。
- 密码注册在 `controller/user.go:Register` 中创建用户后只返回成功，不自动登录。
- OAuth 新用户、微信新用户以及其他登录方式最终都会进入 `setupLogin`。
- 前端登录成功后由 `useAuthRedirect` 调用 `getSelf()`，写入 `auth-store`，再默认导航到 `/dashboard`。
- `_authenticated` 路由在每个会话首次进入时再次调用 `getSelf()`，因此这里是可靠的用户资料同步点。

关键代码位置：

- `model/user.go:22`：用户模型和创建时间。
- `controller/user.go:144`：统一登录响应 `setupLogin`。
- `controller/user.go:461`：当前用户资料 `GetSelf`。
- `web/default/src/routes/_authenticated/route.tsx:28`：认证路由守卫和 `getSelf` 校验。
- `web/default/src/features/auth/hooks/use-auth-redirect.ts:58`：登录后拉取用户资料并跳转控制台。
- `web/default/src/features/docs/quick-start.tsx:43`：已有快速开始教程，可作为欢迎态新增的详细教程出口。

### 2.2 现有首页设置引导

`web/default/src/features/dashboard/components/overview/overview-dashboard.tsx` 已提供：

- 创建 API Key：跳转 `/keys`。
- 添加额度：跳转 `/wallet`。
- 发起请求：跳转 `/playground`。
- 根据现有 API Key、额度和 `request_count` 自动计算完成数。
- 展开/收起状态：`dashboard_overview_setup_guide_expanded`，目前是全浏览器 `localStorage`。

这部分内容应作为新手指引的任务来源和完成态展示，避免维护两套文案、跳转目标和完成规则。

模型选择、客户端配置和排错留在已有 `/docs/quick-start` 文档中；欢迎态如需深入阅读，应新增明确的链接，不把首次欢迎态扩展成冗长向导。

## 3. 推荐方案

采用“服务端资格状态 + 独立幂等完成接口 + 前端轻量指引”的三层设计。

### 3.1 服务端状态模型

在 `User` 增加版本字段：

```go
OnboardingVersion *int `json:"-" gorm:"column:onboarding_version;type:int"`
```

语义如下：

| 值 | 含义 |
| --- | --- |
| `NULL` | 功能上线前的老用户，或明确不参与指引的后台创建用户 |
| `0` | 新注册用户，尚未确认当前版本指引 |
| `1` | 已开始或跳过当前版本指引，不再自动弹出 |

字段表达“用户最后确认过的指引版本”，不表达三项业务任务是否全部完成。使用指针和 `NULL` 的原因是区分“历史用户”和“新用户”，避免迁移后所有存量账户都被当成新用户。以后发布新版时只需提升 `CurrentOnboardingVersion`，不需要重新设计数据表。

初始化规则：

- 公共密码注册、OAuth 注册、微信注册创建的用户显式写入 `0`。
- 当前 Passkey 仅为已登录用户注册凭证，不创建用户；未来若增加 Passkey 直接注册，复用公开注册的 `0` 初始化规则。
- 管理员通过后台创建的用户默认写入 `NULL`，除非产品后续增加“发送新手指引”的明确选项。
- 功能上线前已存在的用户保持 `NULL`；不使用时间窗口、`request_count` 或 `LastLoginAt` 进行推断。
- 初始 root/setup 用户保持 `NULL`。

字段由 GORM `AutoMigrate(&User{})` 管理，兼容 SQLite、MySQL 和 PostgreSQL。测试/从节点部署前必须确认主节点已完成 schema 迁移，因为 slave 节点会跳过迁移。

### 3.2 API 契约

#### `GET /api/user/self`

在现有响应中增加：

```json
{
  "onboarding_required": true,
  "onboarding_version": 0
}
```

建议 `onboarding_required` 由服务端根据 `OnboardingVersion != nil && *OnboardingVersion < CurrentOnboardingVersion` 计算，前端不重复实现资格规则。已确认或不参与的用户返回 `false`；版本字段可返回 `null` 或省略。

#### 登录响应

`setupLogin` 的简略 `data` 中同步返回 `onboarding_required` 和 `onboarding_version`。这样 OAuth/微信等前端直接消费登录响应时不会出现短暂不一致；`getSelf()` 仍是最终权威数据源。

#### `PUT /api/user/self/onboarding`

该接口表示用户已经确认首次欢迎态，而不是声称三项配置全部完成。请求体为空，版本由服务端使用 `CurrentOnboardingVersion` 决定。服务端只取 session 中的用户 ID，并执行条件更新：

```sql
WHERE id = ?
  AND onboarding_version IS NOT NULL
  AND onboarding_version < current_version
```

要求：

- 仅允许当前登录用户调用。
- 重复调用返回成功，接口幂等。
- 不接受客户端任意版本，避免跳过未来指引版本。
- 确认接口不复用 `PUT /api/user/self` 全量更新，避免与资料/设置保存产生字段覆盖和竞态。
- 确认成功后返回当前版本；前端更新 store 或重新请求 `getSelf()`。

### 3.3 前端体验

第一阶段只做首页内的可访问 checklist，不做跨路由遮罩 tour：

1. 在 `Dashboard` 功能层挂载懒加载的 `OnboardingGate`，只有当前 dashboard section 为 `overview` 且服务端标记为 `onboarding_required=true` 时才自动打开欢迎态。这样 gate 能直接控制现有 checklist 的展开状态，也不会阻塞其他控制台页面；未来做跨路由 spotlight 时再上移到 `AuthenticatedLayout`。
2. 欢迎态提供“开始配置”和“跳过，不再显示”两个动作；两者都先调用确认接口，保证欢迎态只自动出现一次。“开始配置”随后展开现有 checklist，“跳过”只关闭欢迎态。
3. 点击关闭、遮罩或按 `Esc` 等同于“跳过”，同样确认当前版本；网络失败时保留 pending，后续进入 overview 可重试。
4. 每步保留现有链接和自动完成规则；三步完成时只展示完成反馈，不再改写首次欢迎状态。
5. 刷新、重新登录、换浏览器或换设备后不重复出现。
6. 用户可以从首页长期 checklist 手动再次查看任务，但不再显示首次欢迎弹窗。

欢迎态和 checklist 首版都放在 `Dashboard` 功能层，由 `OverviewDashboard` 集中管理服务端确认、用户隔离的本地 UI 状态和三步完成规则，并通过子组件 `NewUserOnboardingDialog` 渲染欢迎弹窗。这样可以先保持 MVP 的状态边界清晰；当后续引入跨路由 spotlight 时，再把服务端确认逻辑提取为共享 hook 并上移到 `AuthenticatedLayout`。

前端状态建议：

- `AuthUser` 增加 `onboarding_required?: boolean` 和 `onboarding_version?: number | null`；共享用户 schema 同步接受这两个只读字段。
- `useAuthStore` 只订阅 `state.auth.user`，不新增全局宽订阅。
- `localStorage` 仅作为欢迎动画的临时 UI 状态或异常降级缓存，key 必须包含 user id；不能作为资格或完成状态的权威来源。
- 使用现有 `motion` 和 Base UI Dialog/Popover，不新增 tour 依赖；若未来引入第三方库，必须动态 import。
- 所有文案使用 `useTranslation().t()`，同步 `en/zh/fr/ja/ru/vi`，并检查现有 `zh-TW` 和 `static-keys.ts`。
- 纯状态/版本逻辑沿用仓库现有的 `node:test` 测试模式；组件交互以类型检查、lint、构建和手工键盘/窄屏验收为主，不预设新增 React Testing Library 依赖。

## 4. 分阶段实施

### Phase 0：契约和数据准备（已完成）

- 新增 `CurrentOnboardingVersion` 常量和 `User.OnboardingVersion`。
- 明确各用户创建路径的初始化策略，补充 `GetSelf`、`setupLogin` 和完成接口。
- 为 SQLite/MySQL/PostgreSQL 验证 AutoMigrate；在测试环境先确认 schema，再部署应用。
- 增加后端 API 契约和确认接口幂等测试。

### Phase 1：MVP 首页引导（已完成）

- 扩展 `AuthUser` 和 auth API 类型。
- 在 `OverviewDashboard` 内集中管理资格、确认、错误和 store 更新，新增 `NewUserOnboardingDialog` 负责欢迎态渲染。
- 在 `OverviewDashboard` 复用现有三步任务，加入欢迎态、开始、跳过、完成状态。
- 为按钮、弹层、键盘焦点、Esc、窄屏和 reduced-motion 提供可访问行为。

当前实现同时完成了首页 checklist 的首轮视觉增强：

- 三个步骤直接显示为长条按钮，不再使用左侧圆圈或勾选图标。
- 未进行步骤为白色，已完成或已跳过步骤为绿色；当前步骤放大并使用黄色脉冲提示，且遵循 `prefers-reduced-motion`。
- 当前步骤在右箭头旁提供跳过操作。
- 第三步先展示双卡片选择弹窗，可进入网站 Playground，或复用 CC Switch 一键导入 Agent 客户端。
- 移除原“首个 API 请求”卡片，右侧改为推荐操作。
- 欢迎态仅由服务端 `onboarding_required` 控制；开始、跳过、关闭和 `Esc` 都调用幂等确认接口。开始后强制展开 checklist，跳过后收起；失败时保持欢迎态并提示。
- checklist 中逐步跳过任务；当所有步骤都已完成或跳过时，同样确认服务端 onboarding 状态，避免仅依赖浏览器状态导致欢迎态再次出现。
- checklist 展开状态和步骤跳过状态都按用户 ID 隔离；API Key 查询缓存同样包含用户 ID，并读取前 100 条以降低遗漏已启用 Key 的概率。

### Phase 2：增强引导（可选）

- 根据真实使用数据决定是否需要跨路由 spotlight。
- 若需要，在 `AuthenticatedLayout` 挂载 orchestrator，并给 `/keys`、`/wallet`、`/playground` 增加稳定的 `data-onboarding-id`。
- 目标元素不存在、被角色权限隐藏、侧边栏折叠或移动端抽屉未打开时，自动降级为首页 checklist，不阻塞导航。

### Phase 3：运营和版本化

- 将 `CurrentOnboardingVersion` 作为发布配置审查项。
- 记录开始、跳过、完成事件（若产品需要分析），事件中不记录 API Key 或敏感用户数据。
- 评估管理员是否需要为指定用户重置/重新发送某一版本指引；如需要，再增加受权限保护的管理接口。

## 5. 测试与验收

### 后端

- 新公共注册用户的 `onboarding_version` 为 `0`。
- OAuth、微信等自助注册用户同样为 `0`，且即时登录响应带有 `onboarding_required`；已有账户的 Passkey/Telegram 登录不新建 onboarding 状态。
- 当前 Passkey 注册是已登录用户的凭证绑定，不属于用户创建路径；如果未来支持 Passkey 直接创建新账户，该路径必须复用自助注册的 `0` 初始化规则。
- 旧用户和后台管理员创建用户不触发指引。
- `GET /api/user/self` 与登录响应字段一致。
- 确认接口只影响当前用户；重复请求、并发请求均成功且最终版本正确。
- 数据库迁移在 SQLite、MySQL、PostgreSQL 均可执行。

### 前端

- 首次进入控制台只显示一次欢迎态；刷新、再次登录、换设备不重复。
- 点击开始或跳过后服务端状态立即生效，网络失败时不误报已确认。
- 三步完成规则与现有 API Key、额度、请求数保持一致。
- 普通用户不会被带到管理员专属目标；目标不可见时 checklist 仍可用。
- 移动端、侧边栏折叠、深色主题、语言切换、键盘操作和 reduced-motion 均可用。
- 执行 `bun run typecheck`、涉及文件 lint、`bun run build`。

## 6. 风险与取舍

| 方案 | 结论 | 原因 |
| --- | --- | --- |
| 只用 `created_at`/`last_login_at` 判断首次 | 不采用 | 秒级时间相同、后台创建、历史用户和多登录方式会误判 |
| 只用 `request_count === 0` | 不采用 | 用户可能已使用其他客户端，且无法表示是否看过/跳过引导 |
| 只用浏览器 `localStorage` | 仅作降级 | 无法跨设备，账号之间会串状态，清理缓存后会重复 |
| 将状态塞进 `UserSetting` JSON | 不采用 | 设置接口会重建对象，存在覆盖未知字段的风险 |
| 单独 `user_onboardings` 表 | 暂不采用 | MVP 只有一个版本字段；未来有多套流程、审计和步骤明细时再升级 |
| 首版引入第三方 tour 库 | 不采用 | 增加包体和定位适配成本；现有页面更适合轻量 modal/checklist |

## 7. 待产品确认

1. 是否需要“稍后提醒”而不是严格的一次性欢迎？本方案按“一次”理解，开始、跳过和关闭都会确认当前版本；长期 checklist 仍可手动展开。
2. 管理员后台创建的普通用户是否需要一个“发送新手指引”选项？本方案默认不发送。
3. 是否需要记录引导漏斗（开始、跳过、完成）？本方案默认先不新增埋点。
4. 首版是否只支持默认前端？本方案范围限定为 `web/default`，classic 后续另行评估。

## 8. 推荐验收顺序

先合并 Phase 0 的后端契约和迁移验证，再合并 Phase 1 的首页 MVP；上线后观察新用户完成率、跳过率和支持工单，再决定是否投入 Phase 2 的跨路由 spotlight。
