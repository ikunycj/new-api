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
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { NumberedSteps } from './components/numbered-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const references = [
  [
    'Gemini CLI 认证设置',
    'https://geminicli.com/docs/get-started/authentication/',
  ],
  [
    'Gemini CLI 配置参考',
    'https://geminicli.com/docs/reference/configuration/',
  ],
  ['Gemini CLI 模型路由', 'https://geminicli.com/docs/cli/model-routing/'],
  ['Gemini CLI 官方仓库', 'https://github.com/google-gemini/gemini-cli'],
] as const

export function DocsGemini() {
  const baseUrl = useDocsBaseUrl()
  const dotenvConfig = `GEMINI_API_KEY=此处替换为 API Key
GOOGLE_GEMINI_BASE_URL=${baseUrl}
GEMINI_MODEL=此处替换为准确的模型 ID`
  const powershellConfig = `$env:GEMINI_API_KEY = "此处替换为 API Key"
$env:GOOGLE_GEMINI_BASE_URL = "${baseUrl}"
$env:GEMINI_MODEL = "此处替换为准确的模型 ID"`
  const shellConfig = `export GEMINI_API_KEY="此处替换为 API Key"
export GOOGLE_GEMINI_BASE_URL="${baseUrl}"
export GEMINI_MODEL="此处替换为准确的模型 ID"`

  return (
    <DocsShell
      pageId='gemini'
      title='Gemini CLI'
      description='通过 CC Switch 一键导入，或使用 Gemini CLI 官方支持的 API Key、模型和自定义 Base URL 环境变量手动接入。'
      toc={[
        { id: 'prepare', label: '准备 API Key 和模型' },
        { id: 'cc-switch', label: '使用 CC Switch 导入' },
        { id: 'manual', label: '手动配置' },
        { id: 'verify', label: '启动并验证' },
        { id: 'troubleshooting', label: '常见问题' },
        { id: 'references', label: '官方参考' },
      ]}
    >
      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 准备 API Key 和模型</h2>
        <NumberedSteps
          items={[
            '在 API Key 页面创建或复制密钥。',
            '在模型定价页复制一个支持 Gemini 原生接口的准确模型 ID。',
            '确保 Gemini CLI 为较新版本，自定义 Base URL 是当前版本的正式配置项。',
          ]}
        />
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button nativeButton={false} render={<Link to='/keys' />}>
            <HugeiconsIcon icon={Key01Icon} data-icon='inline-start' />
            打开 API Key
          </Button>
          <Button
            nativeButton={false}
            variant='outline'
            render={<Link to='/pricing' />}
          >
            打开模型定价
          </Button>
        </div>
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>认证方式限制</AlertTitle>
          <AlertDescription>
            Gemini CLI 仅在 gemini-api-key 认证方式下使用
            GOOGLE_GEMINI_BASE_URL。本文配置不适用于 Google 登录或 Vertex AI
            认证。
          </AlertDescription>
        </Alert>
      </section>

      <section id='cc-switch' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 使用 CC Switch 一键导入</h2>
        <NumberedSteps
          items={[
            '安装并打开 CC Switch。',
            '在 API Key 页打开密钥操作菜单，选择 CC Switch 导入。',
            '客户端选择 Gemini，再选择支持 Gemini 接口的模型。',
            `保留服务根地址 ${baseUrl}，不要添加 /v1 或 /v1beta。`,
            '保存并启用服务商，完全退出并重新打开 Gemini CLI。',
          ]}
        />
        <Button
          nativeButton={false}
          className='mt-5'
          variant='outline'
          render={<Link to='/docs/tools/cc-switch' />}
        >
          查看 CC Switch 详细步骤
        </Button>
      </section>

      <section id='manual' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 手动配置</h2>
        <h3 className='mt-5 text-lg font-semibold'>推荐：写入用户 .env</h3>
        <div className='mt-4'>
          <DocsTable
            headers={['系统', '用户环境文件']}
            rows={[
              {
                key: 'windows',
                cells: [
                  'Windows',
                  <code key='windows'>%USERPROFILE%\.gemini\.env</code>,
                ],
              },
              {
                key: 'unix',
                cells: [
                  'macOS / Linux',
                  <code key='unix'>~/.gemini/.env</code>,
                ],
              },
            ]}
          />
        </div>
        <div className='mt-5'>
          <CodeBlock code={dotenvConfig} label='.env' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          Gemini CLI 从当前目录向上查找 .env，再读取用户级
          ~/.gemini/.env，并使用找到的第一份环境文件，而不是合并所有文件。
        </p>
        <h3 className='mt-7 text-lg font-semibold'>临时测试：当前终端变量</h3>
        <div className='mt-4 grid gap-4 lg:grid-cols-2'>
          <CodeBlock code={shellConfig} label='macOS / Linux' />
          <CodeBlock code={powershellConfig} label='PowerShell' />
        </div>
        <h3 className='mt-7 text-lg font-semibold'>Base URL 和认证头</h3>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          <li>GOOGLE_GEMINI_BASE_URL 使用服务根地址，不带 /v1 或 /v1beta。</li>
          <li>默认按 Gemini 原生方式发送 x-goog-api-key。</li>
          <li>
            只有网关明确要求 Bearer Token 时，才设置
            GEMINI_API_KEY_AUTH_MECHANISM=bearer。
          </li>
        </ul>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 启动并验证</h2>
        <NumberedSteps
          items={[
            '运行 gemini。',
            '首次提示认证方式时选择 Use Gemini API key，不要选择 Google 登录或 Vertex AI。',
            '发送一个简短提示并等待完整响应。',
            '打开使用日志，确认请求使用预期的 Gemini 模型。',
          ]}
        />
        <div className='mt-5'>
          <CodeBlock
            code='gemini --model "此处替换为准确的模型 ID"'
            label='临时覆盖模型'
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          模型选择优先级为：--model、GEMINI_MODEL、settings.json 中的
          model.name、默认模型。
        </p>
      </section>

      <section id='troubleshooting' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>5. 常见问题</h2>
        <div className='mt-5 space-y-6'>
          <div>
            <h3 className='font-semibold'>Gemini CLI 仍使用 Google 登录</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              重启 CLI 并选择 Use Gemini API key；确认启动进程能读取
              GEMINI_API_KEY，且项目 .env 没有抢先覆盖用户文件。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>401 或 Invalid API key</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              确认密钥、分组和模型权限有效。默认先用
              x-goog-api-key，仅在网关明确要求时改为 Bearer。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>404 Not Found</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              GOOGLE_GEMINI_BASE_URL 不要带 API 版本路径；模型必须支持 Gemini
              原生接口。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>自定义地址被拒绝</h3>
            <p className='text-muted-foreground mt-2 leading-7'>
              Gemini CLI 要求远程 Base URL 使用 HTTPS。只有 localhost、127.0.0.1
              和 [::1] 可以使用 HTTP。
            </p>
          </div>
        </div>
      </section>

      <section id='references' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>6. 官方参考</h2>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          {references.map(([label, href]) => (
            <li key={href}>
              <a
                href={href}
                target='_blank'
                rel='noopener noreferrer'
                className='text-foreground underline underline-offset-4'
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
