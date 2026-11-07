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
import { ArrowRight01Icon, Download04Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import ClaudeCode from '@lobehub/icons/es/ClaudeCode'
import Codex from '@lobehub/icons/es/Codex'
import DeepSeek from '@lobehub/icons/es/DeepSeek'
import GeminiCLI from '@lobehub/icons/es/GeminiCLI'
import HermesAgent from '@lobehub/icons/es/HermesAgent'
import OpenClaw from '@lobehub/icons/es/OpenClaw'
import OpenCode from '@lobehub/icons/es/OpenCode'
import type { IconType } from '@lobehub/icons/es/types'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import ccSwitchLogo from '@/assets/home/cc-switch-logo.png'
import piLogo from '@/assets/home/pi-logo.svg'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

type ProviderStateKey = 'Enabled' | 'Available' | 'Ready'

interface ProviderPreview {
  id: string
  mark: string
  name: string
  detail?: string
  detailKey?: string
  stateKey: ProviderStateKey
  selected?: boolean
}

interface SupportedClient {
  id: string
  name: string
  downloadUrl: string
  docsPath: string
  icon?: IconType
  image?: string
}

const PROVIDER_PREVIEWS: ProviderPreview[] = [
  {
    id: 'alltokenapi',
    mark: 'A',
    name: 'API Key',
    detailKey: 'Model gateway address',
    stateKey: 'Enabled',
    selected: true,
  },
  {
    id: 'anthropic',
    mark: 'AN',
    name: 'Anthropic',
    detail: 'Claude Code',
    stateKey: 'Available',
  },
  {
    id: 'openrouter',
    mark: 'OR',
    name: 'OpenRouter',
    detailKey: 'OpenAI Compatible',
    stateKey: 'Available',
  },
  {
    id: 'add-provider',
    mark: '+',
    name: 'Add Provider',
    detailKey: 'Import or configure manually',
    stateKey: 'Ready',
  },
]

const CLIENT_TABS = ['CC', 'CD', 'CX', 'G', 'OC', 'OA', 'H']

const SUPPORTED_CLIENTS: SupportedClient[] = [
  {
    id: 'codex',
    name: 'Codex',
    icon: Codex.Color,
    downloadUrl: 'https://github.com/openai/codex/releases',
    docsPath: '/docs/tools/codex',
  },
  {
    id: 'claude-code',
    name: 'Claude Code',
    icon: ClaudeCode.Color,
    downloadUrl: 'https://github.com/anthropics/claude-code#installation',
    docsPath: '/docs/tools/claude-code',
  },
  {
    id: 'gemini',
    name: 'Gemini',
    icon: GeminiCLI.Color,
    downloadUrl: 'https://github.com/google-gemini/gemini-cli/releases',
    docsPath: '/docs/tools/gemini',
  },
  {
    id: 'hermes',
    name: 'Hermes',
    icon: HermesAgent,
    downloadUrl: 'https://github.com/NousResearch/hermes-agent#quick-install',
    docsPath: '/docs/tools/hermes',
  },
  {
    id: 'openclaw',
    name: 'OpenClaw',
    icon: OpenClaw.Color,
    downloadUrl: 'https://github.com/openclaw/openclaw/releases',
    docsPath: '/docs/tools/openclaw',
  },
  {
    id: 'pi',
    name: 'Pi',
    image: piLogo,
    downloadUrl: 'https://github.com/badlogic/pi-mono/releases',
    docsPath: '/docs/api/integration',
  },
  {
    id: 'deepseek-harness',
    name: 'DeepSeek Harness',
    icon: DeepSeek.Color,
    downloadUrl: 'https://github.com/deepseek-ai/deepseek-harness#installation',
    docsPath: '/docs/api/integration',
  },
  {
    id: 'opencode',
    name: 'OpenCode',
    icon: OpenCode,
    downloadUrl: 'https://github.com/anomalyco/opencode/releases',
    docsPath: '/docs/tools/opencode',
  },
]

function CCSwitchStage() {
  const { t } = useTranslation()

  return (
    <Card className='grid gap-0 overflow-hidden rounded-lg py-0 shadow-sm lg:grid-cols-[0.8fr_1.2fr]'>
      <section className='border-border bg-orange-50/70 p-6 sm:p-8 lg:border-e lg:p-10 dark:bg-orange-950/15'>
        <div className='flex items-center gap-3'>
          <img
            src={ccSwitchLogo}
            alt='CC Switch'
            width={54}
            height={54}
            loading='lazy'
            decoding='async'
            className='size-13 shrink-0'
          />
          <strong className='text-xl'>CC Switch</strong>
        </div>

        <Badge variant='outline' className='bg-background/70 mt-7'>
          <span className='size-1.5 rounded-full bg-orange-500' />
          {t('Desktop client')}
        </Badge>

        <h3 className='mt-5 max-w-md text-2xl leading-tight font-semibold sm:text-[28px]'>
          {t('Manage your AI coding workflow in one place')}
        </h3>
        <p className='text-muted-foreground mt-4 max-w-md text-sm leading-7 sm:text-base'>
          {t(
            'Switch clients, providers, and model configurations in one interface.'
          )}
        </p>

        <div className='mt-7 flex flex-col gap-2.5 sm:flex-row'>
          <Button
            size='lg'
            className='h-11 w-full px-5 sm:w-auto'
            render={<Link to='/keys' />}
          >
            {t('One-click import')}
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button
            variant='outline'
            size='lg'
            className='bg-background/70 h-11 w-full px-5 sm:w-auto'
            render={
              <a
                href='https://github.com/farion1231/cc-switch/releases'
                target='_blank'
                rel='noopener noreferrer'
              />
            }
          >
            <HugeiconsIcon icon={Download04Icon} data-icon='inline-start' />
            {t('Download CC Switch')}
          </Button>
        </div>

        <p className='text-muted-foreground mt-5 text-xs leading-5'>
          {t('Supports macOS 12+ · Windows 10+ · Linux')}
        </p>
      </section>

      <div className='bg-muted/60 p-4 sm:p-6 lg:p-7' aria-hidden='true'>
        <div className='border-border bg-background overflow-hidden rounded-lg border shadow-sm'>
          <div className='border-border flex h-10 items-center gap-1.5 border-b px-3'>
            <span className='size-2 rounded-full bg-red-400' />
            <span className='size-2 rounded-full bg-amber-400' />
            <span className='size-2 rounded-full bg-emerald-500' />
          </div>

          <div className='border-border flex min-h-14 flex-col gap-3 border-b p-3 sm:flex-row sm:items-center sm:justify-between'>
            <div className='flex items-center gap-2 text-sm font-semibold'>
              <img
                src={ccSwitchLogo}
                alt=''
                width={24}
                height={24}
                loading='lazy'
                decoding='async'
                className='size-6'
              />
              <span className='text-primary'>CC Switch</span>
            </div>
            <div className='no-scrollbar flex max-w-full items-center gap-1.5 overflow-x-auto'>
              {CLIENT_TABS.map((tab, index) => (
                <span
                  key={tab}
                  className={cn(
                    'bg-muted text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-md font-mono text-[10px] font-semibold',
                    index === 0 && 'bg-orange-500/10 text-orange-600'
                  )}
                >
                  {tab}
                </span>
              ))}
              <span className='flex size-8 shrink-0 items-center justify-center rounded-full bg-orange-500 text-lg text-white'>
                +
              </span>
            </div>
          </div>

          <div className='grid gap-2.5 p-3 sm:p-4'>
            {PROVIDER_PREVIEWS.map((provider) => (
              <div
                key={provider.id}
                className={cn(
                  'border-border grid min-h-16 grid-cols-[2.25rem_minmax(0,1fr)_auto] items-center gap-3 rounded-lg border p-3',
                  provider.selected && 'border-primary/50 bg-primary/5'
                )}
              >
                <span className='bg-muted text-foreground flex size-9 items-center justify-center rounded-lg font-mono text-[10px] font-semibold'>
                  {provider.mark}
                </span>
                <span className='min-w-0'>
                  <strong className='block truncate text-sm'>
                    {provider.name === 'Add Provider'
                      ? t('Add Provider')
                      : provider.name}
                  </strong>
                  <span className='text-primary mt-0.5 block truncate text-[11px]'>
                    {provider.detailKey
                      ? t(provider.detailKey)
                      : provider.detail}
                  </span>
                </span>
                <span className='text-success flex shrink-0 items-center gap-1.5 text-[11px] font-medium max-sm:col-start-2 max-sm:row-start-2'>
                  <span className='bg-success size-1.5 rounded-full' />
                  {t(provider.stateKey)}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  )
}

function ClientCard({ client }: { client: SupportedClient }) {
  const { t } = useTranslation()
  const Icon = client.icon

  return (
    <Card className='gap-4 rounded-lg p-5'>
      <div className='flex min-w-0 items-center gap-3'>
        <span
          className='flex size-10 shrink-0 items-center justify-center'
          aria-hidden='true'
        >
          {Icon ? (
            <Icon width={36} height={36} aria-hidden='true' />
          ) : (
            <img
              src={client.image ?? ''}
              alt=''
              width={40}
              height={40}
              loading='lazy'
              decoding='async'
              className='size-10'
            />
          )}
        </span>
        <h4 className='min-w-0 text-base leading-snug font-semibold'>
          {client.name}
        </h4>
      </div>
      <div className='border-border flex flex-wrap items-center gap-x-4 gap-y-1 border-t pt-3'>
        <a
          href={client.downloadUrl}
          target='_blank'
          rel='noopener noreferrer'
          aria-label={`${t('Download')} ${client.name}`}
          className='text-muted-foreground hover:text-foreground focus-visible:ring-ring inline-flex min-h-10 items-center gap-1.5 rounded-sm text-sm underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:outline-none'
        >
          <HugeiconsIcon icon={Download04Icon} size={16} aria-hidden='true' />
          {t('Download')}
        </a>
        <Link
          to={client.docsPath}
          aria-label={`${client.name} ${t('Usage documentation')}`}
          className='text-primary focus-visible:ring-ring inline-flex min-h-10 items-center gap-1.5 rounded-sm text-sm underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:outline-none'
        >
          {t('Usage documentation')}
          <HugeiconsIcon icon={ArrowRight01Icon} size={16} aria-hidden='true' />
        </Link>
      </div>
    </Card>
  )
}

export function AiClientsSection() {
  const { t } = useTranslation()

  return (
    <section className='bg-muted/30 px-4 py-16 sm:px-6 sm:py-20 lg:py-24'>
      <div className='mx-auto w-full max-w-6xl'>
        <header className='mb-8 sm:mb-10'>
          <p className='text-primary mb-3 text-xs font-semibold uppercase'>
            {t('AI clients')}
          </p>
          <h2 className='text-foreground max-w-4xl text-3xl leading-tight font-semibold sm:text-4xl'>
            {t('Import popular clients in one click with CC Switch')}
          </h2>
          <div className='mt-4 flex justify-end'>
            <Link
              to='/docs/tools/cc-switch'
              className='text-primary focus-visible:ring-ring inline-flex min-h-10 items-center gap-2 rounded-sm text-end text-sm leading-6 underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:outline-none'
            >
              {t('Learn how to configure CC Switch and import in one click')}
              <HugeiconsIcon
                icon={ArrowRight01Icon}
                size={18}
                className='shrink-0'
                aria-hidden='true'
              />
            </Link>
          </div>
        </header>

        <CCSwitchStage />

        <h3 className='mt-10 text-2xl font-semibold sm:mt-12'>
          {t('Support multiple clients')}
        </h3>
        <div className='mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
          {SUPPORTED_CLIENTS.map((client) => (
            <ClientCard key={client.id} client={client} />
          ))}
        </div>
      </div>
    </section>
  )
}
