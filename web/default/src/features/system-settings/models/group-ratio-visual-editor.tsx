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
  Add01Icon,
  ArrowDown01Icon,
  ArrowUp01Icon,
  Delete02Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo, useCallback, useEffect, memo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
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
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import {
  getChannelMonitors,
  getPricingGroupMetrics,
  runChannelMonitor,
  updateChannelMonitor,
} from '@/features/channel-monitors/api'
import {
  ChannelMonitorFormCard,
  ChannelMonitorSheet,
} from '@/features/channel-monitors/components/monitor-sheet'
import { PricingGroupMonitorControl } from '@/features/channel-monitors/components/pricing-group-monitor-control'
import type {
  ChannelMonitor,
  ChannelMonitorSettingsPayload,
  PricingGroupMetrics,
} from '@/features/channel-monitors/types'
import { formatQuota } from '@/lib/format'

import { safeJsonParse } from '../utils/json-parser'

type GroupRatioVisualEditorProps = {
  groupRatio: string
  pricingGroupOrder: string
  pricingGroupRetryPolicy: string
  savedGroupRatio: string
  onChange: (field: string, value: string) => void
  onValidationChange: (isValid: boolean) => void
}

type GroupPricingRow = {
  _id: string
  name: string
  ratio: string
  retryMode: PricingGroupRetryMode
  retryTimes: string
}

type PricingGroupRetryMode = 'fixed' | 'active_channels'

type PricingGroupRetryPolicy = {
  mode: PricingGroupRetryMode
  retry_times: number
}

const sectionCardClassName =
  'relative shadow-sm ring-0 before:pointer-events-none before:absolute before:inset-0 before:rounded-xl before:border before:border-border/90'
const sectionHeaderClassName = 'border-b bg-muted/20'
const EMPTY_CHANNEL_MONITORS: ChannelMonitor[] = []
const EMPTY_PRICING_GROUP_METRICS: PricingGroupMetrics[] = []
const DEFAULT_GROUP_RETRY_TIMES = 3
const MAX_GROUP_RETRY_TIMES = 100
const RETRY_MODE_ITEMS = [
  { value: 'fixed', label: '固定次数' },
  { value: 'active_channels', label: '当前活跃渠道数' },
] as const

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

function parsePricingGroupOrder(value: string): string[] {
  return safeJsonParse<string[]>(value, {
    fallback: [],
    silent: true,
  }).filter((name) => typeof name === 'string' && name.trim() !== '')
}

function parsePricingGroupRetryPolicy(
  value: string
): Record<string, PricingGroupRetryPolicy> {
  const parsed = safeJsonParse<
    Record<string, Partial<PricingGroupRetryPolicy>>
  >(value, {
    fallback: {},
    silent: true,
  })
  const result: Record<string, PricingGroupRetryPolicy> = {}
  for (const [group, policy] of Object.entries(parsed)) {
    const mode: PricingGroupRetryMode =
      policy.mode === 'fixed' ? 'fixed' : 'active_channels'
    const retryTimes = Number(policy.retry_times)
    result[group] = {
      mode,
      retry_times:
        mode === 'fixed' && Number.isInteger(retryTimes)
          ? Math.min(MAX_GROUP_RETRY_TIMES, Math.max(0, retryTimes))
          : 0,
    }
  }
  return result
}

function normalizeRetryTimes(value: string): number {
  if (value.trim() === '') return DEFAULT_GROUP_RETRY_TIMES
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return DEFAULT_GROUP_RETRY_TIMES
  return Math.min(MAX_GROUP_RETRY_TIMES, Math.max(0, Math.trunc(parsed)))
}

function isValidFixedRetryTimes(value: string): boolean {
  if (value.trim() === '') return false
  const retryTimes = Number(value)
  return (
    Number.isInteger(retryTimes) &&
    retryTimes >= 0 &&
    retryTimes <= MAX_GROUP_RETRY_TIMES
  )
}

