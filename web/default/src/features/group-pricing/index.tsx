import { useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { SettingsPageProvider } from '@/features/system-settings/components/settings-page-context'
import { useSystemOptions } from '@/features/system-settings/hooks/use-system-options'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import {
  GroupPricingWorkspace,
  type GroupPricingValues,
} from '@/features/system-settings/models/group-pricing-workspace'

const DEFAULT_GROUP_VALUES: GroupPricingValues = {
  GroupRatio: '{}',
  TopupGroupRatio: '{}',
  UserUsableGroups: '{}',
  GroupGroupRatio: '{}',
  AutoGroups: '[]',
  DefaultUseAutoGroup: false,
  GroupSpecialUsableGroup: '{}',
}

export function GroupPricing() {
  const { t } = useTranslation()
  const [actionsContainer, setActionsContainer] =
    useState<HTMLDivElement | null>(null)
  const optionsQuery = useSystemOptions()
  const updateOption = useUpdateOption()
  const values = useMemo(() => {
    const options = new Map(
      (optionsQuery.data?.data ?? []).map((option) => [
        option.key,
        option.value,
      ])
    )
    return {
      GroupRatio: options.get('GroupRatio') ?? DEFAULT_GROUP_VALUES.GroupRatio,
      TopupGroupRatio:
        options.get('TopupGroupRatio') ?? DEFAULT_GROUP_VALUES.TopupGroupRatio,
      UserUsableGroups:
        options.get('UserUsableGroups') ??
        DEFAULT_GROUP_VALUES.UserUsableGroups,
      GroupGroupRatio:
        options.get('GroupGroupRatio') ?? DEFAULT_GROUP_VALUES.GroupGroupRatio,
      AutoGroups: options.get('AutoGroups') ?? DEFAULT_GROUP_VALUES.AutoGroups,
      DefaultUseAutoGroup:
        options.get('DefaultUseAutoGroup') === 'true' ||
        options.get('DefaultUseAutoGroup') === '1',
      GroupSpecialUsableGroup:
        options.get('group_ratio_setting.group_special_usable_group') ??
        DEFAULT_GROUP_VALUES.GroupSpecialUsableGroup,
    }
  }, [optionsQuery.data?.data])
  const form = useForm<GroupPricingValues>({ defaultValues: values })

  useEffect(() => {
    if (optionsQuery.data) form.reset(values)
  }, [form, optionsQuery.data, values])

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
    <SettingsPageProvider actionsContainer={actionsContainer}>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Group Pricing')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <div
            ref={setActionsContainer}
            className='flex flex-wrap items-center justify-end gap-2'
          />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <GroupPricingWorkspace
            form={form}
            onSave={save}
            isSaving={updateOption.isPending || optionsQuery.isLoading}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </SettingsPageProvider>
  )
}
