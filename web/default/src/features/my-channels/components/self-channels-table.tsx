import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { BadgeListCell, StaticDataTable } from '@/components/data-table'
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

  const columns = [
    {
      id: 'id',
      header: t('ID'),
      className: 'w-20',
      cell: (channel: SelfChannel) => <TableId value={channel.id} />,
    },
    {
      id: 'name',
      header: t('Name'),
      className: 'min-w-52',
      cell: (channel: SelfChannel) => (
        <div className='min-w-0'>
          <TruncatedText text={channel.name} className='font-medium' />
          {channel.remark && (
            <div className='text-muted-foreground mt-1 truncate text-xs'>
              {channel.remark}
            </div>
          )}
        </div>
      ),
    },
    {
      id: 'type',
      header: t('Type'),
      className: 'w-48',
      cell: (channel: SelfChannel) => {
        const label = t(getChannelTypeLabel(channel.type))
        return (
          <ProviderBadge
            iconKey={`${getChannelTypeIcon(channel.type)}.Color`}
            iconSize={18}
            label={label}
            colorText={false}
            copyable={false}
            showDot={false}
          />
        )
      },
    },
    {
      id: 'status',
      header: t('Status'),
      className: 'w-32',
      cell: (channel: SelfChannel) => {
        const config =
          CHANNEL_STATUS_CONFIG[
            channel.status as keyof typeof CHANNEL_STATUS_CONFIG
          ] || CHANNEL_STATUS_CONFIG[0]
        return (
          <StatusBadge
            label={t(config.label)}
            variant={config.variant}
            copyable={false}
          />
        )
      },
    },
    {
      id: 'models',
      header: t('Models'),
      className: 'min-w-56',
      cell: (channel: SelfChannel) => (
        <BadgeListCell
          items={parseModelsList(channel.models).map((model) => (
            <StatusBadge
              key={model}
              label={model}
              autoColor={model}
              size='sm'
              className='font-mono'
            />
          ))}
        />
      ),
    },
    {
      id: 'group',
      header: t('Groups'),
      className: 'min-w-40',
      cell: (channel: SelfChannel) => (
        <BadgeListCell
          items={parseGroupsList(channel.group).map((group) => (
            <GroupBadge key={group} group={group} size='sm' />
          ))}
        />
      ),
    },
    {
      id: 'base_url',
      header: t('Base URL'),
      className: 'min-w-56',
      cell: (channel: SelfChannel) => (
        <TruncatedText
          text={channel.base_url || t('Default endpoint')}
          className='text-muted-foreground text-xs'
        />
      ),
    },
  ]

  return (
    <div className='space-y-3'>
      <Input
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
        placeholder={t('Filter by name...')}
        className='max-w-sm'
        aria-label={t('Filter by name...')}
      />
      <StaticDataTable
        columns={columns}
        data={filteredChannels}
        getRowKey={(channel) => channel.id}
        emptyContent={
          <span className='text-muted-foreground'>
            {filter ? t('No results found') : t('No available channels')}
          </span>
        }
      />
    </div>
  )
}
