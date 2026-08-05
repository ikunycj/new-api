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
import { Key01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { NumberedSteps } from './components/numbered-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

export function DocsApiIntegration() {
  const baseUrl = useDocsBaseUrl()
  const serviceAddress = `${baseUrl}`
  const listModels = `curl "${baseUrl}/v1/models" \\
  -H "Authorization: Bearer sk-your-api-key"`

  return (
    <DocsShell
      pageId='api-integration'
      title='API 模型接口'
      description='All Token API 的主要模型接口、认证方式和协议选择入口。'
      toc={[
        { id: 'service-address', label: '服务地址' },
        { id: 'authentication', label: '认证方式' },
        { id: 'choose-endpoint', label: '选择正确的接口' },
        { id: 'status-codes', label: '常见状态码' },
      ]}
    >
      <section id='service-address' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>服务地址</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          以下示例使用当前站点地址。不同客户端对 Base URL 的要求不同：OpenAI
          兼容 SDK 通常使用带
          <code className='mx-1'>/v1</code>
          的地址，Claude Code 和 Gemini CLI 通常使用不带版本路径的服务根地址。
        </p>
        <div className='mt-5'>
          <CodeBlock code={serviceAddress} label='服务根地址' />
        </div>
      </section>

      <section id='authentication' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>认证方式</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          不同协议使用不同的认证请求头。API Key
          只应保存在服务端、受信任的本地客户端或安全的环境变量中。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={['协议', '认证方式']}
            rows={[
              {
                key: 'openai',
                cells: [
                  'OpenAI 兼容接口',
                  <code key='openai-auth'>
                    Authorization: Bearer sk-your-api-key
                  </code>,
                ],
              },
              {
                key: 'anthropic',
                cells: [
                  'Anthropic Messages',
                  <span key='anthropic-auth'>
                    <code>x-api-key: sk-your-api-key</code>，并提供{' '}
                    <code>anthropic-version</code>
                  </span>,
                ],
              },
              {
                key: 'gemini',
                cells: [
                  'Gemini 原生接口',
                  <code key='gemini-auth'>
                    x-goog-api-key: sk-your-api-key
                  </code>,
                ],
              },
            ]}
          />
        </div>
        <Alert className='mt-5'>
          <HugeiconsIcon icon={Key01Icon} aria-hidden='true' />
          <AlertTitle>先用当前密钥获取模型</AlertTitle>
          <AlertDescription>
            不同 API Key
            的分组和模型限制可能不同。使用生产密钥请求模型列表，才能看到它实际可调用的范围。
          </AlertDescription>
        </Alert>
        <div className='mt-5'>
          <CodeBlock code={listModels} label='cURL' />
        </div>
      </section>

      <section id='choose-endpoint' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>选择正确的接口</h2>
        <NumberedSteps
          items={[
            '使用当前 API Key 请求模型列表。',
            '在模型定价页确认模型支持的端点类型。',
            '使用与端点匹配的请求格式，不要仅根据模型名称推测兼容性。',
            '先发送小型非流式请求，再启用流式输出、工具调用或多模态内容。',
          ]}
        />
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button nativeButton={false} render={<Link to='/keys' />}>
            打开 API 密钥
          </Button>
          <Button
            nativeButton={false}
            variant='outline'
            render={<Link to='/pricing' />}
          >
            打开模型定价
          </Button>
        </div>
      </section>

      <section id='status-codes' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常见状态码</h2>
        <div className='mt-5'>
          <DocsTable
            headers={['状态码', '含义']}
            rows={[
              {
                key: '400',
                cells: [
                  <code key='400'>400</code>,
                  '请求格式、参数或模型不正确',
                ],
              },
              {
                key: '401',
                cells: [<code key='401'>401</code>, 'API Key 缺失或无效'],
              },
              {
                key: '404',
                cells: [<code key='404'>404</code>, '接口、模型或任务不存在'],
              },
              {
                key: '429',
                cells: [
                  <code key='429'>429</code>,
                  '触发速率限制或可用额度不足',
                ],
              },
              {
                key: '5xx',
                cells: [
                  <code key='5xx'>5xx</code>,
                  '中转站或上游模型服务未能完成请求',
                ],
              },
            ]}
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          排查失败请求时，请记录请求时间、HTTP 状态码、错误消息、Request
          ID、模型和端点，并在使用日志中核对。
        </p>
      </section>
    </DocsShell>
  )
}
