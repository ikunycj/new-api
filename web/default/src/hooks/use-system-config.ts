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
import { useEffect } from 'react'

import {
  DEFAULT_SYSTEM_NAME,
  DEFAULT_LOGO,
  LEGACY_DEFAULT_LOGO,
} from '@/lib/constants'
import { applyFaviconToDom } from '@/lib/dom-utils'
import { setPreferredModelOrder } from '@/lib/model-preferences'
import {
  useSystemConfigStore,
  type CurrencyConfig,
  type CurrencyDisplayType,
  type SystemConfig,
  DEFAULT_CURRENCY_CONFIG,
} from '@/stores/system-config-store'

interface StatusApiResponse {
  success: boolean
  data: {
    system_name?: string
    logo?: string
    display_token_stat_enabled?: boolean
    display_in_currency?: boolean
    quota_display_type?: CurrencyDisplayType
    quota_per_unit?: number
    usd_exchange_rate?: number
    custom_currency_symbol?: string
    custom_currency_exchange_rate?: number
    preferred_models?: string[]
  }
}

function toNumber(value: unknown, fallback: number): number {
  if (typeof value === 'number' && !Number.isNaN(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (!Number.isNaN(parsed)) return parsed
  }
  return fallback
}

/**
 * Map `/api/status` response data to our persisted system config structure
 */
export function mapStatusDataToConfig(
  data: StatusApiResponse['data'] | undefined
): Partial<SystemConfig> {
  if (!data) return {}

  setPreferredModelOrder(data.preferred_models)

  const has = (key: keyof StatusApiResponse['data']) => Object.hasOwn(data, key)
  const nextConfig: Partial<SystemConfig> = {}

  if (has('system_name')) {
    nextConfig.systemName = data.system_name || DEFAULT_SYSTEM_NAME
  }
  if (has('logo')) {
    nextConfig.logo =
      !data.logo || data.logo === LEGACY_DEFAULT_LOGO ? DEFAULT_LOGO : data.logo
  }
  if (has('display_token_stat_enabled')) {
    nextConfig.displayTokenStatEnabled = data.display_token_stat_enabled
  }
  if (has('preferred_models')) {
    nextConfig.preferredModels = data.preferred_models ?? []
  }

  const currencyFields: (keyof StatusApiResponse['data'])[] = [
    'display_in_currency',
    'quota_display_type',
    'quota_per_unit',
    'usd_exchange_rate',
    'custom_currency_symbol',
    'custom_currency_exchange_rate',
  ]
  if (currencyFields.some((key) => has(key))) {
    const currency: CurrencyConfig = {
      displayInCurrency:
        data.display_in_currency ?? DEFAULT_CURRENCY_CONFIG.displayInCurrency,
      quotaDisplayType:
        (data.quota_display_type as CurrencyDisplayType | undefined) ??
        DEFAULT_CURRENCY_CONFIG.quotaDisplayType,
      quotaPerUnit: toNumber(
        data.quota_per_unit,
        DEFAULT_CURRENCY_CONFIG.quotaPerUnit
      ),
      usdExchangeRate: toNumber(
        data.usd_exchange_rate,
        DEFAULT_CURRENCY_CONFIG.usdExchangeRate
      ),
      customCurrencySymbol:
        data.custom_currency_symbol?.trim() ||
        DEFAULT_CURRENCY_CONFIG.customCurrencySymbol,
      customCurrencyExchangeRate: toNumber(
        data.custom_currency_exchange_rate,
        DEFAULT_CURRENCY_CONFIG.customCurrencyExchangeRate
      ),
    }
    nextConfig.currency = currency
  }

  return nextConfig
}

// Preload image and return cleanup function
function preloadImage(
  src: string,
  onLoad: () => void,
  onError: () => void
): () => void {
  const img = new Image()
  img.onload = onLoad
  img.onerror = onError
  img.src = src

  return () => {
    img.onload = null
    img.onerror = null
  }
}

/**
 * System configuration hook with auto-loading and logo preloading
 *
 * @example
 * // Other components - use cached config
 * const { systemName, logo, loading } = useSystemConfig()
 */
export function useSystemConfig() {
  const { config, loading, loadedLogoUrl, setLoadedLogoUrl } =
    useSystemConfigStore()

  useEffect(() => {
    document.title = config.systemName
    const metaTitle =
      document.querySelector<HTMLMetaElement>('meta[name="title"]')
    metaTitle?.setAttribute('content', config.systemName)
  }, [config.systemName])

  // Preload logo image when URL changes
  useEffect(() => {
    const { logo } = config

    // Skip if logo is already loaded
    if (!logo || logo === loadedLogoUrl) return

    // Preload new logo
    return preloadImage(
      logo,
      () => {
        setLoadedLogoUrl(logo)
        if (logo !== DEFAULT_LOGO) applyFaviconToDom(logo)
      },
      () => {
        if (logo !== DEFAULT_LOGO) {
          // eslint-disable-next-line no-console
          console.error('Failed to load logo:', logo)
        }
        // Mark as loaded even on error to prevent infinite retry
        setLoadedLogoUrl(logo)
      }
    )
  }, [config.logo, loadedLogoUrl, setLoadedLogoUrl])

  return {
    ...config,
    loading,
    logoLoaded: config.logo === loadedLogoUrl && !!loadedLogoUrl,
  }
}
