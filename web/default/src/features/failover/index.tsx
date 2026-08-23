import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowDown,
  ArrowUp,
  ExternalLink,
  Plus,
  Save,
  Trash2,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getChannels } from '@/features/channels/api'
import { getGroups } from '@/features/users/api'

import {
  getFailoverConfig,
  getFailoverMonitoring,
  updateFailoverConfig,
} from './api'
import type {
  BillingGroupChannel,
  BillingGroupRoute,
  FailoverConfig,
  RoutingMode,
  UpstreamErrorMapping,
} from './types'

let nextTemporaryID = -1

const emptyRoute = (): BillingGroupRoute => ({
  id: nextTemporaryID--,
  billing_group: '',
  name: '',
  mode: 'balanced',
  enabled: true,
  max_total_attempts: 4,
  total_timeout_ms: 30000,
  circuit_failure_threshold: 5,
  circuit_window_seconds: 60,
  circuit_cooldown_seconds: 60,
  circuit_half_open_requests: 1,
  created_time: 0,
  updated_time: 0,
})

const emptyMapping = (): UpstreamErrorMapping => ({
  id: nextTemporaryID--,
  channel_id: 0,
  channel_type: 0,
  raw_code: '',
  status_code: 429,
  alltoken_code: 204001,
  category: 'rate_limit',
  failure_scope: 'channel',
  action: 'switch_channel',
  retryable: true,
  enabled: true,
})

function NumberField(props: {
  label: string
  value: number
  min?: number
  step?: number
  onChange: (value: number) => void
}) {
  return (
    <div className='space-y-1.5'>
      <Label>{props.label}</Label>
      <Input
        type='number'
        min={props.min ?? 0}
        step={props.step ?? 1}
        value={props.value}
        onChange={(event) => props.onChange(Number(event.target.value))}
      />
    </div>
  )
}