function isValidGroupRatio(value: string): boolean {
  if (value.trim() === '') return false
  const ratio = Number(value)
  return Number.isFinite(ratio) && ratio >= 0
}

function isValidGroupPricingRow(row: GroupPricingRow): boolean {
  if (!row.name.trim() || !isValidGroupRatio(row.ratio)) return false
  if (row.retryMode === 'active_channels') return true
  return isValidFixedRetryTimes(row.retryTimes)
}

function groupPricingRowsAreValid(rows: GroupPricingRow[]): boolean {
  const names = new Set<string>()
  for (const row of rows) {
    if (!isValidGroupPricingRow(row)) return false
    const name = row.name.trim()
    if (names.has(name)) return false
    names.add(name)
  }
  return true
}

function getOrderedGroupNames(
  groupRatio: string,
  pricingGroupOrder: string
): string[] {
  const ratioMap = parseRatioMap(groupRatio)
  const configuredNames = Object.keys(ratioMap)
  const configuredOrder = parsePricingGroupOrder(pricingGroupOrder)
  return [
    ...configuredOrder.filter((name) => name in ratioMap),
    ...configuredNames.filter((name) => !configuredOrder.includes(name)),
  ]
}

function buildGroupPricingRows(
  groupRatio: string,
  pricingGroupOrder: string,
  pricingGroupRetryPolicy: string
): GroupPricingRow[] {
  const ratioMap = parseRatioMap(groupRatio)
  const orderedNames = getOrderedGroupNames(groupRatio, pricingGroupOrder)
  const retryPolicies = parsePricingGroupRetryPolicy(pricingGroupRetryPolicy)

  return orderedNames.map((name) => {
    const retryPolicy = retryPolicies[name] ?? {
      mode: 'active_channels',
      retry_times: 0,
    }
    return {
      _id: createGroupPricingId(),
      name,
      ratio: String(normalizeRatio(ratioMap[name])),
      retryMode: retryPolicy.mode,
      retryTimes: String(retryPolicy.retry_times),
    }
  })
}

function serializeGroupPricingRows(rows: GroupPricingRow[]) {
  const groupRatio: Record<string, number> = {}
  const pricingGroupOrder: string[] = []
  const pricingGroupRetryPolicy: Record<string, PricingGroupRetryPolicy> = {}

  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue
    if (!(name in groupRatio)) pricingGroupOrder.push(name)
    groupRatio[name] = normalizeRatio(row.ratio)
    pricingGroupRetryPolicy[name] = {
      mode: row.retryMode,
      retry_times:
        row.retryMode === 'fixed' ? normalizeRetryTimes(row.retryTimes) : 0,
    }
  }

  return {
    GroupRatio: JSON.stringify(groupRatio, null, 2),
    PricingGroupOrder: JSON.stringify(pricingGroupOrder),
    PricingGroupRetryPolicy: JSON.stringify(pricingGroupRetryPolicy, null, 2),
  }
}

function groupPricingSignature(rows: GroupPricingRow[]): string {
  const serialized = serializeGroupPricingRows(rows)
  return JSON.stringify({
    ratios: parseRatioMap(serialized.GroupRatio),
    order: parsePricingGroupOrder(serialized.PricingGroupOrder),
    retryPolicies: parsePricingGroupRetryPolicy(
      serialized.PricingGroupRetryPolicy
    ),
  })
}

function sourceGroupPricingSignature(
  groupRatio: string,
  pricingGroupOrder: string,
  pricingGroupRetryPolicy: string
): string {
  const names = getOrderedGroupNames(groupRatio, pricingGroupOrder)
  const configuredPolicies = parsePricingGroupRetryPolicy(
    pricingGroupRetryPolicy
  )
  const retryPolicies: Record<string, PricingGroupRetryPolicy> = {}
  for (const name of names) {
    retryPolicies[name] = configuredPolicies[name] ?? {
      mode: 'active_channels',
      retry_times: 0,
    }
  }
  return JSON.stringify({
    ratios: parseRatioMap(groupRatio),
    order: names,
    retryPolicies,
  })
}

