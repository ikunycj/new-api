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
import { Info, Plus, Trash2 } from 'lucide-react'
import { useState, useMemo, useEffect, useCallback, memo } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import { safeJsonParse } from '../utils/json-parser'

type GroupRatioVisualEditorProps = {
  groupRatio: string
  topupGroupRatio: string
  userUsableGroups: string
  onChange: (field: string, value: string) => void
}

type GroupPricingRow = {
  _id: string
  name: string
  ratio: string
  topupRatio: string
  selectable: boolean
  description: string
}

type RegistryEntry = {
  name: string
  ratio: number
}

const sectionCardClassName =
  'relative shadow-sm ring-0 before:pointer-events-none before:absolute before:inset-0 before:rounded-xl before:border before:border-border/90'
const sectionHeaderClassName = 'border-b bg-muted/20'

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

function parseUsableMap(value: string): Record<string, string> {
  return safeJsonParse<Record<string, string>>(value, {
    fallback: {},
    silent: true,
  })
}

function buildGroupPricingRows(
  groupRatio: string,
  userUsableGroups: string,
  topupGroupRatio: string
): GroupPricingRow[] {
  const ratioMap = parseRatioMap(groupRatio)
  const usableMap = parseUsableMap(userUsableGroups)
  const topupMap = parseRatioMap(topupGroupRatio)
  const names = new Set([
    ...Object.keys(ratioMap),
    ...Object.keys(usableMap),
    ...Object.keys(topupMap),
  ])

  return [...names].map((name) => ({
    _id: createGroupPricingId(),
    name,
    ratio: String(normalizeRatio(ratioMap[name])),
    topupRatio: Object.hasOwn(topupMap, name) ? String(topupMap[name]) : '',
    selectable: Object.hasOwn(usableMap, name),
    description: String(usableMap[name] ?? ''),
  }))
}

function serializeGroupPricingRows(rows: GroupPricingRow[]) {
  const groupRatio: Record<string, number> = {}
  const userUsableGroups: Record<string, string> = {}
  const topupGroupRatio: Record<string, number> = {}

  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue
    groupRatio[name] = normalizeRatio(row.ratio)
    if (row.selectable) {
      userUsableGroups[name] = row.description
    }
    const topup = row.topupRatio.trim()
    if (topup !== '' && Number.isFinite(Number(topup))) {
      topupGroupRatio[name] = Number(topup)
    }
  }

  return {
    GroupRatio: JSON.stringify(groupRatio, null, 2),
    UserUsableGroups: JSON.stringify(userUsableGroups, null, 2),
    TopupGroupRatio: JSON.stringify(topupGroupRatio, null, 2),
  }
}

function groupPricingSignature(rows: GroupPricingRow[]): string {
  const serialized = serializeGroupPricingRows(rows)
  return JSON.stringify({
    groupRatio: parseRatioMap(serialized.GroupRatio),
    userUsableGroups: parseUsableMap(serialized.UserUsableGroups),
    topupGroupRatio: parseRatioMap(serialized.TopupGroupRatio),
  })
}

function sourceGroupPricingSignature(
  groupRatio: string,
  userUsableGroups: string,
  topupGroupRatio: string
): string {
  return JSON.stringify({
    groupRatio: parseRatioMap(groupRatio),
    userUsableGroups: parseUsableMap(userUsableGroups),
    topupGroupRatio: parseRatioMap(topupGroupRatio),
  })
}

export const GroupRatioVisualEditor = memo(function GroupRatioVisualEditor({
  groupRatio,
  topupGroupRatio,
  userUsableGroups,
  onChange,
}: GroupRatioVisualEditorProps) {
  const [detailGroup, setDetailGroup] = useState<string | null>(null)

  const registry = useMemo<RegistryEntry[]>(() => {
    const ratioMap = parseRatioMap(groupRatio)
    const usableMap = parseUsableMap(userUsableGroups)
    const topupMap = parseRatioMap(topupGroupRatio)
    const names = new Set([
      ...Object.keys(ratioMap),
      ...Object.keys(usableMap),
      ...Object.keys(topupMap),
    ])
    return [...names].map((name) => ({
      name,
      ratio: normalizeRatio(ratioMap[name]),
    }))
  }, [groupRatio, userUsableGroups, topupGroupRatio])

  return (
    <div className='space-y-4'>
      <GroupPricingTable
        groupRatio={groupRatio}
        userUsableGroups={userUsableGroups}
        topupGroupRatio={topupGroupRatio}
        onChange={onChange}
        onShowDetail={setDetailGroup}
      />

      <GroupDetailSheet
        groupName={detailGroup}
        onOpenChange={(open) => {
          if (!open) setDetailGroup(null)
        }}
        registry={registry}
        topupGroupRatio={topupGroupRatio}
        userUsableGroups={userUsableGroups}
      />
    </div>
  )
})

