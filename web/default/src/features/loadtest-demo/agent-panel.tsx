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
import { Copy, Link2, MonitorCog, Play, Square, Trash2 } from 'lucide-react'
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

import {
  cancelLoadTestAgentRun,
  createLoadTestAgentPairing,
  createLoadTestAgentRun,
  deleteLoadTestAgent,
  getLoadTestAgentState,
  type CreateLoadTestAgentRun,
  type LoadTestAgent,
  type LoadTestAgentRun,
} from './api'

type AgentPanelProps = {
  disabled: boolean
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

export function AgentPanel(props: AgentPanelProps) {
  const { t } = useTranslation()
  const [agents, setAgents] = useState<LoadTestAgent[]>([])
  const [onlineBefore, setOnlineBefore] = useState(0)
  const [selectedAgentId, setSelectedAgentId] = useState('')
  const [runs, setRuns] = useState<LoadTestAgentRun[]>([])
  const [pairing, setPairing] = useState<Pairing | null>(null)
  const [loading, setLoading] = useState(true)
  const [starting, setStarting] = useState(false)
  const refreshInFlightRef = useRef(false)

  const refresh = useCallback(async (showError = false) => {
    if (refreshInFlightRef.current) return
    refreshInFlightRef.current = true
    try {
      const state = await getLoadTestAgentState()
      setAgents(state.agents)
      setOnlineBefore(state.online_before)
      setRuns(state.runs)
      setSelectedAgentId((current) => {
        const currentAgent = state.agents.find((agent) => agent.id === current)
        if (
          currentAgent &&
          currentAgent.last_seen_at >= state.online_before
        ) {
          return current
        }
        return (
          state.agents.find(
            (agent) => agent.last_seen_at >= state.online_before
          )?.id ?? ''
        )
      })
    } catch (error) {
      if (showError) {
        toast.error(error instanceof Error ? error.message : t('Request failed'))
      }
    } finally {
      refreshInFlightRef.current = false
      setLoading(false)
    }
  }, [t])

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
      setPairing(await createLoadTestAgentPairing())
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    }
  }, [t])

  const copyPairCommand = useCallback(async () => {
    if (!pairing) return
    const command = `alltoken-loadtest-agent pair --server ${window.location.origin} ${pairing.code}`
    await navigator.clipboard.writeText(command)
    toast.success(t('Copied'))
  }, [pairing, t])

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
        toast.error(error instanceof Error ? error.message : t('Request failed'))
      }
    },
    [refresh, t]
  )

  const removeAgent = useCallback(
    async (agentId: string) => {
      try {
        await deleteLoadTestAgent(agentId)
        await refresh()
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t('Request failed'))
      }
    },
    [refresh, t]
  )

  const onlineAgents = agents.filter(
    (agent) => agent.last_seen_at >= onlineBefore
  )

  return (
    <>
      <Card>
        <CardHeader>
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div>
              <CardTitle className='flex items-center gap-2'>
                <MonitorCog className='size-5' />
                {t('Local load-test agent')}
              </CardTitle>
              <CardDescription className='mt-1'>
                {t(
                  'Run high-volume tests outside the browser and keep results linked to this account.'
                )}
              </CardDescription>
            </div>
            <Button onClick={() => void createPairing()} variant='outline'>
              <Link2 className='size-4' />
              {t('Pair agent')}
            </Button>
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
                  <SelectValue placeholder={t('No online agents')} />
                </SelectTrigger>
                <SelectContent>
                  {onlineAgents.map((agent) => (
                    <SelectItem key={agent.id} value={agent.id}>
                      {agent.name} · {agent.platform}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button
              disabled={
                props.disabled ||
                !props.request ||
                !selectedAgentId ||
                starting
              }
              onClick={() => void start()}
            >
              <Play className='size-4' />
              {t('Start with agent')}
            </Button>
          </div>

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
                    <Button
                      aria-label={t('Remove agent')}
                      onClick={() => void removeAgent(agent.id)}
                      size='icon-xs'
                      variant='ghost'
                    >
                      <Trash2 className='size-3' />
                    </Button>
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
                    <TableHead>{t('Requests')}</TableHead>
                    <TableHead>{t('Success rate')}</TableHead>
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
                        <TableCell>{run.completed}</TableCell>
                        <TableCell>{successRate}%</TableCell>
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

      <Dialog open={pairing !== null} onOpenChange={(open) => !open && setPairing(null)}>
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('Pair local agent')}</DialogTitle>
            <DialogDescription>
              {t('Run this command on the computer that will generate load.')}
            </DialogDescription>
          </DialogHeader>
          {pairing && (
            <div className='space-y-3'>
              <pre className='bg-muted overflow-x-auto rounded-md border p-3 text-xs'>
                alltoken-loadtest-agent pair --server {window.location.origin}{' '}
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
    </>
  )
}
