/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { Eye, EyeOff } from 'lucide-react'
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { DimensionUsageChart } from '@/features/admin-console/components/dimension-usage-chart'
import { DashboardChartControls } from '@/features/dashboard/components/dashboard-chart-controls'
import { FlowCharts } from '@/features/dashboard/components/flow/flow-charts'
import { ConsumptionDistributionChart } from '@/features/dashboard/components/models/consumption-distribution-chart'
import { LogStatCards } from '@/features/dashboard/components/models/log-stat-cards'
import { ModelCharts } from '@/features/dashboard/components/models/model-charts'
import { UserCharts } from '@/features/dashboard/components/users/user-charts'
import { DEFAULT_TIME_GRANULARITY } from '@/features/dashboard/constants'
import { buildDefaultDashboardFilters } from '@/features/dashboard/lib'
import type {
  DashboardFilters,
  QuotaDataItem,
} from '@/features/dashboard/types'

export type AdminAnalyticsSection = 'overview' | 'flow'

interface AdminAnalyticsProps {
  section: AdminAnalyticsSection
}

export function AdminAnalytics(props: AdminAnalyticsProps) {
  const { t } = useTranslation()
  const [modelData, setModelData] = useState<QuotaDataItem[]>([])
  const [dataLoading, setDataLoading] = useState(false)
  const [modelFilters, setModelFilters] = useState<DashboardFilters>(() =>
    buildDefaultDashboardFilters()
  )
  const [flowSensitiveVisible, setFlowSensitiveVisible] = useState(true)

  const handleDataUpdate = useCallback(
    (data: QuotaDataItem[], loading: boolean) => {
      setModelData(data)
      setDataLoading(loading)
    },
    []
  )

  const modelActions = (
    <DashboardChartControls filters={modelFilters} onChange={setModelFilters} />
  )

  const flowActions = (
    <>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon'
              onClick={() => setFlowSensitiveVisible((prev) => !prev)}
              aria-label={
                flowSensitiveVisible
                  ? t('Hide sensitive data')
                  : t('Show sensitive data')
              }
              className='text-muted-foreground hover:text-foreground size-8'
            />
          }
        >
          {flowSensitiveVisible ? <Eye /> : <EyeOff />}
        </TooltipTrigger>
        <TooltipContent>
          {flowSensitiveVisible
            ? t('Hide sensitive data')
            : t('Show sensitive data')}
        </TooltipContent>
      </Tooltip>
      <DashboardChartControls
        filters={modelFilters}
        onChange={setModelFilters}
      />
    </>
  )

  return (
    <div className='space-y-3 sm:space-y-4'>
      {props.section === 'overview' && (
        <LogStatCards
          filters={modelFilters}
          onDataUpdate={handleDataUpdate}
          includeAdminData
        />
      )}

      <div className='flex flex-wrap items-center justify-end gap-2'>
        {props.section === 'overview' && (
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            {modelActions}
          </div>
        )}
        {props.section === 'flow' && (
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            {flowActions}
          </div>
        )}
      </div>

      {props.section === 'overview' && (
        <div className='grid gap-3 md:grid-cols-2'>
          <ConsumptionDistributionChart
            data={modelData}
            loading={dataLoading}
            timeGranularity={
              modelFilters.time_granularity || DEFAULT_TIME_GRANULARITY
            }
            metric={modelFilters.metric}
            compact
          />
          <ModelCharts
            data={modelData}
            loading={dataLoading}
            timeGranularity={
              modelFilters.time_granularity || DEFAULT_TIME_GRANULARITY
            }
            compact
          />
          <DimensionUsageChart
            title='分组用量'
            dimension='group'
            data={modelData}
            loading={dataLoading}
            timeGranularity={
              modelFilters.time_granularity || DEFAULT_TIME_GRANULARITY
            }
            metric={modelFilters.metric}
          />
          <DimensionUsageChart
            title='渠道用量'
            dimension='channel'
            data={modelData}
            loading={dataLoading}
            timeGranularity={
              modelFilters.time_granularity || DEFAULT_TIME_GRANULARITY
            }
            metric={modelFilters.metric}
          />
          <UserCharts filters={modelFilters} dataOverride={modelData} compact />
        </div>
      )}

      {props.section === 'flow' && (
        <FlowCharts
          filters={modelFilters}
          sensitiveVisible={flowSensitiveVisible}
          includeAdminData
        />
      )}
    </div>
  )
}
