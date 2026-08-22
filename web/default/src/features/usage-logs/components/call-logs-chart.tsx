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
import { VChart } from '@visactor/react-vchart'

import { Skeleton } from '@/components/ui/skeleton'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

interface CallLogsChartProps {
  spec: Record<string, unknown> | null
  loading: boolean
  emptyText: string
  className?: string
}

export function CallLogsChart(props: CallLogsChartProps) {
  const { resolvedTheme, themeReady } = useChartTheme()

  if (props.loading) {
    return <Skeleton className={props.className ?? 'h-64 w-full'} />
  }
  if (!props.spec) {
    return (
      <div
        className={`text-muted-foreground flex items-center justify-center text-xs ${props.className ?? 'h-64 w-full'}`}
      >
        {props.emptyText}
      </div>
    )
  }
  if (!themeReady) {
    return <Skeleton className={props.className ?? 'h-64 w-full'} />
  }

  return (
    <div className={props.className ?? 'h-64 w-full'}>
      <VChart
        spec={{
          ...props.spec,
          theme: resolvedTheme === 'dark' ? 'dark' : 'light',
          background: 'transparent',
        }}
        option={VCHART_OPTION}
      />
    </div>
  )
}
