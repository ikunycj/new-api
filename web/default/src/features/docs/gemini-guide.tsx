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
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { GuideSteps } from './components/guide-steps'
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

const CC_SWITCH_RELEASES_URL =
  'https://github.com/farion1231/cc-switch/releases'
const DOCS_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

export function DocsGemini() {
  const baseUrl = useDocsBaseUrl()
  const dotenvConfig = `GEMINI_API_KEY=此处替换为 API Key
GOOGLE_GEMINI_BASE_URL=${baseUrl}
GEMINI_MODEL=此处替换为准确的模型 ID`

  return (
    <DocsShell
      pageId='gemini'
      title='Gemini CLI'
      description='通过 CC Switch 一键导入，或使用 Gemini CLI 官方支持的 API Key、模型和自定义 Base URL 配置文件手动接入。'
      toc={[
        { id: 'prepare', label: '1. 准备 API Key 和模型' },
        { id: 'cc-switch', label: '2. 使用 CC Switch 一键导入' },
        { id: 'manual', label: '3. 手动配置' },
        { id: 'verify', label: '4. 启动并验证' },
        { id: 'troubleshooting', label: '5. 常见问题' },
        { id: 'references', label: '6. 官方参考' },
      ]}
    >
      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 准备 API Key 和模型</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  在{' '}
                  <Link to='/keys' className={DOCS_LINK_CLASS}>
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
                  <Link to='/pricing' className={DOCS_LINK_CLASS}>
                    模型定价页面
                  </Link>{' '}
                  复制一个支持 Gemini 原生接口的准确模型 ID。
                </>
              ),
            },
            {
              content:
                '确保 Gemini CLI 为较新版本，自定义 Base URL 是当前版本的正式配置项。',
            },
          ]}
        />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>配置核验</AlertTitle>
          <AlertDescription>
            Gemini CLI 仅在 gemini-api-key 认证方式下使用
            GOOGLE_GEMINI_BASE_URL。本文配置不适用于 Google 登录或 Vertex AI
            认证。
          </AlertDescription>
        </Alert>
      </section>

      <section id='cc-switch' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 使用 CC Switch 一键导入</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  从{' '}
                  <a
                    href={CC_SWITCH_RELEASES_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                    className={DOCS_LINK_CLASS}
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
                  <Link to='/keys' className={DOCS_LINK_CLASS}>
                    API Key 页面
                  </Link>{' '}
                  打开密钥操作菜单，选择 CC Switch 导入。
                </>
              ),
              screenshots: [CC_SWITCH_SCREENSHOTS.importEntry],
            },
            {
              content: '客户端选择 Gemini，再选择支持 Gemini 接口的模型。',
              screenshots: [CC_SWITCH_SCREENSHOTS.importDialog],
            },
            {
              content: (
                <>
                  保留服务根地址 <code>{baseUrl}</code>，不要添加 /v1 或
                  /v1beta；完整流程可查看{' '}
                  <Link to='/docs/tools/cc-switch' className={DOCS_LINK_CLASS}>
                    CC Switch 详细步骤
                  </Link>
                  。
                </>
              ),
              screenshots: [CC_SWITCH_SCREENSHOTS.confirmImport],
            },
            {
              content: '保存并启用服务商，完全退出并重新打开 Gemini CLI。',
              screenshots: [CC_SWITCH_SCREENSHOTS.imported],
            },
          ]}
        />
      </section>

      <section id='manual' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 手动配置</h2>
        <h3 className='mt-5 text-lg font-semibold'>
          3.1 将 API Key 写入 Gemini CLI 的用户 .env 配置文件
        </h3>
        <div className='mt-4'>
          <DocsTable
            headers={['系统', '用户配置文件']}
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
          ~/.gemini/.env，并使用找到的第一份配置文件，而不是合并所有文件。写入用户级
          .env 后，关闭终端不会清除密钥；请将该文件视为私密配置，不要提交到
          Git。如果项目目录已有 .env，请确认它没有覆盖或遗漏上述配置项。
        </p>
        <h3 className='mt-7 text-lg font-semibold'>
          3.2 Base URL 和认证头说明
        </h3>
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          <li>
            GOOGLE_GEMINI_BASE_URL 使用服务根地址，不带 /v1 或 /v1beta；Google
            Gen AI SDK 会自行拼接 API 版本和模型路径。
          </li>
          <li>默认按 Gemini 原生方式发送 x-goog-api-key。</li>
          <li>
            只有网关明确要求 Authorization: Bearer
            时，才额外设置下面的认证机制。
          </li>
        </ul>
        <div className='mt-4'>
          <CodeBlock code='GEMINI_API_KEY_AUTH_MECHANISM=bearer' label='.env' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          如需使用该认证机制，请将这一行也写入同一个用户级 .env
          配置文件，不要只在当前终端临时设置。
        </p>
        <p className='text-muted-foreground mt-4 leading-7'>
          不要在没有认证错误的情况下随意切换认证头。
        </p>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>4. 启动并验证</h2>
        <GuideSteps
          items={[
            { content: '运行 gemini。' },
            {
              content:
                '首次提示认证方式时选择 Use Gemini API key，不要选择 Google 登录或 Vertex AI。',
            },
            { content: '发送一个简短提示并等待完整响应。' },
            {
              content: (
                <>
                  打开{' '}
                  <Link to='/usage-logs' className={DOCS_LINK_CLASS}>
                    使用日志
                  </Link>
                  ，确认请求使用预期的 Gemini 模型。
                </>
              ),
            },
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
              重启 CLI 并选择 Use Gemini API key；确认用户级
              ~/.gemini/.env（Windows 为 %USERPROFILE%\.gemini\.env）中写入了
              GEMINI_API_KEY，且项目 .env 没有抢先覆盖用户文件。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>401 或 Invalid API key</h3>
            <ul className='text-muted-foreground mt-2 list-disc space-y-2 pl-5 leading-7'>
              <li>确认密钥、分组和模型权限有效。</li>
              <li>
                默认先使用 x-goog-api-key；如果使用日志或网关说明明确要求 Bearer
                Token，再设置 GEMINI_API_KEY_AUTH_MECHANISM=bearer。
              </li>
              <li>
                不要同时用 GOOGLE_API_KEY 配置 Vertex AI，这会改变认证路径。
              </li>
            </ul>
          </div>
          <div>
            <h3 className='font-semibold'>404 Not Found</h3>
            <ul className='text-muted-foreground mt-2 list-disc space-y-2 pl-5 leading-7'>
              <li>GOOGLE_GEMINI_BASE_URL 不要带 API 版本路径。</li>
              <li>
                模型必须支持 Gemini 原生接口；OpenAI 兼容模型不能只靠修改模型 ID
                在 Gemini CLI 中使用。
              </li>
            </ul>
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
