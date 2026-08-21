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
const CODEX_CONFIG_REFERENCE_URL =
  'https://developers.openai.com/codex/config-reference'
const CODEX_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const CODEX_TOC: DocsTocItem[] = [
  { id: 'cc-switch-import', label: '1. 使用 CC Switch 一键导入' },
  { id: 'manual-config', label: '2. 手动配置 config.toml' },
]

const CODEX_VERIFY_STEPS = [
  {
    content:
      '保存 config.toml，然后重启终端以及正在运行的 Codex 应用或 IDE 扩展。',
  },
  { content: '在新的终端中运行 codex，发起一个测试任务。' },
  { content: '如启动失败，请检查环境变量、模型名称和 Base URL 后重试。' },
]

export function DocsCodex() {
  const baseUrl = useDocsBaseUrl()
  const codexConfig = `model = "gpt-5.6-sol"
model_provider = "alltokenapi"

[model_providers.alltokenapi]
name = "All Token API"
base_url = "${baseUrl}/v1"
env_key = "ALLTOKEN_API_KEY"
wire_api = "responses"`
  const powershellApiKey = `[Environment]::SetEnvironmentVariable(
  "ALLTOKEN_API_KEY",
  "sk-your-api-key",
  "User"
)`
  const shellApiKey = `export ALLTOKEN_API_KEY="sk-your-api-key"`
  const windowsConfigPath = '%USERPROFILE%\\.codex\\config.toml'

  return (
    <DocsShell
      pageId='codex'
      title='Codex'
      description='可选择 CC Switch 一键导入，或手动修改 config.toml 接入 Codex。'
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
        <h2 className='text-2xl font-semibold'>2. 手动配置 config.toml</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          Codex 从用户目录中的配置文件读取设置。不同系统的配置文件路径如下：
        </p>
        <div className='mt-4 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code={windowsConfigPath} label='Windows' />
          <CodeBlock code='~/.codex/config.toml' label='macOS / Linux' />
        </div>

        <h3 className='mt-8 text-lg font-semibold'>设置 API Key 环境变量</h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          将 API Key 保存到独立的环境变量中，不要把密钥直接写入 config.toml。
        </p>
        <div className='mt-4 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code={powershellApiKey} label='PowerShell' />
          <CodeBlock code={shellApiKey} label='macOS / Linux' />
        </div>

        <h3 className='mt-8 text-lg font-semibold'>编辑 config.toml</h3>
        <p className='text-muted-foreground mt-2 leading-7'>
          保留现有 Codex
          配置，并合并下面的服务商配置块。需要时，将示例模型替换为当前 API Key
          可用的模型。
        </p>
        <div className='mt-4'>
          <CodeBlock code={codexConfig} label='config.toml' />
        </div>

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
            结尾。配置字段的完整说明请参考{' '}
            <a
              href={CODEX_CONFIG_REFERENCE_URL}
              target='_blank'
              rel='noopener noreferrer'
              className={CODEX_LINK_CLASS}
            >
              Codex 配置参考
            </a>
            。
          </AlertDescription>
        </Alert>
      </section>
    </DocsShell>
  )
}
