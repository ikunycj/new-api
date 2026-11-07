import type { QuotaDataItem } from '@/features/dashboard/types'
import { formatChartTime, type TimeGranularity } from '@/lib/time'

import { formatChannelDisplayName } from './channel-display'
import type { AdminConsoleCacheTrendPoint } from './types'

export interface CacheTrendChartValue {
  Time: string
  CacheRate: number
  CacheInputTokens: number
  CacheReadTokens: number
  CacheWriteTokens: number
  CacheHitRequests: number
  CacheEligibleRequests: number
  Model: string
}

export interface CacheTrendSummary {
  cacheInputTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  cacheHitRequests: number
  cacheEligibleRequests: number
  tokenHitRate: number
  requestHitRate: number
}

interface AreaChartValue {
  Time?: unknown
  Model?: unknown
}

function cacheTrendKey(time: string, model: string): string {
  return `${time}\u0000${model}`
}

function nonNegativeNumber(value: unknown): number {
  const number = Number(value)
  return Number.isFinite(number) && number > 0 ? number : 0
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

  const cacheMetrics = new Map<
    string,
    Omit<CacheTrendChartValue, 'Time' | 'Model'>
  >()
  for (const point of cacheTrendPoints) {
    const model =
      point.channel_id !== undefined
        ? formatChannelDisplayName(point.name, point.channel_id)
        : point.name?.trim()
    const timestamp = Number(point.timestamp)
    if (!model || !models.has(model) || !Number.isFinite(timestamp)) continue

    const time = formatChartTime(timestamp, timeGranularity)
    times.add(time)
    const inputTokens = nonNegativeNumber(point.cache_input_tokens)
    const hitRate = Number(point.cache_hit_rate)
    cacheMetrics.set(cacheTrendKey(time, model), {
      CacheRate:
        inputTokens > 0 && Number.isFinite(hitRate) ? Math.max(hitRate, 0) : 0,
      CacheInputTokens: inputTokens,
      CacheReadTokens: nonNegativeNumber(point.cache_read_tokens),
      CacheWriteTokens: nonNegativeNumber(point.cache_write_tokens),
      CacheHitRequests: nonNegativeNumber(point.cache_hit_requests),
      CacheEligibleRequests: nonNegativeNumber(point.cache_eligible_requests),
    })
  }

  if (times.size === 0) return []

  const values: CacheTrendChartValue[] = []
  for (const time of [...times].sort()) {
    for (const model of [...models].sort()) {
      const metrics = cacheMetrics.get(cacheTrendKey(time, model))
      values.push({
        Time: time,
        CacheRate: metrics?.CacheRate ?? 0,
        CacheInputTokens: metrics?.CacheInputTokens ?? 0,
        CacheReadTokens: metrics?.CacheReadTokens ?? 0,
        CacheWriteTokens: metrics?.CacheWriteTokens ?? 0,
        CacheHitRequests: metrics?.CacheHitRequests ?? 0,
        CacheEligibleRequests: metrics?.CacheEligibleRequests ?? 0,
        Model: model,
      })
    }
  }
  return values
}

export function summarizeCacheTrend(
  values: CacheTrendChartValue[],
  selectedModels?: readonly string[]
): CacheTrendSummary {
  const selectedModelSet = selectedModels ? new Set(selectedModels) : undefined
  const summary = values.reduce(
    (result, value) => {
      if (selectedModelSet && !selectedModelSet.has(value.Model)) return result
      result.cacheInputTokens += nonNegativeNumber(value.CacheInputTokens)
      result.cacheReadTokens += nonNegativeNumber(value.CacheReadTokens)
      result.cacheWriteTokens += nonNegativeNumber(value.CacheWriteTokens)
      result.cacheHitRequests += nonNegativeNumber(value.CacheHitRequests)
      result.cacheEligibleRequests += nonNegativeNumber(
        value.CacheEligibleRequests
      )
      return result
    },
    {
      cacheInputTokens: 0,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
      cacheHitRequests: 0,
      cacheEligibleRequests: 0,
    }
  )

  return {
    ...summary,
    tokenHitRate:
      summary.cacheInputTokens > 0
        ? (summary.cacheReadTokens / summary.cacheInputTokens) * 100
        : 0,
    requestHitRate:
      summary.cacheEligibleRequests > 0
        ? (summary.cacheHitRequests / summary.cacheEligibleRequests) * 100
        : 0,
  }
}
