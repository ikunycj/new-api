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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import type { SystemStatus } from '@/features/auth/types'
import { getStatus } from '@/lib/api'
import { getPublicBootstrap } from '@/lib/public-bootstrap'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { useHydrated } from './use-hydrated'
import { mapStatusDataToConfig } from './use-system-config'

// Get initial cache from localStorage
function getInitialStatus(): SystemStatus | undefined {
  const bootstrapStatus = getPublicBootstrap()?.status
  if (bootstrapStatus) return bootstrapStatus

  try {
    if (typeof window !== 'undefined') {
      const saved = window.localStorage.getItem('status')
      return saved ? (JSON.parse(saved) as SystemStatus) : undefined
    }
  } catch {
    /* empty */
  }
  return undefined
}

export function useStatus() {
  const hydrated = useHydrated()
  const hasBootstrapStatus = !!getPublicBootstrap()?.status
  const [networkEnabled, setNetworkEnabled] = useState(!hasBootstrapStatus)
  const setLoading = useSystemConfigStore((state) => state.setLoading)
  const { data, isLoading, error } = useQuery({
    queryKey: ['status'],
    queryFn: async () => {
      const status = await getStatus()
      try {
        if (status) {
          const { setConfig } = useSystemConfigStore.getState()
          setConfig(mapStatusDataToConfig(status))
        }
      } catch (err) {
        if (import.meta.env.DEV) {
          // eslint-disable-next-line no-console
          console.warn(
            '[useStatus] Failed to sync status to system config',
            err
          )
        }
      }
      // Save to localStorage
      try {
        if (typeof window !== 'undefined' && status) {
          window.localStorage.setItem('status', JSON.stringify(status))
        }
      } catch {
        /* empty */
      }
      return status as SystemStatus | null
    },
    // SSR/bootstrap data is authoritative for first paint. Refresh it only
    // after the page is interactive so public APIs never compete with LCP.
    initialData: getInitialStatus(),
    initialDataUpdatedAt: 0,
    enabled: networkEnabled,
    // Data becomes stale after 5 minutes
    staleTime: 5 * 60 * 1000,
    // Cache expires after 30 minutes
    gcTime: 30 * 60 * 1000,
  })

  useEffect(() => {
    if (!hasBootstrapStatus || networkEnabled) return

    const enableNetwork = () => setNetworkEnabled(true)
    const timeoutId = window.setTimeout(enableNetwork, 5000)
    window.addEventListener('pointerdown', enableNetwork, {
      once: true,
      passive: true,
    })
    window.addEventListener('keydown', enableNetwork, { once: true })

    return () => {
      window.clearTimeout(timeoutId)
      window.removeEventListener('pointerdown', enableNetwork)
      window.removeEventListener('keydown', enableNetwork)
    }
  }, [hasBootstrapStatus, networkEnabled])

  useEffect(() => {
    setLoading(isLoading)
  }, [isLoading, setLoading])

  let status = data ?? null
  if (!hydrated && status) {
    status = { ...status, HeaderNavModules: '' } as SystemStatus
  }

  return {
    status,
    loading: isLoading,
    error,
  }
}
