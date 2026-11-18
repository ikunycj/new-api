import { useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { Pencil } from 'lucide-react'
import { useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'

import { updateChannel } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { channelsQueryKeys, parseModelsList } from '../lib'
import type { Channel } from '../types'

export function TestModelEditor({ channel }: { channel: Channel }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const listId = useId()
  const testModel = channel.test_model?.trim() ?? ''
  const [draft, setDraft] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const value = draft?.trim() ?? ''
  const canSave = !saving && value !== '' && value !== testModel
  const label = t('Test Model')

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canSave) return

    setSaving(true)
    try {
      const response = await updateChannel(channel.id, { test_model: value })
      if (!response.success) {
        toast.error(response.message || t(ERROR_MESSAGES.UPDATE_FAILED))
        return
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() }),
        queryClient.invalidateQueries({
          queryKey: channelsQueryKeys.detail(channel.id),
        }),
      ])
      setDraft(null)
      toast.success(
        t('{{field}} updated to {{value}}', { field: label, value })
      )
    } catch (error) {
      const message = isAxiosError<{ message?: string }>(error)
        ? error.response?.data?.message
        : undefined
      toast.error(message || t(ERROR_MESSAGES.UPDATE_FAILED))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Popover
      open={draft !== null}
      onOpenChange={(open) => {
        if (!saving) setDraft(open ? testModel : null)
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant='ghost'
            size='xs'
            className='max-w-full min-w-0 justify-start px-1'
            title={testModel || label}
          />
        }
      >
        <span className='sr-only'>
          {t('Edit')}: {label},{' '}
        </span>
        <span className='truncate font-mono font-normal'>
          {testModel || '-'}
        </span>
        <Pencil aria-hidden className='text-muted-foreground' />
      </PopoverTrigger>
      <PopoverContent align='start'>
        <PopoverTitle>{label}</PopoverTitle>
        <form className='grid gap-2' onSubmit={save}>
          <Input
            aria-label={label}
            list={listId}
            placeholder={t('Select or enter model name')}
            value={draft ?? ''}
            onChange={(event) => setDraft(event.target.value)}
            disabled={saving}
          />
          {draft !== null && (
            <datalist id={listId}>
              {parseModelsList(channel.models).map((model) => (
                <option key={model} value={model} />
              ))}
            </datalist>
          )}
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              disabled={saving}
              onClick={() => setDraft(null)}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' size='sm' disabled={!canSave}>
              {t('Save')}
            </Button>
          </div>
        </form>
      </PopoverContent>
    </Popover>
  )
}
