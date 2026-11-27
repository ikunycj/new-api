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
import type { TokenUnit } from './types'

// ----------------------------------------------------------------------------
// Pricing Constants
// ----------------------------------------------------------------------------

/** Sort options for pricing models */
export const SORT_OPTIONS = {
  BEST_VALUE: 'best-value',
  AGENT: 'agent',
  CODING: 'coding',
  CHAT: 'chat',
  NAME: 'name',
  PRICE_LOW: 'price-low',
  PRICE_HIGH: 'price-high',
} as const

export type SortOption = (typeof SORT_OPTIONS)[keyof typeof SORT_OPTIONS]

export const SORT_OPTION_VALUES = Object.values(SORT_OPTIONS)

/**
 * Empty filter values represent "all" so a configured pricing group named
 * "all" remains a real catalog entry instead of colliding with UI state.
 */
export const FILTER_ALL = ''

/** Excluded groups */
export const EXCLUDED_GROUPS = ['', 'auto']

/** Quota type values */
export const QUOTA_TYPE_VALUES = {
  TOKEN: 0,
  REQUEST: 1,
} as const

/** Token unit divisors */
export const TOKEN_UNIT_DIVISORS = {
  M: 1,
  K: 1000,
} as const

/** Default token unit for pricing display */
export const DEFAULT_TOKEN_UNIT: TokenUnit = 'M'

/** Currencies supported by the public model catalog. */
export const PRICING_CURRENCIES = {
  CNY: 'CNY',
  USD: 'USD',
} as const

export type PricingCurrency =
  (typeof PRICING_CURRENCIES)[keyof typeof PRICING_CURRENCIES]

export const DEFAULT_PRICING_CURRENCY: PricingCurrency = PRICING_CURRENCIES.CNY

/** View mode options */
export const VIEW_MODES = {
  CARD: 'card',
  TABLE: 'table',
} as const

export type ViewMode = (typeof VIEW_MODES)[keyof typeof VIEW_MODES]

/** Number of model cards revealed in each infinite-scroll batch. */
export const PRICING_CARD_BATCH_SIZE = 18

/** Default page size for the pricing table view. */
export const DEFAULT_PRICING_PAGE_SIZE = 18
