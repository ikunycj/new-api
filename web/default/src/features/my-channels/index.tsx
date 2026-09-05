import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth-store'

import { getSelfChannels } from './api'

export function MyChannels() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const channelsQuery = useQuery({ queryKey: ['self-channels'], queryFn: getSelfChannels })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Channels')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <Card>
          <CardHeader>
            <CardTitle>{t('Available channels')}</CardTitle>
            <p className='text-muted-foreground text-sm'>{t('Read-only view of channels available to your user group.')}</p>
          </CardHeader>
          <CardContent className='space-y-2'>
            <div className='bg-muted/50 rounded-lg p-3 text-sm'>{t('Current group')}: <strong>{user?.group}</strong></div>
            {channelsQuery.isLoading && <p className='text-muted-foreground'>{t('Loading...')}</p>}
            {!channelsQuery.isLoading && channelsQuery.data?.length === 0 && <p className='text-muted-foreground'>{t('No available channels')}</p>}
            {channelsQuery.data?.map((channel) => (
              <div key={channel.id} className='flex items-center justify-between rounded-lg border p-3'>
                <div className='min-w-0'>
                  <div className='truncate font-medium'>{channel.name}</div>
                  <div className='text-muted-foreground truncate text-xs'>{channel.base_url || t('Default endpoint')}</div>
                  <div className='text-muted-foreground mt-1 text-xs'>{t('Models')}: {channel.models || '-'}</div>
                </div>
                <div className='flex items-center gap-2'>
                  <Badge variant='outline'>{channel.group}</Badge>
                  <Badge variant={channel.status === 1 ? 'default' : 'outline'}>{channel.status === 1 ? t('Enabled') : t('Disabled')}</Badge>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
