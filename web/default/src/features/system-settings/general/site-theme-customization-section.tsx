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
import type { ThemeCustomization } from '@/lib/theme-customization'

import {
  ContentLayoutConfig,
  DirConfig,
  FontConfig,
  LayoutConfig,
  ModeConfig,
  PresetConfig,
  RadiusConfig,
  ScaleConfig,
  SidebarConfig,
} from './site-theme-controls'

type SiteThemeCustomizationSectionProps = {
  value: ThemeCustomization
  defaults: ThemeCustomization
  disabled: boolean
  onChange: <K extends keyof ThemeCustomization>(
    key: K,
    value: ThemeCustomization[K]
  ) => void
}

export function SiteThemeCustomizationSection(
  props: SiteThemeCustomizationSectionProps
) {
  return (
    <section
      aria-labelledby='site-theme-settings-heading'
      className='flex flex-col gap-4'
    >
      <h3 id='site-theme-settings-heading' className='text-base font-semibold'>
        主题设置
      </h3>
      <p className='text-muted-foreground text-sm'>保存后对所有用户生效。</p>
      <fieldset
        className='flex min-w-0 flex-col gap-6'
        disabled={props.disabled}
      >
        <ModeConfig
          value={props.value.mode}
          defaultValue={props.defaults.mode}
          disabled={props.disabled}
          onChange={(value) => props.onChange('mode', value)}
        />
        <PresetConfig
          value={props.value.preset}
          defaultValue={props.defaults.preset}
          disabled={props.disabled}
          onChange={(value) => props.onChange('preset', value)}
        />
        <FontConfig
          value={props.value.font}
          defaultValue={props.defaults.font}
          preset={props.value.preset}
          disabled={props.disabled}
          onChange={(value) => props.onChange('font', value)}
        />
        <RadiusConfig
          value={props.value.radius}
          defaultValue={props.defaults.radius}
          disabled={props.disabled}
          onChange={(value) => props.onChange('radius', value)}
        />
        <ScaleConfig
          value={props.value.scale}
          defaultValue={props.defaults.scale}
          disabled={props.disabled}
          onChange={(value) => props.onChange('scale', value)}
        />
        <ContentLayoutConfig
          value={props.value.contentLayout}
          defaultValue={props.defaults.contentLayout}
          disabled={props.disabled}
          onChange={(value) => props.onChange('contentLayout', value)}
        />
        <SidebarConfig
          value={props.value.sidebar}
          defaultValue={props.defaults.sidebar}
          disabled={props.disabled}
          onChange={(value) => props.onChange('sidebar', value)}
        />
        <LayoutConfig
          value={props.value.layout}
          defaultValue={props.defaults.layout}
          disabled={props.disabled}
          onChange={(value) => props.onChange('layout', value)}
        />
        <DirConfig
          value={props.value.direction}
          defaultValue={props.defaults.direction}
          disabled={props.disabled}
          onChange={(value) => props.onChange('direction', value)}
        />
      </fieldset>
    </section>
  )
}
