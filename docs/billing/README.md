# 计费维护入口

计费涉及模型基础价、分组倍率、额度单位、预扣、最终结算和支付到账。排查时先确定使用的是哪一套单位和哪一种计价路径。

## 单位与价格

| 概念 | 含义与源码入口 |
| --- | --- |
| `quota` / `QuotaPerUnit` | 内部额度；基础换算常量见 [common/constants.go](../../common/constants.go) |
| `USDExchangeRate` | 钱包展示相关汇率；不能用它代替模型消费计费汇率 |
| `BillingUSDToCNYRate` | 模型消费计费汇率；读取与校验见 [operation_setting](../../setting/operation_setting/) |
| `ModelRatio` / `ModelPrice` | 按 Token 与固定按次模型价格；配置见 [ratio_setting](../../setting/ratio_setting/) |
| 分组倍率 | 请求实际计费分组对应的倍率；计价入口见 [price.go](../../relay/helper/price.go) |
| 动态表达式 | 价格系数、用量归一化和额度换算遵循 [表达式说明](../../pkg/billingexpr/expr.md) |

价格、汇率、分组倍率和支付产品配置是独立的配置项。模型价格展示与实际销售价需要结合站点配置核对，不能把历史设计中的 `7.3` 或 `0.05` 当作当前环境的固定值。

## 请求计费链路

| 环节 | 核对入口 |
| --- | --- |
| 请求校验和预估 | [请求校验](../../relay/helper/valid_request.go)、[模型计价与预扣估算](../../relay/helper/price.go) |
| 钱包或订阅预扣 | [计费会话](../../service/billing_session.go) |
| 同步请求结算 | [quota.go](../../service/quota.go) |
| 异步任务快照、差额与退款 | [task_billing.go](../../service/task_billing.go) |
| 数值转换与异常审计 | [quota_math.go](../../common/quota_math.go)、[log_info_generate.go](../../service/log_info_generate.go) |

新增计费路径时按 [项目计费安全约定](../../AGENTS.md) 跟踪完整链路；用户可控的数量先在输入边界校验，额度换算复用公共转换函数，异常钳制保留管理员审计信息。

## 支付与账单

用户说明见 [钱包与支付](wallet-and-payments.md)。支付金额、币种、签名和订单入账应核对对应的 `controller/topup*.go`、`controller/subscription_payment*.go`、`model/topup.go` 和外部商户配置。

源码检查不能代替真实支付验收。上线前按实际启用的支付通道核对订单金额、实付币种、到账额度和钱包展示，并分别验证按 Token、按次、表达式及异步任务的消费日志。

## 历史资料

- [人民币钱包设计](../archive/billing/cny-billing-minimal-change-design.md)：原始业务目标、迁移和验收背景。
- [模型计价与账务分析](../archive/billing/model-pricing-and-billing-system.md)：2026-07-28 的源码分析快照，包含当时的运行观察和待验证项。

历史记录中的源码行号、测试结果、商户状态和部署结论都需要在当前环境重新确认。
