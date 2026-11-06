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
import { ApiGatewayIcon, ArrowRight01Icon } from '@hugeicons/core-free-icons'
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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import clientModelsBackdrop from '@/assets/home/client-models-backdrop.webp'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
    <section className='px-4 py-16 sm:px-6 sm:py-20 lg:py-24'>
      <div className='mx-auto w-full max-w-6xl'>
        <SectionHeading
          eyebrow={t('Unified gateway')}
          title={t('One gateway interface connects every model')}
          centered
        />

        <div className='border-border bg-card grid overflow-hidden rounded-xl border shadow-sm lg:grid-cols-[0.9fr_1.1fr]'>
          <div className='flex min-w-0 flex-col justify-center p-6 sm:p-8 lg:p-10'>
            <div className='text-foreground flex flex-wrap items-center gap-x-5 gap-y-3 text-sm font-semibold'>
              <span className='inline-flex items-center gap-2'>
                <Codex.Color width={24} height={24} aria-hidden='true' />
                Codex
              </span>
              <span className='inline-flex items-center gap-2'>
                <ClaudeCode.Color width={24} height={24} aria-hidden='true' />
                Claude Code
              </span>
            </div>
            <h3 className='text-foreground mt-6 text-3xl leading-tight font-semibold text-balance sm:text-4xl'>
              {t(
                'Codex, Claude Code, and any other client can connect to any model without local routing conversion'
              )}
            </h3>
            <ul className='text-muted-foreground mt-5 list-disc space-y-2 pl-5 text-sm leading-7 sm:text-base'>
              <li>
                {t(
                  'All models on this site can be used in clients using any API protocol, without local routing or forwarding.'
                )}
              </li>
              <li>
                {t(
                  'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.'
                )}
              </li>
            </ul>
            <p className='text-muted-foreground mt-5 flex flex-wrap gap-x-3 gap-y-1 text-xs leading-5'>
              <span>OpenAI Chat / Responses</span>
              <span>Anthropic Messages</span>
              <span>Gemini generateContent</span>
            </p>
            <div className='mt-8'>
              <Button
                size='lg'
                nativeButton={false}
                className='h-auto min-h-11 max-w-full px-4 py-3 whitespace-normal'
                render={<Link to='/docs/client-models' />}
              >
                {t('Configure multi-provider models in your client')}
                <HugeiconsIcon
                  icon={ArrowRight01Icon}
                  data-icon='inline-end'
                  aria-hidden='true'
                />
              </Button>
            </div>
          </div>
          <div className='border-border bg-muted/30 relative aspect-[827/750] overflow-hidden border-t lg:aspect-auto lg:border-t-0 lg:border-l'>
            <img
              src={clientModelsBackdrop}
              alt=''
              width={1654}
              height={1500}
              loading='lazy'
              decoding='async'
              className='absolute inset-0 size-full object-cover object-bottom'
            />
          </div>
        </div>

        <div className='border-border relative isolate mt-8 grid min-h-[31rem] items-center gap-7 border-y py-12 md:grid-cols-[1fr_13rem_1fr] md:gap-14 md:px-5'>
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
