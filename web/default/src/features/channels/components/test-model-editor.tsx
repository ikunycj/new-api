import { useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { TruncatedText } from '@/components/truncated-text'
import { Input } from '@/components/ui/input'
import { formatPercent } from '@/lib/format'

import { updateChannel } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { channelsQueryKeys, parseModelsList } from '../lib'
import type { Channel } from '../types'

export function TestModelEditor({
  channel,
  canWrite,
}: {
  channel: Channel
  canWrite: boolean
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const listId = useId()
  const testModel = channel.test_model?.trim() ?? ''
  const [draft, setDraft] = useState(testModel)
  const [savedValue, setSavedValue] = useState(testModel)
  const [saving, setSaving] = useState(false)
  const savingRef = useRef(false)
  const value = canWrite ? draft.trim() : testModel
  const label = t('Test Model')
  const hasProbe = channel.iq_probe_at > 0
  const confidences = channel.iq_probe_model_confidences
  const prediction = channel.iq_probe_prediction
  const topConfidence = channel.iq_probe_confidence
  const draftConfidence = Object.hasOwn(confidences, value)
    ? confidences[value]
    : 0
  const modelConfidence =
    value === testModel ? channel.iq_test_confidence : draftConfidence
  const confidence = hasProbe ? formatPercent(modelConfidence * 100) : '-'

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
    <div className='flex w-full min-w-0 flex-col gap-1 text-start [contain:inline-size]'>
      <div className='flex min-w-0 items-center gap-1.5'>
        {canWrite ? (
          <form className='min-w-0 flex-1' onSubmit={save}>
            <Input
              aria-label={label}
              aria-describedby={`${listId}-confidence`}
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
        ) : (
          <TruncatedText
            text={testModel || '-'}
            className='font-mono text-xs'
          />
        )}
        <span
          id={`${listId}-confidence`}
          className='shrink-0 text-xs tabular-nums'
          title='当前测试模型的置信度'
        >
          <span className='sr-only'>当前测试模型的置信度：</span>
          {confidence}
        </span>
      </div>
      {hasProbe && prediction && (
        <div
          className='text-muted-foreground flex min-w-0 items-baseline gap-1 text-[11px]/4! *:text-[11px]/4!'
          title={`最近一次指纹测试最高置信度模型：${prediction}`}
        >
          <span className='sr-only'>最近一次指纹测试最高置信度模型：</span>
          <span className='truncate font-mono'>{prediction}</span>
          <span className='shrink-0 tabular-nums'>
            {formatPercent(topConfidence == null ? null : topConfidence * 100)}
          </span>
        </div>
      )}
    </div>
  )
}
