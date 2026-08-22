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
import { HugeiconsIcon } from '@hugeicons/react'
import type { ComponentProps, ReactNode } from 'react'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

type AdminConsoleIcon = ComponentProps<typeof HugeiconsIcon>['icon']

export type AdminConsoleStatTone = 'blue' | 'orange' | 'green' | 'cyan' | 'red'

const TONE_CLASSES: Record<
  AdminConsoleStatTone,
  { icon: string; border: string }
> = {
  blue: {
    icon: 'bg-blue-100 text-blue-600 dark:bg-blue-950/50 dark:text-blue-300',
    border: 'hover:border-blue-200 dark:hover:border-blue-900',
  },
  orange: {
    icon: 'bg-orange-100 text-orange-600 dark:bg-orange-950/50 dark:text-orange-300',
    border: 'hover:border-orange-200 dark:hover:border-orange-900',
  },
  green: {
    icon: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-300',
    border: 'hover:border-emerald-200 dark:hover:border-emerald-900',
  },
  cyan: {
    icon: 'bg-cyan-100 text-cyan-600 dark:bg-cyan-950/50 dark:text-cyan-300',
    border: 'hover:border-cyan-200 dark:hover:border-cyan-900',
  },
  red: {
    icon: 'bg-rose-100 text-rose-600 dark:bg-rose-950/50 dark:text-rose-300',
    border: 'hover:border-rose-200 dark:hover:border-rose-900',
  },
}

interface AdminConsoleStatCardProps {
  title: string
  value: ReactNode
  detail?: ReactNode
  icon: AdminConsoleIcon
  tone: AdminConsoleStatTone
  loading?: boolean
}

export function AdminConsoleStatCard(props: AdminConsoleStatCardProps) {
  const tone = TONE_CLASSES[props.tone]

  return (
    <article
      className={cn(
        'bg-card group flex min-h-32 items-center rounded-lg border-border/70 p-4 shadow-xs transition-colors hover:shadow-sm sm:p-5',
        tone.border
      )}
    >
      <div className='flex w-full items-center gap-3'>
        <div
          className={cn(
            'grid size-10 shrink-0 place-items-center rounded-lg',
            tone.icon
          )}
        >
          <HugeiconsIcon
            icon={props.icon}
            className='size-5'
            aria-hidden='true'
          />
        </div>
        <div className='min-w-0 flex-1'>
          <div className='text-muted-foreground truncate text-sm font-medium'>
            {props.title}
          </div>
          {props.loading ? (
            <Skeleton className='mt-1.5 h-8 w-32' />
          ) : (
            <div className='text-foreground mt-0.5 flex min-w-0 items-baseline font-mono text-2xl font-semibold tabular-nums'>
              {props.value}
            </div>
          )}
          {props.loading ? (
            <Skeleton className='mt-1.5 h-4 w-36' />
          ) : (
            <div className='text-muted-foreground mt-1.5 min-h-4 text-sm leading-5'>
              {props.detail}
            </div>
          )}
        </div>
      </div>
    </article>
  )
}
