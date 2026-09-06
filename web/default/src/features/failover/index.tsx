import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleOff, ExternalLink, Plus, Save, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getChannels } from '@/features/channels/api'

import {
  getFailoverConfig,
  getFailoverMonitoring,
  updateFailoverConfig,
} from './api'
import type {
  FailoverConfig,
  FailoverMonitoringMetrics,
  UpstreamErrorMapping,
} from './types'

let nextTemporaryID = -1

function createErrorMapping(): UpstreamErrorMapping {
  return {
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
  }
}

function NumberField(props: {
  label: string
  value: number
  min?: number
  onChange: (value: number) => void
}) {
  return (
    <div className='space-y-1.5'>
      <Label>{props.label}</Label>
      <Input
        type='number'
        min={props.min ?? 0}
        value={props.value}
        onChange={(event) => props.onChange(Number(event.target.value))}
      />
    </div>
  )
}

function getMonitoringMetrics(
  t: (key: string) => string,
  metrics?: FailoverMonitoringMetrics
) {
  return [
    [t('Request RPS'), metrics?.request_rps ?? 0],
    [t('Error rate'), `${((metrics?.error_rate ?? 0) * 100).toFixed(2)}%`],
    [t('P95 latency'), `${(metrics?.p95_latency_seconds ?? 0).toFixed(2)}s`],
    [t('Channel switches'), metrics?.channel_switches ?? 0],
    [t('In flight'), metrics?.in_flight ?? 0],
    [t('Open circuits'), metrics?.open_circuits ?? 0],
  ] as const
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
    refetchInterval: (query) =>
      query.state.data?.status === 'disabled' ? false : 10000,
  })
  const channelsQuery = useQuery({
    queryKey: ['channel-routing-channels'],
    queryFn: () => getChannels({ page_size: 1000, id_sort: true }),
  })
  const channels = useMemo(
    () => channelsQuery.data?.data?.items ?? [],
    [channelsQuery.data]
  )
  const channelByID = useMemo(
    () => new Map(channels.map((channel) => [channel.id, channel])),
    [channels]
  )
  const config = draft ?? configQuery.data
  const monitoringDisabled = monitoringQuery.data?.status === 'disabled'
  const monitoringUnavailable = monitoringQuery.isError
  const monitoringLoading = monitoringQuery.isPending
  const saveMutation = useMutation({
    mutationFn: updateFailoverConfig,
    onSuccess: async () => {
      toast.success(t('Error mappings saved'))
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
    if (config) setDraft(updater(structuredClone(config)))
  }

  if (!config) return <div className='p-6 text-sm'>{t('Loading')}</div>

  return (
    <div className='mx-auto w-full max-w-7xl space-y-5 p-4 md:p-6'>
      <header className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <h1 className='text-xl font-semibold'>{t('Monitoring & Alerts')}</h1>
          <p className='text-muted-foreground text-sm'>
            {t('Manage channel error mappings and monitor channel health.')}
          </p>
        </div>
        <Button
          onClick={() => saveMutation.mutate(config)}
          disabled={!draft || saveMutation.isPending}
        >
          <Save className='size-4' /> {t('Save')}
        </Button>
      </header>

      <Tabs defaultValue='errors'>
        <TabsList>
          <TabsTrigger value='errors'>{t('Error mappings')}</TabsTrigger>
          <TabsTrigger value='monitoring'>
            {t('Channel monitoring')}
          </TabsTrigger>
        </TabsList>
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
                onChange={(statusCode) =>
                  updateConfig((current) => {
                    current.error_mappings[index] = {
                      ...mapping,
                      status_code: statusCode,
                    }
                    return current
                  })
                }
              />
              <NumberField
                label={t('AllToken code')}
                value={mapping.alltoken_code}
                min={100000}
                onChange={(alltoken_code) =>
                  updateConfig((current) => {
                    current.error_mappings[index] = {
                      ...mapping,
                      alltoken_code,
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
                error_mappings: [
                  ...current.error_mappings,
                  createErrorMapping(),
                ],
              }))
            }
          >
            <Plus className='size-4' /> {t('Add error mapping')}
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
            {monitoringQuery.data?.status !== 'disabled' &&
            monitoringQuery.data?.grafana_url ? (
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
                <ExternalLink className='size-4' /> {t('Open Grafana')}
              </Button>
              ) : null}
          </div>
          {monitoringDisabled && (
            <div className='space-y-4 border p-4'>
              <div className='flex items-start gap-3'>
                <CircleOff className='mt-0.5 size-5 shrink-0 text-muted-foreground' />
                <div className='space-y-1'>
                  <div className='font-medium'>{t('Monitoring disabled')}</div>
                  <p className='text-muted-foreground text-sm'>
                    {t('Prometheus, Alertmanager, Grafana, metrics listener, and structured event logs are disabled.')}
                  </p>
                </div>
              </div>
              <div className='grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3'>
                {[
                  [t('Prometheus'), t('Disabled')],
                  [t('Alertmanager'), t('Disabled')],
                  [t('Grafana'), t('Disabled')],
                  [t('Metrics listener'), t('Disabled')],
                  [t('Structured event logs'), t('Disabled')],
                  [t('Nginx logs'), t('Retained')],
                  [t('Channel probes'), t('Still running')],
                ].map(([label, value]) => (
                  <div key={label} className='border-b py-2'>
                    <div className='text-muted-foreground'>{label}</div>
                    <div className='font-medium'>{value}</div>
                  </div>
                ))}
              </div>
            </div>
          )}
          {!monitoringDisabled && monitoringUnavailable && (
            <div className='border p-4 text-sm'>
              <div className='font-medium'>{t('Monitoring data unavailable')}</div>
              <div className='text-muted-foreground'>
                {t('The monitoring status endpoint could not be reached.')}
              </div>
            </div>
          )}
          {!monitoringDisabled && !monitoringUnavailable && monitoringLoading && (
            <div className='border p-4 text-sm text-muted-foreground'>
              {t('Loading')}
            </div>
          )}
          {!monitoringDisabled &&
            !monitoringUnavailable &&
            !monitoringLoading && (
            <>
              <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
                {getMonitoringMetrics(t, monitoringQuery.data?.metrics).map(
                  ([label, value]) => (
                    <div key={label} className='border-b py-3'>
                      <div className='text-muted-foreground text-sm'>{label}</div>
                      <div className='mt-1 text-xl font-semibold'>{value}</div>
                    </div>
                  )
                )}
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
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
