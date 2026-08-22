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
import { Activity, Coins, Gauge, Rows3 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import {
  formatBillingCurrencyFromUSD,
  getCurrencyDisplay,
  getCurrencyLabel,
} from '@/lib/currency'

import type { LogAnalyticsSummary } from '../types'

function formatCompactNumber(value: number): string {
  const absolute = Math.abs(value)
  if (absolute >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

function formatDuration(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0s'
  if (value < 1) return `${Math.round(value * 1000)}ms`
  return `${value.toFixed(value >= 10 ? 0 : 1)}s`
}

interface CallLogsSummaryProps {
  summary?: LogAnalyticsSummary
  loading: boolean
}

export function CallLogsSummary(props: CallLogsSummaryProps) {
  const { t } = useTranslation()
  const summary = props.summary
  const { config } = getCurrencyDisplay()
  const items = [
    {
      label: t('Total Requests'),
      value: summary?.total_requests.toLocaleString() ?? '0',
      detail: `${t('Successful Requests')}: ${(summary?.success_requests ?? 0).toLocaleString()} / ${t('Failed Requests')}: ${(summary?.failed_requests ?? 0).toLocaleString()}`,
      icon: Rows3,
      tone: 'text-sky-600 dark:text-sky-400',
    },
    {
      label: t('Total Tokens'),
      value: formatCompactNumber(summary?.total_tokens ?? 0),
      detail: `${t('Input')}: ${formatCompactNumber(summary?.input_tokens ?? 0)} / ${t('Output')}: ${formatCompactNumber(summary?.output_tokens ?? 0)} / ${t('Cache Read')}: ${formatCompactNumber(summary?.cache_read_tokens ?? 0)} / ${t('Cache Creation')}: ${formatCompactNumber(summary?.cache_write_tokens ?? 0)}`,
      icon: Activity,
      tone: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: t('Total Spend'),
      value: formatBillingCurrencyFromUSD(
        (summary?.total_quota ?? 0) / config.quotaPerUnit
      ),
      detail: `${t('Cost')} ${formatBillingCurrencyFromUSD(
        (summary?.total_quota ?? 0) / config.quotaPerUnit,
        { showSymbol: false, compact: true }
      )} ${getCurrencyLabel()}`,
      icon: Coins,
      tone: 'text-amber-600 dark:text-amber-400',
    },
    {
      label: t('Average Duration'),
      value: formatDuration(summary?.average_use_time ?? 0),
      detail: `P90: ${formatDuration(summary?.p90_use_time ?? 0)} / P99: ${formatDuration(summary?.p99_use_time ?? 0)}`,
      icon: Gauge,
      tone: 'text-rose-600 dark:text-rose-400',
    },
  ]

  return (
    <section className='bg-card/50 rounded-lg border p-2.5 sm:p-3'>
      <div className='grid divide-y sm:grid-cols-2 sm:divide-x sm:divide-y-0 xl:grid-cols-4'>
        {items.map((item) => {
          const Icon = item.icon
          return (
            <div key={item.label} className='min-w-0 px-3 py-3 first:pl-1 xl:py-2'>
              <div className='flex items-center justify-between gap-2'>
                <span className='text-muted-foreground text-xs font-medium'>
                  {item.label}
                </span>
                <Icon className={`size-4 ${item.tone}`} aria-hidden='true' />
              </div>
              {props.loading ? (
                <Skeleton className='mt-2 h-7 w-24' />
              ) : (
                <div className='mt-1 font-mono text-xl font-semibold tabular-nums'>
                  {item.value}
                </div>
              )}
              <p className='text-muted-foreground mt-1 truncate text-[11px]' title={item.detail}>
                {item.detail}
              </p>
            </div>
          )
        })}
      </div>
    </section>
  )
}
