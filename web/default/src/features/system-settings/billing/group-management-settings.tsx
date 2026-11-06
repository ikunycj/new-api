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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { SettingsPageFrame } from '../components/settings-page'
import { useSystemOptions, getOptionValue } from '../hooks/use-system-options'
import { RatioSettingsCard } from '../models/ratio-settings-card'
import type { BillingSettings } from '../types'
import {
  defaultBillingSettings,
  getGroupDefaults,
  getModelDefaults,
} from './settings-defaults'
import { UserGroupManagementSection } from './user-group-management-section'

export type GroupManagementTab = 'user-groups' | 'pricing-groups'

type GroupManagementSettingsProps = {
  activeTab: GroupManagementTab
  onTabChange: (tab: GroupManagementTab) => void
}

export function GroupManagementSettings({
  activeTab,
  onTabChange,
}: GroupManagementSettingsProps) {
  const { t } = useTranslation()
  const { data, isLoading, isError, error, refetch } = useSystemOptions({
    skipErrorHandler: true,
  })
  const settings = useMemo(
    () => getOptionValue(data?.data, defaultBillingSettings) as BillingSettings,
    [data?.data]
  )

  if (isLoading) {
    return (
      <SettingsPageFrame title={t('Group Management')}>
        <div className='text-muted-foreground flex min-h-40 items-center justify-center text-sm'>
          {t('Loading settings...')}
        </div>
      </SettingsPageFrame>
    )
  }

  if (isError) {
    return (
      <SettingsPageFrame title={t('Group Management')}>
        <div className='flex min-h-40 flex-col items-center justify-center gap-3 text-center'>
          <p className='text-destructive text-sm'>
            分组设置加载失败：{error.message}
          </p>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => void refetch()}
          >
            重试
          </Button>
        </div>
      </SettingsPageFrame>
    )
  }

  return (
    <SettingsPageFrame title={t('Group Management')}>
      <Tabs
        value={activeTab}
        onValueChange={(value) => onTabChange(value as GroupManagementTab)}
        className='min-h-0 gap-4'
      >
        <TabsList className='grid w-full max-w-md grid-cols-2'>
          <TabsTrigger value='pricing-groups'>定价分组</TabsTrigger>
          <TabsTrigger value='user-groups'>用户分组</TabsTrigger>
        </TabsList>
        <TabsContent value='pricing-groups' className='min-h-0'>
          <RatioSettingsCard
            titleKey='定价分组'
            modelDefaults={getModelDefaults(settings)}
            groupDefaults={getGroupDefaults(settings)}
            toolPricesDefault={settings['tool_price_setting.prices']}
            visibleTabs={['groups']}
          />
        </TabsContent>
        <TabsContent value='user-groups' className='min-h-0'>
          <UserGroupManagementSection />
        </TabsContent>
      </Tabs>
    </SettingsPageFrame>
  )
}
