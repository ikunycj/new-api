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
  Coins01Icon,
  Key01Icon,
  Layers01Icon,
  MoneyReceive01Icon,
  ServerStack01Icon,
  UserAdd01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { Eye, EyeOff } from 'lucide-react'
import { lazy, Suspense, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { DashboardChartControls } from '@/features/dashboard/components/dashboard-chart-controls'
import { buildDefaultDashboardFilters } from '@/features/dashboard/lib'
import type { DashboardFilters } from '@/features/dashboard/types'
import { formatLocalCurrencyAmount } from '@/lib/currency'
import { formatQuota } from '@/lib/format'

import type { AdminAnalyticsSection } from './admin-analytics'
import {
  getAdminConsoleRealtimeStats,
  getAdminConsoleStats,
  getAdminConsoleSystemLoad,
} from './api'
import {
  AdminConsoleStatCard,
  type AdminConsoleStatTone,
} from './components/admin-console-stat-card'
import type { AdminConsoleDataState } from './types'

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
  loading?: boolean
}

function ConsoleCardGrid(props: { data: AdminConsoleDataState }) {
  const stats = props.data.stats
  const realtimeStats = props.data.realtimeStats
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
      title: '今日响应',
      value: props.data.realtimeStatsError
        ? '--'
        : formatDuration(realtimeStats?.response_seconds ?? 0),
      detail: props.data.statsError
        ? '统计数据加载失败'
        : `P50 ${formatDuration(stats?.performance.today_response_p50_seconds ?? 0)} / P90 ${formatDuration(stats?.performance.today_response_p90_seconds ?? 0)} / P99 ${formatDuration(stats?.performance.today_response_p99_seconds ?? 0)}`,
      icon: Clock01Icon,
      tone: 'chart-4',
      loading: props.data.statsLoading || props.data.realtimeStatsLoading,
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
          loading={item.loading ?? props.data.statsLoading}
        />
      ))}
    </div>
  )
}

export function AdminConsole() {
  const { t } = useTranslation()
  const [activeView, setActiveView] =
    useState<AdminAnalyticsSection>('overview')
  const [modelFilters, setModelFilters] = useState<DashboardFilters>(() =>
    buildDefaultDashboardFilters()
  )
  const [flowSensitiveVisible, setFlowSensitiveVisible] = useState(true)
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
  const realtimeStatsQuery = useQuery({
    queryKey: ['admin-console-realtime'],
    queryFn: getAdminConsoleRealtimeStats,
    enabled: activeView === 'overview',
    staleTime: 5_000,
    refetchInterval: 5_000,
    placeholderData: (previous) => previous,
  })
  const consoleData: AdminConsoleDataState = {
    stats: statsQuery.data,
    realtimeStats: realtimeStatsQuery.data,
    systemLoad: systemLoadQuery.data,
    statsLoading: statsQuery.isLoading && !statsQuery.data,
    realtimeStatsLoading:
      realtimeStatsQuery.isLoading && !realtimeStatsQuery.data,
    systemLoadLoading: systemLoadQuery.isLoading && !systemLoadQuery.data,
    statsError: statsQuery.isError,
    realtimeStatsError: realtimeStatsQuery.isError,
    systemLoadError: systemLoadQuery.isError,
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>管理控制台</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <div className='flex flex-wrap items-center justify-between gap-1.5 sm:gap-2'>
            <Tabs
              value={activeView}
              onValueChange={(value) =>
                setActiveView(value as AdminAnalyticsSection)
              }
            >
              <TabsList>
                <TabsTrigger value='overview'>总览</TabsTrigger>
                <TabsTrigger value='flow'>流量</TabsTrigger>
              </TabsList>
            </Tabs>

            <div className='flex shrink-0 flex-wrap items-center gap-1.5 sm:gap-2'>
              {activeView === 'flow' && (
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() =>
                          setFlowSensitiveVisible((visible) => !visible)
                        }
                        aria-label={
                          flowSensitiveVisible
                            ? t('Hide sensitive data')
                            : t('Show sensitive data')
                        }
                        className='text-muted-foreground hover:text-foreground size-8'
                      />
                    }
                  >
                    {flowSensitiveVisible ? <Eye /> : <EyeOff />}
                  </TooltipTrigger>
                  <TooltipContent>
                    {flowSensitiveVisible
                      ? t('Hide sensitive data')
                      : t('Show sensitive data')}
                  </TooltipContent>
                </Tooltip>
              )}
              <DashboardChartControls
                filters={modelFilters}
                onChange={setModelFilters}
              />
            </div>
          </div>

          {activeView === 'overview' ? (
            <div className='flex flex-col gap-4'>
              <ConsoleCardGrid data={consoleData} />
              <Suspense fallback={<Skeleton className='h-96 w-full' />}>
                <LazyAdminAnalytics
                  section='overview'
                  filters={modelFilters}
                  flowSensitiveVisible={flowSensitiveVisible}
                  adminData={consoleData}
                />
              </Suspense>
            </div>
          ) : (
            <Suspense fallback={<Skeleton className='h-96 w-full' />}>
              <LazyAdminAnalytics
                section={activeView}
                filters={modelFilters}
                flowSensitiveVisible={flowSensitiveVisible}
                adminData={consoleData}
              />
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
