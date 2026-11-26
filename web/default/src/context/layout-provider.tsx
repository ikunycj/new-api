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
import { createContext, useContext } from 'react'

import { useThemeCustomization } from '@/context/theme-customization-provider'
import type { SidebarLayout, SidebarStyle } from '@/lib/theme-customization'

export type Collapsible = 'offcanvas' | 'icon'
export type Variant = SidebarStyle

type SidebarLayoutSettings = {
  collapsible: Collapsible
  defaultOpen: boolean
}

// eslint-disable-next-line react-refresh/only-export-components
export function getSidebarLayoutSettings(
  layout: SidebarLayout
): SidebarLayoutSettings {
  if (layout === 'offcanvas') {
    return { collapsible: 'offcanvas', defaultOpen: false }
  }

  return { collapsible: 'icon', defaultOpen: layout === 'default' }
}

type LayoutContextType = {
  collapsible: Collapsible
  variant: Variant
}

const LayoutContext = createContext<LayoutContextType | null>(null)

type LayoutProviderProps = {
  children: React.ReactNode
}

export function LayoutProvider(props: LayoutProviderProps) {
  const { customization } = useThemeCustomization()
  const layoutSettings = getSidebarLayoutSettings(customization.layout)

  return (
    <LayoutContext.Provider
      value={{
        collapsible: layoutSettings.collapsible,
        variant: customization.sidebar,
      }}
    >
      {props.children}
    </LayoutContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useLayout() {
  const context = useContext(LayoutContext)
  if (!context) {
    throw new Error('useLayout must be used within a LayoutProvider')
  }
  return context
}