type GroupPricingTableProps = {
  groupRatio: string
  userUsableGroups: string
  topupGroupRatio: string
  onChange: (field: string, value: string) => void
  onShowDetail: (name: string) => void
}

function GroupPricingTable({
  groupRatio,
  userUsableGroups,
  topupGroupRatio,
  onChange,
  onShowDetail,
}: GroupPricingTableProps) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<GroupPricingRow[]>(() =>
    buildGroupPricingRows(groupRatio, userUsableGroups, topupGroupRatio)
  )

  useEffect(() => {
    const incomingSignature = sourceGroupPricingSignature(
      groupRatio,
      userUsableGroups,
      topupGroupRatio
    )
    setRows((currentRows) => {
      if (groupPricingSignature(currentRows) === incomingSignature) {
        return currentRows
      }
      return buildGroupPricingRows(
        groupRatio,
        userUsableGroups,
        topupGroupRatio
      )
    })
  }, [groupRatio, userUsableGroups, topupGroupRatio])

  const emitRows = useCallback(
    (nextRows: GroupPricingRow[]) => {
      setRows(nextRows)
      const serialized = serializeGroupPricingRows(nextRows)
      onChange('GroupRatio', serialized.GroupRatio)
      onChange('UserUsableGroups', serialized.UserUsableGroups)
      onChange('TopupGroupRatio', serialized.TopupGroupRatio)
    },
    [onChange]
  )

  const updateRow = useCallback(
    (
      id: string,
      field: Exclude<keyof GroupPricingRow, '_id'>,
      value: string | number | boolean
    ) => {
      emitRows(
        rows.map((row) => (row._id === id ? { ...row, [field]: value } : row))
      )
    },
    [emitRows, rows]
  )

  const addRow = useCallback(() => {
    const existingNames = new Set(rows.map((row) => row.name))
    let index = 1
    let name = `group_${index}`
    while (existingNames.has(name)) {
      index += 1
      name = `group_${index}`
    }
    emitRows([
      ...rows,
      {
        _id: createGroupPricingId(),
        name,
        ratio: '1',
        topupRatio: '',
        selectable: true,
        description: '',
      },
    ])
  }, [emitRows, rows])

  const removeRow = useCallback(
    (id: string) => {
      emitRows(rows.filter((row) => row._id !== id))
    },
    [emitRows, rows]
  )

  const duplicateNames = useMemo(() => {
    const counts = new Map<string, number>()
    for (const row of rows) {
      const name = row.name.trim()
      if (!name) continue
      counts.set(name, (counts.get(name) ?? 0) + 1)
    }
    return [...counts.entries()]
      .filter(([, count]) => count > 1)
      .map(([name]) => name)
  }, [rows])

  return (
    <Card className={sectionCardClassName}>
      <CardHeader className={sectionHeaderClassName}>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
          <div>
            <CardTitle>{t('Pricing groups')}</CardTitle>
            <CardDescription>
              {t(
                'All group names live here. Ratio applies when calls are billed as this group; top-up ratio applies to users whose account is in this group.'
              )}
            </CardDescription>
          </div>
          <Button onClick={addRow} size='sm' className='sm:self-start'>
            <Plus className='mr-2 h-4 w-4' />
            {t('Add group')}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className='space-y-3'>
          <StaticDataTable
            data={rows}
            getRowKey={(row) => row._id}
            emptyClassName='text-muted-foreground h-20 text-sm'
            emptyContent={t('No groups yet. Add a group to get started.')}
            columns={[
              {
                id: 'group',
                header: t('Group name'),
                className: 'min-w-40',
                cell: (row) => (
                  <Input
                    value={row.name}
                    onChange={(event) =>
                      updateRow(row._id, 'name', event.target.value)
                    }
                    aria-invalid={duplicateNames.includes(row.name.trim())}
                  />
                ),
              },
              {
                id: 'ratio',
                header: t('Ratio'),
                className: 'w-28',
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
                id: 'topup-ratio',
                header: t('Top-up ratio'),
                className: 'w-28',
                cell: (row) => (
                  <Input
                    type='number'
                    min={0}
                    step={0.1}
                    value={row.topupRatio}
                    placeholder={t('Not set')}
                    onChange={(event) =>
                      updateRow(row._id, 'topupRatio', event.target.value)
                    }
                  />
                ),
              },
              {
                id: 'selectable',
                header: t('User selectable'),
                className: 'w-28 text-center',
                cell: (row) => (
                  <div className='flex justify-center'>
                    <Checkbox
                      checked={row.selectable}
                      onCheckedChange={(checked) =>
                        updateRow(row._id, 'selectable', checked === true)
                      }
                      aria-label={t('User selectable')}
                    />
                  </div>
                ),
              },
              {
                id: 'description',
                header: t('Description'),
                className: 'min-w-56',
                cell: (row) =>
                  row.selectable ? (
                    <Input
                      value={row.description}
                      placeholder={t('Group description')}
                      onChange={(event) =>
                        updateRow(row._id, 'description', event.target.value)
                      }
                    />
                  ) : (
                    <span className='text-muted-foreground px-3 text-sm'>
                      -
                    </span>
                  ),
              },
              {
                id: 'actions',
                header: t('Actions'),
                className: 'text-right',
                cellClassName: 'text-right',
                cell: (row) => (
                  <div className='flex justify-end gap-1'>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => onShowDetail(row.name.trim())}
                      disabled={!row.name.trim()}
                      aria-label={t('Details')}
                    >
                      <Info className='h-4 w-4' />
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
                ),
              },
            ]}
          />

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
  )
}

