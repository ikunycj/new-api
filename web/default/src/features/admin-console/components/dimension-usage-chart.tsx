import { useQuery } from '@tanstack/react-query'
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
import { getPricingGroups } from '@/features/channels/api'
import { processChartData } from '@/features/dashboard/lib'
import type { QuotaDataItem } from '@/features/dashboard/types'
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
      ),
    [
      chartRadius,
      dimensionData,
      dimensionLoading,
      props.timeGranularity,
      t,
    ]
  )
  const Icon = props.dimension === 'group' ? Layers3 : RadioTower
  const spec = chartData.spec_area
  let chartContent = themeReady ? (
    <VChart
      key={`${props.dimension}-${props.timeGranularity}-${resolvedTheme}`}
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
      </div>

      <div className='h-64 p-1.5 sm:p-2'>{chartContent}</div>
    </div>
  )
}
