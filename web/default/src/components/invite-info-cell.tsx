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
import { ChevronLeft, ChevronRight, Search } from 'lucide-react'
import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  InputGroup,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'

export type InviteInfo = {
  invited_count?: number
  income_cents?: number
  inviter_user_id?: number
  inviter_username?: string
  inviter_email?: string
}

type InviteActivity = {
  id: number
  invitee_user_id: number
  username: string
  email: string
  operation_type: 'invitation' | 'top_up'
  topup_amount_cents: number
  cashback_amount_cents: number
  cumulative_cashback_cents: number
  occurred_at: number
}

type InviteActivityResponse = {
  success: boolean
  message?: string
  data?: {
    items: InviteActivity[]
    total: number
  }
}

const PAGE_SIZE = 10

function formatCNY(cents: number) {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'CNY',
  }).format(cents / 100)
}

function inviterLabel(info: InviteInfo) {
  if (!info.inviter_user_id) return ''
  const identity = info.inviter_email || info.inviter_username || ''
  return identity
    ? `${identity} ${info.inviter_user_id}`
    : String(info.inviter_user_id)
}

function operationLabel(
  t: (key: string) => string,
  operationType: InviteActivity['operation_type']
) {
  return operationType === 'invitation' ? t('Invitation') : t('Top-up')
}

function InviteActivityDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number
}) {
  const { t } = useTranslation()
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [operationType, setOperationType] = useState('')
  const [page, setPage] = useState(1)

  useEffect(() => {
    if (!props.open) return
    setKeywordInput('')
    setKeyword('')
    setOperationType('')
    setPage(1)
  }, [props.open, props.userId])

  const query = useQuery({
    queryKey: [
      'admin',
      'affiliate',
      'invite-activities',
      props.userId,
      keyword,
      operationType,
      page,
    ],
    queryFn: async () => {
      const response = await api.get<InviteActivityResponse>(
        `/api/affiliate/admin/users/${props.userId}/invite-activities`,
        {
          params: {
            keyword,
            operation_type: operationType,
            p: page,
            page_size: PAGE_SIZE,
          },
        }
      )
      return response.data
    },
    enabled: props.open && props.userId > 0,
    select: (response) => response.data ?? { items: [], total: 0 },
  })
  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  let activityBody: ReactNode
  if (query.isLoading) {
    activityBody = (
      <TableRow>
        <TableCell colSpan={7} className='h-32 text-center'>
          <Spinner className='mx-auto' />
        </TableCell>
      </TableRow>
    )
  } else if (query.isError) {
    activityBody = (
      <TableRow>
        <TableCell colSpan={7} className='h-32 text-center'>
          {t('Loading failed')}
        </TableCell>
      </TableRow>
    )
  } else if (items.length === 0) {
    activityBody = (
      <TableRow>
        <TableCell colSpan={7} className='h-32 text-center'>
          {t('No referral records')}
        </TableCell>
      </TableRow>
    )
  } else {
    activityBody = items.map((item) => (
      <TableRow key={`${item.operation_type}-${item.id}`}>
        <TableCell>{item.invitee_user_id}</TableCell>
        <TableCell>{item.username || '-'}</TableCell>
        <TableCell>{item.email || '-'}</TableCell>
        <TableCell>{operationLabel(t, item.operation_type)}</TableCell>
        <TableCell>{formatCNY(item.topup_amount_cents)}</TableCell>
        <TableCell>{formatCNY(item.cashback_amount_cents)}</TableCell>
        <TableCell>{formatCNY(item.cumulative_cashback_cents)}</TableCell>
      </TableRow>
    ))
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='flex max-h-[90vh] max-w-[calc(100%-2rem)] flex-col overflow-hidden sm:max-w-6xl'>
        <DialogHeader>
          <DialogTitle>{t('Invitation details')}</DialogTitle>
          <DialogDescription>
            {t('View invitation and successful top-up records for this user.')}
          </DialogDescription>
        </DialogHeader>
        <div className='flex min-h-0 flex-col gap-4'>
          <form className='flex flex-wrap items-center gap-2' onSubmit={search}>
            <InputGroup className='min-w-64 flex-1'>
              <InputGroupInput
                value={keywordInput}
                placeholder={t('Search by user ID, username, or email')}
                onChange={(event) => setKeywordInput(event.target.value)}
              />
              <InputGroupButton type='submit' aria-label={t('Search')}>
                <Search />
              </InputGroupButton>
            </InputGroup>
            <NativeSelect
              value={operationType}
              aria-label={t('Operation type')}
              onChange={(event) => {
                setOperationType(event.target.value)
                setPage(1)
              }}
            >
              <NativeSelectOption value=''>
                {t('All operations')}
              </NativeSelectOption>
              <NativeSelectOption value='invitation'>
                {t('Invitation')}
              </NativeSelectOption>
              <NativeSelectOption value='top_up'>
                {t('Top-up')}
              </NativeSelectOption>
            </NativeSelect>
          </form>
          <div className='max-h-[65vh] min-h-0 overflow-auto rounded-lg border'>
            <Table className='min-w-[900px]'>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('User ID')}</TableHead>
                  <TableHead>{t('Username')}</TableHead>
                  <TableHead>{t('Email')}</TableHead>
                  <TableHead>{t('Operation type')}</TableHead>
                  <TableHead>{t('Top-up amount')}</TableHead>
                  <TableHead>{t('Cashback amount')}</TableHead>
                  <TableHead>{t('Cumulative cashback')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>{activityBody}</TableBody>
            </Table>
          </div>
          {total > 0 ? (
            <div className='flex items-center justify-between gap-3'>
              <span className='text-muted-foreground text-xs'>
                {t('Showing')} {(page - 1) * PAGE_SIZE + 1}-
                {Math.min(page * PAGE_SIZE, total)} {t('of')} {total}
              </span>
              <div className='flex items-center gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  className='size-8 p-0'
                  disabled={page <= 1}
                  onClick={() => setPage((current) => current - 1)}
                  aria-label={t('Previous page')}
                >
                  <ChevronLeft />
                </Button>
                <span className='text-muted-foreground min-w-12 text-center text-sm tabular-nums'>
                  {page} / {totalPages}
                </span>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  className='size-8 p-0'
                  disabled={page >= totalPages}
                  onClick={() => setPage((current) => current + 1)}
                  aria-label={t('Next page')}
                >
                  <ChevronRight />
                </Button>
              </div>
            </div>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function InviteInfoCell(props: {
  userId: number
  info?: InviteInfo
  className?: string
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const info = props.info ?? {}
  const inviter = inviterLabel(info)

  return (
    <>
      <div
        className={cn('max-w-80 min-w-64 space-y-1 text-sm', props.className)}
      >
        <div className='font-medium'>
          {t('Invited: {{count}}', { count: info.invited_count ?? 0 })}{' '}
          {t('Revenue: {{amount}}', {
            amount: formatCNY(info.income_cents ?? 0),
          })}
        </div>
        <div className='text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1'>
          <span>
            {t('Inviter: {{name}}', {
              name: inviter || t('No Inviter'),
            })}
          </span>
          <Button
            type='button'
            variant='link'
            size='sm'
            className='h-auto p-0'
            onClick={() => setOpen(true)}
          >
            {t('Details')}
          </Button>
        </div>
      </div>
      <InviteActivityDialog
        open={open}
        onOpenChange={setOpen}
        userId={props.userId}
      />
    </>
  )
}
