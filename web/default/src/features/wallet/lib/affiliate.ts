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
import { formatQuotaAsCNY, quotaUnitsToCNY } from '@/lib/format'

import type { AffiliateSummary } from '../types'

// ============================================================================
// Affiliate Functions
// ============================================================================

/**
 * Generate affiliate registration link
 */
export function generateAffiliateLink(affCode: string): string {
  if (typeof window === 'undefined') return ''
  return `${window.location.origin}/sign-up?aff=${affCode}`
}

export function formatAffiliateCashback(cents: number, quota = 0): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'CNY',
  }).format(cents / 100 + (quota === 0 ? 0 : quotaUnitsToCNY(quota)))
}

export function formatAffiliateAccountCashback(
  summary: AffiliateSummary | undefined,
  field: 'pending' | 'available' | 'transferred' | 'lifetime_earned'
): string {
  const cents = summary?.cash_account[`${field}_cents`] ?? 0
  const quota = summary?.account[`${field}_quota`] ?? 0
  if (summary?.has_sub_agent_rewards && cents > 0 && quota > 0) {
    return formatAffiliateCashback(cents, quota)
  }
  if (summary?.has_sub_agent_rewards && quota > 0) {
    return formatQuotaAsCNY(quota)
  }
  const useCashAccount =
    (summary?.cash_account.lifetime_earned_cents ?? 0) > 0 ||
    summary?.campaign.enabled === true
  return useCashAccount
    ? formatAffiliateCashback(cents)
    : formatQuotaAsCNY(quota)
}
