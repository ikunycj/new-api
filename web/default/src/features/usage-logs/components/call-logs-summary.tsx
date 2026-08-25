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
  Activity01Icon,
  Coins01Icon,
  GaugeIcon,
  TableRowsSplitIcon,
} from '@hugeicons/core-free-icons'
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
import {
  formatBillingCurrencyFromUSD,
  getCurrencyDisplay,
} from '@/lib/currency'

import type { LogAnalyticsSummary } from '../types'

type SummaryIcon = ComponentProps<typeof HugeiconsIcon>['icon']

function formatCompactNumber(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return new Intl.NumberFormat('zh-CN', {
    notation: Math.abs(value) >= 10_000 ? 'compact' : 'standard',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatDuration(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 秒'
  if (value < 1) return `${Math.round(value * 1000)} 毫秒`
  return `${value.toFixed(value >= 10 ? 0 : 1)} 秒`
}

interface SummaryItem {
  label: string
  value: string
  detail: ReactNode
  icon: SummaryIcon
  tone: IconBadgeTone
}

interface CallLogsSummaryProps {
  summary?: LogAnalyticsSummary
  loading: boolean
}

export function CallLogsSummary(props: CallLogsSummaryProps) {
  const summary = props.summary
  const { config } = getCurrencyDisplay()
  const totalSpend = formatBillingCurrencyFromUSD(
    (summary?.total_quota ?? 0) / config.quotaPerUnit
  )
  const items: SummaryItem[] = [
    {
      label: '请求总数',
      value: formatCompactNumber(summary?.total_requests ?? 0),
      detail: `成功 ${formatCompactNumber(summary?.success_requests ?? 0)} / 失败 ${formatCompactNumber(summary?.failed_requests ?? 0)}`,
      icon: TableRowsSplitIcon,
      tone: 'chart-1',
    },
    {
      label: 'Token 总量',
      value: formatCompactNumber(summary?.total_tokens ?? 0),
      detail: `输入 ${formatCompactNumber(summary?.input_tokens ?? 0)} / 输出 ${formatCompactNumber(summary?.output_tokens ?? 0)} / 缓存读取 ${formatCompactNumber(summary?.cache_read_tokens ?? 0)} / 缓存写入 ${formatCompactNumber(summary?.cache_write_tokens ?? 0)}`,
      icon: Activity01Icon,
      tone: 'chart-2',
    },
    {
      label: '实际费用',
      value: totalSpend,
      detail: '按成功请求的实际扣费汇总',
      icon: Coins01Icon,
      tone: 'chart-3',
    },
    {
      label: '平均耗时',
      value: formatDuration(summary?.average_use_time ?? 0),
      detail: `P90 ${formatDuration(summary?.p90_use_time ?? 0)} / P99 ${formatDuration(summary?.p99_use_time ?? 0)}`,
      icon: GaugeIcon,
      tone: 'chart-4',
    },
  ]

  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {items.map((item) => (
        <Card key={item.label} size='sm' className='min-h-28'>
          <CardHeader>
            <CardTitle className='text-muted-foreground text-xs font-medium'>
              {item.label}
            </CardTitle>
            <CardAction>
              <IconBadge tone={item.tone} size='sm'>
                <HugeiconsIcon icon={item.icon} aria-hidden='true' />
              </IconBadge>
            </CardAction>
          </CardHeader>
          <CardContent className='flex min-w-0 flex-1 flex-col justify-end gap-1'>
            {props.loading ? (
              <Skeleton className='h-7 w-24' />
            ) : (
              <div className='font-mono text-xl font-semibold tabular-nums'>
                {item.value}
              </div>
            )}
            {props.loading ? (
              <Skeleton className='h-4 w-36' />
            ) : (
              <p
                className='text-muted-foreground line-clamp-2 text-xs leading-4'
                title={
                  typeof item.detail === 'string' ? item.detail : undefined
                }
              >
                {item.detail}
              </p>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
