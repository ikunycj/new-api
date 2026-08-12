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
import { ArrowRight01Icon, LinkSquare01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import ClaudeCode from '@lobehub/icons/es/ClaudeCode'
import Cline from '@lobehub/icons/es/Cline'
import Codex from '@lobehub/icons/es/Codex'
import Cursor from '@lobehub/icons/es/Cursor'
import GeminiCLI from '@lobehub/icons/es/GeminiCLI'
import HermesAgent from '@lobehub/icons/es/HermesAgent'
import KiloCode from '@lobehub/icons/es/KiloCode'
import OpenClaw from '@lobehub/icons/es/OpenClaw'
import OpenCode from '@lobehub/icons/es/OpenCode'
import type { IconType } from '@lobehub/icons/es/types'
import { Link } from '@tanstack/react-router'

import ccSwitchLogo from '@/assets/home/cc-switch-logo.png'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

import { DocsShell } from './components/docs-shell'

const INTEGRATION_TOC = [
  { id: 'prepare', label: '开始前准备' },
  { id: 'agent-clients', label: 'Agent 客户端' },
  { id: 'ai-clients', label: 'AI 客户端' },
]

type AgentGuide = {
  label: string
  description: string
  to: string
  website: string
  icon?: IconType
  image?: string
}

const AGENT_GUIDES: AgentGuide[] = [
  {
    label: 'CC Switch',
    description: '统一管理多个 Agent 客户端和服务商配置。',
    to: '/docs/tools/cc-switch',
    website: 'https://ccswitch.io/zh/',
    image: ccSwitchLogo,
  },
  {
    label: 'Claude Code',
    description: '在终端中使用 Claude 进行编码和项目协作。',
    to: '/docs/tools/claude-code',
    website: 'https://code.claude.com/',
    icon: ClaudeCode.Color,
  },
  {
    label: 'Codex',
    description: '连接 OpenAI Codex，执行代码任务和工作流。',
    to: '/docs/tools/codex',
    website: 'https://openai.com/codex/',
    icon: Codex.Color,
  },
  {
    label: 'Gemini CLI',
    description: '通过命令行使用 Gemini 模型完成开发任务。',
    to: '/docs/tools/gemini',
    website: 'https://geminicli.com/',
    icon: GeminiCLI.Color,
  },
  {
    label: 'Hermes Agent',
    description: '使用 Hermes Agent 连接统一 API 服务。',
    to: '/docs/tools/hermes',
    website: 'https://hermes-agent.nousresearch.com/',
    icon: HermesAgent,
  },
  {
    label: 'OpenClaw',
    description: '为 OpenClaw 配置自定义模型服务商和接口。',
    to: '/docs/tools/openclaw',
    website: 'https://docs.openclaw.ai/',
    icon: OpenClaw.Color,
  },
  {
    label: 'OpenCode',
    description: '在终端工作流中接入 OpenCode 模型服务商。',
    to: '/docs/tools/opencode',
    website: 'https://opencode.ai/',
    icon: OpenCode,
  },
  {
    label: 'Cursor',
    description: '为 Cursor 配置兼容接口、API Key 和模型。',
    to: '/docs/api/integration',
    website: 'https://cursor.com/',
    icon: Cursor,
  },
  {
    label: 'Cline',
    description: '在 Cline 中接入自定义 OpenAI 兼容服务。',
    to: '/docs/api/integration',
    website: 'https://cline.bot/',
    icon: Cline,
  },
  {
    label: 'Kilo Code',
    description: '使用自定义 Base URL 连接 Kilo Code。',
    to: '/docs/api/integration',
    website: 'https://kilocode.ai/',
    icon: KiloCode,
  },
]

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
        <ol className='border-border mt-5 divide-y border-y'>
          <li className='flex gap-4 py-3 leading-7'>
            <span className='text-muted-foreground w-6 shrink-0 font-mono text-xs'>
              01
            </span>
            <span>在 AllTokenAPI 工作台创建 API Key。</span>
          </li>
          <li className='flex gap-4 py-3 leading-7'>
            <span className='text-muted-foreground w-6 shrink-0 font-mono text-xs'>
              02
            </span>
            <span>从模型列表获取准确的 Model ID。</span>
          </li>
          <li className='flex gap-4 py-3 leading-7'>
            <span className='text-muted-foreground w-6 shrink-0 font-mono text-xs'>
              03
            </span>
            <span>
              确认工具支持的协议类型：OpenAI Compatible、OpenAI
              Responses、Anthropic Claude Messages 或 Gemini Compatible。
            </span>
          </li>
        </ol>
        <p className='border-border bg-muted/30 mt-4 border-l-2 px-4 py-3 leading-7'>
          建议把密钥保存到本地安全位置。文档中的{' '}
          <code>AllTokenAPI_API_KEY</code> 只是占位示例。
        </p>
      </section>

      <section id='agent-clients' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Agent 客户端</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          推荐使用 CC Switch 管理 Agent 客户端配置。本站 API Key 页面支持为
          Claude Code、Codex 和 Gemini CLI
          发起一键导入，其他已收录客户端可按对应指南手动配置。
        </p>
        <div className='mt-5 grid gap-3 sm:grid-cols-2'>
          {AGENT_GUIDES.map((guide) => {
            const Icon = guide.icon
            return (
              <Card
                key={guide.label}
                className='group border-border/70 bg-background hover:border-primary/40 hover:bg-muted/20 h-full min-h-[188px] rounded-lg py-0 shadow-none transition-[border-color,background-color,box-shadow] hover:shadow-sm'
              >
                <CardHeader className='gap-0 p-4'>
                  <div className='flex min-h-[112px] items-start gap-3'>
                    <span className='border-border/70 bg-muted/40 group-hover:bg-background flex size-11 shrink-0 items-center justify-center rounded-md border shadow-xs transition-colors'>
                      {guide.image ? (
                        <img
                          src={guide.image}
                          alt=''
                          width={40}
                          height={40}
                          loading='lazy'
                          decoding='async'
                          className='size-8 rounded-md object-contain'
                        />
                      ) : (
                        Icon && (
                          <Icon width={28} height={28} aria-hidden='true' />
                        )
                      )}
                    </span>
                    <div className='min-w-0 flex-1'>
                      <CardTitle className='truncate text-[0.95rem] leading-6 font-semibold'>
                        {guide.label}
                      </CardTitle>
                      <CardDescription className='text-muted-foreground mt-1.5 line-clamp-2 min-h-10 text-xs leading-5'>
                        {guide.description}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardFooter className='border-border/60 mt-auto flex-wrap gap-2 border-t bg-transparent px-4 py-3'>
                  <Button
                    size='sm'
                    className='flex-1 sm:flex-none'
                    render={<Link to={guide.to} />}
                  >
                    查看接入指南
                    <HugeiconsIcon
                      icon={ArrowRight01Icon}
                      data-icon='inline-end'
                    />
                  </Button>
                  <Button
                    size='sm'
                    variant='outline'
                    className='flex-1 sm:flex-none'
                    render={
                      <a
                        href={guide.website}
                        target='_blank'
                        rel='noopener noreferrer'
                      />
                    }
                  >
                    官网
                    <HugeiconsIcon
                      icon={LinkSquare01Icon}
                      data-icon='inline-end'
                    />
                  </Button>
                </CardFooter>
              </Card>
            )
          })}
        </div>
      </section>

      <section id='ai-clients' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>AI 客户端</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          AI 客户端需要根据自身支持的协议填写 Base URL、API Key 和 Model ID。
          请先查看 API 模型接口，再按客户端的配置说明接入。
        </p>
        <div className='border-border bg-muted/30 mt-5 flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between'>
          <div>
            <p className='font-medium'>统一 API 模型接口</p>
            <p className='text-muted-foreground mt-1 text-sm leading-6'>
              查看鉴权、Base URL、协议和请求示例。
            </p>
          </div>
          <Button
            variant='outline'
            render={<Link to='/docs/api/integration' />}
          >
            查看接口文档
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>
      </section>
    </DocsShell>
  )
}
