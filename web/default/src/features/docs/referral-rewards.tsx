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
import { DocsTable } from './components/docs-table'
import { NumberedSteps } from './components/numbered-steps'

export function DocsReferralRewards() {
  return (
    <DocsShell
      pageId='referral-rewards'
      title='推荐奖励'
      description='了解邀请码、邀请关系、注册与充值奖励、待结算时间以及自动转入钱包规则。'
      toc={[
        { id: 'how-it-works', label: '奖励如何运作' },
        { id: 'share-link', label: '获取并分享推荐链接' },
        { id: 'reward-states', label: '奖励状态' },
        { id: 'transfer', label: '转入钱包余额' },
        { id: 'records', label: '查看奖励记录' },
        { id: 'rules', label: '规则与安全' },
        { id: 'faq', label: '常见问题' },
      ]}
    >
      <section id='how-it-works' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>奖励如何运作</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          每个用户都有自己的邀请码和推荐链接。新用户通过推荐链接注册后，系统会绑定一次邀请关系；实际奖励类型、比例、固定额度、最低充值金额和结算等待时间由当时生效的活动规则决定。
        </p>
        <NumberedSteps
          items={[
            '邀请人复制自己的推荐链接并分享给尚未注册的新用户。',
            '新用户通过该链接完成注册，邀请关系绑定到双方账户。',
            '活动配置了注册奖励时，系统按触发条件发放；配置了充值奖励时，在符合条件的在线充值成功后计算。',
            '需要等待的奖励先进入待结算状态，到期后由系统自动转入主钱包。',
            '转入完成后，奖励额度会进入邀请人的主钱包，可用于模型调用或站点允许的消费。',
          ]}
        />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>页面展示的当前规则才是准确信息</AlertTitle>
          <AlertDescription>
            推荐活动可以暂停或调整，部分用户也可能使用单独规则。不要根据旧截图或他人的奖励金额推断自己的实际比例。
          </AlertDescription>
        </Alert>
      </section>

      <section id='share-link' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>获取并分享推荐链接</h2>
        <NumberedSteps
          items={[
            '打开钱包页面的推荐奖励卡片。',
            '复制推荐链接或邀请码，确认链接中的邀请码与卡片显示一致。',
            '把完整链接发送给尚未注册的新用户。',
            '请对方从该链接进入注册页并完成注册，不要在注册后再尝试补绑邀请码。',
          ]}
        />
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button nativeButton={false} render={<Link to='/wallet' />}>
            打开钱包
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>
      </section>

      <section id='reward-states' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>奖励状态</h2>
        <div className='mt-5'>
          <DocsTable
            headers={['状态', '含义', '需要做什么']}
            rows={[
              {
                key: 'pending',
                cells: [
                  '待结算',
                  '奖励已经产生，但仍处于活动规则规定的等待期',
                  '等待系统自动释放，不要重复操作充值订单',
                ],
              },
              {
                key: 'available',
                cells: [
                  '自动转入',
                  '等待期结束后由系统自动转入主钱包',
                  '无需邀请人手动操作',
                ],
              },
              {
                key: 'partial',
                cells: [
                  '转入中',
                  '系统正在处理奖励转入主钱包',
                  '刷新奖励记录查看最新状态',
                ],
              },
              {
                key: 'transferred',
                cells: [
                  '已转入',
                  '奖励已经进入主钱包余额',
                  '到钱包或转入记录核对余额变化',
                ],
              },
              {
                key: 'adjusted',
                cells: [
                  '已调整',
                  '管理员因订单、退款或规则核对进行了修正',
                  '查看记录备注，有疑问时提供记录时间联系客服',
                ],
              },
            ]}
          />
        </div>
      </section>

      <section id='transfer' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>自动转入钱包余额</h2>
        <NumberedSteps
          items={[
            '奖励先按照当前规则进入待结算状态。',
            '等待时间结束后，系统自动把可用奖励转入邀请人的主钱包。',
            '转入完成后，主钱包余额增加，奖励记录会显示为已转入。',
            '如果奖励仍处于待结算状态，请等待系统结算任务完成后再刷新页面。',
          ]}
        />
        <p className='text-muted-foreground mt-4 leading-7'>
          推荐奖励转入是站内额度划转，不是向银行卡、支付宝或其他外部账户提现。邀请人不需要手动提交转入请求。
        </p>
      </section>

      <section id='records' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>查看奖励记录</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          推荐奖励中心会展示邀请人数、符合条件的人数、待结算奖励、可转入奖励和累计奖励。站点允许时，还可以查看被邀请人的充值事件；邮箱等身份信息会按权限和隐私规则处理。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={['记录', '用于核对']}
            rows={[
              {
                key: 'topups',
                cells: [
                  '邀请充值记录',
                  '被邀请人的脱敏标识、充值时间、符合条件与否、对应奖励及释放时间',
                ],
              },
              {
                key: 'transfers',
                cells: [
                  '余额转入记录',
                  '转入数量、转入前后推荐余额、主钱包变动和处理结果',
                ],
              },
              {
                key: 'wallet',
                cells: [
                  '钱包与使用日志',
                  '奖励转入后的主钱包余额，以及后续模型请求的实际消耗',
                ],
              },
            ]}
          />
        </div>
      </section>

      <section id='rules' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>规则与安全</h2>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          <li>
            邀请关系通常在注册时绑定，同一被邀请人不能重复绑定多个邀请人。
          </li>
          <li>
            是否发放注册奖励、哪些充值合格、奖励按比例还是固定额度，均以活动页面和钱包当前展示为准。
          </li>
          <li>充值退款、撤销、异常订单或风控处理可能导致奖励被调整。</li>
          <li>不要通过批量注册、自我邀请、虚假充值或其他方式规避活动规则。</li>
          <li>
            分享推荐链接即可，不要向被邀请人索取密码、API
            Key、付款验证码或完整支付凭证。
          </li>
        </ul>
      </section>

      <section id='faq' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常见问题</h2>
        <div className='mt-5 space-y-6'>
          <div>
            <h3 className='font-semibold'>
              为什么有人注册了，但邀请人数没有变化？
            </h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              确认对方是否在注册前通过完整推荐链接进入，以及是否已经拥有账户。注册后再打开推荐链接通常不会补绑关系。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>为什么充值后奖励仍是待结算？</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              活动可能设置等待期，用于确认支付结果和订单状态。达到释放时间后系统会自动处理，无需重复充值。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>为什么看不到被邀请人的充值记录？</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              该能力由活动规则控制。管理员关闭可见性时，用户仍可看到自己的奖励汇总，但不能浏览邀请充值明细。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>奖励没有自动转入怎么办？</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              先确认奖励是否仍在等待期。等待时间结束后刷新页面；如果长时间仍未转入，请保留奖励记录、时间和错误信息联系客服。
            </p>
          </div>
        </div>
      </section>
    </DocsShell>
  )
}
