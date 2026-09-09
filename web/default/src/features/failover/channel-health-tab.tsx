import { useQuery } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { CircleOff, Info } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceLine,
  XAxis,
  YAxis,
} from 'recharts'

import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { getChannelHealthSnapshot } from './api'
import type { ChannelHealthSnapshot } from './types'

// The backend keeps health state in memory only, so there is no history
// endpoint to chart. We poll the snapshot and accumulate the series client
// side; it starts empty on mount and is lost on unmount, which is called out
// in the UI so nobody mistakes a short line for a young channel.
const POLL_INTERVAL_MS = 15_000
const MAX_HISTORY_POINTS = 120

const PROBE_ROUTE = '__probe__'

type HistoryPoint = {
  t: number
  // Keyed by `${channelId}:${route}`, matching seriesKey().
  [seriesKey: string]: number
}

function seriesKey(snapshot: ChannelHealthSnapshot) {
  return `${snapshot.channel_id}:${snapshot.route}`
}

function formatScore(value: number) {
  return value.toFixed(3)
}

function formatLatency(ms: number) {
  if (!ms || ms <= 0) return '—'
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.round(ms)}ms`
}

function formatClock(ts: number) {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(
    d.getMinutes()
  ).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

function relativeTime(iso: string, t: TFunction) {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return '—'
  const secs = Math.max(0, Math.round((Date.now() - then) / 1000))
  if (secs < 60) return t('{{n}}s ago', { n: secs })
  if (secs < 3600) return t('{{n}}m ago', { n: Math.round(secs / 60) })
  return t('{{n}}h ago', { n: Math.round(secs / 3600) })
}

// Score colour is advisory only: it never feeds back into routing.
function scoreTone(score: number, confident: boolean) {
  if (!confident) return 'text-muted-foreground'
  if (score >= 0.85) return 'text-emerald-600 dark:text-emerald-400'
  if (score >= 0.5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

const LINE_COLOURS = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
]

export function ChannelHealthTab() {
  const { t } = useTranslation()
  const [trafficFilter, setTrafficFilter] = useState<'all' | 'real' | 'probe'>(
    'all'
  )

  const healthQuery = useQuery({
    queryKey: ['channel-health-snapshot'],
    queryFn: getChannelHealthSnapshot,
    refetchInterval: POLL_INTERVAL_MS,
    refetchOnWindowFocus: false,
  })

  // Client-accumulated trend. useRef keeps the series stable across the
  // re-renders that each poll triggers.
  const historyRef = useRef<HistoryPoint[]>([])
  const [history, setHistory] = useState<HistoryPoint[]>([])
  const lastStampRef = useRef<number>(0)

  const snapshots = useMemo(
    () => healthQuery.data?.channels ?? [],
    [healthQuery.data]
  )

  useEffect(() => {
    if (snapshots.length === 0) return
    // dataUpdatedAt changes on every successful fetch, including ones that
    // return identical values; dedupe so a refetch storm cannot inflate the
    // series.
    const stamp = healthQuery.dataUpdatedAt
    if (!stamp || stamp === lastStampRef.current) return
    lastStampRef.current = stamp

    const point: HistoryPoint = { t: stamp }
    for (const s of snapshots) point[seriesKey(s)] = s.score

    const next = [...historyRef.current, point].slice(-MAX_HISTORY_POINTS)
    historyRef.current = next
    setHistory(next)
  }, [snapshots, healthQuery.dataUpdatedAt])

  const filtered = useMemo(() => {
    if (trafficFilter === 'real') {
      return snapshots.filter((s) => s.route !== PROBE_ROUTE)
    }
    if (trafficFilter === 'probe') {
      return snapshots.filter((s) => s.route === PROBE_ROUTE)
    }
    return snapshots
  }, [snapshots, trafficFilter])

  // Latency is scored against each channel's own baseline, so the score never
  // reveals that one upstream is simply slower than another. This ranking adds
  // the cross-channel view the score deliberately omits.
  const latencyRanking = useMemo(() => {
    const withLatency = filtered.filter((s) => s.last_latency_ms > 0)
    if (withLatency.length === 0) return { rows: [], max: 0 }
    const rows = [...withLatency].sort(
      (a, b) => a.last_latency_ms - b.last_latency_ms
    )
    return { rows, max: rows.at(-1)?.last_latency_ms ?? 0 }
  }, [filtered])

  const byFamily = useMemo(() => {
    const groups = new Map<string, ChannelHealthSnapshot[]>()
    for (const s of filtered) {
      const list = groups.get(s.family) ?? []
      list.push(s)
      groups.set(s.family, list)
    }
    for (const list of groups.values()) list.sort((a, b) => b.score - a.score)
    return [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]))
  }, [filtered])

  const chartConfig = useMemo(() => {
    const config: ChartConfig = {}
    filtered.forEach((s, i) => {
      config[seriesKey(s)] = {
        label: `CH${s.channel_id}${s.route === PROBE_ROUTE ? ' (probe)' : ''}`,
        color: LINE_COLOURS[i % LINE_COLOURS.length],
      }
    })
    return config
  }, [filtered])

  const data = healthQuery.data

  if (healthQuery.isLoading) {
    return (
      <div className='text-muted-foreground border p-4 text-sm'>
        {t('Loading')}
      </div>
    )
  }

  if (healthQuery.isError) {
    return (
      <div className='border p-4 text-sm'>
        <div className='font-medium'>{t('Health data unavailable')}</div>
        <div className='text-muted-foreground'>
          {healthQuery.error instanceof Error
            ? healthQuery.error.message
            : t('The channel health endpoint could not be reached.')}
        </div>
      </div>
    )
  }

  if (!data?.enabled) {
    return (
      <div className='space-y-4 border p-4'>
        <div className='flex items-start gap-3'>
          <CircleOff className='text-muted-foreground mt-0.5 size-5 shrink-0' />
          <div className='space-y-1'>
            <div className='font-medium'>
              {t('Channel health scoring is disabled')}
            </div>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Enable it under System Settings → Models → Routing Reliability. While disabled no samples are recorded and routing is unaffected.'
              )}
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h3 className='font-medium'>{t('Channel health scores')}</h3>
          <p className='text-muted-foreground text-sm'>
            {data.active
              ? t(
                  'Mode is active: scores influence weighted routes. Priority routes are unaffected.'
                )
              : t(
                  'Mode is observe: scores are recorded but never influence routing.'
                )}
          </p>
        </div>
        <NativeSelect
          value={trafficFilter}
          onChange={(e) =>
            setTrafficFilter(e.target.value as 'all' | 'real' | 'probe')
          }
          className='w-44'
        >
          <NativeSelectOption value='all'>
            {t('All traffic')}
          </NativeSelectOption>
          <NativeSelectOption value='real'>
            {t('Real traffic only')}
          </NativeSelectOption>
          <NativeSelectOption value='probe'>
            {t('Probe traffic only')}
          </NativeSelectOption>
        </NativeSelect>
      </div>

      <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        {[
          [t('Mode'), data.active ? t('Active') : t('Observe')],
          [
            t('Probing'),
            data.config.probe_enabled ? t('Enabled') : t('Disabled'),
          ],
          [t('Min samples'), String(data.config.min_samples)],
          [t('Half-life'), `${data.config.half_life_seconds}s`],
        ].map(([label, value]) => (
          <div key={label} className='border-b py-3'>
            <div className='text-muted-foreground text-sm'>{label}</div>
            <div className='mt-1 text-xl font-semibold'>{value}</div>
          </div>
        ))}
      </div>

      {data.probe_stats && (
        <div className='text-muted-foreground text-sm'>
          {t(
            'Last probe run {{when}}: {{probed}} probed, {{skipped}} skipped (recently active), {{failed}} failed.',
            {
              when: relativeTime(data.probe_stats.last_run_at, t),
              probed: data.probe_stats.last_probed,
              skipped: data.probe_stats.last_skipped,
              failed: data.probe_stats.last_failed,
            }
          )}
        </div>
      )}

      <div>
        <div className='mb-2 flex items-baseline justify-between gap-3'>
          <h4 className='font-medium'>{t('Score trend')}</h4>
          <span className='text-muted-foreground text-xs'>
            {t(
              'Collected in this browser tab since it was opened, every {{n}}s. Not persisted server side.',
              { n: POLL_INTERVAL_MS / 1000 }
            )}
          </span>
        </div>
        {history.length < 2 ? (
          <div className='text-muted-foreground border p-4 text-sm'>
            {t(
              'Waiting for the second sample. The trend needs at least two polls.'
            )}
          </div>
        ) : (
          <ChartContainer config={chartConfig} className='h-64 w-full'>
            <LineChart data={history} margin={{ left: 4, right: 12, top: 8 }}>
              <CartesianGrid vertical={false} strokeDasharray='3 3' />
              <XAxis
                dataKey='t'
                tickFormatter={formatClock}
                tickLine={false}
                axisLine={false}
                fontSize={11}
                minTickGap={40}
              />
              <YAxis
                domain={[0, 1]}
                ticks={[0, 0.25, 0.5, 0.75, 1]}
                tickLine={false}
                axisLine={false}
                fontSize={11}
                width={32}
              />
              <ReferenceLine
                y={0.5}
                stroke='var(--muted-foreground)'
                strokeDasharray='2 4'
              />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(v) => formatClock(Number(v))}
                  />
                }
              />
              {filtered.map((s) => (
                <Line
                  key={seriesKey(s)}
                  dataKey={seriesKey(s)}
                  type='monotone'
                  stroke={`var(--color-${seriesKey(s)})`}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={false}
                  connectNulls
                />
              ))}
            </LineChart>
          </ChartContainer>
        )}
      </div>

      <div>
        <div className='mb-2 flex items-baseline gap-2'>
          <h4 className='font-medium'>{t('Latency comparison')}</h4>
          <span className='text-muted-foreground text-xs'>
            {t(
              'The latency score grades each channel against its own baseline, so a consistently slow channel still scores well. This ranking is the cross-channel view.'
            )}
          </span>
        </div>
        {latencyRanking.rows.length === 0 ? (
          <div className='text-muted-foreground border p-4 text-sm'>
            {t('No latency samples yet.')}
          </div>
        ) : (
          <div className='space-y-1'>
            {latencyRanking.rows.map((s) => (
              <div
                key={seriesKey(s)}
                className='flex items-center gap-3 border-b py-2 text-sm'
              >
                <span className='w-28 shrink-0 font-medium'>
                  CH{s.channel_id}
                  {s.route === PROBE_ROUTE && (
                    <span className='text-muted-foreground ml-1 text-xs'>
                      {t('probe')}
                    </span>
                  )}
                </span>
                <div className='bg-muted h-2 flex-1 overflow-hidden'>
                  <div
                    className='bg-foreground/60 h-full'
                    style={{
                      width: `${Math.max(
                        2,
                        (s.last_latency_ms / latencyRanking.max) * 100
                      )}%`,
                    }}
                  />
                </div>
                <span className='w-20 shrink-0 text-right tabular-nums'>
                  {formatLatency(s.last_latency_ms)}
                </span>
                <span className='text-muted-foreground w-28 shrink-0 text-right text-xs tabular-nums'>
                  {t('base {{v}}', { v: formatLatency(s.latency_baseline_ms) })}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {byFamily.map(([family, rows]) => (
        <div key={family}>
          <div className='mb-2 flex items-baseline gap-2'>
            <h4 className='font-medium'>
              {t('Family')}: {family}
            </h4>
            <span className='text-muted-foreground text-xs'>
              {t(
                'Scores are only comparable within a family — Claude competes with Claude, GPT with GPT.'
              )}
            </span>
          </div>
          <div className='overflow-x-auto'>
            <table className='w-full text-sm'>
              <thead>
                <tr className='text-muted-foreground border-b text-left'>
                  <th className='py-2 pr-3 font-normal'>{t('Channel')}</th>
                  <th className='py-2 pr-3 font-normal'>{t('Source')}</th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Score')}
                  </th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Availability')}
                  </th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Latency score')}
                  </th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Latency')}
                  </th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Success')}
                  </th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Failure')}
                  </th>
                  <th className='py-2 pr-3 text-right font-normal'>
                    {t('Samples')}
                  </th>
                  <th className='py-2 pr-3 font-normal'>{t('Confidence')}</th>
                  <th className='py-2 text-right font-normal'>
                    {t('Updated')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((s) => (
                  <tr key={seriesKey(s)} className='border-b'>
                    <td className='py-2 pr-3 font-medium'>CH{s.channel_id}</td>
                    <td className='text-muted-foreground py-2 pr-3 text-xs'>
                      {s.route === PROBE_ROUTE ? t('probe') : s.route}
                    </td>
                    <td
                      className={`py-2 pr-3 text-right font-semibold tabular-nums ${scoreTone(
                        s.score,
                        s.confident
                      )}`}
                    >
                      {formatScore(s.score)}
                    </td>
                    <td className='py-2 pr-3 text-right tabular-nums'>
                      {formatScore(s.availability)}
                    </td>
                    <td className='py-2 pr-3 text-right tabular-nums'>
                      {formatScore(s.latency_score)}
                    </td>
                    <td className='py-2 pr-3 text-right tabular-nums'>
                      {formatLatency(s.last_latency_ms)}
                    </td>
                    <td className='py-2 pr-3 text-right tabular-nums'>
                      {s.successes}
                    </td>
                    <td className='py-2 pr-3 text-right tabular-nums'>
                      {s.failures}
                    </td>
                    <td className='py-2 pr-3 text-right tabular-nums'>
                      {s.samples.toFixed(1)}
                    </td>
                    <td className='py-2 pr-3'>
                      {s.confident ? (
                        <span className='text-muted-foreground text-xs'>
                          {t('confident')}
                        </span>
                      ) : (
                        <span
                          className='inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400'
                          title={t(
                            'Fewer than {{n}} samples: the score is blended toward the neutral prior and should not be trusted yet.',
                            { n: data.config.min_samples }
                          )}
                        >
                          <Info className='size-3' />
                          {t('low samples')}
                        </span>
                      )}
                    </td>
                    <td className='text-muted-foreground py-2 text-right text-xs'>
                      {relativeTime(s.updated_at, t)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ))}

      {filtered.length === 0 && (
        <div className='text-muted-foreground border p-4 text-sm'>
          {t('No health samples recorded yet for this filter.')}
        </div>
      )}
    </div>
  )
}
