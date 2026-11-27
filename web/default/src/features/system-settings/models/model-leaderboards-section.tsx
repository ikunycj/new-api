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
import { useId } from 'react'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import {
  SettingsSwitchContent,
  SettingsSwitchRow,
} from '../components/settings-form-layout'
import {
  leaderboardErrorMessage,
  type useModelLeaderboardSettings,
} from './use-model-leaderboard-settings'

const beijingDateTime = new Intl.DateTimeFormat('zh-CN', {
  timeZone: 'Asia/Shanghai',
  dateStyle: 'medium',
  timeStyle: 'short',
})

type ModelLeaderboardsSectionProps = {
  settings: ReturnType<typeof useModelLeaderboardSettings>
  isSaving: boolean
}

export function ModelLeaderboardsSection(props: ModelLeaderboardsSectionProps) {
  const id = useId()
  const { query, sync, enabled, recommendedModels, setDraft } = props.settings
  const saved = query.data
  const isBusy = props.isSaving || sync.isPending

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

      {saved && enabled !== undefined && (
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
                setDraft((current) => ({ ...current, enabled: checked }))
              }
            />
          </SettingsSwitchRow>

          <div className='flex flex-wrap items-center gap-2'>
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

          <div className='space-y-2 border-t pt-4'>
            <Label
              htmlFor={`${id}-recommended-models`}
              className='font-semibold'
            >
              推荐模型榜单
            </Label>
            <p
              id={`${id}-recommended-models-hint`}
              className='text-muted-foreground text-xs'
            >
              输入模型 ID
              后按回车添加。拖动标签左侧手柄，或聚焦手柄后按左右方向键调整顺序。模型广场按此顺序展示「性价比优先」的推荐模型。
            </p>
            <MultiSelect
              id={`${id}-recommended-models`}
              aria-describedby={`${id}-recommended-models-hint`}
              selected={recommendedModels}
              options={[]}
              onChange={(models) =>
                setDraft((current) => ({
                  ...current,
                  recommendedModels: models,
                }))
              }
              allowCreate
              placeholder='输入模型 ID'
              createLabel='Add model "{{value}}"'
              emptyText='暂无推荐模型'
              reorderLabel={(label, position) =>
                `移动 ${label}，第 ${position} 位`
              }
              disabled={isBusy}
            />
          </div>
        </>
      )}
    </section>
  )
}
