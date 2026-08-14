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
import { getModuleAccessFromStatus } from '@/lib/nav-modules'

import { IkunHome } from './ikun-home'

interface DefaultHomeProps {
  isAuthenticated: boolean
}

export function DefaultHome(props: DefaultHomeProps) {
  const { status } = useStatus()
  const pricingAccess = getModuleAccessFromStatus(
    status as Record<string, unknown> | null,
    'pricing'
  )
  const catalogAvailable =
    pricingAccess.enabled &&
    (!pricingAccess.requireAuth || props.isAuthenticated)

  return (
    <IkunHome
      catalogAvailable={catalogAvailable}
      isAuthenticated={props.isAuthenticated}
    />
  )
}
