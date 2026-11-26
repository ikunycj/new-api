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
import { createContext, useContext, useEffect, useMemo, useState } from 'react'

import type { ThemeMode } from '@/lib/theme-customization'

type ResolvedTheme = Exclude<ThemeMode, 'system'>

type ThemeProviderState = {
  theme: ThemeMode
  resolvedTheme: ResolvedTheme
}

function getInitialResolvedTheme(mode: ThemeMode): ResolvedTheme {
  if (typeof document === 'undefined') {
    return mode === 'dark' ? 'dark' : 'light'
  }
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const ThemeContext = createContext<ThemeProviderState>({
  theme: 'system',
  resolvedTheme: 'light',
})

export function ThemeProvider(props: {
  children: React.ReactNode
  mode: ThemeMode
}) {
  const [resolvedTheme, setResolvedTheme] = useState(() =>
    getInitialResolvedTheme(props.mode)
  )

  useEffect(() => {
    const root = document.documentElement
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const applyTheme = () => {
      let nextResolvedTheme: ResolvedTheme
      if (props.mode === 'system') {
        nextResolvedTheme = mediaQuery.matches ? 'dark' : 'light'
      } else {
        nextResolvedTheme = props.mode
      }
      root.dataset.themeMode = props.mode
      root.classList.remove('light', 'dark')
      root.classList.add(nextResolvedTheme)
      setResolvedTheme(nextResolvedTheme)

      const metaThemeColor = document.querySelector("meta[name='theme-color']")
      metaThemeColor?.setAttribute(
        'content',
        nextResolvedTheme === 'dark' ? '#020817' : '#fff'
      )
    }

    applyTheme()
    if (props.mode !== 'system') return

    mediaQuery.addEventListener('change', applyTheme)
    return () => mediaQuery.removeEventListener('change', applyTheme)
  }, [props.mode])

  const contextValue = useMemo(
    () => ({ theme: props.mode, resolvedTheme }),
    [props.mode, resolvedTheme]
  )

  return <ThemeContext value={contextValue}>{props.children}</ThemeContext>
}

// eslint-disable-next-line react-refresh/only-export-components
export const useTheme = () => useContext(ThemeContext)
