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
import { Activity, AlertTriangle, Play, RefreshCw, Square } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useChatPresets } from '@/features/chat/hooks/use-chat-presets'
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  LOAD_TEST_DEFAULT_DURATION_SECONDS,
  LOAD_TEST_DEFAULT_RPS,
  LOAD_TEST_MAX_CONCURRENCY,
  LOAD_TEST_MAX_DURATION_SECONDS,
  LOAD_TEST_MAX_REQUESTS,
  LOAD_TEST_MAX_RPS,
  LOAD_TEST_MIN_DURATION_SECONDS,
  LOAD_TEST_MIN_RPS,
  LOAD_TEST_MODEL,
  LOAD_TEST_MODELS,
  getLoadTestChannelStats,
  loadLoadTestKeys,
  sendLoadTestRequest,
  type LoadTestChannelStats,
  type LoadTestKey,
  type LoadTestRoutingStrategy,
  type LoadTestRequestResult,
} from './api'

const LOAD_TEST_ROUTING_STRATEGIES: Array<{
  value: LoadTestRoutingStrategy
  label: string
}> = [
  { value: 'cost_first', label: 'Cost first' },
  { value: 'balanced', label: 'Balanced cost and stability' },
  { value: 'stability_first', label: 'Stability first' },
  { value: 'pro_cost_first', label: 'Claude Pro cost first' },
  { value: 'pro_stability_first', label: 'Claude Pro stability first' },
]

type LoadTestSlot = {
  id: string
  keyId: string
  weight: number
  strategy: LoadTestRoutingStrategy
}

type RunStatus = 'idle' | 'loading-keys' | 'running' | 'complete'

type RunStats = {
  completed: number
  failures: number
  latencies: number[]
  successes: number
  statusCodes: Record<string, number>
  errorCodes: Record<string, number>
  keyCounts: Record<string, number>
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
}

const EMPTY_STATS: RunStats = {
  completed: 0,
  failures: 0,
  latencies: [],
  successes: 0,
  statusCodes: {},
  errorCodes: {},
  keyCounts: {},
  inputTokens: 0,
  outputTokens: 0,
  cacheReadTokens: 0,
  cacheWriteTokens: 0,
}

