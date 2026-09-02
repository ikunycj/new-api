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
import { getPricingGroups } from '@/features/channels/api'
import { processChartData } from '@/features/dashboard/lib'
import type { DashboardMetric, QuotaDataItem } from '@/features/dashboard/types'
import { useThemeRadiusPx } from '@/lib/theme-radius'
import { formatChartTime, type TimeGranularity } from '@/lib/time'
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
  const timezoneOffset = -new Date().getTimezoneOffset()
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

  const pricingGroupNames = useMemo(
    () =>
      pricingGroupsQuery.data?.success
        ? (pricingGroupsQuery.data.data ?? [])
        : [],
    [pricingGroupsQuery.data]
  )
  const dimensionData = useMemo(() => {
    if (props.dimension === 'group') {
      const pricingGroupSet = new Set(pricingGroupNames)
      return props.data.flatMap((item) => {
        const pricingGroup = item.use_group?.trim()
        if (!pricingGroup || !pricingGroupSet.has(pricingGroup)) return []
        return [{ ...item, model_name: pricingGroup }]
      })
    }

    return props.data.map((item) => ({
      ...item,
      model_name:
        item.channel_name?.trim() ||
        (item.channel_id ? `渠道 #${item.channel_id}` : '未记录渠道'),
    }))
  }, [pricingGroupNames, props.data, props.dimension])
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
    return points.filter((point) => pricingGroupSet.has(point.name.trim()))
  }, [
    cacheTrendQuery.data,
    pricingGroupNames,
    pricingGroupsQuery.data?.success,
    props.dimension,
  ])
  const cacheTrendHasData = cacheTrendPoints.some(
    (point) => point.cache_input_tokens > 0
  )
  const spec = useMemo(() => {
    const cacheValues = cacheTrendPoints
      .filter(
        (point) =>
          point.cache_input_tokens > 0 && Number.isFinite(point.cache_hit_rate)
      )
      .map((point) => ({
        Time: formatChartTime(point.timestamp, props.timeGranularity),
        CacheRate: point.cache_hit_rate,
        Model: point.name.trim(),
      }))
    if (cacheValues.length === 0) return chartData.spec_area

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
      data: [areaData, { id: 'cacheTrendData', values: cacheValues }],
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
                  key: t('Cache hit rate'),
                  value: (datum: Record<string, unknown>) => {
                    const rate = Number(datum?.CacheRate)
                    return Number.isFinite(rate) ? `${rate.toFixed(2)}%` : '-'
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
  }, [cacheTrendPoints, chartData.spec_area, props.timeGranularity, t])
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
      <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
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
        {cacheTrendHasData && (
          <span className='text-muted-foreground text-xs'>· 缓存命中率</span>
        )}
      </div>

      <div className='h-64 p-1.5 sm:p-2'>{chartContent}</div>
    </div>
  )
}
