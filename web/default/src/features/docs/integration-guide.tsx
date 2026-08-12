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
import { ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'

import { DocsShell } from './components/docs-shell'

const INTEGRATION_TOC = [
  { id: 'prepare', label: '开始前准备' },
  { id: 'import', label: '导入配置' },
]

const AGENT_GUIDES = [
  { label: 'Claude Code', to: '/docs/tools/claude-code' },
  { label: 'Codex', to: '/docs/tools/codex' },
  { label: 'Gemini CLI', to: '/docs/tools/gemini' },
  { label: 'Hermes', to: '/docs/tools/hermes' },
  { label: 'OpenClaw', to: '/docs/tools/openclaw' },
  { label: 'OpenCode', to: '/docs/tools/opencode' },
] as const

export function DocsIntegrationGuide() {
  return (
    <DocsShell
      pageId='integration-guide'
      title='集成指南'
      description='将 AllTokenAPI 接入 Claude Code、Codex、Cursor、Cline、Kilo Code 等常用 Agent 工具。'
      toc={INTEGRATION_TOC}
    >
      <p className='text-muted-foreground leading-7'>
        AllTokenAPI 同时提供 OpenAI 兼容、Anthropic Claude Messages 兼容和
        Gemini 兼容接口。大多数 Agent、IDE 插件和桌面客户端只要能自定义 Base
        URL、API Key 和 Model ID，就可以接入。
      </p>

      <section id='prepare' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>开始前准备</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          在配置任何工具之前，先准备好以下信息：
        </p>
        <ol className='mt-4 flex list-decimal flex-col gap-2 ps-6 leading-7'>
          <li>在 AllTokenAPI 工作台创建 API Key。</li>
          <li>从模型列表获取准确的 Model ID。</li>
          <li>
            确认工具支持的协议类型：OpenAI Compatible、OpenAI
            Responses、Anthropic Claude Messages 或 Gemini Compatible。
          </li>
        </ol>
        <p className='text-muted-foreground mt-4 leading-7'>
          建议把密钥保存到本地安全位置。文档中的{' '}
          <code>AllTokenAPI_API_KEY</code> 只是占位示例。
        </p>
      </section>

      <section id='import' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>导入配置</h2>
        <h3 className='mt-6 text-lg font-semibold'>1. Agent 客户端</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          推荐使用{' '}
          <a
            href='https://ccswitch.io/zh/'
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary font-medium underline-offset-4 hover:underline'
          >
            CC Switch
          </a>
          管理 Agent 客户端配置。本站 API Key 页面支持为 Claude Code、Codex 和
          Gemini CLI 发起一键导入，其他已收录客户端可按对应指南手动配置。
        </p>
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button
            variant='outline'
            render={<Link to='/docs/tools/cc-switch' />}
          >
            查看 CC Switch 配置
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          {AGENT_GUIDES.map((guide) => (
            <Button
              key={guide.to}
              variant='ghost'
              render={<Link to={guide.to} />}
            >
              {guide.label}
            </Button>
          ))}
        </div>
        <h3 className='mt-8 text-lg font-semibold'>3. AI 客户端</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          AI 客户端需要根据自身支持的协议填写 Base URL、API Key 和 Model ID。
          请先查看{' '}
          <Link
            to='/docs/api/integration'
            className='text-primary font-medium underline-offset-4 hover:underline'
          >
            API 模型接口
          </Link>
          ，再按客户端的配置说明接入。
        </p>
      </section>
    </DocsShell>
  )
}
