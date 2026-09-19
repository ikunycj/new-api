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

/** The three series the trend can plot. Their magnitudes differ too much to
 *  share one y-axis (a percentage and a token count cannot coexist), so the
 *  chart shows one at a time behind a toggle. */
const TREND_METRICS = [
  { key: 'requests', label: 'RPM' },
  { key: 'tokens', label: 'TPM' },
  { key: 'cache', label: 'Cache' },
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
  const isCache = metric === 'cache'
  let metricLabel = 'TPM'
  if (isRequests) metricLabel = 'RPM'
  else if (isCache) metricLabel = t('Cache Hit Rate')

  const spec = useMemo(() => {
    if (props.series.length === 0) return null

    const { textColor, gridColor } = getChartThemeTokens(resolvedTheme)
    const values = props.series.map((bucket) => {
      let value: number | null
      if (isCache) {
        // A minute with no cache-reporting request has no rate to plot. null
        // leaves a gap in the line, which is honest: interpolating across it
        // or substituting 0 would both invent a measurement that never
        // happened.
        value =
          bucket.input_tokens_total > 0
            ? (bucket.cache_read_tokens / bucket.input_tokens_total) * 100
            : null
      } else if (isRequests) {
        value = bucket.requests
      } else {
        value = bucket.tokens
      }

      return { time: dayjs(bucket.timestamp * 1000).format('HH:mm'), value }
    })

    let colorIndex = 5
    if (isRequests) colorIndex = 0
    else if (isCache) colorIndex = 2

    return {
      type: 'line' as const,
      data: [{ id: 'realtimeTrend', values }],
      xField: 'time',
      yField: 'value',
      color: [getChartColor(colorIndex)],
      line: {
        style: { lineWidth: 2, curveType: 'monotone' },
      },
      // A gap-heavy cache series can leave isolated points with no neighbours
      // to connect to, which would render as an invisible line.
      point: { visible: isCache },
      area: { visible: false },
      legends: { visible: false },
      padding: { top: 8, right: 12, bottom: 4, left: 4 },
      tooltip: {
        mark: {
          title: { value: (datum: { time: string }) => datum.time },
          content: [
            {
              key: metricLabel,
              value: (datum: { value: number | null }) => {
                if (datum.value == null) return '--'
                return isCache
                  ? `${formatNumber(Math.round(datum.value * 10) / 10, props.locale)}%`
                  : formatNumber(datum.value, props.locale)
              },
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
          // Pin the percentage axis to 0-100 so the cache line is read against
          // a fixed scale; autoscaling would make a 91%-to-93% wobble look
          // like a collapse.
          ...(isCache ? { min: 0, max: 100 } : {}),
          label: {
            formatMethod: (value: number) =>
              isCache ? `${value}%` : formatCompactNumber(value, props.locale),
            style: { fill: textColor, fontSize: 10 },
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: gridColor },
          },
        },
      ],
    }
  }, [
    isCache,
    isRequests,
    metricLabel,
    props.locale,
    props.series,
    resolvedTheme,
  ])

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
              {/* RPM and TPM are acronyms and stay as-is; "Cache" is a word
                  and gets translated. */}
              {option.key === 'cache' ? t(option.label) : option.label}
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
