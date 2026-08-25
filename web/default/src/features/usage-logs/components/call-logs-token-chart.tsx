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
import { Settings02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import dayjs from '@/lib/dayjs'

import type { LogAnalyticsGranularity, LogTokenTrendPoint } from '../types'
import { CallLogsChart } from './call-logs-chart'

type TokenSeries = 'input_tokens' | 'output_tokens' | 'cache_hit_rate'

const SERIES_OPTIONS: Array<{ value: TokenSeries; label: string }> = [
  { value: 'input_tokens', label: '输入 Token' },
  { value: 'output_tokens', label: '输出 Token' },
  { value: 'cache_hit_rate', label: '缓存命中率' },
]

interface CallLogsTokenChartProps {
  data: LogTokenTrendPoint[]
  granularity: LogAnalyticsGranularity
  loading: boolean
}

export function CallLogsTokenChart(props: CallLogsTokenChartProps) {
  const [visible, setVisible] = useState<TokenSeries[]>([
    'input_tokens',
    'output_tokens',
    'cache_hit_rate',
  ])

  const spec = useMemo(() => {
    if (props.data.length === 0 || visible.length === 0) return null

    const formatTimestamp = (timestamp: number) =>
      dayjs
        .unix(timestamp)
        .format(props.granularity === 'day' ? 'MM-DD' : 'MM-DD HH:mm')
    const tokenSeries = SERIES_OPTIONS.filter(
      (item) => item.value !== 'cache_hit_rate' && visible.includes(item.value)
    )
    const tokenValues = props.data.flatMap((point) =>
      tokenSeries.map((item) => ({
        label: formatTimestamp(point.timestamp),
        series: item.label,
        amount: point[item.value],
      }))
    )
    const rateValues = visible.includes('cache_hit_rate')
      ? props.data.map((point) => ({
          label: formatTimestamp(point.timestamp),
          series: '缓存命中率',
          amount: point.cache_hit_rate,
        }))
      : []
    const data: Array<Record<string, unknown>> = []
    const series: Array<Record<string, unknown>> = []
    const axes: Array<Record<string, unknown>> = [
      {
        orient: 'bottom',
        label: { autoHide: true, autoRotate: true },
      },
    ]

    if (tokenValues.length > 0) {
      data.push({ id: 'token-series', values: tokenValues })
      series.push({
        type: 'line',
        id: 'tokens',
        dataId: 'token-series',
        xField: 'label',
        yField: 'amount',
        seriesField: 'series',
        point: { visible: false },
        line: { style: { lineWidth: 2 } },
      })
      axes.push({
        orient: 'left',
        seriesId: 'tokens',
        label: { formatMethod: (value: number) => formatCompact(value) },
      })
    }
    if (rateValues.length > 0) {
      data.push({ id: 'rate-series', values: rateValues })
      series.push({
        type: 'line',
        id: 'cache-rate',
        dataId: 'rate-series',
        xField: 'label',
        yField: 'amount',
        seriesField: 'series',
        point: { visible: false },
        line: { style: { lineWidth: 2, lineDash: [4, 3] } },
      })
      axes.push({
        orient: 'right',
        seriesId: 'cache-rate',
        min: 0,
        max: 100,
        label: { formatMethod: (value: number) => `${value.toFixed(0)}%` },
      })
    }

    return {
      type: 'common',
      data,
      series,
      axes,
      legends: { visible: true, orient: 'top' },
      tooltip: { dimension: { visible: true } },
    }
  }, [props.data, props.granularity, visible])

  const toggleSeries = (series: TokenSeries, checked: boolean) => {
    setVisible((current) => {
      if (checked) {
        return current.includes(series) ? current : [...current, series]
      }
      return current.length === 1
        ? current
        : current.filter((item) => item !== series)
    })
  }

  return (
    <Card size='sm' className='min-w-0'>
      <CardHeader>
        <CardTitle>Token 用量趋势</CardTitle>
        <CardAction>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant='outline'
                  size='icon'
                  aria-label='选择显示指标'
                />
              }
            >
              <HugeiconsIcon icon={Settings02Icon} aria-hidden='true' />
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end'>
              <DropdownMenuGroup>
                {SERIES_OPTIONS.map((item) => (
                  <DropdownMenuCheckboxItem
                    key={item.value}
                    checked={visible.includes(item.value)}
                    onCheckedChange={(checked) =>
                      toggleSeries(item.value, checked)
                    }
                  >
                    {item.label}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </CardAction>
      </CardHeader>
      <CardContent className='px-2 pb-1'>
        <CallLogsChart
          spec={spec}
          loading={props.loading}
          emptyText='暂无分析数据'
          ariaLabel='Token 用量与缓存命中率趋势图'
        />
      </CardContent>
    </Card>
  )
}

function formatCompact(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}
