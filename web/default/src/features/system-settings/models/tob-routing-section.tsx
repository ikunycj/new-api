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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, Plus, Save, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { getChannelTypeLabel } from '@/features/channels/lib/channel-utils'
import type { Channel } from '@/features/channels/types'
import { updateFailoverConfig } from '@/features/failover/api'
import type {
  BillingGroupChannel,
  BillingGroupRoute,
  FailoverConfig,
} from '@/features/failover/types'

import {
  channelBelongsToGroup,
  reorderBillingGroupChannels,
} from './group-pricing-utils'

let nextTemporaryID = -1

const defaultRouteSettings = {
  max_total_attempts: 1,
  total_timeout_ms: 30000,
  circuit_failure_threshold: 5,
  circuit_window_seconds: 60,
  circuit_cooldown_seconds: 60,
  circuit_half_open_requests: 1,
}

function createRoute(): BillingGroupRoute {
  return {
    id: nextTemporaryID--,
    billing_group: '',
    name: '',
    mode: 'balanced',
    enabled: false,
    ...defaultRouteSettings,
    created_time: 0,
    updated_time: 0,
  }
}

type ToBRoutingSectionProps = {
  config?: FailoverConfig
  channels: Channel[]
  groupNames: string[]
  isLoading: boolean
}

