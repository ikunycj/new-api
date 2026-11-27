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
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'

import {
  LoadingSkeleton,
  EmptyState,
  PricingTable,
  PricingToolbar,
  ModelCardGrid,
  ModelDetailsDrawer,
} from './components'
import { PRICING_CURRENCIES, VIEW_MODES } from './constants'
import { useFilters } from './hooks/use-filters'
import { usePricingData } from './hooks/use-pricing-data'
import { expandModelsByGroup, getCatalogGroups } from './lib/model-helpers'

export function Pricing() {
  const { t } = useTranslation()
  const [selectedModelName, setSelectedModelName] = useState<string | null>(
    null
  )

  const {
    models,
    vendors,
    groupRatio,
    catalogGroups,
    endpointMap,
    isLoading,
    priceRate,
    billingUSDToCNYRate,
  } = usePricingData()

  const availableGroups = useMemo(
    () => getCatalogGroups(catalogGroups),
    [catalogGroups]
  )
  const displayModels = useMemo(
    () =>
      expandModelsByGroup(models, availableGroups, groupRatio, catalogGroups),
    [models, availableGroups, groupRatio, catalogGroups]
  )

  const {
    searchInput,
    sortingRule,
    vendorFilters,
    groupFilters,
    tokenUnit,
    displayCurrency,
    viewMode,
    setSearchInput,
    setSortingRule,
    toggleVendorFilter,
    toggleGroupFilter,
    setTokenUnit,
    setDisplayCurrency,
    setViewMode,
    filteredModels,
    hasActiveFilters,
    clearFilters,
    clearSearch,
  } = useFilters(displayModels)
  const showPricesInCny = displayCurrency === PRICING_CURRENCIES.CNY

  const handleModelClick = useCallback((modelName: string) => {
    setSelectedModelName(modelName)
  }, [])

  const selectedModel = useMemo(
    () =>
      selectedModelName
        ? (models || []).find(
            (model) => model.model_name === selectedModelName
          ) || null
        : null,
    [models, selectedModelName]
  )

  const handleClearAll = useCallback(() => {
    clearFilters()
    clearSearch()
  }, [clearFilters, clearSearch])

  const renderPricingContent = () => {
    if (filteredModels.length === 0) {
      return (
        <EmptyState
          searchQuery={searchInput}
          hasActiveFilters={hasActiveFilters}
          onClearFilters={handleClearAll}
        />
      )
    }

    if (viewMode === VIEW_MODES.CARD) {
      return (
        <ModelCardGrid
          models={filteredModels}
          onModelClick={handleModelClick}
          priceRate={priceRate}
          usdExchangeRate={billingUSDToCNYRate}
          tokenUnit={tokenUnit}
          showRechargePrice={showPricesInCny}
        />
      )
    }

    return (
      <PricingTable
        models={filteredModels}
        priceRate={priceRate}
        usdExchangeRate={billingUSDToCNYRate}
        tokenUnit={tokenUnit}
        showRechargePrice={showPricesInCny}
        onModelClick={handleModelClick}
      />
    )
  }

  if (isLoading) {
    return (
      <PublicLayout showMainContainer={false} showCustomerServiceInfo={false}>
        <div className='mx-auto w-full max-w-[1280px] px-4 pt-20 pb-10 sm:px-6 lg:px-8'>
          <LoadingSkeleton viewMode={viewMode} />
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false} showCustomerServiceInfo={false}>
      <PageTransition className='mx-auto w-full max-w-[1280px] px-4 pt-20 pb-10 sm:px-6 lg:px-8'>
        <header className='mb-4'>
          <h1 className='text-3xl leading-tight font-bold sm:text-4xl'>
            {t('Model Square')}
          </h1>
        </header>

        <main className='flex min-w-0 flex-col gap-5'>
          <PricingToolbar
            searchInput={searchInput}
            onSearchChange={setSearchInput}
            onClearSearch={clearSearch}
            sortingRule={sortingRule}
            onSortingRuleChange={setSortingRule}
            tokenUnit={tokenUnit}
            onTokenUnitChange={setTokenUnit}
            displayCurrency={displayCurrency}
            onCurrencyChange={setDisplayCurrency}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            vendorFilters={vendorFilters}
            groupFilters={groupFilters}
            onVendorToggle={toggleVendorFilter}
            onGroupToggle={toggleGroupFilter}
            vendors={vendors || []}
            groups={availableGroups}
            groupLabels={catalogGroups || {}}
            groupRatios={groupRatio || {}}
            models={models || []}
          />

          {renderPricingContent()}
        </main>

        {selectedModel && (
          <ModelDetailsDrawer
            open={Boolean(selectedModel)}
            onOpenChange={(open) => {
              if (!open) setSelectedModelName(null)
            }}
            model={selectedModel}
            groupRatio={groupRatio || {}}
            catalogGroups={catalogGroups || {}}
            endpointMap={
              (endpointMap as Record<
                string,
                { path?: string; method?: string }
              >) || {}
            }
            priceRate={priceRate ?? 1}
            usdExchangeRate={billingUSDToCNYRate ?? 1}
            tokenUnit={tokenUnit}
            showRechargePrice={showPricesInCny}
          />
        )}
      </PageTransition>
    </PublicLayout>
  )
}
