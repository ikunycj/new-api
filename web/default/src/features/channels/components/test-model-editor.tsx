import { useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Input } from '@/components/ui/input'

import { updateChannel } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { channelsQueryKeys, parseModelsList } from '../lib'
import type { Channel } from '../types'

export function TestModelEditor({ channel }: { channel: Channel }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const listId = useId()
  const testModel = channel.test_model?.trim() ?? ''
  const [draft, setDraft] = useState(testModel)
  const [savedValue, setSavedValue] = useState(testModel)
  const [saving, setSaving] = useState(false)
  const savingRef = useRef(false)
  const value = draft.trim()
  const label = t('Test Model')

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setDraft(testModel)
    setSavedValue(testModel)
  }, [testModel])

  async function save(event?: FormEvent<HTMLFormElement>) {
    event?.preventDefault()
    if (savingRef.current) return
    if (value === savedValue) {
      if (draft !== savedValue) setDraft(savedValue)
      return
    }
    if (value === '') {
      setDraft(savedValue)
      if (savedValue !== '') {
        toast.error(t(ERROR_MESSAGES.REQUIRED_TEST_MODEL))
      }
      return
    }

    savingRef.current = true
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
      setDraft(value)
      setSavedValue(value)
      toast.success(
        t('{{field}} updated to {{value}}', { field: label, value })
      )
    } catch (error) {
      const message = isAxiosError<{ message?: string }>(error)
        ? error.response?.data?.message
        : undefined
      toast.error(message || t(ERROR_MESSAGES.UPDATE_FAILED))
    } finally {
      savingRef.current = false
      setSaving(false)
    }
  }

  return (
    <form className='w-full min-w-0' onSubmit={save}>
      <Input
        aria-label={label}
        list={listId}
        placeholder={t('Select or enter model name')}
        title={draft || label}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={() => void save()}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            event.preventDefault()
            setDraft(savedValue)
          }
        }}
        disabled={saving}
        className='h-7 rounded-md px-2 font-mono text-xs'
      />
      <datalist id={listId}>
        {parseModelsList(channel.models).map((model) => (
          <option key={model} value={model} />
        ))}
      </datalist>
    </form>
  )
}
