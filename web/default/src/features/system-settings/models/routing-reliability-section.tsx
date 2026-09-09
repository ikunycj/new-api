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
import { useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { parseHttpStatusCodeRules } from '@/lib/http-status-code-rules'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const numericString = z.string().refine((value) => {
  const trimmed = value.trim()
  if (!trimmed) return true
  return !Number.isNaN(Number(trimmed)) && Number(trimmed) >= 0
}, 'Enter a non-negative number or leave empty')

const channelTestModes = ['scheduled_all', 'passive_recovery'] as const
type ChannelTestMode = (typeof channelTestModes)[number]

// The two health modes differ in one way that matters operationally: `observe`
// records scores and exposes them for inspection but never lets them touch
// channel selection, which makes it the safe way to validate the signal on
// production traffic. `active` additionally allows the weighted router to
// consume them.
const channelHealthModes = ['observe', 'active'] as const
type ChannelHealthMode = (typeof channelHealthModes)[number]

function normalizeChannelHealthMode(value?: string): ChannelHealthMode {
  return value === 'active' ? 'active' : 'observe'
}

const circuitPolicySchema = z.object({
  failure_threshold: z.number().int().min(1).max(10000),
  window_seconds: z.number().int().min(1).max(86400),
  cooldown_seconds: z.number().int().min(1).max(86400),
  half_open_requests: z.number().int().min(1).max(100),
})

const channelCircuitConfigSchema = z.object({
  default: circuitPolicySchema,
  modes: z.object({
    cost_first: circuitPolicySchema,
    stability_first: circuitPolicySchema,
  }),
  presets: z
    .array(
      z.object({
        key: z.string().trim().min(1),
        label: z.string().trim().min(1),
        failure_threshold: z.number().int().min(1).max(10000),
        window_seconds: z.number().int().min(1).max(86400),
        cooldown_seconds: z.number().int().min(1).max(86400),
        half_open_requests: z.number().int().min(1).max(100),
      })
    )
    .min(1)
    .max(20),
})

const routingReliabilitySchema = z
  .object({
    RetryTimes: z.coerce.number().min(0).max(10),
    ChannelCircuitEnabled: z.boolean(),
    ChannelCircuitConfig: z.string().superRefine((value, ctx) => {
      try {
        if (!channelCircuitConfigSchema.safeParse(JSON.parse(value)).success) {
          ctx.addIssue({
            code: 'custom',
            message: 'Invalid JSON format or values out of allowed range',
          })
        }
      } catch {
        ctx.addIssue({ code: 'custom', message: 'Invalid JSON format' })
      }
    }),
    ChannelDisableThreshold: numericString,
    // These bounds are deliberately identical to the server-side validation in
    // model/option.go, so an out-of-range value is rejected here instead of
    // costing a round-trip that ends in a 400.
    ChannelHealthEnabled: z.boolean(),
    ChannelHealthMode: z.enum(channelHealthModes),
    ChannelHealthHalfLifeSeconds: z.coerce.number().int().min(5).max(86400),
    ChannelHealthMinSamples: z.coerce.number().int().min(0).max(1000),
    ChannelHealthLatencyHalfLifeSeconds: z.coerce
      .number()
      .int()
      .min(5)
      .max(86400),
    ChannelHealthStateTTLSeconds: z.coerce.number().int().min(60).max(604800),
    ChannelHealthProbeEnabled: z.boolean(),
    ChannelHealthProbeIntervalSeconds: z.coerce
      .number()
      .int()
      .min(10)
      .max(86400),
    ChannelHealthProbeIdleGraceSeconds: z.coerce
      .number()
      .int()
      .min(0)
      .max(86400),
    AutomaticDisableChannelEnabled: z.boolean(),
    AutomaticEnableChannelEnabled: z.boolean(),
    AutomaticDisableKeywords: z.string(),
    AutomaticDisableStatusCodes: z.string(),
    AutomaticRetryStatusCodes: z.string(),
    monitor_setting: z.object({
      auto_test_channel_enabled: z.boolean(),
      auto_test_channel_minutes: z.coerce
        .number()
        .int()
        .min(1, 'Interval must be at least 1 minute'),
      channel_test_mode: z.enum(channelTestModes),
    }),
  })
  .superRefine((values, ctx) => {
    const disableParsed = parseHttpStatusCodeRules(
      values.AutomaticDisableStatusCodes
    )
    if (!disableParsed.ok) {
      ctx.addIssue({
        code: 'custom',
        path: ['AutomaticDisableStatusCodes'],
        message: `Invalid status code rules: ${disableParsed.invalidTokens.join(
          ', '
        )}`,
      })
    }

    const retryParsed = parseHttpStatusCodeRules(
      values.AutomaticRetryStatusCodes
    )
    if (!retryParsed.ok) {
      ctx.addIssue({
        code: 'custom',
        path: ['AutomaticRetryStatusCodes'],
        message: `Invalid status code rules: ${retryParsed.invalidTokens.join(
          ', '
        )}`,
      })
    }
  })

type RoutingReliabilityFormValues = z.output<typeof routingReliabilitySchema>
type RoutingReliabilityFormInput = z.input<typeof routingReliabilitySchema>

type RoutingReliabilitySectionProps = {
  defaultValues: {
    RetryTimes: number
    ChannelCircuitEnabled: boolean
    ChannelCircuitConfig: string
    ChannelHealthEnabled: boolean
    ChannelHealthMode: ChannelHealthMode
    ChannelHealthHalfLifeSeconds: number
    ChannelHealthMinSamples: number
    ChannelHealthLatencyHalfLifeSeconds: number
    ChannelHealthStateTTLSeconds: number
    ChannelHealthProbeEnabled: boolean
    ChannelHealthProbeIntervalSeconds: number
    ChannelHealthProbeIdleGraceSeconds: number
    ChannelDisableThreshold: string
    AutomaticDisableChannelEnabled: boolean
    AutomaticEnableChannelEnabled: boolean
    AutomaticDisableKeywords: string
    AutomaticDisableStatusCodes: string
    AutomaticRetryStatusCodes: string
    'monitor_setting.auto_test_channel_enabled': boolean
    'monitor_setting.auto_test_channel_minutes': number
    'monitor_setting.channel_test_mode': ChannelTestMode
  }
}

function normalizeLineEndings(value: string) {
  return value.replaceAll('\r\n', '\n')
}

function normalizeJsonDocument(value: string) {
  const raw = value.trim()
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

type NormalizedRoutingReliabilityValues = {
  RetryTimes: number
  ChannelCircuitEnabled: boolean
  ChannelCircuitConfig: string
  ChannelHealthEnabled: boolean
  ChannelHealthMode: ChannelHealthMode
  ChannelHealthHalfLifeSeconds: number
  ChannelHealthMinSamples: number
  ChannelHealthLatencyHalfLifeSeconds: number
  ChannelHealthStateTTLSeconds: number
  ChannelHealthProbeEnabled: boolean
  ChannelHealthProbeIntervalSeconds: number
  ChannelHealthProbeIdleGraceSeconds: number
  ChannelDisableThreshold: string
  AutomaticDisableChannelEnabled: boolean
  AutomaticEnableChannelEnabled: boolean
  AutomaticDisableKeywords: string
  AutomaticDisableStatusCodes: string
  AutomaticRetryStatusCodes: string
  'monitor_setting.auto_test_channel_enabled': boolean
  'monitor_setting.auto_test_channel_minutes': number
  'monitor_setting.channel_test_mode': ChannelTestMode
}

function normalizeChannelTestMode(value?: string): ChannelTestMode {
  return value === 'passive_recovery' ? 'passive_recovery' : 'scheduled_all'
}

const buildFormDefaults = (
  defaults: RoutingReliabilitySectionProps['defaultValues']
): RoutingReliabilityFormInput => ({
  RetryTimes: defaults.RetryTimes ?? 0,
  ChannelCircuitEnabled: defaults.ChannelCircuitEnabled,
  ChannelCircuitConfig: defaults.ChannelCircuitConfig ?? '{}',
  ChannelHealthEnabled: defaults.ChannelHealthEnabled,
  ChannelHealthMode: normalizeChannelHealthMode(defaults.ChannelHealthMode),
  ChannelHealthHalfLifeSeconds: defaults.ChannelHealthHalfLifeSeconds,
  ChannelHealthMinSamples: defaults.ChannelHealthMinSamples,
  ChannelHealthLatencyHalfLifeSeconds:
    defaults.ChannelHealthLatencyHalfLifeSeconds,
  ChannelHealthStateTTLSeconds: defaults.ChannelHealthStateTTLSeconds,
  ChannelHealthProbeEnabled: defaults.ChannelHealthProbeEnabled,
  ChannelHealthProbeIntervalSeconds: defaults.ChannelHealthProbeIntervalSeconds,
  ChannelHealthProbeIdleGraceSeconds:
    defaults.ChannelHealthProbeIdleGraceSeconds,
  ChannelDisableThreshold: defaults.ChannelDisableThreshold ?? '',
  AutomaticDisableChannelEnabled: defaults.AutomaticDisableChannelEnabled,
  AutomaticEnableChannelEnabled: defaults.AutomaticEnableChannelEnabled,
  AutomaticDisableKeywords: normalizeLineEndings(
    defaults.AutomaticDisableKeywords ?? ''
  ),
  AutomaticDisableStatusCodes: defaults.AutomaticDisableStatusCodes ?? '',
  AutomaticRetryStatusCodes: defaults.AutomaticRetryStatusCodes ?? '',
  monitor_setting: {
    auto_test_channel_enabled:
      defaults['monitor_setting.auto_test_channel_enabled'],
    auto_test_channel_minutes:
      defaults['monitor_setting.auto_test_channel_minutes'],
    channel_test_mode: normalizeChannelTestMode(
      defaults['monitor_setting.channel_test_mode']
    ),
  },
})

const normalizeDefaults = (
  defaults: RoutingReliabilitySectionProps['defaultValues']
): NormalizedRoutingReliabilityValues => ({
  RetryTimes: defaults.RetryTimes ?? 0,
  ChannelCircuitEnabled: defaults.ChannelCircuitEnabled,
  ChannelCircuitConfig: normalizeJsonDocument(defaults.ChannelCircuitConfig),
  ChannelHealthEnabled: defaults.ChannelHealthEnabled,
  ChannelHealthMode: normalizeChannelHealthMode(defaults.ChannelHealthMode),
  ChannelHealthHalfLifeSeconds: defaults.ChannelHealthHalfLifeSeconds,
  ChannelHealthMinSamples: defaults.ChannelHealthMinSamples,
  ChannelHealthLatencyHalfLifeSeconds:
    defaults.ChannelHealthLatencyHalfLifeSeconds,
  ChannelHealthStateTTLSeconds: defaults.ChannelHealthStateTTLSeconds,
  ChannelHealthProbeEnabled: defaults.ChannelHealthProbeEnabled,
  ChannelHealthProbeIntervalSeconds: defaults.ChannelHealthProbeIntervalSeconds,
  ChannelHealthProbeIdleGraceSeconds:
    defaults.ChannelHealthProbeIdleGraceSeconds,
  ChannelDisableThreshold: (defaults.ChannelDisableThreshold ?? '').trim(),
  AutomaticDisableChannelEnabled: defaults.AutomaticDisableChannelEnabled,
  AutomaticEnableChannelEnabled: defaults.AutomaticEnableChannelEnabled,
  AutomaticDisableKeywords: normalizeLineEndings(
    defaults.AutomaticDisableKeywords ?? ''
  ),
  AutomaticDisableStatusCodes: parseHttpStatusCodeRules(
    defaults.AutomaticDisableStatusCodes ?? ''
  ).normalized,
  AutomaticRetryStatusCodes: parseHttpStatusCodeRules(
    defaults.AutomaticRetryStatusCodes ?? ''
  ).normalized,
  'monitor_setting.auto_test_channel_enabled':
    defaults['monitor_setting.auto_test_channel_enabled'],
  'monitor_setting.auto_test_channel_minutes':
    defaults['monitor_setting.auto_test_channel_minutes'],
  'monitor_setting.channel_test_mode': normalizeChannelTestMode(
    defaults['monitor_setting.channel_test_mode']
  ),
})

const normalizeFormValues = (
  values: RoutingReliabilityFormValues
): NormalizedRoutingReliabilityValues => ({
  RetryTimes: values.RetryTimes,
  ChannelCircuitEnabled: values.ChannelCircuitEnabled,
  ChannelCircuitConfig: normalizeJsonDocument(values.ChannelCircuitConfig),
  ChannelHealthEnabled: values.ChannelHealthEnabled,
  ChannelHealthMode: values.ChannelHealthMode,
  ChannelHealthHalfLifeSeconds: values.ChannelHealthHalfLifeSeconds,
  ChannelHealthMinSamples: values.ChannelHealthMinSamples,
  ChannelHealthLatencyHalfLifeSeconds:
    values.ChannelHealthLatencyHalfLifeSeconds,
  ChannelHealthStateTTLSeconds: values.ChannelHealthStateTTLSeconds,
  ChannelHealthProbeEnabled: values.ChannelHealthProbeEnabled,
  ChannelHealthProbeIntervalSeconds: values.ChannelHealthProbeIntervalSeconds,
  ChannelHealthProbeIdleGraceSeconds: values.ChannelHealthProbeIdleGraceSeconds,
  ChannelDisableThreshold: values.ChannelDisableThreshold.trim(),
  AutomaticDisableChannelEnabled: values.AutomaticDisableChannelEnabled,
  AutomaticEnableChannelEnabled: values.AutomaticEnableChannelEnabled,
  AutomaticDisableKeywords: normalizeLineEndings(
    values.AutomaticDisableKeywords
  ),
  AutomaticDisableStatusCodes: parseHttpStatusCodeRules(
    values.AutomaticDisableStatusCodes
  ).normalized,
  AutomaticRetryStatusCodes: parseHttpStatusCodeRules(
    values.AutomaticRetryStatusCodes
  ).normalized,
  'monitor_setting.auto_test_channel_enabled':
    values.monitor_setting.auto_test_channel_enabled,
  'monitor_setting.auto_test_channel_minutes':
    values.monitor_setting.auto_test_channel_minutes,
  'monitor_setting.channel_test_mode': values.monitor_setting.channel_test_mode,
})

export function RoutingReliabilitySection({
  defaultValues,
}: RoutingReliabilitySectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<NormalizedRoutingReliabilityValues>(
    normalizeDefaults(defaultValues)
  )

  const formDefaults = useMemo(
    () => buildFormDefaults(defaultValues),
    [defaultValues]
  )

  const form = useForm<
    RoutingReliabilityFormInput,
    unknown,
    RoutingReliabilityFormValues
  >({
    resolver: zodResolver(routingReliabilitySchema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const autoDisableStatusCodes = form.watch('AutomaticDisableStatusCodes')
  const autoRetryStatusCodes = form.watch('AutomaticRetryStatusCodes')
  const channelTestMode = form.watch('monitor_setting.channel_test_mode')
  // The tuning knobs are meaningless while scoring is off, and probing is a
  // strict sub-switch of scoring on the backend too (IsChannelHealthProbeEnabled
  // requires both flags), so the UI mirrors that dependency rather than offering
  // controls that silently do nothing.
  const healthEnabled = form.watch('ChannelHealthEnabled')
  const healthMode = form.watch('ChannelHealthMode')
  const probeEnabled = form.watch('ChannelHealthProbeEnabled')
  const autoDisableParsed = useMemo(
    () => parseHttpStatusCodeRules(autoDisableStatusCodes),
    [autoDisableStatusCodes]
  )
  const autoRetryParsed = useMemo(
    () => parseHttpStatusCodeRules(autoRetryStatusCodes),
    [autoRetryStatusCodes]
  )

  const onSubmit = async (values: RoutingReliabilityFormValues) => {
    const normalized = normalizeFormValues(values)
    const updates = (
      Object.keys(normalized) as Array<keyof NormalizedRoutingReliabilityValues>
    ).filter((key) => normalized[key] !== baselineRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      const value = normalized[key]
      await updateOption.mutateAsync({
        key,
        value,
      })
    }

    baselineRef.current = normalized
  }

  return (
    <SettingsSection title={t('Routing Reliability')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <div className='flex min-w-0 flex-col gap-4'>
            <div className='bg-muted/20 rounded-lg border p-4'>
              <FormField
                control={form.control}
                name='ChannelCircuitEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Circuit protection')}</FormLabel>
                      <FormDescription>
                        {field.value ? t('Enabled') : t('Disabled')}
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
            </div>

            <FormField
              control={form.control}
              name='ChannelCircuitConfig'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Circuit configuration (JSON)')}</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={18}
                      className='font-mono text-xs'
                      spellCheck={false}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Central defaults for balanced, cost-first, stability-first, and quick presets.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <Separator />

          <div className='flex min-w-0 flex-col gap-4'>
            <div className='flex flex-col gap-1'>
              <h4 className='text-sm font-medium'>
                {t('Real-time channel health scoring')}
              </h4>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Scores every channel from live traffic using a time-decayed availability and latency average. Scores are bucketed per model family, so Claude channels are only compared with Claude channels and GPT with GPT.'
                )}
              </p>
            </div>

            <div className='grid min-w-0 gap-6 lg:grid-cols-2'>
              <FormField
                control={form.control}
                name='ChannelHealthEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Health scoring')}</FormLabel>
                      <FormDescription>
                        {field.value
                          ? t('Collecting samples')
                          : t(
                              'Off: no samples are collected and routing is unchanged'
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
                name='ChannelHealthMode'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Scoring mode')}</FormLabel>
                    <Select
                      items={[
                        { value: 'observe', label: t('Observe only') },
                        {
                          value: 'active',
                          label: t('Active (affects routing)'),
                        },
                      ]}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={!healthEnabled}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='observe'>
                            {t('Observe only')}
                          </SelectItem>
                          <SelectItem value='active'>
                            {t('Active (affects routing)')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {healthMode === 'active'
                        ? t(
                            'Live scores replace the previous-day success rate in the availability input. Only groups using the weighted routing strategy are affected; priority groups still take their first candidate.'
                          )
                        : t(
                            'Scores are recorded and visible but never influence channel selection. Safe to enable on production traffic.'
                          )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='grid min-w-0 gap-6 lg:grid-cols-2 xl:grid-cols-4'>
              <FormField
                control={form.control}
                name='ChannelHealthHalfLifeSeconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Availability half-life (seconds)')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={5}
                        max={86400}
                        step={1}
                        disabled={!healthEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'How fast old samples lose weight. Shorter reacts quicker but is noisier.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ChannelHealthMinSamples'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum samples')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        max={1000}
                        step={1}
                        disabled={!healthEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Below this effective sample count the score is blended toward neutral, so a barely-used channel is neither blindly trusted nor unfairly condemned.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ChannelHealthLatencyHalfLifeSeconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Latency half-life (seconds)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={5}
                        max={86400}
                        step={1}
                        disabled={!healthEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "Decay of the channel's own latency baseline, which the newest response is scored against."
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ChannelHealthStateTTLSeconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('State TTL (seconds)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={60}
                        max={604800}
                        step={1}
                        disabled={!healthEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Idle entries are evicted after this long so removed channels do not linger in memory.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='grid min-w-0 gap-6 lg:grid-cols-3'>
              <FormField
                control={form.control}
                name='ChannelHealthProbeEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Synthetic probing')}</FormLabel>
                      <FormDescription>
                        {t(
                          "Probes idle channels with their family's cheapest model so they still have a fresh score. Costs upstream quota."
                        )}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={!healthEnabled}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              <FormField
                control={form.control}
                name='ChannelHealthProbeIntervalSeconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Probe interval (seconds)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={10}
                        max={86400}
                        step={1}
                        disabled={!healthEnabled || !probeEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('How often the probe loop wakes up.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ChannelHealthProbeIdleGraceSeconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Probe idle grace (seconds)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        max={86400}
                        step={1}
                        disabled={!healthEnabled || !probeEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Channels that real traffic touched within this window are skipped, since real evidence beats a synthetic call.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>

          <Separator />

          <div className='flex min-w-0 flex-col gap-4'>
            <div className='flex flex-col gap-1'>
              <h4 className='text-sm font-medium'>{t('Request retry')}</h4>
            </div>
            <div className='grid min-w-0 gap-6 xl:grid-cols-[minmax(12rem,24rem)_minmax(0,1fr)]'>
              <FormField
                control={form.control}
                name='RetryTimes'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Retry Times')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min='0'
                        max='10'
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Number of times to retry failed requests (0-10)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='AutomaticRetryStatusCodes'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Auto-retry status codes')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('e.g. 401, 403, 429, 500-599')}
                        value={field.value}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Accepts comma-separated status codes and inclusive ranges.'
                      )}{' '}
                      {autoRetryParsed.ok &&
                        autoRetryParsed.normalized &&
                        autoRetryParsed.normalized !== field.value.trim() && (
                          <span className='text-muted-foreground'>
                            {t('Normalized:')} {autoRetryParsed.normalized}
                          </span>
                        )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>

          <Separator />

          <div className='flex min-w-0 flex-col gap-4'>
            <div className='flex flex-col gap-1'>
              <h4 className='text-sm font-medium'>
                {t('Channel health checks')}
              </h4>
            </div>
            <div className='grid min-w-0 gap-6 lg:grid-cols-3'>
              <FormField
                control={form.control}
                name='monitor_setting.auto_test_channel_enabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Scheduled channel tests')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Automatically probe all channels in the background'
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
                name='monitor_setting.channel_test_mode'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Channel test mode')}</FormLabel>
                    <Select
                      items={[
                        {
                          value: 'scheduled_all',
                          label: t('Scheduled full test'),
                        },
                        {
                          value: 'passive_recovery',
                          label: t('Passive recovery only'),
                        },
                      ]}
                      value={field.value}
                      onValueChange={field.onChange}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='scheduled_all'>
                            {t('Scheduled full test')}
                          </SelectItem>
                          <SelectItem value='passive_recovery'>
                            {t('Passive recovery only')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {t(
                        'Scheduled full test probes non-manually-disabled channels; passive recovery only checks auto-disabled channels after real request failures.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='monitor_setting.auto_test_channel_minutes'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Test interval (minutes)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        step={1}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {channelTestMode === 'passive_recovery'
                        ? t(
                            'How frequently the system checks auto-disabled channels for recovery'
                          )
                        : t('How frequently the system tests all channels')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='AutomaticEnableChannelEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Re-enable on success')}</FormLabel>
                      <FormDescription>
                        {t(
                          'Bring channels back online after successful checks'
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
            </div>
          </div>

          <Separator />

          <div className='flex min-w-0 flex-col gap-4'>
            <div className='flex flex-col gap-1'>
              <h4 className='text-sm font-medium'>{t('Auto-disable rules')}</h4>
            </div>
            <div className='grid min-w-0 gap-6 lg:grid-cols-2'>
              <FormField
                control={form.control}
                name='AutomaticDisableChannelEnabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Disable on failure')}</FormLabel>
                      <FormDescription>
                        {t('Automatically disable channels when tests fail')}
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
                name='ChannelDisableThreshold'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Disable threshold (seconds)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        step={1}
                        value={field.value}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Automatically disable channels exceeding this response time'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='AutomaticDisableStatusCodes'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Auto-disable status codes')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('e.g. 401, 403, 429, 500-599')}
                        value={field.value}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Accepts comma-separated status codes and inclusive ranges.'
                      )}{' '}
                      {autoDisableParsed.ok &&
                        autoDisableParsed.normalized &&
                        autoDisableParsed.normalized !== field.value.trim() && (
                          <span className='text-muted-foreground'>
                            {t('Normalized:')} {autoDisableParsed.normalized}
                          </span>
                        )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='AutomaticDisableKeywords'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Failure keywords')}</FormLabel>
                    <FormControl>
                      <Textarea
                        rows={6}
                        placeholder={t('one keyword per line')}
                        {...field}
                        onChange={(event) => field.onChange(event.target.value)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'If an upstream error contains any of these keywords (case insensitive), the channel will be disabled automatically.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
