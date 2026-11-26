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
import { Radio as RadioPrimitive } from '@base-ui/react/radio'
import { RadioGroup as Radio } from '@base-ui/react/radio-group'
import { CircleCheck, RotateCcw } from 'lucide-react'
import type { SVGProps } from 'react'
import { useTranslation } from 'react-i18next'

import { IconDir } from '@/assets/custom/icon-dir'
import { IconLayoutCompact } from '@/assets/custom/icon-layout-compact'
import { IconLayoutDefault } from '@/assets/custom/icon-layout-default'
import { IconLayoutFull } from '@/assets/custom/icon-layout-full'
import { IconSidebarFloating } from '@/assets/custom/icon-sidebar-floating'
import { IconSidebarInset } from '@/assets/custom/icon-sidebar-inset'
import { IconSidebarSidebar } from '@/assets/custom/icon-sidebar-sidebar'
import { IconThemeDark } from '@/assets/custom/icon-theme-dark'
import { IconThemeLight } from '@/assets/custom/icon-theme-light'
import { IconThemeSystem } from '@/assets/custom/icon-theme-system'
import { Button } from '@/components/ui/button'
import {
  type ContentLayout,
  type Direction,
  resolveThemeFont,
  type SidebarLayout,
  type SidebarStyle,
  THEME_PRESETS,
  type ThemeFont,
  type ThemeMode,
  type ThemePreset,
  type ThemeRadius,
  type ThemeScale,
} from '@/lib/theme-customization'
import { cn } from '@/lib/utils'

const Item = RadioPrimitive.Root

function SectionTitle(props: {
  title: string
  showReset?: boolean
  onReset?: () => void
  className?: string
}) {
  const { t } = useTranslation()

  return (
    <div
      className={cn(
        'text-muted-foreground mb-2 flex items-center gap-2 text-sm font-semibold',
        props.className
      )}
    >
      {props.title}
      {props.showReset && props.onReset && (
        <Button
          type='button'
          size='icon'
          variant='secondary'
          className='size-4'
          onClick={props.onReset}
          aria-label={t('Reset')}
        >
          <RotateCcw className='size-3' aria-hidden='true' />
        </Button>
      )}
    </div>
  )
}

type IllustratedOption<T extends string> = {
  value: T
  label: string
  icon: (props: SVGProps<SVGSVGElement>) => React.ReactElement
}

function RadioGroupItem(props: {
  item: IllustratedOption<string>
  preserveIconColors?: boolean
}) {
  return (
    <Item
      value={props.item.value}
      className={cn('group outline-none', 'transition duration-200 ease-in')}
      aria-label={props.item.label}
    >
      <div
        className={cn(
          'ring-border relative rounded-md ring-[1px]',
          'group-data-checked:ring-primary group-data-checked:shadow-2xl',
          'group-focus-visible:ring-2'
        )}
        aria-hidden='true'
      >
        <CircleCheck
          className={cn(
            'fill-primary size-6 stroke-white',
            'group-data-unchecked:hidden',
            'absolute top-0 right-0 translate-x-1/2 -translate-y-1/2'
          )}
          aria-hidden='true'
        />
        <props.item.icon
          className={cn(
            'w-full',
            !props.preserveIconColors &&
              'stroke-primary fill-primary group-data-unchecked:stroke-muted-foreground group-data-unchecked:fill-muted-foreground'
          )}
          aria-hidden='true'
        />
      </div>
      <div className='mt-1 text-xs'>{props.item.label}</div>
    </Item>
  )
}

type SiteThemeFieldProps<T extends string> = {
  value: T
  defaultValue: T
  disabled: boolean
  onChange: (value: T) => void
}

function IllustratedThemeConfig<T extends string>(
  props: SiteThemeFieldProps<T> & {
    title: string
    options: IllustratedOption<T>[]
    preserveIconColors?: boolean
    className?: string
  }
) {
  return (
    <div>
      <SectionTitle
        title={props.title}
        showReset={props.value !== props.defaultValue}
        onReset={() => props.onChange(props.defaultValue)}
      />
      <Radio<T>
        value={props.value}
        disabled={props.disabled}
        onValueChange={props.onChange}
        className={cn(
          'grid w-full max-w-md grid-cols-3 gap-4',
          props.className
        )}
        aria-label={props.title}
      >
        {props.options.map((item) => (
          <RadioGroupItem
            key={item.value}
            item={item}
            preserveIconColors={props.preserveIconColors}
          />
        ))}
      </Radio>
    </div>
  )
}

