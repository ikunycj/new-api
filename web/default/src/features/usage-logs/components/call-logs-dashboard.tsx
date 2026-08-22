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
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

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
  const { t } = useTranslation()
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
      user_keyword: userKeyword || undefined,
      user_limit: userLimit,
    }
  }, [granularity, searchParams, userKeyword, userLimit])

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['call-logs-analytics', analyticsParams],
    queryFn: () => getLogAnalytics(analyticsParams),
    staleTime: 30_000,
    placeholderData: (previous) => previous,
  })
  const analytics = data?.data
  const loading = isLoading || (isFetching && !analytics)

  return (
    <>
      <CallLogsSummary summary={analytics?.summary} loading={loading} />
      <section className='bg-card/50 rounded-lg border p-2.5 sm:p-3'>
        <div className='mb-2 flex items-center gap-2'>
          <Tabs
            value={granularity}
            onValueChange={(value) =>
              setGranularity(value as LogAnalyticsGranularity)
            }
          >
            <TabsList>
              <TabsTrigger value='hour'>{t('Hourly')}</TabsTrigger>
              <TabsTrigger value='day'>{t('Daily')}</TabsTrigger>
            </TabsList>
          </Tabs>
          {isFetching && analytics && (
            <span className='text-muted-foreground text-xs'>{t('Updating...')}</span>
          )}
        </div>
        <div className='grid overflow-hidden rounded-md border lg:grid-cols-2'>
          <div className='border-b lg:border-r'>
            <CallLogsTokenChart
              data={analytics?.token_trend ?? []}
              granularity={granularity}
              loading={loading}
            />
          </div>
          <div className='border-b'>
            <CallLogsUserChart
              data={analytics?.user_trend ?? []}
              granularity={granularity}
              userKeyword={userKeyword}
              userLimit={userLimit}
              loading={loading}
              onUserKeywordChange={setUserKeyword}
              onUserLimitChange={setUserLimit}
            />
          </div>
          <div className='border-b lg:border-r lg:border-b-0'>
            <CallLogsDistributionChart
              title={t('Group Usage Distribution')}
              data={analytics?.group_distribution ?? []}
              loading={loading}
            />
          </div>
          <CallLogsDistributionChart
            title={t('Model Distribution')}
            data={analytics?.model_distribution ?? []}
            loading={loading}
          />
        </div>
      </section>
    </>
  )
}
