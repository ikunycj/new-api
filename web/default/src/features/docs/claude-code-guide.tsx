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
import { Alert02Icon, InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { CC_SWITCH_SCREENSHOTS } from './components/cc-switch-screenshots'
import { CodeBlock } from './components/code-block'
import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { GuideSteps } from './components/guide-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const CCSWITCH_RELEASES_URL = 'https://github.com/farion1231/cc-switch/releases'
const CLAUDE_LLM_GATEWAY_REFERENCE_URL =
  'https://code.claude.com/docs/en/llm-gateway-connect'
const CLAUDE_MODEL_REFERENCE_URL =
  'https://code.claude.com/docs/en/model-config'
const CLAUDE_SETTINGS_REFERENCE_URL = 'https://code.claude.com/docs/en/settings'
const CLAUDE_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const CLAUDE_TOC: DocsTocItem[] = [
  { id: 'prepare', label: '1. 准备 API Key 和模型' },
  { id: 'cc-switch-import', label: '2. 使用 CC Switch 一键导入' },
  { id: 'manual-configuration', label: '3. 手动配置' },
  { id: 'verify', label: '4. 启动并验证' },
  { id: 'troubleshooting', label: '5. 常见问题' },
  { id: 'references', label: '6. 官方参考' },
]

export function DocsClaudeCode() {
  const baseUrl = useDocsBaseUrl()
  const settingsJson = `{
  "env": {
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_AUTH_TOKEN": "此处替换为 API Key"
  },
  "model": "此处替换为准确的模型 ID"
}`
  const windowsSettingsPath = '%USERPROFILE%\\.claude\\settings.json'

  return (
    <DocsShell
      pageId='claude-code'
      title='Claude Code'
      description='可通过 CC Switch 一键导入，或使用 Claude Code 用户级 settings.json 配置文件手动接入。'
      toc={CLAUDE_TOC}
    >
      <Alert>
        <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
        <AlertTitle>配置核验</AlertTitle>
        <AlertDescription>
          本文根据 Claude Code 官方的 LLM Gateway 与模型配置文档核验。All Token
          API 使用 Bearer Token，因此手动配置使用
          <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
            ANTHROPIC_AUTH_TOKEN
          </code>
          。
        </AlertDescription>
      </Alert>

      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 准备 API Key 和模型</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  在{' '}
                  <Link to='/keys' className={CLAUDE_LINK_CLASS}>
                    API Key 页面
                  </Link>{' '}
                  创建或复制密钥。
                </>
              ),
            },
            {
              content: (
                <>
                  在{' '}
                  <Link to='/pricing' className={CLAUDE_LINK_CLASS}>
                    模型定价页面
                  </Link>{' '}
                  复制一个支持 Anthropic Messages 接口的准确模型 ID。
                </>
              ),
            },
            {
              content: '更新 Claude Code，避免旧版本缺少网关或模型配置能力。',
            },
          ]}
        />
        <div className='mt-5'>
          <CodeBlock code='claude update' label='终端' />
        </div>
      </section>

      <section id='cc-switch-import' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 使用 CC Switch 一键导入</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  从{' '}
                  <a
                    href={CCSWITCH_RELEASES_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                    className={CLAUDE_LINK_CLASS}
                  >
                    CC Switch 发布页
                  </a>{' '}
                  安装并打开 CC Switch。
                </>
              ),
              screenshots: [CC_SWITCH_SCREENSHOTS.download],
            },
            {
              content: (
                <>
                  在{' '}
                  <Link to='/keys' className={CLAUDE_LINK_CLASS}>
                    API Key 页面
                  </Link>{' '}
                  打开密钥的操作菜单，选择 CC Switch 导入。
                </>
              ),
              screenshots: [CC_SWITCH_SCREENSHOTS.importEntry],
            },
            {
              content: '客户端选择 Claude，模型选择刚才确认的 Claude 模型。',
              screenshots: [CC_SWITCH_SCREENSHOTS.importDialog],
            },
            {
              content: (
                <>
                  保留生成的服务根地址 <code>{baseUrl}</code>，不要手动添加
                  /v1；详细说明可查看{' '}
                  <Link
                    to='/docs/tools/cc-switch'
                    className={CLAUDE_LINK_CLASS}
                  >
                    CC Switch 一键导入指南
                  </Link>
                  。
                </>
              ),
              screenshots: [CC_SWITCH_SCREENSHOTS.confirmImport],
            },
            {
              content:
                '在 CC Switch 中保存并启用服务商，然后完全退出并重新打开 Claude Code。',
              screenshots: [CC_SWITCH_SCREENSHOTS.imported],
            },
          ]}
        />
      </section>

      <section id='manual-configuration' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 手动配置</h2>

        <h3 className='mt-6 text-lg font-semibold'>
          3.1 在用户 settings.json 中直接保存密钥
        </h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          用户配置对所有项目生效，也能被 Claude Code 的后台 Agent
          读取。不同系统的用户配置文件路径如下：
        </p>
        <div className='mt-4 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code={windowsSettingsPath} label='Windows' />
          <CodeBlock code='~/.claude/settings.json' label='macOS / Linux' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          将以下字段合并到现有 JSON 中，不要删除原有的权限、插件或 MCP 配置：
        </p>
        <div className='mt-4'>
          <CodeBlock code={settingsJson} label='settings.json' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          这里的 <code>env</code> 是 Claude Code 配置文件中的字段；保存后由
          Claude Code 在新会话中加载，不需要再在终端里手动设置这些配置项。
        </p>
        <Alert className='mt-6' variant='destructive'>
          <HugeiconsIcon icon={Alert02Icon} aria-hidden='true' />
          <AlertTitle>不要提交密钥</AlertTitle>
          <AlertDescription>
            不要把密钥写入项目共享的
            <code className='mx-1 rounded px-1.5 py-0.5 text-sm'>
              .claude/settings.json
            </code>
            。如需项目级配置，应使用已加入
            <code className='mx-1 rounded px-1.5 py-0.5 text-sm'>
              .gitignore
            </code>
            的
            <code className='mx-1 rounded px-1.5 py-0.5 text-sm'>
              .claude/settings.local.json
            </code>
            。
          </AlertDescription>
        </Alert>

        <h3 className='mt-8 text-lg font-semibold'>
          3.2 为什么 Base URL 不带 /v1
        </h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          Claude Code 会在
          <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
            ANTHROPIC_BASE_URL
          </code>
          后请求
          <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
            /v1/messages
          </code>
          。如果 Base URL 已经包含 /v1，最终可能形成重复路径并返回 404。
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          Claude Code 配置文件中的凭据字段与请求头对应关系如下：
        </p>
        <div className='mt-5 rounded-lg border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>配置字段</TableHead>
                <TableHead>请求头</TableHead>
                <TableHead>适用场景</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>
                  <code>ANTHROPIC_AUTH_TOKEN</code>
                </TableCell>
                <TableCell>
                  <code>Authorization: Bearer ...</code>
                </TableCell>
                <TableCell>All Token API、Bearer Token 网关</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <code>ANTHROPIC_API_KEY</code>
                </TableCell>
                <TableCell>
                  <code>x-api-key: ...</code>
                </TableCell>
                <TableCell>明确要求 Anthropic x-api-key 的网关</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>不要同时配置两个凭据字段</AlertTitle>
          <AlertDescription>
            本站使用 ANTHROPIC_AUTH_TOKEN。不要同时设置 ANTHROPIC_AUTH_TOKEN 和
            ANTHROPIC_API_KEY，以免出现认证来源冲突。
          </AlertDescription>
        </Alert>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 启动并验证</h2>
        <GuideSteps
          items={[
            {
              content:
                '启动 claude；用户级配置会对新会话生效，不依赖特定终端。',
            },
            {
              content:
                '如果出现登录页，说明网关凭据没有被读取；不要选择 Claude 订阅登录，先检查配置文件路径和 JSON 格式。',
            },
            {
              content:
                '进入会话后运行 /status，核对服务地址、认证字段和当前模型。',
            },
            {
              content: (
                <>
                  发送一个简短测试请求，再到{' '}
                  <Link to='/usage-logs' className={CLAUDE_LINK_CLASS}>
                    使用日志
                  </Link>{' '}
                  确认请求模型和状态。
                </>
              ),
            },
          ]}
        />
        <p className='text-muted-foreground mt-5 leading-7'>
          在
          <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
            /status
          </code>
          中应确认以下内容：
        </p>
        <ul className='mt-3 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>
            Anthropic base URL 为 <code>{baseUrl}</code>；
          </li>
          <li>
            Auth token or API key 显示
            <code className='mx-1'>ANTHROPIC_AUTH_TOKEN</code>；
          </li>
          <li>当前模型为预期的准确模型 ID。</li>
        </ul>
        <p className='text-muted-foreground mt-4 leading-7'>
          也可以在启动时临时指定模型：
        </p>
        <div className='mt-4'>
          <CodeBlock
            code='claude --model "此处替换为准确的模型 ID"'
            label='终端'
          />
        </div>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>5. 常见问题</h2>

        <h3 className='mt-6 text-lg font-semibold'>启动后仍要求登录</h3>
        <ul className='mt-3 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>
            确认配置写在 <code>~/.claude/settings.json</code>
            ，而不是其他同名文件。
          </li>
          <li>
            确认 <code>env</code> 位于 JSON 顶层。
          </li>
          <li>
            运行 <code>/logout</code> 可清除与网关凭据冲突的历史登录状态。
          </li>
        </ul>

        <h3 className='mt-6 text-lg font-semibold'>
          401、Unauthorized 或 Incorrect API key
        </h3>
        <ul className='mt-3 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>确认密钥没有多余空格或引号。</li>
          <li>
            All Token API 应使用 <code>ANTHROPIC_AUTH_TOKEN</code>
            ，不要误用只发送 <code>x-api-key</code> 的
            <code className='ms-1'>ANTHROPIC_API_KEY</code>。
          </li>
          <li>
            确认密钥未过期，并在
            <Link to='/keys' className={CLAUDE_LINK_CLASS}>
              API Key 页面
            </Link>
            检查其分组和模型权限。
          </li>
        </ul>

        <h3 className='mt-6 text-lg font-semibold'>404 Not Found</h3>
        <ul className='mt-3 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>
            <code>ANTHROPIC_BASE_URL</code> 应为 <code>{baseUrl}</code>
            ，不要带 /v1 或 /v1/messages。
          </li>
          <li>所选模型必须支持 Anthropic Messages 接口。</li>
        </ul>

        <h3 className='mt-6 text-lg font-semibold'>模型不可用</h3>
        <ul className='mt-3 flex list-disc flex-col gap-2 ps-6 leading-7'>
          <li>
            使用
            <Link to='/pricing' className={CLAUDE_LINK_CLASS}>
              模型定价页面
            </Link>
            显示的完整模型 ID，不要使用展示名称。
          </li>
          <li>
            可用 <code>claude --model &lt;模型ID&gt;</code>
            排除已保存模型选择的影响。
          </li>
          <li>
            自定义网关只保证其声明支持的接口；部分
            Beta、文件上传或远程功能可能无法透传。
          </li>
        </ul>
      </section>

      <section id='references' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>6. 官方参考</h2>
        <ul className='mt-4 flex list-disc flex-col gap-3 ps-6 leading-7'>
          <li>
            <a
              href={CLAUDE_LLM_GATEWAY_REFERENCE_URL}
              target='_blank'
              rel='noopener noreferrer'
              className={CLAUDE_LINK_CLASS}
            >
              连接 Claude Code 到 LLM Gateway
            </a>
          </li>
          <li>
            <a
              href={CLAUDE_MODEL_REFERENCE_URL}
              target='_blank'
              rel='noopener noreferrer'
              className={CLAUDE_LINK_CLASS}
            >
              Claude Code 模型配置
            </a>
          </li>
          <li>
            <a
              href={CLAUDE_SETTINGS_REFERENCE_URL}
              target='_blank'
              rel='noopener noreferrer'
              className={CLAUDE_LINK_CLASS}
            >
              Claude Code 设置文件
            </a>
          </li>
        </ul>
      </section>
    </DocsShell>
  )
}
