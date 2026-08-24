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
import {
  Activity,
  AlertTriangle,
  Play,
  RefreshCw,
  Square,
  Trash2,
} from 'lucide-react'
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
import { Textarea } from '@/components/ui/textarea'
import { useChatPresets } from '@/features/chat/hooks/use-chat-presets'
import { fetchApiKeyModels } from '@/features/keys/api'
import { QUOTA_TYPE_VALUES } from '@/features/pricing/constants'
import { useAuthStore } from '@/stores/auth-store'

import {
  LOAD_TEST_DEFAULT_DURATION_SECONDS,
  LOAD_TEST_DEFAULT_PROMPT,
  LOAD_TEST_DEFAULT_RPS,
  LOAD_TEST_MAX_CONCURRENCY,
  LOAD_TEST_MAX_DURATION_SECONDS,
  LOAD_TEST_MAX_PROMPT_CHARS,
  LOAD_TEST_MAX_REQUESTS,
  LOAD_TEST_MAX_RPS,
  LOAD_TEST_MIN_DURATION_SECONDS,
  LOAD_TEST_MIN_RPS,
  buildLoadTestRequestBody,
  getLoadTestApiBaseUrl,
  getLoadTestChannelStats,
  getLoadTestEndpointPath,
  getLoadTestModels,
  loadLoadTestKeys,
  loadLoadTestPricing,
  sendLoadTestRequest,
  type LoadTestChannelStats,
  type LoadTestKey,
  type LoadTestModel,
  type LoadTestPricing,
  type LoadTestRequestResult,
} from './api'
import {
  clearPersistedLoadTestRuns,
  loadPersistedLoadTestRuns,
  savePersistedLoadTestRun,
  type RunStats,
} from './storage'

type RunStatus = 'idle' | 'loading-keys' | 'running' | 'complete'

