import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { GroupBadge } from '@/components/group-badge'
import { ProviderBadge } from '@/components/provider-badge'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { TruncatedText } from '@/components/truncated-text'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import { CHANNEL_STATUS_CONFIG } from '../../channels/constants'
import {
  getChannelTypeIcon,
  getChannelTypeLabel,
  parseGroupsList,
  parseModelsList,
} from '../../channels/lib'
import type { SelfChannel } from '../types'

type SelfChannelsTableProps = {
  channels: SelfChannel[]
}

function SelfChannelMetric(props: { label: string; children: ReactNode }) {
  return (
    <div className='flex min-w-0 items-start justify-between gap-3'>
      <span className='text-muted-foreground min-w-0 shrink-0 text-xs'>
        {props.label}
      </span>
      <div className='min-w-0 flex-1 text-right text-sm'>{props.children}</div>
    </div>
  )
}

function SelfChannelDetailsSheet(props: {
  channel: SelfChannel | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const channel = props.channel
  if (!channel) return null

  const status =
    CHANNEL_STATUS_CONFIG[
      channel.status as keyof typeof CHANNEL_STATUS_CONFIG
    ] || CHANNEL_STATUS_CONFIG[0]
  const typeLabel = t(getChannelTypeLabel(channel.type))
  const models = parseModelsList(channel.models)
  const groups = parseGroupsList(channel.group)

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className='sm:max-w-2xl'>
        <SheetHeader>
          <SheetTitle className='flex items-center gap-3'>
            <ProviderBadge
              iconKey={`${getChannelTypeIcon(channel.type)}.Color`}
              iconSize={22}
              label={typeLabel}
              colorText={false}
              copyable={false}
              showDot={false}
            />
            <span className='truncate'>{channel.name}</span>
          </SheetTitle>
          <SheetDescription>
            {t('Read-only view of channels available to your user group.')}
          </SheetDescription>
        </SheetHeader>
        <div className='min-h-0 flex-1 space-y-4 overflow-y-auto px-4 pb-6'>
          <div className='rounded-lg border p-4'>
            <div className='mb-3 text-base font-semibold'>
              {t('Basic Information')}
            </div>
            <div className='grid gap-3 sm:grid-cols-2'>
              <SelfChannelMetric label={t('ID')}>
                <TableId value={`#${channel.id}`} />
              </SelfChannelMetric>
              <SelfChannelMetric label={t('Status')}>
                <StatusBadge
                  label={t(status.label)}
                  variant={status.variant}
                  copyable={false}
                />
              </SelfChannelMetric>
              <SelfChannelMetric label={t('Type')}>
                <span>{typeLabel}</span>
              </SelfChannelMetric>
              <SelfChannelMetric label={t('Groups')}>
                <div className='flex flex-wrap justify-end gap-1'>
                  {groups.map((group) => (
                    <GroupBadge key={group} group={group} size='sm' />
                  ))}
                </div>
              </SelfChannelMetric>
            </div>
          </div>

          <div className='rounded-lg border p-4'>
            <div className='mb-3 text-base font-semibold'>{t('API URL')}</div>
            <TruncatedText
              text={channel.base_url || t('Default endpoint')}
              className='max-w-full'
            />
          </div>

          <div className='rounded-lg border p-4'>
            <div className='mb-3 text-base font-semibold'>{t('Models')}</div>
            {models.length > 0 ? (
              <div className='flex flex-wrap gap-1.5'>
                {models.map((model) => (
                  <StatusBadge
                    key={model}
                    label={model}
                    autoColor={model}
                    size='sm'
                    className='max-w-full font-mono'
                  />
                ))}
              </div>
            ) : (
              <span className='text-muted-foreground'>-</span>
            )}
          </div>

          {channel.model_mapping && (
            <div className='rounded-lg border p-4'>
              <div className='mb-3 text-base font-semibold'>
                {t('Model Mapping')}
              </div>
              <pre className='bg-muted/50 overflow-x-auto rounded-md p-3 text-xs whitespace-pre-wrap'>
                {channel.model_mapping}
              </pre>
            </div>
          )}

          {channel.remark && (
            <div className='rounded-lg border p-4'>
              <div className='mb-2 text-base font-semibold'>{t('Remark')}</div>
              <p className='text-sm'>{channel.remark}</p>
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function SelfChannelCard(props: { channel: SelfChannel }) {
  const { t } = useTranslation()
  const channel = props.channel
  const status =
    CHANNEL_STATUS_CONFIG[
      channel.status as keyof typeof CHANNEL_STATUS_CONFIG
    ] || CHANNEL_STATUS_CONFIG[0]
  const typeLabel = t(getChannelTypeLabel(channel.type))
  const groups = parseGroupsList(channel.group)
  const models = parseModelsList(channel.models)
  const visibleModels = models.slice(0, 8)
  const hiddenModelCount = Math.max(0, models.length - visibleModels.length)

  return (
    <div className='flex min-w-0 flex-col gap-3 rounded-xl border bg-(--table-row) p-4'>
      <div className='flex items-center justify-between gap-2'>
        <ProviderBadge
          iconKey={`${getChannelTypeIcon(channel.type)}.Color`}
          iconSize={20}
          label={typeLabel}
          colorText={false}
          copyable={false}
          showDot={false}
        />
        <StatusBadge
          label={t(status.label)}
          variant={status.variant}
          copyable={false}
        />
      </div>

      <div className='min-w-0'>
        <div className='text-muted-foreground text-xs'>
          <TableId value={`#${channel.id}`} />
        </div>
        <div className='truncate text-base font-semibold'>{channel.name}</div>
      </div>

      <div className='grid grid-cols-1 gap-x-5 gap-y-2 sm:grid-cols-2'>
        <SelfChannelMetric label={t('Models')}>
          {models.length > 0 ? (
            <div className='flex flex-wrap justify-end gap-1'>
              {visibleModels.map((model) => (
                <StatusBadge
                  key={model}
                  label={model}
                  autoColor={model}
                  size='sm'
                  className='max-w-44 font-mono'
                />
              ))}
              {hiddenModelCount > 0 && (
                <StatusBadge
                  label={`+${hiddenModelCount}`}
                  variant='neutral'
                  copyable={false}
                />
              )}
            </div>
          ) : (
            <span className='text-muted-foreground'>-</span>
          )}
        </SelfChannelMetric>
        <SelfChannelMetric label={t('Groups')}>
          <div className='flex flex-wrap justify-end gap-1'>
            {groups.map((group) => (
              <GroupBadge key={group} group={group} size='sm' />
            ))}
          </div>
        </SelfChannelMetric>
        <SelfChannelMetric label={t('Base URL')}>
          <TruncatedText
            text={channel.base_url || t('Default endpoint')}
            className='max-w-full text-xs'
          />
        </SelfChannelMetric>
        <SelfChannelMetric label={t('Status')}>
          <StatusBadge
            label={t(status.label)}
            variant={status.variant}
            copyable={false}
          />
        </SelfChannelMetric>
      </div>

      {channel.remark && (
        <div className='border-border/70 border-t pt-2'>
          <div className='text-muted-foreground mb-1 text-xs'>
            {t('Remark')}
          </div>
          <div className='text-sm'>{channel.remark}</div>
        </div>
      )}
    </div>
  )
}

export function SelfChannelsTable(props: SelfChannelsTableProps) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState('')
  const [selectedChannel, setSelectedChannel] = useState<SelfChannel | null>(
    null
  )

  const filteredChannels = useMemo(() => {
    const keyword = filter.trim().toLowerCase()
    if (!keyword) return props.channels

    return props.channels.filter((channel) =>
      [
        channel.name,
        channel.base_url,
        channel.models,
        channel.group,
        channel.remark,
      ]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword))
    )
  }, [filter, props.channels])

  return (
    <div className='space-y-3'>
      <Input
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
        placeholder={t('Filter by name...')}
        className='max-w-sm'
        aria-label={t('Filter by name...')}
      />
      {filteredChannels.length === 0 ? (
        <div className='text-muted-foreground rounded-lg border p-8 text-center text-sm'>
          {filter ? t('No results found') : t('No available channels')}
        </div>
      ) : (
        <div className='grid grid-cols-1 gap-4 lg:grid-cols-3'>
          {filteredChannels.map((channel) => (
            <button
              key={channel.id}
              type='button'
              className='focus-visible:ring-ring text-left transition hover:-translate-y-0.5 hover:shadow-sm focus-visible:ring-2 focus-visible:outline-none'
              onClick={() => setSelectedChannel(channel)}
              aria-label={`${t('View')} ${channel.name}`}
            >
              <SelfChannelCard channel={channel} />
            </button>
          ))}
        </div>
      )}
      <SelfChannelDetailsSheet
        channel={selectedChannel}
        open={selectedChannel !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedChannel(null)
        }}
      />
    </div>
  )
}