export function ToBRoutingSection(props: ToBRoutingSectionProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<FailoverConfig | null>(null)
  const [selectedRouteID, setSelectedRouteID] = useState<number | null>(null)
  const config = draft ?? props.config
  const channelByID = useMemo(
    () => new Map(props.channels.map((channel) => [channel.id, channel])),
    [props.channels]
  )
  const selectedRoute =
    config?.routes.find((route) => route.id === selectedRouteID) ??
    config?.routes[0]
  const saveMutation = useMutation({
    mutationFn: updateFailoverConfig,
    onSuccess: async () => {
      toast.success(t('Channel routing saved'))
      setDraft(null)
      setSelectedRouteID(null)
      await queryClient.invalidateQueries({
        queryKey: ['channel-routing-config'],
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const updateConfig = (
    updater: (current: FailoverConfig) => FailoverConfig
  ) => {
    if (!config) return
    setDraft(updater(structuredClone(config)))
  }

  const routeEntries = (route: BillingGroupRoute) =>
    config?.route_channels
      .filter((entry) => entry.billing_group_route_id === route.id)
      .sort((a, b) => b.priority - a.priority || a.id - b.id) ?? []

  const updateRoute = (
    routeIndex: number,
    patch: Partial<BillingGroupRoute>
  ) => {
    updateConfig((current) => {
      current.routes[routeIndex] = {
        ...current.routes[routeIndex],
        ...patch,
      }
      return current
    })
  }

  const updateEntry = (
    target: BillingGroupChannel,
    patch: Partial<BillingGroupChannel>
  ) => {
    updateConfig((current) => {
      const index = current.route_channels.findIndex(
        (entry) =>
          entry.billing_group_route_id === target.billing_group_route_id &&
          entry.channel_id === target.channel_id
      )
      if (index >= 0) current.route_channels[index] = { ...target, ...patch }
      return current
    })
  }

  const moveEntry = (
    route: BillingGroupRoute,
    target: BillingGroupChannel,
    direction: -1 | 1
  ) => {
    updateConfig((current) => {
      const entries = current.route_channels.filter(
        (entry) => entry.billing_group_route_id === route.id
      )
      const targetType = channelByID.get(target.channel_id)?.type ?? 0
      const protocolChannelIDs = new Set(
        entries.flatMap((entry) =>
          (channelByID.get(entry.channel_id)?.type ?? 0) === targetType
            ? [entry.channel_id]
            : []
        )
      )
      const reordered = reorderBillingGroupChannels(
        entries,
        target.channel_id,
        direction,
        protocolChannelIDs
      )
      const priorities = new Map(
        reordered.map((entry) => [entry.channel_id, entry.priority])
      )
      current.route_channels = current.route_channels.map((entry) => {
        if (entry.billing_group_route_id !== route.id) return entry
        const priority = priorities.get(entry.channel_id)
        return priority == null ? entry : { ...entry, priority }
      })
      return current
    })
  }

  const saveConfig = () => {
    if (!config) return
    saveMutation.mutate(structuredClone(config))
  }

  if (props.isLoading || !config) {
    return <div className='py-10 text-center text-sm'>{t('Loading')}</div>
  }

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-end justify-between gap-3'>
        <div className='w-full max-w-sm space-y-1.5'>
          <Label htmlFor='tob-route-select'>{t('Billing group')}</Label>
          <NativeSelect
            id='tob-route-select'
            className='w-full'
            value={selectedRoute?.id ?? ''}
            onChange={(event) => setSelectedRouteID(Number(event.target.value))}
          >
            <NativeSelectOption value=''>
              {t('Select a group')}
            </NativeSelectOption>
            {config.routes.map((route) => (
              <NativeSelectOption key={route.id} value={route.id}>
                {route.billing_group || t('Select a group')}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
        <div className='flex gap-2'>
          <Button
            variant='outline'
            onClick={() => {
              const route = createRoute()
              updateConfig((current) => ({
                ...current,
                routes: [...current.routes, route],
              }))
              setSelectedRouteID(route.id)
            }}
          >
            <Plus className='size-4' />
            {t('Add billing group route')}
          </Button>
          <Button
            onClick={saveConfig}
            disabled={!draft || saveMutation.isPending}
          >
            <Save className='size-4' />
            {t('Save')}
          </Button>
        </div>
      </div>

      {config.routes.map((route, routeIndex) => {
        if (route.id !== selectedRoute?.id) return null
        const entries = routeEntries(route)
        const entriesByProtocol = new Map<number, BillingGroupChannel[]>()
        for (const entry of entries) {
          const channelType = channelByID.get(entry.channel_id)?.type ?? 0
          const protocolEntries = entriesByProtocol.get(channelType) ?? []
          protocolEntries.push(entry)
          entriesByProtocol.set(channelType, protocolEntries)
        }
        return (
          <section key={route.id} className='space-y-5'>
            <div className='grid gap-3 border-y py-4 md:grid-cols-[1fr_auto_auto] md:items-end'>
              <div className='space-y-1.5'>
                <Label>{t('Billing group')}</Label>
                {route.id < 0 ? (
                  <NativeSelect
                    className='w-full'
                    value={route.billing_group}
                    onChange={(event) =>
                      updateRoute(routeIndex, {
                        billing_group: event.target.value,
                        name: event.target.value,
                      })
                    }
                  >
                    <NativeSelectOption value=''>
                      {t('Select a group')}
                    </NativeSelectOption>
                    {props.groupNames
                      .filter(
                        (group) =>
                          !config.routes.some(
                            (candidate) =>
                              candidate.id !== route.id &&
                              candidate.billing_group === group
                          )
                      )
                      .map((group) => (
                        <NativeSelectOption key={group} value={group}>
                          {group}
                        </NativeSelectOption>
                      ))}
                  </NativeSelect>
                ) : (
                  <Input value={route.billing_group} disabled />
                )}
              </div>
              <div className='flex items-center gap-2 pb-2'>
                <Switch
                  checked={route.enabled}
                  onCheckedChange={(enabled) =>
                    updateRoute(routeIndex, { enabled })
                  }
                />
                <Label>{t('Enabled')}</Label>
              </div>
              <Button
                variant='ghost'
                size='icon'
                title={t('Delete')}
                onClick={() => {
                  updateConfig((current) => ({
                    ...current,
                    routes: current.routes.filter(
                      (candidate) => candidate.id !== route.id
                    ),
                    route_channels: current.route_channels.filter(
                      (entry) => entry.billing_group_route_id !== route.id
                    ),
                  }))
                  setSelectedRouteID(null)
                }}
              >
                <Trash2 className='size-4' />
              </Button>
            </div>

            {[...entriesByProtocol.entries()].map(
              ([channelType, protocolEntries]) => (
                <div
                  key={channelType}
                  className='overflow-hidden rounded-md border'
                >
                  <div className='bg-muted/40 border-b px-4 py-3'>
                    <h3 className='text-sm font-semibold'>
                      {t(getChannelTypeLabel(channelType))} {t('Protocol')}
                    </h3>
                  </div>
                  <div className='overflow-x-auto'>
                    <table className='w-full min-w-[640px] text-sm'>
                      <thead className='text-muted-foreground border-b text-left'>
                        <tr>
                          <th className='px-4 py-2'>{t('Order')}</th>
                          <th className='px-4 py-2'>{t('Channel')}</th>
                          <th className='px-4 py-2'>
                            {t('Attempts on this channel')}
                          </th>
                          <th className='px-4 py-2 text-right'>
                            {t('Actions')}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {protocolEntries.map((entry, index) => {
                          const channel = channelByID.get(entry.channel_id)
                          return (
                            <tr
                              key={`${entry.billing_group_route_id}-${entry.channel_id}`}
                              className='border-b last:border-b-0'
                            >
                              <td className='px-4 py-2'>{index + 1}</td>
                              <td className='px-4 py-2'>
                                <div className='font-medium'>
                                  {channel?.name ?? `#${entry.channel_id}`}
                                </div>
                                {channel ? (
                                  <div className='text-muted-foreground text-xs'>
                                    #{channel.id}
                                  </div>
                                ) : null}
                              </td>
                              <td className='px-4 py-2'>
                                <Input
                                  className='w-24'
                                  type='number'
                                  min={1}
                                  value={entry.max_attempts}
                                  onChange={(event) =>
                                    updateEntry(entry, {
                                      max_attempts: Number(event.target.value),
                                    })
                                  }
                                />
                              </td>
                              <td className='px-4 py-2 text-right'>
                                <div className='flex justify-end gap-1'>
                                  <Button
                                    variant='ghost'
                                    size='icon'
                                    title={t('Move up')}
                                    disabled={index === 0}
                                    onClick={() => moveEntry(route, entry, -1)}
                                  >
                                    <ArrowUp className='size-4' />
                                  </Button>
                                  <Button
                                    variant='ghost'
                                    size='icon'
                                    title={t('Move down')}
                                    disabled={
                                      index === protocolEntries.length - 1
                                    }
                                    onClick={() => moveEntry(route, entry, 1)}
                                  >
                                    <ArrowDown className='size-4' />
                                  </Button>
                                  <Button
                                    variant='ghost'
                                    size='icon'
                                    title={t('Delete')}
                                    onClick={() =>
                                      updateConfig((current) => ({
                                        ...current,
                                        route_channels:
                                          current.route_channels.filter(
                                            (candidate) =>
                                              candidate.billing_group_route_id !==
                                                entry.billing_group_route_id ||
                                              candidate.channel_id !==
                                                entry.channel_id
                                          ),
                                      }))
                                    }
                                  >
                                    <Trash2 className='size-4' />
                                  </Button>
                                </div>
                              </td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              )
            )}

            <NativeSelect
              className='w-full max-w-sm'
              value=''
              onChange={(event) => {
                const channelID = Number(event.target.value)
                const channel = channelByID.get(channelID)
                if (!channel) return
                updateConfig((current) => {
                  const currentEntries = current.route_channels.filter(
                    (entry) => entry.billing_group_route_id === route.id
                  )
                  const lastPriority = currentEntries.reduce(
                    (lowest, entry) => Math.min(lowest, entry.priority),
                    1
                  )
                  current.route_channels.push({
                    id: nextTemporaryID--,
                    billing_group_route_id: route.id,
                    channel_id: channelID,
                    priority:
                      currentEntries.length === 0 ? 1 : lastPriority - 1,
                    weight: 0,
                    max_attempts: 1,
                    enabled: true,
                    cost_factor: 1,
                  })
                  return current
                })
              }}
            >
              <NativeSelectOption value=''>
                {t('Add channel')}
              </NativeSelectOption>
              {props.channels
                .filter(
                  (channel) =>
                    channelBelongsToGroup(channel, route.billing_group) &&
                    !entries.some((entry) => entry.channel_id === channel.id)
                )
                .map((channel) => (
                  <NativeSelectOption key={channel.id} value={channel.id}>
                    {channel.name} (#{channel.id})
                  </NativeSelectOption>
                ))}
            </NativeSelect>
          </section>
        )
      })}

      {config.routes.length === 0 ? (
        <div className='text-muted-foreground py-10 text-center text-sm'>
          {t('No ToB groups configured')}
        </div>
      ) : null}
    </div>
  )
}
