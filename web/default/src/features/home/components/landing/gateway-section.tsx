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
import { ApiGatewayIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import Claude from '@lobehub/icons/es/Claude'
import ClaudeCode from '@lobehub/icons/es/ClaudeCode'
import Cline from '@lobehub/icons/es/Cline'
import Codex from '@lobehub/icons/es/Codex'
import Cursor from '@lobehub/icons/es/Cursor'
import DeepSeek from '@lobehub/icons/es/DeepSeek'
import Gemini from '@lobehub/icons/es/Gemini'
import GeminiCLI from '@lobehub/icons/es/GeminiCLI'
import Moonshot from '@lobehub/icons/es/Moonshot'
import OpenAI from '@lobehub/icons/es/OpenAI'
import OpenCode from '@lobehub/icons/es/OpenCode'
import Qwen from '@lobehub/icons/es/Qwen'
import RooCode from '@lobehub/icons/es/RooCode'
import type { IconType } from '@lobehub/icons/es/types'
import Windsurf from '@lobehub/icons/es/Windsurf'
import XAI from '@lobehub/icons/es/XAI'
import Zhipu from '@lobehub/icons/es/Zhipu'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

import { MultiGroupRoutingCard } from './multi-group-routing-card'
import { SectionHeading } from './section-heading'

interface GatewayLogoNode {
  name: string
  icon: IconType
}

const MODEL_PROVIDERS: GatewayLogoNode[] = [
  { name: 'OpenAI', icon: OpenAI },
  { name: 'Claude', icon: Claude.Color },
  { name: 'Gemini', icon: Gemini.Color },
  { name: 'DeepSeek', icon: DeepSeek.Color },
  { name: 'Kimi', icon: Moonshot },
  { name: 'GLM', icon: Zhipu.Color },
  { name: '千问', icon: Qwen.Color },
  { name: 'xAI', icon: XAI },
]

const AGENT_CLIENTS: GatewayLogoNode[] = [
  { name: 'Codex', icon: Codex.Color },
  { name: 'Claude Code', icon: ClaudeCode.Color },
  { name: 'Gemini CLI', icon: GeminiCLI.Color },
  { name: 'Cursor', icon: Cursor },
  { name: 'Cline', icon: Cline },
  { name: 'Roo Code', icon: RooCode },
  { name: 'OpenCode', icon: OpenCode },
  { name: 'Windsurf', icon: Windsurf },
]

function GatewayLogoCard({ node }: { node: GatewayLogoNode }) {
  const Icon = node.icon

  return (
    <div
      className='border-border bg-card text-muted-foreground flex h-16 w-20 flex-col items-center justify-center gap-1.5 rounded-lg border px-2 shadow-xs'
      title={node.name}
    >
      <Icon width={28} height={28} aria-hidden='true' />
      <span className='max-w-full truncate text-[11px] leading-none font-medium'>
        {node.name}
      </span>
    </div>
  )
}

function GatewayLogoGrid({
  nodes,
  direction,
}: {
  nodes: GatewayLogoNode[]
  direction: 'to-gateway' | 'to-agents'
}) {
  return (
    <div
      className={cn(
        'relative z-10 grid grid-cols-3 justify-center gap-3 sm:grid-cols-4 md:grid-cols-[5rem_5rem]',
        direction === 'to-gateway' ? 'md:justify-end' : 'md:justify-start'
      )}
    >
      <span className='bg-border absolute top-8 bottom-8 left-1/2 z-0 hidden w-px md:block' />
      <span
        className={cn(
          'bg-primary/45 absolute top-1/2 z-0 hidden h-px w-[calc(50%+3.5rem)] md:block',
          direction === 'to-gateway' ? 'left-1/2' : 'right-1/2'
        )}
      >
        <span className='text-primary/60 absolute top-1/2 right-0 size-0 translate-x-full -translate-y-1/2 border-y-4 border-l-6 border-y-transparent border-l-current' />
      </span>
      {nodes.map((node, index) => (
        <div key={node.name} className='relative'>
          <span
            className={cn(
              'bg-border absolute top-1/2 z-0 hidden h-px w-1.5 md:block',
              index % 2 === 0
                ? 'right-0 translate-x-full'
                : 'left-0 -translate-x-full'
            )}
          />
          <GatewayLogoCard node={node} />
        </div>
      ))}
      {direction === 'to-gateway' ? (
        <span className='text-primary/45 absolute top-full left-1/2 h-7 w-px bg-current md:hidden' />
      ) : null}
    </div>
  )
}

export function GatewaySection() {
  const { t } = useTranslation()
  const { systemName, logo } = useSystemConfig()
  const displayName = systemName || 'New API'
  const displayLogo = logo || '/logo.png'

  return (
    <section className='px-4 py-20 sm:px-6 sm:py-24 lg:py-28'>
      <div className='mx-auto w-full max-w-6xl'>
        <SectionHeading
          eyebrow={t('Unified gateway')}
          title={t('One gateway interface connects every model')}
          description={t(
            'Connect models from leading providers to the agent tools you already use.'
          )}
          centered
        />

        <div className='border-border relative isolate grid min-h-[31rem] items-center gap-7 border-y py-12 md:grid-cols-[1fr_13rem_1fr] md:gap-14 md:px-12'>
          <GatewayLogoGrid nodes={MODEL_PROVIDERS} direction='to-gateway' />

          <div className='border-primary/60 bg-card after:text-primary/45 relative z-10 mx-auto flex min-h-36 w-full max-w-52 flex-col items-center justify-center rounded-lg border p-5 text-center shadow-xs after:absolute after:top-full after:left-1/2 after:h-7 after:w-px after:bg-current after:content-[""] md:after:hidden'>
            <span className='border-border bg-background flex size-12 items-center justify-center overflow-hidden rounded-lg border'>
              <img
                src={displayLogo}
                alt=''
                className='size-full object-contain'
              />
            </span>
            <strong className='mt-3 max-w-full truncate text-sm'>
              {displayName}
            </strong>
            <span className='text-muted-foreground mt-1 flex items-center gap-1.5 text-xs'>
              <HugeiconsIcon icon={ApiGatewayIcon} className='size-3.5' />
              API Gateway
            </span>
          </div>

          <GatewayLogoGrid nodes={AGENT_CLIENTS} direction='to-agents' />

          <div className='flex flex-wrap justify-center gap-2 md:col-span-3'>
            <Badge variant='outline'>{t('Unified authentication')}</Badge>
            <Badge variant='outline'>{t('Protocol adaptation')}</Badge>
            <Badge variant='outline'>{t('Routing logs')}</Badge>
          </div>
        </div>

        <MultiGroupRoutingCard />
      </div>
    </section>
  )
}
