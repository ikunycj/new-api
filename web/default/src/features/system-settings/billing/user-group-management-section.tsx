/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { MultiSelect, type Option } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { UserGroupTableRow } from './user-group-table-row'
import {
  createUserGroup,
  deleteUserGroup,
  getPricingGroupNames,
  getUserGroupSummaries,
  updateUserGroup,
  type UpdateUserGroupRequest,
  type UserGroupSummary,
} from './user-groups-api'

const USER_GROUP_QUERY_KEY = ['admin', 'user-groups'] as const

type UpdateUserGroupVariables = {
  group: UserGroupSummary
  request: UpdateUserGroupRequest
  closeDialog: boolean
}

export function UserGroupManagementSection() {
  const queryClient = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<UserGroupSummary | null>(
    null
  )
  const [editTarget, setEditTarget] = useState<UserGroupSummary | null>(null)
  const [editName, setEditName] = useState('')
  const [topupRatio, setTopupRatio] = useState('1')
  const [pricingGroups, setPricingGroups] = useState<string[]>([])
  const [allPricingGroups, setAllPricingGroups] = useState(true)

  const openEditDialog = (group: UserGroupSummary) => {
    const selectedGroups = group.pricing_groups ?? []
    const isAll =
      group.pricing_groups_all === true || selectedGroups.includes('*')
    setEditName(group.name)
    setTopupRatio(String(group.topup_ratio ?? 1))
    setAllPricingGroups(isAll)
    setPricingGroups(isAll ? [] : selectedGroups)
    setEditTarget(group)
  }

  const groupsQuery = useQuery({
    queryKey: USER_GROUP_QUERY_KEY,
    queryFn: getUserGroupSummaries,
  })

  const pricingGroupsQuery = useQuery({
    queryKey: ['pricing-groups'],
    queryFn: getPricingGroupNames,
  })

  const pricingGroupOptions = useMemo<Option[]>(
    () =>
      (pricingGroupsQuery.data ?? []).map((group) => ({
        label: group,
        value: group,
      })),
    [pricingGroupsQuery.data]
  )

  const createMutation = useMutation({
    mutationFn: createUserGroup,
    onSuccess: async () => {
      setName('')
      setCreateOpen(false)
      toast.success('用户分组已创建')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: USER_GROUP_QUERY_KEY }),
        queryClient.invalidateQueries({ queryKey: ['groups'] }),
      ])
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '创建用户分组失败')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteUserGroup,
    onSuccess: async () => {
      setDeleteTarget(null)
      toast.success('用户分组已删除')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: USER_GROUP_QUERY_KEY }),
        queryClient.invalidateQueries({ queryKey: ['groups'] }),
      ])
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '删除用户分组失败')
    },
  })

  const updateMutation = useMutation({
    mutationFn: (variables: UpdateUserGroupVariables) =>
      updateUserGroup(variables.group.name, variables.request),
    onSuccess: async (updatedGroup, variables) => {
      queryClient.setQueryData<UserGroupSummary[]>(
        USER_GROUP_QUERY_KEY,
        (groups) =>
          groups?.map((group) =>
            group.id === updatedGroup.id
              ? {
                  ...updatedGroup,
                  active_today: group.active_today,
                  active_month: group.active_month,
                }
              : group
          )
      )
      if (variables.closeDialog) setEditTarget(null)
      toast.success('用户分组已更新')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: USER_GROUP_QUERY_KEY }),
        queryClient.invalidateQueries({ queryKey: ['groups'] }),
        queryClient.invalidateQueries({ queryKey: ['users'] }),
      ])
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '更新用户分组失败')
    },
  })

  const submitCreate = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedName = name.trim()
    if (!trimmedName) {
      toast.error('请输入用户分组名')
      return
    }
    createMutation.mutate(trimmedName)
  }

  const submitEdit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!editTarget) return

    const trimmedName = editName.trim()
    if (!trimmedName) {
      toast.error('请输入用户分组名')
      return
    }
    const parsedRatio = Number(topupRatio)
    if (
      topupRatio.trim() === '' ||
      !Number.isFinite(parsedRatio) ||
      parsedRatio < 0
    ) {
      toast.error('请输入有效的充值倍率')
      return
    }
    if (!allPricingGroups && pricingGroups.length === 0) {
      toast.error('请选择至少一个定价分组，或启用“全部”')
      return
    }

    updateMutation.mutate({
      group: editTarget,
      request: {
        name: editTarget.name === 'default' ? editTarget.name : trimmedName,
        topup_ratio: parsedRatio,
        pricing_groups: allPricingGroups ? ['*'] : pricingGroups,
        pricing_groups_all: allPricingGroups,
      },
      closeDialog: true,
    })
  }

  let groupsContent: ReactNode
  if (groupsQuery.isLoading) {
    groupsContent = (
      <div className='flex min-h-40 items-center justify-center'>
        <Spinner />
      </div>
    )
  } else if (groupsQuery.isError) {
    groupsContent = (
      <div className='flex min-h-40 flex-col items-center justify-center gap-3 p-6 text-center'>
        <p className='text-destructive text-sm'>用户分组加载失败</p>
        <Button
          variant='outline'
          size='sm'
          onClick={() => void groupsQuery.refetch()}
        >
          重试
        </Button>
      </div>
    )
  } else if (groupsQuery.data && groupsQuery.data.length > 0) {
    groupsContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>名称</TableHead>
            <TableHead>用户数</TableHead>
            <TableHead>充值倍率</TableHead>
            <TableHead>定价分组</TableHead>
            <TableHead>创建时间</TableHead>
            <TableHead>修改时间</TableHead>
            <TableHead className='text-right'>操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {groupsQuery.data.map((group) => (
            <UserGroupTableRow
              key={JSON.stringify([
                group.id,
                group.name,
                group.topup_ratio,
                group.pricing_groups_all === true,
                group.pricing_groups ?? [],
              ])}
              group={group}
              pricingGroupOptions={pricingGroupOptions}
              pricingGroupsUnavailable={
                pricingGroupsQuery.isLoading || pricingGroupsQuery.isError
              }
              disabled={updateMutation.isPending}
              saving={
                updateMutation.isPending &&
                updateMutation.variables?.group.id === group.id
              }
              onUpdate={(request) =>
                updateMutation.mutateAsync({
                  group,
                  request,
                  closeDialog: false,
                })
              }
              onEdit={() => openEditDialog(group)}
              onDelete={() => setDeleteTarget(group)}
            />
          ))}
        </TableBody>
      </Table>
    )
  } else {
    groupsContent = (
      <Empty className='min-h-40 rounded-none border-0'>
        <EmptyHeader>
          <EmptyTitle>暂无用户分组</EmptyTitle>
          <EmptyDescription>创建一个用户分组后会显示在这里。</EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <section className='bg-card rounded-xl border'>
      <div className='flex justify-end border-b p-4'>
        <Button size='sm' onClick={() => setCreateOpen(true)}>
          <Plus />
          新增用户分组
        </Button>
      </div>

      {groupsContent}

      <Dialog
        open={createOpen}
        onOpenChange={(open) => {
          if (createMutation.isPending) return
          setCreateOpen(open)
          if (!open) setName('')
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新增用户分组</DialogTitle>
            <DialogDescription>请输入新的用户分组名。</DialogDescription>
          </DialogHeader>
          <form id='create-user-group-form' onSubmit={submitCreate}>
            <Input
              autoFocus
              value={name}
              maxLength={64}
              placeholder='例如：企业用户'
              onChange={(event) => setName(event.target.value)}
              disabled={createMutation.isPending}
            />
          </form>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => setCreateOpen(false)}
              disabled={createMutation.isPending}
            >
              取消
            </Button>
            <Button
              type='submit'
              form='create-user-group-form'
              disabled={createMutation.isPending}
            >
              {createMutation.isPending && <Spinner />}
              创建
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={editTarget !== null}
        onOpenChange={(open) => {
          if (updateMutation.isPending) return
          if (!open) setEditTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑用户分组</DialogTitle>
            <DialogDescription>
              {editTarget ? `修改“${editTarget.name}”的分组配置。` : ''}
            </DialogDescription>
          </DialogHeader>
          <form id='edit-user-group-form' onSubmit={submitEdit}>
            <FieldGroup>
              <Field
                data-disabled={
                  updateMutation.isPending || editTarget?.name === 'default'
                }
              >
                <FieldLabel htmlFor='user-group-name'>名称</FieldLabel>
                <Input
                  id='user-group-name'
                  value={editName}
                  maxLength={64}
                  title={
                    editTarget?.name === 'default'
                      ? 'default 分组名称不可修改'
                      : undefined
                  }
                  disabled={
                    updateMutation.isPending || editTarget?.name === 'default'
                  }
                  onChange={(event) => setEditName(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='user-group-topup-ratio'>
                  充值倍率
                </FieldLabel>
                <Input
                  id='user-group-topup-ratio'
                  type='number'
                  min={0}
                  step={0.1}
                  value={topupRatio}
                  onChange={(event) => setTopupRatio(event.target.value)}
                  disabled={updateMutation.isPending}
                />
              </Field>
              <Field>
                <FieldLabel>定价分组</FieldLabel>
                <Field orientation='horizontal'>
                  <Checkbox
                    id='all-pricing-groups'
                    checked={allPricingGroups}
                    onCheckedChange={(checked) => {
                      const checkedAll = checked === true
                      setAllPricingGroups(checkedAll)
                      if (checkedAll) setPricingGroups([])
                    }}
                    disabled={updateMutation.isPending}
                  />
                  <FieldLabel
                    htmlFor='all-pricing-groups'
                    className='font-normal'
                  >
                    全部
                  </FieldLabel>
                </Field>
                <MultiSelect
                  id='user-group-pricing-groups'
                  options={pricingGroupOptions}
                  selected={pricingGroups}
                  onChange={(groups) => {
                    setPricingGroups(groups)
                    if (groups.length > 0) setAllPricingGroups(false)
                  }}
                  placeholder={
                    pricingGroupsQuery.isLoading
                      ? '加载定价分组...'
                      : '选择定价分组'
                  }
                  emptyText='暂无定价分组'
                  disabled={
                    pricingGroupsQuery.isError || updateMutation.isPending
                  }
                  maxVisibleChips={4}
                />
                {pricingGroupsQuery.isError && (
                  <FieldError>定价分组加载失败</FieldError>
                )}
              </Field>
            </FieldGroup>
          </form>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => setEditTarget(null)}
              disabled={updateMutation.isPending}
            >
              取消
            </Button>
            <Button
              type='submit'
              form='edit-user-group-form'
              disabled={updateMutation.isPending || pricingGroupsQuery.isError}
            >
              {updateMutation.isPending && <Spinner />}
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open && !deleteMutation.isPending) setDeleteTarget(null)
        }}
        title='删除用户分组'
        desc={`确定删除用户分组“${deleteTarget?.name ?? ''}”吗？分组中仍有用户时无法删除。`}
        confirmText='删除'
        destructive
        handleConfirm={() => {
          if (deleteTarget) deleteMutation.mutate(deleteTarget.name)
        }}
        isLoading={deleteMutation.isPending}
      />
    </section>
  )
}
