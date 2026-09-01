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
  ChartUpIcon,
  DashboardSpeed01Icon,
  Layers01Icon,
} from '@hugeicons/core-free-icons'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { IconBadge } from '@/components/ui/icon-badge'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import {
  AdminConsoleStatCard,
  type AdminConsoleStatTone,
} from '@/features/admin-console/components/admin-console-stat-card'
import type { AdminConsoleDataState } from '@/features/admin-console/types'
import { getUserQuotaDates } from '@/features/dashboard/api'
import { useModelStatCardsConfig } from '@/features/dashboard/hooks/use-dashboard-config'
import {
  buildQueryParams,
  calculateDashboardStats,
  getDefaultDays,
} from '@/features/dashboard/lib'
import type {
  QuotaDataItem,
  DashboardFilters,
} from '@/features/dashboard/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'
import { computeTimeRange } from '@/lib/time'
import { cn } from '@/lib/utils'

interface LogStatCardsBaseProps {
  filters?: DashboardFilters
  onDataUpdate?: (data: QuotaDataItem[], loading: boolean) => void
}

interface UserLogStatCardsProps extends LogStatCardsBaseProps {
  includeAdminData: false
  adminData?: never
}

interface AdminLogStatCardsProps extends LogStatCardsBaseProps {
  includeAdminData: true
  adminData: AdminConsoleDataState
}

type LogStatCardsProps = UserLogStatCardsProps | AdminLogStatCardsProps

interface AdminLogStatItem {
  title: string
  value: ReactNode
  detail: ReactNode
  icon: typeof Activity01Icon
  tone: AdminConsoleStatTone
  loading: boolean
}

const MAX_INLINE_STAT_CHARS = 9

