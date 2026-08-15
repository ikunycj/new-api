import {
  Alert02Icon,
  ArrowUpRight01Icon,
  Chart01Icon,
  Refresh01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import { getFailoverMonitoring } from './api'
import type { FailoverMonitoringAlert, FailoverMonitoringSource } from './types'

type MetricItem = {
  label: string
  value: string
  emphasis?: 'danger' | 'warning'
}

const sourceNames: Record<FailoverMonitoringSource['name'], string> = {
  prometheus: 'Prometheus',
  alertmanager: 'Alertmanager',
  grafana: 'Grafana',
}

const sourceStatusLabels: Record<FailoverMonitoringSource['status'], string> = {
  healthy: 'Healthy',
  degraded: 'Degraded',
  unavailable: 'Unavailable',
  not_configured: 'Not configured',
  pending: 'Pending',
}

const alertStatusLabels: Record<string, string> = {
  active: 'Firing',
  firing: 'Firing',
  suppressed: 'Suppressed',
  pending: 'Pending',
  unprocessed: 'Pending',
}

const severityLabels: Record<string, string> = {
  critical: 'Critical',
  warning: 'Warning',
}

function sourceStatusVariant(
  status: FailoverMonitoringSource['status']
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'healthy') return 'default'
  if (status === 'degraded' || status === 'pending') return 'secondary'
  if (status === 'unavailable') return 'destructive'
  return 'outline'
}

function alertStatusVariant(
  alert: FailoverMonitoringAlert
): 'destructive' | 'secondary' | 'outline' {
  if (alert.status === 'suppressed') return 'outline'
  if (alert.severity === 'critical') return 'destructive'
  return 'secondary'
}

function activeAlertEmphasis(
  alerts: FailoverMonitoringAlert[]
): MetricItem['emphasis'] {
  if (alerts.some((alert) => alert.severity === 'critical')) return 'danger'
  if (alerts.length > 0) return 'warning'
  return undefined
}

