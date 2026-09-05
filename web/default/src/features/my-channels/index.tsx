import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Plus, Save, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth-store'

import {
  createSelfChannel,
  deleteSelfChannel,
  getSelfChannelFormData,
  getSelfChannels,
  updateSelfChannel,
} from './api'
import type { SelfChannel, SelfChannelRequest } from './types'

const isToBUser = (group?: string) => ['vip', 'enterprise'].includes((group ?? '').toLowerCase())

export function MyChannels() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<SelfChannel | undefined>()
  const [form, setForm] = useState<SelfChannelRequest>(getSelfChannelFormData())
  const channelsQuery = useQuery({
    queryKey: ['self-channels'],
    queryFn: getSelfChannels,
    enabled: isToBUser(user?.group),
  })

  const saveMutation = useMutation({
    mutationFn: () =>
      selected ? updateSelfChannel(selected.id, form) : createSelfChannel(form),
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(t('Saved'))
      setSelected(undefined)
      setForm(getSelfChannelFormData())
      queryClient.invalidateQueries({ queryKey: ['self-channels'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteSelfChannel(id),
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Delete failed'))
        return
      }
      toast.success(t('Deleted'))
      setSelected(undefined)
      setForm(getSelfChannelFormData())
      queryClient.invalidateQueries({ queryKey: ['self-channels'] })
    },
  })

  const selectChannel = (channel?: SelfChannel) => {
    setSelected(channel)
    setForm(getSelfChannelFormData(channel))
  }

  if (!isToBUser(user?.group)) return null

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Channels')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button onClick={() => selectChannel()}>
          <Plus />
          {t('New Channel')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,28rem)]'>
        <Card>
          <CardHeader>
            <CardTitle>{t('My Channels')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-2'>
            {channelsQuery.isLoading && <p className='text-muted-foreground'>{t('Loading...')}</p>}
            {!channelsQuery.isLoading && channelsQuery.data?.length === 0 && (
              <p className='text-muted-foreground'>{t('No channels yet')}</p>
            )}
            {channelsQuery.data?.map((channel) => (
              <button
                key={channel.id}
                type='button'
                onClick={() => selectChannel(channel)}
                className='hover:bg-muted flex w-full items-center justify-between rounded-lg border p-3 text-left'
              >
                <span className='min-w-0'>
                  <span className='block truncate font-medium'>{channel.name}</span>
                  <span className='text-muted-foreground block truncate text-xs'>
                    {channel.base_url || t('Default endpoint')}
                  </span>
                </span>
                <Badge variant={channel.status === 1 ? 'default' : 'outline'}>
                  {channel.status === 1 ? t('Enabled') : t('Pending review')}
                </Badge>
              </button>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{selected ? t('Edit Channel') : t('New Channel')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-4'>
            <div className='grid gap-2'>
              <Label htmlFor='self-channel-name'>{t('Channel name')}</Label>
              <Input id='self-channel-name' value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='self-channel-key'>{t('API Key')}</Label>
              <Input id='self-channel-key' type='password' value={form.key} onChange={(event) => setForm({ ...form, key: event.target.value })} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='self-channel-base-url'>{t('Base URL')}</Label>
              <Input id='self-channel-base-url' value={form.base_url} onChange={(event) => setForm({ ...form, base_url: event.target.value })} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='self-channel-models'>{t('Models')}</Label>
              <Textarea id='self-channel-models' value={form.models} onChange={(event) => setForm({ ...form, models: event.target.value })} />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='self-channel-remark'>{t('Remark')}</Label>
              <Input id='self-channel-remark' value={form.remark} onChange={(event) => setForm({ ...form, remark: event.target.value })} />
            </div>
            <div className='bg-muted/50 rounded-lg p-3 text-sm'>
              <div>{t('Current group')}: <strong>{user?.group}</strong></div>
              <div>{t('Channel group')}: <strong>default</strong></div>
              <div className='text-muted-foreground mt-1'>{t('New channels are disabled until an administrator approves them.')}</div>
            </div>
            <div className='flex justify-between gap-2'>
              <Button disabled={saveMutation.isPending} onClick={() => saveMutation.mutate()}>
                <Save />
                {t('Save')}
              </Button>
              {selected && (
                <Button variant='destructive' disabled={deleteMutation.isPending} onClick={() => deleteMutation.mutate(selected.id)}>
                  <Trash2 />
                  {t('Delete')}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
