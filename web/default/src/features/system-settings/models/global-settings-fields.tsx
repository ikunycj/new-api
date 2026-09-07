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
import { Add01Icon, Delete02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { GripVertical } from 'lucide-react'
import { Reorder, useDragControls } from 'motion/react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect, type Option } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { ChatCompletionsToResponsesPolicyValues } from './global-settings-form'

type StringListEditorProps = {
  value: string[]
  onChange: (value: string[]) => void
  placeholder: string
  addLabel: string
  emptyLabel: string
  disabled?: boolean
  validate?: (value: string) => boolean
  invalidLabel?: string
  inputAriaLabel: (index: number) => string
  deleteAriaLabel: (value: string) => string
}

export function StringListEditor(props: StringListEditorProps) {
  const [draft, setDraft] = useState('')
  const draftInputRef = useRef<HTMLInputElement>(null)
  const rowKeys = useRef<string[]>([])
  const nextRowKey = useRef(0)

  while (rowKeys.current.length < props.value.length) {
    rowKeys.current.push(`row-${nextRowKey.current++}`)
  }
  if (rowKeys.current.length > props.value.length) {
    rowKeys.current.length = props.value.length
  }

  const addItem = () => {
    const value = draft.trim()
    if (!value) {
      draftInputRef.current?.focus()
      return
    }
    if (props.value.includes(value)) return
    props.onChange([...props.value, value])
    setDraft('')
  }

  const updateItem = (index: number, value: string) => {
    props.onChange(
      props.value.map((item, itemIndex) => (itemIndex === index ? value : item))
    )
  }

  const removeItem = (index: number) => {
    rowKeys.current.splice(index, 1)
    props.onChange(props.value.filter((_, itemIndex) => itemIndex !== index))
  }

  return (
    <div className='space-y-2'>
      {props.value.length === 0 ? (
        <p className='text-muted-foreground rounded-md border border-dashed px-3 py-2 text-sm'>
          {props.emptyLabel}
        </p>
      ) : (
        <div className='space-y-2'>
          {props.value.map((item, index) => {
            const isInvalid = props.validate ? !props.validate(item) : false
            return (
              <div key={rowKeys.current[index]} className='space-y-1'>
                <div className='flex min-w-0 items-center gap-2'>
                  <Input
                    value={item}
                    placeholder={props.placeholder}
                    disabled={props.disabled}
                    aria-label={props.inputAriaLabel(index)}
                    aria-invalid={isInvalid || undefined}
                    onChange={(event) => updateItem(index, event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter') {
                        event.preventDefault()
                      }
                    }}
                  />
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon-sm'
                          disabled={props.disabled}
                          aria-label={props.deleteAriaLabel(item)}
                          onClick={() => removeItem(index)}
                          className='text-muted-foreground hover:text-destructive'
                        />
                      }
                    >
                      <HugeiconsIcon icon={Delete02Icon} aria-hidden='true' />
                    </TooltipTrigger>
                    <TooltipContent>
                      {props.deleteAriaLabel(item)}
                    </TooltipContent>
                  </Tooltip>
                </div>
                {isInvalid && props.invalidLabel && (
                  <p className='text-destructive pl-9 text-xs'>
                    {props.invalidLabel}
                  </p>
                )}
              </div>
            )
          })}
        </div>
      )}
      <div className='flex min-w-0 items-center gap-2'>
        <Input
          ref={draftInputRef}
          value={draft}
          placeholder={props.placeholder}
          disabled={props.disabled}
          aria-label={props.placeholder}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault()
              addItem()
            }
          }}
        />
        <Button
          type='button'
          variant='outline'
          disabled={props.disabled}
          onClick={addItem}
        >
          <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
          {props.addLabel}
        </Button>
      </div>
    </div>
  )
}

type PreferredModelsEditorProps = {
  value: string[]
  onChange: (value: string[]) => void
  placeholder: string
  emptyLabel: string
  deleteAriaLabel: (value: string) => string
  dragAriaLabel: (value: string) => string
  disabled?: boolean
}

