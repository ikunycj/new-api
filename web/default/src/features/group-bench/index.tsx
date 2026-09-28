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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { getPricingGroupNames } from '@/features/system-settings/billing/user-groups-api'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  getGroupBenchConfig,
  getGroupBenchRuns,
  runGroupBenchNow,
  updateGroupBenchConfig,
} from './api'
import { BenchArtifact } from './components/bench-artifact'
import type {
  GroupBenchConfigPayload,
  GroupBenchEndpointType,
  GroupBenchModel,
  GroupBenchRun,
  SvgMetrics,
} from './types'

// rowId only keys the editable rows; it is stripped before saving.
type FormModel = GroupBenchModel & { rowId: string }
type BenchForm = Omit<GroupBenchConfigPayload, 'models'> & {
  models: FormModel[]
}

const ENDPOINT_OPTIONS: { value: GroupBenchEndpointType; label: string }[] = [
  { value: 'openai', label: 'OpenAI Chat' },
  { value: 'openai-response', label: 'OpenAI Responses' },
  { value: 'anthropic', label: 'Anthropic Messages' },
]

const DAY_SECONDS = 24 * 3600

export function GroupBenchSection() {
  const { t } = useTranslation()
  const [selectedGroup, setSelectedGroup] = useState('')
  const groupsQuery = useQuery({
    queryKey: ['pricing-group-names'],
    queryFn: getPricingGroupNames,
  })
  const groups = groupsQuery.data ?? []
  const group = selectedGroup || groups[0] || ''

  return (
    <div className='space-y-4'>
      <div className='flex items-center gap-3'>
        <span className='text-sm font-medium'>{t('Pricing group')}</span>
        <NativeSelect
          value={group}
          onChange={(event) => setSelectedGroup(event.target.value)}
          disabled={groups.length === 0}
          className='min-w-48'
        >
          {groups.map((name) => (
            <NativeSelectOption key={name} value={name}>
              {name}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      </div>
      {groupsQuery.isLoading && <Skeleton className='h-40 w-full' />}
      {group && <GroupBenchPanel key={group} group={group} />}
    </div>
  )
}

function GroupBenchPanel({ group }: { group: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<BenchForm | null>(null)
  const [selectedCell, setSelectedCell] = useState<{
    channelId: number
    model: string
  } | null>(null)

  const configQuery = useQuery({
    queryKey: ['group-bench-config', group],
    queryFn: () => getGroupBenchConfig(group),
  })
  const runsQuery = useQuery({
    queryKey: ['group-bench-runs', group],
    queryFn: () =>
      getGroupBenchRuns({
        group,
        since: Math.floor(Date.now() / 1000) - DAY_SECONDS,
      }),
    refetchInterval: 60_000,
  })

  const config = configQuery.data
  const form: BenchForm | null =
    draft ??
    (config
      ? {
          group,
          enabled: config.enabled,
          preset: config.preset,
          models: config.models.map((item) => ({
            ...item,
            rowId: crypto.randomUUID(),
          })),
          interval_minutes: config.interval_minutes,
        }
      : null)

  const saveMutation = useMutation({
    mutationFn: updateGroupBenchConfig,
    onSuccess: () => {
      setDraft(null)
      toast.success(t('Saved successfully'))
      void queryClient.invalidateQueries({
        queryKey: ['group-bench-config', group],
      })
    },
    onError: (error) => toast.error(error.message || t('Operation failed')),
  })
  const runMutation = useMutation({
    mutationFn: () => runGroupBenchNow(group),
    onSuccess: () => {
      toast.success(t('Bench round queued'))
      void queryClient.invalidateQueries({
        queryKey: ['group-bench-config', group],
      })
    },
    onError: (error) => toast.error(error.message || t('Operation failed')),
  })

  const latestRound = useMemo(() => {
    const runs = runsQuery.data ?? []
    const roundAt = Math.max(0, ...runs.map((run) => run.round_at))
    const roundRuns = runs.filter((run) => run.round_at === roundAt)
    const channels = new Map<number, string>()
    for (const run of roundRuns) channels.set(run.channel_id, run.channel_name)
    const models = [...new Set(roundRuns.map((run) => run.model))].sort()
    const cells = new Map(
      roundRuns.map((run) => [`${run.channel_id}|${run.model}`, run])
    )
    return { roundAt, channels: [...channels], models, cells }
  }, [runsQuery.data])

  if (configQuery.isLoading || !form || !config) {
    return <Skeleton className='h-64 w-full' />
  }

  const updateForm = (changes: Partial<BenchForm>) =>
    setDraft({ ...form, ...changes })

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <CardTitle>{t('Group bench')}</CardTitle>
          <CardDescription>
            {t(
              'Sends the bench prompt to every enabled channel of this group on a schedule. Failures are only recorded and never disable channels.'
            )}
          </CardDescription>
          <CardAction className='flex items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              disabled={runMutation.isPending || config.manual_pending}
              onClick={() => runMutation.mutate()}
            >
              {config.manual_pending ? t('Run queued') : t('Run now')}
            </Button>
            <Button
              size='sm'
              disabled={!draft || saveMutation.isPending}
              onClick={() =>
                saveMutation.mutate({
                  ...form,
                  models: form.models.map(({ model, endpoint_type }) => ({
                    model,
                    endpoint_type,
                  })),
                })
              }
            >
              {t('Save')}
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex flex-wrap items-center gap-6 text-sm'>
            <label className='flex items-center gap-2'>
              <Switch
                checked={form.enabled}
                onCheckedChange={(enabled) => updateForm({ enabled })}
              />
              {t('Scheduled bench')}
            </label>
            <label className='flex items-center gap-2'>
              {t('Interval (minutes)')}
              <Input
                type='number'
                min={15}
                max={1440}
                className='w-24'
                value={form.interval_minutes}
                onChange={(event) =>
                  updateForm({ interval_minutes: Number(event.target.value) })
                }
              />
            </label>
            <label className='flex items-center gap-2'>
              {t('Preset')}
              <NativeSelect
                value={form.preset}
                onChange={(event) => updateForm({ preset: event.target.value })}
              >
                {config.presets.map((preset) => (
                  <NativeSelectOption key={preset.key} value={preset.key}>
                    {preset.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
            <span className='text-muted-foreground'>
              {t('Last run')}: {formatTimestampToDate(config.last_run_at ?? 0)}
              {config.enabled &&
                ` · ${t('Next run')}: ${formatTimestampToDate(config.next_run_at)}`}
            </span>
          </div>

          <div className='space-y-2'>
            <div className='text-sm font-medium'>{t('Models')}</div>
            {form.models.map((benchModel, index) => (
              <div key={benchModel.rowId} className='flex items-center gap-2'>
                <Input
                  className='w-64'
                  list={`group-bench-models-${group}`}
                  value={benchModel.model}
                  placeholder={t('Model name')}
                  onChange={(event) =>
                    updateForm({
                      models: form.models.map((item, i) =>
                        i === index
                          ? { ...item, model: event.target.value }
                          : item
                      ),
                    })
                  }
                />
                <NativeSelect
                  value={benchModel.endpoint_type}
                  onChange={(event) =>
                    updateForm({
                      models: form.models.map((item, i) =>
                        i === index
                          ? {
                              ...item,
                              endpoint_type: event.target
                                .value as GroupBenchEndpointType,
                            }
                          : item
                      ),
                    })
                  }
                >
                  {ENDPOINT_OPTIONS.map((option) => (
                    <NativeSelectOption key={option.value} value={option.value}>
                      {option.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <Button
                  variant='ghost'
                  size='sm'
                  onClick={() =>
                    updateForm({
                      models: form.models.filter((_, i) => i !== index),
                    })
                  }
                >
                  {t('Remove')}
                </Button>
              </div>
            ))}
            <datalist id={`group-bench-models-${group}`}>
              {config.group_models.map((name) => (
                <option key={name} value={name} />
              ))}
            </datalist>
            <Button
              variant='outline'
              size='sm'
              onClick={() =>
                updateForm({
                  models: [
                    ...form.models,
                    {
                      model: '',
                      endpoint_type: 'openai',
                      rowId: crypto.randomUUID(),
                    },
                  ],
                })
              }
            >
              {t('Add model')}
            </Button>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Codex channels always use OpenAI Responses. Channels without the model in this group are skipped.'
              )}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Latest round')}</CardTitle>
          <CardDescription>
            {latestRound.roundAt
              ? formatTimestampToDate(latestRound.roundAt)
              : t('No bench results in the last 24 hours')}
          </CardDescription>
        </CardHeader>
        {latestRound.roundAt > 0 && (
          <CardContent className='overflow-x-auto'>
            <table className='w-full border-separate border-spacing-2 text-sm'>
              <thead>
                <tr>
                  <th className='text-left font-medium'>{t('Channel')}</th>
                  {latestRound.models.map((name) => (
                    <th key={name} className='text-left font-medium'>
                      {name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {latestRound.channels.map(([channelId, channelName]) => (
                  <tr key={channelId}>
                    <td className='align-top whitespace-nowrap'>
                      #{channelId} {channelName}
                    </td>
                    {latestRound.models.map((name) => {
                      const run = latestRound.cells.get(`${channelId}|${name}`)
                      const selected =
                        selectedCell?.channelId === channelId &&
                        selectedCell.model === name
                      return (
                        <td key={name} className='w-56 align-top'>
                          {run ? (
                            <button
                              type='button'
                              className={cn(
                                'w-full rounded-lg p-1 text-left',
                                selected && 'ring-primary ring-2'
                              )}
                              onClick={() =>
                                setSelectedCell({ channelId, model: name })
                              }
                            >
                              <RunTile run={run} />
                            </button>
                          ) : (
                            <span className='text-muted-foreground'>-</span>
                          )}
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        )}
      </Card>

      {selectedCell && (
        <RunTimeline
          group={group}
          channelId={selectedCell.channelId}
          model={selectedCell.model}
        />
      )}
    </div>
  )
}

function RunTimeline({
  group,
  channelId,
  model,
}: {
  group: string
  channelId: number
  model: string
}) {
  const { t } = useTranslation()
  const timelineQuery = useQuery({
    queryKey: ['group-bench-timeline', group, channelId, model],
    queryFn: () =>
      getGroupBenchRuns({
        group,
        channel_id: channelId,
        model,
        since: Math.floor(Date.now() / 1000) - 7 * DAY_SECONDS,
        limit: 168,
      }),
  })
  const runs = timelineQuery.data ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {t('History')} · #{channelId} · {model}
        </CardTitle>
        <CardDescription>{t('Last 7 days, newest first')}</CardDescription>
      </CardHeader>
      <CardContent>
        {timelineQuery.isLoading ? (
          <Skeleton className='h-48 w-full' />
        ) : (
          <div className='flex gap-3 overflow-x-auto pb-2'>
            {runs.map((run) => (
              <div key={run.id} className='w-44 shrink-0'>
                <div className='text-muted-foreground mb-1 text-xs'>
                  {formatTimestampToDate(run.round_at)}
                </div>
                <RunTile run={run} />
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function RunTile({ run }: { run: GroupBenchRun }) {
  const { t } = useTranslation()
  const metrics = run.metrics ? (JSON.parse(run.metrics) as SvgMetrics) : null

  return (
    <div className='space-y-1'>
      {run.has_artifact ? (
        <BenchArtifact runId={run.id} />
      ) : (
        <div className='bg-muted text-destructive flex aspect-square w-full items-center justify-center overflow-hidden rounded-md p-2 text-xs break-all'>
          {run.error_message || t('Failed')}
        </div>
      )}
      <div className='text-muted-foreground text-xs'>
        {(run.latency_ms / 1000).toFixed(1)}s · {run.output_tokens} tok
      </div>
      <div className='flex flex-wrap gap-1'>
        {!run.success && <Badge variant='destructive'>{t('Failed')}</Badge>}
        {metrics && !metrics.animated && (
          <Badge variant='destructive'>{t('Static')}</Badge>
        )}
        {metrics?.animated && (
          <Badge variant='secondary'>
            {t('Animations')} ×
            {metrics.animate +
              metrics.animate_transform +
              metrics.animate_motion +
              metrics.css_keyframes}
          </Badge>
        )}
        {metrics?.has_script && (
          <Badge variant='destructive'>{t('Contains script')}</Badge>
        )}
        {run.truncated && <Badge variant='destructive'>{t('Truncated')}</Badge>}
      </div>
    </div>
  )
}