function MonitoringMetric(props: { item: MetricItem }) {
  return (
    <Card size='sm' className='min-h-24 rounded-lg'>
      <CardHeader>
        <CardTitle className='text-muted-foreground text-xs font-medium'>
          {props.item.label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div
          className={cn(
            'text-2xl font-semibold tabular-nums',
            props.item.emphasis === 'danger' && 'text-destructive',
            props.item.emphasis === 'warning' && 'text-warning'
          )}
        >
          {props.item.value}
        </div>
      </CardContent>
    </Card>
  )
}

function MonitoringSkeleton() {
  return (
    <div className='space-y-5'>
      <div className='grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-5'>
        {Array.from({ length: 10 }, (_, index) => (
          <Skeleton key={index} className='h-24 rounded-lg' />
        ))}
      </div>
      <Skeleton className='h-72 rounded-lg' />
      <Skeleton className='h-[640px] rounded-lg' />
    </div>
  )
}

export function FailoverMonitoring() {
  const { t } = useTranslation()
  const monitoringQuery = useQuery({
    queryKey: ['failover-monitoring'],
    queryFn: getFailoverMonitoring,
    refetchInterval: 10_000,
    staleTime: 5_000,
  })

  if (monitoringQuery.isLoading) {
    return <MonitoringSkeleton />
  }
  if (monitoringQuery.isError) {
    return (
      <Alert variant='destructive'>
        <HugeiconsIcon icon={Alert02Icon} />
        <AlertTitle>{t('Monitoring data unavailable')}</AlertTitle>
        <AlertDescription>{monitoringQuery.error.message}</AlertDescription>
      </Alert>
    )
  }
  if (!monitoringQuery.data) return null

  const snapshot = monitoringQuery.data
  const metricItems: MetricItem[] = [
    {
      label: t('Request rate'),
      value: `${snapshot.metrics.request_rps.toFixed(1)} req/s`,
    },
    {
      label: t('Error rate'),
      value: `${(snapshot.metrics.error_rate * 100).toFixed(2)}%`,
      emphasis: snapshot.metrics.error_rate >= 0.05 ? 'danger' : undefined,
    },
    {
      label: t('P95 latency'),
      value: `${(snapshot.metrics.p95_latency_seconds * 1000).toFixed(0)} ms`,
      emphasis:
        snapshot.metrics.p95_latency_seconds >= 2 ? 'warning' : undefined,
    },
    {
      label: t('In-flight requests'),
      value: snapshot.metrics.in_flight.toFixed(0),
    },
    {
      label: t('Active alerts'),
      value: snapshot.alerts.length.toFixed(0),
      emphasis: activeAlertEmphasis(snapshot.alerts),
    },
    {
      label: t('Open circuits'),
      value: snapshot.metrics.open_circuits.toFixed(0),
      emphasis: snapshot.metrics.open_circuits > 0 ? 'danger' : undefined,
    },
    {
      label: t('Cluster failovers (5m)'),
      value: snapshot.metrics.cluster_failovers.toFixed(0),
    },
    {
      label: t('Pool failovers (5m)'),
      value: snapshot.metrics.pool_failovers.toFixed(0),
    },
    {
      label: t('Database pool usage'),
      value: `${(snapshot.metrics.database_usage * 100).toFixed(1)}%`,
      emphasis: snapshot.metrics.database_usage >= 0.85 ? 'warning' : undefined,
    },
    {
      label: t('Redis timeouts (5m)'),
      value: snapshot.metrics.redis_timeouts.toFixed(0),
      emphasis: snapshot.metrics.redis_timeouts > 0 ? 'warning' : undefined,
    },
  ]

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex flex-wrap items-center gap-2'>
          {snapshot.sources.map((source) => (
            <Badge
              key={source.name}
              variant={sourceStatusVariant(source.status)}
            >
              {sourceNames[source.name]} ·{' '}
              {t(sourceStatusLabels[source.status])}
            </Badge>
          ))}
          <span className='text-muted-foreground text-xs tabular-nums'>
            {t('Last updated')}: {dayjs(snapshot.updated_at).format('HH:mm:ss')}
          </span>
        </div>
        <div className='flex items-center gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={() => monitoringQuery.refetch()}
            disabled={monitoringQuery.isFetching}
          >
            <HugeiconsIcon
              icon={Refresh01Icon}
              data-icon='inline-start'
              className={cn(monitoringQuery.isFetching && 'animate-spin')}
            />
            {t('Refresh')}
          </Button>
          {snapshot.grafana_url && (
            <Button
              variant='outline'
              render={
                <a
                  href={snapshot.grafana_url}
                  target='_blank'
                  rel='noreferrer'
                />
              }
            >
              <HugeiconsIcon
                icon={ArrowUpRight01Icon}
                data-icon='inline-start'
              />
              {t('Open Grafana')}
            </Button>
          )}
        </div>
      </div>

      <div className='grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-5'>
        {metricItems.map((item) => (
          <MonitoringMetric key={item.label} item={item} />
        ))}
      </div>

      <section className='space-y-3'>
        <div className='flex items-center justify-between'>
          <h2 className='text-base font-semibold'>{t('Active alerts')}</h2>
          <Badge variant='outline'>{snapshot.alerts.length}</Badge>
        </div>
        <div className='overflow-hidden rounded-lg border'>
          <div className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Severity')}</TableHead>
                  <TableHead>{t('Alert')}</TableHead>
                  <TableHead>{t('Summary')}</TableHead>
                  <TableHead>{t('Source')}</TableHead>
                  <TableHead>{t('Started at')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {snapshot.alerts.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='text-muted-foreground h-24 text-center'
                    >
                      {t('No active alerts')}
                    </TableCell>
                  </TableRow>
                ) : (
                  snapshot.alerts.map((alert) => (
                    <TableRow
                      key={
                        alert.fingerprint || `${alert.name}-${alert.started_at}`
                      }
                    >
                      <TableCell>
                        <Badge variant={alertStatusVariant(alert)}>
                          {t(alertStatusLabels[alert.status] || 'Unknown')}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={alertStatusVariant(alert)}>
                          {alert.severity
                            ? t(severityLabels[alert.severity] || 'Unknown')
                            : '-'}
                        </Badge>
                      </TableCell>
                      <TableCell className='font-medium'>
                        {alert.name}
                      </TableCell>
                      <TableCell className='min-w-64'>
                        <div>{alert.summary || '-'}</div>
                        {alert.description && (
                          <div className='text-muted-foreground mt-1 text-xs'>
                            {alert.description}
                          </div>
                        )}
                      </TableCell>
                      <TableCell className='whitespace-nowrap'>
                        {[
                          alert.instance,
                          alert.cluster_code && `C${alert.cluster_code}`,
                          alert.pool_tier && `P${alert.pool_tier}`,
                        ]
                          .filter(Boolean)
                          .join(' · ') || '-'}
                      </TableCell>
                      <TableCell className='whitespace-nowrap tabular-nums'>
                        {alert.started_at
                          ? dayjs(alert.started_at).format(
                              'YYYY-MM-DD HH:mm:ss'
                            )
                          : '-'}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </section>

      <section className='space-y-3'>
        <div className='flex items-center gap-2'>
          <HugeiconsIcon icon={Chart01Icon} className='size-4' />
          <h2 className='text-base font-semibold'>
            {t('Full Grafana dashboard')}
          </h2>
        </div>
        {snapshot.grafana_url ? (
          // The URL is operator-controlled and validated by the backend. Grafana
          // requires its normal same-origin script context to render dashboards.
          // oxlint-disable-next-line react/iframe-missing-sandbox
          <iframe
            title={t('Full Grafana dashboard')}
            src={snapshot.grafana_url}
            referrerPolicy='same-origin'
            className='bg-background h-[720px] min-h-[60vh] w-full rounded-lg border'
          />
        ) : (
          <Alert>
            <HugeiconsIcon icon={Chart01Icon} />
            <AlertTitle>{t('Grafana is not configured')}</AlertTitle>
          </Alert>
        )}
      </section>
    </div>
  )
}
