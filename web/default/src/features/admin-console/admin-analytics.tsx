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
import { FlowCharts } from '@/features/dashboard/components/flow/flow-charts'
import { ConsumptionDistributionChart } from '@/features/dashboard/components/models/consumption-distribution-chart'
import { LogStatCards } from '@/features/dashboard/components/models/log-stat-cards'
import { ModelCharts } from '@/features/dashboard/components/models/model-charts'
import { ModelsChartPreferences } from '@/features/dashboard/components/models/models-chart-preferences'
import { ModelsFilter } from '@/features/dashboard/components/models/models-filter-dialog'
import { PerformanceOverview } from '@/features/dashboard/components/models/performance-overview'
import { UserCharts } from '@/features/dashboard/components/users/user-charts'
import { DEFAULT_TIME_GRANULARITY } from '@/features/dashboard/constants'
import {
  buildDefaultDashboardFilters,
  getDefaultDays,
  getSavedChartPreferences,
  getSavedGranularity,
  saveChartPreferences,
} from '@/features/dashboard/lib'
import type {
  DashboardChartPreferences,
  DashboardFilters,
  QuotaDataItem,
  UserChartsFilters,
} from '@/features/dashboard/types'

export type AdminAnalyticsSection = 'overview' | 'flow'

interface AdminAnalyticsProps {
  section: AdminAnalyticsSection
}

export function AdminAnalytics(props: AdminAnalyticsProps) {
  const { t } = useTranslation()
  const [modelData, setModelData] = useState<QuotaDataItem[]>([])
  const [dataLoading, setDataLoading] = useState(false)
  const [chartPreferences, setChartPreferences] =
    useState<DashboardChartPreferences>(() => getSavedChartPreferences())
  const [modelFilters, setModelFilters] = useState<DashboardFilters>(() =>
    buildDefaultDashboardFilters(getSavedChartPreferences())
  )
  const [userChartsFilters, setUserChartsFilters] = useState<UserChartsFilters>(
    () => {
      const granularity = getSavedGranularity()
      return {
        timeGranularity: granularity,
        selectedRange: getDefaultDays(granularity),
        topUserLimit: 10,
      }
    }
  )
  const [flowSensitiveVisible, setFlowSensitiveVisible] = useState(true)

  const handleChartPreferencesChange = useCallback(
    (preferences: DashboardChartPreferences) => {
      setChartPreferences(preferences)
      setModelFilters(buildDefaultDashboardFilters(preferences))
      saveChartPreferences(preferences)
    },
    []
  )

  const handleDataUpdate = useCallback(
    (data: QuotaDataItem[], loading: boolean) => {
      setModelData(data)
      setDataLoading(loading)
    },
    []
  )

  const modelActions = (
    <>
      <ModelsChartPreferences
        preferences={chartPreferences}
        onPreferencesChange={handleChartPreferencesChange}
      />
      <ModelsFilter
        preferences={chartPreferences}
        currentFilters={modelFilters}
        onFilterChange={setModelFilters}
        onReset={() =>
          setModelFilters(buildDefaultDashboardFilters(chartPreferences))
        }
        includeAdminData
      />
    </>
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
      <ModelsFilter
        preferences={chartPreferences}
        currentFilters={modelFilters}
        onFilterChange={setModelFilters}
        onReset={() =>
          setModelFilters(buildDefaultDashboardFilters(chartPreferences))
        }
        titleKey='Flow Filters'
        descriptionKey='Filter the traffic flow view by time range and user.'
        includeAdminData
      />
    </>
  )

  return (
    <div className='space-y-3 sm:space-y-4'>
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
        <>
          <LogStatCards
            filters={modelFilters}
            onDataUpdate={handleDataUpdate}
            includeAdminData
          />
          <PerformanceOverview />
          <ConsumptionDistributionChart
            data={modelData}
            loading={dataLoading}
            defaultChartType={chartPreferences.consumptionDistributionChart}
            timeGranularity={
              modelFilters.time_granularity || DEFAULT_TIME_GRANULARITY
            }
          />
          <ModelCharts
            data={modelData}
            loading={dataLoading}
            defaultChartTab={chartPreferences.modelAnalyticsChart}
            timeGranularity={
              modelFilters.time_granularity || DEFAULT_TIME_GRANULARITY
            }
          />
          <UserCharts
            filters={userChartsFilters}
            onFiltersChange={setUserChartsFilters}
          />
        </>
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