type RunSnapshot = {
  model: string
  prompt: string
  keyName: string
  packageName: string
  durationSeconds: number
  requestsPerSecond: number
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

function calculateOfficialChannelCost(
  channel: LoadTestChannelStats,
  pricing: LoadTestPricing
) {
  if (pricing.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return channel.requests * (pricing.model.model_price ?? 0)
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
  return officialCost
}

function calculateUserCharge(
  channel: LoadTestChannelStats,
  pricing: LoadTestPricing
) {
  return calculateOfficialChannelCost(channel, pricing) * pricing.groupRatio
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
  const [persistedRuns, setPersistedRuns] = useState(() =>
    loadPersistedLoadTestRuns(userId)
  )
  const [historicalPricing, setHistoricalPricing] = useState<
    Record<string, LoadTestPricing | null>
  >({})
  const [persistedRun] = useState(() => persistedRuns[0] ?? null)
  const [keys, setKeys] = useState<LoadTestKey[]>([])
  // Use the masked key value as the UI identity. IDs are database details and
  // can be misleading when accounts are switched in the same browser.
  const [selectedKeyValue, setSelectedKeyValue] = useState('')
  const [models, setModels] = useState<LoadTestModel[]>([])
  const [modelsLoading, setModelsLoading] = useState(false)
  const [selectedModel, setSelectedModel] = useState(persistedRun?.model ?? '')
  const [durationSeconds, setDurationSeconds] = useState(
    String(persistedRun?.durationSeconds || LOAD_TEST_DEFAULT_DURATION_SECONDS)
  )
  const [requestsPerSecond, setRequestsPerSecond] = useState(
    String(persistedRun?.requestsPerSecond || LOAD_TEST_DEFAULT_RPS)
  )
  const [prompt, setPrompt] = useState(
    persistedRun?.prompt ?? LOAD_TEST_DEFAULT_PROMPT
  )
  const [promptCache, setPromptCache] = useState(true)
  const [status, setStatus] = useState<RunStatus>('idle')
  const [stats, setStats] = useState<RunStats>(
    persistedRun?.stats ?? EMPTY_STATS
  )
  const [runId, setRunId] = useState(persistedRun?.runId ?? '')
  const [runKeyName, setRunKeyName] = useState(persistedRun?.keyName ?? '')
  const [runPackageName, setRunPackageName] = useState(
    persistedRun?.packageName ?? ''
  )
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
  const runSnapshotRef = useRef<RunSnapshot | null>(null)
  const persistedRunIdRef = useRef('')

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
    let active = true
    void Promise.all(
      persistedRuns.map(async (run) => {
        try {
          return [
            run.runId,
            await loadLoadTestPricing(run.model, run.packageName),
          ] as const
        } catch {
          return [run.runId, null] as const
        }
      })
    ).then((entries) => {
      if (active) setHistoricalPricing(Object.fromEntries(entries))
    })
    return () => {
      active = false
    }
  }, [persistedRuns])

  useEffect(() => {
    const selectedKey = keys.find((key) => key.key === selectedKeyValue)
    if (!serverAddress || !selectedKey) {
      setModels([])
      setSelectedModel('')
      setModelsLoading(false)
      return
    }

    let active = true
    setModels([])
    setSelectedModel('')
    setModelsLoading(true)

    void fetchApiKeyModels(
      getLoadTestApiBaseUrl(serverAddress),
      selectedKey.secret
    )
      .then((result) => {
        if (!active) return
        if (!result.success) {
          setModels([])
          return
        }

        const availableModels = getLoadTestModels(result.models)
        setModels(availableModels)
        setSelectedModel((current) =>
          availableModels.some((model) => model.id === current)
            ? current
            : (availableModels[0]?.id ?? '')
        )
      })
      .catch(() => {
        if (active) setModels([])
      })
      .finally(() => {
        if (active) setModelsLoading(false)
      })

    return () => {
      active = false
    }
  }, [keys, selectedKeyValue, serverAddress])

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
    const selectedModelOption = models.find(
      (model) => model.id === selectedModel
    )
    if (!serverAddress || !selectedKey || !selectedModelOption) return
    const durationValue =
      durationSeconds.trim() === '' ? Number.NaN : Number(durationSeconds)
    const rpsValue =
      requestsPerSecond.trim() === '' ? Number.NaN : Number(requestsPerSecond)
    if (!prompt.trim()) {
      toast.error(`${t('Prompt')}: ${t('Required')}`)
      return
    }
    if (
      !Number.isFinite(durationValue) ||
      durationValue < LOAD_TEST_MIN_DURATION_SECONDS ||
      durationValue > LOAD_TEST_MAX_DURATION_SECONDS
    ) {
      toast.error(
        t('Duration must be between {{min}} and {{max}} seconds.', {
          min: LOAD_TEST_MIN_DURATION_SECONDS,
          max: LOAD_TEST_MAX_DURATION_SECONDS,
        })
      )
      return
    }
    if (
      !Number.isFinite(rpsValue) ||
      rpsValue < LOAD_TEST_MIN_RPS ||
      rpsValue > LOAD_TEST_MAX_RPS
    ) {
      toast.error(
        t('Requests per second must be between {{min}} and {{max}}.', {
          min: LOAD_TEST_MIN_RPS,
          max: LOAD_TEST_MAX_RPS,
        })
      )
      return
    }
    const controller = new AbortController()
    runAbortRef.current = controller
    runStartedAtRef.current = Date.now()
    const currentRunId = makeRunId()
    activeRunIdRef.current = currentRunId
    runSnapshotRef.current = {
      model: selectedModel,
      prompt,
      keyName: selectedKey.name,
      packageName:
        selectedKey.group?.trim() || selectedKey.group_candidates[0] || '',
      durationSeconds: durationValue,
      requestsPerSecond: rpsValue,
    }
    setRunId(currentRunId)
    setRunKeyName(selectedKey.name)
    setRunPackageName(
      selectedKey.group?.trim() || selectedKey.group_candidates[0] || ''
    )
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
    const durationMs = durationValue * 1000
    const requestIntervalMs = 1000 / rpsValue
    const requestLimit = Math.min(
      LOAD_TEST_MAX_REQUESTS,
      Math.ceil(durationValue * rpsValue)
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
        prompt,
        promptCache,
        controller.signal,
        selectedModelOption.provider,
        selectedModelOption.endpoint
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
    models,
    prompt,
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

  const durationValue =
    durationSeconds.trim() === '' ? Number.NaN : Number(durationSeconds)
  const rpsValue =
    requestsPerSecond.trim() === '' ? Number.NaN : Number(requestsPerSecond)
  const durationMs = Number.isFinite(durationValue) ? durationValue * 1000 : 0
  const progress =
    durationMs > 0 ? Math.min(100, (elapsed / durationMs) * 100) : 0
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
  const inputPricePerMillion = pricing ? pricing.model.model_ratio * 2 : 0
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
  let userCharge = 0
  if (pricing) {
    if (channelStats.length > 0) {
      userCharge = channelStats.reduce(
        (total, channel) => total + calculateUserCharge(channel, pricing),
        0
      )
    } else if (pricing.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
      const officialCost = stats.successes * (pricing.model.model_price ?? 0)
      userCharge = officialCost * pricing.groupRatio
    } else {
      const officialCost =
        (stats.inputTokens / 1_000_000) * inputPricePerMillion +
        (stats.outputTokens / 1_000_000) * outputPricePerMillion +
        (stats.cacheReadTokens / 1_000_000) * cacheReadPricePerMillion +
        (stats.cacheWriteTokens / 1_000_000) * cacheWritePricePerMillion
      userCharge = officialCost * pricing.groupRatio
    }
  }

  const totalTokens =
    channelStats.length > 0
      ? totalChannelTokens
      : stats.inputTokens +
        stats.outputTokens +
        stats.cacheReadTokens +
        stats.cacheWriteTokens
  const averageTokenPrice =
    pricing && totalTokens > 0 ? userCharge / totalTokens : null

  useEffect(() => {
    if (
      !runId ||
      status !== 'complete' ||
      persistedRunIdRef.current === runId
    ) {
      return
    }
    const runSnapshot = runSnapshotRef.current
    if (!runSnapshot) return
    savePersistedLoadTestRun(userId, {
      model: runSnapshot.model,
      prompt: runSnapshot.prompt,
      runId,
      keyName: runSnapshot.keyName,
      packageName: runSnapshot.packageName,
      durationSeconds: runSnapshot.durationSeconds,
      requestsPerSecond: runSnapshot.requestsPerSecond,
      userCharge,
      stats,
      channelStats,
      requestIds: requestIdsRef.current,
    })
    persistedRunIdRef.current = runId
    setPersistedRuns(loadPersistedLoadTestRuns(userId))
  }, [channelStats, runId, stats, status, userId, userCharge])

  const maxRequests =
    Number.isFinite(durationValue) && Number.isFinite(rpsValue)
      ? Math.min(LOAD_TEST_MAX_REQUESTS, Math.ceil(durationValue * rpsValue))
      : 0
  const canRun =
    (status === 'idle' || status === 'complete') &&
    selectedKeyValue !== '' &&
    selectedModel !== '' &&
    prompt.trim() !== '' &&
    !modelsLoading
  const selectedKeyMetadata = keys.find((key) => key.key === selectedKeyValue)
  const selectedModelMetadata = models.find(
    (model) => model.id === selectedModel
  )
  const requestPreview = useMemo(() => {
    if (!selectedModelMetadata) return '{}'
    return JSON.stringify(
      buildLoadTestRequestBody(
        selectedModel,
        prompt,
        promptCache,
        selectedModelMetadata.endpoint
      ),
      null,
      2
    )
  }, [prompt, promptCache, selectedModel, selectedModelMetadata])

  const getHistoricalUserCharge = (run: (typeof persistedRuns)[number]) => {
    const runPricing = historicalPricing[run.runId]
    if (!runPricing || run.channelStats.length === 0) {
      return run.userCharge
    }
    return run.channelStats.reduce(
      (total, channel) => total + calculateUserCharge(channel, runPricing),
      0
    )
  }

  const clearHistory = useCallback(() => {
    clearPersistedLoadTestRuns(userId)
    setPersistedRuns([])
    toast.success(t('Load test history cleared'))
  }, [t, userId])

  const statusLabel = useMemo(() => {
    if (status === 'loading-keys' || modelsLoading) return t('Loading')
    if (status === 'running') return t('Testing...')
    if (status === 'complete') return t('Completed')
    return t('Ready')
  }, [modelsLoading, status, t])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Load Test Demo')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          disabled={
            status === 'loading-keys' || status === 'running' || modelsLoading
          }
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
                    disabled={modelsLoading || models.length === 0}
                    onValueChange={(value) => value && setSelectedModel(value)}
                    value={selectedModel}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue placeholder={t('No available models')} />
                    </SelectTrigger>
                    <SelectContent>
                      {models.map((model) => (
                        <SelectItem key={model.id} value={model.id}>
                          {model.id}
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
                    step={1}
                    value={durationSeconds}
                    onChange={(event) => setDurationSeconds(event.target.value)}
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
                    step={1}
                    value={requestsPerSecond}
                    onChange={(event) =>
                      setRequestsPerSecond(event.target.value)
                    }
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
                <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-sm'>
                  <span>
                    {t('API Key')}:{' '}
                    {selectedKeyMetadata?.name || t('Select API Key')}
                  </span>
                  <span>
                    {t('Package')}:{' '}
                    {selectedKeyMetadata?.group?.trim() ||
                      selectedKeyMetadata?.group_candidates[0] ||
                      '-'}
                  </span>
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
              <div className='grid gap-4 lg:grid-cols-2'>
                <div className='space-y-1.5'>
                  <div className='flex items-center justify-between gap-3'>
                    <Label htmlFor='load-test-prompt'>{t('Prompt')}</Label>
                    <span className='text-muted-foreground text-xs tabular-nums'>
                      {prompt.length.toLocaleString()} /{' '}
                      {LOAD_TEST_MAX_PROMPT_CHARS.toLocaleString()}
                    </span>
                  </div>
                  <Textarea
                    className='min-h-52 resize-y font-mono text-sm'
                    disabled={status === 'running'}
                    id='load-test-prompt'
                    maxLength={LOAD_TEST_MAX_PROMPT_CHARS}
                    onChange={(event) => setPrompt(event.target.value)}
                    value={prompt}
                  />
                </div>
                <div className='min-w-0 space-y-1.5'>
                  <div className='flex items-center justify-between gap-3'>
                    <Label>
                      {t('Request')} {t('Preview')}
                    </Label>
                    {selectedModelMetadata && (
                      <code className='text-muted-foreground text-xs'>
                        POST{' '}
                        {getLoadTestEndpointPath(
                          selectedModelMetadata.endpoint
                        )}
                      </code>
                    )}
                  </div>
                  <pre className='bg-muted/40 h-52 overflow-auto rounded-lg border p-3 text-xs leading-relaxed break-words whitespace-pre-wrap'>
                    {requestPreview}
                  </pre>
                </div>
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
                  <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
                    <span>
                      {t('API Key')}: {runKeyName || '-'}
                    </span>
                    <span>
                      {t('Package')}: {runPackageName || '-'}
                    </span>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className='flex flex-row items-center justify-between gap-3'>
              <div>
                <CardTitle>{t('Load test history')}</CardTitle>
                <CardDescription>
                  {t('Each completed test is saved with its own Run ID.')}
                </CardDescription>
              </div>
              <Button
                disabled={persistedRuns.length === 0 || status === 'running'}
                onClick={clearHistory}
                size='sm'
                variant='outline'
              >
                <Trash2 className='size-4' />
                {t('Clear history')}
              </Button>
            </CardHeader>
            <CardContent>
              {persistedRuns.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No previous load tests')}
                </p>
              ) : (
                <div className='overflow-x-auto rounded-md border'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Run ID')}</TableHead>
                        <TableHead>{t('Completed at')}</TableHead>
                        <TableHead>{t('Test model')}</TableHead>
                        <TableHead>{t('Prompt')}</TableHead>
                        <TableHead>{t('API Key')}</TableHead>
                        <TableHead>{t('Package')}</TableHead>
                        <TableHead>{t('Duration')}</TableHead>
                        <TableHead>{t('Requests')}</TableHead>
                        <TableHead>{t('Success rate')}</TableHead>
                        <TableHead>{t('Input tokens')}</TableHead>
                        <TableHead>{t('Output tokens')}</TableHead>
                        <TableHead>{t('Cache tokens')}</TableHead>
                        <TableHead>{t('Total tokens')}</TableHead>
                        <TableHead>{t('Average token price')}</TableHead>
                        <TableHead>{t('User charge')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {persistedRuns.map((run) => {
                        const historicalUserCharge =
                          getHistoricalUserCharge(run)
                        const historicalTotalTokens =
                          run.stats.inputTokens +
                          run.stats.outputTokens +
                          run.stats.cacheReadTokens +
                          run.stats.cacheWriteTokens
                        return (
                          <TableRow key={run.runId}>
                            <TableCell>
                              <code className='text-xs'>{run.runId}</code>
                            </TableCell>
                            <TableCell className='whitespace-nowrap'>
                              {new Date(run.completedAt).toLocaleString()}
                            </TableCell>
                            <TableCell>{run.model}</TableCell>
                            <TableCell
                              className='max-w-56 truncate'
                              title={run.prompt}
                            >
                              {run.prompt}
                            </TableCell>
                            <TableCell>{run.keyName || '-'}</TableCell>
                            <TableCell>{run.packageName || '-'}</TableCell>
                            <TableCell>{run.durationSeconds}s</TableCell>
                            <TableCell>{run.stats.completed}</TableCell>
                            <TableCell>
                              {run.stats.completed
                                ? `${((run.stats.successes / run.stats.completed) * 100).toFixed(1)}%`
                                : '0.0%'}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {run.stats.inputTokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {run.stats.outputTokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {run.stats.cacheReadTokens.toLocaleString()} /{' '}
                              {run.stats.cacheWriteTokens.toLocaleString()}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {(
                                run.stats.inputTokens +
                                run.stats.outputTokens +
                                run.stats.cacheReadTokens +
                                run.stats.cacheWriteTokens
                              ).toLocaleString()}
                            </TableCell>
                            <TableCell>
                              {historicalTotalTokens > 0
                                ? `$${(historicalUserCharge / historicalTotalTokens).toFixed(8)}`
                                : '-'}
                            </TableCell>
                            <TableCell>
                              ${historicalUserCharge.toFixed(6)}
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

          <div className='text-muted-foreground text-sm font-medium'>
            {t('Current run metrics')}
          </div>
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
              label={t('Average token price')}
              value={
                averageTokenPrice === null
                  ? t('Unavailable')
                  : `$${averageTokenPrice.toFixed(8)}`
              }
            />
            <Metric
              label={t('User charge')}
              value={pricing ? `$${userCharge.toFixed(6)}` : t('Unavailable')}
            />
          </div>
          <Card>
            <CardHeader>
              <CardTitle>{t('Channel token usage and cost')}</CardTitle>
              <CardDescription>
                {t('User charge = official model price × billing group ratio.')}
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
                        <TableHead>{t('Billing group ratio')}</TableHead>
                        <TableHead>{t('Channel cost factor')}</TableHead>
                        <TableHead>{t('User charge')}</TableHead>
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
                              {pricing ? pricing.groupRatio.toFixed(2) : '-'}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              {channel.cost_factor.toFixed(2)}
                            </TableCell>
                            <TableCell className='tabular-nums'>
                              $
                              {pricing
                                ? calculateUserCharge(channel, pricing).toFixed(
                                    6
                                  )
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
