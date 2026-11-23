/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { useId, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Switch } from '@/components/ui/switch'
import {
  MODEL_LEADERBOARDS_QUERY_KEY,
  type ModelLeaderboardKind,
  type ModelLeaderboards,
} from '@/hooks/use-model-leaderboards'
import { api } from '@/lib/api'

import {
  SettingsSwitchContent,
  SettingsSwitchRow,
} from '../components/settings-form-layout'

type LeaderboardSettings = {
  enabled: boolean
  default: ModelLeaderboardKind
}

type LeaderboardResponse = {
  success: boolean
  message: string
  data?: ModelLeaderboards & LeaderboardSettings
}

const leaderboardsPath = '/api/option/model_leaderboards'
const settingsQueryKey = ['admin', 'model-leaderboards'] as const
const leaderboardOptions = [
  { kind: 'agent', label: 'Agent能力' },
  { kind: 'coding', label: 'Coding能力' },
  { kind: 'chat', label: 'Chat能力' },
] as const

const beijingDateTime = new Intl.DateTimeFormat('zh-CN', {
  timeZone: 'Asia/Shanghai',
  dateStyle: 'medium',
  timeStyle: 'short',
})

function readLeaderboardResponse(response: LeaderboardResponse) {
  if (!response.success || !response.data) {
    throw new Error(response.message || '无法读取模型排序榜单')
  }
  return response.data
}

function leaderboardErrorMessage(error: Error, fallback: string) {
  if (isAxiosError<LeaderboardResponse>(error)) {
    return error.response?.data?.message || fallback
  }
  return error.message || fallback
}

