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

const toc = [
  { id: 'quick-links', label: '常用入口' },
  { id: 'find-model', label: '在模型广场查找模型' },
  { id: 'model-details', label: '看懂模型详情' },
  { id: 'billing-modes', label: '三种常见计费方式' },
  { id: 'groups-channels-protocols', label: '分组、渠道和协议' },
  { id: 'balance-and-quota', label: '余额、订阅额度和密钥额度' },
  { id: 'verify-charge', label: '用使用日志核对真实消耗' },
  { id: 'faq', label: '常见问题' },
]

const quotaCheckSteps = [
  '钱包或订阅是否仍有可用额度。',
  '当前 API 密钥是否还有剩余额度。',
  '密钥选择的分组是否支持目标模型和接口。',
  '请求预估额度是否超过当前可用余额。',
]

const verifyChargeSteps = [
  '在模型广场选择具体模型和分组，复制完整模型 ID。',
  '使用准备投入生产的 API 密钥发送一个小请求。',
  '打开使用日志，按时间、模型或密钥名称定位该请求；需要精确排查时可使用 Request ID。',
  '查看列表中的模型、输入/输出 Token、缓存用量、计费分组、费用和响应耗时。',
  '打开详情，核对计费来源、计费方式、输入/输出价格、分组倍率、附加项目和总消耗。',
  '再到钱包或订阅信息中确认余额变化。',
]

