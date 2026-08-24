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

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import type { Channel } from '@/features/channels/types'
import { updateFailoverConfig } from '@/features/failover/api'
import type {
  BillingGroupChannel,
  BillingGroupRoute,
  FailoverConfig,
  RoutingMode,
} from '@/features/failover/types'

import {
  channelBelongsToGroup,
  reorderBillingGroupChannels,
} from './group-pricing-utils'

let nextTemporaryID = -1

const routeDefaults: Record<
  RoutingMode,
  Pick<
    BillingGroupRoute,
    | 'max_total_attempts'
    | 'total_timeout_ms'
    | 'circuit_failure_threshold'
    | 'circuit_window_seconds'
    | 'circuit_cooldown_seconds'
    | 'circuit_half_open_requests'
  >
> = {
  cost_first: {
    max_total_attempts: 6,
    total_timeout_ms: 45000,
    circuit_failure_threshold: 8,
    circuit_window_seconds: 60,
    circuit_cooldown_seconds: 60,
    circuit_half_open_requests: 1,
  },
  balanced: {
    max_total_attempts: 4,
    total_timeout_ms: 30000,
    circuit_failure_threshold: 5,
    circuit_window_seconds: 60,
    circuit_cooldown_seconds: 60,
    circuit_half_open_requests: 1,
  },
  stability_first: {
    max_total_attempts: 3,
    total_timeout_ms: 20000,
    circuit_failure_threshold: 3,
    circuit_window_seconds: 60,
    circuit_cooldown_seconds: 90,
    circuit_half_open_requests: 1,
  },
}

function createRoute(): BillingGroupRoute {
  return {
    id: nextTemporaryID--,
    billing_group: '',
    name: '',
    mode: 'balanced',
    enabled: false,
    ...routeDefaults.balanced,
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
  const getPlanLabel = (mode: RoutingMode) => {
    if (mode === 'cost_first') return 'Cost first'
    if (mode === 'stability_first') return 'Stability first'
    return 'Balanced'
  }

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
      const reordered = reorderBillingGroupChannels(
        entries,
        target.channel_id,
        direction
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
    const synchronized = structuredClone(config)
    synchronized.route_channels = synchronized.route_channels.map((entry) => ({
      ...entry,
      weight: channelByID.get(entry.channel_id)?.weight ?? 0,
    }))
    saveMutation.mutate(synchronized)
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
        return (
          <section key={route.id} className='space-y-5'>
            <div className='grid gap-3 border-y py-4 md:grid-cols-[1fr_180px_auto_auto] md:items-end'>
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
              <div className='space-y-1.5'>
                <Label>{t('Routing strategy')}</Label>
                <NativeSelect
                  className='w-full'
                  value={route.mode}
                  onChange={(event) => {
                    const mode = event.target.value as RoutingMode
                    updateRoute(routeIndex, { mode, ...routeDefaults[mode] })
                  }}
                >
                  <NativeSelectOption value='cost_first'>
                    {t('Cost first')}
                  </NativeSelectOption>
                  <NativeSelectOption value='balanced'>
                    {t('Balanced')}
                  </NativeSelectOption>
                  <NativeSelectOption value='stability_first'>
                    {t('Stability first')}
                  </NativeSelectOption>
                </NativeSelect>
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

            <div className='grid gap-4 sm:grid-cols-3'>
              <div>
                <div className='text-muted-foreground text-xs'>{t('Plan')}</div>
                <StatusBadge variant='info' copyable={false} className='mt-1'>
                  {t(getPlanLabel(route.mode))}
                </StatusBadge>
              </div>
              <div>
                <div className='text-muted-foreground text-xs'>
                  {t('Maximum total attempts')}
                </div>
                <div className='mt-1 font-medium'>
                  {route.max_total_attempts}
                </div>
              </div>
              <div>
                <div className='text-muted-foreground text-xs'>
                  {t('Total timeout (ms)')}
                </div>
                <div className='mt-1 font-medium'>{route.total_timeout_ms}</div>
              </div>
            </div>

            <div className='overflow-x-auto'>
              <table className='w-full min-w-[900px] text-sm'>
                <thead className='text-muted-foreground border-b text-left'>
                  <tr>
                    <th className='py-2 pr-3'>{t('Order')}</th>
                    <th className='py-2 pr-3'>{t('Channel')}</th>
                    <th className='py-2 pr-3'>{t('Channel priority')}</th>
                    <th className='py-2 pr-3'>{t('Weight')}</th>
                    <th className='py-2 pr-3'>{t('Cost factor')}</th>
                    <th className='py-2 pr-3'>
                      {t('Attempts on this channel')}
                    </th>
                    <th className='py-2 text-right'>{t('Actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((entry, index) => {
                    const channel = channelByID.get(entry.channel_id)
                    return (
                      <tr
                        key={`${entry.billing_group_route_id}-${entry.channel_id}`}
                        className='border-b'
                      >
                        <td className='py-2 pr-3'>{index + 1}</td>
                        <td className='py-2 pr-3'>
                          <div className='font-medium'>
                            {channel?.name ?? `#${entry.channel_id}`}
                          </div>
                          {channel ? (
                            <div className='text-muted-foreground text-xs'>
                              #{channel.id}
                            </div>
                          ) : null}
                        </td>
                        <td className='py-2 pr-3'>{channel?.priority ?? 0}</td>
                        <td className='py-2 pr-3'>{channel?.weight ?? 0}</td>
                        <td className='py-2 pr-3'>{entry.cost_factor}</td>
                        <td className='py-2 pr-3'>
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
                        <td className='py-2 text-right'>
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
                              disabled={index === entries.length - 1}
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
                                  route_channels: current.route_channels.filter(
                                    (candidate) =>
                                      candidate.billing_group_route_id !==
                                        entry.billing_group_route_id ||
                                      candidate.channel_id !== entry.channel_id
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
                    weight: channel.weight ?? 0,
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