type GroupDetailSheetProps = {
  groupName: string | null
  onOpenChange: (open: boolean) => void
  registry: RegistryEntry[]
  topupGroupRatio: string
  userUsableGroups: string
}

function GroupDetailSheet(props: GroupDetailSheetProps) {
  const { t } = useTranslation()
  const name = props.groupName

  const detail = useMemo(() => {
    if (!name) return null

    const entry = props.registry.find((item) => item.name === name)
    const topupMap = parseRatioMap(props.topupGroupRatio)
    const usableMap = parseUsableMap(props.userUsableGroups)

    return {
      ratio: entry?.ratio,
      topupRatio: Object.hasOwn(topupMap, name) ? String(topupMap[name]) : null,
      selectable: Object.hasOwn(usableMap, name),
      description: String(usableMap[name] ?? ''),
    }
  }, [name, props.registry, props.topupGroupRatio, props.userUsableGroups])

  return (
    <Sheet open={name !== null} onOpenChange={props.onOpenChange}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-lg')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {t('Group details')}
            {name ? `: ${name}` : ''}
          </SheetTitle>
          <SheetDescription>
            {t('Everything configured for this group, in one place.')}
          </SheetDescription>
        </SheetHeader>

        {detail && (
          <div className={sideDrawerFormClassName('gap-5')}>
            <section className='space-y-2'>
              <h3 className='text-sm font-semibold'>{t('Overview')}</h3>
              <dl className='space-y-1.5 text-sm'>
                <div className='flex justify-between'>
                  <dt className='text-muted-foreground'>{t('Ratio')}</dt>
                  <dd className='font-medium'>{detail.ratio ?? '-'}</dd>
                </div>
                <div className='flex justify-between'>
                  <dt className='text-muted-foreground'>{t('Top-up ratio')}</dt>
                  <dd className='font-medium'>
                    {detail.topupRatio ?? t('Not set')}
                  </dd>
                </div>
                <div className='flex justify-between'>
                  <dt className='text-muted-foreground'>
                    {t('User selectable')}
                  </dt>
                  <dd className='font-medium'>
                    {detail.selectable ? t('Yes') : t('No')}
                  </dd>
                </div>
                {detail.selectable && detail.description && (
                  <div className='flex justify-between gap-4'>
                    <dt className='text-muted-foreground'>
                      {t('Description')}
                    </dt>
                    <dd className='text-right font-medium'>
                      {detail.description}
                    </dd>
                  </div>
                )}
              </dl>
            </section>
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}
