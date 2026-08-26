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
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Users, Loader2 } from 'lucide-react'
import { useEffect, useMemo, useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTheme } from '@/context/theme-provider'
import { getUserQuotaDataByUsers } from '@/features/dashboard/api'
import { processUserChartData } from '@/features/dashboard/lib'
import type {
  DashboardFilters,
  ProcessedUserChartData,
  QuotaDataItem,
} from '@/features/dashboard/types'
import { VCHART_OPTION } from '@/lib/vchart'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

const USER_CHARTS: {
  value: string
  labelKey: string
  specKey: keyof ProcessedUserChartData
}[] = [
  {
    value: 'rank',
    labelKey: 'User Consumption Ranking',
    specKey: 'spec_user_rank',
  },
  {
    value: 'trend',
    labelKey: 'User Consumption Trend',
    specKey: 'spec_user_trend',
  },
]

const TOP_USER_LIMIT_OPTIONS = [5, 10, 20, 50]

interface UserChartsProps {
  filters: DashboardFilters
  compact?: boolean
  dataOverride?: QuotaDataItem[]
}

export function UserCharts(props: UserChartsProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  // The selection is owned by the dashboard parent so it persists across
  // sub-section switches; the rolling window is derived from the chosen range.
  const timeGranularity = props.filters.time_granularity ?? 'hour'
  const [topUserLimit, setTopUserLimit] = useState(10)

  const timeRange = useMemo(() => {
    return {
      start_timestamp: Math.floor(
        (props.filters.start_timestamp?.getTime() ?? Date.now() - 86_400_000) /
          1000
      ),
      end_timestamp: Math.floor(
        (props.filters.end_timestamp?.getTime() ?? Date.now()) / 1000
      ),
    }
  }, [props.filters.end_timestamp, props.filters.start_timestamp])

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    updateTheme()
  }, [resolvedTheme])

  const { data: userData, isLoading } = useQuery({
    queryKey: ['dashboard', 'user-quota', timeRange],
    queryFn: () => getUserQuotaDataByUsers(timeRange),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
  })
  const chartLoading = isLoading && !props.dataOverride
  const chartSourceData = useMemo(
    () => props.dataOverride ?? userData ?? [],
    [props.dataOverride, userData]
  )

  const chartData = useMemo(
    () =>
      processUserChartData(
        chartLoading ? [] : chartSourceData,
        timeGranularity,
        t,
        topUserLimit,
        props.filters.metric
      ),
    [
      chartSourceData,
      chartLoading,
      timeGranularity,
      t,
      topUserLimit,
      props.filters.metric,
    ]
  )

  return (
    <div className={props.compact ? 'contents' : 'space-y-3'}>
      <div
        className={`flex items-center gap-1.5 overflow-x-auto pb-1 sm:gap-2 ${
          props.compact ? 'col-span-full' : ''
        }`}
      >
        <Tabs
          value={String(topUserLimit)}
          onValueChange={(value) => setTopUserLimit(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            <span className='text-muted-foreground px-2 text-xs font-medium whitespace-nowrap'>
              {t('Top Users')}
            </span>
            {TOP_USER_LIMIT_OPTIONS.map((limit) => (
              <TabsTrigger
                key={limit}
                value={String(limit)}
                className='px-2.5 text-xs'
              >
                {t('Top {{count}}', { count: limit })}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        {chartLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      <div className={props.compact ? 'contents' : 'grid gap-3'}>
        {USER_CHARTS.map((chart) => {
          const spec = chartData[chart.specKey]

          return (
            <div
              key={chart.value}
              className='overflow-hidden rounded-lg border'
            >
              <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
                <IconBadge tone='info' size='sm'>
                  <Users />
                </IconBadge>
                <div className='text-sm font-semibold'>{t(chart.labelKey)}</div>
              </div>

              <div
                className={
                  props.compact
                    ? 'h-64 p-1.5 sm:p-2'
                    : 'h-[300px] p-1.5 sm:h-96 sm:p-2'
                }
              >
                {chartLoading ? (
                  <Skeleton className='h-full w-full' />
                ) : (
                  themeReady &&
                  spec && (
                    <VChart
                      key={`user-${chart.value}-${topUserLimit}-${resolvedTheme}`}
                      spec={{
                        ...spec,
                        theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                        background: 'transparent',
                      }}
                      option={VCHART_OPTION}
                    />
                  )
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
