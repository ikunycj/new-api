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
import { ChevronLeft, ChevronRight, RotateCcw, Search } from 'lucide-react'
import {
  useEffect,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { InviteInfoCell } from '@/components/invite-info-cell'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
} from '@/components/ui/input-group'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import {
  deleteAffiliateUserOverride,
  getAffiliateSettings,
  searchAffiliateUserOverrides,
  updateAffiliateSettings,
  saveAffiliateAgents,
} from '../api'
import {
  SettingsForm,
  SettingsSwitchField,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import type {
  AffiliateAgentUpdate,
  AffiliateSettings,
  AffiliateUserOverride,
  AffiliateUserOverrideView,
} from '../types'
import { AffiliateAdjustmentsSection } from './affiliate-adjustments-section'
import {
  AffiliateSubAgentsDialog,
  type AffiliateSubAgentDraft,
} from './affiliate-sub-agents-dialog'

const SECONDS_PER_DAY = 86_400
const AFFILIATE_AGENTS_PAGE_SIZE = 20

type GlobalDraft = {
  enabled: boolean
  rewardRate: string
  holdDays: string
}

type AgentConfigKey = 'rewardRate' | 'holdDays'

type AgentDraft = {
  isNew: boolean
  invitees: AffiliateUserOverrideView[]
  subAgents: Record<number, AffiliateSubAgentDraft>
  rewardRate: string
  holdDays: string
  affiliateDomain: string
  remark: string
  dirty: Record<AgentConfigKey, boolean>
  domainDirty: boolean
  remarkDirty: boolean
}

function settingsToDraft(settings: AffiliateSettings): GlobalDraft {
  return {
    enabled: settings.enabled,
    rewardRate: String(settings.reward_rate_bps / 100),
    holdDays: String(settings.hold_seconds / SECONDS_PER_DAY),
  }
}

function draftToSettings(draft: GlobalDraft): AffiliateSettings | null {
  const rewardRate = Number(draft.rewardRate)
  const holdDays = Number(draft.holdDays)
  if (
    !Number.isFinite(rewardRate) ||
    rewardRate < 0 ||
    rewardRate > 100 ||
    !Number.isInteger(holdDays) ||
    holdDays < 0 ||
    holdDays > 365
  ) {
    return null
  }
  return {
    enabled: draft.enabled,
    reward_rate_bps: Math.round(rewardRate * 100),
    hold_seconds: holdDays * SECONDS_PER_DAY,
  }
}

function affiliateDomainToSubdomain(domain: string | null | undefined) {
  if (!domain) return ''
  const labels = domain.split('.')
  return labels.length >= 3 ? labels[0] : domain
}

function viewToAgentDraft(view: AffiliateUserOverrideView): AgentDraft {
  return {
    isNew: !view.override,
    invitees: [],
    subAgents: {},
    rewardRate: String(view.effective_rule.reward_rate_bps / 100),
    holdDays: String(view.effective_rule.hold_seconds / SECONDS_PER_DAY),
    affiliateDomain: affiliateDomainToSubdomain(
      view.override?.affiliate_domain
    ),
    remark: view.remark,
    dirty: {
      rewardRate: false,
      holdDays: false,
    },
    domainDirty: false,
    remarkDirty: false,
  }
}

function buildAgentOverride(draft: AgentDraft): AffiliateUserOverride | null {
  const rewardRate = Number(draft.rewardRate)
  const holdDays = Number(draft.holdDays)
  if (
    (draft.dirty.rewardRate &&
      (!Number.isFinite(rewardRate) || rewardRate < 0 || rewardRate > 100)) ||
    (draft.dirty.holdDays &&
      (!Number.isInteger(holdDays) || holdDays < 0 || holdDays > 365))
  ) {
    return null
  }

  const override: AffiliateUserOverride = {}
  if (draft.dirty.rewardRate) {
    override.reward_rate_bps = Math.round(rewardRate * 100)
  }
  if (draft.dirty.holdDays) {
    override.hold_seconds = holdDays * SECONDS_PER_DAY
  }
  if (draft.domainDirty) {
    override.affiliate_domain = draft.affiliateDomain.trim()
  }
  if (draft.remarkDirty) {
    override.remark = draft.remark
  }
  return override
}

function hasAgentChanges(draft: AgentDraft): boolean {
  return (
    draft.isNew ||
    draft.invitees.length > 0 ||
    Object.keys(draft.subAgents).length > 0 ||
    draft.domainDirty ||
    draft.dirty.rewardRate ||
    draft.dirty.holdDays ||
    draft.remarkDirty
  )
}

function parseSubAgentRateBps(ratePercent: string): number | null {
  if (!/^(?:\d+|\d*\.\d{1,2})$/.test(ratePercent)) return null
  const [whole, fraction = ''] = ratePercent.split('.')
  const rateBps = Number(whole + fraction.padEnd(2, '0'))
  return Number.isSafeInteger(rateBps) ? rateBps : null
}

function SettingNumberField(props: {
  label: string
  value: string
  unit: string
  description?: string
  min?: number
  max?: number
  step?: number | string
  disabled?: boolean
  onChange: (value: string) => void
}) {
  return (
    <Field>
      <FieldLabel>{props.label}</FieldLabel>
      <InputGroup>
        <InputGroupInput
          type='number'
          value={props.value}
          min={props.min}
          max={props.max}
          step={props.step}
          onChange={(event) => props.onChange(event.target.value)}
          disabled={props.disabled}
        />
        <InputGroupAddon align='inline-end'>
          <InputGroupText>{props.unit}</InputGroupText>
        </InputGroupAddon>
      </InputGroup>
      {props.description ? (
        <FieldDescription>{props.description}</FieldDescription>
      ) : null}
    </Field>
  )
}

function GlobalAffiliateSettings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ['admin', 'affiliate', 'settings'],
    queryFn: getAffiliateSettings,
    select: (response) => response.data,
  })
  const [draft, setDraft] = useState<GlobalDraft>()
  const [dirty, setDirty] = useState(false)

  useEffect(() => {
    if (!query.data) return
    setDraft(settingsToDraft(query.data))
    setDirty(false)
  }, [query.data])

  const mutation = useMutation({
    mutationFn: updateAffiliateSettings,
    onSuccess: async (response) => {
      if (!response.success || !response.data) {
        toast.error(response.message || t('Save failed'))
        return
      }
      setDraft(settingsToDraft(response.data))
      setDirty(false)
      toast.success(t('Referral cashback settings saved'))
      await queryClient.invalidateQueries({ queryKey: ['admin', 'affiliate'] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Save failed'))
    },
  })

  function submit(event?: FormEvent) {
    event?.preventDefault()
    if (!draft) return
    const settings = draftToSettings(draft)
    if (!settings) {
      toast.error(t('Check the cashback settings and try again'))
      return
    }
    mutation.mutate(settings)
  }

  if (!draft) return <Spinner className='mx-auto my-12' />

  return (
    <SettingsForm onSubmit={submit} autoComplete='off'>
      <SettingsPageFormActions
        onSave={() => submit()}
        isSaving={mutation.isPending}
        isSaveDisabled={!dirty}
        saveLabel='Save referral cashback settings'
      />
      <div className='space-y-6'>
        <section className='space-y-3'>
          <div className='rounded-lg border px-3'>
            <SettingsSwitchField
              checked={draft.enabled}
              onCheckedChange={(checked) => {
                setDraft((current) =>
                  current ? { ...current, enabled: checked } : current
                )
                setDirty(true)
              }}
              label={t('Enable referral cashback')}
              description={t(
                'The invitation code remains valid when rewards are disabled.'
              )}
              disabled={mutation.isPending}
              aria-label={t('Enable referral cashback')}
              className='py-3'
            />
          </div>
        </section>

        <section className='space-y-3 border-t pt-5'>
          <h4 className='text-sm font-semibold'>{t('Configuration')}</h4>
          <FieldGroup className='grid max-w-2xl gap-5'>
            <SettingNumberField
              label={t('Cashback rate')}
              value={draft.rewardRate}
              unit='%'
              min={0}
              max={100}
              step='0.01'
              description={t(
                'Every successful online top-up is eligible for percentage cashback.'
              )}
              disabled={mutation.isPending}
              onChange={(value) => {
                setDraft((current) =>
                  current ? { ...current, rewardRate: value } : current
                )
                setDirty(true)
              }}
            />
            <SettingNumberField
              label={t('Hold period')}
              value={draft.holdDays}
              unit={t('Days')}
              min={0}
              max={365}
              step={1}
              description={t(
                'Cashback becomes available after the hold period ends.'
              )}
              disabled={mutation.isPending}
              onChange={(value) => {
                setDraft((current) =>
                  current ? { ...current, holdDays: value } : current
                )
                setDirty(true)
              }}
            />
          </FieldGroup>
        </section>
      </div>
    </SettingsForm>
  )
}

function CopyableText(props: { value: string; className?: string }) {
  const { copyToClipboard } = useCopyToClipboard()
  const displayValue = props.value || '-'

  return (
    <TooltipProvider delay={200}>
      <Tooltip>
        <TooltipTrigger
          render={
            <button
              type='button'
              className={`block max-w-full min-w-0 cursor-copy text-left ${props.className ?? ''}`}
              disabled={!props.value}
              onClick={() => {
                if (props.value) void copyToClipboard(props.value)
              }}
            />
          }
        >
          <span className='block truncate'>{displayValue}</span>
        </TooltipTrigger>
        <TooltipContent side='top' className='max-w-sm break-all'>
          {displayValue}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

function AgentNumberCell(props: {
  label: string
  value: string
  unit: string
  min?: number
  max?: number
  step?: number | string
  disabled: boolean
  onChange: (value: string) => void
}) {
  return (
    <TableCell className='align-top'>
      <div className='grid min-w-36 gap-1.5'>
        <label className='text-muted-foreground text-xs'>{props.label}</label>
        <InputGroup>
          <InputGroupInput
            type='number'
            value={props.value}
            min={props.min}
            max={props.max}
            step={props.step}
            aria-label={props.label}
            onChange={(event) => props.onChange(event.target.value)}
            disabled={props.disabled}
          />
          <InputGroupAddon align='inline-end'>
            <InputGroupText>{props.unit}</InputGroupText>
          </InputGroupAddon>
        </InputGroup>
      </div>
    </TableCell>
  )
}

function AgentSearchResults(props: {
  items: AffiliateUserOverrideView[]
  addedUserIDs: Set<number>
  onAdd: (view: AffiliateUserOverrideView) => void
}) {
  const { t } = useTranslation()
  if (props.items.length === 0) return null
  return (
    <div className='rounded-lg border'>
      <div className='border-b px-4 py-3 text-sm font-medium'>
        {t('Search results')}
      </div>
      <div className='divide-y'>
        {props.items.map((view) => {
          const added =
            props.addedUserIDs.has(view.user_id) || Boolean(view.override)
          return (
            <div
              key={view.user_id}
              className='flex flex-wrap items-center justify-between gap-3 px-4 py-3'
            >
              <div className='min-w-0'>
                <div className='font-medium'>{view.username}</div>
                <div className='text-muted-foreground text-sm'>
                  {view.email || '-'}
                  {view.remark ? ` · ${view.remark}` : ''}
                </div>
              </div>
              <Button
                type='button'
                size='sm'
                variant={added ? 'outline' : 'default'}
                disabled={added}
                onClick={() => props.onAdd(view)}
              >
                {added ? t('Added') : t('Add agent')}
              </Button>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function PersonalizedAffiliateSettings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [addedAgents, setAddedAgents] = useState<
    Record<number, AffiliateUserOverrideView>
  >({})
  const [rowDrafts, setRowDrafts] = useState<Record<number, AgentDraft>>({})
  const [inviteTarget, setInviteTarget] =
    useState<AffiliateUserOverrideView | null>(null)
  const [subAgentTarget, setSubAgentTarget] =
    useState<AffiliateUserOverrideView | null>(null)
  const [inviteSearch, setInviteSearch] = useState('')
  const [inviteKeyword, setInviteKeyword] = useState('')
  const inviteeQuery = useQuery({
    queryKey: ['admin', 'affiliate', 'invitee-search', inviteKeyword],
    queryFn: () =>
      searchAffiliateUserOverrides({
        keyword: inviteKeyword,
        page: 1,
        pageSize: 20,
      }),
    enabled: inviteTarget !== null && inviteKeyword !== '',
    select: (response) => response.data?.items ?? [],
  })

  const agentsQuery = useQuery({
    queryKey: ['admin', 'affiliate', 'agents', page],
    queryFn: () =>
      searchAffiliateUserOverrides({
        keyword: '',
        page,
        pageSize: AFFILIATE_AGENTS_PAGE_SIZE,
      }),
    select: (response) =>
      response.data ?? {
        items: [] as AffiliateUserOverrideView[],
        total: 0,
      },
  })
  const searchQuery = useQuery({
    queryKey: ['admin', 'affiliate', 'agent-search', keyword],
    queryFn: () =>
      searchAffiliateUserOverrides({
        keyword,
        page: 1,
        pageSize: AFFILIATE_AGENTS_PAGE_SIZE,
      }),
    enabled: keyword.trim() !== '',
    select: (response) => response.data?.items ?? [],
  })

  const agentItems = useMemo(() => {
    const items = [
      ...Object.values(addedAgents),
      ...(agentsQuery.data?.items ?? []),
    ]
    const unique = new Map<number, AffiliateUserOverrideView>()
    for (const item of items) unique.set(item.user_id, item)
    return [...unique.values()]
  }, [addedAgents, agentsQuery.data?.items])
  const total = agentsQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / AFFILIATE_AGENTS_PAGE_SIZE))
  const addedUserIDs = useMemo(
    () => new Set(agentItems.map((item) => item.user_id)),
    [agentItems]
  )

  function addAgent(view: AffiliateUserOverrideView) {
    setAddedAgents((current) => ({ ...current, [view.user_id]: view }))
    setRowDrafts((current) => ({
      ...current,
      [view.user_id]: viewToAgentDraft(view),
    }))
    setKeyword('')
    setSearchInput('')
  }

  function updateRow(
    view: AffiliateUserOverrideView,
    key: AgentConfigKey,
    value: string
  ) {
    setRowDrafts((current) => {
      const row = current[view.user_id] ?? viewToAgentDraft(view)
      return {
        ...current,
        [view.user_id]: {
          ...row,
          [key]: value,
          dirty: { ...row.dirty, [key]: true },
        },
      }
    })
  }

  function updateDomain(view: AffiliateUserOverrideView, value: string) {
    setRowDrafts((current) => {
      const row = current[view.user_id] ?? viewToAgentDraft(view)
      return {
        ...current,
        [view.user_id]: { ...row, affiliateDomain: value, domainDirty: true },
      }
    })
  }

  function updateRemark(view: AffiliateUserOverrideView, value: string) {
    setRowDrafts((current) => {
      const row = current[view.user_id] ?? viewToAgentDraft(view)
      return {
        ...current,
        [view.user_id]: { ...row, remark: value, remarkDirty: true },
      }
    })
  }

  function updateSubAgent(
    parent: AffiliateUserOverrideView,
    userId: number,
    change: AffiliateSubAgentDraft | null
  ) {
    setRowDrafts((current) => {
      const row = current[parent.user_id] ?? viewToAgentDraft(parent)
      const subAgents = { ...row.subAgents }
      if (change) {
        subAgents[userId] = change
      } else {
        delete subAgents[userId]
      }
      return {
        ...current,
        [parent.user_id]: { ...row, subAgents },
      }
    })
  }

  const saveMutation = useMutation({
    mutationFn: saveAffiliateAgents,
    onSuccess: async (response) => {
      if (!response.success || !response.data) {
        toast.error(response.message || t('Save failed'))
        return
      }
      const savedIDs = new Set(response.data.map((view) => view.user_id))
      setAddedAgents((current) =>
        Object.fromEntries(
          Object.entries(current).filter(([id]) => !savedIDs.has(Number(id)))
        )
      )
      setRowDrafts((current) =>
        Object.fromEntries(
          Object.entries(current).filter(([id]) => !savedIDs.has(Number(id)))
        )
      )
      toast.success(t('Agent settings saved'))
      await queryClient.invalidateQueries({ queryKey: ['admin', 'affiliate'] })
      await queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Save failed'))
    },
  })

  const removeMutation = useMutation({
    mutationFn: (userId: number) => deleteAffiliateUserOverride(userId),
    onSuccess: async (response, userId) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to remove agent'))
        return
      }
      setAddedAgents((current) => {
        const next = { ...current }
        delete next[userId]
        return next
      })
      setRowDrafts((current) => {
        const next = { ...current }
        delete next[userId]
        return next
      })
      setRemoveTarget(null)
      toast.success(t('Agent removed'))
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'affiliate'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Remove failed'))
    },
  })

  const hasPendingChanges = Object.values(rowDrafts).some(hasAgentChanges)
  const [removeTarget, setRemoveTarget] =
    useState<AffiliateUserOverrideView | null>(null)

  function saveAllAgents() {
    const requests: AffiliateAgentUpdate[] = []
    for (const [userId, draft] of Object.entries(rowDrafts)) {
      if (!hasAgentChanges(draft)) continue
      const override = buildAgentOverride(draft)
      if (!override) {
        toast.error(t('Check the agent settings and try again'))
        return
      }
      const subAgents: NonNullable<AffiliateAgentUpdate['sub_agents']> = []
      for (const child of Object.values(draft.subAgents)) {
        const rateBps =
          child.ratePercent === null
            ? -1
            : parseSubAgentRateBps(child.ratePercent)
        if (rateBps === null) {
          toast.error(
            `请为子代理 ${child.username} 填写可精确保存的非负比例，最多两位小数。`
          )
          return
        }
        subAgents.push({ user_id: child.user_id, rate_bps: rateBps })
      }
      requests.push({
        user_id: Number(userId),
        override,
        invitee_user_ids: draft.invitees.map((invitee) => invitee.user_id),
        ...(subAgents.length > 0 ? { sub_agents: subAgents } : {}),
      })
    }
    if (requests.length > 0) saveMutation.mutate(requests)
  }

  function removeAgent(view: AffiliateUserOverrideView) {
    setRemoveTarget(view)
  }

  function confirmRemoveAgent() {
    if (!removeTarget) return
    if (!removeTarget.override) {
      setAddedAgents((current) => {
        const next = { ...current }
        delete next[removeTarget.user_id]
        return next
      })
      setRowDrafts((current) => {
        const next = { ...current }
        delete next[removeTarget.user_id]
        return next
      })
      setRemoveTarget(null)
      return
    }
    removeMutation.mutate(removeTarget.user_id)
  }

  let content: ReactNode
  if (agentsQuery.isLoading) {
    content = (
      <div className='flex justify-center py-12'>
        <Spinner />
      </div>
    )
  } else if (agentsQuery.isError) {
    content = (
      <Empty className='min-h-40'>
        <EmptyHeader>
          <EmptyTitle>{t('Loading failed')}</EmptyTitle>
        </EmptyHeader>
      </Empty>
    )
  } else {
    content = (
      <div className='min-w-0 overflow-x-auto rounded-lg border'>
        <Table className='min-w-[1400px] table-fixed'>
          <TableHeader>
            <TableRow>
              <TableHead className='w-40'>{t('User')}</TableHead>
              <TableHead className='w-52'>{t('Email')}</TableHead>
              <TableHead className='w-56'>{t('Invite Info')}</TableHead>
              <TableHead className='w-40'>{t('Remark')}</TableHead>
              <TableHead className='w-64'>{t('Affiliate domain')}</TableHead>
              <TableHead className='w-36'>{t('Cashback rate')}</TableHead>
              <TableHead className='w-36'>{t('Hold period')}</TableHead>
              <TableHead className='sticky right-0 z-30 w-40 bg-[var(--table-header)] text-right shadow-[-8px_0_10px_-10px_hsl(var(--foreground))]'>
                {t('Action')}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {agentItems.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className='h-36 text-center'>
                  <div className='mx-auto flex max-w-md flex-col items-center gap-1'>
                    <p className='font-medium'>{t('No agents')}</p>
                    <p className='text-muted-foreground text-sm'>
                      {t('Search for a user above to add an agent.')}
                    </p>
                  </div>
                </TableCell>
              </TableRow>
            ) : null}
            {agentItems.map((view) => {
              const row = rowDrafts[view.user_id] ?? viewToAgentDraft(view)
              const disabled =
                saveMutation.isPending || removeMutation.isPending
              return (
                <TableRow key={view.user_id}>
                  <TableCell className='max-w-40 align-top font-medium'>
                    <CopyableText
                      value={view.username}
                      className='max-w-36 font-medium'
                    />
                    <CopyableText
                      value={String(view.user_id)}
                      className='text-muted-foreground max-w-24 text-xs'
                    />
                  </TableCell>
                  <TableCell className='max-w-52 align-top'>
                    <CopyableText value={view.email} className='max-w-48' />
                  </TableCell>
                  <TableCell className='align-top'>
                    <InviteInfoCell
                      userId={view.user_id}
                      info={view.invite_info}
                      className='min-w-0 break-words whitespace-normal'
                    />
                  </TableCell>
                  <TableCell className='max-w-40 align-top'>
                    <InputGroup className='max-w-40 min-w-0'>
                      <InputGroupInput
                        className='min-w-0 truncate'
                        value={row.remark}
                        title={row.remark || undefined}
                        maxLength={255}
                        aria-label={t('Remark')}
                        onChange={(event) =>
                          updateRemark(view, event.target.value)
                        }
                        disabled={disabled}
                      />
                    </InputGroup>
                  </TableCell>
                  <TableCell className='max-w-64 align-top'>
                    <InputGroup className='max-w-64 min-w-0'>
                      <InputGroupInput
                        className='min-w-0 truncate'
                        value={row.affiliateDomain}
                        title={row.affiliateDomain || undefined}
                        placeholder={t('Enter subdomain, e.g. a')}
                        aria-label={t('Affiliate domain')}
                        onChange={(event) =>
                          updateDomain(view, event.target.value)
                        }
                        disabled={disabled}
                      />
                    </InputGroup>
                  </TableCell>
                  <AgentNumberCell
                    label={t('Cashback rate')}
                    value={row.rewardRate}
                    unit='%'
                    min={0}
                    max={100}
                    step='0.01'
                    disabled={disabled}
                    onChange={(value) => updateRow(view, 'rewardRate', value)}
                  />
                  <AgentNumberCell
                    label={t('Hold period')}
                    value={row.holdDays}
                    unit={t('Days')}
                    min={0}
                    max={365}
                    step={1}
                    disabled={disabled}
                    onChange={(value) => updateRow(view, 'holdDays', value)}
                  />
                  <TableCell className='bg-background sticky right-0 z-10 w-40 text-right align-top shadow-[-8px_0_10px_-10px_hsl(var(--foreground))]'>
                    <div className='flex flex-col items-end gap-2'>
                      <Button
                        type='button'
                        size='sm'
                        variant='outline'
                        disabled={disabled}
                        onClick={() => {
                          setInviteTarget(view)
                          setInviteSearch('')
                          setInviteKeyword('')
                        }}
                      >
                        添加被邀请者
                      </Button>
                      {row.invitees.length > 0 ? (
                        <span className='text-muted-foreground text-xs'>
                          待添加 {row.invitees.length} 人
                        </span>
                      ) : null}
                      <Button
                        type='button'
                        size='sm'
                        variant='outline'
                        disabled={disabled}
                        onClick={() => setSubAgentTarget(view)}
                      >
                        管理子代理
                      </Button>
                      {Object.keys(row.subAgents).length > 0 ? (
                        <span className='text-muted-foreground text-xs'>
                          子代理待保存
                        </span>
                      ) : null}
                      <Button
                        type='button'
                        size='sm'
                        variant='outline'
                        disabled={disabled}
                        onClick={() => removeAgent(view)}
                        aria-label={t('Remove agent')}
                        title={t('Remove agent')}
                      >
                        <RotateCcw data-icon='inline-start' />
                        {t('Remove')}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
        {total > 0 ? (
          <div className='border-border flex items-center justify-between gap-3 border-t px-4 py-3'>
            <div className='text-muted-foreground text-xs'>
              {t('Showing')} {(page - 1) * AFFILIATE_AGENTS_PAGE_SIZE + 1}-
              {Math.min(page * AFFILIATE_AGENTS_PAGE_SIZE, total)} {t('of')}{' '}
              {total}
            </div>
            <div className='flex items-center gap-2'>
              <Button
                type='button'
                variant='outline'
                size='sm'
                className='size-8 p-0'
                disabled={page <= 1}
                onClick={() => setPage((current) => current - 1)}
                aria-label={t('Previous page')}
                title={t('Previous page')}
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
                title={t('Next page')}
              >
                <ChevronRight />
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <div className='grid gap-5'>
      <SettingsPageFormActions
        onSave={saveAllAgents}
        isSaving={saveMutation.isPending}
        isSaveDisabled={!hasPendingChanges}
        saveLabel='Save'
      />
      <form
        className='max-w-xl'
        onSubmit={(event) => {
          event.preventDefault()
          setKeyword(searchInput.trim())
        }}
      >
        <Field>
          <FieldLabel htmlFor='affiliate-agent-search'>
            {t('Add agent')}
          </FieldLabel>
          <InputGroup>
            <InputGroupInput
              id='affiliate-agent-search'
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder='用户 ID、名称或邮箱'
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupButton type='submit' aria-label={t('Search')}>
                <Search />
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </Field>
      </form>

      {searchQuery.isFetching ? <Spinner className='mx-auto' /> : null}
      {!searchQuery.isFetching && keyword ? (
        <AgentSearchResults
          items={searchQuery.data ?? []}
          addedUserIDs={addedUserIDs}
          onAdd={addAgent}
        />
      ) : null}
      {content}
      <Dialog
        open={inviteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setInviteTarget(null)
        }}
      >
        <DialogContent className='sm:max-w-xl'>
          <DialogHeader>
            <DialogTitle>添加被邀请者 · {inviteTarget?.username}</DialogTitle>
            <DialogDescription>
              选择尚未绑定邀请者的用户，最后通过页面右上角保存。历史充值不补发返现。
            </DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(event) => {
              event.preventDefault()
              setInviteKeyword(inviteSearch.trim())
            }}
          >
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor='affiliate-invitee-search'>
                  被邀请者
                </FieldLabel>
                <InputGroup>
                  <InputGroupInput
                    id='affiliate-invitee-search'
                    value={inviteSearch}
                    onChange={(event) => setInviteSearch(event.target.value)}
                    placeholder='用户 ID、名称或邮箱'
                  />
                  <InputGroupAddon align='inline-end'>
                    <InputGroupButton type='submit' aria-label='搜索被邀请者'>
                      <Search />
                    </InputGroupButton>
                  </InputGroupAddon>
                </InputGroup>
              </Field>
            </FieldGroup>
          </form>
          <div className='max-h-80 overflow-y-auto'>
            {inviteeQuery.isFetching ? <Spinner className='mx-auto' /> : null}
            {inviteeQuery.isError ? (
              <p className='text-destructive text-sm'>搜索失败，请重试</p>
            ) : null}
            {inviteKeyword &&
            !inviteeQuery.isFetching &&
            inviteeQuery.data?.length === 0 ? (
              <p className='text-muted-foreground text-sm'>未找到用户</p>
            ) : null}
            {inviteeQuery.data?.map((invitee) => {
              if (!inviteTarget) return null
              const draft =
                rowDrafts[inviteTarget.user_id] ??
                viewToAgentDraft(inviteTarget)
              const staged = Object.values(rowDrafts).some((row) =>
                row.invitees.some((user) => user.user_id === invitee.user_id)
              )
              const bound = Boolean(invitee.invite_info?.inviter_user_id)
              const unavailable =
                staged || bound || invitee.user_id === inviteTarget.user_id
              let addLabel = '添加'
              if (staged) addLabel = '已选择'
              else if (bound) addLabel = '已绑定邀请者'
              return (
                <div
                  key={invitee.user_id}
                  className='flex items-center justify-between gap-3 border-b py-3'
                >
                  <div className='min-w-0'>
                    <p className='truncate font-medium'>
                      {invitee.username} · {invitee.user_id}
                    </p>
                    <p
                      className='text-muted-foreground truncate text-sm'
                      title={invitee.email}
                    >
                      {invitee.email || '-'}
                    </p>
                  </div>
                  <Button
                    type='button'
                    size='sm'
                    disabled={unavailable || saveMutation.isPending}
                    onClick={() =>
                      setRowDrafts((current) => ({
                        ...current,
                        [inviteTarget.user_id]: {
                          ...draft,
                          invitees: [...draft.invitees, invitee],
                        },
                      }))
                    }
                  >
                    {addLabel}
                  </Button>
                </div>
              )
            })}
          </div>
          {inviteTarget
            ? (rowDrafts[inviteTarget.user_id]?.invitees ?? []).map(
                (invitee) => (
                  <div
                    key={invitee.user_id}
                    className='flex items-center justify-between gap-3'
                  >
                    <span className='truncate text-sm'>
                      {invitee.username} · {invitee.email || invitee.user_id}
                    </span>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      disabled={saveMutation.isPending}
                      onClick={() =>
                        setRowDrafts((current) => {
                          const draft = current[inviteTarget.user_id]
                          return {
                            ...current,
                            [inviteTarget.user_id]: {
                              ...draft,
                              invitees: draft.invitees.filter(
                                (user) => user.user_id !== invitee.user_id
                              ),
                            },
                          }
                        })
                      }
                    >
                      移除
                    </Button>
                  </div>
                )
              )
            : null}
        </DialogContent>
      </Dialog>
      {subAgentTarget ? (
        <AffiliateSubAgentsDialog
          key={subAgentTarget.user_id}
          parent={subAgentTarget}
          changes={rowDrafts[subAgentTarget.user_id]?.subAgents ?? {}}
          otherStagedChildIDs={
            new Set(
              Object.entries(rowDrafts)
                .filter(([id]) => Number(id) !== subAgentTarget.user_id)
                .flatMap(([, row]) =>
                  Object.values(row.subAgents).map((child) => child.user_id)
                )
            )
          }
          onChange={(userId, change) =>
            updateSubAgent(subAgentTarget, userId, change)
          }
          onClose={() => setSubAgentTarget(null)}
        />
      ) : null}
      <ConfirmDialog
        open={Boolean(removeTarget)}
        onOpenChange={(open) => {
          if (!open && !removeMutation.isPending) setRemoveTarget(null)
        }}
        title={t('Remove agent')}
        desc='移除代理配置会解除其上级和子代理关系，历史邀请及返现记录会保留。'
        destructive
        confirmText={t('Remove')}
        handleConfirm={confirmRemoveAgent}
        isLoading={removeMutation.isPending}
      />
    </div>
  )
}

export function AffiliateSettingsSection() {
  const { t } = useTranslation()
  return (
    <SettingsSection title={t('Referral Cashback')}>
      <Tabs defaultValue='global'>
        <TabsList className='grid w-full max-w-2xl grid-cols-3 group-data-horizontal/tabs:h-auto'>
          <TabsTrigger
            value='global'
            className='h-auto min-h-9 py-2 text-center leading-tight whitespace-normal'
          >
            {t('Global settings')}
          </TabsTrigger>
          <TabsTrigger
            value='agents'
            className='h-auto min-h-9 py-2 text-center leading-tight whitespace-normal'
          >
            {t('Agents')}
          </TabsTrigger>
          <TabsTrigger
            value='adjustments'
            className='h-auto min-h-9 py-2 text-center leading-tight whitespace-normal'
          >
            {t('Manual adjustments')}
          </TabsTrigger>
        </TabsList>
        <TabsContent value='global' className='pt-5'>
          <GlobalAffiliateSettings />
        </TabsContent>
        <TabsContent value='agents' className='pt-5'>
          <PersonalizedAffiliateSettings />
        </TabsContent>
        <TabsContent value='adjustments' className='pt-5'>
          <AffiliateAdjustmentsSection />
        </TabsContent>
      </Tabs>
    </SettingsSection>
  )
}
