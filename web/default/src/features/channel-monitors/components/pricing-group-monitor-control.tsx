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
import { RefreshIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { ChannelMonitor } from '../types'
import { MonitorStatusBadge } from './monitor-status'

export type PricingGroupMonitorControlProps = {
  groupName: string
  monitor: ChannelMonitor | null
  isPersisted: boolean
  isLoading: boolean
  hasError: boolean
  isUpdating: boolean
  onRetry: () => void
  onConfigure: () => void
  onToggleEnabled: (enabled: boolean) => void
}

export function PricingGroupMonitorControl(
  props: PricingGroupMonitorControlProps
) {
  if (props.isLoading) {
    return (
      <div className='flex w-full items-center gap-2 py-0.5'>
        <Skeleton className='h-5 w-8 shrink-0 rounded-full' />
        <div className='flex min-w-0 flex-1 flex-col gap-2'>
          <Skeleton className='h-5 w-32' />
          <Skeleton className='h-3 w-52 max-w-full' />
        </div>
      </div>
    )
  }

  if (props.hasError) {
    return (
      <div className='flex items-center gap-2 py-0.5'>
        <Badge variant='destructive'>加载失败</Badge>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={props.onRetry}
        >
          <HugeiconsIcon icon={RefreshIcon} data-icon='inline-start' />
          重试
        </Button>
      </div>
    )
  }

  if (!props.monitor) {
    return (
      <div className='flex items-center gap-2 py-0.5'>
        <Badge variant='secondary'>未配置</Badge>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={!props.isPersisted || props.isUpdating}
                aria-label={`配置 ${props.groupName} 的分组监控`}
                onClick={props.onConfigure}
              />
            }
          >
            配置监控
          </TooltipTrigger>
          <TooltipContent>
            {props.isPersisted ? '配置分组监控' : '请先保存定价分组'}
          </TooltipContent>
        </Tooltip>
        {!props.isPersisted && (
          <span className='text-muted-foreground text-xs'>请先保存分组</span>
        )}
      </div>
    )
  }

  return (
    <div className='flex w-full items-center gap-2 py-0.5'>
      <Switch
        size='sm'
        checked={props.monitor.enabled}
        disabled={props.isUpdating}
        aria-label={`${props.monitor.enabled ? '停用' : '启用'} ${props.groupName} 的定时监控`}
        onCheckedChange={props.onToggleEnabled}
      />

      <div className='flex min-w-0 flex-1 flex-wrap items-center gap-1.5'>
        <MonitorStatusBadge status={props.monitor.status} />
        <code
          className='text-muted-foreground min-w-0 flex-1 truncate text-xs'
          title={props.monitor.test_model}
        >
          {props.monitor.test_model}
        </code>
        <Badge variant='outline' className='hidden xl:inline-flex'>
          {props.monitor.visible ? '用户可见' : '对用户隐藏'}
        </Badge>
      </div>
    </div>
  )
}
