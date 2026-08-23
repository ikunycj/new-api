/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import { BarChart3, RefreshCw } from 'lucide-react'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getAdminPlans,
  getPackageComparison,
} from '@/features/dashboard/api'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatQuota } from '@/lib/format'

const TIME_RANGES = [
  { value: 1, labelKey: '24 hours' },
  { value: 7, labelKey: '7 days' },
  { value: 30, labelKey: '30 days' },
] as const

export function PackageComparison() {
  const { t, i18n } = useTranslation()
  const [selectedPlans, setSelectedPlans] = useState<string[]>([])
  const [selectedDays, setSelectedDays] = useState(1)
  const plansQuery = useQuery({
    queryKey: ['admin-package-comparison-plans'],
    queryFn: async () => (await getAdminPlans()).data ?? [],
  })
  const planOptions = useMemo(
    () =>
      (plansQuery.data ?? []).map((item) => ({
        value: String(item.plan.id),
        label: item.plan.title,
      })),
    [plansQuery.data]
  )
  const timeRange = useMemo(() => {
    const endTimestamp = Math.floor(Date.now() / 1000)
    return {
      startTimestamp: endTimestamp - selectedDays * 24 * 60 * 60,
      endTimestamp,
    }
  }, [selectedDays])

  useEffect(() => {
    if (selectedPlans.length === 0 && planOptions.length > 0) {
      setSelectedPlans(planOptions.slice(0, 3).map((option) => option.value))
    }
  }, [planOptions, selectedPlans.length])

  const comparisonQuery = useQuery({
    queryKey: ['package-comparison', selectedPlans, timeRange],
    enabled: selectedPlans.length > 0,
    queryFn: async () =>
      getPackageComparison({
        plan_ids: selectedPlans.map(Number),
        start_timestamp: timeRange.startTimestamp,
        end_timestamp: timeRange.endTimestamp,
      }),
  })
  const stats = comparisonQuery.data?.data?.plans ?? []
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  let comparisonContent: ReactNode
  if (plansQuery.isLoading || comparisonQuery.isLoading) {
    comparisonContent = (
      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
        {[0, 1, 2].map((item) => (
          <Skeleton key={item} className='h-52' />
        ))}
      </div>
    )
  } else if (stats.length === 0) {
    comparisonContent = (
      <div className='text-muted-foreground border py-10 text-center text-sm'>
        {t('Select at least one plan to compare usage')}
      </div>
    )
  } else {
    comparisonContent = (
      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
        {stats.map((stat) => (
          <Card key={stat.plan_id} size='sm'>
            <CardHeader className='border-b'>
              <CardTitle className='flex items-center justify-between gap-3'>
                <span className='truncate'>
                  {stat.plan_title || t('Unknown plan')}
                </span>
                <span className='text-muted-foreground shrink-0 text-xs font-normal'>
                  {stat.currency} {stat.plan_price.toFixed(2)}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className='grid grid-cols-2 gap-x-4 gap-y-3'>
              <Metric
                label={t('Requests')}
                value={formatNumber(stat.requests, locale)}
              />
              <Metric
                label={t('Channel hit rate')}
                value={`${(stat.channel_hit_rate * 100).toFixed(1)}%`}
              />
              <Metric
                label={t('Input tokens')}
                value={formatNumber(stat.prompt_tokens, locale)}
              />
              <Metric
                label={t('Output tokens')}
                value={formatNumber(stat.completion_tokens, locale)}
              />
              <Metric
                label={t('Total tokens')}
                value={formatNumber(stat.total_tokens, locale)}
              />
              <Metric
                label={t('Average latency')}
                value={`${stat.average_latency_ms.toFixed(0)} ms`}
              />
              <Metric label={t('Usage cost')} value={formatQuota(stat.quota)} />
              <Metric
                label={t('Plan quota')}
                value={
                  stat.plan_quota > 0
                    ? formatQuota(stat.plan_quota)
                    : t('Unlimited')
                }
              />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2 border-b pb-3'>
        <div className='flex items-center gap-2 text-sm font-medium'>
          <BarChart3 className='size-4' aria-hidden='true' />
          {t('Package comparison')}
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          <div className='flex items-center rounded-lg border p-0.5'>
            {TIME_RANGES.map((range) => (
              <Button
                key={range.value}
                variant={selectedDays === range.value ? 'secondary' : 'ghost'}
                size='sm'
                onClick={() => setSelectedDays(range.value)}
              >
                {t(range.labelKey)}
              </Button>
            ))}
          </div>
          <MultiSelect
            options={planOptions}
            selected={selectedPlans}
            onChange={setSelectedPlans}
            placeholder={t('Select plans')}
            maxVisibleChips={3}
            className='min-w-52'
          />
          <Button
            variant='outline'
            size='icon'
            onClick={() => void comparisonQuery.refetch()}
            disabled={comparisonQuery.isFetching}
            aria-label={t('Refresh')}
          >
            <RefreshCw
              className={comparisonQuery.isFetching ? 'animate-spin' : ''}
            />
          </Button>
        </div>
      </div>
      <p className='text-muted-foreground text-xs'>
        {t('Usage is attributed to the plan selected when each request is billed.')}
      </p>
      {comparisonContent}
    </div>
  )
}

function Metric(props: { label: string; value: string }) {
  return (
    <div className='min-w-0 text-sm'>
      <div className='text-muted-foreground truncate text-xs'>
        {props.label}
      </div>
      <div className='mt-0.5 truncate font-medium'>{props.value}</div>
    </div>
  )
}