function makeRunId() {
  return `demo-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}

function percentile(values: number[], ratio: number) {
  if (values.length === 0) return 0
  const sorted = [...values].sort((a, b) => a - b)
  return sorted[
    Math.min(sorted.length - 1, Math.ceil(sorted.length * ratio) - 1)
  ]
}

function incrementCounter(counter: Record<string, number>, key: string) {
  counter[key] = (counter[key] ?? 0) + 1
}

function formatDuration(milliseconds: number) {
  return `${Math.max(0, Math.floor(milliseconds / 1000))}s`
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className='bg-muted/40 rounded-lg px-3 py-2'>
      <div className='text-muted-foreground text-xs'>{label}</div>
      <div className='mt-1 text-lg font-semibold tabular-nums'>{value}</div>
    </div>
  )
}

function CounterList({
  counters,
  emptyLabel,
}: {
  counters: Record<string, number>
  emptyLabel: string
}) {
  const entries = Object.entries(counters).sort(([, a], [, b]) => b - a)
  if (entries.length === 0) {
    return <div className='text-muted-foreground text-sm'>{emptyLabel}</div>
  }

  return (
    <div className='space-y-2'>
      {entries.map(([key, count]) => (
        <div
          className='flex items-center justify-between gap-3 text-sm'
          key={key}
        >
          <code className='truncate text-xs'>{key}</code>
          <Badge variant='secondary'>{count}</Badge>
        </div>
      ))}
    </div>
  )
}

export function LoadTestDemo() {
  const { t } = useTranslation()
  const { serverAddress } = useChatPresets()
  const quotaPerUnit = useSystemConfigStore(
    (state) => state.config.currency.quotaPerUnit
  )
  const [keys, setKeys] = useState<LoadTestKey[]>([])
  const [loadTestSlots, setLoadTestSlots] = useState<LoadTestSlot[]>([
    { id: 'slot-a', keyId: '', weight: 34, strategy: 'cost_first' },
    { id: 'slot-b', keyId: '', weight: 33, strategy: 'balanced' },
    { id: 'slot-c', keyId: '', weight: 33, strategy: 'stability_first' },
  ])
  const [selectedModel, setSelectedModel] = useState(LOAD_TEST_MODEL)
  const [durationSeconds, setDurationSeconds] = useState(
    LOAD_TEST_DEFAULT_DURATION_SECONDS
  )
  const [requestsPerSecond, setRequestsPerSecond] = useState(
    LOAD_TEST_DEFAULT_RPS
  )
  const [promptCache, setPromptCache] = useState(true)
  const [status, setStatus] = useState<RunStatus>('idle')
  const [stats, setStats] = useState<RunStats>(EMPTY_STATS)
  const [runId, setRunId] = useState('')
  const [elapsed, setElapsed] = useState(0)
  const [channelStats, setChannelStats] = useState<LoadTestChannelStats[]>([])
  const runAbortRef = useRef<AbortController | null>(null)
  const requestIdsRef = useRef<string[]>([])
  const activeRunIdRef = useRef('')
  const runStartedAtRef = useRef(0)

  const loadKeys = useCallback(async () => {
    setStatus('loading-keys')
    try {
      const loadedKeys = await loadLoadTestKeys()
      setKeys(loadedKeys)
      setLoadTestSlots((current) =>
        current.map((slot, index) => {
          const currentKeyStillExists = loadedKeys.some(
            (key) => String(key.id) === slot.keyId
          )
          const fallbackKey = loadedKeys[index]
          let keyId = ''
          if (currentKeyStillExists) {
            keyId = slot.keyId
          } else if (fallbackKey) {
            keyId = String(fallbackKey.id)
          }
          return {
            ...slot,
            keyId,
          }
        })
      )
    } catch {
      setKeys([])
      toast.error(t('Failed to load API keys'))
    } finally {
      setStatus((current) => (current === 'loading-keys' ? 'idle' : current))
    }
  }, [t])

  useEffect(() => {
    void loadKeys()
    return () => runAbortRef.current?.abort()
  }, [loadKeys])

  useEffect(() => {
    if (status !== 'running') return
    const timer = window.setInterval(() => {
      setElapsed(Date.now() - runStartedAtRef.current)
    }, 250)
    return () => window.clearInterval(timer)
  }, [status])

  const recordResult = useCallback((result: LoadTestRequestResult) => {
    if (result.requestId) requestIdsRef.current.push(result.requestId)
    setStats((current) => {
      const next = {
        ...current,
        completed: current.completed + 1,
        failures: current.failures + (result.success ? 0 : 1),
        successes: current.successes + (result.success ? 1 : 0),
        latencies: [...current.latencies, result.latency],
        statusCodes: { ...current.statusCodes },
        errorCodes: { ...current.errorCodes },
        keyCounts: { ...current.keyCounts },
        inputTokens: current.inputTokens + (result.usage?.inputTokens ?? 0),
        outputTokens: current.outputTokens + (result.usage?.outputTokens ?? 0),
        cacheReadTokens:
          current.cacheReadTokens + (result.usage?.cacheReadTokens ?? 0),
        cacheWriteTokens:
          current.cacheWriteTokens + (result.usage?.cacheWriteTokens ?? 0),
      }
      incrementCounter(next.statusCodes, String(result.status))
      if (result.errorCode) incrementCounter(next.errorCodes, result.errorCode)
      incrementCounter(next.keyCounts, result.keyName)
      return next
    })
  }, [])

  const run = useCallback(async () => {
    const activeSlots = loadTestSlots.flatMap((slot) => {
      const key = keys.find((item) => String(item.id) === slot.keyId)
      return key ? [{ ...slot, key }] : []
    })
    if (!serverAddress || activeSlots.length === 0) return
    const totalWeight = activeSlots.reduce(
      (total, slot) => total + Math.max(0, slot.weight),
      0
    )
    if (totalWeight <= 0) {
      toast.error(t('Load test limits are invalid'))
      return
    }
    if (
      !Number.isFinite(durationSeconds) ||
      !Number.isFinite(requestsPerSecond) ||
      durationSeconds < LOAD_TEST_MIN_DURATION_SECONDS ||
      durationSeconds > LOAD_TEST_MAX_DURATION_SECONDS ||
      requestsPerSecond < LOAD_TEST_MIN_RPS ||
      requestsPerSecond > LOAD_TEST_MAX_RPS
    ) {
      toast.error(t('Load test limits are invalid'))
      return
    }
    const controller = new AbortController()
    runAbortRef.current = controller
    runStartedAtRef.current = Date.now()
    const currentRunId = makeRunId()
    activeRunIdRef.current = currentRunId
    setRunId(currentRunId)
    setElapsed(0)
    setStats(EMPTY_STATS)
    setChannelStats([])
    requestIdsRef.current = []
    setStatus('running')

    const inFlight = new Set<Promise<void>>()
    const durationMs = durationSeconds * 1000
    const requestIntervalMs = 1000 / requestsPerSecond
    const requestLimit = Math.min(
      LOAD_TEST_MAX_REQUESTS,
      Math.ceil(durationSeconds * requestsPerSecond)
    )
    const deadline = Date.now() + durationMs
    let sentRequests = 0
    while (
      Date.now() < deadline &&
      sentRequests < requestLimit &&
      !controller.signal.aborted
    ) {
      if (inFlight.size >= LOAD_TEST_MAX_CONCURRENCY) {
        await Promise.race(inFlight)
      }

      let weightedSlot = activeSlots[0]
      let cursor = sentRequests % totalWeight
      for (const slot of activeSlots) {
        cursor -= Math.max(0, slot.weight)
        if (cursor < 0) {
          weightedSlot = slot
          break
        }
      }
      const request = sendLoadTestRequest(
        serverAddress,
        weightedSlot.key,
        selectedModel,
        currentRunId,
        promptCache,
        weightedSlot.strategy,
        controller.signal
      ).then(recordResult)
      inFlight.add(request)
      void request.then(() => inFlight.delete(request))
      sentRequests += 1
      await new Promise((resolve) =>
        window.setTimeout(resolve, requestIntervalMs)
      )
    }

    await Promise.all(inFlight)
    if (activeRunIdRef.current !== currentRunId) return

    try {
      setChannelStats(await getLoadTestChannelStats(requestIdsRef.current))
    } catch {
      setChannelStats([])
      toast.error(t('Channel statistics unavailable'))
    }

    runAbortRef.current = null
    activeRunIdRef.current = ''
    setElapsed(Math.min(durationMs, Date.now() - runStartedAtRef.current))
    setStatus('complete')
  }, [
    durationSeconds,
    keys,
    promptCache,
    recordResult,
    requestsPerSecond,
    loadTestSlots,
    selectedModel,
    serverAddress,
    t,
  ])

  const stop = useCallback(() => {
    runAbortRef.current?.abort()
    runAbortRef.current = null
    activeRunIdRef.current = ''
    setStatus('complete')
  }, [])

  const durationMs = durationSeconds * 1000
  const progress = Math.min(100, (elapsed / durationMs) * 100)
  const successRate = stats.completed
    ? ((stats.successes / stats.completed) * 100).toFixed(1)
    : '0.0'
  const p50 = percentile(stats.latencies, 0.5)
  const p95 = percentile(stats.latencies, 0.95)
  const cacheAttemptTokens =
    stats.inputTokens + stats.cacheReadTokens + stats.cacheWriteTokens
  const cacheHitRate = cacheAttemptTokens
    ? ((stats.cacheReadTokens / cacheAttemptTokens) * 100).toFixed(1)
    : '0.0'
  const totalChargedQuota = channelStats.reduce(
    (total, item) => total + item.charged_quota,
    0
  )
  const totalChannelTokens = channelStats.reduce(
    (total, item) =>
      total +
      (item.input_tokens_total > 0
        ? item.input_tokens_total
        : item.input_tokens +
          item.cache_read_tokens +
          item.cache_write_tokens) +
      item.output_tokens,
    0
  )
  const poolsUsed = new Set(
    channelStats.map((channel) => channel.pool_name).filter(Boolean)
  )
  const clustersUsed = new Set(
    channelStats
      .map((channel) => channel.cluster_id)
      .filter((clusterId) => clusterId > 0)
  )
  const keyCostRows = channelStats.reduce((rows, channel) => {
    const existing = rows.get(channel.token_id) ?? {
      requests: 0,
      quota: 0,
      channels: new Set<number>(),
    }
    existing.requests += channel.requests
    existing.quota += channel.charged_quota
    existing.channels.add(channel.channel_id)
    rows.set(channel.token_id, existing)
    return rows
  }, new Map<number, { requests: number; quota: number; channels: Set<number> }>())
  const actualCost =
    channelStats.length > 0 && quotaPerUnit > 0
      ? totalChargedQuota / quotaPerUnit
      : 0
  const maxRequests = Math.min(
    LOAD_TEST_MAX_REQUESTS,
    Math.ceil(durationSeconds * requestsPerSecond)
  )
  const canRun =
    (status === 'idle' || status === 'complete') &&
    loadTestSlots.some((slot) => slot.keyId !== '')

  const statusLabel = useMemo(() => {
    if (status === 'loading-keys') return t('Loading')
    if (status === 'running') return t('Testing...')
    if (status === 'complete') return t('Completed')
    return t('Ready')
  }, [status, t])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Load Test Demo')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          disabled={status === 'loading-keys' || status === 'running'}
          onClick={() => void loadKeys()}
          size='sm'
          variant='outline'
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto w-full max-w-5xl space-y-4'>
          <Card>
            <CardHeader>
              <div className='flex items-start justify-between gap-3'>
                <div>
                  <CardTitle className='flex items-center gap-2'>
                    <Activity className='text-primary size-5' />
                    {t('Load Test Demo')}
                  </CardTitle>
                  <CardDescription className='mt-1'>
                    {t(
                      'Configure a bounded duration and request rate for a controlled test.'
                    )}
                  </CardDescription>
                </div>
                <Badge variant={status === 'running' ? 'default' : 'secondary'}>
                  {statusLabel}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='space-y-3'>
                <div className='flex items-center justify-between gap-3'>
                  <div>
                    <Label>{t('API Key')}</Label>
                    <p className='text-muted-foreground text-xs'>
                      {t('Weighted by request count')}
                    </p>
                  </div>
                  <Badge variant='outline'>{t('Unified package')}</Badge>
                </div>
                {loadTestSlots.map((slot, index) => (
                  <div
                    className='grid gap-3 rounded-lg border p-3 sm:grid-cols-[minmax(0,1fr)_120px_minmax(0,1fr)]'
                    key={slot.id}
                  >
                    <div className='space-y-1.5'>
                      <Label>{`${t('API Key')} ${index + 1}`}</Label>
                      <Select
                        onValueChange={(value) =>
                          setLoadTestSlots((current) =>
                            current.map((item, itemIndex) =>
                              itemIndex === index
                                ? { ...item, keyId: value ?? '' }
                                : item
                            )
                          )
                        }
                        value={slot.keyId}
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue placeholder={t('Select API Key')} />
                        </SelectTrigger>
                        <SelectContent>
                          {keys
                            .filter(
                              (key) =>
                                String(key.id) === slot.keyId ||
                                !loadTestSlots.some(
                                  (item, itemIndex) =>
                                    itemIndex !== index &&
                                    item.keyId === String(key.id)
                                )
                            )
                            .map((key) => (
                              <SelectItem key={key.id} value={String(key.id)}>
                                {key.name} ·{' '}
                                {key.group || key.group_candidates[0] || '-'}
                              </SelectItem>
                            ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className='space-y-1.5'>
                      <Label>{t('Weight')}</Label>
                      <Input
                        min={0}
                        max={100}
                        step={1}
                        type='number'
                        value={slot.weight}
                        onChange={(event) => {
                          const value = Number(event.target.value)
                          setLoadTestSlots((current) =>
                            current.map((item, itemIndex) =>
                              itemIndex === index
                                ? {
                                    ...item,
                                    weight: Number.isFinite(value)
                                      ? Math.min(100, Math.max(0, value))
                                      : 0,
                                  }
                                : item
                            )
                          )
                        }}
                      />
                    </div>
                    <div className='space-y-1.5'>
                      <Label>{t('Routing strategy')}</Label>
                      <Select
                        onValueChange={(value) =>
                          setLoadTestSlots((current) =>
                            current.map((item, itemIndex) =>
                              itemIndex === index
                                ? {
                                    ...item,
                                    strategy: value as LoadTestRoutingStrategy,
                                  }
                                : item
                            )
                          )
                        }
                        value={slot.strategy}
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {LOAD_TEST_ROUTING_STRATEGIES.map((strategy) => (
                            <SelectItem
                              key={strategy.value}
                              value={strategy.value}
                            >
                              {t(strategy.label)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                ))}
              </div>
              <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
                <div className='space-y-1.5'>
                  <Label>{t('Test model')}</Label>
                  <Select
                    onValueChange={(value) => value && setSelectedModel(value)}
                    value={selectedModel}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {LOAD_TEST_MODELS.map((model) => (
                        <SelectItem key={model} value={model}>
                          {model}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='load-test-duration'>{t('Duration')}</Label>
                  <Input
                    id='load-test-duration'
                    type='number'
                    min={LOAD_TEST_MIN_DURATION_SECONDS}
                    max={LOAD_TEST_MAX_DURATION_SECONDS}
                    step={1}
                    value={durationSeconds}
                    onChange={(event) => {
                      const value = Number(event.target.value)
                      setDurationSeconds(
                        Number.isFinite(value)
                          ? Math.min(
                              LOAD_TEST_MAX_DURATION_SECONDS,
                              Math.max(LOAD_TEST_MIN_DURATION_SECONDS, value)
                            )
                          : LOAD_TEST_MIN_DURATION_SECONDS
                      )
                    }}
                  />
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: {{min}}-{{max}} seconds', {
                      min: LOAD_TEST_MIN_DURATION_SECONDS,
                      max: LOAD_TEST_MAX_DURATION_SECONDS,
                    })}
                  </p>
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='load-test-rps'>
                    {t('Requests per second')}
                  </Label>
                  <Input
                    id='load-test-rps'
                    type='number'
                    min={LOAD_TEST_MIN_RPS}
                    max={LOAD_TEST_MAX_RPS}
                    step={1}
                    value={requestsPerSecond}
                    onChange={(event) => {
                      const value = Number(event.target.value)
                      setRequestsPerSecond(
                        Number.isFinite(value)
                          ? Math.min(
                              LOAD_TEST_MAX_RPS,
                              Math.max(LOAD_TEST_MIN_RPS, value)
                            )
                          : LOAD_TEST_MIN_RPS
                      )
                    }}
                  />
                  <p className='text-muted-foreground text-xs'>
                    {t('Allowed range: {{min}}-{{max}} RPS', {
                      min: LOAD_TEST_MIN_RPS,
                      max: LOAD_TEST_MAX_RPS,
                    })}
                  </p>
                </div>
              </div>

              {keys.length === 0 ? (
                <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-start gap-2 rounded-lg border p-3 text-sm'>
                  <AlertTriangle className='mt-0.5 size-4 shrink-0' />
                  <span>{t('No keys found')}</span>
                </div>
              ) : (
                <div className='text-muted-foreground text-sm'>
                  {t('API Keys')}: {keys.map((key) => key.name).join(', ')}
                </div>
              )}

              <div className='flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3'>
                <div>
                  <Label htmlFor='load-test-cache'>{t('Prompt Cache')}</Label>
                  <p className='text-muted-foreground text-xs'>
                    {t('Prompt Cache')}
                  </p>
                </div>
                <Switch
                  id='load-test-cache'
                  checked={promptCache}
                  onCheckedChange={setPromptCache}
                />
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('Maximum requests for this run')}: {maxRequests} ·{' '}
                {t('Maximum concurrency')}: {LOAD_TEST_MAX_CONCURRENCY}
              </div>

              <div className='flex flex-wrap items-center gap-2'>
                <Button disabled={!canRun} onClick={() => void run()}>
                  <Play className='size-4' />
                  {t('Start test')}
                </Button>
                <Button
                  disabled={status !== 'running'}
                  onClick={stop}
                  variant='outline'
                >
                  <Square className='size-4' />
                  {t('Stop testing')}
                </Button>
              </div>

              {(status === 'running' || status === 'complete') && (
                <div className='space-y-2'>
                  <div className='flex justify-between text-xs'>
                    <span>{formatDuration(elapsed)}</span>
                    <span>{durationSeconds}s</span>
                  </div>
                  <Progress value={progress} />
                  <div className='text-muted-foreground text-xs'>
                    {t('Run ID')}: <code>{runId}</code>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-6'>
            <Metric label={t('Requests')} value={String(stats.completed)} />
            <Metric label={t('Failed')} value={String(stats.failures)} />
            <Metric label={t('Success rate')} value={`${successRate}%`} />
            <Metric label={t('Cache hit rate')} value={`${cacheHitRate}%`} />
            <Metric label={t('P50 latency')} value={p50 ? `${p50}ms` : '-'} />
            <Metric label={t('P95 latency')} value={p95 ? `${p95}ms` : '-'} />
          </div>

          <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
            <Metric
              label={t('Input tokens')}
              value={stats.inputTokens.toLocaleString()}
            />
            <Metric
              label={t('Output tokens')}
              value={stats.outputTokens.toLocaleString()}
            />
            <Metric
              label={t('Cache tokens')}
              value={`${stats.cacheReadTokens.toLocaleString()} / ${stats.cacheWriteTokens.toLocaleString()}`}
            />
            <Metric
              label={t('Actual cost')}
              value={`$${actualCost.toFixed(6)}`}
            />
          </div>
          <Card>
            <CardHeader>
              <CardTitle>{t('Channel token usage and cost')}</CardTitle>
              <CardDescription>{t('Actual cost')}</CardDescription>
            </CardHeader>
            <CardContent>
              {channelStats.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No channel usage recorded yet')}
                </p>
              ) : (
                <div className='overflow-x-auto rounded-md border'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Channel')}</TableHead>
                        <TableHead>{t('Pool')}</TableHead>
                        <TableHead>{t('Requests')}</TableHead>
                        <TableHead>{t('Prompt tokens')}</TableHead>
                        <TableHead>{t('Input total')}</TableHead>
                        <TableHead>{t('Output tokens')}</TableHead>
                        <TableHead>{t('Cache tokens')}</TableHead>
                        <TableHead>{t('Total tokens')}</TableHead>
                        <TableHead>{t('Token share')}</TableHead>
                        <TableHead>{t('Cost factor')}</TableHead>
                        <TableHead>{t('Actual cost')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {channelStats.map((channel) => {
                        const inputTotalTokens =
                          channel.input_tokens_total > 0
                            ? channel.input_tokens_total
                            : channel.input_tokens +
                              channel.cache_read_tokens +
                              channel.cache_write_tokens
                        const channelTokens =
                          inputTotalTokens + channel.output_tokens
                        const share = totalChannelTokens
                          ? (channelTokens / totalChannelTokens) * 100
                          : 0
                        return (
                          <TableRow key={channel.channel_id}>
                            <TableCell>
                              #{channel.channel_id} {channel.channel_name}
                              {channel.cluster_id > 0
                                ? ` · C${channel.cluster_id}`
                                : ''}
                            </TableCell>
                            <TableCell>{channel.pool_name || '-'}</TableCell>
                            <TableCell className='tabular-nums'>
                              {channel.requests}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {channel.input_tokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {inputTotalTokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {channel.output_tokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {channel.cache_read_tokens.toLocaleString()} /{' '}
                              {channel.cache_write_tokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {channelTokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {share.toFixed(2)}%
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {channel.cost_factor.toFixed(2)}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              $
                              {(quotaPerUnit > 0
                                ? channel.charged_quota / quotaPerUnit
                                : 0
                              ).toFixed(6)}
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
          {channelStats.length > 0 && (
            <div className='grid gap-4 sm:grid-cols-3'>
              <Metric
                label={t('Channel')}
                value={String(channelStats.length)}
              />
              <Metric label={t('Pool')} value={String(poolsUsed.size)} />
              <Metric label={t('Clusters')} value={String(clustersUsed.size)} />
            </div>
          )}
          <p className='text-muted-foreground text-xs'>{t('Actual cost')}</p>

          <div className='grid gap-4 lg:grid-cols-3'>
            <Card size='sm'>
              <CardHeader>
                <CardTitle>{t('HTTP status')}</CardTitle>
              </CardHeader>
              <CardContent>
                <CounterList
                  counters={stats.statusCodes}
                  emptyLabel={t('No requests sent yet')}
                />
              </CardContent>
            </Card>
            <Card size='sm'>
              <CardHeader>
                <CardTitle>{t('Error codes')}</CardTitle>
              </CardHeader>
              <CardContent>
                <CounterList
                  counters={stats.errorCodes}
                  emptyLabel={t('No errors')}
                />
              </CardContent>
            </Card>
            <Card size='sm'>
              <CardHeader>
                <CardTitle>{t('Key usage')}</CardTitle>
              </CardHeader>
              <CardContent>
                {keyCostRows.size === 0 ? (
                  <CounterList
                    counters={stats.keyCounts}
                    emptyLabel={t('No requests sent yet')}
                  />
                ) : (
                  <div className='space-y-2 text-sm'>
                    {[...keyCostRows.entries()].map(([tokenId, row]) => {
                      const key = keys.find((item) => item.id === tokenId)
                      return (
                        <div
                          className='flex items-center justify-between gap-3'
                          key={tokenId}
                        >
                          <span className='truncate'>
                            {key?.name ?? `#${tokenId}`}
                          </span>
                          <span className='shrink-0 tabular-nums'>
                            {row.requests} · $
                            {(quotaPerUnit > 0
                              ? row.quota / quotaPerUnit
                              : 0
                            ).toFixed(6)}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
