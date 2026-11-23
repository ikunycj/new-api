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
import { skipToken, useQuery } from '@tanstack/react-query'
import { useId, useMemo } from 'react'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Switch } from '@/components/ui/switch'

import { getPricingGroupModels } from '../api'

type PricingGroupModelAllowlistFieldProps = {
  groupName: string | null
  value: string[] | null
  onChange: (models: string[] | null) => void
}

export function PricingGroupModelAllowlistField(
  props: PricingGroupModelAllowlistFieldProps
) {
  const fieldId = useId()
  const groupName = props.groupName
  const enabled = props.value !== null
  const modelsQuery = useQuery({
    queryKey: ['pricing-group-models', groupName],
    queryFn:
      enabled && groupName !== null
        ? () => getPricingGroupModels(groupName)
        : skipToken,
  })
  const options = useMemo(
    () =>
      (modelsQuery.data ?? []).map((model) => ({ label: model, value: model })),
    [modelsQuery.data]
  )

  return (
    <FieldGroup className='gap-3 rounded-md border p-3'>
      <Field orientation='horizontal' className='items-center'>
        <FieldContent>
          <FieldLabel htmlFor={fieldId}>模型白名单</FieldLabel>
          <FieldDescription id={`${fieldId}-description`} className='text-xs'>
            {groupName === null
              ? '请先保存分组，再配置模型白名单。'
              : '从本组启用渠道支持的模型中选择，限制本组可列出和调用的模型。'}
          </FieldDescription>
        </FieldContent>
        <Switch
          id={fieldId}
          aria-describedby={`${fieldId}-description`}
          checked={enabled}
          disabled={groupName === null && !enabled}
          onCheckedChange={(checked) => props.onChange(checked ? [] : null)}
        />
      </Field>
      {props.value !== null && (
        <Field data-invalid={props.value.length === 0}>
          <FieldLabel htmlFor={`${fieldId}-models`}>允许的模型</FieldLabel>
          <MultiSelect
            id={`${fieldId}-models`}
            options={options}
            selected={props.value}
            onChange={(models) => props.onChange([...models].sort())}
            placeholder='搜索允许的模型'
            emptyText={
              modelsQuery.isError ? '模型列表加载失败' : '没有匹配的模型'
            }
            maxVisibleChips={4}
            hideUnlistedSelectedItems
          />
          {modelsQuery.isLoading && (
            <FieldDescription role='status'>正在加载模型…</FieldDescription>
          )}
          {modelsQuery.isError && (
            <div className='flex flex-wrap items-center gap-2'>
              <FieldError>模型列表加载失败</FieldError>
              <Button
                type='button'
                variant='outline'
                size='xs'
                onClick={() => void modelsQuery.refetch()}
              >
                重试
              </Button>
            </div>
          )}
          {modelsQuery.isSuccess && options.length === 0 && (
            <FieldDescription>本组暂无启用渠道提供模型。</FieldDescription>
          )}
          {props.value.length === 0 && (
            <FieldError>请至少选择一个模型。</FieldError>
          )}
        </Field>
      )}
    </FieldGroup>
  )
}
