import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo } from 'react'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const schema = z.object({
  RelayTimeout: z.coerce.number().int().min(0).max(3600),
  StreamingTimeout: z.coerce.number().int().min(1).max(3600),
  RelayIdleConnTimeout: z.coerce.number().int().min(0).max(3600),
  StreamClientWriteTimeout: z.coerce.number().int().min(1).max(600),
  ShutdownTimeoutSeconds: z.coerce.number().int().min(1).max(900),
})

type Values = z.infer<typeof schema>
type Props = { defaultValues: Values }

export function TimeoutSettingsSection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const defaults = useMemo(() => props.defaultValues, [props.defaultValues])
  const form = useForm<z.input<typeof schema>, unknown, Values>({ resolver: zodResolver(schema), defaultValues: defaults })

  useEffect(() => form.reset(defaults), [defaults, form])

  const onSubmit = async (values: Values) => {
    for (const [key, value] of Object.entries(values)) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  return (
    <SettingsSection title={t('Timeouts')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions onSave={form.handleSubmit(onSubmit)} isSaving={updateOption.isPending} />
          <p className='text-muted-foreground text-sm'>{t('Changes apply immediately to new requests. Gateway proxy timeouts remain managed by the host configuration.')}</p>
          <TimeoutField form={form} name='RelayTimeout' label={t('Upstream total timeout (seconds)')} hint={t('0 means unlimited.')} min={0} max={3600} />
          <TimeoutField form={form} name='StreamingTimeout' label={t('Streaming idle timeout (seconds)')} min={1} max={3600} />
          <TimeoutField form={form} name='RelayIdleConnTimeout' label={t('Upstream idle connection timeout (seconds)')} hint={t('0 means unlimited.')} min={0} max={3600} />
          <TimeoutField form={form} name='StreamClientWriteTimeout' label={t('Client write timeout (seconds)')} min={1} max={600} />
          <TimeoutField form={form} name='ShutdownTimeoutSeconds' label={t('Shutdown grace period (seconds)')} min={1} max={900} />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}

function TimeoutField({ form, name, label, hint, min, max }: { form: UseFormReturn<z.input<typeof schema>, unknown, Values>; name: keyof Values; label: string; hint?: string; min: number; max: number }) {
  return (
    <FormField control={form.control} name={name} render={({ field }) => (
      <FormItem>
        <FormLabel>{label}</FormLabel>
        <FormControl><Input type='number' min={min} max={max} step={1} {...safeNumberFieldProps(field)} /></FormControl>
        <p className='text-muted-foreground text-xs'>{hint ?? `${min}-${max} seconds`}</p>
        <FormMessage />
      </FormItem>
    )} />
  )
}
