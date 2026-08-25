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
  Copy,
  Link2,
  MonitorCog,
  Play,
  Server,
  Square,
  Trash2,
  Pencil,
} from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useIsAdmin } from '@/hooks/use-admin'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import {
  cancelLoadTestAgentRun,
  createLoadTestAgentPairing,
  createLoadTestAgentRun,
  deleteLoadTestAgent,
  getLoadTestAgentState,
  loadLoadTestPricing,
  updateManagedLoadTestAgentCapacity,
  type CreateLoadTestAgentRun,
  type LoadTestAgent,
  type LoadTestAgentRun,
  type LoadTestPricing,
} from './api'
import { calculateLoadTestUserCharge, getLoadTestTotalTokens } from './pricing'

type AgentPanelProps = {
  disabled: boolean
  mode: 'managed' | 'local'
  request: Omit<CreateLoadTestAgentRun, 'agent_id'> | null
}

type Pairing = {
  code: string
  expires_at: number
}

const ACTIVE_RUN_STATUSES = new Set([
  'queued',
  'dispatched',
  'running',
  'cancel_requested',
])

function formatMemory(bytes: number) {
  if (bytes <= 0) return '-'
  return `${(bytes / 1024 ** 3).toFixed(1)} GB`
}

function loadTestPricingKey(model: string, packageName: string) {
  return `${model}\u0000${packageName}`
}

