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
import { QUOTA_TYPE_VALUES, SORT_OPTIONS, type SortOption } from '../constants'
import type { PricingDisplayModel, PricingModel } from '../types'
import { getDynamicPricingTiers, isDynamicPricingModel } from './dynamic-price'
import { calculateTokenPrice } from './price'

// ----------------------------------------------------------------------------
// Filter Utilities
// ----------------------------------------------------------------------------

/**
 * Filter models by search query
 */
function filterBySearch(
  models: PricingDisplayModel[],
  query: string
): PricingDisplayModel[] {
  if (!query) return models

  const lowerQuery = query.trim().toLowerCase()
  if (!lowerQuery) return models

  return models.filter(
    (m) =>
      m.model_name?.toLowerCase().includes(lowerQuery) ||
      m.vendor_name?.toLowerCase().includes(lowerQuery) ||
      m.enable_groups?.some((group) => group.toLowerCase().includes(lowerQuery))
  )
}

/**
 * Filter models by vendor
 */
function filterByVendor(
  models: PricingDisplayModel[],
  vendors: readonly string[]
): PricingDisplayModel[] {
  if (vendors.length === 0) return models
  const selectedVendors = new Set(vendors)
  return models.filter((model) => selectedVendors.has(model.vendor_name || ''))
}

/**
 * Filter models by group
 */
export function filterByGroup<T extends PricingModel>(
  models: T[],
  groups: readonly string[]
): T[] {
  if (groups.length === 0) return models
  const selectedGroups = new Set(groups)
  return models.filter((model) =>
    model.enable_groups?.some((group) => selectedGroups.has(group))
  )
}

function getModelInputPrice(model: PricingDisplayModel): number | undefined {
  if (isDynamicPricingModel(model)) {
    for (const tier of getDynamicPricingTiers(model)) {
      // Use the first input price shown on the card. Zero token rates in
      // parsed tiers also represent unconfigured fields and are not shown.
      if (
        tier.billingUnit === 'token' &&
        typeof tier.inputPrice === 'number' &&
        tier.inputPrice > 0
      ) {
        return tier.inputPrice * model.display_group_ratio
      }
    }
    return undefined
  }

  if (model.quota_type === QUOTA_TYPE_VALUES.TOKEN) {
    return calculateTokenPrice(model, 'input', model.display_group_ratio)
  }
}

/**
 * Sort models by specified option
 */
export function sortModels(
  models: PricingDisplayModel[],
  sortBy: SortOption,
  rankedModelOrder: readonly string[] = []
): PricingDisplayModel[] {
  if (sortBy === SORT_OPTIONS.PRICE_LOW || sortBy === SORT_OPTIONS.PRICE_HIGH) {
    const direction = sortBy === SORT_OPTIONS.PRICE_LOW ? 1 : -1
    return models
      .map((model) => ({ model, price: getModelInputPrice(model) }))
      .sort((a, b) => {
        if (a.price === b.price) return 0
        if (a.price === undefined) return 1
        if (b.price === undefined) return -1
        return direction * (a.price - b.price)
      })
      .map(({ model }) => model)
  }

  const sorted = [...models]

  switch (sortBy) {
    case SORT_OPTIONS.AGENT:
    case SORT_OPTIONS.CODING:
    case SORT_OPTIONS.CHAT:
    case SORT_OPTIONS.BEST_VALUE: {
      const modelRanks = new Map<string, number>(
        rankedModelOrder.map((modelName, index) => [modelName, index])
      )
      sorted.sort((a, b) => {
        const aRank = modelRanks.get(a.model_name) ?? rankedModelOrder.length
        const bRank = modelRanks.get(b.model_name) ?? rankedModelOrder.length

        return (
          aRank - bRank ||
          (a.model_name || '').localeCompare(b.model_name || '')
        )
      })
      break
    }
    case SORT_OPTIONS.NAME:
      sorted.sort((a, b) =>
        (a.model_name || '').localeCompare(b.model_name || '')
      )
      break
  }

  return sorted
}

/**
 * Apply all filters and sorting to models
 */
export function filterAndSortModels(
  models: PricingDisplayModel[],
  filters: {
    search: string
    vendors: readonly string[]
    groups: readonly string[]
    sortBy: SortOption
  },
  rankedModelOrder: readonly string[] = []
): PricingDisplayModel[] {
  let result = filterBySearch(models, filters.search)
  result = filterByVendor(result, filters.vendors)
  result = filterByGroup(result, filters.groups)
  return sortModels(result, filters.sortBy, rankedModelOrder)
}

/**
 * Parse tags from comma-separated string
 */
export function parseTags(tagsString?: string): string[] {
  if (!tagsString) return []
  return tagsString
    .split(/[,;|\s]+/)
    .map((t) => t.trim())
    .filter(Boolean)
}
