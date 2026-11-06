/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const DOC_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const COMPATIBILITY_TOC = [
  { id: 'supported-formats', label: '支持的三种请求格式' },
  { id: 'request-examples', label: '标准请求示例' },
]

export function DocsApiCompatibility() {
  const baseUrl = useDocsBaseUrl()
  const chatRequest = `curl "${baseUrl}/v1/chat/completions" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'`
  const responsesRequest = `curl "${baseUrl}/v1/responses" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "input": "你好"
  }'`
  const claudeRequest = `curl "${baseUrl}/v1/messages" \\
  -H "x-api-key: sk-your-api-key" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'`

  return (
    <DocsShell
      pageId='api-compatibility'
      title='接口选择与兼容性'
      description='同一个 API Key 和网关地址，支持客户端使用三种主流文本请求格式；对于已配置且兼容的上游，new-api 会在服务端完成协议转换。'
      toc={COMPATIBILITY_TOC}
    >
      <section id='supported-formats' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>支持的三种请求格式</h2>
        <p className='mt-3 leading-7'>
          客户端可以使用下面任意一种格式发起请求。客户端协议与上游协议不同时，new-api
          会按模型和渠道配置转换请求、响应以及流式事件。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={[
              '请求格式',
              '请求路径',
              '核心请求字段',
              '官方接口文档',
              '常见客户端',
            ]}
            rows={[
              {
                key: 'chat',
                cells: [
                  'OpenAI Chat Completions',
                  <code key='path'>POST /v1/chat/completions</code>,
                  <code key='field'>messages</code>,
                  <a
                    key='docs'
                    href='https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create'
                    target='_blank'
                    rel='noreferrer'
                    className={DOC_LINK_CLASS}
                  >
                    OpenAI Chat Completions API Reference
                  </a>,
                  'OpenAI SDK、DeepSeek 兼容客户端及其他 OpenAI Chat 客户端',
                ],
              },
              {
                key: 'responses',
                cells: [
                  'OpenAI Responses',
                  <code key='path'>POST /v1/responses</code>,
                  <code key='field'>input</code>,
                  <a
                    key='docs'
                    href='https://developers.openai.com/api/reference/resources/responses/methods/create'
                    target='_blank'
                    rel='noreferrer'
                    className={DOC_LINK_CLASS}
                  >
                    OpenAI Responses API Reference
                  </a>,
                  'Codex 及使用 OpenAI Responses API 的客户端',
                ],
              },
              {
                key: 'claude',
                cells: [
                  'Anthropic Messages',
                  <code key='path'>POST /v1/messages</code>,
                  <>
                    <code>messages</code>、<code>max_tokens</code>
                  </>,
                  <a
                    key='docs'
                    href='https://docs.anthropic.com/en/api/messages'
                    target='_blank'
                    rel='noreferrer'
                    className={DOC_LINK_CLASS}
                  >
                    Anthropic Messages API Reference
                  </a>,
                  'Claude Code、Anthropic SDK 及兼容客户端',
                ],
              },
            ]}
            minWidth='lg'
          />
        </div>
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>客户端不需要关心上游协议</AlertTitle>
          <AlertDescription>
            例如，Codex 可以继续发送 Responses 请求调用 Claude 或 OpenAI Chat
            模型；Claude Code 也可以继续发送 Messages 请求调用 OpenAI Chat
            模型。转换能力取决于当前模型和渠道是否支持并配置了对应路由，不代表每个上游模型都支持全部格式。
          </AlertDescription>
        </Alert>
      </section>

      <section id='request-examples' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>标准请求示例</h2>

        <h3 className='mt-6 text-xl font-semibold'>
          1. OpenAI Chat Completions
        </h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          使用 <code>messages</code> 表示对话历史。
        </p>
        <div className='mt-4'>
          <CodeBlock code={chatRequest} label='cURL' />
        </div>

        <h3 className='mt-8 text-xl font-semibold'>2. OpenAI Responses</h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          使用 <code>input</code> 表示用户输入，Codex 默认使用这一类接口。
        </p>
        <div className='mt-4'>
          <CodeBlock code={responsesRequest} label='cURL' />
        </div>

        <h3 className='mt-8 text-xl font-semibold'>3. Anthropic Messages</h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          使用 <code>messages</code> 和 <code>max_tokens</code>，并携带{' '}
          <code>anthropic-version</code> 请求头。
        </p>
        <div className='mt-4'>
          <CodeBlock code={claudeRequest} label='cURL' />
        </div>

        <p className='text-muted-foreground mt-6 leading-7'>
          将示例中的 <code>sk-your-api-key</code> 和 <code>your-model-id</code>{' '}
          替换为实际 API Key 与模型 ID。模型 ID 以当前密钥请求{' '}
          <code>GET /v1/models</code> 返回的列表为准。
        </p>
      </section>
    </DocsShell>
  )
}
