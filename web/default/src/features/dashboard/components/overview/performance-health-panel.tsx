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
import { HeartPulse } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { IconBadge } from '@/components/ui/icon-badge'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { GroupStatusPanel } from '@/features/channel-monitors/group-status'

const PERFORMANCE_WINDOW_HOURS = [24, 24 * 7, 24 * 30] as const
type PerformanceWindowHours = (typeof PERFORMANCE_WINDOW_HOURS)[number]

export function PerformanceHealthPanel() {
  const { t } = useTranslation()
  const [windowHours, setWindowHours] = useState<PerformanceWindowHours>(24)

  return (
    <section className='bg-card h-full overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex flex-wrap items-center gap-2 border-b px-4 py-3 sm:flex-nowrap sm:px-5'>
        <IconBadge tone='success' size='sm'>
          <HeartPulse />
        </IconBadge>
        <h3 className='text-sm font-semibold'>{t('Performance health')}</h3>
        <ToggleGroup
          value={[String(windowHours)]}
          onValueChange={(values) => {
            const next = Number(values[0])
            if (
              next === PERFORMANCE_WINDOW_HOURS[0] ||
              next === PERFORMANCE_WINDOW_HOURS[1] ||
              next === PERFORMANCE_WINDOW_HOURS[2]
            ) {
              setWindowHours(next)
            }
          }}
          className='bg-muted/60 ml-auto grid w-full grid-cols-3 rounded-md border p-0.5 sm:w-fit'
          aria-label={t('Period')}
        >
          <ToggleGroupItem
            value='24'
            size='sm'
            className='h-7 min-w-0 px-1 text-[10px] leading-none whitespace-nowrap sm:px-2 sm:text-[11px]'
          >
            {t('Last 24 hours')}
          </ToggleGroupItem>
          <ToggleGroupItem
            value='168'
            size='sm'
            className='h-7 min-w-0 px-1 text-[10px] leading-none whitespace-nowrap sm:px-2 sm:text-[11px]'
          >
            {t('Last 7 days')}
          </ToggleGroupItem>
          <ToggleGroupItem
            value='720'
            size='sm'
            className='h-7 min-w-0 px-1 text-[10px] leading-none whitespace-nowrap sm:px-2 sm:text-[11px]'
          >
            {t('Last 30 days')}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      <div className='p-4 sm:p-5'>
        <GroupStatusPanel periodHours={windowHours} />
      </div>
    </section>
  )
}
