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

import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'

type AdminConsoleIcon = ComponentProps<typeof HugeiconsIcon>['icon']

export type AdminConsoleStatTone = Extract<
  IconBadgeTone,
  'chart-1' | 'chart-2' | 'chart-3' | 'chart-4' | 'chart-5'
>

interface AdminConsoleStatCardProps {
  title: string
  value: ReactNode
  detail?: ReactNode
  icon: AdminConsoleIcon
  tone: AdminConsoleStatTone
  loading?: boolean
}

export function AdminConsoleStatCard(props: AdminConsoleStatCardProps) {
  return (
    <Card size='sm' className='min-h-28'>
      <CardHeader>
        <CardTitle className='text-muted-foreground text-xs font-medium'>
          {props.title}
        </CardTitle>
        <CardAction>
          <IconBadge tone={props.tone} size='md'>
            <HugeiconsIcon icon={props.icon} aria-hidden='true' />
          </IconBadge>
        </CardAction>
      </CardHeader>
      <CardContent className='flex min-w-0 flex-1 flex-col justify-end gap-1'>
        {props.loading ? (
          <Skeleton className='h-7 w-28' />
        ) : (
          <div className='text-foreground min-w-0 font-mono text-xl font-semibold tabular-nums'>
            {props.value}
          </div>
        )}
        {props.loading ? (
          <Skeleton className='h-4 w-36' />
        ) : (
          <div className='text-muted-foreground min-h-4 text-xs leading-4'>
            {props.detail}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
