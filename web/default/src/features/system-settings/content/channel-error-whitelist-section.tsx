/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'

import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const ruleSchema = z.object({
  match_type: z.enum(['status', 'error']),
  match_value: z.string(),
  response_mode: z.enum(['passthrough', 'custom']),
  custom_code: z.string().optional(),
  custom_type: z.string().optional(),
  custom_message: z.string().optional(),
})

type ChannelErrorRule = z.infer<typeof ruleSchema>
type DraftRule = ChannelErrorRule & { id: string }

type ChannelErrorWhitelistSectionProps = { data: string }

function parseRules(data: string): { rules: DraftRule[]; error: string } {
  try {
    const parsed: unknown = JSON.parse(data || '[]')
    const rules = z.array(ruleSchema).parse(parsed)
    return {
      rules: rules.map((rule) => ({ ...rule, id: crypto.randomUUID() })),
      error: '',
    }
  } catch (error) {
    return { rules: [], error: String(error) }
  }
}

export function ChannelErrorWhitelistSection({
  data,
}: ChannelErrorWhitelistSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [draft, setDraft] = useState(() => parseRules(data))
  const rules = draft.rules

  useEffect(() => setDraft(parseRules(data)), [data])

  const updateRule = (index: number, patch: Partial<ChannelErrorRule>) => {
    setDraft((current) => ({
      ...current,
      rules: current.rules.map((rule, ruleIndex) =>
        ruleIndex === index ? { ...rule, ...patch } : rule
      ),
    }))
  }

  const handleSave = async () => {
    const normalized = rules.map((rule) => ({
      match_type: rule.match_type,
      match_value: rule.match_value,
      response_mode: rule.response_mode,
      ...(rule.response_mode === 'custom'
        ? {
            custom_code: rule.custom_code?.trim() ?? '',
            custom_type: rule.custom_type?.trim() ?? '',
            custom_message: rule.custom_message?.trim() ?? '',
          }
        : {}),
    }))
    if (normalized.some((rule) => !rule.match_value.trim())) {
      toast.error(t('Each whitelist rule needs a match value'))
      return
    }
    if (normalized.some((rule) => rule.response_mode === 'custom' && (!rule.custom_code || !rule.custom_type || !rule.custom_message))) {
      toast.error(t('All three custom error fields are required'))
      return
    }
    const keys = normalized.map(
      (rule) => `${rule.match_type}\u0000${rule.match_value}`
    )
    if (new Set(keys).size !== keys.length) {
      toast.error(t('Duplicate whitelist rules are not allowed'))
      return
    }
    updateOption.mutate({
      key: 'console_setting.channel_error_whitelist',
      value: JSON.stringify(normalized),
    })
  }

  return (
    <SettingsSection title={t('Channel Error Whitelist')}>
      <div className='text-muted-foreground space-y-1 text-sm'>
        <p>{t('HTTP status rules match the final 4xx or 5xx status exactly.')}</p>
        <p>{t('Error rules use a case-insensitive fuzzy match against error.code, error.type, or error.message.')}</p>
        <p>{t('Channel identity is always displayed as channel 1 / auto.')}</p>
      </div>
      {draft.error && (
        <p role='alert' className='text-destructive text-sm'>
          {t('Failed to load')}: {draft.error}
        </p>
      )}
      <div className='space-y-3'>
        {rules.map((rule, index) => (
          <div
            className='border-border bg-muted/20 grid gap-3 rounded-lg border p-3 md:grid-cols-[9rem_minmax(0,1fr)_9rem_minmax(0,1fr)_auto] md:items-start'
            key={rule.id}
          >
            <NativeSelect
              aria-label={t('Whitelist match type')}
              value={rule.match_type}
              disabled={updateOption.isPending}
              onChange={(event) =>
                updateRule(index, {
                  match_type: event.target.value as ChannelErrorRule['match_type'],
                })
              }
            >
              <NativeSelectOption value='status'>
                {t('HTTP status code')}
              </NativeSelectOption>
              <NativeSelectOption value='error'>
                {t('Error field fuzzy match')}
              </NativeSelectOption>
            </NativeSelect>
            <Input
              aria-label={t('Whitelist match value')}
              placeholder={
                rule.match_type === 'status'
                  ? t('For example: 429')
                  : t('For example: quota or invalid_request')
              }
              value={rule.match_value}
              maxLength={512}
              disabled={updateOption.isPending}
              onChange={(event) =>
                updateRule(index, { match_value: event.target.value })
              }
            />
            <NativeSelect
              aria-label={t('Whitelist response mode')}
              value={rule.response_mode}
              disabled={updateOption.isPending}
              onChange={(event) =>
                updateRule(index, {
                  response_mode: event.target.value as ChannelErrorRule['response_mode'],
                })
              }
            >
              <NativeSelectOption value='passthrough'>
                {t('Pass through approved message')}
              </NativeSelectOption>
              <NativeSelectOption value='custom'>
                {t('Use custom response')}
              </NativeSelectOption>
            </NativeSelect>
            {rule.response_mode === 'custom' ? (
              <div className='grid gap-2'>
                <Input
                  aria-label={t('Custom error code')}
                  placeholder={t('For example: service_error')}
                  value={rule.custom_code ?? ''}
                  maxLength={2000}
                  disabled={updateOption.isPending}
                  onChange={(event) => updateRule(index, { custom_code: event.target.value })}
                />
                <Input
                  aria-label={t('Custom error type')}
                  placeholder={t('For example: server_error')}
                  value={rule.custom_type ?? ''}
                  maxLength={2000}
                  disabled={updateOption.isPending}
                  onChange={(event) => updateRule(index, { custom_type: event.target.value })}
                />
                <Textarea
                  aria-label={t('Custom error message')}
                  className='min-h-8'
                  placeholder={t('Message shown to the user')}
                  value={rule.custom_message ?? ''}
                  maxLength={2000}
                  disabled={updateOption.isPending}
                  onChange={(event) => updateRule(index, { custom_message: event.target.value })}
                />
              </div>
            ) : (
              <div className='text-muted-foreground px-2 py-2 text-sm'>
                {t('The original error.code, error.type, and error.message are shown.')}
              </div>
            )}
            <Button
              aria-label={t('Remove whitelist rule')}
              onClick={() =>
                setDraft((current) => ({
                  ...current,
                  rules: current.rules.filter((item) => item.id !== rule.id),
                }))
              }
              size='icon'
              type='button'
              variant='ghost'
              disabled={updateOption.isPending}
            >
              <Trash2 aria-hidden='true' className='size-4' />
            </Button>
          </div>
        ))}
      </div>
      <div className='flex flex-wrap gap-2'>
        <Button
          onClick={() =>
            setDraft((current) => ({
              ...current,
              rules: [...current.rules, {
                id: crypto.randomUUID(),
                match_type: 'status',
                match_value: '',
                response_mode: 'custom',
                custom_code: 'service_error',
                custom_type: 'server_error',
                custom_message: '',
              }],
            }))
          }
          disabled={Boolean(draft.error) || updateOption.isPending || rules.length >= 200}
          type='button'
          variant='outline'
        >
          <Plus aria-hidden='true' className='mr-2 size-4' />
          {t('Add whitelist rule')}
        </Button>
        <Button disabled={Boolean(draft.error) || updateOption.isPending} onClick={handleSave} type='button'>
          <Save aria-hidden='true' className='mr-2 size-4' />
          {t('Save whitelist')}
        </Button>
      </div>
    </SettingsSection>
  )
}