function PreferredModelCard(props: {
  model: string
  index: number
  total: number
  onRemove: () => void
  onMove: (direction: -1 | 1) => void
  deleteAriaLabel: string
  dragAriaLabel: string
  disabled?: boolean
}) {
  const dragControls = useDragControls()

  return (
    <Reorder.Item
      as='div'
      value={props.model}
      dragListener={false}
      dragControls={dragControls}
      className='bg-muted/30 flex min-w-0 items-center gap-2 rounded-md border px-2 py-1.5 transition-colors'
      whileDrag={{
        scale: 1.02,
        zIndex: 20,
        boxShadow:
          '0 12px 24px rgb(0 0 0 / 0.16), 0 0 0 2px color-mix(in oklch, var(--primary) 30%, transparent)',
      }}
    >
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              variant='ghost'
              size='icon-sm'
              disabled={props.disabled || props.total < 2}
              className='text-muted-foreground size-7 shrink-0 cursor-grab touch-none active:cursor-grabbing'
              aria-label={props.dragAriaLabel}
              onPointerDown={(event) => {
                if (props.disabled || props.total < 2) return
                if (event.pointerType === 'mouse' && event.button !== 0) return
                dragControls.start(event)
              }}
              onKeyDown={(event) => {
                if (props.disabled) return
                if (event.key === 'ArrowUp') {
                  event.preventDefault()
                  props.onMove(-1)
                } else if (event.key === 'ArrowDown') {
                  event.preventDefault()
                  props.onMove(1)
                }
              }}
            />
          }
        >
          <GripVertical className='size-4' aria-hidden='true' />
        </TooltipTrigger>
        <TooltipContent>{props.dragAriaLabel}</TooltipContent>
      </Tooltip>
      <span className='text-muted-foreground w-5 shrink-0 text-center text-xs tabular-nums'>
        {props.index + 1}
      </span>
      <span className='min-w-0 flex-1 truncate text-sm'>{props.model}</span>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              variant='ghost'
              size='icon-sm'
              disabled={props.disabled}
              aria-label={props.deleteAriaLabel}
              onClick={props.onRemove}
              className='text-muted-foreground hover:text-destructive size-7 shrink-0'
            />
          }
        >
          <HugeiconsIcon icon={Delete02Icon} aria-hidden='true' />
        </TooltipTrigger>
        <TooltipContent>{props.deleteAriaLabel}</TooltipContent>
      </Tooltip>
    </Reorder.Item>
  )
}

export function PreferredModelsEditor(props: PreferredModelsEditorProps) {
  const { t } = useTranslation()

  const moveItem = (index: number, direction: -1 | 1) => {
    const targetIndex = index + direction
    if (targetIndex < 0 || targetIndex >= props.value.length) return
    const next = [...props.value]
    const [item] = next.splice(index, 1)
    next.splice(targetIndex, 0, item)
    props.onChange(next)
  }

  return (
    <div className='space-y-2'>
      <MultiSelect
        selected={props.value}
        options={[]}
        onChange={props.onChange}
        allowCreate
        placeholder={props.placeholder}
        createLabel={t('Add model "{{value}}"')}
        emptyText={t('No models added')}
        renderSelectedSummary={(values) =>
          t('{{count}} items', { count: values.length })
        }
        disabled={props.disabled}
      />

      {props.value.length === 0 ? (
        <p className='text-muted-foreground rounded-md border border-dashed px-3 py-2 text-sm'>
          {props.emptyLabel}
        </p>
      ) : (
        <Reorder.Group
          axis='y'
          values={props.value}
          onReorder={props.onChange}
          className='space-y-2'
        >
          {props.value.map((model, index) => (
            <PreferredModelCard
              key={model}
              model={model}
              index={index}
              total={props.value.length}
              onRemove={() =>
                props.onChange(props.value.filter((_, i) => i !== index))
              }
              onMove={(direction) => moveItem(index, direction)}
              deleteAriaLabel={props.deleteAriaLabel(model)}
              dragAriaLabel={props.dragAriaLabel(model)}
              disabled={props.disabled}
            />
          ))}
        </Reorder.Group>
      )}
    </div>
  )
}

export type GlobalPolicyEditorProps = {
  value: ChatCompletionsToResponsesPolicyValues
  onChange: (value: ChatCompletionsToResponsesPolicyValues) => void
  channelOptions: Option[]
  channelTypeOptions: Option[]
  disabled?: boolean
}

