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
  formatBillingCurrencyFromUSD,
  getCurrencyDisplay,
} from '@/lib/currency'

const USAGE_COST_FORMAT_OPTIONS = {
  abbreviate: false,
  digitsLarge: 2,
  digitsSmall: 4,
} as const

function normalizeUsageValue(value: number | null | undefined): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
    ? value
    : 0
}

/** Format raw quota units as the configured monetary billing display. */
export function formatApiKeyUsageCost(
  quota: number | null | undefined,
  locale?: Intl.LocalesArgument
): string {
  const quotaPerUnit = getCurrencyDisplay().config.quotaPerUnit
  const amountUSD = normalizeUsageValue(quota) / quotaPerUnit
  return formatBillingCurrencyFromUSD(amountUSD, {
    ...USAGE_COST_FORMAT_OPTIONS,
    locale,
  })
}

/**
 * Format API key token usage with a compact, stable unit suffix.
 *
 * Usage metrics can grow beyond the range where an ungrouped integer is
 * useful in a table cell, so values are represented as K, M, or B while
 * retaining up to two meaningful decimal places.
 */
export function formatApiKeyTokens(
  value: number | null | undefined,
  locale?: Intl.LocalesArgument
): string {
  const tokens = normalizeUsageValue(value)

  if (tokens < 1_000) {
    return new Intl.NumberFormat(locale, {
      maximumFractionDigits: 0,
    }).format(tokens)
  }

  let divisor = 1_000
  let suffix = 'K'
  if (tokens >= 1_000_000_000) {
    divisor = 1_000_000_000
    suffix = 'B'
  } else if (tokens >= 1_000_000) {
    divisor = 1_000_000
    suffix = 'M'
  }

  const scaled = tokens / divisor
  const formatted = new Intl.NumberFormat(locale, {
    maximumFractionDigits: 2,
  }).format(scaled)

  return `${formatted}${suffix}`
}
