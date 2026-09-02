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
import { mapStatusDataToConfig } from '@/hooks/use-system-config'
import {
  useSystemConfigStore,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

export interface PublicStatus {
  [key: string]: unknown
  custom_currency_exchange_rate?: number
  custom_currency_symbol?: string
  display_in_currency?: boolean
  display_token_stat_enabled?: boolean
  logo?: string
  customer_service?: string
  quota_display_type?: CurrencyDisplayType
  quota_per_unit?: number
  server_address?: string
  system_name?: string
  usd_exchange_rate?: number
}

export interface PublicBootstrap {
  docs?: {
    file_name: string
    route: string
  }
  home_page_content?: string
  home_page_content_loaded?: boolean
  locale: string
  setup: boolean
  status: PublicStatus
}

declare global {
  interface Window {
    __NEW_API_PUBLIC_BOOTSTRAP__?: PublicBootstrap
  }
}

let prerenderBootstrap: PublicBootstrap | undefined

export function setPrerenderBootstrap(bootstrap: PublicBootstrap): void {
  prerenderBootstrap = bootstrap
}

export function getPublicBootstrap(): PublicBootstrap | undefined {
  if (typeof window !== 'undefined') {
    return window.__NEW_API_PUBLIC_BOOTSTRAP__
  }
  return prerenderBootstrap
}

export function initializePublicBootstrap(): void {
  const bootstrap = getPublicBootstrap()
  if (!bootstrap?.status) return

  const store = useSystemConfigStore.getState()
  store.setConfig(mapStatusDataToConfig(bootstrap.status))
  store.setLoading(false)
  if (bootstrap.status.logo) {
    store.setLoadedLogoUrl(bootstrap.status.logo)
  }
}
