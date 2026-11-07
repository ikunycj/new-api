/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

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
  Cancel01Icon,
  CheckmarkCircle02Icon,
  DragDropVerticalIcon,
  Key01Icon,
  Route01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import Claude from '@lobehub/icons/es/Claude'
import OpenAI from '@lobehub/icons/es/OpenAI'
import type { IconType } from '@lobehub/icons/es/types'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

interface DemoGroup {
  icon: IconType
  name: string
  priority: number
  ratio: string
}

type RouteStatus = 'selected' | 'standby' | 'unavailable'

const DEMO_GROUPS: DemoGroup[] = [
  { name: 'ChatGPT Plus', ratio: '0.01x', priority: 1, icon: OpenAI },
  { name: 'Claude Economy', ratio: '0.02x', priority: 2, icon: Claude.Color },
  { name: 'ChatGPT Pro', ratio: '0.05x', priority: 3, icon: OpenAI },
  { name: 'Claude Priority', ratio: '0.08x', priority: 4, icon: Claude.Color },
]

function PriorityGroupRow({ group }: { group: DemoGroup }) {
  const { t } = useTranslation()
  const Icon = group.icon

  return (
    <div className='border-border bg-background grid min-h-14 grid-cols-[auto_auto_minmax(0,1fr)_auto] items-center gap-2.5 rounded-lg border px-3 py-2.5 shadow-xs'>
      <HugeiconsIcon
        icon={DragDropVerticalIcon}
        className='text-muted-foreground size-4'
        aria-hidden='true'
      />
      <span
        className='bg-muted text-muted-foreground flex size-7 items-center justify-center rounded-md font-mono text-xs font-semibold tabular-nums'
        aria-label={t('Priority {{priority}}', {
          priority: group.priority,
        })}
      >
        {group.priority}
      </span>
      <span className='flex min-w-0 items-center gap-2.5'>
        <span className='border-border bg-card flex size-8 shrink-0 items-center justify-center rounded-md border'>
          <Icon width={20} height={20} aria-hidden='true' />
        </span>
        <span className='truncate text-sm font-medium'>{group.name}</span>
      </span>
      <Badge variant='outline' className='font-mono tabular-nums'>
        {group.ratio}
      </Badge>
    </div>
  )
}

function RouteGroupNode({
  group,
  status,
}: {
  group: DemoGroup
  status: RouteStatus
}) {
  const { t } = useTranslation()
  const Icon = group.icon
  let statusLabel = t('Standby')
  if (status === 'selected') statusLabel = t('Selected')
  if (status === 'unavailable') statusLabel = t('Unavailable')

  return (
    <div
      className={cn(
        'border-border bg-background flex min-w-0 items-center gap-2.5 rounded-lg border p-3',
        status === 'selected' && 'border-primary/45 bg-primary/5',
        status === 'unavailable' && 'border-dashed opacity-60'
      )}
    >
      <span className='border-border bg-card flex size-9 shrink-0 items-center justify-center rounded-md border'>
        <Icon width={22} height={22} aria-hidden='true' />
      </span>
      <span className='min-w-0 flex-1'>
        <span className='block truncate text-sm font-medium'>{group.name}</span>
        <span className='text-muted-foreground mt-1 block font-mono text-[11px] tabular-nums'>
          #{group.priority} · {group.ratio}
        </span>
      </span>
      <span
        className={cn(
          'inline-flex shrink-0 items-center gap-1 text-[11px] font-medium',
          status === 'selected' && 'text-primary',
          status === 'unavailable' && 'text-destructive',
          status === 'standby' && 'text-muted-foreground'
        )}
      >
        {status === 'selected' ? (
          <HugeiconsIcon
            icon={CheckmarkCircle02Icon}
            className='size-3.5'
            aria-hidden='true'
          />
        ) : null}
        {status === 'unavailable' ? (
          <HugeiconsIcon
            icon={Cancel01Icon}
            className='size-3.5'
            aria-hidden='true'
          />
        ) : null}
        {statusLabel}
      </span>
    </div>
  )
}