function findPricingGroupMonitor(
  row: GroupPricingRow,
  monitorByName: ReadonlyMap<string, ChannelMonitor>
): ChannelMonitor | null {
  return monitorByName.get(row.name.trim()) ?? null
}

export const GroupRatioVisualEditor = memo(function GroupRatioVisualEditor({
  groupRatio,
  pricingGroupOrder,
  pricingGroupRetryPolicy,
  savedGroupRatio,
  onChange,
  onValidationChange,
}: GroupRatioVisualEditorProps) {
  return (
    <GroupPricingTable
      groupRatio={groupRatio}
      pricingGroupOrder={pricingGroupOrder}
      pricingGroupRetryPolicy={pricingGroupRetryPolicy}
      savedGroupRatio={savedGroupRatio}
      onChange={onChange}
      onValidationChange={onValidationChange}
    />
  )
})

type GroupPricingTableProps = {
  groupRatio: string
  pricingGroupOrder: string
  pricingGroupRetryPolicy: string
  savedGroupRatio: string
  onChange: (field: string, value: string) => void
  onValidationChange: (isValid: boolean) => void
}

function GroupPricingTable({
  groupRatio,
  pricingGroupOrder,
  pricingGroupRetryPolicy,
  savedGroupRatio,
  onChange,
  onValidationChange,
}: GroupPricingTableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [rows, setRows] = useState<GroupPricingRow[]>(() =>
    buildGroupPricingRows(
      groupRatio,
      pricingGroupOrder,
      pricingGroupRetryPolicy
    )
  )
  const [monitorEditor, setMonitorEditor] = useState<{
    monitor: ChannelMonitor | null
    pricingGroupName: string
  } | null>(null)
  const [detailRowId, setDetailRowId] = useState<string | null>(null)
  const [deleteRowId, setDeleteRowId] = useState<string | null>(null)
  const incomingSignature = sourceGroupPricingSignature(
    groupRatio,
    pricingGroupOrder,
    pricingGroupRetryPolicy
  )
  const parsedRows = useMemo(
    () =>
      buildGroupPricingRows(
        groupRatio,
        pricingGroupOrder,
        pricingGroupRetryPolicy
      ),
    [groupRatio, pricingGroupOrder, pricingGroupRetryPolicy]
  )
  const currentRows =
    groupPricingSignature(rows) === incomingSignature ? rows : parsedRows
  const currentRowsAreValid = useMemo(
    () => groupPricingRowsAreValid(currentRows),
    [currentRows]
  )
  useEffect(() => {
    onValidationChange(currentRowsAreValid)
  }, [currentRowsAreValid, onValidationChange])
  const persistedGroupNames = useMemo(
    () => new Set(Object.keys(parseRatioMap(savedGroupRatio))),
    [savedGroupRatio]
  )
  const monitorsQuery = useQuery({
    queryKey: ['channel-monitors'],
    queryFn: getChannelMonitors,
    refetchInterval: 10_000,
  })
  const metricsQuery = useQuery({
    queryKey: ['pricing-group-metrics'],
    queryFn: getPricingGroupMetrics,
    refetchInterval: 3_000,
    refetchIntervalInBackground: false,
  })
  const monitors = monitorsQuery.data ?? EMPTY_CHANNEL_MONITORS
  const metrics = metricsQuery.data ?? EMPTY_PRICING_GROUP_METRICS
  const monitorByName = useMemo(
    () => new Map(monitors.map((monitor) => [monitor.pricing_group, monitor])),
    [monitors]
  )
  const metricsByName = useMemo(
    () => new Map(metrics.map((metric) => [metric.pricing_group, metric])),
    [metrics]
  )
  const detailRow = currentRows.find((row) => row._id === detailRowId) ?? null
  const detailMonitor = detailRow
    ? findPricingGroupMonitor(detailRow, monitorByName)
    : null
  const deleteRow = currentRows.find((row) => row._id === deleteRowId) ?? null

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
      onChange('PricingGroupOrder', serialized.PricingGroupOrder)
      onChange('PricingGroupRetryPolicy', serialized.PricingGroupRetryPolicy)
    },
    [onChange]
  )

  const updateRow = useCallback(
    (
      id: string,
      field: 'name' | 'ratio' | 'retryMode' | 'retryTimes',
      value: string
    ) => {
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
        retryMode: 'active_channels',
        retryTimes: '0',
      },
    ])
  }, [currentRows, emitRows])

  const removeRow = useCallback(
    (id: string) => {
      emitRows(currentRows.filter((row) => row._id !== id))
    },
    [currentRows, emitRows]
  )

  const moveRow = useCallback(
    (id: string, direction: -1 | 1) => {
      const sourceIndex = currentRows.findIndex((row) => row._id === id)
      const targetIndex = sourceIndex + direction
      if (
        sourceIndex < 0 ||
        targetIndex < 0 ||
        targetIndex >= currentRows.length
      ) {
        return
      }
      const nextRows = [...currentRows]
      const [moved] = nextRows.splice(sourceIndex, 1)
      nextRows.splice(targetIndex, 0, moved)
      emitRows(nextRows)
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
  const hasEmptyName = currentRows.some((row) => row.name.trim() === '')
  const hasInvalidRatio = currentRows.some(
    (row) => !isValidGroupRatio(row.ratio)
  )
  const hasInvalidRetryTimes = currentRows.some(
    (row) =>
      row.retryMode === 'fixed' && !isValidFixedRetryTimes(row.retryTimes)
  )

  return (
    <>
      <Card className={sectionCardClassName}>
        <CardHeader className={sectionHeaderClassName}>
          <Button onClick={addRow} size='sm' className='justify-self-end'>
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add group')}
          </Button>
        </CardHeader>
        <CardContent>
          <div className='space-y-3'>
            <StaticDataTable
              className='w-full'
              tableClassName='w-full min-w-[106rem] table-fixed'
              data={currentRows}
              getRowKey={(row) => row._id}
              emptyClassName='text-muted-foreground h-20 text-sm'
              emptyContent={t('No groups yet. Add a group to get started.')}
              columns={[
                {
                  id: 'order',
                  header: '顺序',
                  className: 'w-20',
                  cellClassName: 'w-20',
                  cell: (row, index) => (
                    <div className='flex items-center gap-0.5'>
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon-sm'
                        disabled={index === 0}
                        aria-label={`上移 ${row.name || '未命名分组'}`}
                        title='上移'
                        onClick={() => moveRow(row._id, -1)}
                      >
                        <HugeiconsIcon icon={ArrowUp01Icon} />
                      </Button>
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon-sm'
                        disabled={index === currentRows.length - 1}
                        aria-label={`下移 ${row.name || '未命名分组'}`}
                        title='下移'
                        onClick={() => moveRow(row._id, 1)}
                      >
                        <HugeiconsIcon icon={ArrowDown01Icon} />
                      </Button>
                    </div>
                  ),
                },
                {
                  id: 'group',
                  header: t('Group name'),
                  className: 'w-44',
                  cellClassName: 'w-44',
                  cell: (row) => {
                    return (
                      <Input
                        value={row.name}
                        onChange={(event) =>
                          updateRow(row._id, 'name', event.target.value)
                        }
                        aria-invalid={
                          !row.name.trim() ||
                          duplicateNames.includes(row.name.trim())
                        }
                      />
                    )
                  },
                },
                {
                  id: 'ratio',
                  header: t('Ratio'),
                  className: 'w-20',
                  cellClassName: 'w-20',
                  cell: (row) => (
                    <Input
                      type='number'
                      min={0}
                      step={0.1}
                      value={row.ratio}
                      aria-invalid={!isValidGroupRatio(row.ratio)}
                      onChange={(event) =>
                        updateRow(row._id, 'ratio', event.target.value)
                      }
                    />
                  ),
                },
                {
                  id: 'usage',
                  header: '用量',
                  className: 'w-52',
                  cellClassName: 'w-52',
                  cell: (row) => (
                    <GroupUsageCell
                      metrics={metricsByName.get(row.name.trim())}
                      isLoading={metricsQuery.isLoading}
                    />
                  ),
                },
                {
                  id: 'channels',
                  header: '渠道数',
                  className: 'w-36',
                  cellClassName: 'w-36',
                  cell: (row) => (
                    <GroupChannelCountCell
                      metrics={metricsByName.get(row.name.trim())}
                      isLoading={metricsQuery.isLoading}
                    />
                  ),
                },
                {
                  id: 'activity',
                  header: '活跃',
                  className: 'w-52',
                  cellClassName: 'w-52',
                  cell: (row) => (
                    <GroupActivityCell
                      metrics={metricsByName.get(row.name.trim())}
                      isLoading={metricsQuery.isLoading}
                    />
                  ),
                },
                {
                  id: 'retries',
                  header: '重试次数',
                  className: 'w-72',
                  cellClassName: 'w-72',
                  cell: (row) => (
                    <RetryPolicyControl
                      mode={row.retryMode}
                      retryTimes={row.retryTimes}
                      isInvalid={
                        row.retryMode === 'fixed' &&
                        !isValidFixedRetryTimes(row.retryTimes)
                      }
                      onModeChange={(value) =>
                        updateRow(row._id, 'retryMode', value)
                      }
                      onRetryTimesChange={(value) =>
                        updateRow(row._id, 'retryTimes', value)
                      }
                    />
                  ),
                },
                {
                  id: 'monitor',
                  header: '分组监控',
                  className: 'w-[22rem]',
                  cellClassName: 'w-[22rem]',
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
                      />
                    )
                  },
                },
                {
                  id: 'actions',
                  header: t('Actions'),
                  className: 'w-40 text-right',
                  cellClassName: 'w-40 text-right',
                  cell: (row) => {
                    const monitor = findPricingGroupMonitor(row, monitorByName)
                    const isRunning =
                      runMonitorMutation.isPending &&
                      runMonitorMutation.variables === monitor?.id
                    return (
                      <div className='flex justify-end gap-1'>
                        <Button
                          type='button'
                          variant='ghost'
                          size='sm'
                          disabled={!monitor || isRunning}
                          onClick={() => {
                            if (monitor) runMonitorMutation.mutate(monitor.id)
                          }}
                          aria-label='测试'
                        >
                          {isRunning && <Spinner data-icon='inline-start' />}
                          测试
                        </Button>
                        <Button
                          type='button'
                          variant='ghost'
                          size='sm'
                          onClick={() => setDetailRowId(row._id)}
                          disabled={!row.name.trim()}
                          aria-label='编辑'
                        >
                          编辑
                        </Button>
                        <Button
                          type='button'
                          variant='ghost'
                          size='sm'
                          onClick={() => setDeleteRowId(row._id)}
                          aria-label={t('Delete')}
                        >
                          <HugeiconsIcon icon={Delete02Icon} />
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

            {metricsQuery.isError && (
              <p className='text-destructive text-sm'>
                分组指标加载失败：{metricsQuery.error.message}
              </p>
            )}

            {duplicateNames.length > 0 && (
              <p className='text-destructive text-sm'>
                {t('Duplicate group names: {{names}}', {
                  names: duplicateNames.join(', '),
                })}
              </p>
            )}
            {hasEmptyName && (
              <p className='text-destructive text-sm'>分组名称不能为空</p>
            )}
            {hasInvalidRatio && (
              <p className='text-destructive text-sm'>
                倍率必须是大于等于 0 的数值
              </p>
            )}
            {hasInvalidRetryTimes && (
              <p className='text-destructive text-sm'>
                固定重试次数必须是 0 到 100 之间的整数
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
        monitor={detailMonitor}
        onOpenChange={(open) => {
          if (!open) setDetailRowId(null)
        }}
        onChange={(field, value) => {
          if (!detailRow) return
          updateRow(detailRow._id, field, value)
        }}
        isPersisted={
          detailRow !== null &&
          persistedGroupNames.has(detailRow.name.trim()) &&
          !duplicateNames.includes(detailRow.name.trim())
        }
        nameInvalid={
          detailRow !== null &&
          (!detailRow.name.trim() ||
            duplicateNames.includes(detailRow.name.trim()))
        }
      />

      <ConfirmDialog
        open={deleteRow !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteRowId(null)
        }}
        title='确认删除分组？'
        desc={
          deleteRow
            ? `删除“${deleteRow.name || '未命名分组'}”后，保存设置将同时移除该分组的监控配置和历史记录。`
            : ''
        }
        destructive
        confirmText='删除'
        handleConfirm={() => {
          if (!deleteRow) return
          removeRow(deleteRow._id)
          setDeleteRowId(null)
        }}
      />
    </>
  )
}

function formatMetricTokens(tokens: number): string {
  if (!Number.isFinite(tokens)) return '-'
  const absolute = Math.abs(tokens)
  let divisor = 1
  let suffix = ''
  if (absolute >= 1_000_000_000) {
    divisor = 1_000_000_000
    suffix = 'B'
  } else if (absolute >= 1_000_000) {
    divisor = 1_000_000
    suffix = 'M'
  } else if (absolute >= 1_000) {
    divisor = 1_000
    suffix = 'K'
  }
  const value = tokens / divisor
  let digits = 2
  if (Math.abs(value) >= 100) {
    digits = 0
  } else if (Math.abs(value) >= 10) {
    digits = 1
  }
  return `${Number(value.toFixed(digits))}${suffix}`
}

type GroupMetricCellProps = {
  metrics?: PricingGroupMetrics
  isLoading: boolean
}

function GroupUsageCell(props: GroupMetricCellProps) {
  if (props.isLoading) {
    return <span className='text-muted-foreground text-xs'>加载中...</span>
  }
  if (!props.metrics) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }
  return (
    <div className='space-y-0.5 text-xs leading-5'>
      <div>
        <span className='text-muted-foreground'>今日：</span>
        <span className='font-medium'>
          {formatMetricTokens(props.metrics.usage.today.tokens)}/
          {formatQuota(props.metrics.usage.today.quota)}
        </span>
      </div>
      <div>
        <span className='text-muted-foreground'>昨日：</span>
        <span className='font-medium'>
          {formatMetricTokens(props.metrics.usage.yesterday.tokens)}/
          {formatQuota(props.metrics.usage.yesterday.quota)}
        </span>
      </div>
      <div>
        <span className='text-muted-foreground'>累计：</span>
        <span className='font-medium'>
          {formatMetricTokens(props.metrics.usage.total.tokens)}/
          {formatQuota(props.metrics.usage.total.quota)}
        </span>
      </div>
    </div>
  )
}

function GroupChannelCountCell(props: GroupMetricCellProps) {
  if (props.isLoading) {
    return <span className='text-muted-foreground text-xs'>加载中...</span>
  }
  if (!props.metrics) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }
  return (
    <div className='space-y-0.5 text-xs leading-5'>
      <div>
        <span className='text-muted-foreground'>可用：</span>
        <span className='font-semibold'>
          {props.metrics.channels.available}个渠道
        </span>
      </div>
      <div>
        <span className='text-muted-foreground'>总量：</span>
        <span className='font-medium'>
          {props.metrics.channels.total}个渠道
        </span>
      </div>
    </div>
  )
}

function GroupActivityCell(props: GroupMetricCellProps) {
  if (props.isLoading) {
    return <span className='text-muted-foreground text-xs'>加载中...</span>
  }
  if (!props.metrics) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }
  return (
    <div
      className='text-xs leading-5 font-medium'
      title='连接数按当前正在处理的请求统计'
    >
      {`活跃用户${props.metrics.activity.users}/活跃连接${props.metrics.activity.connections}`}
    </div>
  )
}

type GroupDetailSheetProps = {
  row: GroupPricingRow | null
  monitor: ChannelMonitor | null
  onOpenChange: (open: boolean) => void
  onChange: (
    field: 'name' | 'ratio' | 'retryMode' | 'retryTimes',
    value: string
  ) => void
  isPersisted: boolean
  nameInvalid: boolean
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
          <SheetTitle>编辑{name ? `：${name.trim()}` : ''}</SheetTitle>
          <SheetDescription>
            修改分组倍率、重试策略和监控功能。
          </SheetDescription>
        </SheetHeader>

        {props.row && (
          <div className={sideDrawerFormClassName('gap-5')}>
            <div className='space-y-2'>
              <Label htmlFor='group-detail-name'>{t('Group name')}</Label>
              <Input
                id='group-detail-name'
                value={props.row.name}
                aria-invalid={props.nameInvalid}
                onChange={(event) => props.onChange('name', event.target.value)}
              />
            </div>
            <div className='space-y-2'>
              <Label>重试次数</Label>
              <RetryPolicyControl
                mode={props.row.retryMode}
                retryTimes={props.row.retryTimes}
                isInvalid={
                  props.row.retryMode === 'fixed' &&
                  !isValidFixedRetryTimes(props.row.retryTimes)
                }
                onModeChange={(value) => props.onChange('retryMode', value)}
                onRetryTimesChange={(value) =>
                  props.onChange('retryTimes', value)
                }
              />
              <p className='text-muted-foreground text-xs'>
                首次请求失败后允许再次尝试的次数；动态模式按本次请求可用的渠道数计算。API
                Key 和路由配置可以进一步限制实际重试次数。
              </p>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='group-detail-ratio'>{t('Ratio')}</Label>
              <Input
                id='group-detail-ratio'
                type='number'
                min={0}
                step={0.1}
                value={props.row.ratio}
                aria-invalid={!isValidGroupRatio(props.row.ratio)}
                onChange={(event) =>
                  props.onChange('ratio', event.target.value)
                }
              />
            </div>
            <Separator />
            <section className='flex flex-col gap-3'>
              <ChannelMonitorFormCard
                monitor={props.monitor}
                pricingGroupName={props.row.name.trim()}
                disabled={!props.isPersisted}
              />
              {!props.isPersisted && (
                <p className='text-muted-foreground text-xs'>
                  请先保存分组后配置监控。
                </p>
              )}
            </section>
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}

type RetryPolicyControlProps = {
  mode: PricingGroupRetryMode
  retryTimes: string
  isInvalid: boolean
  onModeChange: (value: PricingGroupRetryMode) => void
  onRetryTimesChange: (value: string) => void
}

function RetryPolicyControl(props: RetryPolicyControlProps) {
  return (
    <div className='flex min-w-0 items-center gap-2'>
      <Select
        items={RETRY_MODE_ITEMS}
        value={props.mode}
        onValueChange={(value) => {
          if (value === 'fixed' || value === 'active_channels') {
            props.onModeChange(value)
          }
        }}
      >
        <SelectTrigger className='w-40'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            <SelectItem value='fixed'>固定次数</SelectItem>
            <SelectItem value='active_channels'>当前活跃渠道数</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
      {props.mode === 'fixed' && (
        <Input
          type='number'
          min={0}
          max={MAX_GROUP_RETRY_TIMES}
          step={1}
          value={props.retryTimes}
          aria-invalid={props.isInvalid}
          onChange={(event) => props.onRetryTimesChange(event.target.value)}
          onBlur={() =>
            props.onRetryTimesChange(
              String(normalizeRetryTimes(props.retryTimes))
            )
          }
          aria-label='固定重试次数'
          className='w-20 tabular-nums'
        />
      )}
    </div>
  )
}
