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
  Coins01Icon,
  DashboardSpeed01Icon,
  Key01Icon,
  Layers01Icon,
  MoneyReceive01Icon,
  ServerStack01Icon,
  UserAdd01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { lazy, Suspense, useState, type ReactNode } from 'react'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatLocalCurrencyAmount } from '@/lib/currency'
import { formatQuota } from '@/lib/format'

import type { AdminAnalyticsSection } from './admin-analytics'
import { getAdminConsoleStats, getAdminConsoleSystemLoad } from './api'
import {
  AdminConsoleStatCard,
  type AdminConsoleStatTone,
} from './components/admin-console-stat-card'
import type { AdminConsoleStats, AdminConsoleSystemLoad } from './types'

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

function formatTokenAmount(value: number): string {
  if (!Number.isFinite(value)) return '0.00M'
  const absolute = Math.abs(value)
  const divisor = absolute >= 1_000_000_000 ? 1_000_000_000 : 1_000_000
  const suffix = divisor === 1_000_000_000 ? 'B' : 'M'
  return `${(value / divisor).toFixed(2)}${suffix}`
}

function normalizePercentage(value: number): number {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

function formatPercentage(value: number): string {
  return `${new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: 1,
  }).format(normalizePercentage(value))}%`
}

function SystemLoadMetric(props: { label: string; value: number }) {
  const percentage = normalizePercentage(props.value)

  return (
    <Progress
      value={percentage}
      className='gap-1'
      aria-label={`${props.label}使用率 ${formatPercentage(percentage)}`}
    >
      <div className='flex w-full min-w-0 items-baseline justify-between gap-1'>
        <span className='text-muted-foreground truncate font-sans text-[10px] font-medium'>
          {props.label}
        </span>
        <span className='font-mono text-xs font-semibold tabular-nums'>
          {formatPercentage(percentage)}
        </span>
      </div>
    </Progress>
  )
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
  systemLoad?: AdminConsoleSystemLoad
  systemLoadLoading: boolean
  systemLoadError: boolean
  loading: boolean
}) {
  const stats = props.stats
  const systemLoad = props.systemLoad ?? stats?.system_load
  let systemLoadDetail = props.systemLoadLoading
    ? '正在读取实时负载…'
    : '实时采样 · 每 5 秒更新'
  if (props.systemLoadError) {
    systemLoadDetail = props.systemLoad
      ? '实时负载刷新失败，当前为最近一次数据'
      : '实时负载加载失败'
  }
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
      title: '系统负载',
      value: (
        <div className='grid grid-cols-3 gap-2 font-sans'>
          <SystemLoadMetric
            label='CPU'
            value={systemLoad?.cpu_usage_percent ?? 0}
          />
          <SystemLoadMetric
            label='内存'
            value={systemLoad?.memory_usage_percent ?? 0}
          />
          <SystemLoadMetric
            label='存储'
            value={systemLoad?.storage_usage_percent ?? 0}
          />
        </div>
      ),
      detail: systemLoadDetail,
      icon: DashboardSpeed01Icon,
      tone: 'chart-5',
    },
    {
      title: '今日新增用户',
      value: `+${formatCompactNumber(stats?.users.today ?? 0)}`,
      detail: (
        <div
          className='truncate'
          title={`用户总数 ${formatCompactNumber(stats?.users.total ?? 0)} · 活跃 今日 ${formatCompactNumber(stats?.users.active_today ?? 0)} / 近 7 天 ${formatCompactNumber(stats?.users.active_week ?? 0)} / 近 30 天 ${formatCompactNumber(stats?.users.active_month ?? 0)}`}
        >
          用户总数 {formatCompactNumber(stats?.users.total ?? 0)} · 活跃 今{' '}
          {formatCompactNumber(stats?.users.active_today ?? 0)} / 7天{' '}
          {formatCompactNumber(stats?.users.active_week ?? 0)} / 30天{' '}
          {formatCompactNumber(stats?.users.active_month ?? 0)}
        </div>
      ),
      icon: UserAdd01Icon,
      tone: 'chart-4',
    },
    {
      title: '今日请求',
      value: formatCompactNumber(stats?.requests.today ?? 0),
      detail: `本月 ${formatCompactNumber(stats?.requests.month ?? 0)} / 累计 ${formatCompactNumber(stats?.requests.total ?? 0)}`,
      icon: ChartUpIcon,
      tone: 'chart-3',
    },
    {
      title: '今日收入',
      value: formatLocalCurrencyAmount(stats?.revenue.today ?? 0),
      detail: `本月 ${formatLocalCurrencyAmount(stats?.revenue.month ?? 0, { compact: true })} / 累计 ${formatLocalCurrencyAmount(stats?.revenue.total ?? 0, { compact: true })}`,
      icon: MoneyReceive01Icon,
      tone: 'chart-3',
    },
    {
      title: '扣费额度',
      value: formatQuota(stats?.quota.today ?? 0),
      detail: `本月 ${formatQuota(stats?.quota.month ?? 0)} / 累计 ${formatQuota(stats?.quota.total ?? 0)}`,
      icon: Coins01Icon,
      tone: 'chart-2',
    },
    {
      title: 'Token 数',
      value: formatTokenAmount(stats?.tokens.today ?? 0),
      detail: `本月 ${formatTokenAmount(stats?.tokens.month ?? 0)} / 累计 ${formatTokenAmount(stats?.tokens.total ?? 0)}`,
      icon: Layers01Icon,
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
  const systemLoadQuery = useQuery({
    queryKey: ['admin-console-system-load'],
    queryFn: getAdminConsoleSystemLoad,
    enabled: activeView === 'overview',
    staleTime: 5_000,
    refetchInterval: 5_000,
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
                systemLoad={systemLoadQuery.data}
                systemLoadLoading={
                  systemLoadQuery.isLoading && !systemLoadQuery.data
                }
                systemLoadError={systemLoadQuery.isError}
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
