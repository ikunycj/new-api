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
/// <reference lib="es2023.intl" />

import {
  ArrowRightLeft,
  ChevronLeft,
  ChevronRight,
  ReceiptText,
  Search,
  X,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  InputGroup,
  InputGroupAddon,
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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatQuotaAsCNY, formatTimestampToDate } from '@/lib/format'

import { AFFILIATE_TOP_UP_PAGE_SIZE } from '../../constants'
import {
  formatAffiliateAccountCashback,
  formatAffiliateCashback,
} from '../../lib/affiliate'
import type {
  AffiliateBalanceTransferResponse,
  AffiliateInviteeTopUp,
  AffiliateInviteeTopUpsQuery,
  AffiliateSummary,
  AffiliateTopUpStatus,
} from '../../types'

interface AffiliateCenterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  customerService?: string
  summary?: AffiliateSummary
  topUps: AffiliateInviteeTopUp[]
  topUpsTotal: number
  topUpQuery: AffiliateInviteeTopUpsQuery
  onTopUpQueryChange: (query: AffiliateInviteeTopUpsQuery) => void
  loading: boolean
  transferring: boolean
  onTransfer: (request: {
    amount_quota?: number
    cash_cents?: number
    request_key: string
  }) => Promise<AffiliateBalanceTransferResponse>
}

interface ListPaginationProps {
  page: number
  total: number
  onPageChange: (page: number) => void
}

function ListPagination(props: ListPaginationProps) {
  const { t } = useTranslation()
  if (props.total <= 0) return null

  const totalPages = Math.max(
    1,
    Math.ceil(props.total / AFFILIATE_TOP_UP_PAGE_SIZE)
  )
  return (
    <div className='border-border mt-4 flex items-center justify-between gap-3 border-t pt-3'>
      <div className='text-muted-foreground text-xs'>
        {t('Showing')} {(props.page - 1) * AFFILIATE_TOP_UP_PAGE_SIZE + 1}-
        {Math.min(props.page * AFFILIATE_TOP_UP_PAGE_SIZE, props.total)}{' '}
        {t('of')} {props.total}
      </div>
      <div className='flex items-center gap-2'>
        <Button
          type='button'
          variant='outline'
          size='sm'
          className='size-8 p-0'
          disabled={props.page <= 1}
          onClick={() => props.onPageChange(props.page - 1)}
          aria-label={t('Previous page')}
          title={t('Previous page')}
        >
          <ChevronLeft />
        </Button>
        <span className='text-muted-foreground min-w-12 text-center text-sm tabular-nums'>
          {props.page} / {totalPages}
        </span>
        <Button
          type='button'
          variant='outline'
          size='sm'
          className='size-8 p-0'
          disabled={props.page >= totalPages}
          onClick={() => props.onPageChange(props.page + 1)}
          aria-label={t('Next page')}
          title={t('Next page')}
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  )
}

function getStatusLabel(
  t: (key: string) => string,
  status: AffiliateTopUpStatus
) {
  const labels: Record<AffiliateTopUpStatus, string> = {
    unqualified: t('Not qualified'),
    pending: t('Frozen'),
    available: t('Awaiting transfer'),
    transferred: t('Transferred'),
    adjusted: t('Adjusted'),
  }
  return labels[status]
}

function getStatusClass(status: AffiliateTopUpStatus) {
  if (status === 'available') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700'
  }
  if (status === 'pending') {
    return 'border-amber-200 bg-amber-50 text-amber-700'
  }
  if (status === 'transferred') {
    return 'border-border bg-muted text-muted-foreground'
  }
  return 'border-border bg-background text-muted-foreground'
}

function getAvailableReward(topUp: AffiliateInviteeTopUp) {
  if (topUp.cash_reward_id > 0) return topUp.available_reward_cents
  return topUp.available_reward_quota ?? topUp.reward_quota
}

function formatReward(topUp: AffiliateInviteeTopUp, amount: number) {
  return topUp.cash_reward_id > 0
    ? formatAffiliateCashback(amount)
    : formatQuotaAsCNY(amount)
}

function getTopUpRowKey(topUp: AffiliateInviteeTopUp): string {
  if (topUp.source_agent_user_id) {
    return `sub-agent:${topUp.id}:${topUp.reward_id}:${topUp.cash_reward_id}`
  }
  return `direct:${topUp.id}`
}

function formatCashbackRule(
  t: (key: string, options?: Record<string, unknown>) => string,
  topUp: AffiliateInviteeTopUp
) {
  if (topUp.reward_mode === 'fixed') {
    return t('Fixed {{amount}}', {
      amount: formatQuotaAsCNY(topUp.fixed_reward_quota),
    })
  }
  return `${new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 2,
  }).format(topUp.reward_rate_bps / 100)}%`
}

