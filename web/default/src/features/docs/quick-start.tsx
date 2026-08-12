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
import {
  ArrowRight01Icon,
  InformationCircleIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { NumberedSteps } from './components/numbered-steps'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const QUICK_START_TOC = [
  { id: 'prepare', label: '开始前准备' },
  { id: 'cc-switch', label: '1. CC Switch 一键导入' },
  { id: 'playground', label: '2. 网页端对话' },
  { id: 'manual', label: '3. 手动配置客户端' },
  { id: 'verify', label: '验证与排错' },
]

export function DocsQuickStart() {
  const baseUrl = useDocsBaseUrl()
  const openAiBaseUrl = `${baseUrl}/v1`
  const curlExample = `curl "${openAiBaseUrl}/chat/completions" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "messages": [{"role":"user","content":"你好"}]
  }'`

  return (
    <DocsShell
      pageId='quick-start'
      title='快速开始'
      description='准备 API Key 和模型 ID，然后选择一种接入方式开始使用 AllTokenAPI。'
      toc={QUICK_START_TOC}
    >
      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>开始前准备</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          三种方式使用同一套 API Key
          和模型权限。先完成下面两项准备，后续步骤会更顺利。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={['准备项', '说明']}
            rows={[
              {
                key: 'api-key',
                cells: [
                  <Link
                    key='api-key-link'
                    to='/keys'
                    className='text-primary font-medium underline-offset-4 hover:underline'
                  >
                    API Key
                  </Link>,
                  '在 API Key 页面创建密钥，并确认密钥状态正常。',
                ],
              },
              {
                key: 'model-id',
                cells: [
                  <Link
                    key='model-id-link'
                    to='/docs/model-pricing'
                    className='text-primary font-medium underline-offset-4 hover:underline'
                  >
                    模型 ID
                  </Link>,
                  '复制模型列表中的完整 ID，不要使用展示名称或自行猜测模型名。',
                ],
              },
            ]}
            minWidth='sm'
          />
        </div>
      </section>

      <section id='cc-switch' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 使用 CC Switch 一键导入</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          如果你使用 Claude Code、Codex 或 Gemini CLI，推荐先使用 CC Switch
          导入配置，避免手动编辑多个配置文件。
        </p>
        <NumberedSteps
          items={[
            '安装并打开 CC Switch。',
            '在 API Key 页面找到要使用的密钥，打开操作菜单并选择 CC Switch 导入。',
            '选择目标客户端和模型，检查服务地址与模型 ID。',
            '确认导入并在 CC Switch 中启用配置，然后重启客户端。',
          ]}
        />
        <div className='mt-6 flex flex-wrap gap-3'>
          <Button render={<Link to='/docs/tools/cc-switch' />}>
            查看完整配置步骤
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button variant='outline' render={<Link to='/keys' />}>
            打开 API Key 页面
          </Button>
        </div>
      </section>

      <section id='playground' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 直接在网页端开启对话</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          不想配置本地客户端时，可以直接打开网页对话。进入后选择模型，发送一条简短消息即可确认
          API Key 和模型是否可用。
        </p>
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button variant='outline' render={<Link to='/playground' />}>
            打开网页对话
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button variant='ghost' render={<Link to='/docs/model-pricing' />}>
            查看模型定价
          </Button>
        </div>
      </section>

      <section id='manual' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>3. 手动配置客户端</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          对于支持自定义 Base URL、API Key 和模型 ID
          的客户端，可以手动填写配置。具体字段名称以客户端自身文档为准。
        </p>
        <div className='mt-5'>
          <DocsTable
            headers={['配置项', '填写内容']}
            rows={[
              {
                key: 'base-url',
                cells: [
                  <code key='base-url-code'>Base URL</code>,
                  <code key='base-url-value'>{baseUrl}/v1</code>,
                ],
              },
              {
                key: 'api-key-value',
                cells: [
                  <code key='api-key-code'>API Key</code>,
                  '填写在 API Key 页面创建的密钥，不要把密钥提交到代码仓库。',
                ],
              },
              {
                key: 'model-id-value',
                cells: [
                  <code key='model-id-code'>Model ID</code>,
                  '填写模型列表中的完整模型 ID。',
                ],
              },
            ]}
            minWidth='sm'
          />
        </div>
        <div className='mt-5'>
          <CodeBlock code={curlExample} label='cURL / OpenAI Compatible' />
        </div>
        <div className='mt-6 flex flex-wrap gap-3'>
          <Button render={<Link to='/docs/integrations' />}>
            查看客户端集成指南
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button
            variant='outline'
            render={<Link to='/docs/api/integration' />}
          >
            查看 API 模型接口
          </Button>
        </div>
      </section>

      <section id='verify' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>验证与排错</h2>
        <NumberedSteps
          items={[
            '重启客户端或重新打开网页对话，确保新配置已经被读取。',
            '发送一条简短测试消息，并在使用日志中确认请求状态和模型。',
            '如果返回 401，检查 API Key 是否过期、是否多了空格，以及客户端使用的鉴权字段是否正确。',
            '如果返回 404，检查 Base URL 是否重复拼接 /v1，并确认模型支持当前请求协议。',
          ]}
        />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>不要混用协议请求体</AlertTitle>
          <AlertDescription>
            OpenAI Chat Completions 使用 <code>messages</code>，Responses 使用{' '}
            <code>input</code>，Claude Messages 使用 <code>messages</code> 和{' '}
            <code>max_tokens</code>。协议和请求体必须与客户端配置保持一致。
          </AlertDescription>
        </Alert>
      </section>
    </DocsShell>
  )
}