export function DocsModelPricing() {
  return (
    <DocsShell
      pageId='model-pricing'
      title='模型定价与消耗'
      description='在调用模型前，建议先确认模型 ID、可用分组、接口类型和计费方式。不要只根据模型名称或厂商公开价格估算费用；本站最终扣费还可能受到实际处理请求的分组、接口协议、缓存或多模态用量、动态计费规则等因素影响。'
      toc={toc}
    >
      <Alert>
        <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
        <AlertTitle>价格以页面和日志为准</AlertTitle>
        <AlertDescription className='leading-6'>
          模型和价格可能随服务配置调整。下单或批量调用前，请以
          <Link
            to='/pricing'
            className='text-foreground font-medium underline underline-offset-4'
          >
            模型广场
          </Link>
          当前展示为准；已经发生的请求，以
          <Link
            to='/usage-logs'
            className='text-foreground font-medium underline underline-offset-4'
          >
            使用日志
          </Link>
          记录的实际消耗为准。
        </AlertDescription>
      </Alert>

      <section id='quick-links' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常用入口</h2>
        <ul className='text-muted-foreground marker:text-foreground/40 mt-4 list-disc space-y-2 pl-6 leading-7'>
          <li>
            <Link
              to='/pricing'
              className='text-foreground font-medium underline underline-offset-4'
            >
              模型广场
            </Link>
            ：查找模型、比较价格、确认分组和接口类型。
          </li>
          <li>
            <Link
              to='/keys'
              className='text-foreground font-medium underline underline-offset-4'
            >
              API 密钥
            </Link>
            ：创建密钥并设置固定分组或多个候选分组。
          </li>
          <li>
            <Link
              to='/usage-logs'
              className='text-foreground font-medium underline underline-offset-4'
            >
              使用日志
            </Link>
            ：查看每次请求的模型、用量、计费分组和最终消耗。
          </li>
          <li>
            <Link
              to='/wallet'
              className='text-foreground font-medium underline underline-offset-4'
            >
              钱包
            </Link>
            ：查看余额、已用额度和本站已启用的充值或订阅方式。
          </li>
        </ul>
        <p className='text-muted-foreground mt-4 leading-7'>
          接口路径和请求示例可继续阅读
          <Link
            to='/docs/api/integration'
            className='text-foreground font-medium underline underline-offset-4'
          >
            API 模型接口
          </Link>
          ，充值、兑换和订阅说明见
          <Link
            to='/docs/payment'
            className='text-foreground font-medium underline underline-offset-4'
          >
            计费与支付
          </Link>
          。
        </p>
      </section>

      <section id='find-model' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>在模型广场查找模型</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          进入模型广场后，可以直接搜索模型
          ID，也可以搜索模型描述、标签或供应商名称。模型较多时，可组合使用以下筛选项：
        </p>
        <ul className='text-muted-foreground marker:text-foreground/40 mt-4 list-disc space-y-2 pl-6 leading-7'>
          <li>
            <strong className='text-foreground'>供应商：</strong>
            按模型所属厂商筛选。
          </li>
          <li>
            <strong className='text-foreground'>定价分组：</strong>
            只查看指定分组可用的模型及对应价格。
          </li>
          <li>
            <strong className='text-foreground'>计费类型：</strong>
            区分按 Token、按次和动态计费模型。
          </li>
          <li>
            <strong className='text-foreground'>接口类型：</strong>
            筛选
            Chat、Responses、Anthropic、Gemini、Embedding、图片或视频等接口。
          </li>
          <li>
            <strong className='text-foreground'>模型标签：</strong>
            按推理、视觉、工具调用等标签缩小范围，具体标签以页面为准。
          </li>
        </ul>
        <div className='text-muted-foreground mt-4 space-y-3 leading-7'>
          <p>
            页面还支持推荐排序、名称排序、价格升降序、卡片或表格视图，以及
            CNY/USD、每 1M/1K Token 的显示切换。这些选项主要用于比较，切换货币或
            Token 单位不会改变请求的实际计费规则。
          </p>
          <p>
            登录后，模型列表和分组价格会结合当前账户可用范围展示。未指定分组筛选时，列表中的摘要价格可能采用当前可用分组中的较低价格，适合横向比较，但不代表任意
            API
            密钥都一定按这个价格结算。准备正式调用时，应选择具体分组再核对价格。
          </p>
        </div>
        <Button className='mt-5' render={<Link to='/pricing' />}>
          打开模型广场
          <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
        </Button>
      </section>

      <section id='model-details' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>看懂模型详情</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          点击模型卡片或表格行，可打开模型详情。首先复制页面中的
          <strong className='text-foreground'>完整模型 ID</strong>
          ，调用时必须保持大小写、连字符和后缀一致，不要凭印象简写。
        </p>
        <div className='border-border mt-5 overflow-x-auto rounded-lg border'>
          <table className='w-full min-w-[680px] text-left text-sm'>
            <thead className='bg-muted/50 text-foreground'>
              <tr>
                <th className='px-4 py-3 font-medium'>区域</th>
                <th className='px-4 py-3 font-medium'>重点查看内容</th>
              </tr>
            </thead>
            <tbody className='divide-border divide-y'>
              <tr>
                <td className='px-4 py-3 font-medium'>概览</td>
                <td className='text-muted-foreground px-4 py-3'>
                  模型说明、能力、输入输出模态、上下文信息等；未配置的数据不会显示
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>定价</td>
                <td className='text-muted-foreground px-4 py-3'>
                  基础输入价、输出价，以及可能存在的缓存、图片、音频价格
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>分组价格</td>
                <td className='text-muted-foreground px-4 py-3'>
                  当前账户可用分组、分组倍率和各分组的实际展示价格
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>动态定价</td>
                <td className='text-muted-foreground px-4 py-3'>
                  阶梯价格、匹配条件和附加倍率；仅在该模型配置了相关规则时显示
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>模型信息</td>
                <td className='text-muted-foreground px-4 py-3'>
                  供应商、标签、计费类型、可用分组和支持的接口类型
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>性能</td>
                <td className='text-muted-foreground px-4 py-3'>
                  有监测数据时展示延迟、吞吐或可用性，用于辅助选择，不代表每次请求结果
                </td>
              </tr>
              <tr>
                <td className='px-4 py-3 font-medium'>API</td>
                <td className='text-muted-foreground px-4 py-3'>
                  当前模型支持的请求路径、方法和代码示例；调用前应按所选协议核对
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>支持模型不等于支持所有协议</AlertTitle>
          <AlertDescription className='leading-6'>
            同一个模型可能支持 Chat Completions，但未必支持 Responses、Claude
            Messages 或 Gemini 原生接口。请以模型详情中的接口类型和 API
            示例为准。
          </AlertDescription>
        </Alert>
      </section>

      <section id='billing-modes' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>三种常见计费方式</h2>

        <div className='mt-5'>
          <h3 className='text-lg font-semibold'>按 Token 计费</h3>
          <p className='text-muted-foreground mt-2 leading-7'>
            按实际可计费的输入和输出 Token
            计算，输入与输出通常不是同一个单价。请求还可能包含单独计价的项目，例如：
          </p>
          <ul className='text-muted-foreground marker:text-foreground/40 mt-3 list-disc space-y-2 pl-6 leading-7'>
            <li>缓存读取和缓存写入；</li>
            <li>图片或音频输入、音频输出；</li>
            <li>网页搜索、文件搜索、图片生成等工具调用；</li>
            <li>其他在模型详情或请求日志中明确列出的附加用量。</li>
          </ul>
          <p className='text-muted-foreground mt-3 leading-7'>
            因此，只用“总 Token ×
            一个单价”往往无法精确复算多模态、缓存或工具调用请求。
          </p>
        </div>

        <div className='mt-7'>
          <h3 className='text-lg font-semibold'>按次计费</h3>
          <p className='text-muted-foreground mt-2 leading-7'>
            按次模型使用页面展示的基础单价结算。这里的“一次”需要结合具体接口理解，可能是一轮生成，也可能继续受到图片数量、时长、分辨率、质量等请求参数影响。发送批量图片、长视频或高规格任务前，应先用小任务测试并查看日志。
          </p>
        </div>

        <div className='mt-7'>
          <h3 className='text-lg font-semibold'>动态或阶梯计费</h3>
          <p className='text-muted-foreground mt-2 leading-7'>
            部分模型会根据请求或响应属性匹配不同阶梯，也可能在满足特定条件时追加倍率。模型广场会在能够解析规则时展示阶梯表和条件说明。此类模型不要只看列表中的单个摘要价格，应打开详情查看完整规则，并以请求日志中的匹配结果和总消耗为准。
          </p>
        </div>
      </section>

      <section id='groups-channels-protocols' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          分组、渠道和协议为什么会影响价格
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          同一个模型可能同时出现在多个分组中，不同分组的倍率和可用接口可能不同：
        </p>
        <ul className='text-muted-foreground marker:text-foreground/40 mt-4 list-disc space-y-2 pl-6 leading-7'>
          <li>
            <strong className='text-foreground'>固定分组密钥：</strong>
            请求按该密钥所选分组路由和计费。
          </li>
          <li>
            <strong className='text-foreground'>多候选分组密钥：</strong>
            系统按候选顺序尝试；启用跨分组重试时，最终按实际成功处理请求的分组计费。
          </li>
          <li>
            <strong className='text-foreground'>自动分组：</strong>
            实际分组可能随当时可用路由变化，应在日志中确认。
          </li>
        </ul>
        <div className='text-muted-foreground mt-4 space-y-3 leading-7'>
          <p>
            多个候选分组共用该 API
            密钥的额度限制，不会为每个候选分组单独生成一份额度。模型广场中的“最低可用价格”也不保证就是多候选请求最终命中的价格。
          </p>
          <p>
            渠道本身不一定以“渠道加价”的形式单独展示，但它会影响请求最终由哪个分组和上游能力处理、是否发生模型映射，以及上游返回了哪些
            Token、缓存或多模态用量。协议也会决定可用参数和计费用量的结构，例如
            Chat、Responses、Claude、Gemini、图片和视频接口记录的用量并不完全相同。
          </p>
          <p>
            如果日志显示发生了模型映射，应分别关注“请求模型”和“实际模型”。遇到同名模型、不同后缀或不同接口时，不要假设价格和能力完全一致。
          </p>
        </div>
      </section>

      <section id='balance-and-quota' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>余额、订阅额度和密钥额度</h2>
        <div className='text-muted-foreground mt-3 space-y-3 leading-7'>
          <p>
            请求可能从钱包余额或有效订阅额度中扣除，具体取决于账户当前可用的计费方式和偏好。钱包余额与订阅额度可能遵循不同规则，不能互相当作同一种余额；使用日志会标明本次请求的计费来源。
          </p>
          <p>
            API
            密钥还可以设置自身的额度限制。它用于限制该密钥可消耗的额度，并不改变模型单价、分组倍率或账户资金来源。额度不足时，请依次检查：
          </p>
        </div>
        <NumberedSteps items={quotaCheckSteps} />
        <p className='text-muted-foreground mt-5 leading-7'>
          普通同步请求通常会在完成后按实际用量结算。图片、视频等异步任务可能在提交时先占用额度，任务完成、失败或退款后再更新最终结果，因此短时间内看到的余额变化不一定就是最终消耗。
        </p>
        <Button
          className='mt-5'
          variant='outline'
          render={<Link to='/wallet' />}
        >
          查看钱包和订阅
          <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
        </Button>
      </section>

      <section id='verify-charge' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>用使用日志核对真实消耗</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          正式接入前，建议先进行一次小额、非流式测试：
        </p>
        <NumberedSteps items={verifyChargeSteps} />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>日志是最终核对依据</AlertTitle>
          <AlertDescription className='leading-6'>
            日志中的“费用/总消耗”是该请求的最终核对依据。向客服反馈计费问题时，建议提供请求时间、模型
            ID 和 Request ID；不要发送完整 API 密钥。
          </AlertDescription>
        </Alert>
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button variant='outline' render={<Link to='/keys' />}>
            打开 API 密钥
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button variant='outline' render={<Link to='/usage-logs' />}>
            打开使用日志
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>
      </section>

      <section id='faq' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常见问题</h2>
        <div className='mt-5 space-y-7'>
          <div>
            <h3 className='text-lg font-semibold'>
              模型广场能看到模型，为什么调用仍提示不可用？
            </h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              先检查 API
              密钥的固定分组或候选分组，再确认请求使用的接口类型是否在模型详情中列出。模型存在于广场，不代表当前密钥的每个分组和每种协议都能调用。
            </p>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>
              实际扣费为什么高于列表中的价格？
            </h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              常见原因包括：列表展示了可用分组中的较低价格，而请求由另一个分组成功处理；输出
              Token
              较多；产生了缓存写入、图片、音频或工具调用费用；命中了动态阶梯或附加倍率；异步任务按数量、时长或质量结算。请展开对应日志的计费详情逐项核对。
            </p>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>价格会变化吗？</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              会。模型定价、分组倍率、计费汇率、动态规则和可用渠道都可能调整。历史请求不会用当前页面价格重新展示，已发生的费用应以当时生成的使用日志为准。
            </p>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>
              为什么找不到对应的使用日志？
            </h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              确认请求的 Base URL 指向本站、使用的是预期 API
              密钥，并检查日志的时间范围和类型筛选。如果客户端在请求到达本站前就失败，站内不会生成对应的模型使用记录。
            </p>
          </div>
        </div>
      </section>
    </DocsShell>
  )
}