function formatStatNumber(value: number, locale: Intl.LocalesArgument) {
  const fullValue = formatNumber(value, locale)
  const displayValue =
    fullValue.length > MAX_INLINE_STAT_CHARS
      ? formatCompactNumber(value, locale)
      : fullValue

  return {
    displayValue,
    fullValue,
  }
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

export function LogStatCards(props: LogStatCardsProps) {
  const { i18n } = useTranslation()
  const statCardsConfig = useModelStatCardsConfig()
  const [stats, setStats] = useState<{
    totalQuota: number
    totalCount: number
    totalTokens: number
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const [timeRangeMinutes, setTimeRangeMinutes] = useState(0)

  const { filters, onDataUpdate } = props

  useEffect(() => {
    const abortController = new AbortController()
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)

    setError(false)
    onDataUpdate?.([], true)

    const timeRange = computeTimeRange(
      getDefaultDays(filters?.time_granularity),
      filters?.start_timestamp,
      filters?.end_timestamp
    )
    const timeDiff = (timeRange.end_timestamp - timeRange.start_timestamp) / 60
    setTimeRangeMinutes(timeDiff)

    void getUserQuotaDates(
      buildQueryParams(timeRange, filters),
      props.includeAdminData
    )
      .then((res) => {
        if (abortController.signal.aborted) return
        const data = res?.data || []
        setStats(calculateDashboardStats(data))
        onDataUpdate?.(data, false)
      })
      .catch(() => {
        if (abortController.signal.aborted) return
        setStats(null)
        setError(true)
        onDataUpdate?.([], false)
      })
      .finally(() => {
        if (!abortController.signal.aborted) {
          setLoading(false)
        }
      })

    return () => {
      abortController.abort()
    }
  }, [filters, onDataUpdate, props.includeAdminData])

  const adaptedStats = {
    rpm: stats?.totalCount ?? 0,
    quota: stats?.totalQuota ?? 0,
    tpm: stats?.totalTokens ?? 0,
  }

  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const modelConfigs = props.includeAdminData ? [] : statCardsConfig
  const modelItems = modelConfigs.map((config) => {
    const rawValue = config.getValue(adaptedStats, timeRangeMinutes)
    const formatted =
      config.key === 'quota'
        ? {
            displayValue: formatQuota(rawValue),
            fullValue: formatQuota(rawValue),
          }
        : formatStatNumber(rawValue, locale)

    return {
      title: config.title,
      value: formatted.displayValue,
      fullValue: formatted.fullValue,
      desc: config.description,
      icon: config.icon,
      iconTone: config.iconTone,
      loading,
      error,
    }
  })

  if (props.includeAdminData) {
    const adminData = props.adminData
    const adminStats = adminData.stats
    const realtimeStats = adminData.realtimeStats
    const systemLoad = adminData.systemLoad
    const rpm = formatStatNumber(realtimeStats?.rpm ?? 0, locale)
    const tpm = formatStatNumber(realtimeStats?.tpm ?? 0, locale)
    const concurrency = formatStatNumber(
      realtimeStats?.current_concurrency ?? 0,
      locale
    )
    const adminLoading = adminData.statsLoading
    const adminError = adminData.statsError
    const realtimeLoading = adminData.realtimeStatsLoading
    const realtimeError = adminData.realtimeStatsError
    const cardLoading = adminLoading || realtimeLoading
    let systemLoadDetail = '实时采样 · 每 5 秒更新'
    if (adminData.systemLoadLoading) {
      systemLoadDetail = '正在读取实时负载…'
    } else if (adminData.systemLoadError) {
      systemLoadDetail = adminData.systemLoad
        ? '实时负载刷新失败，当前为最近一次数据'
        : '实时负载加载失败'
    }
    const adminItems: AdminLogStatItem[] = [
      {
        title: '实时并发',
        value: realtimeError ? '--' : concurrency.displayValue,
        detail: adminError
          ? '统计数据加载失败'
          : `本月 P50 ${formatCompactNumber(adminStats?.performance.month_concurrency_p50 ?? 0, locale)} / P90 ${formatCompactNumber(adminStats?.performance.month_concurrency_p90 ?? 0, locale)} / P95 ${formatCompactNumber(adminStats?.performance.month_concurrency_p95 ?? 0, locale)}`,
        icon: Activity01Icon,
        tone: 'chart-2',
        loading: cardLoading,
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
        loading: Boolean(adminData.systemLoadLoading && !systemLoad),
      },
      {
        title: '今日 RPM',
        value: realtimeError ? '--' : rpm.displayValue,
        detail: adminError
          ? '统计数据加载失败'
          : `P50 ${formatCompactNumber(adminStats?.performance.today_rpm_p50 ?? 0, locale)} / P90 ${formatCompactNumber(adminStats?.performance.today_rpm_p90 ?? 0, locale)} / P99 ${formatCompactNumber(adminStats?.performance.today_rpm_p99 ?? 0, locale)}`,
        icon: ChartUpIcon,
        tone: 'chart-2',
        loading: cardLoading,
      },
      {
        title: '今日 TPM',
        value: realtimeError ? '--' : tpm.displayValue,
        detail: adminError
          ? '统计数据加载失败'
          : `P50 ${formatCompactNumber(adminStats?.performance.today_tpm_p50 ?? 0, locale)} / P90 ${formatCompactNumber(adminStats?.performance.today_tpm_p90 ?? 0, locale)} / P99 ${formatCompactNumber(adminStats?.performance.today_tpm_p99 ?? 0, locale)}`,
        icon: Layers01Icon,
        tone: 'chart-5',
        loading: cardLoading,
      },
    ]

    return (
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {adminItems.map((item) => (
          <AdminConsoleStatCard
            key={item.title}
            title={item.title}
            value={item.value}
            detail={item.detail}
            icon={item.icon}
            tone={item.tone}
            loading={item.loading}
          />
        ))}
      </div>
    )
  }

  const items = modelItems

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='divide-border/60 grid min-w-0 grid-cols-2 divide-x sm:grid-cols-3 lg:grid-cols-5'>
        {items.map((it, idx) => {
          const Icon = it.icon
          let valueContent
          if (it.loading) {
            valueContent = (
              <div className='mt-1 flex flex-col gap-1 sm:mt-2 sm:gap-1.5'>
                <Skeleton className='h-5 w-16 sm:h-7 sm:w-20' />
                <Skeleton className='hidden h-3.5 w-28 md:block' />
              </div>
            )
          } else if (it.error) {
            valueContent = (
              <>
                <div className='text-muted-foreground mt-1 font-mono text-base leading-tight font-bold tracking-tight tabular-nums sm:mt-2 sm:text-2xl sm:leading-normal'>
                  --
                </div>
                <div className='text-muted-foreground/40 mt-1 hidden text-xs md:block'>
                  {it.desc}
                </div>
              </>
            )
          } else {
            valueContent = (
              <>
                <div
                  className='text-foreground mt-1 max-w-full truncate font-mono text-base leading-tight font-bold tracking-tight tabular-nums sm:mt-2 sm:text-2xl sm:leading-normal'
                  title={it.fullValue}
                >
                  {it.value}
                </div>
                <div className='text-muted-foreground/60 mt-1 hidden text-xs md:block'>
                  {it.desc}
                </div>
              </>
            )
          }

          return (
            <div
              key={it.title}
              className={cn(
                'min-w-0 px-2.5 py-1.5 sm:px-5 sm:py-4',
                idx === items.length - 1 &&
                  items.length % 2 !== 0 &&
                  'col-span-2 sm:col-span-1'
              )}
            >
              <div className='flex min-w-0 items-center gap-1.5 sm:gap-2'>
                <IconBadge
                  tone={it.iconTone}
                  size='stat'
                  className='size-4 rounded-sm sm:size-7 sm:rounded-md [&>svg]:size-2.5 sm:[&>svg]:size-3.5'
                >
                  <Icon />
                </IconBadge>
                <div className='text-muted-foreground truncate text-[11px] leading-4 font-medium tracking-wide uppercase sm:text-xs sm:tracking-wider'>
                  {it.title}
                </div>
              </div>

              {valueContent}
            </div>
          )
        })}
      </div>
    </div>
  )
}
