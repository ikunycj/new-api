/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

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
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { MultiSelect, type Option } from '@/components/multi-select'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
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
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { getChannels } from '@/features/channels/api'
import { CHANNEL_TYPE_OPTIONS } from '@/features/channels/constants'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  GlobalPolicyEditor,
  PreferredModelsEditor,
} from './global-settings-fields'
import {
  canonicalizeGlobalModelSettings,
  globalModelSettingsFormSchema,
  parseGlobalModelSettings,
  serializeGlobalModelSettings,
  type GlobalModelSettingsFormInput,
  type GlobalModelSettingsFormValues,
  type GlobalModelSettingsRawValues,
} from './global-settings-form'

type GlobalSettingsCardProps = {
  defaultValues: GlobalModelSettingsRawValues
}

function appendUnknownOptions(
  options: Option[],
  values: string[],
  label: (value: string) => string
): Option[] {
  const known = new Set(options.map((option) => option.value))
  const unknown = values
    .filter((value) => !known.has(value))
    .map((value) => ({ value, label: label(value) }))
  return [...options, ...unknown]
}

export function GlobalSettingsCard(props: GlobalSettingsCardProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const defaultPassThrough =
    props.defaultValues['global.pass_through_request_enabled']
  const defaultThinkingBlacklist =
    props.defaultValues['global.thinking_model_blacklist']
  const defaultResponsePolicy =
    props.defaultValues['global.chat_completions_to_responses_policy']
  const defaultPreferredModels = props.defaultValues.PreferredModels
  const defaultPingEnabled =
    props.defaultValues['general_setting.ping_interval_enabled']
  const defaultPingInterval =
    props.defaultValues['general_setting.ping_interval_seconds']
  const parsedDefaults = useMemo(
    () =>
      parseGlobalModelSettings({
        'global.pass_through_request_enabled': defaultPassThrough,
        'global.thinking_model_blacklist': defaultThinkingBlacklist,
        'global.chat_completions_to_responses_policy': defaultResponsePolicy,
        PreferredModels: defaultPreferredModels,
        'general_setting.ping_interval_enabled': defaultPingEnabled,
        'general_setting.ping_interval_seconds': defaultPingInterval,
      }),
    [
      defaultPassThrough,
      defaultThinkingBlacklist,
      defaultResponsePolicy,
      defaultPreferredModels,
      defaultPingEnabled,
      defaultPingInterval,
    ]
  )

  const form = useForm<
    GlobalModelSettingsFormInput,
    unknown,
    GlobalModelSettingsFormValues
  >({
    resolver: zodResolver(globalModelSettingsFormSchema),
    defaultValues: parsedDefaults,
  })

  useEffect(() => {
    form.reset(parsedDefaults)
  }, [form, parsedDefaults])

  const channelsQuery = useQuery({
    queryKey: ['global-settings-channels'],
    queryFn: () => getChannels({ p: 1, page_size: 100, id_sort: true }),
  })
  const channelItems = channelsQuery.data?.data?.items
  const channels = useMemo(() => channelItems ?? [], [channelItems])
  const channelOptions = useMemo(
    () =>
      channels.map((channel) => ({
        value: String(channel.id),
        label: `${channel.name} (#${channel.id})`,
      })),
    [channels]
  )
  const channelTypeOptions = useMemo(
    () =>
      CHANNEL_TYPE_OPTIONS.map((option) => ({
        value: String(option.value),
        label: t(option.label),
      })),
    [t]
  )

  const pingEnabled = form.watch('general_setting.ping_interval_enabled')

  const onSubmit = async (values: GlobalModelSettingsFormValues) => {
    const flattenedDefaults = canonicalizeGlobalModelSettings(
      props.defaultValues
    )
    const flattenedValues = serializeGlobalModelSettings(values)
    const updates = Object.entries(flattenedValues).filter(
      ([key, value]) =>
        value !== flattenedDefaults[key as keyof GlobalModelSettingsRawValues]
    )

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  return (
    <SettingsSection title={t('Global Model Configuration')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <div className='space-y-6 lg:col-span-2'>
            <FormField
              control={form.control}
              name='global.pass_through_request_enabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Enable Request Passthrough')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />

            <FormField
              control={form.control}
              name='global.thinking_model_blacklist'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Models that skip thinking suffix processing')}
                  </FormLabel>
                  <FormControl>
                    <MultiSelect
                      selected={field.value}
                      options={[]}
                      onChange={field.onChange}
                      allowCreate
                      placeholder={t('Add a model name')}
                      createLabel={t('Add model "{{value}}"')}
                      emptyText={t('No models added')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Models listed here will not automatically append or remove -thinking / -nothinking suffixes.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='PreferredModels'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Preferred Models')}</FormLabel>
                  <FormControl>
                    <PreferredModelsEditor
                      value={field.value}
                      onChange={field.onChange}
                      placeholder={t('Enter a model name')}
                      emptyLabel={t(
                        'No preferred models. The default order will be used.'
                      )}
                      deleteAriaLabel={(model) =>
                        t('Delete {{value}}', { value: model })
                      }
                      dragAriaLabel={(model) =>
                        t('Drag to reorder {{group}}', { group: model })
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Models are tried in this order when selecting a default model. Add one model per row.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='global.chat_completions_to_responses_policy'
              render={({ field }) => {
                const selectedChannelIds = field.value.channel_ids.map(String)
                const selectedChannelTypes =
                  field.value.channel_types.map(String)
                const channelOptionsWithLegacy = appendUnknownOptions(
                  channelOptions,
                  selectedChannelIds,
                  (value) => t('Channel #{{id}}', { id: value })
                )
                const channelTypeOptionsWithLegacy = appendUnknownOptions(
                  channelTypeOptions,
                  selectedChannelTypes,
                  (value) => t('Channel type #{{id}}', { id: value })
                )

                return (
                  <FormItem className='gap-4'>
                    <FormLabel>
                      {t('ChatCompletions -> Responses Compatibility')}
                    </FormLabel>
                    <FormDescription>
                      {t(
                        'Convert selected Chat Completions requests to the Responses API without editing a policy object.'
                      )}
                    </FormDescription>
                    <Alert>
                      <AlertTitle>{t('Warning')}</AlertTitle>
                      <AlertDescription>
                        {t(
                          'This feature is experimental. The client request must still match the selected upstream protocol.'
                        )}
                      </AlertDescription>
                    </Alert>

                    <FormControl>
                      <GlobalPolicyEditor
                        value={field.value}
                        onChange={field.onChange}
                        channelOptions={channelOptionsWithLegacy}
                        channelTypeOptions={channelTypeOptionsWithLegacy}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )
              }}
            />
          </div>

          <Separator />

          <FormField
            control={form.control}
            name='general_setting.ping_interval_enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Keep-alive Ping')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Periodically send ping frames to keep streaming connections active.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='general_setting.ping_interval_seconds'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Ping Interval (seconds)')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    disabled={!pingEnabled}
                    className='w-24'
                    value={
                      field.value === undefined || field.value === null
                        ? ''
                        : String(field.value)
                    }
                    onChange={(event) => field.onChange(event.target.value)}
                    onBlur={field.onBlur}
                    name={field.name}
                    ref={field.ref}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Recommended to keep this high to avoid upstream throttling.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
