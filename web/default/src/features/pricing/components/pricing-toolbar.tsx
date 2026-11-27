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
import { Radio } from '@base-ui/react/radio'
import {
  ArrowDownAZ,
  ArrowDownNarrowWide,
  ArrowDownWideNarrow,
  Bot,
  Code,
  Grid2X2,
  MessageSquare,
  Star,
  Table2,
} from 'lucide-react'
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import { RadioGroup } from '@/components/ui/radio-group'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import {
  PRICING_CURRENCIES,
  SORT_OPTIONS,
  VIEW_MODES,
  type PricingCurrency,
  type SortOption,
  type ViewMode,
} from '../constants'
import { formatGroupRatio, getConfiguredGroupRatio } from '../lib/model-helpers'
import type { PricingModel, PricingVendor, TokenUnit } from '../types'
import { SearchBar } from './search-bar'

type SegmentOption = {
  value: string
  label?: string
  icon?: React.ComponentType<{ className?: string }>
  tooltip?: string
}

export interface PricingToolbarProps {
  searchInput: string
  onSearchChange: (value: string) => void
  onClearSearch: () => void
  sortingRule: SortOption
  onSortingRuleChange: (value: SortOption) => void
  tokenUnit: TokenUnit
  onTokenUnitChange: (value: TokenUnit) => void
  displayCurrency: PricingCurrency
  onCurrencyChange: (value: PricingCurrency) => void
  viewMode: ViewMode
  onViewModeChange: (value: ViewMode) => void
  vendorFilters: string[]
  groupFilters: string[]
  onVendorToggle: (value: string) => void
  onGroupToggle: (value: string) => void
  vendors: PricingVendor[]
  groups: string[]
  groupLabels: Record<string, string>
  groupRatios: Record<string, number>
  models: PricingModel[]
}

