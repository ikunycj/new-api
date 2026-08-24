import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StatusBadge } from '@/components/status-badge'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import type { Channel } from '@/features/channels/types'

import {
  channelBelongsToGroup,
  type GroupPricingSnapshot,
} from './group-pricing-utils'

type ToCGroupsSectionProps = {
  groups: GroupPricingSnapshot[]
  channels: Channel[]
  toBGroupNames: ReadonlySet<string>
}

const LEGACY_TOC_GROUPS = ['ChatGPT Plus', 'ChatGPT Pro'] as const

export function ToCGroupsSection(props: ToCGroupsSectionProps) {
  const { t } = useTranslation()
  const [selectedGroupName, setSelectedGroupName] = useState<string | null>(
    null
  )
  const groups = useMemo(() => {
    const configured = props.groups.filter(
      (group) => !props.toBGroupNames.has(group.name)
    )
    const configuredNames = new Set(configured.map((group) => group.name))
    const legacy = LEGACY_TOC_GROUPS.filter(
      (name) => !configuredNames.has(name)
    ).map((name) => ({
      name,
      ratio: 1,
      topupRatio: null,
      selectable: false,
      description: '',
    }))
    return [...configured, ...legacy]
  }, [props.groups, props.toBGroupNames])
  const selectedGroup =
    groups.find((group) => group.name === selectedGroupName) ?? groups[0]
  const channels = useMemo(() => {
    if (!selectedGroup) return []
    return props.channels
      .filter((channel) => channelBelongsToGroup(channel, selectedGroup.name))
      .sort(
        (a, b) =>
          (b.priority ?? 0) - (a.priority ?? 0) || a.name.localeCompare(b.name)
      )
  }, [props.channels, selectedGroup])

  if (!selectedGroup) {
    return (
      <div className='text-muted-foreground py-10 text-center text-sm'>
        {t('No ToC groups configured')}
      </div>
    )
  }

  return (
    <div className='space-y-6'>
      <div className='max-w-sm space-y-1.5'>
        <label className='text-sm font-medium' htmlFor='toc-group-select'>
          {t('Billing group')}
        </label>
        <NativeSelect
          id='toc-group-select'
          className='w-full'
          value={selectedGroup.name}
          onChange={(event) => setSelectedGroupName(event.target.value)}
        >
          {groups.map((group) => (
            <NativeSelectOption key={group.name} value={group.name}>
              {group.name}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      </div>

      <section className='space-y-3'>
        <div>
          <h3 className='font-medium'>{t('Group pricing')}</h3>
          <p className='text-muted-foreground text-sm'>
            {t(
              'ToC pricing is read-only here and is managed in Basic information.'
            )}
          </p>
        </div>
        <dl className='grid gap-4 border-y py-4 sm:grid-cols-2 lg:grid-cols-4'>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('Group name')}</dt>
            <dd className='mt-1 font-medium'>{selectedGroup.name}</dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>{t('Ratio')}</dt>
            <dd className='mt-1 font-medium'>{selectedGroup.ratio}</dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>
              {t('Top-up ratio')}
            </dt>
            <dd className='mt-1 font-medium'>
              {selectedGroup.topupRatio ?? t('Not set')}
            </dd>
          </div>
          <div>
            <dt className='text-muted-foreground text-xs'>
              {t('User selectable')}
            </dt>
            <dd className='mt-1'>
              <StatusBadge
                variant={selectedGroup.selectable ? 'success' : 'neutral'}
                copyable={false}
              >
                {t(selectedGroup.selectable ? 'Yes' : 'No')}
              </StatusBadge>
            </dd>
          </div>
        </dl>
        {selectedGroup.description ? (
          <p className='text-muted-foreground text-sm'>
            {selectedGroup.description}
          </p>
        ) : null}
      </section>

      <section className='space-y-3'>
        <div>
          <h3 className='font-medium'>{t('Channels')}</h3>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Channel priority and weight are read-only and managed on the Channels page.'
            )}
          </p>
        </div>
        <StaticDataTable
          data={channels}
          getRowKey={(channel) => channel.id}
          emptyContent={t('No channels configured for this group')}
          columns={[
            {
              id: 'channel',
              header: t('Channel'),
              className: 'min-w-48',
              cell: (channel) => (
                <div>
                  <div className='font-medium'>{channel.name}</div>
                  <div className='text-muted-foreground text-xs'>
                    #{channel.id}
                  </div>
                </div>
              ),
            },
            {
              id: 'priority',
              header: t('Channel priority'),
              className: 'w-32',
              cell: (channel) => channel.priority ?? 0,
            },
            {
              id: 'weight',
              header: t('Weight'),
              className: 'w-28',
              cell: (channel) => channel.weight ?? 0,
            },
            {
              id: 'models',
              header: t('Models'),
              className: 'min-w-64',
              cell: (channel) => (
                <span className='text-muted-foreground break-words'>
                  {channel.models || '-'}
                </span>
              ),
            },
          ]}
        />
      </section>
    </div>
  )
}
