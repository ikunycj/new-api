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
  Clock01Icon,
  Layers01Icon,
} from '@hugeicons/core-free-icons'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getAdminConsoleRealtimeStats,
  getAdminConsoleStats,
} from '@/features/admin-console/api'
import {
  AdminConsoleStatCard,
  type AdminConsoleStatTone,
} from '@/features/admin-console/components/admin-console-stat-card'
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
import { useAuthStore } from '@/stores/auth-store'

interface LogStatCardsProps {
  filters?: DashboardFilters
  onDataUpdate?: (data: QuotaDataItem[], loading: boolean) => void
  includeAdminData?: boolean
}

interface AdminLogStatItem {
  title: string
  value: string
  detail: string
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

function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '0 秒'
  if (seconds < 1) return `${Math.round(seconds * 1000)} 毫秒`
  return `${seconds.toFixed(seconds >= 10 ? 0 : 2)} 秒`
}

export function LogStatCards(props: LogStatCardsProps) {
  const { i18n } = useTranslation()
  const statCardsConfig = useModelStatCardsConfig()
  const user = useAuthStore((state) => state.auth.user)
  const includeAdminData =
    props.includeAdminData ?? !!(user?.role && user.role >= 10)
  const adminStatsQuery = useQuery({
    queryKey: ['admin-console-stats'],
    queryFn: getAdminConsoleStats,
    enabled: includeAdminData,
    staleTime: 30_000,
    refetchInterval: 60_000,
    placeholderData: (previous) => previous,
  })
  const realtimeStatsQuery = useQuery({
    queryKey: ['admin-console-realtime'],
    queryFn: getAdminConsoleRealtimeStats,
    enabled: includeAdminData,
    staleTime: 5_000,
    refetchInterval: 5_000,
    placeholderData: (previous) => previous,
  })
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
      includeAdminData
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
  }, [filters, includeAdminData, onDataUpdate])

  const adaptedStats = {
    rpm: stats?.totalCount ?? 0,
    quota: stats?.totalQuota ?? 0,
    tpm: stats?.totalTokens ?? 0,
  }

  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const modelConfigs = includeAdminData ? [] : statCardsConfig
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

  if (includeAdminData) {
    const adminStats = adminStatsQuery.data
    const realtimeStats = realtimeStatsQuery.data
    const rpm = formatStatNumber(realtimeStats?.rpm ?? 0, locale)
    const tpm = formatStatNumber(realtimeStats?.tpm ?? 0, locale)
    const concurrency = formatStatNumber(
      realtimeStats?.current_concurrency ?? 0,
      locale
    )
    const adminLoading = adminStatsQuery.isLoading && !adminStats
    const adminError = adminStatsQuery.isError
    const realtimeLoading =
      realtimeStatsQuery.isLoading && !realtimeStatsQuery.data
    const realtimeError = realtimeStatsQuery.isError
    const cardLoading = adminLoading || realtimeLoading
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
        title: '今日响应',
        value: realtimeError
          ? '--'
          : formatDuration(realtimeStats?.response_seconds ?? 0),
        detail: adminError
          ? '统计数据加载失败'
          : `P50 ${formatDuration(adminStats?.performance.today_response_p50_seconds ?? 0)} / P90 ${formatDuration(adminStats?.performance.today_response_p90_seconds ?? 0)} / P99 ${formatDuration(adminStats?.performance.today_response_p99_seconds ?? 0)}`,
        icon: Clock01Icon,
        tone: 'chart-4',
        loading: cardLoading,
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
