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
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const DOC_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const COMPATIBILITY_TOC = [
  { id: 'model-check', label: '先做这一步' },
  { id: 'choose-endpoint', label: '怎么选接口' },
  { id: 'official-docs', label: '官方参数文档' },
  { id: 'protocol-rules', label: '协议不要混用' },
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
      title='接口选择与兼容性'
      description='先确认模型支持的端点，再使用对应协议、请求路径和请求体。'
      toc={COMPATIBILITY_TOC}
    >
      <section id='model-check' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>先做这一步</h2>
        <div className='mt-5'>
          <CodeBlock code={modelsRequest} label='cURL' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          只使用返回的模型
          ID。模型必须支持准备调用的接口；能聊天不代表能生成图片、向量、音频或视频。
        </p>
      </section>

      <section id='choose-endpoint' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>怎么选接口</h2>
        <DocsTable
          headers={['需求', '接口']}
          rows={[
            { key: 'chat', cells: ['普通文字聊天', 'Chat Completions'] },
            { key: 'responses', cells: ['推理、工具调用、Codex', 'Responses'] },
            { key: 'claude', cells: ['Claude Code', 'Claude Messages'] },
            {
              key: 'gemini',
              cells: ['Gemini CLI 或 Gemini 原生 SDK', 'Gemini 原生接口'],
            },
            { key: 'completions', cells: ['旧版文本续写', 'Completions'] },
            { key: 'images', cells: ['图片生成或编辑', 'Images'] },
            { key: 'videos', cells: ['异步视频', 'Videos'] },
          ]}
        />
        <p className='text-muted-foreground mt-4 leading-7'>
          OpenAI 格式的模型列表会返回 <code>supported_endpoint_types</code>
          。优先选择同时出现在模型列表和请求路径中的能力，不要仅凭模型名称猜测接口支持情况。
        </p>
      </section>

      <section id='official-docs' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>官方参数文档</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          本站只说明本站地址和最小请求，完整参数以官方文档为准。
        </p>
        <DocsTable
          headers={['协议', '官方文档']}
          rows={[
            {
              key: 'chat',
              cells: [
                'Chat',
                <a
                  key='link'
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
                  key='link'
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
                'Claude',
                <a
                  key='link'
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
                'Gemini',
                <a
                  key='link'
                  className={DOC_LINK_CLASS}
                  href='https://ai.google.dev/api/generate-content'
                  target='_blank'
                  rel='noreferrer'
                >
                  Generate Content
                </a>,
              ],
            },
            {
              key: 'images',
              cells: [
                '图片',
                <a
                  key='link'
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
                '音频',
                <a
                  key='link'
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
                '视频',
                <a
                  key='link'
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
      </section>

      <section id='protocol-rules' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>协议不要混用</h2>
        <ul className='text-muted-foreground mt-4 list-disc space-y-2 ps-6 leading-7'>
          <li>Chat 请求体不能直接发送到 Responses、Claude 或 Gemini 路径。</li>
          <li>
            Claude Code 使用服务根地址；OpenAI SDK 通常使用带 /v1 的地址。
          </li>
          <li>
            图片编辑、音频转写、音频翻译和视频创建使用 multipart/form-data。
          </li>
          <li>Realtime 使用 WebSocket。</li>
          <li>
            /v1/responses/compact 只用于明确支持该端点的 OpenAI/Codex 模型。
          </li>
          <li>/v1/edits 当前按图片编辑处理。</li>
        </ul>
      </section>

      <section id='special-tasks' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>专用任务路径</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          以下任务接口统一使用{' '}
          <code>Authorization: Bearer sk-your-api-key</code>
          ，请求体按对应服务格式填写。
        </p>
        <DocsTable
          headers={['服务', '路径']}
          rows={[
            {
              key: 'kling-text',
              cells: [
                'Kling 文生视频',
                <code key='path'>POST /kling/v1/videos/text2video</code>,
              ],
            },
            {
              key: 'kling-image',
              cells: [
                'Kling 图生视频',
                <code key='path'>POST /kling/v1/videos/image2video</code>,
              ],
            },
            {
              key: 'midjourney',
              cells: [
                'Midjourney',
                <code key='path'>{'POST /mj/submit/{action}'}</code>,
              ],
            },
            {
              key: 'suno',
              cells: [
                'Suno',
                <code key='path'>{'POST /suno/submit/{action}'}</code>,
              ],
            },
            {
              key: 'jimeng',
              cells: ['即梦', <code key='path'>POST /jimeng/</code>],
            },
          ]}
        />
        <Alert className='mt-5'>
          <AlertTitle>使用对应服务商的请求格式</AlertTitle>
          <AlertDescription>
            不要向这些接口发送 OpenAI Chat 请求体。
          </AlertDescription>
        </Alert>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>常见错误</h2>
        <DocsTable
          headers={['错误', '先检查']}
          rows={[
            { key: '401', cells: ['401', 'API Key、请求头、Base URL'] },
            {
              key: '400',
              cells: ['400', '模型 ID、路径、必填字段、请求体格式'],
            },
            {
              key: '404',
              cells: ['404', '是否重复或遗漏 /v1，模型或任务 ID 是否正确'],
            },
            { key: '403', cells: ['403', 'API Key 的模型、IP、分组权限'] },
            { key: '429', cells: ['429', '余额、额度、速率限制和上游负载'] },
          ]}
        />
        <p className='text-muted-foreground mt-5 leading-7'>
          收到 200 但业务结果失败时，继续查看响应体中的 error
          或任务状态；异步视频、Kling、Midjourney、Suno
          和即梦需要先提交任务，再轮询任务结果。
        </p>
      </section>
    </DocsShell>
  )
}
