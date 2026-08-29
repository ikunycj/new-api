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

import { Button } from '@/components/ui/button'
import { Link } from '@tanstack/react-router'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { SettingsPageFrame } from '../components/settings-page'
import { useSystemOptions } from '../hooks/use-system-options'
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
  const { isLoading, isError, error, refetch } = useSystemOptions()

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
          <div className='flex flex-col gap-3 rounded-lg border p-4'>
            <p className='text-muted-foreground text-sm'>
              定价分组、ToB/ToC 路由、权重和调度策略继续在现有分组定价工作区维护。
            </p>
            <Button render={<Link to='/group-pricing' />}>打开分组定价</Button>
          </div>
        </TabsContent>
        <TabsContent value='user-groups' className='min-h-0'>
          <UserGroupManagementSection />
        </TabsContent>
      </Tabs>
    </SettingsPageFrame>
  )
}
