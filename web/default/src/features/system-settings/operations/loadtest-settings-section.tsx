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
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const loadTestSchema = z.object({
  loadtest_setting: z.object({
    max_duration_seconds: z.coerce.number().int().min(5).max(3600),
    max_rps: z.coerce.number().int().min(1).max(10000),
    max_concurrency: z.coerce.number().int().min(1).max(10000),
    max_output_tokens: z.coerce.number().int().min(1).max(8192),
    request_timeout_seconds: z.coerce.number().int().min(1).max(600),
  }),
})

type LoadTestFormInput = z.input<typeof loadTestSchema>
type LoadTestFormValues = z.output<typeof loadTestSchema>

type LoadTestSettingsDefaults = {
  'loadtest_setting.max_duration_seconds': number
  'loadtest_setting.max_rps': number
  'loadtest_setting.max_concurrency': number
  'loadtest_setting.max_output_tokens': number
  'loadtest_setting.request_timeout_seconds': number
}

type Props = { defaultValues: LoadTestSettingsDefaults }

function buildDefaults(
  defaultValues: LoadTestSettingsDefaults
): LoadTestFormInput {
  return {
    loadtest_setting: {
      max_duration_seconds:
        defaultValues['loadtest_setting.max_duration_seconds'],
      max_rps: defaultValues['loadtest_setting.max_rps'],
      max_concurrency: defaultValues['loadtest_setting.max_concurrency'],
      max_output_tokens: defaultValues['loadtest_setting.max_output_tokens'],
      request_timeout_seconds:
        defaultValues['loadtest_setting.request_timeout_seconds'],
    },
  }
}

export function LoadTestSettingsSection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formDefaults = useMemo(
    () => buildDefaults(defaultValues),
    [defaultValues]
  )
  const form = useForm<LoadTestFormInput, unknown, LoadTestFormValues>({
    resolver: zodResolver(loadTestSchema),
    defaultValues: formDefaults,
  })

  useEffect(() => {
    form.reset(formDefaults)
  }, [form, formDefaults])

  const onSubmit = async (values: LoadTestFormValues) => {
    const updates = [
      [
        'loadtest_setting.max_duration_seconds',
        values.loadtest_setting.max_duration_seconds,
      ],
      ['loadtest_setting.max_rps', values.loadtest_setting.max_rps],
      [
        'loadtest_setting.max_concurrency',
        values.loadtest_setting.max_concurrency,
      ],
      [
        'loadtest_setting.max_output_tokens',
        values.loadtest_setting.max_output_tokens,
      ],
      [
        'loadtest_setting.request_timeout_seconds',
        values.loadtest_setting.request_timeout_seconds,
      ],
    ] as const
    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  return (
    <SettingsSection title={t('Load Test Limits')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <p className='text-muted-foreground text-sm'>
            {t(
              'These limits apply to every load-test demo run.'
            )}
          </p>
          <div className='grid grid-cols-1 gap-4 md:grid-cols-5'>
            <FormField
              control={form.control}
              name='loadtest_setting.max_duration_seconds'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Maximum duration (seconds)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={5}
                      max={3600}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: 5-3600 seconds')}
                  </p>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='loadtest_setting.request_timeout_seconds'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Request timeout (seconds)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={1}
                      max={600}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: 1-600 seconds')}
                  </p>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='loadtest_setting.max_rps'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Maximum requests per second')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={1}
                      max={10000}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: 1-10000 RPS')}
                  </p>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='loadtest_setting.max_concurrency'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Maximum concurrency')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={1}
                      max={10000}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: 1-10000 concurrent requests')}
                  </p>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='loadtest_setting.max_output_tokens'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max output tokens')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={1}
                      max={8192}
                      step={1}
                      {...safeNumberFieldProps(field)}
                    />
                  </FormControl>
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: {{min}}-{{max}} tokens', {
                      min: 1,
                      max: 8192,
                    })}
                  </p>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
