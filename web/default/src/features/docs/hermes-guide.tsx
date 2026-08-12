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
import { InformationCircleIcon, Key01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import { CodeBlock } from './components/code-block'
import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { NumberedSteps } from './components/numbered-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const HERMES_TOC: DocsTocItem[] = [
  { id: 'prepare', label: '1. 准备 API Key、模型和接口类型' },
  { id: 'wizard', label: '2. 推荐：使用 hermes model 向导' },
  { id: 'manual', label: '3. 手动配置' },
  { id: 'verify', label: '4. 验证配置' },
  { id: 'reload', label: '5. 配置何时生效' },
  { id: 'troubleshooting', label: '6. 常见问题' },
  { id: 'references', label: '7. 官方参考' },
]

const HERMES_REFERENCES = [
  [
    'Hermes AI Providers',
    'https://hermes-agent.nousresearch.com/docs/integrations/providers',
  ],
  [
    'Hermes Configuration',
    'https://hermes-agent.nousresearch.com/docs/user-guide/configuration',
  ],
  [
    'Hermes Configuring Models',
    'https://hermes-agent.nousresearch.com/docs/user-guide/configuring-models',
  ],
  [
    'Hermes CLI Commands',
    'https://hermes-agent.nousresearch.com/docs/reference/cli-commands',
  ],
  ['Hermes Agent 官方仓库', 'https://github.com/NousResearch/hermes-agent'],
] as const

const DOCS_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