export function GlobalPolicyEditor(props: GlobalPolicyEditorProps) {
  const { t } = useTranslation()
  const policy = props.value
  const setPolicy = (
    changes: Partial<ChatCompletionsToResponsesPolicyValues>
  ) => props.onChange({ ...policy, ...changes })

  const channelIds = policy.channel_ids.map(String)
  const channelTypes = policy.channel_types.map(String)

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-4 rounded-lg border px-3 py-2.5'>
        <div className='min-w-0 space-y-0.5'>
          <p className='text-sm font-medium'>{t('Enable this policy')}</p>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Convert matching Chat Completions requests to the Responses API.'
            )}
          </p>
        </div>
        <Switch
          checked={policy.enabled}
          disabled={props.disabled}
          aria-label={t('Enable this policy')}
          onCheckedChange={(enabled) => setPolicy({ enabled })}
        />
      </div>

      <div className='space-y-2'>
        <p className='text-sm font-medium'>{t('Apply to')}</p>
        <ToggleGroup
          value={[policy.all_channels ? 'all' : 'specific']}
          onValueChange={(values) => {
            const next = values[0]
            if (next === 'all') setPolicy({ all_channels: true })
            if (next === 'specific') setPolicy({ all_channels: false })
          }}
          disabled={props.disabled}
          className='inline-flex h-8 w-fit max-w-full'
        >
          <ToggleGroupItem value='all' className='h-8 min-w-0 px-3 text-sm'>
            {t('All channels')}
          </ToggleGroupItem>
          <ToggleGroupItem
            value='specific'
            className='h-8 min-w-0 px-3 text-sm'
          >
            {t('Selected channels')}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      {!policy.all_channels && (
        <div className='grid gap-4 md:grid-cols-2'>
          <div className='space-y-2'>
            <p className='text-sm font-medium'>{t('Channels')}</p>
            <MultiSelect
              selected={channelIds}
              options={props.channelOptions}
              onChange={(values) =>
                setPolicy({
                  channel_ids: values
                    .map(Number)
                    .filter(
                      (value) => Number.isSafeInteger(value) && value > 0
                    ),
                })
              }
              allowCreate
              placeholder={t('Search by channel name or ID')}
              createLabel={t('Add channel ID "{{value}}"')}
              emptyText={t('No matching channels')}
              disabled={props.disabled}
              maxVisibleChips={4}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Leave empty to match by channel type only.')}
            </p>
          </div>
          <div className='space-y-2'>
            <p className='text-sm font-medium'>{t('Channel types')}</p>
            <MultiSelect
              selected={channelTypes}
              options={props.channelTypeOptions}
              onChange={(values) =>
                setPolicy({
                  channel_types: values
                    .map(Number)
                    .filter(
                      (value) => Number.isSafeInteger(value) && value > 0
                    ),
                })
              }
              placeholder={t('Select channel types')}
              emptyText={t('No matching channel types')}
              disabled={props.disabled}
              maxVisibleChips={4}
            />
            <p className='text-muted-foreground text-xs'>
              {t('A request matches when its channel ID or type is selected.')}
            </p>
          </div>
        </div>
      )}

      <div className='space-y-2'>
        <p className='text-sm font-medium'>{t('Model patterns')}</p>
        <StringListEditor
          value={policy.model_patterns}
          onChange={(model_patterns) => setPolicy({ model_patterns })}
          placeholder={t('Enter a model regular expression')}
          addLabel={t('Add pattern')}
          emptyLabel={t(
            'No model patterns. Add at least one to enable matching.'
          )}
          validate={(pattern) => {
            try {
              new RegExp(pattern)
              return pattern.trim().length > 0
            } catch {
              return false
            }
          }}
          invalidLabel={t('Enter a valid regular expression')}
          inputAriaLabel={(index) =>
            t('Model pattern {{index}}', { index: index + 1 })
          }
          deleteAriaLabel={(pattern) =>
            t('Delete {{value}}', { value: pattern })
          }
          disabled={props.disabled}
        />
      </div>
    </div>
  )
}
