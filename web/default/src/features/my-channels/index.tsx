import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth-store'

import { getSelfChannels } from './api'
import { SelfChannelsTable } from './components/self-channels-table'

export function MyChannels() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const channelsQuery = useQuery({
    queryKey: ['self-channels'],
    queryFn: getSelfChannels,
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Channels')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <Card>
          <CardHeader>
            <CardTitle>{t('Available channels')}</CardTitle>
            <p className='text-muted-foreground text-sm'>
              {t('Read-only view of channels available to your user group.')}
            </p>
          </CardHeader>
          <CardContent className='space-y-3'>
            <div className='bg-muted/50 rounded-lg p-3 text-sm'>
              {t('Current group')}: <strong>{user?.group}</strong>
            </div>
            {channelsQuery.isLoading && (
              <p className='text-muted-foreground'>{t('Loading...')}</p>
            )}
            {!channelsQuery.isLoading && channelsQuery.data && (
              <SelfChannelsTable channels={channelsQuery.data} />
            )}
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
