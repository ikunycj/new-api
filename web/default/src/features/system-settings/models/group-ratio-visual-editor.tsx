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
import { Plus, Trash2 } from 'lucide-react'
import { useState, useMemo, useCallback, memo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  getChannelMonitors,
  runChannelMonitor,
  updateChannelMonitor,
} from '@/features/channel-monitors/api'
import { ChannelMonitorSheet } from '@/features/channel-monitors/components/monitor-sheet'
import { PricingGroupMonitorControl } from '@/features/channel-monitors/components/pricing-group-monitor-control'
import type {
  ChannelMonitor,
  ChannelMonitorSettingsPayload,
} from '@/features/channel-monitors/types'

import { safeJsonParse } from '../utils/json-parser'

type GroupRatioVisualEditorProps = {
  groupRatio: string
  savedGroupRatio: string
  onChange: (field: string, value: string) => void
}

type GroupPricingRow = {
  _id: string
  name: string
  ratio: string
}

const sectionCardClassName =
  'relative shadow-sm ring-0 before:pointer-events-none before:absolute before:inset-0 before:rounded-xl before:border before:border-border/90'
const sectionHeaderClassName = 'border-b bg-muted/20'
const EMPTY_CHANNEL_MONITORS: ChannelMonitor[] = []

let groupPricingIdCounter = 0
function createGroupPricingId() {
  groupPricingIdCounter += 1
  return `gpr_${groupPricingIdCounter}`
}

function normalizeRatio(value: unknown): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 1
}

function parseRatioMap(value: string): Record<string, number> {
  return safeJsonParse<Record<string, number>>(value, {
    fallback: {},
    silent: true,
  })
}

function buildGroupPricingRows(groupRatio: string): GroupPricingRow[] {
  const ratioMap = parseRatioMap(groupRatio)

  return Object.keys(ratioMap).map((name) => ({
    _id: createGroupPricingId(),
    name,
    ratio: String(normalizeRatio(ratioMap[name])),
  }))
}

function serializeGroupPricingRows(rows: GroupPricingRow[]) {
  const groupRatio: Record<string, number> = {}

  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue
    groupRatio[name] = normalizeRatio(row.ratio)
  }

  return {
    GroupRatio: JSON.stringify(groupRatio, null, 2),
  }
}

function groupPricingSignature(rows: GroupPricingRow[]): string {
  const serialized = serializeGroupPricingRows(rows)
  return JSON.stringify(parseRatioMap(serialized.GroupRatio))
}

function sourceGroupPricingSignature(groupRatio: string): string {
  return JSON.stringify(parseRatioMap(groupRatio))
}

function findPricingGroupMonitor(
  row: GroupPricingRow,
  monitorByName: ReadonlyMap<string, ChannelMonitor>
): ChannelMonitor | null {
  return monitorByName.get(row.name.trim()) ?? null
}

export const GroupRatioVisualEditor = memo(function GroupRatioVisualEditor({
  groupRatio,
  savedGroupRatio,
  onChange,
}: GroupRatioVisualEditorProps) {
  return (
    <GroupPricingTable
      groupRatio={groupRatio}
      savedGroupRatio={savedGroupRatio}
      onChange={onChange}
    />
  )
})

type GroupPricingTableProps = {
  groupRatio: string
  savedGroupRatio: string
  onChange: (field: string, value: string) => void
}

