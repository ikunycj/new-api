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
*/
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Layers3, RadioTower } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import { getAdminConsoleCacheTrend } from '@/features/admin-console/api'
import {
  buildCacheTrendChartValues,
  summarizeCacheTrend,
} from '@/features/admin-console/cache-trend'
import { formatChannelDisplayName } from '@/features/admin-console/channel-display'
import { getPricingGroups } from '@/features/channels/api'
import { processChartData } from '@/features/dashboard/lib'
import type { DashboardMetric, QuotaDataItem } from '@/features/dashboard/types'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { useThemeRadiusPx } from '@/lib/theme-radius'
import type { TimeGranularity } from '@/lib/time'
import { VCHART_OPTION } from '@/lib/vchart'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

interface DimensionUsageChartProps {
  title: string
  dimension: 'group' | 'channel'
  data: QuotaDataItem[]
  loading?: boolean
  timeGranularity: TimeGranularity
  metric?: DashboardMetric
  startTimestamp?: number
  endTimestamp?: number
}

export function DimensionUsageChart(props: DimensionUsageChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { customization } = useThemeCustomization()
  const chartRadius = useThemeRadiusPx(
    '--radius-md',
    `${customization.preset}:${customization.radius}`
  )
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)
  const pricingGroupsQuery = useQuery({
    queryKey: ['pricing-groups'],
    queryFn: getPricingGroups,
    enabled: props.dimension === 'group',
  })
  const cacheTrendEnabled =
    !props.loading &&
    props.startTimestamp !== undefined &&
    props.endTimestamp !== undefined
  const timezoneOffset = new Date().getTimezoneOffset()
  const cacheTrendQuery = useQuery({
    queryKey: [
      'admin-console-cache-trend',
      props.dimension,
      props.startTimestamp,
      props.endTimestamp,
      props.timeGranularity,
      timezoneOffset,
    ],
    queryFn: () => {
      if (
        props.startTimestamp === undefined ||
        props.endTimestamp === undefined
      ) {
        return Promise.resolve([])
      }
      return getAdminConsoleCacheTrend({
        dimension: props.dimension,
        start_timestamp: props.startTimestamp,
        end_timestamp: props.endTimestamp,
        granularity: props.timeGranularity,
        timezone_offset: timezoneOffset,
      })
    },
    enabled: cacheTrendEnabled,
    staleTime: 60_000,
  })

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (module) => module.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }

    void updateTheme()
  }, [resolvedTheme])

  const pricingGroupLabels = useMemo(() => {
    if (!pricingGroupsQuery.data?.success) return new Map<string, string>()
    return new Map(
      (pricingGroupsQuery.data.data ?? []).map((group) => [
        group,
        pricingGroupsQuery.data?.display_names?.[group] || group,
      ])
    )
  }, [pricingGroupsQuery.data])
  const pricingGroupNames = useMemo(
    () => [...pricingGroupLabels.keys()],
    [pricingGroupLabels]
  )
  const dimensionData = useMemo(() => {
    if (props.dimension === 'group') {
      const pricingGroupSet = new Set(pricingGroupNames)
      return props.data.flatMap((item) => {
        const pricingGroup = item.use_group?.trim()
        if (!pricingGroup || !pricingGroupSet.has(pricingGroup)) return []
        return [
          {
            ...item,
            model_name: pricingGroupLabels.get(pricingGroup) || pricingGroup,
          },
        ]
      })
    }

    return props.data.map((item) => ({
      ...item,
      model_name: formatChannelDisplayName(item.channel_name, item.channel_id),
    }))
  }, [pricingGroupLabels, pricingGroupNames, props.data, props.dimension])
  const pricingGroupsFailed =
    props.dimension === 'group' &&
    (pricingGroupsQuery.isError || pricingGroupsQuery.data?.success === false)
  const dimensionLoading =
    props.loading ||
    (props.dimension === 'group' && pricingGroupsQuery.isLoading)

  const chartData = useMemo(
    () =>
      processChartData(
        dimensionLoading ? [] : dimensionData,
        props.timeGranularity,
        t,
        chartRadius,
        props.metric
      ),
    [
      chartRadius,
      dimensionData,
      dimensionLoading,
      props.metric,
      props.timeGranularity,
      t,
    ]
  )
  const Icon = props.dimension === 'group' ? Layers3 : RadioTower
  const cacheTrendPoints = useMemo(() => {
    const points = cacheTrendQuery.data ?? []
    if (props.dimension !== 'group') return points
    if (!pricingGroupsQuery.data?.success) return []
    const pricingGroupSet = new Set(pricingGroupNames)
    return points.flatMap((point) => {
      const groupName = point.name.trim()
      if (!pricingGroupSet.has(groupName)) return []
      return [
        {
          ...point,
          name: pricingGroupLabels.get(groupName) || groupName,
        },
      ]
    })
  }, [
    cacheTrendQuery.data,
    pricingGroupLabels,
    pricingGroupNames,
    pricingGroupsQuery.data?.success,
    props.dimension,
  ])
  const cacheTrendValues = useMemo(() => {
    if (!cacheTrendQuery.isSuccess || dimensionData.length === 0) return []
    const areaValues = (chartData.spec_area.data?.[0]?.values ?? []) as {
      Time?: unknown
      Model?: unknown
    }[]
    return buildCacheTrendChartValues(
      dimensionData,
      areaValues,
      cacheTrendPoints,
      props.timeGranularity
    )
  }, [
    cacheTrendPoints,
    cacheTrendQuery.isSuccess,
    chartData.spec_area.data,
    dimensionData,
    props.timeGranularity,
  ])
  const cacheTrendHasSeries = cacheTrendValues.length > 0
  const cacheTrendSummary = useMemo(
    () => summarizeCacheTrend(cacheTrendValues),
    [cacheTrendValues]
  )
  const spec = useMemo(() => {
    if (cacheTrendValues.length === 0) return chartData.spec_area

    const areaData = chartData.spec_area.data?.[0] ?? {
      id: 'areaData',
      values: [],
    }
    const usageSeriesId = 'dimension-usage-series'
    const cacheSeriesId = 'cache-rate-series'
    const areaAxes =
      Array.isArray(chartData.spec_area.axes) &&
      chartData.spec_area.axes.length > 0
        ? chartData.spec_area.axes
        : [
            { orient: 'bottom', type: 'band' },
            {
              orient: 'left',
              type: 'linear',
              seriesId: usageSeriesId,
            },
          ]
    return {
      ...chartData.spec_area,
      type: 'common',
      data: [areaData, { id: 'cacheTrendData', values: cacheTrendValues }],
      series: [
        {
          id: usageSeriesId,
          type: 'area',
          dataId: areaData.id,
          xField: 'Time',
          yField: 'Usage',
          seriesField: 'Model',
          stack: false,
          area: chartData.spec_area.area,
          line: chartData.spec_area.line,
          point: chartData.spec_area.point,
        },
        {
          id: cacheSeriesId,
          type: 'line',
          dataId: 'cacheTrendData',
          xField: 'Time',
          yField: 'CacheRate',
          seriesField: 'Model',
          zIndex: 10,
          line: {
            style: {
              lineWidth: 2,
              lineDash: [4, 3],
              curveType: 'monotone',
            },
          },
          point: { visible: false },
          tooltip: {
            mark: {
              content: [
                {
                  key: 'Token 命中率',
                  value: (datum: Record<string, unknown>) => {
                    const inputTokens = Number(datum?.CacheInputTokens)
                    const rate = Number(datum?.CacheRate)
                    return inputTokens > 0 && Number.isFinite(rate)
                      ? `${rate.toFixed(2)}%`
                      : '-'
                  },
                },
                {
                  key: '缓存命中 Token',
                  value: (datum: Record<string, unknown>) =>
                    formatNumber(Number(datum?.CacheReadTokens) || 0),
                },
                {
                  key: '缓存写入 Token',
                  value: (datum: Record<string, unknown>) =>
                    formatNumber(Number(datum?.CacheWriteTokens) || 0),
                },
                {
                  key: '命中请求',
                  value: (datum: Record<string, unknown>) => {
                    const hitRequests = Number(datum?.CacheHitRequests) || 0
                    const eligibleRequests =
                      Number(datum?.CacheEligibleRequests) || 0
                    const rate =
                      eligibleRequests > 0
                        ? ` (${((hitRequests / eligibleRequests) * 100).toFixed(1)}%)`
                        : ''
                    return `${formatNumber(hitRequests)} / ${formatNumber(eligibleRequests)}${rate}`
                  },
                },
              ],
            },
          },
        },
      ],
      axes: [
        ...areaAxes,
        {
          id: 'cache-rate-axis',
          orient: 'right',
          type: 'linear',
          seriesId: cacheSeriesId,
          min: 0,
          max: 100,
          visible: true,
          label: {
            formatMethod: (value: number | string) => `${value}%`,
          },
        },
      ],
      color: chartData.spec_area.color,
      legends: chartData.spec_area.legends,
      tooltip: chartData.spec_area.tooltip,
    }
  }, [cacheTrendValues, chartData.spec_area])
  let chartContent = themeReady ? (
    <VChart
      key={`${props.dimension}-${props.metric}-${props.timeGranularity}-${resolvedTheme}`}
      spec={{
        ...spec,
        title: { visible: false },
        theme: resolvedTheme === 'dark' ? 'dark' : 'light',
        background: 'transparent',
      }}
      option={VCHART_OPTION}
    />
  ) : null

  if (dimensionData.length === 0) {
    const emptyTitle =
      props.dimension === 'group' ? '暂无可归属的定价分组用量' : '暂无渠道用量'
    const emptyDescription =
      props.dimension === 'group'
        ? '仅统计定价分组管理中当前存在的分组'
        : '所选时间范围内没有渠道调用记录'
    chartContent = (
      <Empty className='h-full border-0 py-8'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Icon />
          </EmptyMedia>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          <EmptyDescription>{emptyDescription}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }
  if (pricingGroupsFailed) {
    chartContent = (
      <Empty className='h-full border-0 py-8'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Layers3 />
          </EmptyMedia>
          <EmptyTitle>定价分组目录加载失败</EmptyTitle>
          <EmptyDescription>
            无法读取定价分组管理中的分组，请稍后重试
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }
  if (dimensionLoading) {
    chartContent = <Skeleton className='h-full w-full' />
  }

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='space-y-1.5 border-b px-3 py-2 sm:px-5 sm:py-3'>
        <div className='flex w-full items-center gap-2'>
          <IconBadge
            tone={props.dimension === 'group' ? 'chart-2' : 'chart-3'}
            size='sm'
          >
            <Icon />
          </IconBadge>
          <div className='text-sm font-semibold'>{props.title}</div>
          <span className='text-muted-foreground text-xs'>
            合计 {chartData.totalCountDisplay}
          </span>
        </div>
        {cacheTrendHasSeries && (
          <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 pl-8 text-xs tabular-nums'>
            <span>
              缓存命中{' '}
              <strong className='text-foreground/85 font-medium'>
                {formatCompactNumber(cacheTrendSummary.cacheReadTokens)}
              </strong>{' '}
              Token
            </span>
            <span>
              Token 命中率{' '}
              <strong className='text-foreground/85 font-medium'>
                {cacheTrendSummary.cacheInputTokens > 0
                  ? `${cacheTrendSummary.tokenHitRate.toFixed(1)}%`
                  : '-'}
              </strong>
            </span>
            <span>
              请求命中{' '}
              <strong className='text-foreground/85 font-medium'>
                {formatCompactNumber(cacheTrendSummary.cacheHitRequests)} /{' '}
                {formatCompactNumber(cacheTrendSummary.cacheEligibleRequests)}
                {cacheTrendSummary.cacheEligibleRequests > 0
                  ? ` (${cacheTrendSummary.requestHitRate.toFixed(1)}%)`
                  : ''}
              </strong>
            </span>
            {cacheTrendSummary.cacheWriteTokens > 0 && (
              <span>
                缓存写入{' '}
                <strong className='text-foreground/85 font-medium'>
                  {formatCompactNumber(cacheTrendSummary.cacheWriteTokens)}
                </strong>{' '}
                Token
              </span>
            )}
          </div>
        )}
      </div>

      <div className='h-64 p-1.5 sm:p-2'>{chartContent}</div>
    </div>
  )
}
