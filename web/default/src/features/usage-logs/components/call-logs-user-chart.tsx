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
import { Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import dayjs from '@/lib/dayjs'

import type {
  LogAnalyticsGranularity,
  LogAnalyticsUserLimit,
  LogUserTrendPoint,
} from '../types'
import { CallLogsChart } from './call-logs-chart'

interface CallLogsUserChartProps {
  data: LogUserTrendPoint[]
  granularity: LogAnalyticsGranularity
  userKeyword: string
  userLimit: LogAnalyticsUserLimit
  loading: boolean
  onUserKeywordChange: (value: string) => void
  onUserLimitChange: (value: LogAnalyticsUserLimit) => void
}

export function CallLogsUserChart(props: CallLogsUserChartProps) {
  const { t } = useTranslation()
  const [draftKeyword, setDraftKeyword] = useState(props.userKeyword)
  const spec = useMemo(() => {
    if (props.data.length === 0) return null
    const values = props.data.map((point) => ({
      ...point,
      label: dayjs.unix(point.timestamp).format(
        props.granularity === 'day' ? 'MM-DD' : 'MM-DD HH:mm'
      ),
      user: point.remark || point.email || point.username || `#${point.user_id}`,
    }))
    return {
      type: 'line',
      data: [{ id: 'users', values }],
      xField: 'label',
      yField: 'tokens',
      seriesField: 'user',
      point: { visible: false },
      legends: { visible: true, orient: 'top' },
      axes: [
        { orient: 'bottom', label: { autoHide: true, autoRotate: true } },
        {
          orient: 'left',
          label: {
            formatMethod: (value: number) =>
              new Intl.NumberFormat(undefined, {
                notation: 'compact',
                maximumFractionDigits: 1,
              }).format(value),
          },
        },
      ],
      tooltip: { dimension: { visible: true } },
    }
  }, [props.data, props.granularity])

  const applyUserKeyword = () => props.onUserKeywordChange(draftKeyword.trim())

  return (
    <div className='min-w-0 p-3 sm:p-4'>
      <div className='mb-2 flex flex-wrap items-center gap-2'>
        <h3 className='mr-auto text-sm font-semibold'>{t('User Usage Trend')}</h3>
        <div className='flex min-w-0 flex-1 sm:max-w-64'>
          <Input
            value={draftKeyword}
            onChange={(event) => setDraftKeyword(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') applyUserKeyword()
            }}
            placeholder={t('Search users by remark, email, username, or ID...')}
            className='h-8 rounded-r-none text-xs'
          />
          <Button
            type='button'
            variant='outline'
            size='icon'
            className='size-8 rounded-l-none border-l-0'
            onClick={applyUserKeyword}
            aria-label={t('Search')}
          >
            <Search aria-hidden='true' />
          </Button>
        </div>
        <Tabs
          value={String(props.userLimit)}
          onValueChange={(value) =>
            props.onUserLimitChange(Number(value) as LogAnalyticsUserLimit)
          }
        >
          <TabsList>
            <TabsTrigger value='10'>Top 10</TabsTrigger>
            <TabsTrigger value='20'>Top 20</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
      <CallLogsChart
        spec={spec}
        loading={props.loading}
        emptyText={t('No analytics data available')}
      />
    </div>
  )
}
