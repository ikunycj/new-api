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
import { useStatus } from '@/hooks/use-status'

export const DOCS_PRODUCTION_BASE_URL = 'https://alltokenapi.com'

export function resolveDocsBaseUrl(configuredAddress?: unknown): string {
  if (import.meta.env.PROD) {
    return DOCS_PRODUCTION_BASE_URL
  }

  if (
    typeof configuredAddress === 'string' &&
    configuredAddress.trim().length > 0
  ) {
    return configuredAddress.trim().replace(/\/+$/, '')
  }

  return typeof window === 'undefined' ? '' : window.location.origin
}

export function useDocsBaseUrl(): string {
  const { status } = useStatus()
  return resolveDocsBaseUrl(status?.server_address)
}