function GroupPricingTable({
  groupRatio,
  savedGroupRatio,
  onChange,
}: GroupPricingTableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [rows, setRows] = useState<GroupPricingRow[]>(() =>
    buildGroupPricingRows(groupRatio)
  )
  const [monitorEditor, setMonitorEditor] = useState<{
    monitor: ChannelMonitor | null
    pricingGroupName: string
  } | null>(null)
  const [detailRowId, setDetailRowId] = useState<string | null>(null)
  const incomingSignature = sourceGroupPricingSignature(groupRatio)
  const parsedRows = useMemo(
    () => buildGroupPricingRows(groupRatio),
    [groupRatio]
  )
  const currentRows =
    groupPricingSignature(rows) === incomingSignature ? rows : parsedRows
  const persistedGroupNames = useMemo(
    () => new Set(Object.keys(parseRatioMap(savedGroupRatio))),
    [savedGroupRatio]
  )

  const monitorsQuery = useQuery({
    queryKey: ['channel-monitors'],
    queryFn: getChannelMonitors,
    refetchInterval: 10_000,
  })
  const monitors = monitorsQuery.data ?? EMPTY_CHANNEL_MONITORS
  const monitorByName = useMemo(
    () => new Map(monitors.map((monitor) => [monitor.pricing_group, monitor])),
    [monitors]
  )
  const detailRow =
    currentRows.find((row) => row._id === detailRowId) ?? null

  const updateMonitorMutation = useMutation({
    mutationFn: ({
      monitor,
      enabled,
    }: {
      monitor: ChannelMonitor
      enabled: boolean
    }) => {
      const payload: ChannelMonitorSettingsPayload = {
        test_model: monitor.test_model,
        interval_seconds: monitor.interval_seconds,
        timeout_seconds: monitor.timeout_seconds,
        retry_count: monitor.retry_count,
        enabled,
        visible: monitor.visible,
        availability_boost_percent: monitor.availability_boost_percent,
      }
      return updateChannelMonitor(monitor.id, payload)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['channel-monitors'] }),
        queryClient.invalidateQueries({ queryKey: ['group-status'] }),
      ])
    },
    onError: (error) => toast.error(error.message || '更新监控失败'),
  })

  const runMonitorMutation = useMutation({
    mutationFn: runChannelMonitor,
    onSuccess: async (result) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['channel-monitors'] }),
        queryClient.invalidateQueries({ queryKey: ['group-status'] }),
      ])
      if (result.result.success) toast.success('可用性测试成功')
      else toast.error('可用性测试失败')
    },
    onError: (error) => toast.error(error.message || '可用性测试失败'),
  })

  const emitRows = useCallback(
    (nextRows: GroupPricingRow[]) => {
      setRows(nextRows)
      const serialized = serializeGroupPricingRows(nextRows)
      onChange('GroupRatio', serialized.GroupRatio)
    },
    [onChange]
  )

  const updateRow = useCallback(
    (id: string, field: 'name' | 'ratio', value: string) => {
      emitRows(
        currentRows.map((row) =>
          row._id === id ? { ...row, [field]: value } : row
        )
      )
    },
    [currentRows, emitRows]
  )

  const addRow = useCallback(() => {
    const existingNames = new Set(currentRows.map((row) => row.name))
    let index = 1
    let name = `group_${index}`
    while (existingNames.has(name)) {
      index += 1
      name = `group_${index}`
    }
    emitRows([
      ...currentRows,
      {
        _id: createGroupPricingId(),
        name,
        ratio: '1',
      },
    ])
  }, [currentRows, emitRows])

  const removeRow = useCallback(
    (id: string) => {
      emitRows(currentRows.filter((row) => row._id !== id))
    },
    [currentRows, emitRows]
  )

  const duplicateNames = useMemo(() => {
    const counts = new Map<string, number>()
    for (const row of currentRows) {
      const name = row.name.trim()
      if (!name) continue
      counts.set(name, (counts.get(name) ?? 0) + 1)
    }
    return [...counts.entries()]
      .filter(([, count]) => count > 1)
      .map(([name]) => name)
  }, [currentRows])

  return (
    <>
      <Card className={sectionCardClassName}>
        <CardHeader className={sectionHeaderClassName}>
          <Button onClick={addRow} size='sm' className='justify-self-end'>
            <Plus className='mr-2 h-4 w-4' />
            {t('Add group')}
          </Button>
        </CardHeader>
        <CardContent>
          <div className='space-y-3'>
            <StaticDataTable
              data={currentRows}
              getRowKey={(row) => row._id}
              emptyClassName='text-muted-foreground h-20 text-sm'
              emptyContent={t('No groups yet. Add a group to get started.')}
              columns={[
                {
                  id: 'group',
                  header: t('Group name'),
                  className: 'w-52',
                  cell: (row) => {
                    return (
                      <Input
                        value={row.name}
                        onChange={(event) =>
                          updateRow(row._id, 'name', event.target.value)
                        }
                        aria-invalid={duplicateNames.includes(row.name.trim())}
                      />
                    )
                  },
                },
                {
                  id: 'ratio',
                  header: t('Ratio'),
                  className: 'w-24',
                  cell: (row) => (
                    <Input
                      type='number'
                      min={0}
                      step={0.1}
                      value={row.ratio}
                      onChange={(event) =>
                        updateRow(row._id, 'ratio', event.target.value)
                      }
                    />
                  ),
                },
                {
                  id: 'monitor',
                  header: '分组监控',
                  className: 'w-[25rem]',
                  cell: (row) => {
                    const groupName = row.name.trim()
                    const monitor = findPricingGroupMonitor(row, monitorByName)
                    return (
                      <PricingGroupMonitorControl
                        groupName={groupName || '未命名分组'}
                        monitor={monitor}
                        isPersisted={
                          groupName !== '' &&
                          persistedGroupNames.has(groupName) &&
                          !duplicateNames.includes(groupName)
                        }
                        isLoading={monitorsQuery.isLoading}
                        hasError={monitorsQuery.isError}
                        isUpdating={
                          updateMonitorMutation.isPending &&
                          updateMonitorMutation.variables?.monitor.id ===
                            monitor?.id
                        }
                        isRunning={
                          runMonitorMutation.isPending &&
                          runMonitorMutation.variables === monitor?.id
                        }
                        onConfigure={() => {
                          if (!groupName) return
                          setMonitorEditor({
                            monitor,
                            pricingGroupName: groupName,
                          })
                        }}
                        onRetry={() => void monitorsQuery.refetch()}
                        onToggleEnabled={(enabled) => {
                          if (monitor) {
                            updateMonitorMutation.mutate({ monitor, enabled })
                          }
                        }}
                        onRun={() => {
                          if (monitor) runMonitorMutation.mutate(monitor.id)
                        }}
                      />
                    )
                  },
                },
                {
                  id: 'actions',
                  header: t('Actions'),
                  className: 'w-20 text-right',
                  cellClassName: 'w-20 text-right',
                  cell: (row) => {
                    return (
                      <div className='flex justify-end gap-1'>
                        <Button
                          variant='ghost'
                          size='sm'
                          onClick={() => setDetailRowId(row._id)}
                          disabled={!row.name.trim()}
                          aria-label='详情'
                        >
                          详情
                        </Button>
                        <Button
                          variant='ghost'
                          size='sm'
                          onClick={() => removeRow(row._id)}
                          aria-label={t('Delete')}
                        >
                          <Trash2 className='h-4 w-4' />
                        </Button>
                      </div>
                    )
                  },
                },
              ]}
            />

            {monitorsQuery.isError && (
              <p className='text-destructive text-sm'>
                分组监控加载失败：{monitorsQuery.error.message}
              </p>
            )}

            {duplicateNames.length > 0 && (
              <p className='text-destructive text-sm'>
                {t('Duplicate group names: {{names}}', {
                  names: duplicateNames.join(', '),
                })}
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      <ChannelMonitorSheet
        open={monitorEditor !== null}
        monitor={monitorEditor?.monitor ?? null}
        pricingGroupName={monitorEditor?.pricingGroupName ?? ''}
        onOpenChange={(open) => {
          if (!open) setMonitorEditor(null)
        }}
      />

      <GroupDetailSheet
        row={detailRow}
        monitor={detailRow ? findPricingGroupMonitor(detailRow, monitorByName) : null}
        onOpenChange={(open) => {
          if (!open) setDetailRowId(null)
        }}
        onChange={(field, value) => {
          if (!detailRow) return
          updateRow(detailRow._id, field, value)
        }}
        onConfigure={() => {
          if (!detailRow) return
          const groupName = detailRow.name.trim()
          if (!groupName) return
          setMonitorEditor({
            monitor: findPricingGroupMonitor(detailRow, monitorByName),
            pricingGroupName: groupName,
          })
        }}
        isPersisted={
          detailRow !== null &&
          persistedGroupNames.has(detailRow.name.trim()) &&
          !duplicateNames.includes(detailRow.name.trim())
        }
      />
    </>
  )
}

type GroupDetailSheetProps = {
  row: GroupPricingRow | null
  monitor: ChannelMonitor | null
  onOpenChange: (open: boolean) => void
  onChange: (field: 'name' | 'ratio', value: string) => void
  onConfigure: () => void
  isPersisted: boolean
}

function GroupDetailSheet(props: GroupDetailSheetProps) {
  const { t } = useTranslation()
  const name = props.row?.name ?? ''

  return (
    <Sheet open={props.row !== null} onOpenChange={props.onOpenChange}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-lg')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            详情{name ? `：${name.trim()}` : ''}
          </SheetTitle>
          <SheetDescription>
            在此修改分组倍率和分组监控参数。
          </SheetDescription>
        </SheetHeader>

        {props.row && (
          <div className={sideDrawerFormClassName('gap-5')}>
            <div className='space-y-2'>
              <Label htmlFor='group-detail-name'>{t('Group name')}</Label>
              <Input
                id='group-detail-name'
                value={props.row.name}
                onChange={(event) => props.onChange('name', event.target.value)}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='group-detail-ratio'>{t('Ratio')}</Label>
              <Input
                id='group-detail-ratio'
                type='number'
                min={0}
                step={0.1}
                value={props.row.ratio}
                onChange={(event) => props.onChange('ratio', event.target.value)}
              />
            </div>
            <section className='space-y-2 border-t pt-4'>
              <h3 className='text-sm font-semibold'>分组监控</h3>
              <p className='text-muted-foreground text-sm'>
                {props.monitor
                  ? `已配置：${props.monitor.test_model}，每 ${props.monitor.interval_seconds} 秒测试 ${props.monitor.retry_count} 次`
                  : '尚未配置分组监控'}
              </p>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={!props.isPersisted}
                onClick={props.onConfigure}
              >
                {props.monitor ? '编辑监控参数' : '配置分组监控'}
              </Button>
              {!props.isPersisted && (
                <p className='text-muted-foreground text-xs'>请先保存分组后配置监控。</p>
              )}
            </section>
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}
