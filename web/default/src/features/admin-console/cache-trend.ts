import type { QuotaDataItem } from '@/features/dashboard/types'
import { formatChartTime, type TimeGranularity } from '@/lib/time'

import type { AdminConsoleCacheTrendPoint } from './types'

export interface CacheTrendChartValue {
  Time: string
  CacheRate: number
  Model: string
}

interface AreaChartValue {
  Time?: unknown
  Model?: unknown
}

function cacheTrendKey(time: string, model: string): string {
  return `${time}\u0000${model}`
}

export function buildCacheTrendChartValues(
  dimensionData: QuotaDataItem[],
  areaValues: AreaChartValue[],
  cacheTrendPoints: AdminConsoleCacheTrendPoint[],
  timeGranularity: TimeGranularity
): CacheTrendChartValue[] {
  const models = new Set<string>()
  for (const item of dimensionData) {
    const model = item.model_name?.trim()
    if (model) models.add(model)
  }
  if (models.size === 0) return []

  const times = new Set<string>()
  for (const value of areaValues) {
    const time = typeof value.Time === 'string' ? value.Time.trim() : ''
    if (time) times.add(time)
  }
  if (times.size === 0) {
    for (const item of dimensionData) {
      const timestamp = Number(item.created_at)
      if (Number.isFinite(timestamp)) {
        times.add(formatChartTime(timestamp, timeGranularity))
      }
    }
  }

  const rates = new Map<string, number>()
  for (const point of cacheTrendPoints) {
    const model = point.name?.trim()
    const timestamp = Number(point.timestamp)
    if (!model || !models.has(model) || !Number.isFinite(timestamp)) continue

    const time = formatChartTime(timestamp, timeGranularity)
    times.add(time)
    const inputTokens = Number(point.cache_input_tokens)
    const hitRate = Number(point.cache_hit_rate)
    rates.set(
      cacheTrendKey(time, model),
      inputTokens > 0 && Number.isFinite(hitRate) ? hitRate : 0
    )
  }

  if (times.size === 0) return []

  const values: CacheTrendChartValue[] = []
  for (const time of [...times].sort()) {
    for (const model of [...models].sort()) {
      values.push({
        Time: time,
        CacheRate: rates.get(cacheTrendKey(time, model)) ?? 0,
        Model: model,
      })
    }
  }
  return values
}
