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
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const DOC_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const TEXT_CHAT_TOC = [
  { id: 'chat-completions', label: 'Chat Completions' },
  { id: 'responses', label: 'Responses' },
  { id: 'claude-messages', label: 'Claude Messages' },
  { id: 'gemini', label: 'Gemini 原生对话' },
  { id: 'completions', label: '传统 Completions' },
]

export function DocsApiTextChat() {
  const baseUrl = useDocsBaseUrl()
  const openAiBaseUrl = `${baseUrl}/v1`
  const chatRequest = `curl "${openAiBaseUrl}/chat/completions" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "messages": [{"role":"user","content":"你好"}]
  }'`
  const responsesRequest = `curl "${openAiBaseUrl}/responses" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "input": "解释一下 API 网关",
    "instructions": "用中文简短回答"
  }'`
  const claudeRequest = `curl "${openAiBaseUrl}/messages" \\
  -H "x-api-key: sk-your-api-key" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-claude-model-id",
    "max_tokens": 512,
    "messages": [{"role":"user","content":"你好"}]
  }'`
  const geminiRequest = `curl "${baseUrl}/v1beta/models/your-gemini-model-id:generateContent" \\
  -H "x-goog-api-key: sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [{"role":"user","parts":[{"text":"你好"}]}]
  }'`
  const completionRequest = `curl "${openAiBaseUrl}/completions" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"your-model-id","prompt":"Once upon a time","max_tokens":64}'`
  const claudeCodeConfig = `{
  "env": {
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_AUTH_TOKEN": "sk-your-api-key"
  }
}`

  return (
    <DocsShell
      pageId='api-text-chat'
      title='文本与对话'
      description='按请求协议选择 Chat、Responses、Claude Messages 或 Gemini 原生接口，并复制最小请求。'
      toc={TEXT_CHAT_TOC}
    >
      <section id='chat-completions' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Chat Completions</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/chat'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Chat Completions API
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          请求路径为 <code>POST /v1/chat/completions</code>，必填字段是{' '}
          <code>model</code> 和 <code>messages</code>。
        </p>
        <div className='mt-5'>
          <CodeBlock code={chatRequest} label='cURL' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          常用参数包括 <code>temperature</code>、<code>top_p</code>、{' '}
          <code>max_tokens</code>、<code>max_completion_tokens</code>、{' '}
          <code>stream</code>、<code>tools</code>、<code>tool_choice</code> 和{' '}
          <code>response_format</code>。流式请求将 <code>stream</code> 设置为{' '}
          <code>true</code>，并使用支持 SSE 的客户端读取响应。
        </p>
      </section>

      <section id='responses' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Responses</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/responses'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Responses API
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          请求路径为 <code>POST /v1/responses</code>，必填字段是{' '}
          <code>model</code> 和 <code>input</code>。Responses 与 Chat
          Completions 的请求体不通用，不要发送 <code>messages</code> 字段。
        </p>
        <div className='mt-5'>
          <CodeBlock code={responsesRequest} label='cURL' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          常用参数包括 <code>instructions</code>、<code>max_output_tokens</code>
          、 <code>stream</code>、<code>tools</code>、<code>tool_choice</code>、{' '}
          <code>reasoning</code> 和 <code>previous_response_id</code>
          。请先确认模型列表中的端点类型包含 Responses。
        </p>
        <p className='text-muted-foreground mt-4 leading-7'>
          <strong className='text-foreground'>Responses Compact：</strong>
          请求路径为 <code>POST /v1/responses/compact</code>
          ，仅对模型列表中明确支持该端点的 OpenAI/Codex 模型使用。
        </p>
      </section>

      <section id='claude-messages' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Claude Messages</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://docs.anthropic.com/en/api/messages'
            target='_blank'
            rel='noreferrer'
          >
            Anthropic Messages API
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          请求路径为 <code>POST /v1/messages</code>，必填字段是{' '}
          <code>model</code>、 <code>max_tokens</code> 和 <code>messages</code>
          。鉴权使用 <code>x-api-key</code>，并提供{' '}
          <code>anthropic-version: 2023-06-01</code>。
        </p>
        <div className='mt-5'>
          <CodeBlock code={claudeRequest} label='cURL' />
        </div>
        <div className='mt-5'>
          <CodeBlock
            code={claudeCodeConfig}
            label='Claude Code settings.json'
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          将该片段合并到用户级 <code>~/.claude/settings.json</code>
          （Windows 对应用户配置目录）。Claude Code 使用服务根地址，不要把{' '}
          <code>/v1</code> 拼到 <code>ANTHROPIC_BASE_URL</code> 后面；All Token
          API 的 Claude Code 配置使用 <code>ANTHROPIC_AUTH_TOKEN</code>，对应
          Bearer Token。
        </p>
      </section>

      <section id='gemini' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Gemini 原生对话</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://ai.google.dev/api/generate-content'
            target='_blank'
            rel='noreferrer'
          >
            Gemini Generate Content
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          请求路径为{' '}
          <code>POST /v1beta/models/{'{model}'}:generateContent</code>
          ，必填字段是 <code>contents</code>。使用 <code>x-goog-api-key</code>{' '}
          鉴权，不能把 OpenAI 的 <code>messages</code> 请求体直接发送到 Gemini
          路径。
        </p>
        <div className='mt-5'>
          <CodeBlock code={geminiRequest} label='cURL' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          流式调用将动作改为 <code>streamGenerateContent</code>，通常添加{' '}
          <code>?alt=sse</code>。
        </p>
      </section>

      <section id='completions' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>传统 Completions</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/completions'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Completions API
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          请求路径为 <code>POST /v1/completions</code>，必填字段是{' '}
          <code>model</code> 和 <code>prompt</code>。新模型优先使用 Chat 或
          Responses。
        </p>
        <div className='mt-5'>
          <CodeBlock code={completionRequest} label='cURL' />
        </div>
        <Alert className='mt-5'>
          <AlertTitle>不要混用协议请求体</AlertTitle>
          <AlertDescription>
            Chat 使用 <code>messages</code>，Responses 使用 <code>input</code>
            ，Claude 使用 <code>max_tokens</code> 和 <code>messages</code>
            ，Gemini 原生接口使用 <code>contents</code>。模型 ID
            相同也不能省略协议对应的请求格式。
          </AlertDescription>
        </Alert>
        <p className='text-muted-foreground mt-5 leading-7'>
          继续查看{' '}
          <Link className={DOC_LINK_CLASS} to='/docs/api/multimodal'>
            多模态接口
          </Link>{' '}
          和{' '}
          <Link className={DOC_LINK_CLASS} to='/docs/api/compatibility'>
            兼容性与限制
          </Link>
          。
        </p>
      </section>
    </DocsShell>
  )
}
