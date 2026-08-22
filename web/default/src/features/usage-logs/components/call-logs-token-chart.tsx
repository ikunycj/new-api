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
import { Settings2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import dayjs from '@/lib/dayjs'

import type {
  LogAnalyticsGranularity,
  LogTokenTrendPoint,
} from '../types'
import { CallLogsChart } from './call-logs-chart'

type TokenSeries = 'input_tokens' | 'output_tokens' | 'cache_hit_rate'

interface CallLogsTokenChartProps {
  data: LogTokenTrendPoint[]
  granularity: LogAnalyticsGranularity
  loading: boolean
}

export function CallLogsTokenChart(props: CallLogsTokenChartProps) {
  const { t } = useTranslation()
  const [visible, setVisible] = useState<TokenSeries[]>([
    'input_tokens',
    'output_tokens',
    'cache_hit_rate',
  ])
  const seriesOptions = useMemo<Array<{ value: TokenSeries; label: string; color: string }>>(
    () => [
      { value: 'input_tokens', label: t('Input'), color: '#0ea5e9' },
      { value: 'output_tokens', label: t('Output'), color: '#f97316' },
      { value: 'cache_hit_rate', label: t('Cache Hit Rate'), color: '#10b981' },
    ],
    [t]
  )

  const spec = useMemo(() => {
    if (props.data.length === 0 || visible.length === 0) return null
    const values = props.data.flatMap((point) => {
      const label = dayjs.unix(point.timestamp).format(
        props.granularity === 'day' ? 'MM-DD' : 'MM-DD HH:mm'
      )
      return seriesOptions
        .filter((item) => visible.includes(item.value))
        .map((item) => ({
          label,
          series: item.label,
          value: item.value,
          amount: item.value === 'cache_hit_rate' ? point.cache_hit_rate : point[item.value],
        }))
    })
    return {
      type: 'line',
      data: [{ id: 'tokens', values }],
      xField: 'label',
      yField: 'amount',
      seriesField: 'series',
      point: { visible: false },
      line: { style: { lineWidth: 2 } },
      legends: {
        visible: true,
        orient: 'top',
      },
      axes: [
        { orient: 'bottom', label: { autoHide: true, autoRotate: true } },
        {
          orient: 'left',
          label: { formatMethod: (value: number) => formatCompact(value) },
        },
        {
          orient: 'right',
          visible: visible.includes('cache_hit_rate'),
          label: { formatMethod: (value: number) => `${value.toFixed(0)}%` },
        },
      ],
      tooltip: { dimension: { visible: true } },
    }
  }, [props.data, props.granularity, seriesOptions, visible])

  const toggleSeries = (series: TokenSeries, checked: boolean) => {
    setVisible((current) => {
      if (checked) return current.includes(series) ? current : [...current, series]
      if (current.length === 1) return current
      return current.filter((item) => item !== series)
    })
  }

  return (
    <div className='min-w-0 p-3 sm:p-4'>
      <div className='mb-2 flex items-center justify-between gap-2'>
        <h3 className='text-sm font-semibold'>{t('Token Usage Trend')}</h3>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant='outline' size='icon' className='size-8' aria-label={t('Select Series')} />
            }
          >
            <Settings2 aria-hidden='true' />
          </DropdownMenuTrigger>
          <DropdownMenuContent align='end'>
            {seriesOptions.map((item) => (
              <DropdownMenuCheckboxItem
                key={item.value}
                checked={visible.includes(item.value)}
                onCheckedChange={(checked) => toggleSeries(item.value, checked)}
              >
                <span className='size-2 rounded-full' style={{ backgroundColor: item.color }} />
                {item.label}
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <CallLogsChart
        spec={spec}
        loading={props.loading}
        emptyText={t('No analytics data available')}
      />
    </div>
  )
}

function formatCompact(value: number): string {
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}