export function ModelLeaderboardsSection() {
  const id = useId()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<Partial<LeaderboardSettings>>({})
  const query = useQuery({
    queryKey: settingsQueryKey,
    queryFn: async () => {
      const { data } = await api.get<LeaderboardResponse>(leaderboardsPath, {
        skipBusinessError: true,
        skipErrorHandler: true,
      })
      return readLeaderboardResponse(data)
    },
  })

  function refreshModelOrder(data: ModelLeaderboards & LeaderboardSettings) {
    queryClient.setQueryData(settingsQueryKey, data)
    return Promise.all([
      queryClient.invalidateQueries({ queryKey: MODEL_LEADERBOARDS_QUERY_KEY }),
      queryClient.invalidateQueries({ queryKey: ['status'] }),
    ])
  }

  const save = useMutation({
    mutationFn: async (settings: LeaderboardSettings) => {
      const { data } = await api.put<LeaderboardResponse>(
        leaderboardsPath,
        settings,
        { skipBusinessError: true, skipErrorHandler: true }
      )
      return readLeaderboardResponse(data)
    },
    onSuccess: (data) => {
      setDraft({})
      toast.success('已保存榜单设置')
      return refreshModelOrder(data)
    },
    onError: (error) =>
      toast.error(leaderboardErrorMessage(error, '保存榜单设置失败')),
  })

  const sync = useMutation({
    mutationFn: async () => {
      const { data } = await api.post<LeaderboardResponse>(
        `${leaderboardsPath}/sync`,
        undefined,
        { timeout: 45_000, skipBusinessError: true, skipErrorHandler: true }
      )
      return readLeaderboardResponse(data)
    },
    onSuccess: (data) => {
      toast.success('已同步三个能力榜单')
      return refreshModelOrder(data)
    },
    // Keep the upstream failure visible in the section until the next attempt.
    onError: () => {},
  })

  const saved = query.data
  const enabled = draft.enabled ?? saved?.enabled
  const defaultBoard = draft.default ?? saved?.default
  const isBusy = save.isPending || sync.isPending
  const hasChanges =
    enabled !== saved?.enabled || defaultBoard !== saved?.default

  return (
    <section aria-labelledby={`${id}-title`} className='space-y-4'>
      <div className='flex flex-wrap items-baseline gap-x-3 gap-y-1'>
        <h3 id={`${id}-title`} className='text-sm font-semibold'>
          模型排序榜单
        </h3>
        <a
          className='text-muted-foreground hover:text-foreground text-xs underline underline-offset-4'
          href='https://huggingface.co/datasets/lmarena-ai/leaderboard-dataset'
          target='_blank'
          rel='noreferrer'
        >
          Arena.ai 榜单数据（CC BY 4.0）
        </a>
      </div>

      {query.isPending && (
        <p role='status' className='text-muted-foreground text-sm'>
          正在读取模型排序榜单…
        </p>
      )}
      {query.isError && (
        <div className='flex items-center gap-3'>
          <p role='alert' className='text-destructive text-sm'>
            {leaderboardErrorMessage(query.error, '读取模型排序榜单失败')}
          </p>
          <Button
            variant='outline'
            size='sm'
            onClick={() => void query.refetch()}
          >
            重试
          </Button>
        </div>
      )}

      {saved && enabled !== undefined && defaultBoard && (
        <>
          <SettingsSwitchRow>
            <SettingsSwitchContent>
              <Label htmlFor={`${id}-schedule`}>定时同步</Label>
              <p
                id={`${id}-schedule-hint`}
                className='text-muted-foreground text-xs'
              >
                开启后每天北京时间 04:00 同步三个能力榜单。
              </p>
            </SettingsSwitchContent>
            <Switch
              id={`${id}-schedule`}
              aria-describedby={`${id}-schedule-hint`}
              checked={enabled}
              disabled={isBusy}
              onCheckedChange={(checked) =>
                setDraft({ ...draft, enabled: checked })
              }
            />
          </SettingsSwitchRow>

          <div className='space-y-2'>
            <p id={`${id}-default`} className='text-sm font-medium'>
              默认榜单
            </p>
            <RadioGroup
              aria-labelledby={`${id}-default`}
              value={defaultBoard}
              className='grid-cols-1 sm:grid-cols-3'
              disabled={isBusy}
              onValueChange={(value) => {
                const option = leaderboardOptions.find(
                  ({ kind }) => kind === value
                )
                if (option) setDraft({ ...draft, default: option.kind })
              }}
            >
              {leaderboardOptions.map(({ kind, label }) => (
                <div
                  key={kind}
                  className='flex items-center gap-2 rounded-lg border px-3 py-2.5'
                >
                  <RadioGroupItem
                    id={`${id}-${kind}`}
                    value={kind}
                    aria-describedby={`${id}-${kind}-count`}
                  />
                  <Label htmlFor={`${id}-${kind}`} className='cursor-pointer'>
                    {label}
                  </Label>
                  <span
                    id={`${id}-${kind}-count`}
                    className='text-muted-foreground ml-auto text-xs tabular-nums'
                  >
                    {saved.boards[kind].length} 条
                  </span>
                </div>
              ))}
            </RadioGroup>
          </div>

          <div className='flex flex-wrap items-center gap-2'>
            <Button
              disabled={isBusy || !hasChanges}
              onClick={() => save.mutate({ enabled, default: defaultBoard })}
            >
              {save.isPending ? '保存中…' : '保存榜单设置'}
            </Button>
            <Button
              variant='outline'
              disabled={isBusy}
              onClick={() => sync.mutate()}
            >
              {sync.isPending ? '同步中…' : '立即同步'}
            </Button>
            <p role='status' className='text-muted-foreground text-xs'>
              {saved.updated_at
                ? `上次同步于 ${beijingDateTime.format(new Date(saved.updated_at))}（北京时间）`
                : '尚未同步'}
            </p>
          </div>
          {sync.isError && (
            <p role='alert' className='text-destructive text-sm'>
              {leaderboardErrorMessage(sync.error, '同步榜单失败')}
            </p>
          )}
        </>
      )}
    </section>
  )
}
