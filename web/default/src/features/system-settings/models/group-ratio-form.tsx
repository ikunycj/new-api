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
import { memo, useCallback, useState } from 'react'
import { useFormState, useWatch, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Form } from '@/components/ui/form'

import { SettingsPageActionsPortal } from '../components/settings-page-context'
import { GroupRatioVisualEditor } from './group-ratio-visual-editor'

type GroupFormValues = {
  GroupRatio: string
  PricingGroupEnabled: string
  PricingGroupRemark: string
  PricingGroupOrder: string
  PricingGroupRetryPolicy: string
  PricingGroupRoutingStrategy: string
}

type GroupRatioFormProps = {
  form: UseFormReturn<GroupFormValues>
  savedGroupRatio: string
  onSave: (values: GroupFormValues) => Promise<void>
  isSaving: boolean
}

export const GroupRatioForm = memo(function GroupRatioForm({
  form,
  savedGroupRatio,
  onSave,
  isSaving,
}: GroupRatioFormProps) {
  const { t } = useTranslation()
  const [isEditorValid, setIsEditorValid] = useState(true)
  const [
    groupRatio,
    pricingGroupEnabled,
    pricingGroupRemark,
    pricingGroupOrder,
    pricingGroupRetryPolicy,
    pricingGroupRoutingStrategy,
  ] = useWatch({
    control: form.control,
    name: [
      'GroupRatio',
      'PricingGroupEnabled',
      'PricingGroupRemark',
      'PricingGroupOrder',
      'PricingGroupRetryPolicy',
      'PricingGroupRoutingStrategy',
    ],
  })
  const { isValid } = useFormState({ control: form.control })

  const handleFieldChange = useCallback(
    (field: keyof GroupFormValues, value: string) => {
      form.setValue(field, value, {
        shouldValidate: true,
        shouldDirty: true,
      })
    },
    [form]
  )

  return (
    <Form {...form}>
      <SettingsPageActionsPortal>
        <Button
          type='button'
          size='sm'
          onClick={form.handleSubmit(onSave)}
          disabled={isSaving || !isEditorValid || !isValid}
        >
          {isSaving ? t('Saving...') : t('Save')}
        </Button>
      </SettingsPageActionsPortal>
      <GroupRatioVisualEditor
        groupRatio={groupRatio}
        pricingGroupEnabled={pricingGroupEnabled}
        pricingGroupRemark={pricingGroupRemark}
        pricingGroupOrder={pricingGroupOrder}
        pricingGroupRetryPolicy={pricingGroupRetryPolicy}
        pricingGroupRoutingStrategy={pricingGroupRoutingStrategy}
        savedGroupRatio={savedGroupRatio}
        onValidationChange={setIsEditorValid}
        onChange={(field, value) =>
          handleFieldChange(field as keyof GroupFormValues, value)
        }
      />
    </Form>
  )
})
