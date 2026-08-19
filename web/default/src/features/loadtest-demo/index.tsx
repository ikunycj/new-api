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
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useChatPresets } from '@/features/chat/hooks/use-chat-presets'

import {
  LOAD_TEST_DURATION_MS,
  LOAD_TEST_INTERVAL_MS,
  LOAD_TEST_MODEL,
  loadClaudeLoadTestKeys,
  loadClaudeLoadTestPricing,
  sendClaudeLoadTestRequest,
  type LoadTestKey,
  type LoadTestPricing,
  type LoadTestRequestResult,
} from './api'

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
  const [keys, setKeys] = useState<LoadTestKey[]>([])
  const [selectedKeyId, setSelectedKeyId] = useState('')
  const [selectedModel, setSelectedModel] = useState(LOAD_TEST_MODEL)
  const [status, setStatus] = useState<RunStatus>('idle')
  const [stats, setStats] = useState<RunStats>(EMPTY_STATS)
  const [runId, setRunId] = useState('')
  const [elapsed, setElapsed] = useState(0)
  const [pricing, setPricing] = useState<LoadTestPricing | null>(null)
  const runAbortRef = useRef<AbortController | null>(null)
  const activeRunIdRef = useRef('')
  const runStartedAtRef = useRef(0)

  const loadKeys = useCallback(async () => {
    setStatus('loading-keys')
    try {
      const loadedKeys = await loadClaudeLoadTestKeys()
      setKeys(loadedKeys)
      setSelectedKeyId((current) =>
        loadedKeys.some((key) => String(key.id) === current)
          ? current
          : String(loadedKeys[0]?.id ?? '')
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
    const selectedKey = keys.find((key) => String(key.id) === selectedKeyId)
    if (!serverAddress || !selectedKey) return
    const controller = new AbortController()
    runAbortRef.current = controller
    runStartedAtRef.current = Date.now()
    const currentRunId = makeRunId()
    activeRunIdRef.current = currentRunId
    setRunId(currentRunId)
    setElapsed(0)
    setStats(EMPTY_STATS)
    setStatus('running')

    try {
      setPricing(
        await loadClaudeLoadTestPricing(
          selectedModel,
          selectedKey.group?.trim() || selectedKey.group_candidates[0] || ''
        )
      )
    } catch {
      setPricing(null)
    }

    const inFlight = new Set<Promise<void>>()
    const deadline = Date.now() + LOAD_TEST_DURATION_MS
    while (Date.now() < deadline && !controller.signal.aborted) {
      if (inFlight.size >= 10) await Promise.race(inFlight)

      const request = sendClaudeLoadTestRequest(
        serverAddress,
        selectedKey,
        selectedModel,
        currentRunId,
        controller.signal
      ).then(recordResult)
      inFlight.add(request)
      void request.then(() => inFlight.delete(request))
      await new Promise((resolve) =>
        window.setTimeout(resolve, LOAD_TEST_INTERVAL_MS)
      )
    }

    await Promise.all(inFlight)
    if (activeRunIdRef.current !== currentRunId) return

    runAbortRef.current = null
    activeRunIdRef.current = ''
    setElapsed(
      Math.min(LOAD_TEST_DURATION_MS, Date.now() - runStartedAtRef.current)
    )
    setStatus('complete')
  }, [keys, recordResult, selectedKeyId, selectedModel, serverAddress])

  const stop = useCallback(() => {
    runAbortRef.current?.abort()
    runAbortRef.current = null
    activeRunIdRef.current = ''
    setStatus('complete')
  }, [])

  const progress = Math.min(100, (elapsed / LOAD_TEST_DURATION_MS) * 100)
  const successRate = stats.completed
    ? ((stats.successes / stats.completed) * 100).toFixed(1)
    : '0.0'
  const p50 = percentile(stats.latencies, 0.5)
  const p95 = percentile(stats.latencies, 0.95)
  const cacheableInputTokens = stats.inputTokens + stats.cacheReadTokens
  const cacheHitRate = cacheableInputTokens
    ? ((stats.cacheReadTokens / cacheableInputTokens) * 100).toFixed(1)
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
  const estimatedCost =
    (stats.inputTokens / 1_000_000) * inputPricePerMillion +
    (stats.outputTokens / 1_000_000) * outputPricePerMillion +
    (stats.cacheReadTokens / 1_000_000) * cacheReadPricePerMillion +
    (stats.cacheWriteTokens / 1_000_000) * cacheWritePricePerMillion
  const canRun =
    (status === 'idle' || status === 'complete') && selectedKeyId !== ''

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
                    {t('Claude Load Test Demo')}
                  </CardTitle>
                  <CardDescription className='mt-1'>
                    {t(
                      'The demo runs for 60 seconds with a gentle request rate.'
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
                    onValueChange={(value) => value && setSelectedKeyId(value)}
                    value={selectedKeyId}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue placeholder={t('Select API Key')} />
                    </SelectTrigger>
                    <SelectContent>
                      {keys.map((key) => (
                        <SelectItem key={key.id} value={String(key.id)}>
                          {key.name}
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
                      <SelectItem value='claude-opus-4-8'>
                        claude-opus-4-8
                      </SelectItem>
                      <SelectItem value='claude-sonnet-4-6'>
                        claude-sonnet-4-6
                      </SelectItem>
                      <SelectItem value='claude-3-7-sonnet'>
                        claude-3-7-sonnet
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <Metric label={t('Duration')} value='60s' />
                <Metric label={t('Requests per second')} value='2' />
              </div>

              {keys.length === 0 ? (
                <div className='border-destructive/30 bg-destructive/5 text-destructive flex items-start gap-2 rounded-lg border p-3 text-sm'>
                  <AlertTriangle className='mt-0.5 size-4 shrink-0' />
                  <span>{t('No Claude load-test keys found')}</span>
                </div>
              ) : (
                <div className='text-muted-foreground text-sm'>
                  {t('Claude load-test keys')}:{' '}
                  {keys.map((key) => key.name).join(', ')}
                </div>
              )}

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
                    <span>60s</span>
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