function ModelRoute({
  model,
  first,
  firstStatus,
  second,
  transitionLabel,
}: {
  model: string
  first: DemoGroup
  firstStatus: RouteStatus
  second: DemoGroup
  transitionLabel: string
}) {
  return (
    <div className='border-border bg-card rounded-lg border p-3.5'>
      <div className='mb-3 flex items-center gap-2'>
        <code className='bg-muted max-w-full truncate rounded-md px-2 py-1 text-xs font-semibold'>
          {model}
        </code>
      </div>
      <div className='grid items-center gap-2.5 xl:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]'>
        <RouteGroupNode group={first} status={firstStatus} />
        <div className='text-primary flex items-center justify-center gap-1.5 text-[11px] font-medium xl:flex-col xl:gap-0.5'>
          <HugeiconsIcon
            icon={ArrowRight01Icon}
            className='size-4 rotate-90 xl:rotate-0'
            aria-hidden='true'
          />
          <span>{transitionLabel}</span>
        </div>
        <RouteGroupNode
          group={second}
          status={firstStatus === 'unavailable' ? 'selected' : 'standby'}
        />
      </div>
    </div>
  )
}

export function MultiGroupRoutingCard() {
  const { t } = useTranslation()

  return (
    <Card className='mt-8 gap-0 rounded-lg py-0 shadow-sm'>
      <div className='border-border flex flex-col gap-4 border-b p-5 sm:flex-row sm:items-start sm:justify-between sm:p-7'>
        <div className='flex items-start gap-3'>
          <span className='bg-primary/10 text-primary flex size-10 shrink-0 items-center justify-center rounded-lg'>
            <HugeiconsIcon icon={Key01Icon} className='size-5' />
          </span>
          <div>
            <p className='text-primary text-xs font-semibold uppercase'>
              {t('Model group routing')}
            </p>
            <h3 className='mt-1.5 text-xl font-semibold sm:text-2xl'>
              {t('One API key, multiple model groups')}
            </h3>
            <p className='text-muted-foreground mt-2 max-w-2xl text-sm leading-6'>
              {t(
                'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.'
              )}
            </p>
          </div>
        </div>
        <Badge variant='outline' className='w-fit shrink-0'>
          {t('Example configuration')}
        </Badge>
      </div>

      <div className='grid gap-0 lg:grid-cols-[0.82fr_1.18fr]'>
        <section className='border-border bg-muted/25 p-5 sm:p-7 lg:border-e'>
          <div className='flex items-center justify-between gap-3'>
            <div>
              <p className='text-sm font-semibold'>
                {t('Selected group order')}
              </p>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t('Drag to change priority')}
              </p>
            </div>
            <Badge variant='secondary'>
              {DEMO_GROUPS.length} {t('Groups')}
            </Badge>
          </div>

          <div className='border-border bg-card mt-4 flex items-center gap-2 rounded-lg border px-3 py-2.5'>
            <HugeiconsIcon
              icon={Key01Icon}
              className='text-muted-foreground size-4'
              aria-hidden='true'
            />
            <span className='text-muted-foreground text-xs'>
              {t('API Key')}
            </span>
            <code className='ml-auto font-mono text-xs'>
              sk-all-models-••••••
            </code>
          </div>

          <div className='mt-3 space-y-2'>
            {DEMO_GROUPS.map((group) => (
              <PriorityGroupRow key={group.name} group={group} />
            ))}
          </div>
        </section>

        <section className='p-5 sm:p-7'>
          <div className='flex items-start gap-3'>
            <span className='bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-lg'>
              <HugeiconsIcon icon={Route01Icon} className='size-4.5' />
            </span>
            <div>
              <p className='text-sm font-semibold'>
                {t('Model-aware routing')}
              </p>
              <p className='text-muted-foreground mt-1 text-xs leading-5'>
                {t('The request only tries groups that support its model.')}
              </p>
            </div>
          </div>

          <div className='mt-4 space-y-3'>
            <ModelRoute
              model='gpt-6-astra'
              first={DEMO_GROUPS[0]}
              firstStatus='unavailable'
              second={DEMO_GROUPS[2]}
              transitionLabel={t('Automatic fallback')}
            />
            <ModelRoute
              model='claude-sonnet-5'
              first={DEMO_GROUPS[1]}
              firstStatus='selected'
              second={DEMO_GROUPS[3]}
              transitionLabel={t('Next in order')}
            />
          </div>
        </section>
      </div>

      <div className='border-border bg-muted/35 flex items-start gap-2.5 border-t px-5 py-4 sm:px-7'>
        <HugeiconsIcon
          icon={Route01Icon}
          className='text-primary mt-0.5 size-4 shrink-0'
          aria-hidden='true'
        />
        <p className='text-muted-foreground text-xs leading-5'>
          {t(
            'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.'
          )}
        </p>
      </div>
    </Card>
  )
}