export function AgentPanel(props: AgentPanelProps) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const [agents, setAgents] = useState<LoadTestAgent[]>([])
  const [onlineBefore, setOnlineBefore] = useState(0)
  const [selectedAgentId, setSelectedAgentId] = useState('')
  const [runs, setRuns] = useState<LoadTestAgentRun[]>([])
  const [pricingByKey, setPricingByKey] = useState<
    Record<string, LoadTestPricing | null>
  >({})
  const [pairing, setPairing] = useState<Pairing | null>(null)
  const [capacityAgent, setCapacityAgent] = useState<LoadTestAgent | null>(null)
  const [capacityRPS, setCapacityRPS] = useState('')
  const [capacityConcurrency, setCapacityConcurrency] = useState('')
  const [savingCapacity, setSavingCapacity] = useState(false)
  const [loading, setLoading] = useState(true)
  const [starting, setStarting] = useState(false)
  const refreshInFlightRef = useRef(false)
  const pricingByKeyRef = useRef<Record<string, LoadTestPricing | null>>({})

  const refresh = useCallback(
    async (showError = false) => {
      if (refreshInFlightRef.current) return
      refreshInFlightRef.current = true
      try {
        const state = await getLoadTestAgentState()
        const nextAgents =
          props.mode === 'managed' ? state.managed_agents : state.local_agents
        setAgents(nextAgents)
        setOnlineBefore(state.online_before)
        const nextRuns = state.runs.filter(
          (run) => run.agent_managed === (props.mode === 'managed')
        )
        setRuns(nextRuns)
        const missingPricing = new Map<
          string,
          { model: string; packageName: string }
        >()
        for (const run of nextRuns) {
          const key = loadTestPricingKey(run.model, run.package_name)
          if (!Object.hasOwn(pricingByKeyRef.current, key)) {
            missingPricing.set(key, {
              model: run.model,
              packageName: run.package_name,
            })
          }
        }
        if (missingPricing.size > 0) {
          for (const key of missingPricing.keys()) {
            pricingByKeyRef.current[key] = null
          }
          const pricingEntries = await Promise.all(
            [...missingPricing.entries()].map(
              async ([key, { model, packageName }]) => {
                try {
                  return [
                    key,
                    await loadLoadTestPricing(model, packageName),
                  ] as const
                } catch {
                  return [key, null] as const
                }
              }
            )
          )
          for (const [key, pricing] of pricingEntries) {
            pricingByKeyRef.current[key] = pricing
          }
          setPricingByKey({ ...pricingByKeyRef.current })
        }
        setSelectedAgentId((current) => {
          const currentAgent = nextAgents.find((agent) => agent.id === current)
          if (
            currentAgent &&
            currentAgent.last_seen_at >= state.online_before
          ) {
            return current
          }
          return (
            nextAgents.find(
              (agent) => agent.last_seen_at >= state.online_before
            )?.id ?? ''
          )
        })
      } catch (error) {
        if (showError) {
          toast.error(
            error instanceof Error ? error.message : t('Request failed')
          )
        }
      } finally {
        refreshInFlightRef.current = false
        setLoading(false)
      }
    },
    [props.mode, t]
  )

  const hasActiveRun = runs.some((run) => ACTIVE_RUN_STATUSES.has(run.status))

  useEffect(() => {
    void refresh(true)
    const timer = window.setInterval(
      () => {
        if (!document.hidden) void refresh()
      },
      hasActiveRun ? 2000 : 10000
    )
    return () => window.clearInterval(timer)
  }, [hasActiveRun, refresh])

  const createPairing = useCallback(async () => {
    try {
      setPairing(await createLoadTestAgentPairing(props.mode === 'managed'))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    }
  }, [props.mode, t])

  const copyPairCommand = useCallback(async () => {
    if (!pairing) return
    let serverURL = window.location.origin
    if (
      props.mode === 'managed' &&
      window.location.protocol === 'http:' &&
      !['127.0.0.1', 'localhost', '::1'].includes(window.location.hostname)
    ) {
      serverURL = 'http://127.0.0.1:3000'
    }
    const command = `alltoken-loadtest-agent pair --server ${serverURL} ${pairing.code}`
    await navigator.clipboard.writeText(command)
    toast.success(t('Copied'))
  }, [pairing, props.mode, t])

  const start = useCallback(async () => {
    if (!props.request || !selectedAgentId) return
    setStarting(true)
    try {
      await createLoadTestAgentRun({
        ...props.request,
        agent_id: selectedAgentId,
      })
      toast.success(t('Agent load test queued'))
      await refresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    } finally {
      setStarting(false)
    }
  }, [props.request, refresh, selectedAgentId, t])

  const stop = useCallback(
    async (runId: string) => {
      try {
        await cancelLoadTestAgentRun(runId)
        await refresh()
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Request failed')
        )
      }
    },
    [refresh, t]
  )

  const removeAgent = useCallback(
    async (agentId: string) => {
      try {
        await deleteLoadTestAgent(agentId, props.mode === 'managed')
        await refresh()
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Request failed')
        )
      }
    },
    [props.mode, refresh, t]
  )

  const editCapacity = useCallback((agent: LoadTestAgent) => {
    setCapacityAgent(agent)
    setCapacityRPS(String(agent.max_rps))
    setCapacityConcurrency(String(agent.max_concurrency))
  }, [])

  const saveCapacity = useCallback(async () => {
    if (!capacityAgent) return
    const maxRPS = Number(capacityRPS)
    const maxConcurrency = Number(capacityConcurrency)
    if (
      !Number.isInteger(maxRPS) ||
      maxRPS < 1 ||
      !Number.isInteger(maxConcurrency) ||
      maxConcurrency < 1
    ) {
      toast.error(t('Agent capacity must be a positive integer'))
      return
    }
    setSavingCapacity(true)
    try {
      await updateManagedLoadTestAgentCapacity(capacityAgent.id, {
        max_rps: maxRPS,
        max_concurrency: maxConcurrency,
      })
      setCapacityAgent(null)
      await refresh()
      toast.success(t('Agent capacity updated'))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    } finally {
      setSavingCapacity(false)
    }
  }, [capacityAgent, capacityConcurrency, capacityRPS, refresh, t])

  const onlineAgents = agents.filter(
    (agent) => agent.last_seen_at >= onlineBefore
  )
  const selectedAgent = onlineAgents.find(
    (agent) => agent.id === selectedAgentId
  )

  return (
    <>
      <Card>
        <CardHeader>
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div>
              <CardTitle className='flex items-center gap-2'>
                {props.mode === 'managed' ? (
                  <Server className='size-5' />
                ) : (
                  <MonitorCog className='size-5' />
                )}
                {props.mode === 'managed'
                  ? t('Server load test')
                  : t('Local load-test agent')}
              </CardTitle>
              <CardDescription className='mt-1'>
                {props.mode === 'managed'
                  ? t(
                      'Run the test on a shared load generator managed by the platform.'
                    )
                  : t(
                      'Run high-volume tests outside the browser and keep results linked to this account.'
                    )}
              </CardDescription>
            </div>
            {(props.mode === 'local' || isAdmin) && (
              <Button onClick={() => void createPairing()} variant='outline'>
                <Link2 className='size-4' />
                {props.mode === 'managed'
                  ? t('Add server agent')
                  : t('Pair agent')}
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex flex-wrap items-end gap-3'>
            <div className='min-w-64 flex-1 space-y-1.5'>
              <span className='text-sm font-medium'>{t('Load generator')}</span>
              <Select
                disabled={loading || onlineAgents.length === 0}
                onValueChange={(value) => value && setSelectedAgentId(value)}
                value={selectedAgentId}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('No online agents')}>
                    {selectedAgent
                      ? `${selectedAgent.name} · ${selectedAgent.max_rps} RPS · C${selectedAgent.max_concurrency}`
                      : null}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {onlineAgents.map((agent) => (
                    <SelectItem key={agent.id} value={agent.id}>
                      {agent.name} · {agent.cpu_cores} CPU ·{' '}
                      {formatMemory(agent.memory_bytes)} · {agent.max_rps} RPS ·
                      C{agent.max_concurrency}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button
              disabled={
                props.disabled || !props.request || !selectedAgentId || starting
              }
              onClick={() => void start()}
            >
              <Play className='size-4' />
              {t('Start with agent')}
            </Button>
          </div>

          {agents.length === 0 && !loading && (
            <p className='text-muted-foreground text-sm'>
              {props.mode === 'managed'
                ? t('No server load generator is available.')
                : t('No local agent is paired.')}
            </p>
          )}

          {agents.length > 0 && (
            <div className='flex flex-wrap gap-2'>
              {agents.map((agent) => {
                const online = agent.last_seen_at >= onlineBefore
                return (
                  <div
                    className='flex items-center gap-2 rounded-md border px-2 py-1.5 text-sm'
                    key={agent.id}
                  >
                    <span>{agent.name}</span>
                    <Badge variant={online ? 'default' : 'secondary'}>
                      {online ? t('Online') : t('Offline')}
                    </Badge>
                    <span className='text-muted-foreground'>
                      {agent.cpu_cores} CPU · {formatMemory(agent.memory_bytes)}{' '}
                      · {agent.max_rps} RPS · C{agent.max_concurrency}
                    </span>
                    {(props.mode === 'local' || isAdmin) && (
                      <Button
                        aria-label={t('Remove agent')}
                        onClick={() => void removeAgent(agent.id)}
                        size='icon-xs'
                        variant='ghost'
                      >
                        <Trash2 className='size-3' />
                      </Button>
                    )}
                    {props.mode === 'managed' && isAdmin && (
                      <Button
                        aria-label={t('Edit')}
                        onClick={() => editCapacity(agent)}
                        size='icon-xs'
                        variant='ghost'
                      >
                        <Pencil className='size-3' />
                      </Button>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {runs.length > 0 && (
            <div className='overflow-x-auto rounded-md border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Run ID')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('API Key')}</TableHead>
                    <TableHead>{t('Test model')}</TableHead>
                    <TableHead>{t('Duration')}</TableHead>
                    <TableHead>{t('Requests')}</TableHead>
                    <TableHead>{t('Success rate')}</TableHead>
                    <TableHead>{t('Input tokens')}</TableHead>
                    <TableHead>{t('Output tokens')}</TableHead>
                    <TableHead>{t('Cache tokens')}</TableHead>
                    <TableHead>{t('Total tokens')}</TableHead>
                    <TableHead>{t('Average token price')}</TableHead>
                    <TableHead>{t('User charge')}</TableHead>
                    <TableHead>P95</TableHead>
                    <TableHead>{t('Errors')}</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {runs.map((run) => {
                    const successRate = run.completed
                      ? ((run.successes / run.completed) * 100).toFixed(1)
                      : '0.0'
                    const usage = {
                      successes: run.successes,
                      inputTokens: run.input_tokens,
                      outputTokens: run.output_tokens,
                      cacheReadTokens: run.cache_read_tokens,
                      cacheWriteTokens: run.cache_write_tokens,
                    }
                    const totalTokens = getLoadTestTotalTokens(usage)
                    const runPricing =
                      pricingByKey[
                        loadTestPricingKey(run.model, run.package_name)
                      ]
                    const userCharge = runPricing
                      ? calculateLoadTestUserCharge(usage, runPricing)
                      : null
                    const active = ACTIVE_RUN_STATUSES.has(run.status)
                    return (
                      <TableRow key={run.id}>
                        <TableCell>
                          <code className='text-xs'>{run.id}</code>
                        </TableCell>
                        <TableCell>
                          <Badge variant={active ? 'default' : 'secondary'}>
                            {t(run.status)}
                          </Badge>
                        </TableCell>
                        <TableCell>{run.key_name}</TableCell>
                        <TableCell>{run.model}</TableCell>
                        <TableCell>{run.duration_seconds}s</TableCell>
                        <TableCell className='tabular-nums'>
                          {run.completed.toLocaleString()}
                        </TableCell>
                        <TableCell>{successRate}%</TableCell>
                        <TableCell className='tabular-nums'>
                          {run.input_tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className='tabular-nums'>
                          {run.output_tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className='whitespace-nowrap tabular-nums'>
                          {run.cache_read_tokens.toLocaleString()} /{' '}
                          {run.cache_write_tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className='tabular-nums'>
                          {totalTokens.toLocaleString()}
                        </TableCell>
                        <TableCell className='whitespace-nowrap tabular-nums'>
                          {userCharge !== null && totalTokens > 0
                            ? `$${(userCharge / totalTokens).toFixed(8)}`
                            : '-'}
                        </TableCell>
                        <TableCell className='whitespace-nowrap tabular-nums'>
                          {userCharge === null
                            ? '-'
                            : `$${userCharge.toFixed(6)}`}
                        </TableCell>
                        <TableCell>{Math.round(run.p95_ms)} ms</TableCell>
                        <TableCell>
                          {Object.entries(run.error_counts)
                            .map(([code, count]) => `${code}: ${count}`)
                            .join(', ') || '-'}
                        </TableCell>
                        <TableCell>
                          {active && (
                            <Button
                              aria-label={t('Stop testing')}
                              onClick={() => void stop(run.id)}
                              size='icon-sm'
                              variant='outline'
                            >
                              <Square className='size-3' />
                            </Button>
                          )}
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

      <Dialog
        open={pairing !== null}
        onOpenChange={(open) => !open && setPairing(null)}
      >
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>
              {props.mode === 'managed'
                ? t('Pair server agent')
                : t('Pair local agent')}
            </DialogTitle>
            <DialogDescription>
              {t('Run this command on the computer that will generate load.')}
            </DialogDescription>
          </DialogHeader>
          {pairing && (
            <div className='space-y-3'>
              <pre className='bg-muted overflow-x-auto rounded-md border p-3 text-xs'>
                {props.mode === 'managed' &&
                window.location.protocol === 'http:' &&
                !['127.0.0.1', 'localhost', '::1'].includes(
                  window.location.hostname
                )
                  ? 'alltoken-loadtest-agent pair --server http://127.0.0.1:3000 '
                  : `alltoken-loadtest-agent pair --server ${window.location.origin} `}
                {pairing.code}
              </pre>
              <p className='text-muted-foreground text-xs'>
                {t('Pairing code expires in 5 minutes.')}
              </p>
            </div>
          )}
          <DialogFooter>
            <Button onClick={() => void copyPairCommand()}>
              <Copy className='size-4' />
              {t('Copy command')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={capacityAgent !== null}
        onOpenChange={(open) => !open && setCapacityAgent(null)}
      >
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('Edit server agent capacity')}</DialogTitle>
            <DialogDescription>
              {t('These limits apply to tasks submitted to this server agent.')}
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-2'>
            <div className='grid gap-2'>
              <Label htmlFor='managed-agent-max-rps'>
                {t('Maximum requests per second')}
              </Label>
              <Input
                id='managed-agent-max-rps'
                min={1}
                max={10000}
                onChange={(event) => setCapacityRPS(event.target.value)}
                type='number'
                value={capacityRPS}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='managed-agent-max-concurrency'>
                {t('Maximum concurrency')}
              </Label>
              <Input
                id='managed-agent-max-concurrency'
                min={1}
                max={10000}
                onChange={(event) => setCapacityConcurrency(event.target.value)}
                type='number'
                value={capacityConcurrency}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              disabled={savingCapacity}
              onClick={() => void saveCapacity()}
            >
              {t('Save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
