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
import { Alert02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useMemo, useState } from 'react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { getLogAnalytics } from '../api'
import { LOG_TYPE_ALL_VALUE } from '../constants'
import { buildApiParams } from '../lib/utils'
import type {
  GetLogAnalyticsParams,
  LogAnalyticsGranularity,
  LogAnalyticsUserLimit,
} from '../types'
import { CallLogsDistributionChart } from './call-logs-distribution-chart'
import { CallLogsSummary } from './call-logs-summary'
import { CallLogsTokenChart } from './call-logs-token-chart'
import { CallLogsUserChart } from './call-logs-user-chart'

const route = getRouteApi('/_authenticated/usage-logs/$section')

export default function CallLogsDashboard() {
  const searchParams = route.useSearch()
  const [granularity, setGranularity] =
    useState<LogAnalyticsGranularity>('hour')
  const [userKeyword, setUserKeyword] = useState('')
  const [userLimit, setUserLimit] = useState<LogAnalyticsUserLimit>(10)

  const analyticsParams = useMemo<GetLogAnalyticsParams>(() => {
    const params = buildApiParams({
      page: 1,
      pageSize: 1,
      searchParams,
      isAdmin: true,
    })
    return {
      type:
        params.type === Number(LOG_TYPE_ALL_VALUE) ? undefined : params.type,
      start_timestamp: params.start_timestamp,
      end_timestamp: params.end_timestamp,
      keyword: params.keyword,
      model_name: params.model_name,
      token_name: params.token_name,
      channel: params.channel,
      group: params.group,
      granularity,
      timezone_offset: new Date().getTimezoneOffset(),
      user_keyword: userKeyword || undefined,
      user_limit: userLimit,
    }
  }, [granularity, searchParams, userKeyword, userLimit])

  const analyticsQuery = useQuery({
    queryKey: ['call-logs-analytics', analyticsParams],
    queryFn: async () => {
      const response = await getLogAnalytics(analyticsParams)
      if (!response.success || !response.data) {
        throw new Error(response.message || '调用分析数据加载失败')
      }
      return response.data
    },
    staleTime: 30_000,
    placeholderData: (previous) => previous,
  })
  const analytics = analyticsQuery.data
  const loading =
    analyticsQuery.isLoading || (analyticsQuery.isFetching && !analytics)

  return (
    <section aria-label='调用分析' className='flex flex-col gap-3'>
      <CallLogsSummary summary={analytics?.summary} loading={loading} />
      <div className='flex flex-wrap items-center gap-2'>
        <Tabs
          value={granularity}
          onValueChange={(value) =>
            setGranularity(value as LogAnalyticsGranularity)
          }
        >
          <TabsList>
            <TabsTrigger value='hour'>按小时</TabsTrigger>
            <TabsTrigger value='day'>按天</TabsTrigger>
          </TabsList>
        </Tabs>
        {analyticsQuery.isFetching && analytics && (
          <span className='text-muted-foreground text-xs'>正在更新...</span>
        )}
      </div>
      {analyticsQuery.isError && (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={Alert02Icon} aria-hidden='true' />
          <AlertTitle>调用分析数据加载失败</AlertTitle>
          <AlertDescription>请调整筛选条件或稍后重试。</AlertDescription>
        </Alert>
      )}
      <div className='grid gap-3 lg:grid-cols-2'>
        <CallLogsTokenChart
          data={analytics?.token_trend ?? []}
          granularity={granularity}
          loading={loading}
        />
        <CallLogsUserChart
          data={analytics?.user_trend ?? []}
          granularity={granularity}
          userKeyword={userKeyword}
          userLimit={userLimit}
          loading={loading}
          onUserKeywordChange={setUserKeyword}
          onUserLimitChange={setUserLimit}
        />
        <CallLogsDistributionChart
          title='定价分组用量分布'
          data={analytics?.group_distribution ?? []}
          loading={loading}
        />
        <CallLogsDistributionChart
          title='模型用量分布'
          data={analytics?.model_distribution ?? []}
          loading={loading}
        />
      </div>
    </section>
  )
}
