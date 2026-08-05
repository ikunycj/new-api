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

const OPENCLAW_TOC: DocsTocItem[] = [
  { id: 'prepare', label: '1. 准备 API Key 和模型' },
  { id: 'save-key', label: '2. 保存 API Key' },
  { id: 'config', label: '3. 编辑 openclaw.json' },
  { id: 'verify', label: '4. 验证配置' },
  { id: 'reload', label: '5. 配置何时生效' },
  { id: 'troubleshooting', label: '6. 常见问题' },
  { id: 'references', label: '7. 官方参考' },
]

const OPENCLAW_REFERENCES = [
  ['OpenClaw 配置指南', 'https://docs.openclaw.ai/gateway/configuration'],
  [
    '自定义服务商与 Base URL',
    'https://docs.openclaw.ai/gateway/config-tools#custom-providers-and-base-urls',
  ],
  ['OpenClaw config 命令', 'https://docs.openclaw.ai/cli/config'],
  ['OpenClaw models 命令', 'https://docs.openclaw.ai/cli/models'],
] as const

const DOCS_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

export function DocsOpenClaw() {
  const baseUrl = useDocsBaseUrl()
  const dotenvConfig = 'ALLTOKEN_API_KEY=此处替换为 API Key'
  const shellApiKey = 'export ALLTOKEN_API_KEY="此处替换为 API Key"'
  const powershellApiKey = '$env:ALLTOKEN_API_KEY = "此处替换为 API Key"'
  const openClawConfig = `{
  models: {
    mode: "merge",
    providers: {
      alltokenapi: {
        baseUrl: "${baseUrl}/v1",
        apiKey: "\${ALLTOKEN_API_KEY}",
        api: "openai-responses",
        models: [
          {
            id: "此处替换为准确的模型 ID",
            name: "此处替换为准确的模型 ID",
          },
        ],
      },
    },
  },
  agents: {
    defaults: {
      model: {
        primary: "alltokenapi/此处替换为准确的模型 ID",
      },
    },
  },
}`

  return (
    <DocsShell
      pageId='openclaw'
      title='OpenClaw'
      description='将本服务注册为 OpenClaw 的自定义模型服务商，再把默认 Agent 指向该服务商。'
      toc={OPENCLAW_TOC}
    >
      <Alert>
        <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
        <AlertTitle>配置核验</AlertTitle>
        <AlertDescription>
          OpenClaw 当前使用 ~/.openclaw/openclaw.json，格式为
          JSON5。自定义服务商配置位于 models.providers，默认模型位于
          agents.defaults.model.primary。
        </AlertDescription>
      </Alert>

      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 准备 API Key 和模型</h2>
        <NumberedSteps
          items={[
            '在 API Key 页面创建密钥。',
            '在模型定价页面复制准确的模型 ID。',
            '根据模型支持的接口选择 OpenClaw api 值和 Base URL，不要只根据模型名称猜测接口。',
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
            headers={['模型接口', 'OpenClaw api 值', 'Base URL']}
            rows={[
              {
                key: 'responses',
                cells: [
                  <code key='endpoint'>/v1/responses</code>,
                  <code key='api'>openai-responses</code>,
                  <code key='url'>{baseUrl}/v1</code>,
                ],
              },
              {
                key: 'chat-completions',
                cells: [
                  <code key='endpoint'>/v1/chat/completions</code>,
                  <code key='api'>openai-completions</code>,
                  <code key='url'>{baseUrl}/v1</code>,
                ],
              },
              {
                key: 'anthropic-messages',
                cells: [
                  <code key='endpoint'>/v1/messages</code>,
                  <code key='api'>anthropic-messages</code>,
                  <code key='url'>{baseUrl}</code>,
                ],
              },
            ]}
          />
        </div>
      </section>

      <section id='save-key' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 保存 API Key</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          OpenClaw 会读取父进程环境变量、当前目录的 .env，以及
          ~/.openclaw/.env。Gateway 以服务方式运行时，使用全局 .env 最稳定。
        </p>
        <div className='mt-5 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code='~/.openclaw/.env' label='macOS / Linux' />
          <CodeBlock code='%USERPROFILE%\.openclaw\.env' label='Windows' />
        </div>
        <div className='mt-4'>
          <CodeBlock code={dotenvConfig} label='.env' />
        </div>
        <h3 className='mt-8 text-lg font-semibold'>临时设置环境变量</h3>
        <div className='mt-4 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code={shellApiKey} label='macOS / Linux' />
          <CodeBlock code={powershellApiKey} label='PowerShell' />
        </div>
      </section>

      <section id='config' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 编辑 openclaw.json</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          先运行下面的命令查看当前实际生效的配置文件，再把示例合并到该文件。示例按
          Responses 接口编写；如果所选模型只支持 Chat Completions，请把 api 改为
          openai-completions。
        </p>
        <div className='mt-5'>
          <CodeBlock code='openclaw config file' label='终端' />
        </div>
        <div className='mt-4'>
          <CodeBlock code={openClawConfig} label='openclaw.json' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          需要替换三处模型 ID，且大小写必须一致。models.mode 默认为
          merge，显式写出是为了避免误用 replace 后隐藏内置模型目录。
        </p>
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>JSON5 与环境变量</AlertTitle>
          <AlertDescription>
            OpenClaw
            允许注释、未加引号的键和尾随逗号。$&#123;ALLTOKEN_API_KEY&#125;
            会在加载配置时解析；变量缺失或为空会直接导致配置加载失败。
          </AlertDescription>
        </Alert>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 验证配置</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          先执行只读检查，确认配置、服务商模型和默认模型都已生效。
        </p>
        <div className='mt-5'>
          <CodeBlock
            code={`openclaw config validate
openclaw models list --provider alltokenapi
openclaw models status`}
            label='只读检查'
          />
        </div>
        <ul className='text-muted-foreground mt-4 list-disc space-y-2 pl-5 leading-7'>
          <li>config validate 成功。</li>
          <li>模型列表中出现 alltokenapi/模型ID。</li>
          <li>models status 的默认模型与认证状态正确。</li>
        </ul>
        <p className='text-muted-foreground mt-5 leading-7'>
          需要真实发起一次最小模型请求时，可以运行以下探测命令。它会消耗少量
          Token 并可能触发限流。
        </p>
        <div className='mt-4'>
          <CodeBlock
            code='openclaw models status --probe --probe-provider alltokenapi'
            label='真实请求探测'
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          完成后启动一个简短 Agent 任务，并在
          <Link to='/usage-logs' className={DOCS_LINK_CLASS}>
            使用日志
          </Link>
          中确认请求模型、接口和状态。
        </p>
      </section>

      <section id='reload' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>5. 配置何时生效</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          OpenClaw 默认使用 gateway.reload.mode: hybrid。models 和 agents
          的修改可热加载，通常不需要手动重启 Gateway。
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          如果已将热加载设为
          off，或运行中的会话仍保留旧模型，请按当前部署方式重启
          Gateway。已有会话可能继续使用创建时的模型，新会话会读取新默认值。
        </p>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>6. 常见问题</h2>
        <div className='mt-5 space-y-7'>
          <div>
            <h3 className='text-lg font-semibold'>
              Invalid config 或 Gateway 拒绝启动
            </h3>
            <div className='mt-3'>
              <CodeBlock
                code={`openclaw config validate
openclaw doctor`}
                label='诊断命令'
              />
            </div>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>检查 JSON5 层级和逗号。</li>
              <li>检查 ALLTOKEN_API_KEY 是否可被 Gateway 进程读取。</li>
              <li>不要添加文档中不存在的字段；OpenClaw 会拒绝未知字段。</li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>404 或请求路径错误</h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>Responses 与 Chat Completions 的 Base URL 都以 /v1 结尾。</li>
              <li>anthropic-messages 使用不带 /v1 的服务根地址。</li>
              <li>
                自定义 OpenAI 兼容服务商未填写 api 时默认走
                openai-completions，不会自动改用 Responses。
              </li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>模型显示但无法调用</h3>
            <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
              <li>检查服务商模型 ID 与默认模型中的 ID 是否完全一致。</li>
              <li>
                确认模型支持工具调用；能普通对话不代表能完成 Agent 工具循环。
              </li>
              <li>运行真实请求探测，区分认证、格式和模型错误。</li>
            </ul>
          </div>
          <div>
            <h3 className='text-lg font-semibold'>
              不确定 contextWindow 或 maxTokens
            </h3>
            <p className='text-muted-foreground mt-3 leading-7'>
              不要猜测这些值。未确认时省略
              reasoning、input、cost、contextWindow、contextTokens 和
              maxTokens，让服务端限制请求；只有拿到准确模型元数据后再补充。
            </p>
          </div>
        </div>
      </section>

      <section id='references' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>7. 官方参考</h2>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          {OPENCLAW_REFERENCES.map(([label, href]) => (
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
