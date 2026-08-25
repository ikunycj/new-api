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
import type { LoadTestPricing } from './api'
import { QUOTA_TYPE_VALUES } from '../pricing/constants'

export type LoadTestUsageTotals = {
  successes: number
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
}

export function getLoadTestTotalTokens(usage: LoadTestUsageTotals): number {
  return (
    usage.inputTokens +
    usage.outputTokens +
    usage.cacheReadTokens +
    usage.cacheWriteTokens
  )
}

export function calculateLoadTestUserCharge(
  usage: LoadTestUsageTotals,
  pricing: LoadTestPricing
): number {
  if (pricing.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return (
      usage.successes * (pricing.model.model_price ?? 0) * pricing.groupRatio
    )
  }

  const inputPricePerMillion = pricing.model.model_ratio * 2
  const outputPricePerMillion =
    inputPricePerMillion * pricing.model.completion_ratio
  const cacheReadPricePerMillion =
    inputPricePerMillion * (pricing.model.cache_ratio ?? 1)
  const cacheWritePricePerMillion =
    inputPricePerMillion * (pricing.model.create_cache_ratio ?? 1)
  const officialCost =
    (usage.inputTokens / 1_000_000) * inputPricePerMillion +
    (usage.outputTokens / 1_000_000) * outputPricePerMillion +
    (usage.cacheReadTokens / 1_000_000) * cacheReadPricePerMillion +
    (usage.cacheWriteTokens / 1_000_000) * cacheWritePricePerMillion

  return officialCost * pricing.groupRatio
}
