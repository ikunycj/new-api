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
import { useSearch } from '@tanstack/react-router'
import { useMemo, useCallback, useState } from 'react'

import {
  useModelLeaderboards,
  type ModelLeaderboardKind,
} from '@/hooks/use-model-leaderboards'

import {
  SORT_OPTIONS,
  DEFAULT_TOKEN_UNIT,
  DEFAULT_PRICING_CURRENCY,
  PRICING_CURRENCIES,
  VIEW_MODES,
  type PricingCurrency,
  type ViewMode,
} from '../constants'
import { filterAndSortModels } from '../lib/filters'
import type { PricingModel, TokenUnit } from '../types'

type FilterState = {
  search?: string
  sort?: string
  leaderboard?: ModelLeaderboardKind
  vendors?: string[]
  groups?: string[]
  tokenUnit?: TokenUnit
  currency?: PricingCurrency
  view?: ViewMode
}

function normalizeViewMode(value: unknown): ViewMode {
  if (value === VIEW_MODES.TABLE) {
    return VIEW_MODES.TABLE
  }
  return VIEW_MODES.CARD
}

function parseSearchValues(value?: string): string[] | undefined {
  const values = value
    ?.split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  return values && values.length > 0 ? values : undefined
}

export function useFilters(models: PricingModel[]) {
  const { data: leaderboards } = useModelLeaderboards()
  const search = useSearch({ from: '/pricing/' })
  const [filterState, setFilterState] = useState<FilterState>(() => ({
    search: search.search,
    sort: search.sort,
    vendors: parseSearchValues(search.vendor),
    groups: parseSearchValues(search.group),
    tokenUnit: search.tokenUnit,
    currency: search.currency,
    view: search.view,
  }))

  const searchInput = filterState.search || ''
  const sortBy = filterState.sort || SORT_OPTIONS.RECOMMENDED
  const activeLeaderboard =
    filterState.leaderboard ?? leaderboards?.default ?? 'coding'
  const selectedLeaderboard =
    sortBy === SORT_OPTIONS.RECOMMENDED ? activeLeaderboard : null
  const leaderboardEntries = selectedLeaderboard
    ? leaderboards?.boards[selectedLeaderboard]
    : undefined
  const rankedModelOrder = useMemo(() => {
    const rankedEntries = [...(leaderboardEntries ?? [])].sort(
      (a, b) => a.rank - b.rank
    )
    return [...new Set(rankedEntries.flatMap((entry) => entry.model_ids ?? []))]
  }, [leaderboardEntries])
  const vendorFilters = useMemo(
    () => filterState.vendors ?? [],
    [filterState.vendors]
  )
  const groupFilters = useMemo(
    () => filterState.groups ?? [],
    [filterState.groups]
  )
  const tokenUnit: TokenUnit =
    filterState.tokenUnit === 'K' ? 'K' : DEFAULT_TOKEN_UNIT
  const displayCurrency: PricingCurrency =
    filterState.currency === PRICING_CURRENCIES.USD
      ? PRICING_CURRENCIES.USD
      : DEFAULT_PRICING_CURRENCY
  const viewMode = normalizeViewMode(filterState.view)

  const updateFilters = useCallback((updates: Record<string, unknown>) => {
    setFilterState((prev) => {
      const next: Record<string, unknown> = { ...prev, ...updates }
      for (const key of Object.keys(next)) {
        if (next[key] === undefined || next[key] === null) {
          delete next[key]
        }
      }
      return next as FilterState
    })
  }, [])

  const setSearchInput = useCallback(
    (v: string) => updateFilters({ search: v || undefined }),
    [updateFilters]
  )
  const setSortBy = useCallback(
    (v: string) =>
      updateFilters({
        sort: v === SORT_OPTIONS.RECOMMENDED ? undefined : v,
      }),
    [updateFilters]
  )
  const setLeaderboard = useCallback(
    (value: ModelLeaderboardKind) =>
      updateFilters({ leaderboard: value, sort: undefined }),
    [updateFilters]
  )
  const toggleSelection = useCallback(
    (key: 'vendors' | 'groups', value: string) => {
      setFilterState((previous) => {
        const current = previous[key] || []
        const next = current.includes(value)
          ? current.filter((item) => item !== value)
          : [...current, value]

        return { ...previous, [key]: next.length > 0 ? next : undefined }
      })
    },
    []
  )
  const toggleVendorFilter = useCallback(
    (value: string) => toggleSelection('vendors', value),
    [toggleSelection]
  )
  const toggleGroupFilter = useCallback(
    (value: string) => toggleSelection('groups', value),
    [toggleSelection]
  )
  const setTokenUnit = useCallback(
    (v: TokenUnit) =>
      updateFilters({ tokenUnit: v === DEFAULT_TOKEN_UNIT ? undefined : v }),
    [updateFilters]
  )
  const setDisplayCurrency = useCallback(
    (value: PricingCurrency) =>
      updateFilters({
        currency: value === DEFAULT_PRICING_CURRENCY ? undefined : value,
      }),
    [updateFilters]
  )
  const setViewMode = useCallback(
    (v: ViewMode) =>
      updateFilters({ view: v === VIEW_MODES.CARD ? undefined : v }),
    [updateFilters]
  )

  const filteredModels = useMemo(() => {
    if (!models || models.length === 0) return []

    return filterAndSortModels(
      models,
      {
        search: searchInput,
        vendors: vendorFilters,
        groups: groupFilters,
        sortBy,
      },
      rankedModelOrder
    )
  }, [
    models,
    searchInput,
    vendorFilters,
    groupFilters,
    sortBy,
    rankedModelOrder,
  ])

  const hasActiveFilters = vendorFilters.length > 0 || groupFilters.length > 0

  const clearFilters = useCallback(() => {
    updateFilters({
      vendors: undefined,
      groups: undefined,
    })
  }, [updateFilters])

  const clearSearch = useCallback(() => {
    updateFilters({ search: undefined })
  }, [updateFilters])

  return {
    searchInput,
    sortBy,
    selectedLeaderboard,
    vendorFilters,
    groupFilters,
    tokenUnit,
    displayCurrency,
    viewMode,
    setSearchInput,
    setSortBy,
    setLeaderboard,
    toggleVendorFilter,
    toggleGroupFilter,
    setTokenUnit,
    setDisplayCurrency,
    setViewMode,
    filteredModels,
    hasActiveFilters,
    clearFilters,
    clearSearch,
  }
}
