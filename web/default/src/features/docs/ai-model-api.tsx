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

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const DOC_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const API_TOC = [
  { id: 'prepare', label: '1. 准备信息' },
  { id: 'model-id', label: '2. 获取模型 ID' },
  { id: 'choose-endpoint', label: '3. 选择接口' },
  { id: 'first-request', label: '4. 第一次请求' },
  { id: 'authentication', label: '认证头' },
  { id: 'details', label: '详细说明' },
]

export function DocsApiIntegration() {
  const baseUrl = useDocsBaseUrl()
  const openAiBaseUrl = `${baseUrl}/v1`
  const openAiModels = `curl "${openAiBaseUrl}/models" \\
  -H "Authorization: Bearer sk-your-api-key"`
  const geminiModels = `curl "${baseUrl}/v1beta/models" \\
  -H "x-goog-api-key: sk-your-api-key"`
  const openAiModelExample = `{
  "id": "your-model-id",
  "object": "model"
}`
  const firstRequest = `curl "${openAiBaseUrl}/chat/completions" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "messages": [{"role": "user", "content": "你好"}]
  }'`
  const pythonSdk = `from openai import OpenAI

client = OpenAI(
    api_key="sk-your-api-key",
    base_url="${openAiBaseUrl}",
)
response = client.chat.completions.create(
    model="your-model-id",
    messages=[{"role": "user", "content": "你好"}],
)
print(response.choices[0].message.content)`

  return (
    <DocsShell
      pageId='api-integration'
      title='API 模型接口'
      description='先用模型列表确认可用模型，再按协议发送第一条请求。完整参数以对应官方 API 文档为准。'
      toc={API_TOC}
    >
      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 准备信息</h2>
        <ul className='text-muted-foreground mt-4 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>在 API Key 页面创建一枚密钥。</li>
          <li>从对应协议的模型列表接口复制模型 ID。</li>
          <li>按客户端要求填写 Base URL。</li>
        </ul>
        <p className='text-muted-foreground mt-4 leading-7'>
          API Key
          只放在服务端配置、受信任的本地客户端私有配置文件或凭据存储中，不要提交到代码仓库、网页前端或公开日志。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={['客户端', 'Base URL']}
            rows={[
              {
                key: 'openai-client',
                cells: [
                  'OpenAI SDK、Cursor 等兼容客户端',
                  <code key='openai-base-url'>{openAiBaseUrl}</code>,
                ],
              },
              {
                key: 'claude-gemini-cli',
                cells: [
                  'Claude Code、Gemini CLI',
                  <code key='root-base-url'>{baseUrl}</code>,
                ],
              },
              {
                key: 'http-client',
                cells: [
                  '直接发送 HTTP 请求',
                  <code key='http-base-url'>{baseUrl}</code>,
                ],
              },
            ]}
          />
        </div>
      </section>

      <section id='model-id' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 获取模型 ID</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          OpenAI 兼容接口和 Claude 使用 <code>/v1/models</code>；Gemini
          原生接口使用 <code>/v1beta/models</code>。不要把 Gemini 的{' '}
          <code>x-goog-api-key</code> 请求发到 <code>/v1/models</code>。
        </p>
        <div className='mt-5 grid gap-5 lg:grid-cols-2'>
          <CodeBlock code={openAiModels} label='OpenAI / Claude' />
          <CodeBlock code={geminiModels} label='Gemini 原生接口' />
        </div>
        <p className='text-muted-foreground mt-5 leading-7'>
          OpenAI 兼容格式复制响应中的 <code>data[].id</code>，例如：
        </p>
        <div className='mt-4'>
          <CodeBlock code={openAiModelExample} label='JSON' />
        </div>
        <ul className='text-muted-foreground mt-5 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>
            Gemini 格式复制响应中的 <code>models[].name</code>
            ，调用时使用其中的模型 ID。
          </li>
          <li>
            OpenAI 格式还会返回 <code>supported_endpoint_types</code>
            ，接口报能力不匹配时先检查该字段。
          </li>
        </ul>
        <p className='text-muted-foreground mt-4 leading-7'>
          模型 ID 必须完全一致。不同 API Key 返回的模型可能不同。
        </p>
      </section>

      <section id='choose-endpoint' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 选择接口</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          本站只补充地址、鉴权和最小请求。字段含义、完整参数和响应结构请以官方文档为准。
        </p>
        <div className='mt-5'>
          <DocsTable
            minWidth='lg'
            headers={['用途', '本站接口', '官方文档']}
            rows={[
              {
                key: 'chat',
                cells: [
                  '普通聊天',
                  <code key='chat-path'>POST /v1/chat/completions</code>,
                  <a
                    key='chat-link'
                    className={DOC_LINK_CLASS}
                    href='https://platform.openai.com/docs/api-reference/chat'
                    target='_blank'
                    rel='noreferrer'
                  >
                    OpenAI Chat Completions
                  </a>,
                ],
              },
              {
                key: 'responses',
                cells: [
                  '推理、工具调用、Codex',
                  <code key='responses-path'>POST /v1/responses</code>,
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
                  'Claude 协议',
                  <code key='claude-path'>POST /v1/messages</code>,
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
                  'Gemini 原生协议',
                  <code key='gemini-path'>
                    {'POST /v1beta/models/{model}:generateContent'}
                  </code>,
                  <a
                    key='gemini-link'
                    className={DOC_LINK_CLASS}
                    href='https://ai.google.dev/api/generate-content'
                    target='_blank'
                    rel='noreferrer'
                  >
                    Gemini Generate Content
                  </a>,
                ],
              },
              {
                key: 'images',
                cells: [
                  '图片',
                  <code key='image-path'>POST /v1/images/generations</code>,
                  <a
                    key='image-link'
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
                  '音频（暂无模型）',
                  <code key='audio-path'>POST /v1/audio/speech</code>,
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
                key: 'video',
                cells: [
                  '视频（暂无模型）',
                  <code key='video-path'>POST /v1/videos</code>,
                  <a
                    key='video-link'
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

      <section id='first-request' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 第一次请求</h2>
        <div className='mt-5'>
          <CodeBlock code={firstRequest} label='cURL' />
        </div>
        <div className='mt-5'>
          <CodeBlock code={pythonSdk} label='Python OpenAI SDK' />
        </div>
      </section>

      <section id='authentication' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>认证头</h2>
        <DocsTable
          headers={['协议', '请求头']}
          rows={[
            {
              key: 'openai-auth',
              cells: [
                'OpenAI 兼容接口',
                <code key='openai-auth-value'>
                  Authorization: Bearer sk-your-api-key
                </code>,
              ],
            },
            {
              key: 'anthropic-auth',
              cells: [
                'Anthropic',
                <span key='anthropic-auth-value'>
                  <code>x-api-key: sk-your-api-key</code>、
                  <code>anthropic-version: 2023-06-01</code>
                </span>,
              ],
            },
            {
              key: 'gemini-auth',
              cells: [
                'Gemini',
                <code key='gemini-auth-value'>
                  x-goog-api-key: sk-your-api-key
                </code>,
              ],
            },
          ]}
        />
      </section>

      <section id='details' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>详细说明</h2>
        <ul className='mt-4 list-disc space-y-2 ps-6 leading-7'>
          <li>
            <Link className={DOC_LINK_CLASS} to='/docs/api/text-chat'>
              文本与对话
            </Link>
          </li>
          <li>
            <Link className={DOC_LINK_CLASS} to='/docs/api/multimodal'>
              多模态接口
            </Link>
          </li>
          <li>
            <Link className={DOC_LINK_CLASS} to='/docs/api/compatibility'>
              兼容性与限制
            </Link>
          </li>
        </ul>
      </section>
    </DocsShell>
  )
}
