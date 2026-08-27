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
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { CodeBlock } from './components/code-block'
import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { GuideSteps } from './components/guide-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const OPENCODE_TOC: DocsTocItem[] = [
  { id: 'scope', label: '1. 选择配置范围' },
  { id: 'prepare', label: '2. 准备模型和接口类型' },
  { id: 'config', label: '3. 在 opencode.json 中直接保存密钥' },
  { id: 'verify', label: '4. 选择并验证模型' },
  { id: 'troubleshooting', label: '5. 常见问题' },
  { id: 'references', label: '6. 官方参考' },
]

const OPENCODE_REFERENCES = [
  ['OpenCode 服务商配置', 'https://opencode.ai/docs/providers/'],
  ['OpenCode 配置文件', 'https://opencode.ai/docs/config/'],
  ['OpenCode 官方仓库', 'https://github.com/anomalyco/opencode'],
] as const

const DOCS_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

export function DocsOpenCode() {
  const baseUrl = useDocsBaseUrl()
  const providerConfig = `{
  "$schema": "https://opencode.ai/config.json",
  "model": "alltokenapi/此处替换为准确的模型 ID",
  "small_model": "alltokenapi/此处替换为准确的模型 ID",
  "provider": {
    "alltokenapi": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "All Token API",
      "options": {
        "baseURL": "${baseUrl}/v1",
        "apiKey": "此处替换为 API Key"
      },
      "models": {
        "此处替换为准确的模型 ID": {
          "name": "此处替换为准确的模型 ID"
        }
      }
    }
  }
}`

  return (
    <DocsShell
      pageId='opencode'
      title='OpenCode'
      description='在 OpenCode 中保存本服务凭据，并将其注册为自定义 OpenAI 兼容服务商。'
      toc={OPENCODE_TOC}
    >
      <Alert>
        <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
        <AlertTitle>配置核验</AlertTitle>
        <AlertDescription>
          本文主流程将 API Key 直接写入用户级
          opencode.json，因此关闭终端后仍然有效。也可以使用 OpenCode 的 /connect
          保存凭据，但不要同时依赖两种来源。
        </AlertDescription>
      </Alert>

      <section id='scope' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 选择配置范围</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          OpenCode 支持 JSON 和
          JSONC，多个配置源会合并，后加载的配置覆盖冲突字段。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={['范围', '配置文件']}
            rows={[
              {
                key: 'global',
                cells: [
                  '全局',
                  <code key='path'>~/.config/opencode/opencode.json</code>,
                ],
              },
              {
                key: 'project',
                cells: [
                  '项目',
                  <span key='path'>
                    项目根目录的 <code>opencode.json</code>
                  </span>,
                ],
              },
              {
                key: 'custom',
                cells: [
                  '自定义文件',
                  <span key='path'>
                    <code>OPENCODE_CONFIG</code> 指向的文件
                  </span>,
                ],
              },
            ]}
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          服务商通常适合放在全局配置。项目配置可以提交到
          Git，因此不要在其中写明文密钥。
        </p>
      </section>

      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 准备模型和接口类型</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  在{' '}
                  <Link to='/keys' className={DOCS_LINK_CLASS}>
                    API Key 页面
                  </Link>{' '}
                  创建密钥。
                </>
              ),
            },
            {
              content: (
                <>
                  在{' '}
                  <Link to='/pricing' className={DOCS_LINK_CLASS}>
                    模型定价页面
                  </Link>{' '}
                  复制准确模型 ID。
                </>
              ),
            },
            {
              content: '确认模型使用 Chat Completions 还是 Responses 接口。',
            },
          ]}
        />
        <div className='mt-6'>
          <DocsTable
            headers={['模型接口', 'npm 适配器', 'Base URL']}
            rows={[
              {
                key: 'chat-completions',
                cells: [
                  <code key='endpoint'>/v1/chat/completions</code>,
                  <code key='adapter'>@ai-sdk/openai-compatible</code>,
                  <code key='url'>{baseUrl}/v1</code>,
                ],
              },
              {
                key: 'responses',
                cells: [
                  <code key='endpoint'>/v1/responses</code>,
                  <code key='adapter'>@ai-sdk/openai</code>,
                  <code key='url'>{baseUrl}/v1</code>,
                ],
              },
            ]}
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          以下主教程按 Chat Completions 编写。Responses
          模型必须切换适配器，不能只修改模型 ID。
        </p>
      </section>

      <section id='config' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          3. 在 opencode.json 中直接保存密钥
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          在全局配置文件 ~/.config/opencode/opencode.json（Windows
          对应用户配置目录）中合并以下内容。不要把含有明文密钥的项目级
          opencode.json 提交到 Git。
        </p>
        <div className='mt-5'>
          <CodeBlock code={providerConfig} label='opencode.json' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          需要替换四处模型 ID。small_model
          用同一模型可避免标题生成等轻量任务落到其他服务商；有更便宜且兼容的模型时，可以单独替换。
        </p>
        <h3 className='mt-8 text-lg font-semibold'>Responses 模型</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          如果模型明确使用 /v1/responses，只修改 npm 适配器；其余 Provider
          ID、Base URL 和模型引用保持不变。
        </p>
        <div className='mt-4'>
          <CodeBlock code='"npm": "@ai-sdk/openai"' label='opencode.json' />
        </div>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 选择并验证模型</h2>
        <GuideSteps
          items={[
            { content: '启动 opencode。' },
            { content: '在 TUI 中输入 /models，选择 alltokenapi/模型ID。' },
            { content: '发送一个简短提示，或执行一次非交互测试。' },
            {
              content: (
                <>
                  在{' '}
                  <Link to='/usage-logs' className={DOCS_LINK_CLASS}>
                    使用日志
                  </Link>{' '}
                  确认请求模型、状态和费用。
                </>
              ),
            },
          ]}
        />
        <div className='mt-5'>
          <CodeBlock code='opencode run "只回复 OK"' label='非交互测试' />
        </div>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>5. 常见问题</h2>
        <div className='mt-5 space-y-7'>
          <div>
            <h3 className='text-lg font-semibold'>
              /models 中没有 alltokenapi
            </h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>检查 opencode.json 的生效范围和 JSON/JSONC 语法。</li>
              <li>
                Provider ID 必须在 model、small_model 和 provider 配置中都写成
                alltokenapi。
              </li>
              <li>模型必须定义在 provider.alltokenapi.models 中。</li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>认证失败</h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>
                确认 provider.alltokenapi.options.apiKey 已替换为有效密钥。
              </li>
              <li>
                如果曾使用 /connect，不要同时保留错误的 auth.json
                凭据；排障时只保留一种来源。
              </li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>
              404、流式输出或工具调用失败
            </h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>Chat Completions 使用 @ai-sdk/openai-compatible。</li>
              <li>Responses 使用 @ai-sdk/openai。</li>
              <li>两种 OpenAI 兼容接口的 Base URL 都应以 /v1 结尾。</li>
              <li>
                普通对话成功但工具调用失败，通常表示模型或接口不完整支持 Tool
                Calling。
              </li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>上下文长度显示不准确</h3>
            <p className='text-muted-foreground mt-3 leading-7'>
              确认准确数值后，可在模型下添加
              limit。不要从模型名称猜测限制；错误值会影响 OpenCode
              的上下文压缩判断。
            </p>
            <div className='mt-4'>
              <CodeBlock
                code={`"limit": {
  "context": 128000,
  "output": 32000
}`}
                label='模型限制示例'
              />
            </div>
          </div>
        </div>
      </section>

      <section id='references' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>6. 官方参考</h2>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          {OPENCODE_REFERENCES.map(([label, href]) => (
            <li key={href}>
              <a
                href={href}
                target='_blank'
                rel='noopener noreferrer'
                className={DOCS_LINK_CLASS}
              >
                {label}
              </a>
            </li>
          ))}
        </ul>
      </section>
    </DocsShell>
  )
}
