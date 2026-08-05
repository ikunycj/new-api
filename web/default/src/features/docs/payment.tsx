/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  ArrowRight01Icon,
  InformationCircleIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import { DocsShell } from './components/docs-shell'
import { NumberedSteps } from './components/numbered-steps'

const PAYMENT_TOC = [
  { id: 'wallet-overview', label: '钱包中的数值' },
  { id: 'online-top-up', label: '在线充值' },
  { id: 'referral-rewards', label: '推荐奖励' },
  { id: 'billing-records', label: '充值记录与使用日志' },
  { id: 'settlement-refunds', label: '预扣、结算与退还' },
  { id: 'faq', label: '常见问题' },
]

const TOP_UP_STEPS = [
  '打开钱包，在“添加资金”区域选择预设档位，或手动输入充值数量。',
  '确认输入值不低于页面显示的最低充值数量。',
  '选择页面提供的支付方式。',
  '在确认窗口核对充值数量、折扣、实付金额和支付方式。',
  '跳转到支付服务商页面并完成付款，然后返回钱包。',
  '刷新页面，并在“账单历史”中核对订单状态和到账结果。',
]

const REFERRAL_STEPS = [
  '复制自己的推荐链接，并仅通过站点允许的渠道分享。',
  '在“推荐计划”卡片中查看待转奖励和活动进度。',
  '有可转奖励时，点击“转入余额”。',
  '输入不低于页面所示最低值、且不超过待转奖励的数量。',
  '转入成功后，确认待转奖励减少、钱包余额增加。',
]

const SETTLEMENT_STEPS = [
  '请求开始前，根据模型、参数和预计用量预扣一部分钱包或订阅额度。',
  '请求完成后，根据实际用量结算，多退少补。',
  '请求在需要回滚的阶段失败时，系统会退还相应的预扣额度。',
  '异步任务失败或最终费用发生变化时，系统会进行退款或差额结算，并在使用日志中留下相应记录。',
]

const SUBSCRIPTION_BILLING_CHECKS = [
  '套餐是否仍在有效期内。',
  '套餐额度是否已经用尽。',
  '当前模型是否符合套餐规则。',
  '计费偏好是否设为“钱包优先”或“仅钱包”。',
  '使用日志中的计费来源和订阅结算详情。',
]

function FaqItem(props: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className='text-lg font-semibold'>{props.title}</h3>
      <div className='text-muted-foreground mt-2 leading-7'>
        {props.children}
      </div>
    </section>
  )
}

