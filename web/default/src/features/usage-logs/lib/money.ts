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
import { useSystemConfigStore } from '@/stores/system-config-store'

interface UsageLogQuotaSnapshot {
  billingUSDToCNYRate?: number
  quotaPerUnit?: number
}

function positiveOrFallback(value: number | undefined, fallback: number) {
  return value != null && Number.isFinite(value) && value > 0 ? value : fallback
}

export function usageLogQuotaToUSD(
  quota: number,
  snapshot?: UsageLogQuotaSnapshot
): number {
  if (!Number.isFinite(quota)) return 0

  const configuredQuotaPerUnit =
    useSystemConfigStore.getState().config.currency.quotaPerUnit
  const quotaPerUnit = positiveOrFallback(
    snapshot?.quotaPerUnit,
    positiveOrFallback(configuredQuotaPerUnit, 500_000)
  )
  const billingRate = positiveOrFallback(snapshot?.billingUSDToCNYRate, 1)

  return quota / quotaPerUnit / billingRate
}

export function formatUsageLogUSD(amount: number | null | undefined): string {
  if (amount == null || !Number.isFinite(amount)) return '-'

  const maximumFractionDigits = Math.abs(amount) < 0.01 ? 6 : 4
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 0,
    maximumFractionDigits,
  }).format(amount)
}

export function formatUsageLogQuotaUSD(
  quota: number,
  snapshot?: UsageLogQuotaSnapshot
): string {
  return formatUsageLogUSD(usageLogQuotaToUSD(quota, snapshot))
}

export function formatUsageLogUSDMicros(
  micros: number | null | undefined
): string {
  if (micros == null || !Number.isFinite(micros)) return '-'
  return formatUsageLogUSD(micros / 1_000_000)
}