export function ModeConfig(props: SiteThemeFieldProps<ThemeMode>) {
  return (
    <IllustratedThemeConfig
      {...props}
      title='明暗模式'
      preserveIconColors
      options={[
        { value: 'system', label: '跟随系统', icon: IconThemeSystem },
        { value: 'light', label: '浅色', icon: IconThemeLight },
        { value: 'dark', label: '深色', icon: IconThemeDark },
      ]}
    />
  )
}

export function PresetConfig(props: SiteThemeFieldProps<ThemePreset>) {
  const { t } = useTranslation()
  return (
    <div>
      <SectionTitle
        title='配色方案'
        showReset={props.value !== props.defaultValue}
        onReset={() => props.onChange(props.defaultValue)}
      />
      <Radio<ThemePreset>
        value={props.value}
        disabled={props.disabled}
        onValueChange={props.onChange}
        className='grid w-full grid-cols-4 gap-3'
        aria-label='选择配色方案'
      >
        {THEME_PRESETS.map((preset) => (
          <Item
            key={preset.value}
            value={preset.value}
            className='group flex flex-col items-stretch outline-none'
            aria-label={t(`preset.${preset.value}`)}
          >
            <div
              className={cn(
                'ring-border relative h-12 rounded-md ring-[1px] transition',
                'group-data-checked:ring-primary group-data-checked:shadow-md',
                'group-focus-visible:ring-2',
                'group-hover:ring-primary/60'
              )}
            >
              <div
                aria-hidden='true'
                className='absolute inset-0 rounded-md'
                style={{
                  background:
                    preset.value === 'default'
                      ? 'linear-gradient(135deg, oklch(0.68 0.2 25) 0%, oklch(0.8 0.17 85) 25%, oklch(0.72 0.18 155) 50%, oklch(0.66 0.19 245) 75%, oklch(0.68 0.2 315) 100%)'
                      : `linear-gradient(135deg, ${preset.swatches[0]} 0%, ${preset.swatches[1] ?? preset.swatches[0]} 100%)`,
                }}
              />
              <CircleCheck
                className={cn(
                  'fill-primary absolute top-0 right-0 z-10 size-5 translate-x-1/2 -translate-y-1/2 stroke-white',
                  'group-data-unchecked:hidden'
                )}
                aria-hidden='true'
              />
            </div>
            <div className='mt-1.5 truncate text-center text-xs'>
              {t(`preset.${preset.value}`)}
            </div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

/**
 * Font options shown in the theme settings.
 *
 * Each option renders a live "Aa" preview in the font it represents.
 * The automatic option previews the default font for the draft color preset.
 */
const FONT_OPTIONS: {
  value: ThemeFont
  label: string
}[] = [
  { value: 'default', label: '自动' },
  { value: 'sans', label: '无衬线' },
  { value: 'serif', label: '衬线' },
]

export function FontConfig(
  props: SiteThemeFieldProps<ThemeFont> & { preset: ThemePreset }
) {
  return (
    <div>
      <SectionTitle
        title='字体'
        showReset={props.value !== props.defaultValue}
        onReset={() => props.onChange(props.defaultValue)}
      />
      <Radio<ThemeFont>
        value={props.value}
        disabled={props.disabled}
        onValueChange={props.onChange}
        className='grid w-full grid-cols-3 gap-4'
        aria-label='选择正文字体'
      >
        {FONT_OPTIONS.map((option) => (
          <Item
            key={option.value}
            value={option.value}
            className='group flex flex-col items-stretch outline-none'
            aria-label={option.label}
          >
            <div
              className={cn(
                'ring-border relative h-12 rounded-md ring-[1px] transition',
                'group-data-checked:ring-primary group-data-checked:shadow-md',
                'group-focus-visible:ring-2',
                'group-hover:ring-primary/60'
              )}
            >
              <CircleCheck
                className={cn(
                  'fill-primary absolute top-0 right-0 z-10 size-5 translate-x-1/2 -translate-y-1/2 stroke-white',
                  'group-data-unchecked:hidden'
                )}
                aria-hidden='true'
              />
              <span
                aria-hidden='true'
                className='text-foreground absolute inset-0 flex items-center justify-center text-lg leading-none font-medium'
                style={{
                  fontFamily: `var(--font-${resolveThemeFont(option.value, props.preset)})`,
                }}
              >
                Aa
              </span>
            </div>
            <div className='mt-1.5 text-center text-xs'>{option.label}</div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

const RADIUS_OPTIONS: {
  value: ThemeRadius
  label: string
  // CSS border-radius value used to render the visual preview corner.
  preview: string
}[] = [
  { value: 'default', label: '自动', preview: '1rem' },
  { value: 'none', label: '0', preview: '0' },
  { value: 'sm', label: '0.3', preview: '0.3rem' },
  { value: 'md', label: '0.5', preview: '0.5rem' },
  { value: 'lg', label: '0.75', preview: '0.75rem' },
  { value: 'xl', label: '1.0', preview: '1rem' },
]

export function RadiusConfig(props: SiteThemeFieldProps<ThemeRadius>) {
  return (
    <div>
      <SectionTitle
        title='圆角'
        showReset={props.value !== props.defaultValue}
        onReset={() => props.onChange(props.defaultValue)}
      />
      <Radio<ThemeRadius>
        value={props.value}
        disabled={props.disabled}
        onValueChange={props.onChange}
        className='grid w-full grid-cols-6 gap-2'
        aria-label='选择圆角'
      >
        {RADIUS_OPTIONS.map((option) => (
          <Item
            key={option.value}
            value={option.value}
            className='group flex flex-col items-stretch outline-none'
            aria-label={option.label}
          >
            <div
              className={cn(
                'ring-border relative h-12 rounded-md ring-[1px] transition',
                'group-data-checked:ring-primary group-data-checked:shadow-md',
                'group-focus-visible:ring-2',
                'group-hover:ring-primary/60'
              )}
            >
              <CircleCheck
                className={cn(
                  'fill-primary absolute top-0 right-0 z-10 size-5 translate-x-1/2 -translate-y-1/2 stroke-white',
                  'group-data-unchecked:hidden'
                )}
                aria-hidden='true'
              />
              <span
                aria-hidden='true'
                className='border-foreground/70 absolute top-2.5 left-2.5 size-3.5 border-t-[1.5px] border-l-[1.5px]'
                style={{ borderTopLeftRadius: option.preview }}
              />
            </div>
            <div className='mt-1.5 text-center text-xs'>{option.label}</div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

/**
 * Visual preview rows for the density preset. Each row's height represents
 * the relative line-height density (compact = tight rows, comfortable = wide).
 */
function ScalePreview(props: { rows: number; rowGap: string }) {
  return (
    <div
      aria-hidden='true'
      className='absolute inset-2.5 flex flex-col justify-center'
      style={{ gap: props.rowGap }}
    >
      {Array.from({ length: props.rows }, (_, index) => 85 - index * 10).map(
        (width) => (
          <span
            key={width}
            className='bg-foreground/60 block h-[2px] rounded-full'
            style={{ width: `${width}%` }}
          />
        )
      )}
    </div>
  )
}

export function ScaleConfig(props: SiteThemeFieldProps<ThemeScale>) {
  const scaleOptions: {
    value: ThemeScale
    label: string
    rows: number
    rowGap: string
  }[] = [
    { value: 'sm', label: '紧凑', rows: 4, rowGap: '3px' },
    { value: 'default', label: '默认', rows: 3, rowGap: '6px' },
    { value: 'lg', label: '舒适', rows: 2, rowGap: '10px' },
    { value: 'xl', label: '超大', rows: 1, rowGap: '14px' },
  ]
  return (
    <div>
      <SectionTitle
        title='界面密度'
        showReset={props.value !== props.defaultValue}
        onReset={() => props.onChange(props.defaultValue)}
      />
      <Radio<ThemeScale>
        value={props.value}
        disabled={props.disabled}
        onValueChange={props.onChange}
        className='grid w-full grid-cols-4 gap-3'
        aria-label='选择界面密度'
      >
        {scaleOptions.map((option) => (
          <Item
            key={option.value}
            value={option.value}
            className='group flex flex-col items-stretch outline-none'
            aria-label={option.label}
          >
            <div
              className={cn(
                'ring-border relative h-12 rounded-md ring-[1px] transition',
                'group-data-checked:ring-primary group-data-checked:shadow-md',
                'group-focus-visible:ring-2',
                'group-hover:ring-primary/60'
              )}
            >
              <CircleCheck
                className={cn(
                  'fill-primary absolute top-0 right-0 z-10 size-5 translate-x-1/2 -translate-y-1/2 stroke-white',
                  'group-data-unchecked:hidden'
                )}
                aria-hidden='true'
              />
              <ScalePreview rows={option.rows} rowGap={option.rowGap} />
            </div>
            <div className='mt-1.5 truncate text-center text-xs'>
              {option.label}
            </div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

export function SidebarConfig(props: SiteThemeFieldProps<SidebarStyle>) {
  return (
    <IllustratedThemeConfig
      {...props}
      title='侧边栏样式'
      options={[
        { value: 'inset', label: '内嵌', icon: IconSidebarInset },
        { value: 'floating', label: '悬浮', icon: IconSidebarFloating },
        { value: 'sidebar', label: '标准', icon: IconSidebarSidebar },
      ]}
    />
  )
}

export function LayoutConfig(props: SiteThemeFieldProps<SidebarLayout>) {
  return (
    <IllustratedThemeConfig
      {...props}
      title='布局'
      options={[
        { value: 'default', label: '展开侧边栏', icon: IconLayoutDefault },
        { value: 'icon', label: '仅显示图标', icon: IconLayoutCompact },
        { value: 'offcanvas', label: '隐藏侧边栏', icon: IconLayoutFull },
      ]}
    />
  )
}

export function ContentLayoutConfig(props: SiteThemeFieldProps<ContentLayout>) {
  return (
    <div>
      <SectionTitle
        title='内容宽度'
        showReset={props.value !== props.defaultValue}
        onReset={() => props.onChange(props.defaultValue)}
      />
      <Radio<ContentLayout>
        value={props.value}
        disabled={props.disabled}
        onValueChange={props.onChange}
        className='grid w-full grid-cols-2 gap-4'
        aria-label='选择内容宽度'
      >
        {[
          { value: 'full', label: '全宽' },
          { value: 'centered', label: '居中' },
        ].map((option) => (
          <Item
            key={option.value}
            value={option.value}
            className='group flex flex-col items-stretch outline-none'
            aria-label={option.label}
          >
            <div
              className={cn(
                'ring-border relative h-12 rounded-md ring-[1px] transition',
                'group-data-checked:ring-primary group-data-checked:shadow-md',
                'group-focus-visible:ring-2',
                'group-hover:ring-primary/60'
              )}
            >
              <CircleCheck
                className={cn(
                  'fill-primary absolute top-0 right-0 z-10 size-5 translate-x-1/2 -translate-y-1/2 stroke-white',
                  'group-data-unchecked:hidden'
                )}
                aria-hidden='true'
              />
              <ContentLayoutPreview centered={option.value === 'centered'} />
            </div>
            <div className='mt-1.5 truncate text-center text-xs'>
              {option.label}
            </div>
          </Item>
        ))}
      </Radio>
    </div>
  )
}

/**
 * Mini "page" mock used as the visual preview for content-width options.
 * `full` fills horizontally, `centered` clamps the body to a narrow column.
 */
function ContentLayoutPreview(props: { centered: boolean }) {
  return (
    <div aria-hidden='true' className='absolute inset-2 flex flex-col gap-1.5'>
      <span className='bg-foreground/40 block h-1.5 w-full rounded-sm' />
      <div
        className={cn(
          'flex flex-1 flex-col gap-1',
          props.centered ? 'mx-auto w-1/2' : 'w-full'
        )}
      >
        <span className='bg-foreground/60 block h-[2px] w-full rounded-full' />
        <span className='bg-foreground/60 block h-[2px] w-3/4 rounded-full' />
      </div>
    </div>
  )
}

export function DirConfig(props: SiteThemeFieldProps<Direction>) {
  return (
    <IllustratedThemeConfig
      {...props}
      title='方向'
      className='max-w-2xs grid-cols-2'
      options={[
        {
          value: 'ltr',
          label: '从左到右',
          icon: (props) => <IconDir {...props} dir='ltr' />,
        },
        {
          value: 'rtl',
          label: '从右到左',
          icon: (props) => <IconDir {...props} dir='rtl' />,
        },
      ]}
    />
  )
}