export function DocsHermes() {
  const baseUrl = useDocsBaseUrl()
  const providerConfig = `providers:
  alltokenapi:
    api: ${baseUrl}/v1
    key_env: ALLTOKEN_API_KEY
    transport: chat_completions
    default_model: 此处替换为准确的模型 ID

model:
  provider: custom:alltokenapi
  default: 此处替换为准确的模型 ID`

  return (
    <DocsShell
      pageId='hermes'
      title='Hermes Agent'
      description='通过 hermes model 向导或 ~/.hermes/config.yaml，将本服务配置为 Hermes 的命名自定义服务商。'
      toc={HERMES_TOC}
    >
      <Alert>
        <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
        <AlertTitle>配置核验</AlertTitle>
        <AlertDescription>
          Hermes 当前推荐使用 providers.&lt;名称&gt; 定义自定义端点，并用
          model.provider: custom:&lt;名称&gt; 选择它。旧版 custom_providers
          列表和 model.base_url/api_mode 仍兼容，但不是本文主配置格式。
        </AlertDescription>
      </Alert>

      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          1. 准备 API Key、模型和接口类型
        </h2>
        <NumberedSteps
          items={[
            '在 API Key 页面创建密钥。',
            '在模型定价页面复制准确的模型 ID。',
            '根据模型实际接口选择 Hermes transport 和 Base URL。',
          ]}
        />
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button render={<Link to='/keys' />}>
            <HugeiconsIcon icon={Key01Icon} data-icon='inline-start' />
            打开 API Key
          </Button>
          <Button variant='outline' render={<Link to='/pricing' />}>
            打开模型定价
          </Button>
        </div>
        <div className='mt-6'>
          <DocsTable
            headers={['模型接口', 'Hermes transport', 'Base URL']}
            rows={[
              {
                key: 'chat-completions',
                cells: [
                  <code key='endpoint'>/v1/chat/completions</code>,
                  <code key='transport'>chat_completions</code>,
                  <code key='url'>{baseUrl}/v1</code>,
                ],
              },
              {
                key: 'responses',
                cells: [
                  <code key='endpoint'>/v1/responses</code>,
                  <code key='transport'>codex_responses</code>,
                  <code key='url'>{baseUrl}/v1</code>,
                ],
              },
              {
                key: 'anthropic-messages',
                cells: [
                  <code key='endpoint'>/v1/messages</code>,
                  <code key='transport'>anthropic_messages</code>,
                  <code key='url'>{baseUrl}</code>,
                ],
              },
            ]}
          />
        </div>
      </section>

      <section id='wizard' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          2. 推荐：使用 hermes model 向导
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          在 Hermes 会话外的系统终端运行完整服务商配置向导：
        </p>
        <div className='mt-5'>
          <CodeBlock code='hermes model' label='终端' />
        </div>
        <NumberedSteps
          items={[
            '选择 Custom endpoint。',
            '输入与接口匹配的 Base URL。',
            '输入 API Key 和准确模型 ID。',
            '选择 API compatibility mode：Chat Completions 使用 chat_completions，Responses 使用 codex_responses，Anthropic Messages 使用 anthropic_messages。',
            '保存并退出向导。',
          ]}
        />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>向导与会话命令的区别</AlertTitle>
          <AlertDescription>
            hermes model 是完整的服务商配置向导；会话内的 /model
            只能切换已经配置好的服务商，不能新增端点或录入密钥。
          </AlertDescription>
        </Alert>
      </section>

      <section id='manual' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 手动配置</h2>
        <h3 className='mt-5 text-lg font-semibold'>3.1 保存密钥</h3>
        <div className='mt-4'>
          <DocsTable
            headers={['系统', '密钥文件']}
            rows={[
              {
                key: 'windows',
                cells: [
                  'Windows',
                  <code key='path'>%USERPROFILE%\.hermes\.env</code>,
                ],
              },
              {
                key: 'unix',
                cells: [
                  'macOS / Linux',
                  <code key='path'>~/.hermes/.env</code>,
                ],
              },
            ]}
          />
        </div>
        <div className='mt-5'>
          <CodeBlock code='ALLTOKEN_API_KEY=此处替换为 API Key' label='.env' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          也可以让 Hermes 将密钥写入自己的 .env：
        </p>
        <div className='mt-4'>
          <CodeBlock
            code='hermes config set ALLTOKEN_API_KEY "此处替换为 API Key"'
            label='终端'
          />
        </div>

        <h3 className='mt-8 text-lg font-semibold'>3.2 编辑 config.yaml</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          以下示例按 Chat Completions 编写。Responses 模型请把 transport 改为
          codex_responses；Anthropic Messages 模型还要把 api 改为不带 /v1
          的服务根地址。
        </p>
        <div className='mt-5'>
          <CodeBlock code={providerConfig} label='~/.hermes/config.yaml' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          将配置合并到现有 config.yaml，不要覆盖终端、工具、Gateway
          或其他服务商设置。
        </p>
        <p className='text-muted-foreground mt-5 leading-7'>
          如果希望端点不请求 /models，可在服务商配置中增加：
        </p>
        <div className='mt-4'>
          <CodeBlock
            code={`providers:
  alltokenapi:
    discover_models: false`}
            label='config.yaml'
          />
        </div>
        <p className='text-muted-foreground mt-5 leading-7'>
          如果希望在配置中直接引用环境变量，也支持：
        </p>
        <div className='mt-4'>
          <CodeBlock
            code={`providers:
  alltokenapi:
    api_key: \${ALLTOKEN_API_KEY}`}
            label='config.yaml'
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          不过 key_env: ALLTOKEN_API_KEY 更适合当前 providers
          格式。$&#123;VAR&#125; 和 $&#123;env:VAR&#125; 都能解析；变量缺失时
          Hermes 会保留占位符并记录警告，而不是静默使用空值。
        </p>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 验证配置</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          先确认当前配置和密钥文件路径，再检查服务商与最终模型路由。
        </p>
        <div className='mt-5'>
          <CodeBlock
            code={`hermes config path
hermes config env-path
hermes config check
hermes config get providers.alltokenapi --json
hermes config get model --json
hermes status`}
            label='配置检查'
          />
        </div>
        <p className='text-muted-foreground mt-5 leading-7'>
          最后执行一次最小请求，并到使用日志确认实际模型、接口和请求状态。
        </p>
        <div className='mt-4'>
          <CodeBlock code='hermes -z "只回复 OK"' label='最小请求' />
        </div>
        <Button
          className='mt-5'
          variant='outline'
          render={<Link to='/usage-logs' />}
        >
          打开使用日志
        </Button>
      </section>

      <section id='reload' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>5. 配置何时生效</h2>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          <li>新启动的 CLI 会话会读取新配置。</li>
          <li>
            已经运行的会话继续使用创建时的模型；会话内可用 /model
            切换已配置服务商。
          </li>
          <li>Messaging Gateway 的新会话读取新默认模型。</li>
        </ul>
        <p className='text-muted-foreground mt-5 leading-7'>
          需要强制所有会话重新读取时，运行：
        </p>
        <div className='mt-4'>
          <CodeBlock code='hermes gateway restart' label='终端' />
        </div>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>6. 常见问题</h2>
        <div className='mt-5 space-y-7'>
          <div>
            <h3 className='text-lg font-semibold'>404 或接口路径错误</h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>
                chat_completions 和 codex_responses 的 api 通常以 /v1 结尾。
              </li>
              <li>
                anthropic_messages 使用服务根地址，由 Hermes 拼接 /v1/messages。
              </li>
              <li>
                transport 留空时 Hermes 会尝试按 URL
                自动判断，但中转地址通常无法可靠反映协议，建议显式设置。
              </li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>Hermes 仍使用旧服务商</h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>
                model.provider 必须为 custom:alltokenapi，不是 alltokenapi 或
                custom。
              </li>
              <li>
                model.default 与 providers.alltokenapi.default_model
                应使用相同的准确模型 ID。
              </li>
              <li>已运行会话不会自动切换，启动新会话或使用 /model。</li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>认证失败</h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>
                运行 hermes config env-path，确认密钥写入当前 Profile 的 .env。
              </li>
              <li>检查 key_env 拼写与 .env 中的变量名一致。</li>
              <li>
                如果 Hermes 以服务方式运行，重启 Gateway 以重新载入密钥环境。
              </li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>
              普通对话成功但 Agent 工具调用失败
            </h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>确认模型支持 Tool Calling。</li>
              <li>确认 transport 与模型实际接口一致。</li>
              <li>
                Responses 接口必须使用 codex_responses，不能用 chat_completions
                侥幸兼容。
              </li>
            </ul>
          </div>
        </div>
      </section>

      <section id='references' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>7. 官方参考</h2>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          {HERMES_REFERENCES.map(([label, href]) => (
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
