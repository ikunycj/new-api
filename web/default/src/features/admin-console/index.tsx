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
import { lazy, Suspense, useState, type ReactNode } from 'react'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatLocalCurrencyAmount } from '@/lib/currency'

import type { AdminAnalyticsSection } from './admin-analytics'
import { getAdminConsoleStats } from './api'
import {
  AdminConsoleStatCard,
  type AdminConsoleStatTone,
} from './components/admin-console-stat-card'
import type { AdminConsoleStats } from './types'

const LazyAdminAnalytics = lazy(() =>
  import('./admin-analytics').then((module) => ({
    default: module.AdminAnalytics,
  }))
)

function formatCompactNumber(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return new Intl.NumberFormat('zh-CN', {
    notation: Math.abs(value) >= 10_000 ? 'compact' : 'standard',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '0 秒'
  if (seconds < 1) return `${Math.round(seconds * 1000)} 毫秒`
  return `${seconds.toFixed(seconds >= 10 ? 0 : 2)} 秒`
}

interface ConsoleStatItem {
  title: string
  value: ReactNode
  detail: ReactNode
  icon: Parameters<typeof AdminConsoleStatCard>[0]['icon']
  tone: AdminConsoleStatTone
}

function ConsoleCardGrid(props: {
  stats?: AdminConsoleStats
  loading: boolean
}) {
  const stats = props.stats
  const items: ConsoleStatItem[] = [
    {
      title: 'API 密钥',
      value: formatCompactNumber(stats?.api_keys.total ?? 0),
      detail: (
        <>
          <span className='text-success'>
            今日使用 {formatCompactNumber(stats?.api_keys.active ?? 0)}
          </span>
          {' / '}启用 {formatCompactNumber(stats?.api_keys.enabled ?? 0)}
        </>
      ),
      icon: Key01Icon,
      tone: 'chart-1',
    },
    {
      title: '渠道',
      value: formatCompactNumber(stats?.channels.total ?? 0),
      detail: (
        <>
          <span className='text-success'>
            启用 {formatCompactNumber(stats?.channels.enabled ?? 0)}
          </span>
          {' / '}
          <span className='text-destructive'>
            异常 {formatCompactNumber(stats?.channels.auto_disabled ?? 0)}
          </span>
        </>
      ),
      icon: ServerStack01Icon,
      tone: 'chart-2',
    },
    {
      title: '今日请求',
      value: formatCompactNumber(stats?.requests.today ?? 0),
      detail: `累计 ${formatCompactNumber(stats?.requests.total ?? 0)}`,
      icon: ChartUpIcon,
      tone: 'chart-3',
    },
    {
      title: '今日新增用户',
      value: `+${formatCompactNumber(stats?.users.today ?? 0)}`,
      detail: `用户总数 ${formatCompactNumber(stats?.users.total ?? 0)}`,
      icon: UserAdd01Icon,
      tone: 'chart-4',
    },
    {
      title: '今日 Token 用量',
      value: formatCompactNumber(stats?.tokens.today ?? 0),
      detail: `累计 ${formatCompactNumber(stats?.tokens.total ?? 0)}`,
      icon: CubeIcon,
      tone: 'chart-5',
    },
    {
      title: '今日收入',
      value: formatLocalCurrencyAmount(stats?.revenue.today ?? 0),
      detail: `本月 ${formatLocalCurrencyAmount(stats?.revenue.month ?? 0, { compact: true })} / 累计 ${formatLocalCurrencyAmount(stats?.revenue.total ?? 0, { compact: true })}`,
      icon: MoneyReceive01Icon,
      tone: 'chart-3',
    },
    {
      title: '实时性能',
      value: (
        <>
          {formatCompactNumber(stats?.performance.rpm ?? 0)}
          <span className='text-muted-foreground ml-1 text-xs font-medium'>
            RPM
          </span>
        </>
      ),
      detail: `${formatCompactNumber(stats?.performance.tpm ?? 0)} TPM`,
      icon: FlashIcon,
      tone: 'chart-2',
    },
    {
      title: '今日平均响应',
      value: formatDuration(stats?.performance.average_response_seconds ?? 0),
      detail: `活跃用户：今日 ${formatCompactNumber(stats?.users.active_today ?? 0)} / 近 7 天 ${formatCompactNumber(stats?.users.active_week ?? 0)} / 近 30 天 ${formatCompactNumber(stats?.users.active_month ?? 0)}`,
      icon: Clock01Icon,
      tone: 'chart-4',
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

export function AdminConsole() {
  const [activeView, setActiveView] = useState<
    'overview' | AdminAnalyticsSection
  >('overview')
  const statsQuery = useQuery({
    queryKey: ['admin-console-stats'],
    queryFn: getAdminConsoleStats,
    staleTime: 30_000,
    refetchInterval: 60_000,
    placeholderData: (previous) => previous,
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>管理控制台</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <Tabs
            value={activeView}
            onValueChange={(value) => setActiveView(value as typeof activeView)}
          >
            <TabsList>
              <TabsTrigger value='overview'>总览</TabsTrigger>
              <TabsTrigger value='flow'>流量</TabsTrigger>
            </TabsList>
          </Tabs>

          {activeView === 'overview' ? (
            <div className='flex flex-col gap-4'>
              <ConsoleCardGrid
                stats={statsQuery.data}
                loading={statsQuery.isLoading && !statsQuery.data}
              />
              <Suspense
                fallback={
                  <div className='space-y-3'>
                    <Skeleton className='h-9 w-72' />
                    <Skeleton className='h-96 w-full' />
                  </div>
                }
              >
                <LazyAdminAnalytics section='overview' />
              </Suspense>
            </div>
          ) : (
            <Suspense
              fallback={
                <div className='space-y-3'>
                  <Skeleton className='h-9 w-72' />
                  <Skeleton className='h-96 w-full' />
                </div>
              }
            >
              <LazyAdminAnalytics section={activeView} />
            </Suspense>
          )}

          {statsQuery.isError && (
            <Alert variant='destructive'>
              <HugeiconsIcon icon={Alert02Icon} aria-hidden='true' />
              <AlertTitle>统计数据加载失败</AlertTitle>
              <AlertDescription>请稍后重试。</AlertDescription>
            </Alert>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
