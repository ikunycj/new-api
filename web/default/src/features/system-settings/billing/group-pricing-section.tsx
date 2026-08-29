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
import { useCallback, useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'

import { useUpdateOption } from '../hooks/use-update-option'
import {
  GroupPricingWorkspace,
  type GroupPricingValues,
} from '../models/group-pricing-workspace'
import type { BillingSettings } from '../types'

const DEFAULT_GROUP_VALUES: GroupPricingValues = {
  GroupRatio: '{}',
  TopupGroupRatio: '{}',
  UserUsableGroups: '{}',
  GroupGroupRatio: '{}',
  AutoGroups: '[]',
  DefaultUseAutoGroup: false,
  GroupSpecialUsableGroup: '{}',
}

export function BillingGroupPricingSection({
  settings,
}: {
  settings: BillingSettings
}) {
  const updateOption = useUpdateOption()
  const values = useMemo<GroupPricingValues>(
    () => ({
      GroupRatio: settings.GroupRatio ?? DEFAULT_GROUP_VALUES.GroupRatio,
      TopupGroupRatio:
        settings.TopupGroupRatio ?? DEFAULT_GROUP_VALUES.TopupGroupRatio,
      UserUsableGroups:
        settings.UserUsableGroups ?? DEFAULT_GROUP_VALUES.UserUsableGroups,
      GroupGroupRatio:
        settings.GroupGroupRatio ?? DEFAULT_GROUP_VALUES.GroupGroupRatio,
      AutoGroups: settings.AutoGroups ?? DEFAULT_GROUP_VALUES.AutoGroups,
      DefaultUseAutoGroup: settings.DefaultUseAutoGroup,
      GroupSpecialUsableGroup:
        settings['group_ratio_setting.group_special_usable_group'] ??
        DEFAULT_GROUP_VALUES.GroupSpecialUsableGroup,
    }),
    [settings]
  )
  const form = useForm<GroupPricingValues>({ defaultValues: values })

  useEffect(() => {
    form.reset(values)
  }, [form, values])

  const save = useCallback(
    async (nextValues: GroupPricingValues) => {
      const updates: Array<[string, string | boolean]> = [
        ['GroupRatio', nextValues.GroupRatio],
        ['TopupGroupRatio', nextValues.TopupGroupRatio],
        ['UserUsableGroups', nextValues.UserUsableGroups],
        ['GroupGroupRatio', nextValues.GroupGroupRatio],
        ['AutoGroups', nextValues.AutoGroups],
        ['DefaultUseAutoGroup', nextValues.DefaultUseAutoGroup],
        [
          'group_ratio_setting.group_special_usable_group',
          nextValues.GroupSpecialUsableGroup,
        ],
      ]
      for (const [key, value] of updates) {
        await updateOption.mutateAsync({ key, value })
      }
    },
    [updateOption]
  )

  return (
    <GroupPricingWorkspace
      form={form}
      onSave={save}
      isSaving={updateOption.isPending}
    />
  )
}
