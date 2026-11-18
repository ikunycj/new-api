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
export const STRATEGY_WEIGHT_FIELDS = [
  ['价格', 'priceWeight'],
  ['可用性', 'availabilityWeight'],
  ['负载', 'loadWeight'],
  ['昨日平均TTFT', 'ttftWeight'],
  ['最近测试TTFT', 'recentTestTTFTWeight'],
  ['测试模型置信度', 'iqTestConfidenceWeight'],
] as const

export type StrategyWeightDraft = Record<
  (typeof STRATEGY_WEIGHT_FIELDS)[number][1],
  string
>

export function strategyWeightPercentages(
  draft: StrategyWeightDraft
): number[] | null {
  const weights = STRATEGY_WEIGHT_FIELDS.map(([, field]) =>
    draft[field].trim() === '' ? Number.NaN : Number(draft[field])
  )
  if (weights.some((weight) => !Number.isFinite(weight) || weight < 0)) {
    return null
  }
  const largest = Math.max(...weights)
  if (largest === 0) return null
  const scaled = weights.map((weight) => weight / largest)
  const total = scaled.reduce((sum, weight) => sum + weight, 0)
  return scaled.map((weight) => (weight / total) * 100)
}
