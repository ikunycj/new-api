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
import { QUOTA_TYPE_VALUES } from '@/features/pricing/constants'
import { useAuthStore } from '@/stores/auth-store'

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
  loadLoadTestPricing,
  sendLoadTestRequest,
  type LoadTestChannelStats,
  type LoadTestKey,
  type LoadTestPricing,
  type LoadTestRequestResult,
} from './api'
import {
  loadPersistedLoadTestRun,
  savePersistedLoadTestRun,
  type RunStats,
} from './storage'

type RunStatus = 'idle' | 'loading-keys' | 'running' | 'complete'

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

function accumulateResult(
  current: RunStats,
  result: LoadTestRequestResult
): RunStats {
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
}

function formatDuration(milliseconds: number) {
  return `${Math.max(0, Math.floor(milliseconds / 1000))}s`
}

function calculateChannelCost(
  channel: LoadTestChannelStats,
  pricing: LoadTestPricing
) {
  if (pricing.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return (
      channel.requests *
      (pricing.model.model_price ?? 0) *
      pricing.groupRatio *
      channel.cost_factor
    )
  }

  const officialInputPricePerMillion = pricing.model.model_ratio * 2
  const officialOutputPricePerMillion =
    officialInputPricePerMillion * pricing.model.completion_ratio
  const officialCacheReadPricePerMillion =
    officialInputPricePerMillion * (pricing.model.cache_ratio ?? 1)
  const officialCacheWritePricePerMillion =
    officialInputPricePerMillion * (pricing.model.create_cache_ratio ?? 1)
  const baseInputTokens = Math.max(
    0,
    channel.input_tokens_total > 0
      ? channel.input_tokens_total -
          channel.cache_read_tokens -
          channel.cache_write_tokens
      : channel.input_tokens
  )
  const officialCost =
    (baseInputTokens / 1_000_000) * officialInputPricePerMillion +
    (channel.output_tokens / 1_000_000) * officialOutputPricePerMillion +
    (channel.cache_read_tokens / 1_000_000) * officialCacheReadPricePerMillion +
    (channel.cache_write_tokens / 1_000_000) * officialCacheWritePricePerMillion
  return officialCost * pricing.groupRatio * channel.cost_factor
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
  const userId = useAuthStore((state) => state.auth.user?.id)
  const [persistedRun] = useState(() => loadPersistedLoadTestRun(userId))
  const [keys, setKeys] = useState<LoadTestKey[]>([])
  // Use the masked key value as the UI identity. IDs are database details and
  // can be misleading when accounts are switched in the same browser.
  const [selectedKeyValue, setSelectedKeyValue] = useState('')
  const [selectedModel, setSelectedModel] = useState(
    persistedRun?.model ?? LOAD_TEST_MODEL
  )
  const [durationSeconds, setDurationSeconds] = useState(
    LOAD_TEST_DEFAULT_DURATION_SECONDS
  )
  const [requestsPerSecond, setRequestsPerSecond] = useState(
    LOAD_TEST_DEFAULT_RPS
  )
  const [promptCache, setPromptCache] = useState(true)
  const [status, setStatus] = useState<RunStatus>('idle')
  const [stats, setStats] = useState<RunStats>(
    persistedRun?.stats ?? EMPTY_STATS
  )
  const [runId, setRunId] = useState(persistedRun?.runId ?? '')
  const [elapsed, setElapsed] = useState(0)
  const [pricing, setPricing] = useState<LoadTestPricing | null>(null)
  const [channelStats, setChannelStats] = useState<LoadTestChannelStats[]>(
    persistedRun?.channelStats ?? []
  )
  const runAbortRef = useRef<AbortController | null>(null)
  const requestIdsRef = useRef<string[]>(persistedRun?.requestIds ?? [])
  const statsRef = useRef<RunStats>(persistedRun?.stats ?? EMPTY_STATS)
  const activeRunIdRef = useRef('')
  const runStartedAtRef = useRef(0)

  const loadKeys = useCallback(async () => {
    setStatus('loading-keys')
    try {
      const loadedKeys = await loadLoadTestKeys()
      setKeys(loadedKeys)
      setSelectedKeyValue((current) =>
        loadedKeys.some((key) => key.key === current)
          ? current
          : (loadedKeys[0]?.key ?? '')
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
    if (!runId || status === 'running') return
    savePersistedLoadTestRun(userId, {
      model: selectedModel,
      runId,
      stats,
      channelStats,
      requestIds: requestIdsRef.current,
    })
  }, [channelStats, runId, selectedModel, stats, status, userId])

  useEffect(() => {
    if (status !== 'running') return
    const timer = window.setInterval(() => {
      setElapsed(Date.now() - runStartedAtRef.current)
    }, 250)
    return () => window.clearInterval(timer)
  }, [status])

  const recordResult = useCallback((result: LoadTestRequestResult) => {
    if (result.requestId) requestIdsRef.current.push(result.requestId)
    statsRef.current = accumulateResult(statsRef.current, result)
    setStats(statsRef.current)
  }, [])

  const run = useCallback(async () => {
    const selectedKey = keys.find((key) => key.key === selectedKeyValue)
    if (!serverAddress || !selectedKey) return
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
    statsRef.current = EMPTY_STATS
    setChannelStats([])
    requestIdsRef.current = []
    setStatus('running')

    try {
      setPricing(
        await loadLoadTestPricing(
          selectedModel,
          selectedKey.group?.trim() || selectedKey.group_candidates[0] || ''
        )
      )
    } catch {
      setPricing(null)
    }

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

      const request = sendLoadTestRequest(
        serverAddress,
        selectedKey,
        selectedModel,
        currentRunId,
        promptCache,
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
      let settledStats: LoadTestChannelStats[] = []
      for (let attempt = 0; attempt < 30; attempt += 1) {
        settledStats = await getLoadTestChannelStats(requestIdsRef.current)
        const recordedRequests = settledStats.reduce(
          (total, channel) => total + channel.requests,
          0
        )
        if (recordedRequests >= statsRef.current.successes) break
        await new Promise((resolve) => window.setTimeout(resolve, 1000))
        if (activeRunIdRef.current !== currentRunId) return
      }
      setChannelStats(settledStats)
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
    selectedKeyValue,
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
  const inputPricePerMillion = pricing
    ? pricing.model.model_ratio * 2 * pricing.groupRatio
    : 0
  const outputPricePerMillion = pricing
    ? inputPricePerMillion * pricing.model.completion_ratio
    : 0
  const cacheReadPricePerMillion = pricing
    ? inputPricePerMillion * (pricing.model.cache_ratio ?? 1)
    : 0
  const cacheWritePricePerMillion = pricing
    ? inputPricePerMillion * (pricing.model.create_cache_ratio ?? 1)
    : 0
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
  const billingGroupsUsed = new Set(
    channelStats.map((channel) => channel.billing_group).filter(Boolean)
  )
  let estimatedCost = 0
  if (pricing) {
    if (channelStats.length > 0) {
      estimatedCost = channelStats.reduce(
        (total, channel) => total + calculateChannelCost(channel, pricing),
        0
      )
    } else if (pricing.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
      estimatedCost =
        stats.successes * (pricing.model.model_price ?? 0) * pricing.groupRatio
    } else {
      estimatedCost =
        (stats.inputTokens / 1_000_000) * inputPricePerMillion +
        (stats.outputTokens / 1_000_000) * outputPricePerMillion +
        (stats.cacheReadTokens / 1_000_000) * cacheReadPricePerMillion +
        (stats.cacheWriteTokens / 1_000_000) * cacheWritePricePerMillion
    }
  }
  const maxRequests = Math.min(
    LOAD_TEST_MAX_REQUESTS,
    Math.ceil(durationSeconds * requestsPerSecond)
  )
  const canRun =
    (status === 'idle' || status === 'complete') && selectedKeyValue !== ''

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
              <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
                <div className='space-y-1.5'>
                  <Label>{t('API Key')}</Label>
                  <Select
                    onValueChange={(value) =>
                      value && setSelectedKeyValue(value)
                    }
                    value={selectedKeyValue}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue placeholder={t('Select API Key')} />
                    </SelectTrigger>
                    <SelectContent>
                      {keys.map((key) => (
                        <SelectItem key={key.key} value={key.key}>
                          {key.name} ·{' '}
                          {key.group || key.group_candidates[0] || '-'}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
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
                  {t('API Key')}: {selectedKeyValue || t('Select API Key')}
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
              label={t('Estimated cost')}
              value={
                pricing ? `$${estimatedCost.toFixed(6)}` : t('Unavailable')
              }
            />
          </div>
          <Card>
            <CardHeader>
              <CardTitle>{t('Channel token usage and cost')}</CardTitle>
              <CardDescription>
                {t(
                  'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.'
                )}
              </CardDescription>
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
                        <TableHead>{t('Billing group')}</TableHead>
                        <TableHead>{t('Requests')}</TableHead>
                        <TableHead>{t('Prompt tokens')}</TableHead>
                        <TableHead>{t('Input total')}</TableHead>
                        <TableHead>{t('Output tokens')}</TableHead>
                        <TableHead>{t('Cache tokens')}</TableHead>
                        <TableHead>{t('Total tokens')}</TableHead>
                        <TableHead>{t('Token share')}</TableHead>
                        <TableHead>{t('Cost factor')}</TableHead>
                        <TableHead>{t('Channel cost')}</TableHead>
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
                              {channel.billing_group
                                ? ` · ${channel.billing_group}`
                                : ''}
                            </TableCell>
                            <TableCell>
                              {channel.billing_group || '-'}
                            </TableCell>
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
                              {pricing
                                ? calculateChannelCost(
                                    channel,
                                    pricing
                                  ).toFixed(6)
                                : '0.000000'}
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
              <Metric
                label={t('Billing groups')}
                value={String(billingGroupsUsed.size)}
              />
            </div>
          )}
          <p className='text-muted-foreground text-xs'>
            {pricing
              ? t(
                  'Estimated cost uses the selected model and group pricing snapshot. Token pricing includes input, output, cache read, and cache write usage; request pricing charges successful requests.'
                )
              : t('Pricing is unavailable until the test starts.')}
          </p>

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
                <CounterList
                  counters={stats.keyCounts}
                  emptyLabel={t('No requests sent yet')}
                />
              </CardContent>
            </Card>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