export function DocsPayment() {
  return (
    <DocsShell
      pageId='payment'
      title='计费与支付'
      description='通过钱包查看余额与累计用量、在线充值、管理推荐奖励，并核对充值订单和模型请求的实际扣费。'
      toc={PAYMENT_TOC}
    >
      <section id='wallet-overview' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>先认识钱包中的几个数值</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          中转站的资金与计费入口集中在“钱包”页面。钱包顶部会显示以下账户统计：
        </p>
        <ul className='text-muted-foreground mt-4 list-disc space-y-2 pl-5 leading-7'>
          <li>
            <strong className='text-foreground'>当前余额：</strong>
            尚未使用的钱包额度。
          </li>
          <li>
            <strong className='text-foreground'>累计用量：</strong>
            账户历史累计消耗的额度，不等于本期账单。
          </li>
          <li>
            <strong className='text-foreground'>API 请求数：</strong>
            账户累计发出的 API 请求数量。
          </li>
        </ul>
        <p className='text-muted-foreground mt-4 leading-7'>
          余额的展示单位由站点设置决定，可能显示为货币或额度。充值页面同时出现“充值数量”和“实付金额”时，请以确认窗口中的实付金额为准；预设档位折扣、账户分组充值比例和站点换算规则都可能使两者不完全相同。
        </p>
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button render={<Link to='/wallet' />} nativeButton={false}>
            打开钱包
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button
            variant='outline'
            render={<Link to='/docs/model-pricing' />}
            nativeButton={false}
          >
            查看模型定价与消耗
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>
      </section>

      <section id='online-top-up' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>在线充值</h2>
        <h3 className='mt-5 text-lg font-semibold'>充值步骤</h3>
        <NumberedSteps items={TOP_UP_STEPS} />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>避免重复支付</AlertTitle>
          <AlertDescription>
            部分支付渠道会打开新窗口，部分渠道会在当前标签页跳转。浏览器拦截弹窗时，可以允许本站弹窗后重新发起；已经生成订单或完成付款时，不要因为页面没有立即跳转而连续重复支付。
          </AlertDescription>
        </Alert>
      </section>

      <section id='referral-rewards' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>推荐奖励</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          站点启用推荐计划后，钱包会显示推荐链接、待转奖励、累计奖励和邀请人数。
        </p>
        <NumberedSteps items={REFERRAL_STEPS} />
        <p className='text-muted-foreground mt-4 leading-7'>
          是否发放注册奖励、充值奖励，哪些充值符合条件，以及是否允许查看被邀请人的充值记录，均由当前活动规则决定。管理员尚未确认相关合规条款时，奖励转入功能不可用。
        </p>
      </section>

      <section id='billing-records' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>充值记录与使用日志</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          “账单历史”和“使用日志”记录的是不同事情：
        </p>
        <div className='mt-5 overflow-x-auto rounded-lg border'>
          <table className='w-full min-w-[680px] text-left text-sm'>
            <thead className='bg-muted/60'>
              <tr>
                <th className='px-4 py-3 font-semibold'>入口</th>
                <th className='px-4 py-3 font-semibold'>用途</th>
                <th className='px-4 py-3 font-semibold'>主要信息</th>
              </tr>
            </thead>
            <tbody className='divide-y'>
              <tr>
                <td className='px-4 py-3 font-medium'>钱包 → 账单历史</td>
                <td className='text-muted-foreground px-4 py-3'>
                  查询在线充值订单
                </td>
                <td className='text-muted-foreground px-4 py-3'>
                  订单号、创建时间、支付方式、充值数量、实付金额、订单状态
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>
                  <Link
                    to='/usage-logs'
                    className='text-primary underline-offset-4 hover:underline'
                  >
                    使用日志
                  </Link>
                </td>
                <td className='text-muted-foreground px-4 py-3'>
                  查询模型请求和实际扣费
                </td>
                <td className='text-muted-foreground px-4 py-3'>
                  模型、令牌、输入/输出 Token、费用、耗时、请求状态、计费来源
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <h3 className='mt-7 text-lg font-semibold'>充值订单状态</h3>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          <li>
            <strong className='text-foreground'>成功：</strong>
            支付回调已经确认，充值额度已入账。
          </li>
          <li>
            <strong className='text-foreground'>待处理：</strong>
            订单已经创建，但系统尚未确认支付成功。
          </li>
          <li>
            <strong className='text-foreground'>已过期：</strong>
            订单已经失效，不应继续使用原订单完成支付。
          </li>
        </ul>
        <p className='text-muted-foreground mt-4 leading-7'>
          账单历史支持按订单号搜索。联系客服处理充值问题时，应提供订单号、付款时间、实付金额和支付服务商凭证；不要提供账户密码、API
          密钥或完整兑换码。
        </p>
        <p className='text-muted-foreground mt-4 leading-7'>
          使用日志中的费用才是单次模型请求的最终计费依据。订阅支付的请求会标记为“订阅”，详情中还可查看套餐实例、预扣额度、结算差额、最终消耗和套餐剩余额度。
        </p>
      </section>

      <section id='settlement-refunds' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>预扣、结算与退还</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          模型调用通常不是在发出请求前就能确定最终费用，因此系统采用“预扣后结算”的流程：
        </p>
        <NumberedSteps items={SETTLEMENT_STEPS} />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>额度退还不等于支付退款</AlertTitle>
          <AlertDescription>
            API
            计费额度会退回原计费来源：钱包扣费退回钱包，订阅扣费退回对应订阅。它不等同于将已经支付的充值款或套餐购买款原路退回银行卡、支付宝或其他外部支付账户。
          </AlertDescription>
        </Alert>
        <p className='text-muted-foreground mt-4 leading-7'>
          当前用户端没有通用的自助撤销充值、取消套餐或原路退款入口。支付退款、误购或重复付款需要联系站点客服按实际订单处理；在得到明确处理结果前，不要再次提交同一笔付款。
        </p>
      </section>

      <section id='faq' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常见问题</h2>
        <div className='mt-6 space-y-8'>
          <FaqItem title='钱包没有显示在线支付方式'>
            <p>
              在线充值可能尚未启用，支付合规确认尚未完成，或当前支付渠道暂不可用。可以使用页面仍然显示的兑换码或套餐方式；若所有入口都不可用，请联系站点客服。
            </p>
          </FaqItem>

          <FaqItem title='充值金额为什么和实付金额不同'>
            <p>
              充值数量代表加入账户的额度或展示单位，实付金额还会受到支付币种、站点换算、账户分组比例和充值档位折扣影响。提交订单前以确认窗口为准。
            </p>
          </FaqItem>

          <FaqItem title='已经付款，但余额没有变化'>
            <p>
              先不要重复付款。返回钱包刷新余额，再打开“账单历史”按订单号查询：
            </p>
            <ul className='mt-3 list-disc space-y-2 pl-5'>
              <li>
                状态为“成功”但余额仍不正确时，保留订单号和付款凭证联系客服；
              </li>
              <li>状态为“待处理”时，可能是支付回调尚未完成，稍后再次刷新；</li>
              <li>账单中完全没有订单时，核对支付页面对应的站点和付款结果。</li>
            </ul>
          </FaqItem>

          <FaqItem title='订单一直处于待处理状态'>
            <p>
              不要继续支付原订单或连续创建多笔订单。保留支付服务商的订单号、本站账单订单号、付款时间和金额，交由客服核对。普通用户无法手动把待处理订单改为成功。
            </p>
          </FaqItem>

          <FaqItem title='兑换码无法使用'>
            <p>
              检查是否输入完整、是否带空格，并确认没有重复提交。由于页面不会暴露兑换失败的细分原因，仍然失败时请向兑换码来源方或站点客服核实。
            </p>
          </FaqItem>

          <FaqItem title='有订阅，为什么仍然扣了钱包'>
            <p>依次检查：</p>
            <NumberedSteps items={SUBSCRIPTION_BILLING_CHECKS} />
          </FaqItem>

          <FaqItem title='请求失败后余额看起来没有立即恢复'>
            <p>
              先刷新钱包，再查看该请求的使用日志或异步任务退款记录。部分异步任务需要等到任务状态确认后才会结算。如果日志显示已经退还但余额仍不一致，请提供请求时间、模型、日志记录和请求
              ID 联系客服。
            </p>
          </FaqItem>
        </div>
      </section>
    </DocsShell>
  )
}
