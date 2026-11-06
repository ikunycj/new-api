/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { CalendarDays } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  TIME_GRANULARITY_OPTIONS,
  TIME_RANGE_PRESETS,
} from '@/features/dashboard/constants'
import { getDashboardPresetRange } from '@/features/dashboard/lib'
import type {
  DashboardFilters,
  DashboardMetric,
  DashboardRangePreset,
} from '@/features/dashboard/types'
import type { TimeGranularity } from '@/lib/time'

interface DashboardChartControlsProps {
  filters: DashboardFilters
  onChange: (filters: DashboardFilters) => void
}

export function DashboardChartControls(props: DashboardChartControlsProps) {
  const { t } = useTranslation()
  const [customOpen, setCustomOpen] = useState(false)
  const [customStart, setCustomStart] = useState(props.filters.start_timestamp)
  const [customEnd, setCustomEnd] = useState(props.filters.end_timestamp)
  const customRangeValid = Boolean(
    customStart && customEnd && customStart.getTime() <= customEnd.getTime()
  )

  const updateRange = (key: string) => {
    const { start, end } = getDashboardPresetRange(key)
    props.onChange({
      ...props.filters,
      start_timestamp: start,
      end_timestamp: end,
      range_preset: key as DashboardRangePreset,
    })
  }

  const applyCustomRange = () => {
    if (!customStart || !customEnd) return
    props.onChange({
      ...props.filters,
      start_timestamp: customStart,
      end_timestamp: customEnd,
      range_preset: 'custom',
    })
    setCustomOpen(false)
  }

  return (
    <div className='flex max-w-full flex-wrap items-center justify-end gap-2'>
      <div className='flex min-w-0 items-center gap-1.5'>
        <span className='text-muted-foreground hidden text-xs sm:inline'>
          {t('Range')}
        </span>
        <Tabs
          value={props.filters.range_preset ?? 'today'}
          onValueChange={updateRange}
        >
          <TabsList className='max-w-full overflow-x-auto'>
            {TIME_RANGE_PRESETS.map((item) => (
              <TabsTrigger
                key={item.key}
                value={item.key}
                className='px-2.5 text-xs'
              >
                {t(item.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <Dialog
          open={customOpen}
          onOpenChange={(open) => {
            if (open) {
              setCustomStart(props.filters.start_timestamp)
              setCustomEnd(props.filters.end_timestamp)
            }
            setCustomOpen(open)
          }}
          trigger={
            <Button
              variant={
                props.filters.range_preset === 'custom'
                  ? 'secondary'
                  : 'outline'
              }
              size='icon-sm'
              aria-label={t('Custom Time Range')}
            >
              <CalendarDays />
            </Button>
          }
          title={t('Custom Time Range')}
          description={t('Select the start and end time for the dashboard.')}
          contentClassName='sm:max-w-md'
          contentHeight='auto'
          footer={
            <Button onClick={applyCustomRange} disabled={!customRangeValid}>
              {t('Apply')}
            </Button>
          }
        >
          <div className='grid gap-4'>
            <div className='grid gap-2'>
              <Label>{t('Start Time')}</Label>
              <DateTimePicker
                value={customStart}
                onChange={(value) => setCustomStart(value || undefined)}
              />
            </div>
            <div className='grid gap-2'>
              <Label>{t('End Time')}</Label>
              <DateTimePicker
                value={customEnd}
                onChange={(value) => setCustomEnd(value || undefined)}
              />
            </div>
          </div>
        </Dialog>
      </div>

      <div className='flex items-center gap-1.5'>
        <span className='text-muted-foreground hidden text-xs sm:inline'>
          {t('Granularity')}
        </span>
        <Tabs
          value={props.filters.time_granularity ?? 'hour'}
          onValueChange={(value) =>
            props.onChange({
              ...props.filters,
              time_granularity: value as TimeGranularity,
            })
          }
        >
          <TabsList>
            {TIME_GRANULARITY_OPTIONS.filter(
              (item) => item.value !== 'week'
            ).map((item) => (
              <TabsTrigger key={item.value} value={item.value}>
                {t(item.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      <Tabs
        value={props.filters.metric ?? 'tokens'}
        onValueChange={(value) =>
          props.onChange({ ...props.filters, metric: value as DashboardMetric })
        }
      >
        <TabsList>
          <TabsTrigger value='tokens'>{t('By Token')}</TabsTrigger>
          <TabsTrigger value='quota'>{t('By Cost')}</TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  )
}
