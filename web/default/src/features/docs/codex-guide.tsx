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

import { CC_SWITCH_SCREENSHOTS } from './components/cc-switch-screenshots'
import { CodeBlock } from './components/code-block'
import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { GuideSteps } from './components/guide-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const CCSWITCH_RELEASES_URL = 'https://github.com/farion1231/cc-switch/releases'
const CODEX_AUTH_REFERENCE_URL = 'https://developers.openai.com/codex/auth'
const CODEX_CONFIG_REFERENCE_URL =
  'https://developers.openai.com/codex/config-reference'
const CODEX_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const CODEX_TOC: DocsTocItem[] = [
  { id: 'cc-switch-import', label: '1. 使用 CC Switch 一键导入' },
  { id: 'manual-config', label: '2. 编辑 auth.json 和 config.toml' },
]

const CODEX_VERIFY_STEPS = [
  {
    content:
      '保存 auth.json 和 config.toml，然后重启正在运行的 Codex 应用或 IDE 扩展。',
  },
  {
    content:
      '在任意终端中运行 codex，发起一个测试任务；认证信息由 auth.json 提供。',
  },
  {
    content:
      '如启动失败，运行 codex doctor 检查认证文件是否被读取，再检查模型名称和 Base URL。',
  },
]

export function DocsCodex() {
  const baseUrl = useDocsBaseUrl()
  const authJson = `{
  "OPENAI_API_KEY": "sk-your-api-key"
}`
  const codexConfig = `cli_auth_credentials_store = "file"
model = "gpt-5.6-sol"
model_provider = "OpenAI"

[model_providers.OpenAI]
name = "All Token API"
base_url = "${baseUrl}/v1"
requires_openai_auth = true
wire_api = "responses"`
  const windowsAuthPath = '%USERPROFILE%\\.codex\\auth.json'
  const windowsConfigPath = '%USERPROFILE%\\.codex\\config.toml'

  return (
    <DocsShell
      pageId='codex'
      title='Codex'
      description='可选择 CC Switch 一键导入，或手动编辑 auth.json 和 config.toml 接入 Codex。'
      toc={CODEX_TOC}
    >
      <section id='cc-switch-import' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 使用 CC Switch 一键导入</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          CC Switch 是最快的接入方式，无需手动编辑 TOML，即可导入 API
          Key、所选模型和 Codex 服务地址。
        </p>
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
                    className={CODEX_LINK_CLASS}
                  >
                    CC Switch 发布页
                  </a>{' '}
                  安装应用，并确认系统已注册 ccswitch:// 协议。
                </>
              ),
              screenshots: [CC_SWITCH_SCREENSHOTS.download],
            },
            {
              content: (
                <>
                  打开{' '}
                  <Link to='/keys' className={CODEX_LINK_CLASS}>
                    API Key 页面
                  </Link>
                  ，找到要使用的密钥。
                </>
              ),
              screenshots: [
                CC_SWITCH_SCREENSHOTS.apiKey,
                CC_SWITCH_SCREENSHOTS.apiKeyDetails,
              ],
            },
            {
              content: '打开该行的操作菜单，选择 CC Switch 导入。',
              screenshots: [CC_SWITCH_SCREENSHOTS.importEntry],
            },
            {
              content:
                '选择 Codex，再选择当前 API Key 可用的主模型，并保留自动生成的 /v1 地址。',
              screenshots: [CC_SWITCH_SCREENSHOTS.importDialog],
            },
            {
              content:
                '确认浏览器提示，然后在 CC Switch 中检查、保存并启用导入的服务商。',
              screenshots: [
                CC_SWITCH_SCREENSHOTS.confirmImport,
                CC_SWITCH_SCREENSHOTS.imported,
              ],
            },
          ]}
        />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>导入的配置</AlertTitle>
          <AlertDescription>
            Codex 导入内容包含所选 API Key、主模型和以 /v1
            结尾的服务地址。请仅在自己的设备上确认导入。
          </AlertDescription>
        </Alert>
      </section>

      <section id='manual-config' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          2. 编辑 auth.json 和 config.toml
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          本流程将 Codex 的认证凭据保存到用户目录中的 auth.json，并从
          config.toml 读取服务商、模型和接口设置。不同系统的文件路径如下：
        </p>
        <div className='mt-4 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code={windowsAuthPath} label='Windows auth.json' />
          <CodeBlock code={windowsConfigPath} label='Windows' />
          <CodeBlock
            code='~/.codex/auth.json'
            label='macOS / Linux auth.json'
          />
          <CodeBlock code='~/.codex/config.toml' label='macOS / Linux' />
        </div>

        <h3 className='mt-8 text-lg font-semibold'>编辑 auth.json</h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          在用户级 auth.json 中写入 API Key。默认路径为
          ~/.codex/auth.json；Windows 使用
          %USERPROFILE%\.codex\auth.json。如果文件中已有其他认证字段，只更新
          OPENAI_API_KEY，不要覆盖整个文件。
        </p>
        <div className='mt-4'>
          <CodeBlock code={authJson} label='auth.json' />
        </div>
        <p className='text-muted-foreground mt-2 leading-7'>
          保存后，关闭终端也不会清除该配置。文件模式下的 auth.json
          包含明文密钥，请仅保存在自己的设备上，不要提交到 Git
          或发送给他人。密钥只保存在 auth.json；config.toml 仅用于服务商、模型和
          接口设置，不要依赖当前终端的临时设置。
        </p>

        <h3 className='mt-8 text-lg font-semibold'>编辑 config.toml</h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          保留现有 Codex 配置，并合并下面的服务商配置块；已有
          <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
            cli_auth_credentials_store
          </code>
          时，将其改为
          <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
            file
          </code>
          ，不要重复添加。需要时，将示例模型替换为 auth.json 中 API Key
          可用的模型。
        </p>
        <div className='mt-4'>
          <CodeBlock code={codexConfig} label='config.toml' />
        </div>

        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>为什么对话历史丢失</AlertTitle>
          <AlertDescription>
            如果当前 config.toml 中的
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              model_provider
            </code>
            与创建对话时使用的值不一致，原有对话历史可能不会显示。这通常不代表记录已被删除；将
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              model_provider
            </code>
            改回创建对话时的值（包括大小写），重启 Codex 后使用
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              codex resume
            </code>
            查找历史。
          </AlertDescription>
        </Alert>

        <h3 className='mt-8 text-lg font-semibold'>重启并验证</h3>
        <GuideSteps items={CODEX_VERIFY_STEPS} />
        <div className='mt-5'>
          <CodeBlock code='codex' label='终端' />
        </div>

        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>配置说明</AlertTitle>
          <AlertDescription>
            请保持
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              wire_api = &quot;responses&quot;
            </code>
            ，并确保本服务的 Base URL 以
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              /v1
            </code>
            结尾。{' '}
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              cli_auth_credentials_store = &quot;file&quot;
            </code>{' '}
            会让 Codex 使用 auth.json 保存和读取凭据；{' '}
            <code className='bg-muted mx-1 rounded px-1.5 py-0.5 text-sm'>
              requires_openai_auth = true
            </code>{' '}
            会让自定义服务商使用该凭据；config.toml
            只保留服务商、模型和接口设置。 配置字段的完整说明请参考{' '}
            <a
              href={CODEX_CONFIG_REFERENCE_URL}
              target='_blank'
              rel='noopener noreferrer'
              className={CODEX_LINK_CLASS}
            >
              Codex 配置参考
            </a>
            ，认证文件说明请参考{' '}
            <a
              href={CODEX_AUTH_REFERENCE_URL}
              target='_blank'
              rel='noopener noreferrer'
              className={CODEX_LINK_CLASS}
            >
              Codex 认证参考
            </a>
            。
          </AlertDescription>
        </Alert>
      </section>
    </DocsShell>
  )
}
