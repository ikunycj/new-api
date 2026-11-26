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
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
} from '@/components/ui/input-group'
import { Spinner } from '@/components/ui/spinner'

import { getAffiliateSubAgents, searchAffiliateUserOverrides } from '../api'
import type { AffiliateSubAgent, AffiliateUserOverrideView } from '../types'

export type AffiliateSubAgentDraft = Omit<AffiliateSubAgent, 'rate_bps'> & {
  ratePercent: string | null
}

function formatSubAgentRatePercent(rateBps: number): string {
  const digits = String(rateBps).padStart(3, '0')
  const whole = digits.slice(0, -2)
  const fraction = digits.slice(-2).replace(/0+$/, '')
  return fraction ? `${whole}.${fraction}` : whole
}

export function AffiliateSubAgentsDialog(props: {
  parent: AffiliateUserOverrideView
  changes: Record<number, AffiliateSubAgentDraft>
  otherStagedChildIDs: Set<number>
  onChange: (userId: number, draft: AffiliateSubAgentDraft | null) => void
  onClose: () => void
}) {
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const bindings = useQuery({
    queryKey: ['admin', 'affiliate', 'sub-agents', props.parent.user_id],
    queryFn: () => getAffiliateSubAgents(props.parent.user_id),
    enabled: props.parent.override !== null,
  })
  const search = useQuery({
    queryKey: ['admin', 'affiliate', 'sub-agent-search', keyword],
    queryFn: async () => {
      const response = await searchAffiliateUserOverrides({
        keyword,
        page: 1,
        pageSize: 20,
      })
      if (!response.success) {
        throw new Error(response.message || '用户搜索失败')
      }
      return response.data?.items ?? []
    },
    enabled: keyword !== '',
  })

  const saved = props.parent.override ? (bindings.data ?? []) : []
  const savedByID = new Map(saved.map((item) => [item.user_id, item]))
  const rows = [
    ...Object.values(props.changes).filter(
      (item) => !savedByID.has(item.user_id)
    ),
    ...saved.map(
      (item) =>
        props.changes[item.user_id] ?? {
          user_id: item.user_id,
          username: item.username,
          email: item.email,
          ratePercent: formatSubAgentRatePercent(item.rate_bps),
        }
    ),
  ]
  const loading = bindings.isLoading
  const failed = bindings.isError

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
    >
      <DialogContent className='flex max-h-[90dvh] flex-col sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>管理子代理 · {props.parent.username}</DialogTitle>
          <DialogDescription>
            同步返现按受邀用户本笔实付金额计算。子代理返现比例和同步返现比例均为
            5% 时，受邀用户充值 ¥100，子代理和主代理各获得
            ¥5。变更从页面保存后的新充值生效。
          </DialogDescription>
        </DialogHeader>
        <div className='min-h-0 space-y-4 overflow-y-auto'>
          <form
            onSubmit={(event) => {
              event.preventDefault()
              const nextKeyword = searchInput.trim()
              if (nextKeyword && nextKeyword === keyword) {
                void search.refetch()
              } else {
                setKeyword(nextKeyword)
              }
            }}
          >
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor='affiliate-sub-agent-search'>
                  添加子代理
                </FieldLabel>
                <InputGroup>
                  <InputGroupInput
                    id='affiliate-sub-agent-search'
                    value={searchInput}
                    onChange={(event) => setSearchInput(event.target.value)}
                    placeholder='用户 ID、名称或邮箱'
                    disabled={loading || failed}
                  />
                  <InputGroupAddon align='inline-end'>
                    <InputGroupButton
                      type='submit'
                      aria-label='搜索用户'
                      disabled={loading || failed}
                    >
                      <Search />
                    </InputGroupButton>
                  </InputGroupAddon>
                </InputGroup>
              </Field>
            </FieldGroup>
          </form>
          {search.isFetching ? <Spinner className='mx-auto' /> : null}
          {search.isError ? (
            <p role='alert' className='text-destructive'>
              搜索失败，请重试。
            </p>
          ) : null}
          {keyword && !search.isFetching && search.data?.length === 0 ? (
            <p className='text-muted-foreground'>未找到用户。</p>
          ) : null}
          {search.data?.length ? (
            <div className='max-h-48 divide-y overflow-y-auto rounded-md border'>
              {search.data.map((user) => {
                let unavailableReason = ''
                if (user.user_id === props.parent.user_id) {
                  unavailableReason = '当前代理'
                } else if (
                  savedByID.has(user.user_id) ||
                  props.changes[user.user_id]
                ) {
                  unavailableReason = '已选择'
                } else if (props.otherStagedChildIDs.has(user.user_id)) {
                  unavailableReason = '其他代理下有待保存变更'
                }
                return (
                  <div
                    key={user.user_id}
                    className='flex items-center justify-between gap-3 px-3 py-2'
                  >
                    <div className='min-w-0'>
                      <div className='truncate font-medium'>
                        {user.username} · {user.user_id}
                      </div>
                      {user.email ? (
                        <div className='text-muted-foreground truncate text-xs'>
                          {user.email}
                        </div>
                      ) : null}
                    </div>
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      disabled={loading || failed || Boolean(unavailableReason)}
                      onClick={() => {
                        props.onChange(user.user_id, {
                          user_id: user.user_id,
                          username: user.username,
                          email: user.email,
                          ratePercent: '',
                        })
                        setSearchInput('')
                        setKeyword('')
                      }}
                    >
                      {unavailableReason || '添加'}
                    </Button>
                  </div>
                )
              })}
            </div>
          ) : null}
          <section aria-label='当前子代理'>
            <h3 className='mb-2 text-sm font-medium'>当前子代理</h3>
            {loading ? <Spinner className='mx-auto' /> : null}
            {failed ? (
              <div className='flex items-center justify-between gap-2'>
                <p role='alert' className='text-destructive'>
                  加载子代理失败。
                </p>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  onClick={() => void bindings.refetch()}
                >
                  重试
                </Button>
              </div>
            ) : null}
            {!loading && !failed && rows.length === 0 ? (
              <Empty className='min-h-20 border'>
                <EmptyHeader>
                  <EmptyTitle>暂无子代理</EmptyTitle>
                </EmptyHeader>
              </Empty>
            ) : null}
            {rows.length > 0 ? (
              <div className='divide-y rounded-md border'>
                {rows.map((row) => {
                  const existing = savedByID.get(row.user_id)
                  const staged = props.changes[row.user_id]
                  const ratePercent = row.ratePercent
                  const removed = ratePercent === null
                  let actionLabel = existing ? '解除绑定' : '移除'
                  if (removed) actionLabel = '撤销'
                  const fieldId = `affiliate-sub-agent-rate-${row.user_id}`
                  return (
                    <div
                      key={row.user_id}
                      className='grid gap-2 px-3 py-2 sm:grid-cols-[1fr_8rem_auto] sm:items-start'
                    >
                      <div className='min-w-0 self-center'>
                        <div className='truncate font-medium'>
                          {row.username} · {row.user_id}
                        </div>
                        {row.email ? (
                          <div className='text-muted-foreground truncate text-xs'>
                            {row.email}
                          </div>
                        ) : null}
                        {staged ? (
                          <div className='text-muted-foreground text-xs'>
                            {removed ? '待解除绑定' : '待保存'}
                          </div>
                        ) : null}
                      </div>
                      {!removed ? (
                        <Field>
                          <FieldLabel htmlFor={fieldId} className='text-xs'>
                            同步返现比例
                          </FieldLabel>
                          <InputGroup>
                            <InputGroupInput
                              id={fieldId}
                              type='text'
                              inputMode='decimal'
                              placeholder='例如 5'
                              value={ratePercent}
                              disabled={failed}
                              onChange={(event) => {
                                const updatedRate = event.target.value
                                const unchanged =
                                  existing &&
                                  updatedRate ===
                                    formatSubAgentRatePercent(existing.rate_bps)
                                props.onChange(
                                  row.user_id,
                                  unchanged
                                    ? null
                                    : { ...row, ratePercent: updatedRate }
                                )
                              }}
                            />
                            <InputGroupAddon align='inline-end'>
                              <InputGroupText>%</InputGroupText>
                            </InputGroupAddon>
                          </InputGroup>
                        </Field>
                      ) : (
                        <div />
                      )}
                      <Button
                        type='button'
                        size='sm'
                        variant='ghost'
                        className='self-end justify-self-start'
                        disabled={failed}
                        aria-label={`${actionLabel} ${row.username}`}
                        onClick={() =>
                          props.onChange(
                            row.user_id,
                            existing && !removed
                              ? { ...row, ratePercent: null }
                              : null
                          )
                        }
                      >
                        {actionLabel}
                      </Button>
                    </div>
                  )
                })}
              </div>
            ) : null}
          </section>
        </div>
        <DialogFooter>
          <Button type='button' onClick={props.onClose}>
            完成
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
