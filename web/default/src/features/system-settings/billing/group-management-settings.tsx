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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

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

export function GroupManagementSettings() {
  const { t } = useTranslation()
  const { data, isLoading } = useSystemOptions()
  const settings = useMemo(
    () => getOptionValue(data?.data, defaultBillingSettings) as BillingSettings,
    [data?.data]
  )

  return (
    <SettingsPageFrame title={t('Group Management')}>
      {isLoading ? (
        <div className='text-muted-foreground flex min-h-40 items-center justify-center text-sm'>
          {t('Loading settings...')}
        </div>
      ) : (
        <>
          <UserGroupManagementSection />
          <RatioSettingsCard
            titleKey='定价分组'
            modelDefaults={getModelDefaults(settings)}
            groupDefaults={getGroupDefaults(settings)}
            toolPricesDefault={settings['tool_price_setting.prices']}
            visibleTabs={['groups']}
          />
        </>
      )}
    </SettingsPageFrame>
  )
}