function SegmentedControl(props: {
  options: SegmentOption[]
  value: string
  onChange: (value: string) => void
  ariaLabel: string
}) {
  return (
    <div
      role='group'
      aria-label={props.ariaLabel}
      className='bg-muted/60 inline-flex h-8 shrink-0 items-center rounded-lg border p-0.5'
    >
      {props.options.map((option) => {
        const Icon = option.icon
        const isActive = option.value === props.value
        const button = (
          <button
            key={option.value}
            type='button'
            onClick={() => props.onChange(option.value)}
            aria-pressed={isActive}
            className={cn(
              'inline-flex h-full items-center justify-center rounded-md text-xs font-medium transition-all',
              Icon && !option.label ? 'w-7' : 'gap-1.5 px-3',
              isActive
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {Icon && <Icon className='size-3.5' />}
            {option.label}
          </button>
        )

        if (!option.tooltip) {
          return button
        }

        return (
          <Tooltip key={option.value}>
            <TooltipTrigger render={button} />
            <TooltipContent side='bottom' className='text-xs'>
              {option.tooltip}
            </TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}

export function PricingToolbar(props: PricingToolbarProps) {
  const { t } = useTranslation()
  const sortingRulesLabelId = useId()
  const sortingOptions = [
    { value: SORT_OPTIONS.BEST_VALUE, label: t('Best Value'), icon: Star },
    { value: SORT_OPTIONS.AGENT, label: t('Agent capability'), icon: Bot },
    { value: SORT_OPTIONS.CODING, label: t('Coding capability'), icon: Code },
    {
      value: SORT_OPTIONS.CHAT,
      label: t('Chat capability'),
      icon: MessageSquare,
    },
    {
      value: SORT_OPTIONS.PRICE_LOW,
      label: t('Price: Low to High'),
      icon: ArrowDownNarrowWide,
    },
    {
      value: SORT_OPTIONS.PRICE_HIGH,
      label: t('Price: High to Low'),
      icon: ArrowDownWideNarrow,
    },
    { value: SORT_OPTIONS.NAME, label: t('Name'), icon: ArrowDownAZ },
  ] as const

  const handleTokenUnitChange = (value: string) => {
    props.onTokenUnitChange(value as TokenUnit)
  }

  const handleViewModeChange = (value: string) => {
    props.onViewModeChange(value as ViewMode)
  }

  const handleCurrencyChange = (value: string) => {
    props.onCurrencyChange(value as PricingCurrency)
  }

  return (
    <section className='flex flex-col gap-3'>
      <div className='flex flex-col gap-3'>
        <section className='space-y-2'>
          <h2
            id={sortingRulesLabelId}
            className='text-muted-foreground text-xs font-semibold'
          >
            {t('Sorting rules')}
          </h2>
          <RadioGroup
            aria-labelledby={sortingRulesLabelId}
            value={props.sortingRule}
            onValueChange={(value) => {
              const option = sortingOptions.find(
                (option) => option.value === value
              )
              if (option) props.onSortingRuleChange(option.value)
            }}
            className='flex flex-wrap'
          >
            {sortingOptions.map(({ value, label, icon: Icon }) => (
              <Radio.Root
                key={value}
                value={value}
                className='group border-border/70 bg-background text-muted-foreground hover:bg-muted/60 hover:text-foreground data-checked:border-primary/50 data-checked:bg-primary/10 data-checked:text-foreground focus-visible:ring-ring/50 relative inline-flex min-h-9 items-center justify-center gap-2 rounded-md border px-3 py-1.5 text-sm font-medium transition-colors outline-none focus-visible:ring-2 motion-reduce:transition-none pointer-coarse:after:absolute pointer-coarse:after:inset-x-0 pointer-coarse:after:-inset-y-1'
              >
                <Icon
                  aria-hidden='true'
                  className='group-data-checked:text-primary size-4 shrink-0'
                />
                {label}
              </Radio.Root>
            ))}
          </RadioGroup>
        </section>

        <section aria-label={t('Provider')} className='space-y-2'>
          <h2 className='text-muted-foreground text-xs font-semibold'>
            {t('Provider')}
          </h2>
          <div className='flex flex-wrap gap-2' role='group'>
            {props.vendors.map((vendor) => {
              const selected = props.vendorFilters.includes(vendor.name)
              const count = props.models.filter(
                (model) => model.vendor_name === vendor.name
              ).length
              if (count === 0) return null
              return (
                <button
                  key={vendor.id}
                  type='button'
                  aria-pressed={selected}
                  onClick={() => props.onVendorToggle(vendor.name)}
                  className={cn(
                    'inline-flex min-h-9 items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors',
                    selected
                      ? 'border-primary/50 bg-primary/10 text-foreground'
                      : 'border-border/70 bg-background text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                  )}
                >
                  {vendor.icon && (
                    <span className='shrink-0'>
                      {getLobeIcon(vendor.icon, 18)}
                    </span>
                  )}
                  <span className='max-w-40 truncate'>{vendor.name}</span>
                  <span className='text-xs tabular-nums opacity-70'>
                    {count}
                  </span>
                </button>
              )
            })}
          </div>
        </section>

        <section aria-label={t('Pricing group')} className='space-y-2'>
          <h2 className='text-muted-foreground text-xs font-semibold'>
            {t('Pricing group')}
          </h2>
          <div className='flex flex-wrap gap-2' role='group'>
            {props.groups.map((group) => {
              const selected = props.groupFilters.includes(group)
              const count = props.models.filter((model) =>
                model.enable_groups?.includes(group)
              ).length
              const ratio = formatGroupRatio(
                getConfiguredGroupRatio(props.groupRatios, group)
              )
              return (
                <button
                  key={group}
                  type='button'
                  aria-pressed={selected}
                  onClick={() => props.onGroupToggle(group)}
                  className={cn(
                    'inline-flex min-h-9 items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors',
                    selected
                      ? 'border-primary/50 bg-primary/10 text-foreground'
                      : 'border-border/70 bg-background text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                  )}
                >
                  <span className='max-w-48 truncate'>
                    {props.groupLabels[group] || group}
                  </span>
                  {ratio && (
                    <span className='bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-xs tabular-nums'>
                      {ratio}
                    </span>
                  )}
                  <span className='text-xs tabular-nums opacity-70'>
                    {count}
                  </span>
                </button>
              )
            })}
          </div>
        </section>
      </div>

      <div className='border-border/70 flex flex-wrap items-center gap-2 border-t pt-3'>
        <div className='flex min-w-0 grow basis-72 items-center justify-between gap-2'>
          <SearchBar
            value={props.searchInput}
            onChange={props.onSearchChange}
            onClear={props.onClearSearch}
            className='w-full max-w-56 min-w-0'
          />
          <SegmentedControl
            options={[
              { value: PRICING_CURRENCIES.CNY, label: '¥ CNY' },
              { value: PRICING_CURRENCIES.USD, label: '$ USD' },
            ]}
            value={props.displayCurrency}
            onChange={handleCurrencyChange}
            ariaLabel={t('Currency')}
          />
        </div>

        <div className='flex flex-wrap items-center gap-2'>
          <SegmentedControl
            options={[
              { value: 'M', label: '/1M' },
              { value: 'K', label: '/1K' },
            ]}
            value={props.tokenUnit}
            onChange={handleTokenUnitChange}
            ariaLabel={t('Token unit')}
          />

          <SegmentedControl
            options={[
              {
                value: VIEW_MODES.CARD,
                icon: Grid2X2,
                tooltip: t('Card view'),
              },
              {
                value: VIEW_MODES.TABLE,
                icon: Table2,
                tooltip: t('Table view'),
              },
            ]}
            value={props.viewMode}
            onChange={handleViewModeChange}
            ariaLabel={t('View mode')}
          />
        </div>
      </div>
    </section>
  )
}
