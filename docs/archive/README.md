# 历史设计与分析

这些文档保留决策背景、业务约束、实现思路和历史验收记录。正文中的“当前”“已完成”“待实现”均指原文基线；没有在本次文档整理中重新执行部署、真实支付或性能验收。

| 文档 | 保留价值与使用边界 |
| --- | --- |
| [人民币钱包设计](billing/cny-billing-minimal-change-design.md) | 钱包面值、模型汇率、价格展示、迁移与验收的原始目标；具体配置值不是环境默认值 |
| [计价与账务分析](billing/model-pricing-and-billing-system.md) | 2026-07-28 源码分析与支付风险清单；旧运行状态、行号和测试结果仅供追溯 |
| [邀请返现产品设计](design/affiliate-cashback-product-design.md) | 邀请归因、冻结、奖励快照、权限与账务设计；现金/额度模型和 API 需核对当前实现 |
| [邀请返现管理页方案](design/affiliate-cashback-management-ui-change-proposal.md) | 从双列表改为充值记录管理的交互背景及 SVG 稿；接口字段是当时建议 |
| [新用户引导方案](design/new-user-onboarding-plan.md) | 版本状态、历史用户区分和后续阶段；核心版本字段已有实现，可选阶段不代表已落地 |
| [首屏性能方案](design/frontend-ssr-first-screen-architecture-design.md) | 2026-08-02/03 测量基线、优化顺序及当时的本地验收；不代表现在线上性能 |
| [三协议互转设计](design/openai-responses-claude-messages-interoperability-design.md) | 2026-08-03 的兼容子集、工具与流式事件取舍；“Claude Responses 尚未接线”的旧前提已失效 |

当前维护入口见 [文档总索引](../README.md)、[计费维护](../billing/README.md) 和 [前端文档维护](../development/frontend-docs.md)。历史版本和被替代的迁移稿可从 Git 记录查询。