function CashbackRule(props: { topUp: AffiliateInviteeTopUp }) {
  const { t } = useTranslation()
  const topUp = props.topUp
  if (!topUp.source_agent_user_id) {
    return formatCashbackRule(t, topUp)
  }
  const rate = new Intl.NumberFormat(undefined, {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format(`${topUp.reward_rate_bps}e-4` as `${number}`)
  return (
    <>
      {topUp.legacy_sub_agent_rate ? (
        <Tooltip>
          <TooltipTrigger
            type='button'
            className='cursor-help underline decoration-dotted underline-offset-4'
          >
            {rate}
          </TooltipTrigger>
          <TooltipContent className='whitespace-normal'>
            {t('This rate was applied to the sub-agent’s cashback.')}
          </TooltipContent>
        </Tooltip>
      ) : (
        rate
      )}
      <div className='text-muted-foreground !text-xs'>
        {topUp.source_agent_username || `#${topUp.source_agent_user_id}`}{' '}
        {topUp.source_agent_email}
      </div>
    </>
  )
}

function HiddenTopUpFallback(props: {
  availableQuota: number
  availableCashCents: number
  transferring: boolean
  onTransfer: () => void
}) {
  const { t } = useTranslation()
  return (
    <Empty className='min-h-48'>
      <EmptyHeader>
        <EmptyMedia variant='icon'>
          <ReceiptText />
        </EmptyMedia>
        <EmptyTitle>{t('Invitee top-up records are hidden')}</EmptyTitle>
        <EmptyDescription>
          {t('The administrator has disabled invitee top-up visibility.')}
        </EmptyDescription>
      </EmptyHeader>
      <Button
        type='button'
        disabled={
          (props.availableCashCents <= 0 && props.availableQuota <= 0) ||
          props.transferring
        }
        onClick={props.onTransfer}
      >
        {props.transferring ? (
          <Spinner data-icon='inline-start' />
        ) : (
          <ArrowRightLeft data-icon='inline-start' />
        )}
        {t('Transfer available cashback')}
      </Button>
    </Empty>
  )
}

export function AffiliateCenterDialog(props: AffiliateCenterDialogProps) {
  const { t } = useTranslation()
  const [searchInput, setSearchInput] = useState(props.topUpQuery.keyword ?? '')
  const account = props.summary?.account
  const cashAccount = props.summary?.cash_account
  const rule = props.summary?.rule
  const queryKeyword = props.topUpQuery.keyword ?? ''
  const queryStatus = props.topUpQuery.status
  const queryStartAt = props.topUpQuery.startAt
  const queryEndAt = props.topUpQuery.endAt
  const onTopUpQueryChange = props.onTopUpQueryChange

  useEffect(() => {
    const keyword = searchInput.trim()
    if (keyword === queryKeyword) return
    const timeout = window.setTimeout(() => {
      onTopUpQueryChange({
        page: 1,
        keyword: keyword || undefined,
        status: queryStatus,
        startAt: queryStartAt,
        endAt: queryEndAt,
      })
    }, 300)
    return () => window.clearTimeout(timeout)
  }, [
    onTopUpQueryChange,
    queryEndAt,
    queryKeyword,
    queryStartAt,
    queryStatus,
    searchInput,
  ])

  async function transferAllAvailable() {
    const availableCashCents = cashAccount?.available_cents ?? 0
    if (availableCashCents > 0) {
      await props.onTransfer({
        cash_cents: availableCashCents,
        request_key: crypto.randomUUID(),
      })
      return
    }
    const availableQuota = account?.available_quota ?? 0
    if (availableQuota <= 0) return
    await props.onTransfer({
      amount_quota: availableQuota,
      request_key: crypto.randomUUID(),
    })
  }

  function selectStatus(status?: AffiliateTopUpStatus) {
    props.onTopUpQueryChange({
      ...props.topUpQuery,
      page: 1,
      keyword: searchInput.trim() || undefined,
      status,
    })
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[90vh] min-w-0 overflow-x-hidden overflow-y-auto sm:max-w-6xl'>
        <DialogHeader>
          <DialogTitle>{t('Referral Cashback')}</DialogTitle>
          <DialogDescription className='break-words whitespace-pre-wrap'>
            {t(
              'Eligible cashback is transferred to your balance automatically.'
            )}{' '}
            {props.customerService
              ? t('To withdraw, contact the administrator: {{contact}}', {
                  contact: props.customerService,
                })
              : t('To withdraw, contact the administrator.')}
          </DialogDescription>
        </DialogHeader>

        <div className='grid grid-cols-2 gap-3 sm:grid-cols-3'>
          {[
            [
              t('Frozen'),
              formatAffiliateAccountCashback(props.summary, 'pending'),
            ],
            [
              t('Transferred'),
              formatAffiliateAccountCashback(props.summary, 'transferred'),
            ],
            [t('Invites'), String(props.summary?.referral_count ?? 0)],
          ].map(([label, value]) => (
            <div key={String(label)} className='border-border border-l-2 pl-3'>
              <div className='text-muted-foreground text-xs'>{label}</div>
              <div className='mt-1 font-semibold tabular-nums'>{value}</div>
            </div>
          ))}
        </div>

        {rule?.show_invitee_topups ? (
          <div className='min-w-0 space-y-4 pt-2'>
            <div className='flex flex-wrap items-center gap-2'>
              <InputGroup className='min-w-48 flex-1 lg:max-w-80'>
                <InputGroupAddon align='inline-start'>
                  <Search aria-hidden='true' />
                </InputGroupAddon>
                <InputGroupInput
                  aria-label={t('Search sub-agent, email, or recharge time')}
                  value={searchInput}
                  onChange={(event) => setSearchInput(event.target.value)}
                  placeholder={t('Search sub-agent, email, or recharge time')}
                />
                {searchInput ? (
                  <InputGroupAddon align='inline-end'>
                    <InputGroupButton
                      type='button'
                      aria-label={t('Clear search')}
                      title={t('Clear search')}
                      onClick={() => setSearchInput('')}
                    >
                      <X />
                    </InputGroupButton>
                  </InputGroupAddon>
                ) : null}
              </InputGroup>
              <div className='flex shrink-0 items-center gap-2'>
                <label
                  htmlFor='affiliate-top-up-status'
                  className='text-muted-foreground text-xs sm:text-sm'
                >
                  {t('Select status')}
                </label>
                <NativeSelect
                  id='affiliate-top-up-status'
                  aria-label={t('Select status')}
                  value={props.topUpQuery.status ?? ''}
                  onChange={(event) =>
                    selectStatus(
                      (event.target.value || undefined) as
                        | AffiliateTopUpStatus
                        | undefined
                    )
                  }
                  className='w-40 shrink-0'
                >
                  <NativeSelectOption value=''>{t('All')}</NativeSelectOption>
                  <NativeSelectOption value='pending'>
                    {t('Frozen')}
                  </NativeSelectOption>
                  <NativeSelectOption value='available'>
                    {t('Awaiting transfer')}
                  </NativeSelectOption>
                  <NativeSelectOption value='transferred'>
                    {t('Transferred')}
                  </NativeSelectOption>
                </NativeSelect>
              </div>
            </div>

            <div className='min-w-0 overflow-x-auto'>
              <Table className='min-w-[800px]'>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Invitee (privacy protected)')}</TableHead>
                    <TableHead>{t('Top-up time')}</TableHead>
                    <TableHead>{t('Paid amount')}</TableHead>
                    <TableHead>{t('Cashback rule')}</TableHead>
                    <TableHead>{t('Cashback amount')}</TableHead>
                    <TableHead>{t('Unlock time')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {props.loading ? (
                    <TableRow>
                      <TableCell colSpan={6} className='h-32 text-center'>
                        <Spinner className='mx-auto' />
                      </TableCell>
                    </TableRow>
                  ) : (
                    props.topUps.map((topUp) => {
                      const availableReward = getAvailableReward(topUp)
                      const earnedReward =
                        topUp.cash_reward_id > 0
                          ? topUp.reward_cents
                          : topUp.reward_quota
                      return (
                        <TableRow key={getTopUpRowKey(topUp)}>
                          <TableCell>
                            <div className='font-medium whitespace-nowrap'>
                              {topUp.masked_email}
                            </div>
                            {topUp.invited_at > 0 ? (
                              <div className='text-muted-foreground !text-xs whitespace-nowrap'>
                                {t('Invited {{time}}', {
                                  time: formatTimestampToDate(topUp.invited_at),
                                })}
                              </div>
                            ) : null}
                          </TableCell>
                          <TableCell className='whitespace-nowrap'>
                            {formatTimestampToDate(topUp.topup_at)}
                          </TableCell>
                          <TableCell className='whitespace-nowrap'>
                            {formatAffiliateCashback(topUp.paid_cents)}
                          </TableCell>
                          <TableCell className='whitespace-nowrap'>
                            <CashbackRule topUp={topUp} />
                          </TableCell>
                          <TableCell>
                            <div className='font-medium whitespace-nowrap'>
                              {formatReward(topUp, earnedReward)}
                            </div>
                            {topUp.status === 'available' &&
                            availableReward < earnedReward ? (
                              <div className='text-muted-foreground text-xs whitespace-nowrap'>
                                {t('Available cashback')}{' '}
                                {formatReward(topUp, availableReward)}
                              </div>
                            ) : null}
                            <Badge
                              variant='outline'
                              className={getStatusClass(topUp.status)}
                            >
                              {getStatusLabel(t, topUp.status)}
                            </Badge>
                          </TableCell>
                          <TableCell className='whitespace-nowrap'>
                            {formatTimestampToDate(topUp.available_at)}
                          </TableCell>
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>
            </div>
            {!props.loading && props.topUps.length === 0 ? (
              <Empty className='min-h-48'>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <ReceiptText />
                  </EmptyMedia>
                  <EmptyTitle>{t('No invitee top-ups yet')}</EmptyTitle>
                  <EmptyDescription>
                    {t('Successful online wallet top-ups will appear here.')}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : null}
            <ListPagination
              page={props.topUpQuery.page}
              total={props.topUpsTotal}
              onPageChange={(page) =>
                props.onTopUpQueryChange({ ...props.topUpQuery, page })
              }
            />
          </div>
        ) : (
          <HiddenTopUpFallback
            availableQuota={account?.available_quota ?? 0}
            availableCashCents={cashAccount?.available_cents ?? 0}
            transferring={props.transferring}
            onTransfer={transferAllAvailable}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
