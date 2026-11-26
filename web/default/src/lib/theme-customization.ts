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
/**
 * Theme customization constants and types.
 *
 * Lives in `lib/` (not `context/`) so it can be imported alongside the
 * provider without breaking React Fast Refresh boundaries.
 */

export const THEME_PRESETS = [
  {
    value: 'default',
    name: 'Default',
    swatches: ['oklch(0.72 0.18 250)', 'oklch(0.7 0.12 280)'],
  },
  {
    // Inspired by Anthropic's official brand language: warm cream canvas
    // (#faf9f5) paired with clay/coral (#d97757) as the single accent.
    // Swatches preview the canvas → accent gradient that defines the system.
    value: 'anthropic',
    name: 'Anthropic',
    swatches: ['oklch(0.984 0.005 95)', 'oklch(0.685 0.142 38)'],
  },
  {
    value: 'simple-large',
    name: 'Simple Large-font',
    swatches: ['oklch(0.15 0 0)', 'oklch(0.99 0 0)'],
  },
  {
    value: 'underground',
    name: 'Underground',
    swatches: ['oklch(0.5315 0.0694 156.19)', 'oklch(0.5748 0.0862 336.52)'],
  },
  {
    value: 'rose-garden',
    name: 'Rose Garden',
    swatches: ['oklch(0.5827 0.2418 12.23)', 'oklch(0.8131 0.1129 5.67)'],
  },
  {
    value: 'lake-view',
    name: 'Lake View',
    swatches: ['oklch(0.765 0.177 163.22)', 'oklch(0.551 0.0899 200.52)'],
  },
  {
    value: 'sunset-glow',
    name: 'Sunset Glow',
    swatches: ['oklch(0.5591 0.1882 25.33)', 'oklch(0.7938 0.1248 42.42)'],
  },
  {
    value: 'forest-whisper',
    name: 'Forest Whisper',
    swatches: ['oklch(0.5276 0.1072 182.22)', 'oklch(0.5236 0.0505 250.18)'],
  },
  {
    value: 'ocean-breeze',
    name: 'Ocean Breeze',
    swatches: ['oklch(0.5461 0.2152 262.88)', 'oklch(0.5854 0.2041 277.12)'],
  },
  {
    value: 'lavender-dream',
    name: 'Lavender Dream',
    swatches: ['oklch(0.5709 0.1808 306.89)', 'oklch(0.811 0.0589 201.14)'],
  },
  {
    value: 'ikun-love',
    name: 'IKUN LOVE',
    swatches: ['oklch(0.6652 0.1968 37.66)', 'oklch(0.7293 0.1063 215.98)'],
  },
] as const

export type ThemePreset = (typeof THEME_PRESETS)[number]['value']
export type ThemeRadius = 'default' | 'none' | 'sm' | 'md' | 'lg' | 'xl'
export type ThemeScale = 'default' | 'sm' | 'lg' | 'xl'
export type ContentLayout = 'full' | 'centered'
export type ThemeMode = 'system' | 'light' | 'dark'
export type SidebarStyle = 'inset' | 'floating' | 'sidebar'
export type SidebarLayout = 'default' | 'icon' | 'offcanvas'
export type Direction = 'ltr' | 'rtl'

/**
 * Font axis for the theme.
 *
 * - `default` — resolve at runtime from the active preset
 *   (see `PRESET_DEFAULT_FONT`). Anthropic resolves to serif. Other
 *   presets fall back to sans. Mirrors how
 *   `radius: 'default'` defers to a per-preset hint.
 * - `sans` — humanist sans (Public Sans), the project's UI fallback.
 * - `serif` — editorial serif (Lora + CJK fallbacks), the project's
 *   "soul" typography. Inherits across the whole UI; monospace contexts
 *   keep their own family via Tailwind preflight and `.font-mono`.
 */
export type ThemeFont = 'default' | 'sans' | 'serif'

/**
 * The resolved (non-`default`) font value applied to the DOM. The provider
 * always sets `data-theme-font` to one of these concrete values so CSS only
 * needs simple attribute selectors (no `:not()` gymnastics, no per-preset
 * font branches).
 */
export type ResolvedThemeFont = Exclude<ThemeFont, 'default'>

export type ThemeCustomization = {
  mode: ThemeMode
  preset: ThemePreset
  font: ThemeFont
  radius: ThemeRadius
  scale: ThemeScale
  contentLayout: ContentLayout
  sidebar: SidebarStyle
  layout: SidebarLayout
  direction: Direction
}

export const DEFAULT_THEME_CUSTOMIZATION: ThemeCustomization = {
  mode: 'system',
  preset: 'default',
  font: 'default',
  radius: 'default',
  scale: 'default',
  contentLayout: 'full',
  sidebar: 'inset',
  layout: 'default',
  direction: 'ltr',
}

/**
 * Preset-specific defaults. Keep these aligned with the server's first-paint
 * theme attributes.
 */
export const PRESET_DEFAULT_FONT: Partial<
  Record<ThemePreset, ResolvedThemeFont>
> = {
  anthropic: 'serif',
}

/**
 * Resolve the site font and active preset into the concrete font that
 * should drive the DOM. Pure function so it's safe to call inside both the
 * effect that applies the attribute and the UI preview that hints at what
 * `default` will render as.
 */
export function resolveThemeFont(
  font: ThemeFont,
  preset: ThemePreset
): ResolvedThemeFont {
  if (font === 'default') {
    return PRESET_DEFAULT_FONT[preset] ?? 'sans'
  }
  return font
}
