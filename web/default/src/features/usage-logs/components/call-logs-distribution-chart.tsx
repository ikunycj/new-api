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
import { useMemo, useState } from 'react'

import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  formatBillingCurrencyFromUSD,
  getCurrencyDisplay,
} from '@/lib/currency'

import type { LogDistributionItem } from '../types'
import { CallLogsChart } from './call-logs-chart'

type DistributionMode = 'tokens' | 'quota'

interface CallLogsDistributionChartProps {
  title: string
  data: LogDistributionItem[]
  loading: boolean
}

export function CallLogsDistributionChart(
  props: CallLogsDistributionChartProps
) {
  const [mode, setMode] = useState<DistributionMode>('tokens')
  const { config } = getCurrencyDisplay()
  let billingScale = 1
  let billingUnit = 'USD'
  if (config.quotaDisplayType === 'CNY') {
    billingScale = config.usdExchangeRate
    billingUnit = 'CNY'
  } else if (config.quotaDisplayType === 'CUSTOM') {
    billingScale = config.customCurrencyExchangeRate
    billingUnit = config.customCurrencySymbol
  }

  const chartData = useMemo(
    () =>
      props.data.map((row) => ({
        name: row.name === 'Unknown' ? '未知' : row.name,
        value:
          mode === 'tokens'
            ? row.tokens
            : (row.quota / config.quotaPerUnit) * billingScale,
      })),
    [billingScale, config.quotaPerUnit, mode, props.data]
  )
  const spec = useMemo(() => {
    if (
      chartData.every((row) => !Number.isFinite(row.value) || row.value <= 0)
    ) {
      return null
    }
    return {
      type: 'pie',
      data: [{ id: 'distribution', values: chartData }],
      categoryField: 'name',
      valueField: 'value',
      outerRadius: 0.84,
      innerRadius: 0.58,
      legends: { visible: false },
      label: { visible: false },
      tooltip: { mark: { visible: true } },
    }
  }, [chartData])
  const topRows = props.data.slice(0, 8)

  return (
    <Card size='sm' className='min-w-0'>
      <CardHeader>
        <CardTitle>{props.title}</CardTitle>
        <CardAction>
          <Tabs
            value={mode}
            onValueChange={(value) => setMode(value as DistributionMode)}
          >
            <TabsList>
              <TabsTrigger value='tokens'>按 Token</TabsTrigger>
              <TabsTrigger value='quota'>按费用</TabsTrigger>
            </TabsList>
          </Tabs>
        </CardAction>
      </CardHeader>
      <CardContent className='grid min-h-64 items-center gap-3 sm:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]'>
        <CallLogsChart
          spec={spec}
          loading={props.loading}
          emptyText='暂无分析数据'
          ariaLabel={`${props.title}${mode === 'quota' ? `，单位 ${billingUnit}` : ''}`}
          className='h-56'
        />
        <div className='divide-border min-w-0 divide-y'>
          {topRows.map((row, index) => (
            <div
              key={row.name}
              className='flex items-center gap-2 py-1.5 text-xs'
            >
              <span className='text-muted-foreground w-5 shrink-0 text-right tabular-nums'>
                {index + 1}
              </span>
              <span className='min-w-0 flex-1 truncate' title={row.name}>
                {row.name === 'Unknown' ? '未知' : row.name}
              </span>
              <span className='shrink-0 font-mono tabular-nums'>
                {mode === 'tokens'
                  ? formatCompact(row.tokens)
                  : formatBillingCurrencyFromUSD(
                      row.quota / config.quotaPerUnit,
                      { compact: true }
                    )}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function formatCompact(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(value)
}
