# 文档导航

本目录保存仓库维护文档、开发说明和设计记录。线上 `/docs` 的正文在 [前端文档源码](../web/default/src/features/docs/) 中维护；修改这里的 Markdown 不会更新线上页面。

## 开发与运维

| 文档 | 内容 |
| --- | --- |
| [开发、构建与部署](deployment/README.md) | 本地开发、环境配置边界、发布入口、验证与回滚 |
| [前端文档维护](development/frontend-docs.md) | TSX 正文、路由注册、中文预渲染、缓存和图片约定 |
| [翻译术语与贡献入口](i18n/README.md) | 中英、法语、俄语术语表及前端翻译约定 |
| [部署制品说明](../deploy/README.md) | 原生二进制制品目录 |
| [压测工具](../deploy/loadtest/README.md) | 压测配置与执行说明 |

## 渠道与路由

| 文档 | 内容 |
| --- | --- |
| [渠道设置](channel/settings.md) | `setting` 与 `other_settings` 的区别、格式化、思考内容和代理 |
| [自动探测](channel/probes.md) | 探测周期、任务租约、禁用与恢复 |
| [错误码与故障转移](channel/error-code-routing.md) | 分组预算、渠道重试、错误来源和稳定错误码 |
| [动态调度设计](design/pricing-group-dynamic-routing-strategy-design.md) | 策略目标、评分与候选池；实施状态和后续增强见文首说明 |

## 计费与接口

| 文档 | 内容 |
| --- | --- |
| [计费维护入口](billing/README.md) | 单位、计价、预扣与结算的源码入口和排查顺序 |
| [钱包与支付](billing/wallet-and-payments.md) | 充值、账单、奖励和退款的用户说明 |
| [计费表达式](../pkg/billingexpr/expr.md) | 表达式语言、Token 归一化、额度换算和版本规则 |
| [OpenAPI 描述](openapi/README.md) | 管理 API 与模型中转 API 的静态接口描述 |

## 设计、历史记录与素材

- [设计方案目录](design/README.md)：当前方案入口与实施状态说明。
- [历史设计与分析](archive/README.md)：保留计费、返现、新手引导、首屏性能和三协议互转的设计背景；历史运行结果不代表当前环境。
- [素材索引](assets/README.md)：CC Switch 原始截图，供编辑和重新生成发布资源使用。

## 维护约定

- 操作说明放入对应主题目录；实施方案放入 `design/`；已被实现或后续方案替代的设计放入 `archive/`，并注明历史基线与当前源码入口。
- 线上文档以 TSX 为发布源，不再维护一套同内容 Markdown。历史版本可从 Git 查看；需要保留的独有说明和源素材单独分类。
- 新增或移动文档时同步更新本索引和引用。源码链接使用相对路径；避免把临时分支、运行端口、测量结果或未实施建议写成长期有效的事实。
- 配置、数据库和部署操作遵守 [项目协作规则](../AGENTS.md)；文档示例不能授权修改受保护环境配置。
