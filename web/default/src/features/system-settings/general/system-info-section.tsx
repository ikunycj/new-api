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
import { zodResolver } from '@hookform/resolvers/zod'
import { useWatch, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import type { ThemeCustomization } from '@/lib/theme-customization'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { LogoImagesDialog } from './logo-images-dialog'
import { SiteThemeCustomizationSection } from './site-theme-customization-section'

const logoUrlSchema = z
  .union(
    [z.string().url(), z.string().regex(/^\/logo\/[^/?#]+$/), z.literal('')],
    { error: '请输入完整的图片 URL 或 /logo/图片名' }
  )
  .optional()

const _systemInfoSchema = z.object({
  SystemName: z.string().min(1),
  ServerAddress: z.string().optional(),
  TopUpLink: z.string(),
  Logo: logoUrlSchema,
  Footer: z.string().optional(),
  About: z.string().optional(),
  HomePageContent: z.string().optional(),
  CustomerService: z.string().optional(),
  legal: z.object({
    user_agreement: z.string().optional(),
    privacy_policy: z.string().optional(),
  }),
})

const _themeCustomizationSchema = z.object({
  mode: z.enum(['system', 'light', 'dark']),
  preset: z.enum([
    'default',
    'anthropic',
    'simple-large',
    'underground',
    'rose-garden',
    'lake-view',
    'sunset-glow',
    'forest-whisper',
    'ocean-breeze',
    'lavender-dream',
    'ikun-love',
  ]),
  font: z.enum(['default', 'sans', 'serif']),
  radius: z.enum(['default', 'none', 'sm', 'md', 'lg', 'xl']),
  scale: z.enum(['default', 'sm', 'lg', 'xl']),
  contentLayout: z.enum(['full', 'centered']),
  sidebar: z.enum(['inset', 'floating', 'sidebar']),
  layout: z.enum(['default', 'icon', 'offcanvas']),
  direction: z.enum(['ltr', 'rtl']),
})

type SystemInfoSettingsValues = z.infer<typeof _systemInfoSchema>
type SystemInfoFormValues = SystemInfoSettingsValues & {
  siteThemeCustomization: z.infer<typeof _themeCustomizationSchema>
}

type SystemInfoSectionProps = {
  defaultValues: SystemInfoSettingsValues
}

function normalizeValue(value: unknown): string {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : String(value)
}

export function SystemInfoSection({ defaultValues }: SystemInfoSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const themeCustomizationContext = useThemeCustomization()

  const normalizedSystemInfo: SystemInfoSettingsValues = {
    SystemName: normalizeValue(defaultValues.SystemName),
    ServerAddress: normalizeValue(defaultValues.ServerAddress),
    TopUpLink: normalizeValue(defaultValues.TopUpLink),
    Logo: normalizeValue(defaultValues.Logo),
    Footer: normalizeValue(defaultValues.Footer),
    About: normalizeValue(defaultValues.About),
    HomePageContent: normalizeValue(defaultValues.HomePageContent),
    CustomerService: normalizeValue(defaultValues.CustomerService),
    legal: {
      user_agreement: normalizeValue(defaultValues.legal?.user_agreement),
      privacy_policy: normalizeValue(defaultValues.legal?.privacy_policy),
    },
  }

  const systemInfoSchemaWithI18n = z.object({
    SystemName: z.string().min(1, {
      error: () => t('System name is required'),
    }),
    ServerAddress: z.string().optional(),
    TopUpLink: z.string(),
    Logo: logoUrlSchema,
    Footer: z.string().optional(),
    About: z.string().optional(),
    HomePageContent: z.string().optional(),
    CustomerService: z.string().optional(),
    legal: z.object({
      user_agreement: z.string().optional(),
      privacy_policy: z.string().optional(),
    }),
    siteThemeCustomization: _themeCustomizationSchema,
  })

  const normalizedDefaults: SystemInfoFormValues = {
    ...normalizedSystemInfo,
    siteThemeCustomization: themeCustomizationContext.customization,
  }

  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<SystemInfoFormValues>({
      resolver: zodResolver(systemInfoSchemaWithI18n) as Resolver<
        SystemInfoFormValues,
        unknown,
        SystemInfoFormValues
      >,
      defaultValues: normalizedDefaults,
      onSubmit: async (data, changedFields) => {
        const themeFieldPrefix = 'siteThemeCustomization.'
        const themeCustomizationChanged = Object.keys(changedFields).some(
          (key) => key.startsWith(themeFieldPrefix)
        )

        for (const [key, value] of Object.entries(changedFields)) {
          if (key.startsWith(themeFieldPrefix)) continue

          let v = normalizeValue(value)
          if (key === 'ServerAddress') {
            v = v.replace(/\/+$/, '')
          }
          const response = await updateOption.mutateAsync({
            key,
            value: v,
          })
          if (!response.success) return false
        }

        if (themeCustomizationChanged) {
          const response = await updateOption.mutateAsync({
            key: 'SiteThemeCustomization',
            value: JSON.stringify(data.siteThemeCustomization),
          })
          if (!response.success) return false
        }
      },
    })

  const serverAddress = useWatch({
    control: form.control,
    name: 'ServerAddress',
  })
  const siteThemeCustomization = useWatch({
    control: form.control,
    name: 'siteThemeCustomization',
  })

  function updateThemeCustomization<K extends keyof ThemeCustomization>(
    key: K,
    value: ThemeCustomization[K]
  ) {
    const fieldName =
      `siteThemeCustomization.${key}` as `siteThemeCustomization.${keyof ThemeCustomization}`
    const fieldValue =
      value as SystemInfoFormValues['siteThemeCustomization'][keyof ThemeCustomization]
    form.setValue(fieldName, fieldValue, {
      shouldDirty: true,
    })
  }

  return (
    <>
      <FormNavigationGuard when={isDirty} />

      <SettingsSection title={t('System Information')}>
        <Form {...form}>
          <SettingsForm onSubmit={handleSubmit}>
            <SettingsPageFormActions
              onSave={handleSubmit}
              onReset={handleReset}
              isSaving={isSubmitting || updateOption.isPending}
              isResetDisabled={!isDirty}
            />
            <FormDirtyIndicator isDirty={isDirty} />
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='SystemName'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('System Name')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('New API')} {...field} />
                    </FormControl>
                    <FormDescription>
                      {t('The name displayed across the application')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ServerAddress'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Server Address')}</FormLabel>
                    <FormControl>
                      <Input placeholder='https://yourdomain.com' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'The public URL of your server, used for OAuth callbacks, webhooks, and other external integrations'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='TopUpLink'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Top-Up Link')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('https://example.com/topup')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('External link for users to purchase quota')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='Logo'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Logo URL')}</FormLabel>
                    <div className='flex flex-wrap gap-2'>
                      <FormControl>
                        <Input
                          className='min-w-0 flex-1'
                          placeholder={t('https://example.com/logo.png')}
                          {...field}
                        />
                      </FormControl>
                      <LogoImagesDialog
                        serverAddress={serverAddress ?? ''}
                        selectedUrl={field.value ?? ''}
                        onSelect={field.onChange}
                      />
                    </div>
                    <FormDescription>
                      {t('URL to your logo image (optional)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='Footer'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Footer')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          '© 2025 Your Company. All rights reserved.'
                        )}
                        rows={4}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Footer text displayed at the bottom of pages')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='About'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Contact us')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          'Enter HTML code (e.g., <p>Contact details...</p>) or a URL (e.g., https://example.com) to embed it as an iframe'
                        )}
                        rows={4}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Supports HTML markup or iframe embedding. Enter HTML code directly, or provide a complete URL to automatically embed it as an iframe.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <SettingsFormGridItem span='full'>
                <FormField
                  control={form.control}
                  name='HomePageContent'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Home Page Content')}</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder={t('Welcome to our New API...')}
                          rows={6}
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Content displayed on the home page (supports Markdown)'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </SettingsFormGridItem>

              <SettingsFormGridItem span='full'>
                <FormField
                  control={form.control}
                  name='CustomerService'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Customer Service')}</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder={t(
                            'Enter customer service contact information'
                          )}
                          rows={4}
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Information displayed to users for contacting customer service'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </SettingsFormGridItem>

              <FormField
                control={form.control}
                name='legal.user_agreement'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('User Agreement')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          'Provide Markdown, HTML, or an external URL for the user agreement'
                        )}
                        rows={6}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Leave empty to disable the agreement requirement. Supports Markdown, HTML, or a full URL to redirect users.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='legal.privacy_policy'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Privacy Policy')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          'Provide Markdown, HTML, or an external URL for the privacy policy'
                        )}
                        rows={6}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Leave empty to disable the privacy policy requirement. Supports Markdown, HTML, or a full URL to redirect users.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGrid>
            <SiteThemeCustomizationSection
              value={siteThemeCustomization}
              defaults={themeCustomizationContext.defaults}
              disabled={isSubmitting || updateOption.isPending}
              onChange={updateThemeCustomization}
            />
          </SettingsForm>
        </Form>
      </SettingsSection>
    </>
  )
}
