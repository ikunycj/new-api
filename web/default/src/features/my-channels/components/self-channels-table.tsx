import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { GroupBadge } from '@/components/group-badge'
import { ProviderBadge } from '@/components/provider-badge'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { TruncatedText } from '@/components/truncated-text'
import { Input } from '@/components/ui/input'

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
              {models.map((model) => (
                <StatusBadge
                  key={model}
                  label={model}
                  autoColor={model}
                  size='sm'
                  className='font-mono'
                />
              ))}
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
            <SelfChannelCard key={channel.id} channel={channel} />
          ))}
        </div>
      )}
    </div>
  )
}
