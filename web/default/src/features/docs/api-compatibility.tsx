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
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const DOC_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const COMPATIBILITY_TOC = [
  { id: 'model-check', label: '先确认模型' },
  { id: 'protocol', label: '协议与地址' },
  { id: 'official-docs', label: '官方参数文档' },
  { id: 'special-tasks', label: '专用任务路径' },
  { id: 'troubleshooting', label: '常见错误' },
]

export function DocsApiCompatibility() {
  const baseUrl = useDocsBaseUrl()
  const modelsRequest = `curl "${baseUrl}/v1/models" \\
  -H "Authorization: Bearer sk-your-api-key"`

  return (
    <DocsShell
      pageId='api-compatibility'
      title='兼容性与限制'
      description='确认模型能力、协议格式和专用任务路径，减少 Base URL、鉴权和请求体不匹配导致的错误。'
      toc={COMPATIBILITY_TOC}
    >
      <section id='model-check' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>先确认模型</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          只使用当前 API Key 返回的模型
          ID。模型必须支持准备调用的接口；能聊天不代表能生成图片、向量、音频或视频。
        </p>
        <div className='mt-5'>
          <CodeBlock code={modelsRequest} label='OpenAI 格式模型列表' />
        </div>
        <ul className='text-muted-foreground mt-5 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>
            OpenAI 格式读取 <code>data[].id</code> 和{' '}
            <code>supported_endpoint_types</code>。
          </li>
          <li>
            Gemini 原生格式改用 <code>GET /v1beta/models</code> 和{' '}
            <code>x-goog-api-key</code>。
          </li>
          <li>不同 API Key 的分组、模型限制和返回结果可能不同。</li>
        </ul>
      </section>

      <section id='protocol' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>协议与地址</h2>
        <DocsTable
          headers={['场景', '使用方式']}
          rows={[
            {
              key: 'openai-client',
              cells: [
                'OpenAI SDK、Cursor 等兼容客户端',
                <span key='openai-client-value'>
                  Base URL 使用 <code>{baseUrl}/v1</code>，请求头使用{' '}
                  <code>Authorization: Bearer</code>。
                </span>,
              ],
            },
            {
              key: 'claude-code',
              cells: [
                'Claude Code',
                <span key='claude-code-value'>
                  服务根地址使用 <code>{baseUrl}</code>，请求头使用{' '}
                  <code>x-api-key</code>。
                </span>,
              ],
            },
            {
              key: 'gemini-cli',
              cells: [
                'Gemini CLI 或 Gemini 原生 SDK',
                <span key='gemini-cli-value'>
                  服务根地址使用 <code>{baseUrl}</code>，路径以{' '}
                  <code>/v1beta/models</code> 开头。
                </span>,
              ],
            },
            {
              key: 'direct-http',
              cells: [
                '直接发送 HTTP 请求',
                <span key='direct-http-value'>
                  使用服务根地址拼接完整路径，不要重复或遗漏 <code>/v1</code>。
                </span>,
              ],
            },
          ]}
        />
        <Alert className='mt-5'>
          <AlertTitle>请求体也不能混用</AlertTitle>
          <AlertDescription>
            Chat 请求体不能直接发送到 Responses、Claude 或 Gemini
            路径。图片编辑、音频转写、音频翻译和视频创建使用{' '}
            <code>multipart/form-data</code>；Realtime 使用 WebSocket。
          </AlertDescription>
        </Alert>
      </section>

      <section id='official-docs' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>官方参数文档</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          本站文档只说明本站地址和最小请求，完整参数以官方文档为准。
        </p>
        <div className='mt-5'>
          <DocsTable
            minWidth='lg'
            headers={['协议或能力', '官方文档']}
            rows={[
              {
                key: 'chat',
                cells: [
                  'Chat Completions',
                  <a
                    key='chat-link'
                    className={DOC_LINK_CLASS}
                    href='https://platform.openai.com/docs/api-reference/chat'
                    target='_blank'
                    rel='noreferrer'
                  >
                    OpenAI Chat
                  </a>,
                ],
              },
              {
                key: 'responses',
                cells: [
                  'Responses',
                  <a
                    key='responses-link'
                    className={DOC_LINK_CLASS}
                    href='https://platform.openai.com/docs/api-reference/responses'
                    target='_blank'
                    rel='noreferrer'
                  >
                    OpenAI Responses
                  </a>,
                ],
              },
              {
                key: 'claude',
                cells: [
                  'Claude Messages',
                  <a
                    key='claude-link'
                    className={DOC_LINK_CLASS}
                    href='https://docs.anthropic.com/en/api/messages'
                    target='_blank'
                    rel='noreferrer'
                  >
                    Anthropic Messages
                  </a>,
                ],
              },
              {
                key: 'gemini',
                cells: [
                  'Gemini Generate Content',
                  <a
                    key='gemini-link'
                    className={DOC_LINK_CLASS}
                    href='https://ai.google.dev/api/generate-content'
                    target='_blank'
                    rel='noreferrer'
                  >
                    Gemini API
                  </a>,
                ],
              },
              {
                key: 'images',
                cells: [
                  'Images',
                  <a
                    key='images-link'
                    className={DOC_LINK_CLASS}
                    href='https://platform.openai.com/docs/api-reference/images'
                    target='_blank'
                    rel='noreferrer'
                  >
                    OpenAI Images
                  </a>,
                ],
              },
              {
                key: 'audio',
                cells: [
                  'Audio',
                  <a
                    key='audio-link'
                    className={DOC_LINK_CLASS}
                    href='https://platform.openai.com/docs/api-reference/audio'
                    target='_blank'
                    rel='noreferrer'
                  >
                    OpenAI Audio
                  </a>,
                ],
              },
              {
                key: 'videos',
                cells: [
                  'Videos',
                  <a
                    key='videos-link'
                    className={DOC_LINK_CLASS}
                    href='https://platform.openai.com/docs/api-reference/videos'
                    target='_blank'
                    rel='noreferrer'
                  >
                    OpenAI Videos
                  </a>,
                ],
              },
            ]}
          />
        </div>
      </section>

      <section id='special-tasks' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>专用任务路径</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          以下任务接口统一使用{' '}
          <code>Authorization: Bearer sk-your-api-key</code>
          ，请求体按对应服务格式填写，不要发送 OpenAI Chat 请求体。
        </p>
        <div className='mt-5'>
          <DocsTable
            minWidth='lg'
            headers={['服务', '路径']}
            rows={[
              {
                key: 'kling-text',
                cells: [
                  'Kling 文生视频',
                  <code key='kling-text-path'>
                    POST /kling/v1/videos/text2video
                  </code>,
                ],
              },
              {
                key: 'kling-image',
                cells: [
                  'Kling 图生视频',
                  <code key='kling-image-path'>
                    POST /kling/v1/videos/image2video
                  </code>,
                ],
              },
              {
                key: 'midjourney',
                cells: [
                  'Midjourney',
                  <code key='midjourney-path'>
                    POST /mj/submit/{'{action}'}
                  </code>,
                ],
              },
              {
                key: 'suno',
                cells: [
                  'Suno',
                  <code key='suno-path'>POST /suno/submit/{'{action}'}</code>,
                ],
              },
              {
                key: 'jimeng',
                cells: ['即梦', <code key='jimeng-path'>POST /jimeng/</code>],
              },
            ]}
          />
        </div>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常见错误</h2>
        <DocsTable
          headers={['错误', '先检查']}
          rows={[
            {
              key: '401',
              cells: ['401', 'API Key、请求头和 Base URL'],
            },
            {
              key: '400',
              cells: ['400', '模型 ID、路径、必填字段和请求体格式'],
            },
            {
              key: '404',
              cells: ['404', '是否重复或遗漏 /v1，模型或任务 ID 是否正确'],
            },
            {
              key: '403',
              cells: ['403', 'API Key 的模型、IP 和分组权限'],
            },
            {
              key: '429',
              cells: ['429', '余额、额度、速率限制和上游负载'],
            },
          ]}
        />
        <p className='text-muted-foreground mt-5 leading-7'>
          收到 <code>200</code> 但业务结果失败时，继续查看响应体中的{' '}
          <code>error</code> 或任务状态。异步视频、Kling、Midjourney、Suno
          和即梦需要先提交任务，再轮询任务结果。
        </p>
        <p className='text-muted-foreground mt-4 leading-7'>
          继续查看{' '}
          <Link className={DOC_LINK_CLASS} to='/docs/api/text-chat'>
            文本与对话
          </Link>{' '}
          或{' '}
          <Link className={DOC_LINK_CLASS} to='/docs/api/multimodal'>
            多模态接口
          </Link>
          。
        </p>
      </section>
    </DocsShell>
  )
}
