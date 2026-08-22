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
  Alert02Icon,
  ChartUpIcon,
  Clock01Icon,
  CubeIcon,
  FlashIcon,
  Key01Icon,
  MoneyReceive01Icon,
  ServerStack01Icon,
  UserAdd01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'

import { SectionPageLayout } from '@/components/layout'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber } from '@/lib/format'

import { getAdminConsoleStats } from './api'
import { AdminConsoleStatCard } from './components/admin-console-stat-card'
import type { AdminConsoleStats } from './types'

function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '0.00s'
  return `${seconds.toFixed(2)}s`
}

function formatConsoleCompactNumber(value: number): string {
  const absolute = Math.abs(value)
  if (absolute >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2).replace(/\.00$/, '')}B`
  }
  if (absolute >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2).replace(/\.00$/, '')}M`
  }
  if (absolute >= 1_000) {
    return `${(value / 1_000).toFixed(2).replace(/\.00$/, '')}K`
  }
  return formatNumber(value)
}

function formatTokenNumber(value: number): string {
  return value === 0 ? '0M' : formatConsoleCompactNumber(value)
}

function formatRevenue(amount: number): string {
  const value = Number.isFinite(amount) && amount > 0 ? amount : 0
  return `${new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: 2,
  }).format(value)}￥`
}

function ConsoleCardGrid(props: {
  stats?: AdminConsoleStats
  loading: boolean
}) {
  const stats = props.stats
  const items = [
    {
      title: 'API 密钥',
      value: formatNumber(stats?.api_keys.total ?? 0),
      detail: (
        <span>
          <span className='text-emerald-600 dark:text-emerald-400'>
            正在使用 {formatNumber(stats?.api_keys.active ?? 0)}
          </span>{' '}
          <span className='text-muted-foreground'>
            / 启用 {formatNumber(stats?.api_keys.enabled ?? 0)}
          </span>
        </span>
      ),
      icon: Key01Icon,
      tone: 'blue' as const,
    },
    {
      title: '渠道',
      value: formatNumber(stats?.accounts.total ?? 0),
      detail: (
        <span>
          <span className='text-emerald-600 dark:text-emerald-400'>
            {formatNumber(stats?.accounts.enabled ?? 0)} 启用
          </span>{' '}
          <span className='text-rose-600 dark:text-rose-400'>
            {formatNumber(stats?.accounts.auto_disabled ?? 0)} 错误
          </span>
        </span>
      ),
      icon: ServerStack01Icon,
      tone: 'orange' as const,
    },
    {
      title: '今日请求',
      value: formatNumber(stats?.requests.today ?? 0),
      detail: `总计: ${formatNumber(stats?.requests.total ?? 0)}`,
      icon: ChartUpIcon,
      tone: 'green' as const,
    },
    {
      title: '用户',
      value: (
        <span className='text-emerald-600 dark:text-emerald-400'>{`+${formatNumber(stats?.users.today ?? 0)}`}</span>
      ),
      detail: `总计: ${formatNumber(stats?.users.total ?? 0)}`,
      icon: UserAdd01Icon,
      tone: 'green' as const,
    },
    {
      title: 'Token 用量',
      value: (
        <span className='flex min-w-0 flex-nowrap items-baseline gap-x-1 text-sm leading-tight whitespace-nowrap'>
          <span className='whitespace-nowrap'>
            今日 {formatTokenNumber(stats?.tokens.today ?? 0)}
          </span>
          <span className='text-muted-foreground font-medium'>/</span>
          <span className='text-muted-foreground font-medium whitespace-nowrap'>
            总共 {formatTokenNumber(stats?.tokens.total ?? 0)}
          </span>
        </span>
      ),
      icon: CubeIcon,
      tone: 'orange' as const,
    },
    {
      title: '今日收入',
      value: formatRevenue(stats?.revenue.today ?? 0),
      detail: (
        <span>
          <span className='text-emerald-600 dark:text-emerald-400'>
            月收入 {formatRevenue(stats?.revenue.month ?? 0)}
          </span>{' '}
          <span className='text-muted-foreground'>
            / 总收入 {formatRevenue(stats?.revenue.total ?? 0)}
          </span>
        </span>
      ),
      icon: MoneyReceive01Icon,
      tone: 'green' as const,
    },
    {
      title: '性能指标',
      value: (
        <>
          {formatNumber(stats?.performance.rpm ?? 0)}
          <span className='text-muted-foreground ml-1 text-sm font-medium'>
            RPM
          </span>
        </>
      ),
      detail: (
        <>
          <span className='font-semibold text-teal-600 dark:text-teal-400'>
            {formatConsoleCompactNumber(stats?.performance.tpm ?? 0)}
          </span>{' '}
          TPM
        </>
      ),
      icon: FlashIcon,
      tone: 'cyan' as const,
    },
    {
      title: '平均响应',
      value: formatDuration(stats?.performance.average_response_seconds ?? 0),
      detail: (
        <span className='text-xs whitespace-nowrap'>
          {formatNumber(stats?.users.active_today ?? 0)} 活跃用户 / 周活{' '}
          {formatNumber(stats?.users.active_week ?? 0)} / 月活{' '}
          {formatNumber(stats?.users.active_month ?? 0)}
        </span>
      ),
      icon: Clock01Icon,
      tone: 'red' as const,
    },
  ]

  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {items.map((item) => (
        <AdminConsoleStatCard
          key={item.title}
          title={item.title}
          value={item.value}
          detail={item.detail}
          icon={item.icon}
          tone={item.tone}
          loading={props.loading}
        />
      ))}
    </div>
  )
}

function ConsoleLoadingState() {
  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {Array.from({ length: 8 }, (_, index) => (
        <div
          key={index}
          className='bg-card flex min-h-32 items-center rounded-lg border p-4 shadow-xs sm:p-5'
        >
          <div className='flex w-full items-center gap-3'>
            <Skeleton className='size-10 rounded-lg' />
            <div className='min-w-0 flex-1'>
              <Skeleton className='h-4 w-16' />
              <Skeleton className='mt-1.5 h-8 w-28' />
              <Skeleton className='mt-1.5 h-4 w-32' />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

export function AdminConsole() {
  const statsQuery = useQuery({
    queryKey: ['admin-console-stats'],
    queryFn: getAdminConsoleStats,
    staleTime: 30_000,
    refetchInterval: 60_000,
    placeholderData: (previous) => previous,
  })
  const stats = statsQuery.data?.data

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>管理控制台</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div>
          {statsQuery.isLoading && !stats ? (
            <ConsoleLoadingState />
          ) : (
            <ConsoleCardGrid
              stats={stats}
              loading={statsQuery.isFetching && !stats}
            />
          )}
          {statsQuery.isError && (
            <div className='mt-4 flex items-center gap-2 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-300'>
              <HugeiconsIcon
                icon={Alert02Icon}
                className='size-4 shrink-0'
                aria-hidden='true'
              />
              <span>统计数据加载失败，请稍后重试。</span>
            </div>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
