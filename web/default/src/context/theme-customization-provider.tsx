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
import { createContext, useContext, useEffect, useMemo } from 'react'

import {
  DEFAULT_THEME_CUSTOMIZATION,
  resolveThemeFont,
  type ThemeCustomization,
} from '@/lib/theme-customization'

type ThemeCustomizationContextType = {
  defaults: ThemeCustomization
  customization: ThemeCustomization
}

const ThemeCustomizationContext = createContext<ThemeCustomizationContextType>({
  defaults: DEFAULT_THEME_CUSTOMIZATION,
  customization: DEFAULT_THEME_CUSTOMIZATION,
})

export function ThemeCustomizationProvider(props: {
  children: React.ReactNode
  customization: ThemeCustomization
}) {
  const { customization } = props
  const { preset, font, radius, scale, contentLayout } = customization

  useEffect(() => {
    const attributes = {
      'data-theme-preset': preset === 'default' ? null : preset,
      'data-theme-font': resolveThemeFont(font, preset),
      'data-theme-radius': radius === 'default' ? null : radius,
      'data-theme-scale': scale === 'default' ? null : scale,
      'data-theme-content-layout': contentLayout,
    }

    for (const [name, value] of Object.entries(attributes)) {
      if (value === null) {
        document.body.removeAttribute(name)
      } else {
        document.body.setAttribute(name, value)
      }
    }
  }, [preset, font, radius, scale, contentLayout])

  const value = useMemo(
    () => ({ defaults: DEFAULT_THEME_CUSTOMIZATION, customization }),
    [customization]
  )

  return (
    <ThemeCustomizationContext.Provider value={value}>
      {props.children}
    </ThemeCustomizationContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useThemeCustomization() {
  return useContext(ThemeCustomizationContext)
}
