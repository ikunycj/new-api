/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getPricingGroups } from '@/features/channels/api'
import { getUserGroups } from '@/lib/api'

function labelsFromUserGroups(
  groups: Awaited<ReturnType<typeof getUserGroups>>['data']
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(groups ?? {}).map(([group, info]) => [
      group,
      info.desc || group,
    ])
  )
}

/** Resolve display labels while retaining pricing-group keys for all requests. */
export function usePricingGroupLabels(
  adminView: boolean
): Record<string, string> {
  const adminQuery = useQuery({
    queryKey: ['pricing-groups'],
    queryFn: getPricingGroups,
    enabled: adminView,
    staleTime: 0,
  })
  const userQuery = useQuery({
    queryKey: ['user-groups'],
    queryFn: getUserGroups,
    enabled: !adminView,
    staleTime: 0,
  })

  return useMemo(() => {
    if (adminView) return adminQuery.data?.display_names ?? {}
    return labelsFromUserGroups(userQuery.data?.data)
  }, [adminQuery.data?.display_names, adminView, userQuery.data?.data])
}
