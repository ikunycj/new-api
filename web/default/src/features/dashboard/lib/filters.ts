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
import {
  DEFAULT_TIME_GRANULARITY,
  EMPTY_DASHBOARD_FILTERS,
  TIME_RANGE_BY_GRANULARITY,
} from '@/features/dashboard/constants'
import type { DashboardFilters } from '@/features/dashboard/types'
import { getEndOfDay, getStartOfDay, type TimeGranularity } from '@/lib/time'

function isTimeGranularity(value: unknown): value is TimeGranularity {
  return value === 'hour' || value === 'day' || value === 'week'
}

export function cleanFilters<T extends Record<string, unknown>>(
  filters: T
): Partial<T> {
  const cleaned: Partial<T> = {}
  for (const [key, value] of Object.entries(filters)) {
    if (value === undefined || value === null) continue
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (trimmed) cleaned[key as keyof T] = trimmed as T[keyof T]
      continue
    }
    cleaned[key as keyof T] = value as T[keyof T]
  }
  return cleaned
}

export function getSavedGranularity(
  override?: TimeGranularity
): TimeGranularity {
  return override && isTimeGranularity(override)
    ? override
    : DEFAULT_TIME_GRANULARITY
}

export function getDefaultDays(granularity?: TimeGranularity): number {
  return TIME_RANGE_BY_GRANULARITY[getSavedGranularity(granularity)]
}

export function buildDefaultDashboardFilters(): DashboardFilters {
  const { start, end } = getDashboardPresetRange('today')
  return {
    ...EMPTY_DASHBOARD_FILTERS,
    start_timestamp: start,
    end_timestamp: end,
    time_granularity: DEFAULT_TIME_GRANULARITY,
    metric: 'tokens',
    range_preset: 'today',
  }
}

export function getDashboardPresetRange(key: string): {
  start: Date
  end: Date
} {
  const now = new Date()
  if (key === 'yesterday') {
    const yesterday = new Date(now)
    yesterday.setDate(yesterday.getDate() - 1)
    return { start: getStartOfDay(yesterday), end: getEndOfDay(yesterday) }
  }
  if (key === 'week') {
    const start = new Date(now)
    const day = start.getDay() || 7
    start.setDate(start.getDate() - day + 1)
    return { start: getStartOfDay(start), end: getEndOfDay(now) }
  }
  if (key === 'month') {
    const start = new Date(now.getFullYear(), now.getMonth(), 1)
    return { start: getStartOfDay(start), end: getEndOfDay(now) }
  }
  return { start: getStartOfDay(now), end: getEndOfDay(now) }
}

export function buildQueryParams(
  timeRange: { start_timestamp: number; end_timestamp: number },
  filters?: { time_granularity?: TimeGranularity; username?: string }
): {
  start_timestamp: number
  end_timestamp: number
  default_time: string
  username?: string
} {
  return {
    ...timeRange,
    default_time: getSavedGranularity(filters?.time_granularity),
    ...(filters?.username && { username: filters.username }),
  }
}
