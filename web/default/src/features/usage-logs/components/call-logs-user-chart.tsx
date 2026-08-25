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
import { SearchIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMemo, useState } from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group'
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
  const [draftKeyword, setDraftKeyword] = useState(props.userKeyword)
  const spec = useMemo(() => {
    if (props.data.length === 0) return null
    const values = props.data.map((point) => ({
      ...point,
      label: dayjs
        .unix(point.timestamp)
        .format(props.granularity === 'day' ? 'MM-DD' : 'MM-DD HH:mm'),
      user:
        point.remark || point.email || point.username || `#${point.user_id}`,
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
          label: { formatMethod: (value: number) => formatCompact(value) },
        },
      ],
      tooltip: { dimension: { visible: true } },
    }
  }, [props.data, props.granularity])

  const applyUserKeyword = () => {
    props.onUserKeywordChange(draftKeyword.trim())
  }

  return (
    <Card size='sm' className='min-w-0'>
      <CardHeader className='flex flex-col gap-2 sm:flex-row sm:items-center'>
        <CardTitle className='mr-auto'>用户 Token 趋势</CardTitle>
        <div className='flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto'>
          <InputGroup className='min-w-44 flex-1 sm:w-64 sm:flex-none'>
            <InputGroupInput
              value={draftKeyword}
              onChange={(event) => setDraftKeyword(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applyUserKeyword()
              }}
              placeholder='按备注、邮箱、用户名或 ID 搜索'
              aria-label='搜索用户'
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupButton
                size='icon-xs'
                onClick={applyUserKeyword}
                aria-label='搜索'
              >
                <HugeiconsIcon icon={SearchIcon} aria-hidden='true' />
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
          <Tabs
            value={String(props.userLimit)}
            onValueChange={(value) =>
              props.onUserLimitChange(Number(value) as LogAnalyticsUserLimit)
            }
          >
            <TabsList>
              <TabsTrigger value='10'>前 10</TabsTrigger>
              <TabsTrigger value='20'>前 20</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </CardHeader>
      <CardContent className='px-2 pb-1'>
        <CallLogsChart
          spec={spec}
          loading={props.loading}
          emptyText='暂无分析数据'
          ariaLabel='用户 Token 用量趋势图'
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
