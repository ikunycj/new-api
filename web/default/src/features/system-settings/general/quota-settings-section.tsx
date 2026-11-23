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
import { useQuery } from '@tanstack/react-query'
import { useMemo, type ChangeEvent } from 'react'
import type { Resolver } from 'react-hook-form'
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
import { Switch } from '@/components/ui/switch'
import { ApiKeyRoutingGroupsField } from '@/features/keys/components/api-key-routing-groups-field'
import { formatQuota } from '@/lib/format'

import {
  getPricingGroupCatalog,
  PRICING_GROUP_CATALOG_QUERY_KEY,
} from '../billing/user-groups-api'
import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const quotaSchema = z.object({
  QuotaForNewUser: z.coerce.number().min(0),
  DefaultAPITokenGroup: z.string(),
  PreConsumedQuota: z.coerce.number().min(0),
  quota_setting: z.object({
    enable_free_model_pre_consume: z.boolean(),
  }),
})

type QuotaFormValues = z.infer<typeof quotaSchema>
type QuotaInputValue = number | ''

function formatQuotaInputValue(value: QuotaInputValue): string {
  return formatQuota(value === '' ? 0 : value)
}

function readDefaultGroups(value: string, groupNames: string[]): string[] {
  if (!value) return []
  if (value.trimStart().startsWith('[') && !groupNames.includes(value)) {
    try {
      const groups: unknown = JSON.parse(value)
      if (
        Array.isArray(groups) &&
        groups.every((group) => typeof group === 'string')
      ) {
        return groups
      }
    } catch {
      return [value]
    }
  }
  return [value]
}

type QuotaSettingsSectionProps = {
  defaultValues: QuotaFormValues
}

export function QuotaSettingsSection({
  defaultValues,
}: QuotaSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const pricingGroups = useQuery({
    queryKey: PRICING_GROUP_CATALOG_QUERY_KEY,
    queryFn: getPricingGroupCatalog,
  })
  const pricingGroupOptions = useMemo(() => {
    const catalog = pricingGroups.data
    if (!catalog) return []

    return catalog.availableToNewUsers.map((group) => ({
      value: group,
      label: catalog.displayNames[group] || group,
    }))
  }, [pricingGroups.data])
  const handleNumberChange =
    (onChange: (value: QuotaInputValue) => void) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber
      onChange(Number.isNaN(value) ? '' : value)
    }

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<QuotaFormValues>({
      resolver: zodResolver(quotaSchema) as Resolver<
        QuotaFormValues,
        unknown,
        QuotaFormValues
      >,
      defaultValues,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key,
            value: value as string | number | boolean,
          })
        }
      },
    })

  return (
    <SettingsSection title={t('User Defaults')}>
      <FormNavigationGuard when={isDirty} />

      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending || isSubmitting}
          />
          <FormDirtyIndicator isDirty={isDirty} />
          <div className='grid min-w-0 grid-cols-1 gap-6'>
            <FormField
              control={form.control}
              name='QuotaForNewUser'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('New User Quota')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      value={field.value ?? ''}
                      onChange={handleNumberChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Initial quota given to new users ({{formattedQuota}})',
                      {
                        formattedQuota: formatQuotaInputValue(field.value),
                      }
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='DefaultAPITokenGroup'
              render={({ field }) => {
                const selectedGroups = readDefaultGroups(
                  field.value,
                  pricingGroups.data?.names ?? []
                )

                return (
                  <FormItem>
                    <FormLabel>{t('Default API Group')}</FormLabel>
                    <FormControl>
                      <ApiKeyRoutingGroupsField
                        options={pricingGroupOptions}
                        value={selectedGroups}
                        disabled={pricingGroups.isLoading || isSubmitting}
                        onBlur={field.onBlur}
                        ref={field.ref}
                        onValueChange={(groups) => {
                          field.onChange(
                            groups.length > 1
                              ? JSON.stringify(groups)
                              : (groups[0] ?? '')
                          )
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Applies to API keys automatically created at registration when GENERATE_DEFAULT_TOKEN is enabled.'
                      )}
                      {selectedGroups.length > 1 && (
                        <span className='mt-1 block'>
                          {t(
                            'The first group is used first; retryable failures follow the order below.'
                          )}
                        </span>
                      )}
                      {selectedGroups.length === 0 && (
                        <span className='mt-1 block'>
                          {t('Use Account Group')}
                        </span>
                      )}
                    </FormDescription>
                    {pricingGroups.isError && (
                      <p role='status' className='text-destructive text-sm'>
                        {t('Unable to load groups')}
                      </p>
                    )}
                    {pricingGroups.isSuccess &&
                      selectedGroups.some(
                        (group) =>
                          !pricingGroupOptions.some(
                            (option) => option.value === group
                          )
                      ) && (
                        <p role='status' className='text-destructive text-sm'>
                          {t('This API group is unavailable to new users.')}
                        </p>
                      )}
                    <FormMessage />
                  </FormItem>
                )
              }}
            />

            <FormField
              control={form.control}
              name='PreConsumedQuota'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Pre-Consumed Quota')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      value={field.value ?? ''}
                      onChange={handleNumberChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Quota consumed before charging users')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='quota_setting.enable_free_model_pre_consume'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Pre-Consume for Free Models')}</FormLabel>
                    <FormDescription>
                      {t(
                        'When enabled, zero-cost models also pre-consume quota before final settlement.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      disabled={updateOption.isPending}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
