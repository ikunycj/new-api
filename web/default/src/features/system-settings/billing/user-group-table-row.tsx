/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { RotateCcw, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { MultiSelect, type Option } from '@/components/multi-select'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { TableCell, TableRow } from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatTimestampToDate } from '@/lib/format'

import type {
  UpdateUserGroupRequest,
  UserGroupSummary,
} from './user-groups-api'

const ALL_PRICING_GROUPS = '*'

type UserGroupTableRowProps = {
  group: UserGroupSummary
  pricingGroupOptions: Option[]
  pricingGroupsUnavailable: boolean
  disabled: boolean
  saving: boolean
  onUpdate: (request: UpdateUserGroupRequest) => Promise<UserGroupSummary>
  onEdit: () => void
  onDelete: () => void
}

function getPricingGroupSelection(group: UserGroupSummary): string[] {
  const selected = group.pricing_groups ?? []
  if (
    group.pricing_groups_all === true ||
    selected.includes(ALL_PRICING_GROUPS) ||
    selected.length === 0
  ) {
    return [ALL_PRICING_GROUPS]
  }
  return selected
}

function selectionsMatch(left: string[], right: string[]): boolean {
  if (left.length !== right.length) return false
  return left.every((value, index) => value === right[index])
}

export function UserGroupTableRow(props: UserGroupTableRowProps) {
  const isDefault = props.group.name === 'default'
  const originalPricingGroups = getPricingGroupSelection(props.group)
  const [draftName, setDraftName] = useState(props.group.name)
  const [draftTopupRatio, setDraftTopupRatio] = useState(
    String(props.group.topup_ratio ?? 1)
  )
  const [draftPricingGroups, setDraftPricingGroups] = useState(
    originalPricingGroups
  )

  const options = useMemo<Option[]>(
    () => [
      { label: '全部', value: ALL_PRICING_GROUPS },
      ...props.pricingGroupOptions,
    ],
    [props.pricingGroupOptions]
  )

  const trimmedName = draftName.trim()
  const parsedTopupRatio = Number(draftTopupRatio)
  const pricingGroupsAll = draftPricingGroups.includes(ALL_PRICING_GROUPS)
  let pricingGroupsLabel = '未选择'
  if (pricingGroupsAll) {
    pricingGroupsLabel = '全部'
  } else if (draftPricingGroups.length > 0) {
    pricingGroupsLabel = draftPricingGroups.join('、')
  }
  const isDirty =
    trimmedName !== props.group.name ||
    parsedTopupRatio !== (props.group.topup_ratio ?? 1) ||
    !selectionsMatch(draftPricingGroups, originalPricingGroups)

  const resetDraft = () => {
    setDraftName(props.group.name)
    setDraftTopupRatio(String(props.group.topup_ratio ?? 1))
    setDraftPricingGroups(originalPricingGroups)
  }

  const saveDraft = async () => {
    if (!trimmedName) {
      toast.error('请输入用户分组名')
      return
    }
    if (
      draftTopupRatio.trim() === '' ||
      !Number.isFinite(parsedTopupRatio) ||
      parsedTopupRatio < 0
    ) {
      toast.error('请输入有效的充值倍率')
      return
    }
    if (draftPricingGroups.length === 0) {
      toast.error('请选择至少一个定价分组')
      return
    }

    try {
      await props.onUpdate({
        name: isDefault ? props.group.name : trimmedName,
        topup_ratio: parsedTopupRatio,
        pricing_groups: pricingGroupsAll
          ? [ALL_PRICING_GROUPS]
          : draftPricingGroups,
        pricing_groups_all: pricingGroupsAll,
      })
    } catch {
      // The mutation owns error reporting; keep the draft for correction.
    }
  }

  return (
    <TableRow>
      <TableCell className='w-44 min-w-44'>
        <Input
          value={draftName}
          maxLength={64}
          aria-label={`用户分组 ${props.group.name} 的名称`}
          title={isDefault ? 'default 分组名称不可修改' : undefined}
          disabled={props.disabled || isDefault}
          onChange={(event) => setDraftName(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && isDirty) {
              event.preventDefault()
              void saveDraft()
            }
            if (event.key === 'Escape') resetDraft()
          }}
          className='h-7'
        />
      </TableCell>
      <TableCell className='w-32 min-w-32'>
        <dl className='flex flex-col text-xs leading-4 tabular-nums'>
          <div className='flex items-baseline gap-1.5 whitespace-nowrap'>
            <dt className='text-muted-foreground'>日活</dt>
            <dd className='font-medium'>
              {props.group.active_today.toLocaleString()}/
              {props.group.user_count.toLocaleString()}
            </dd>
          </div>
          <div className='flex items-baseline gap-1.5 whitespace-nowrap'>
            <dt className='text-muted-foreground'>月活</dt>
            <dd className='font-medium'>
              {props.group.active_month.toLocaleString()}/
              {props.group.user_count.toLocaleString()}
            </dd>
          </div>
        </dl>
      </TableCell>
      <TableCell className='w-28 min-w-28'>
        <Input
          type='number'
          min={0}
          step={0.1}
          value={draftTopupRatio}
          aria-label={`用户分组 ${props.group.name} 的充值倍率`}
          disabled={props.disabled}
          onChange={(event) => setDraftTopupRatio(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && isDirty) {
              event.preventDefault()
              void saveDraft()
            }
            if (event.key === 'Escape') resetDraft()
          }}
          className='h-7 w-24'
        />
      </TableCell>
      <TableCell className='w-64 max-w-64 min-w-64'>
        <TooltipProvider delay={500}>
          <Tooltip>
            <TooltipTrigger render={<div className='w-64 max-w-64' />}>
              <MultiSelect
                options={options}
                selected={draftPricingGroups}
                onChange={(groups) => {
                  if (
                    groups.includes(ALL_PRICING_GROUPS) &&
                    !draftPricingGroups.includes(ALL_PRICING_GROUPS)
                  ) {
                    setDraftPricingGroups([ALL_PRICING_GROUPS])
                    return
                  }
                  if (
                    groups.includes(ALL_PRICING_GROUPS) &&
                    draftPricingGroups.includes(ALL_PRICING_GROUPS)
                  ) {
                    setDraftPricingGroups(
                      groups.filter((group) => group !== ALL_PRICING_GROUPS)
                    )
                    return
                  }
                  setDraftPricingGroups(groups)
                }}
                placeholder='选择定价分组'
                emptyText='暂无定价分组'
                disabled={props.disabled || props.pricingGroupsUnavailable}
                renderSelectedSummary={() => (
                  <span className='block max-w-44 truncate font-sans'>
                    {pricingGroupsLabel}
                  </span>
                )}
                className='min-h-7 max-w-64 py-0.5'
              />
            </TooltipTrigger>
            <TooltipContent className='max-w-sm break-words whitespace-normal'>
              {pricingGroupsLabel}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell className='text-muted-foreground'>
        {formatTimestampToDate(props.group.created_at)}
      </TableCell>
      <TableCell className='text-muted-foreground'>
        {formatTimestampToDate(props.group.updated_at)}
      </TableCell>
      <TableCell className='text-right'>
        <div className='flex items-center justify-end gap-1'>
          <Button
            variant='outline'
            size='sm'
            disabled={props.disabled || !isDirty}
            onClick={() => void saveDraft()}
          >
            {props.saving && <Spinner />}
            保存
          </Button>
          {isDirty && (
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={`撤销用户分组 ${props.group.name} 的未保存修改`}
              title='撤销未保存的修改'
              disabled={props.disabled}
              onClick={resetDraft}
            >
              <RotateCcw />
            </Button>
          )}
          <Button
            variant='ghost'
            size='sm'
            aria-label={`编辑用户分组 ${props.group.name}`}
            disabled={props.disabled}
            onClick={props.onEdit}
          >
            编辑
          </Button>
          {!isDefault && (
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={`删除用户分组 ${props.group.name}`}
              title={`删除用户分组 ${props.group.name}`}
              disabled={props.disabled}
              onClick={props.onDelete}
            >
              <Trash2 />
            </Button>
          )}
          {isDefault && <Badge variant='outline'>默认</Badge>}
        </div>
      </TableCell>
    </TableRow>
  )
}