export function FailoverConfiguration() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<FailoverConfig | null>(null)
  const configQuery = useQuery({
    queryKey: ['channel-routing-config'],
    queryFn: getFailoverConfig,
  })
  const monitoringQuery = useQuery({
    queryKey: ['channel-routing-monitoring'],
    queryFn: getFailoverMonitoring,
    refetchInterval: 10000,
  })
  const channelsQuery = useQuery({
    queryKey: ['channel-routing-channels'],
    queryFn: () => getChannels({ page_size: 1000, id_sort: true }),
  })
  const groupsQuery = useQuery({
    queryKey: ['channel-routing-billing-groups'],
    queryFn: getGroups,
  })
  const channels = useMemo(
    () => channelsQuery.data?.data?.items ?? [],
    [channelsQuery.data]
  )
  const billingGroups = useMemo(() => {
    const configured = groupsQuery.data?.data ?? []
    const routeGroups =
      configQuery.data?.routes.map((route) => route.billing_group) ?? []
    return [...new Set([...configured, ...routeGroups].filter(Boolean))].sort(
      (a, b) => a.localeCompare(b)
    )
  }, [configQuery.data?.routes, groupsQuery.data?.data])
  const channelByID = useMemo(
    () => new Map(channels.map((channel) => [channel.id, channel])),
    [channels]
  )
  const config = draft ?? configQuery.data

  const saveMutation = useMutation({
    mutationFn: updateFailoverConfig,
    onSuccess: async () => {
      toast.success(t('Channel routing saved'))
      setDraft(null)
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

  const addRoute = () => {
    updateConfig((current) => ({
      ...current,
      routes: [...current.routes, emptyRoute()],
    }))
  }

  const removeRoute = (routeIndex: number) => {
    updateConfig((current) => {
      const route = current.routes[routeIndex]
      return {
        ...current,
        routes: current.routes.filter((_, index) => index !== routeIndex),
        route_channels: current.route_channels.filter(
          (entry) => entry.billing_group_route_id !== route.id
        ),
      }
    })
  }

  const updateRoute = (
    routeIndex: number,
    key: keyof BillingGroupRoute,
    value: string | number | boolean
  ) => {
    updateConfig((current) => {
      current.routes[routeIndex] = {
        ...current.routes[routeIndex],
        [key]: value,
      }
      return current
    })
  }

  const routeEntries = (route: BillingGroupRoute) =>
    config?.route_channels
      .filter((entry) => entry.billing_group_route_id === route.id)
      .sort((a, b) => b.priority - a.priority) ?? []

  const addChannel = (route: BillingGroupRoute, channelID: number) => {
    if (channelID <= 0) return
    updateConfig((current) => {
      if (
        current.route_channels.some(
          (entry) =>
            entry.billing_group_route_id === route.id &&
            entry.channel_id === channelID
        )
      ) {
        return current
      }
      const entries = current.route_channels.filter(
        (entry) => entry.billing_group_route_id === route.id
      )
      const entry: BillingGroupChannel = {
        id: 0,
        billing_group_route_id: route.id,
        channel_id: channelID,
        priority: 100 - entries.length * 10,
        weight: 100,
        max_attempts: 1,
        enabled: true,
        cost_factor: 1,
      }
      current.route_channels.push(entry)
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
      const entries = current.route_channels
        .filter((entry) => entry.billing_group_route_id === route.id)
        .sort((a, b) => b.priority - a.priority)
      const index = entries.findIndex(
        (entry) => entry.channel_id === target.channel_id
      )
      const other = entries[index + direction]
      if (!other) return current
      const targetEntry = entries[index]
      const priority = targetEntry.priority
      targetEntry.priority = other.priority
      other.priority = priority
      return current
    })
  }

  if (!config) {
    return <div className='p-6 text-sm'>{t('Loading')}</div>
  }

  return (
    <div className='mx-auto w-full max-w-7xl space-y-5 p-4 md:p-6'>
      <header className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <h1 className='text-xl font-semibold'>{t('Channel routing')}</h1>
          <p className='text-muted-foreground text-sm'>
            {t('Configure ordered channels for each billing group')}
          </p>
        </div>
        <Button
          onClick={() => saveMutation.mutate(config)}
          disabled={!draft || saveMutation.isPending}
        >
          <Save className='size-4' />
          {t('Save')}
        </Button>
      </header>

      <Tabs defaultValue='routes'>
        <TabsList>
          <TabsTrigger value='routes'>{t('Billing group routes')}</TabsTrigger>
          <TabsTrigger value='errors'>{t('Error mappings')}</TabsTrigger>
          <TabsTrigger value='monitoring'>
            {t('Channel monitoring')}
          </TabsTrigger>
        </TabsList>

        <TabsContent value='routes' className='space-y-4 pt-4'>
          {config.routes.map((route, routeIndex) => (
            <section key={route.id} className='border-b pb-5'>
              <div className='grid gap-3 md:grid-cols-[1fr_1fr_180px_auto_auto] md:items-end'>
                <div className='space-y-1.5'>
                  <Label>{t('Billing group')}</Label>
                  <NativeSelect
                    className='w-full'
                    value={route.billing_group}
                    disabled={groupsQuery.isLoading}
                    onChange={(event) =>
                      updateRoute(
                        routeIndex,
                        'billing_group',
                        event.target.value
                      )
                    }
                  >
                    <NativeSelectOption value=''>
                      {groupsQuery.isLoading
                        ? t('Loading')
                        : t('Select a group')}
                    </NativeSelectOption>
                    {billingGroups.map((group) => (
                      <NativeSelectOption key={group} value={group}>
                        {group}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </div>
                <div className='space-y-1.5'>
                  <Label>{t('Display name')}</Label>
                  <Input
                    value={route.name}
                    onChange={(event) =>
                      updateRoute(routeIndex, 'name', event.target.value)
                    }
                  />
                </div>
                <div className='space-y-1.5'>
                  <Label>{t('Routing strategy')}</Label>
                  <NativeSelect
                    className='w-full'
                    value={route.mode}
                    onChange={(event) =>
                      updateRoute(
                        routeIndex,
                        'mode',
                        event.target.value as RoutingMode
                      )
                    }
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
                    onCheckedChange={(checked) =>
                      updateRoute(routeIndex, 'enabled', checked)
                    }
                  />
                  <Label>{t('Enabled')}</Label>
                </div>
                <Button
                  variant='ghost'
                  size='icon'
                  title={t('Delete')}
                  onClick={() => removeRoute(routeIndex)}
                >
                  <Trash2 className='size-4' />
                </Button>
              </div>

              <div className='mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
                <NumberField
                  label={t('Maximum total attempts')}
                  value={route.max_total_attempts}
                  min={1}
                  onChange={(value) =>
                    updateRoute(routeIndex, 'max_total_attempts', value)
                  }
                />
                <NumberField
                  label={t('Total timeout (ms)')}
                  value={route.total_timeout_ms}
                  min={1000}
                  step={1000}
                  onChange={(value) =>
                    updateRoute(routeIndex, 'total_timeout_ms', value)
                  }
                />
                <NumberField
                  label={t('Circuit failure threshold')}
                  value={route.circuit_failure_threshold}
                  min={1}
                  onChange={(value) =>
                    updateRoute(routeIndex, 'circuit_failure_threshold', value)
                  }
                />
                <NumberField
                  label={t('Circuit cooldown (seconds)')}
                  value={route.circuit_cooldown_seconds}
                  min={1}
                  onChange={(value) =>
                    updateRoute(routeIndex, 'circuit_cooldown_seconds', value)
                  }
                />
              </div>

              <div className='mt-4 overflow-x-auto'>
                <table className='w-full min-w-[760px] text-sm'>
                  <thead className='text-muted-foreground border-b text-left'>
                    <tr>
                      <th className='py-2 pr-3'>{t('Order')}</th>
                      <th className='py-2 pr-3'>{t('Channel')}</th>
                      <th className='py-2 pr-3'>
                        {t('Attempts on this channel')}
                      </th>
                      <th className='py-2 pr-3'>{t('Weight')}</th>
                      <th className='py-2 pr-3'>{t('Cost factor')}</th>
                      <th className='py-2 text-right'>{t('Actions')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {routeEntries(route).map((entry, index) => (
                      <tr
                        key={`${entry.billing_group_route_id}-${entry.channel_id}`}
                        className='border-b'
                      >
                        <td className='py-2 pr-3'>{index + 1}</td>
                        <td className='py-2 pr-3 font-medium'>
                          {channelByID.get(entry.channel_id)?.name ??
                            `#${entry.channel_id}`}
                        </td>
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
                        <td className='py-2 pr-3'>
                          <Input
                            className='w-24'
                            type='number'
                            min={0}
                            value={entry.weight}
                            onChange={(event) =>
                              updateEntry(entry, {
                                weight: Number(event.target.value),
                              })
                            }
                          />
                        </td>
                        <td className='py-2 pr-3'>
                          <Input
                            className='w-24'
                            type='number'
                            min={0.001}
                            step={0.01}
                            value={entry.cost_factor}
                            onChange={(event) =>
                              updateEntry(entry, {
                                cost_factor: Number(event.target.value),
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
                              onClick={() => moveEntry(route, entry, -1)}
                            >
                              <ArrowUp className='size-4' />
                            </Button>
                            <Button
                              variant='ghost'
                              size='icon'
                              title={t('Move down')}
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
                    ))}
                  </tbody>
                </table>
              </div>
              <NativeSelect
                className='mt-3 w-full max-w-sm'
                value=''
                onChange={(event) =>
                  addChannel(route, Number(event.target.value))
                }
              >
                <NativeSelectOption value=''>
                  {t('Add channel')}
                </NativeSelectOption>
                {channels
                  .filter(
                    (channel) =>
                      channel.group
                        .split(',')
                        .map((group) => group.trim())
                        .includes(route.billing_group.trim()) &&
                      !routeEntries(route).some(
                        (entry) => entry.channel_id === channel.id
                      )
                  )
                  .map((channel) => (
                    <NativeSelectOption key={channel.id} value={channel.id}>
                      {channel.name} (#{channel.id})
                    </NativeSelectOption>
                  ))}
              </NativeSelect>
            </section>
          ))}
          <Button variant='outline' onClick={addRoute}>
            <Plus className='size-4' />
            {t('Add billing group route')}
          </Button>
        </TabsContent>

        <TabsContent value='errors' className='space-y-3 pt-4'>
          {config.error_mappings.map((mapping, index) => (
            <div
              key={mapping.id}
              className='grid gap-3 border-b pb-3 md:grid-cols-[1fr_1fr_120px_140px_1fr_1fr_auto] md:items-end'
            >
              <div className='space-y-1.5'>
                <Label>{t('Channel')}</Label>
                <NativeSelect
                  className='w-full'
                  value={mapping.channel_id}
                  onChange={(event) => {
                    const channel = channelByID.get(Number(event.target.value))
                    updateConfig((current) => {
                      current.error_mappings[index] = {
                        ...mapping,
                        channel_id: channel?.id ?? 0,
                        channel_type: channel?.type ?? 0,
                      }
                      return current
                    })
                  }}
                >
                  <NativeSelectOption value={0}>
                    {t('All channels')}
                  </NativeSelectOption>
                  {channels.map((channel) => (
                    <NativeSelectOption key={channel.id} value={channel.id}>
                      {channel.name} (#{channel.id})
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>
              <div className='space-y-1.5'>
                <Label>{t('Upstream error code')}</Label>
                <Input
                  value={mapping.raw_code}
                  onChange={(event) =>
                    updateConfig((current) => {
                      current.error_mappings[index] = {
                        ...mapping,
                        raw_code: event.target.value,
                      }
                      return current
                    })
                  }
                />
              </div>
              <NumberField
                label={t('HTTP status')}
                value={mapping.status_code}
                onChange={(value) =>
                  updateConfig((current) => {
                    current.error_mappings[index] = {
                      ...mapping,
                      status_code: value,
                    }
                    return current
                  })
                }
              />
              <NumberField
                label={t('AllToken code')}
                value={mapping.alltoken_code}
                min={100000}
                onChange={(value) =>
                  updateConfig((current) => {
                    current.error_mappings[index] = {
                      ...mapping,
                      alltoken_code: value,
                    }
                    return current
                  })
                }
              />
              <div className='space-y-1.5'>
                <Label>{t('Failure scope')}</Label>
                <NativeSelect
                  className='w-full'
                  value={mapping.failure_scope}
                  onChange={(event) =>
                    updateConfig((current) => {
                      current.error_mappings[index] = {
                        ...mapping,
                        failure_scope: event.target
                          .value as UpstreamErrorMapping['failure_scope'],
                      }
                      return current
                    })
                  }
                >
                  {['request', 'credential', 'channel', 'provider'].map(
                    (scope) => (
                      <NativeSelectOption key={scope} value={scope}>
                        {scope}
                      </NativeSelectOption>
                    )
                  )}
                </NativeSelect>
              </div>
              <div className='space-y-1.5'>
                <Label>{t('Action')}</Label>
                <NativeSelect
                  className='w-full'
                  value={mapping.action}
                  onChange={(event) =>
                    updateConfig((current) => {
                      current.error_mappings[index] = {
                        ...mapping,
                        action: event.target
                          .value as UpstreamErrorMapping['action'],
                      }
                      return current
                    })
                  }
                >
                  {[
                    'none',
                    'retry_channel',
                    'switch_channel',
                    'retry_later',
                    'abort',
                    'manual',
                  ].map((action) => (
                    <NativeSelectOption key={action} value={action}>
                      {action}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>
              <Button
                variant='ghost'
                size='icon'
                title={t('Delete')}
                onClick={() =>
                  updateConfig((current) => ({
                    ...current,
                    error_mappings: current.error_mappings.filter(
                      (_, mappingIndex) => mappingIndex !== index
                    ),
                  }))
                }
              >
                <Trash2 className='size-4' />
              </Button>
            </div>
          ))}
          <Button
            variant='outline'
            onClick={() =>
              updateConfig((current) => ({
                ...current,
                error_mappings: [...current.error_mappings, emptyMapping()],
              }))
            }
          >
            <Plus className='size-4' />
            {t('Add error mapping')}
          </Button>
        </TabsContent>

        <TabsContent value='monitoring' className='pt-4'>
          <div className='mb-5 flex flex-wrap items-center justify-between gap-3'>
            <div>
              <h3 className='font-medium'>{t('Channel monitoring')}</h3>
              <p className='text-muted-foreground text-sm'>
                {t('Live channel routing health and failover metrics')}
              </p>
            </div>
            {monitoringQuery.data?.grafana_url ? (
              <Button
                variant='outline'
                render={
                  <a
                    href={monitoringQuery.data.grafana_url}
                    target='_blank'
                    rel='noreferrer'
                  />
                }
              >
                <ExternalLink className='size-4' />
                {t('Open Grafana')}
              </Button>
            ) : null}
          </div>
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            {[
              [
                t('Request RPS'),
                monitoringQuery.data?.metrics.request_rps ?? 0,
              ],
              [
                t('Error rate'),
                `${((monitoringQuery.data?.metrics.error_rate ?? 0) * 100).toFixed(2)}%`,
              ],
              [
                t('P95 latency'),
                `${(monitoringQuery.data?.metrics.p95_latency_seconds ?? 0).toFixed(2)}s`,
              ],
              [
                t('Channel switches'),
                monitoringQuery.data?.metrics.channel_switches ?? 0,
              ],
              [t('In flight'), monitoringQuery.data?.metrics.in_flight ?? 0],
              [
                t('Open circuits'),
                monitoringQuery.data?.metrics.open_circuits ?? 0,
              ],
            ].map(([label, value]) => (
              <div key={String(label)} className='border-b py-3'>
                <div className='text-muted-foreground text-sm'>{label}</div>
                <div className='mt-1 text-xl font-semibold'>{value}</div>
              </div>
            ))}
          </div>
          <div className='mt-6 space-y-2'>
            {monitoringQuery.data?.alerts.map((alert) => (
              <div key={alert.fingerprint} className='border-b py-3 text-sm'>
                <div className='font-medium'>
                  {alert.name}
                  {alert.channel_id ? ` · CH${alert.channel_id}` : ''}
                </div>
                <div className='text-muted-foreground'>{alert.summary}</div>
              </div>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
