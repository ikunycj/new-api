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
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import type { RealtimeDimensions } from '../../types'

/** Sentinel for "no filter". Select needs a non-empty string value, and the
 *  empty string is what the API uses to mean unfiltered, so the two cannot be
 *  the same token. */
export const REALTIME_FILTER_ALL = '__all__'

export interface RealtimeFilterValue {
  tokenId: number
  model: string
}

interface RealtimeFilterBarProps {
  dimensions: RealtimeDimensions | null
  value: RealtimeFilterValue
  onChange: (next: RealtimeFilterValue) => void
}

/**
 * Key and model pickers for the realtime panel.
 *
 * Only combinations with live traffic are offered, because the options come
 * from the counters themselves rather than from the user's key list: a key that
 * has sent nothing this window has no data to show and selecting it would only
 * produce an empty chart.
 */
export function RealtimeFilterBar(props: RealtimeFilterBarProps) {
  const { t } = useTranslation()
  const tokens = props.dimensions?.tokens ?? []
  const models = props.dimensions?.models ?? []

  // With nothing to choose between, the controls would be decoration. They stay
  // hidden until the account actually has more than one key or model in play.
  if (tokens.length === 0 && models.length === 0) return null

  const tokenValue =
    props.value.tokenId > 0 ? String(props.value.tokenId) : REALTIME_FILTER_ALL
  const modelValue = props.value.model || REALTIME_FILTER_ALL

  const selectedToken = tokens.find((o) => o.token_id === props.value.tokenId)
  // A key can serve traffic and then be deleted. Falling back to its id keeps
  // the row readable instead of rendering an empty trigger.
  const tokenLabel =
    props.value.tokenId > 0
      ? selectedToken?.token_name || `#${props.value.tokenId}`
      : t('All keys')
  const modelLabel = props.value.model || t('All models')

  return (
    <div className='flex flex-wrap items-center gap-2'>
      {tokens.length > 0 ? (
        <Select
          value={tokenValue}
          onValueChange={(next) =>
            props.onChange({
              ...props.value,
              tokenId: next && next !== REALTIME_FILTER_ALL ? Number(next) : 0,
            })
          }
        >
          <SelectTrigger className='h-7 w-auto min-w-28 text-xs'>
            <SelectValue>{tokenLabel}</SelectValue>
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value={REALTIME_FILTER_ALL}>
                {t('All keys')}
              </SelectItem>
              {tokens.map((option) => (
                <SelectItem
                  key={option.token_id}
                  value={String(option.token_id)}
                >
                  {option.token_name || `#${option.token_id}`}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      ) : null}

      {models.length > 0 ? (
        <Select
          value={modelValue}
          onValueChange={(next) =>
            props.onChange({
              ...props.value,
              model: next && next !== REALTIME_FILTER_ALL ? next : '',
            })
          }
        >
          <SelectTrigger className='h-7 w-auto min-w-32 text-xs'>
            <SelectValue>{modelLabel}</SelectValue>
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value={REALTIME_FILTER_ALL}>
                {t('All models')}
              </SelectItem>
              {models.map((option) => (
                <SelectItem key={option.model} value={option.model ?? ''}>
                  {option.model}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      ) : null}

      {props.dimensions?.truncated ? (
        // The breakdown hit its ring cap. Saying so matters because the totals
        // above stay complete while these lists do not, so a user comparing the
        // two would otherwise see an unexplained gap.
        <span
          className='text-muted-foreground/70 text-[11px]'
          title={t(
            'Too many key and model combinations are active to break them all down. Totals remain complete.'
          )}
        >
          {t('Partial breakdown')}
        </span>
      ) : null}
    </div>
  )
}
