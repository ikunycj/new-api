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
import { VChart } from '@visactor/react-vchart'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { getChartColor } from '@/lib/colors'
import dayjs from '@/lib/dayjs'
import { formatCompactNumber, formatNumber } from '@/lib/format'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import type { RealtimeBucket } from '../../types'

/** The two series the trend can plot. Their magnitudes differ too much to share
 *  one y-axis, so the chart shows one at a time behind a toggle. */
const TREND_METRICS = [
  { key: 'requests', label: 'RPM' },
  { key: 'tokens', label: 'TPM' },
] as const

type TrendMetricKey = (typeof TREND_METRICS)[number]['key']

function getChartThemeTokens(resolvedTheme: string) {
  return {
    textColor:
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.68)'
        : 'rgba(15, 23, 42, 0.58)',
    gridColor:
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.12)'
        : 'rgba(15, 23, 42, 0.12)',
  }
}

interface RealtimeTrendChartProps {
  /** Minute buckets, oldest first. Boundary buckets must already be trimmed. */
  series: RealtimeBucket[]
  loading?: boolean
  locale: Intl.LocalesArgument
}

/**
 * Per-minute RPM / TPM over the last hour.
 *
 * The backend folds its ten-second slots into one-minute buckets, so a bucket's
 * raw count is already a per-minute rate: requests-per-minute and
 * tokens-per-minute need no further division.
 */
export function RealtimeTrendChart(props: RealtimeTrendChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const [metric, setMetric] = useState<TrendMetricKey>('requests')

  const isRequests = metric === 'requests'
  const metricLabel = isRequests ? 'RPM' : 'TPM'

  const spec = useMemo(() => {
    if (props.series.length === 0) return null

    const { textColor, gridColor } = getChartThemeTokens(resolvedTheme)
    const values = props.series.map((bucket) => ({
      time: dayjs(bucket.timestamp * 1000).format('HH:mm'),
      value: isRequests ? bucket.requests : bucket.tokens,
    }))

    return {
      type: 'line' as const,
      data: [{ id: 'realtimeTrend', values }],
      xField: 'time',
      yField: 'value',
      color: [getChartColor(isRequests ? 0 : 5)],
      line: {
        style: { lineWidth: 2, curveType: 'monotone' },
      },
      point: { visible: false },
      area: { visible: false },
      legends: { visible: false },
      padding: { top: 8, right: 12, bottom: 4, left: 4 },
      tooltip: {
        mark: {
          title: { value: (datum: { time: string }) => datum.time },
          content: [
            {
              key: metricLabel,
              value: (datum: { value: number }) =>
                formatNumber(datum.value, props.locale),
            },
          ],
        },
      },
      axes: [
        {
          orient: 'bottom',
          tick: { visible: false },
          label: { style: { fill: textColor, fontSize: 10 } },
        },
        {
          orient: 'left',
          label: {
            formatMethod: (value: number) =>
              formatCompactNumber(value, props.locale),
            style: { fill: textColor, fontSize: 10 },
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: gridColor },
          },
        },
      ],
    }
  }, [isRequests, metricLabel, props.locale, props.series, resolvedTheme])

  if (props.loading) {
    return <div className='h-40 sm:h-48' />
  }

  if (props.series.length === 0) {
    return (
      <div className='text-muted-foreground flex h-40 items-center justify-center text-xs sm:h-48'>
        {t('No requests recorded in the last hour.')}
      </div>
    )
  }

  return (
    <div className='px-3 pt-2 pb-3 sm:px-5'>
      <div className='mb-2 flex items-center justify-between gap-2'>
        <span className='text-muted-foreground text-xs font-medium'>
          {t('Per-minute trend')}
        </span>
        <div className='bg-muted/60 inline-flex h-6 shrink-0 rounded-lg border p-0.5'>
          {TREND_METRICS.map((option) => (
            <button
              key={option.key}
              type='button'
              onClick={() => setMetric(option.key)}
              aria-pressed={metric === option.key}
              className={`rounded-md px-2.5 text-[11px] font-medium transition-colors ${
                metric === option.key
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className='h-40 sm:h-48'>
        {themeReady &&
          spec && (
            // Keyed on the bucket count rather than the data itself: a poll that
            // returns a fresh array with the same shape updates the spec in place
            // instead of tearing the canvas down and rebuilding it every 5s.
            <VChart
              key={`realtime-${metric}-${resolvedTheme}-${props.series.length}`}
              spec={{
                ...spec,
                theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                background: 'transparent',
              }}
              option={VCHART_OPTION}
            />
          )}
      </div>
    </div>
  )
}
