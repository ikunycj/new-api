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
import { ChevronLeft, ChevronRight, Search } from 'lucide-react'
import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
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
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  formatQuotaAsCNY,
  formatTimestampToDate,
  parseQuotaFromCNY,
  quotaUnitsToCNY,
} from '@/lib/format'

import {
  getAffiliateSettings,
  searchAffiliateUserOverrides,
  updateAffiliateSettings,
} from '../api'
import {
  SettingsForm,
  SettingsSwitchField,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import type {
  AffiliateCashbackFrequency,
  AffiliateRegistrationRewardTrigger,
  AffiliateRewardMode,
  AffiliateSettings,
  AffiliateUserOverrideView,
} from '../types'
import { AffiliateAdjustmentsSection } from './affiliate-adjustments-section'

const SECONDS_PER_DAY = 86_400
const AFFILIATE_OVERRIDES_PAGE_SIZE = 20

type RuleFieldKey = keyof AffiliateSettings

type RuleDraft = {
  enabled: boolean
  inviterReward: string
  inviteeReward: string
  registrationRewardTrigger: AffiliateRegistrationRewardTrigger
  rewardMode: AffiliateRewardMode
  cashbackFrequency: AffiliateCashbackFrequency
  rewardRate: string
  fixedReward: string
  unlimitedReward: boolean
  maximumReward: string
  minimumTopUp: string
  holdDays: string
  minimumTransfer: string
  showInviteeTopUps: boolean
}

const ruleFieldKeys: RuleFieldKey[] = [
  'enabled',
  'inviter_reward_quota',
  'invitee_reward_quota',
  'registration_reward_trigger',
  'reward_mode',
  'cashback_frequency',
  'reward_rate_bps',
  'fixed_reward_quota',
  'unlimited_reward',
  'maximum_reward_quota',
  'minimum_topup_cents',
  'hold_seconds',
  'minimum_transfer_quota',
  'show_invitee_topups',
]

function settingsToDraft(settings: AffiliateSettings): RuleDraft {
  return {
    enabled: settings.enabled,
    inviterReward: String(quotaUnitsToCNY(settings.inviter_reward_quota)),
    inviteeReward: String(quotaUnitsToCNY(settings.invitee_reward_quota)),
    registrationRewardTrigger: settings.registration_reward_trigger,
    rewardMode: settings.reward_mode,
    cashbackFrequency: settings.cashback_frequency,
    rewardRate: String(settings.reward_rate_bps / 100),
    fixedReward: String(quotaUnitsToCNY(settings.fixed_reward_quota)),
    unlimitedReward: settings.unlimited_reward,
    maximumReward: String(quotaUnitsToCNY(settings.maximum_reward_quota)),
    minimumTopUp: String(settings.minimum_topup_cents / 100),
    holdDays: String(settings.hold_seconds / SECONDS_PER_DAY),
    minimumTransfer: String(quotaUnitsToCNY(settings.minimum_transfer_quota)),
    showInviteeTopUps: settings.show_invitee_topups,
  }
}

function draftToSettings(draft: RuleDraft): AffiliateSettings | null {
  const rewardRate = Number(draft.rewardRate)
  const minimumTopUp = Number(draft.minimumTopUp)
  const holdDays = Number(draft.holdDays)
  const quotaAmounts = [
    draft.inviterReward,
    draft.inviteeReward,
    draft.fixedReward,
    draft.maximumReward,
    draft.minimumTransfer,
  ].map(Number)
  if (
    !Number.isFinite(rewardRate) ||
    rewardRate < 0 ||
    rewardRate > 100 ||
    !Number.isFinite(minimumTopUp) ||
    minimumTopUp < 0 ||
    !Number.isInteger(holdDays) ||
    holdDays < 0 ||
    holdDays > 365 ||
    quotaAmounts.some((amount) => !Number.isFinite(amount) || amount < 0) ||
    (!draft.unlimitedReward && Number(draft.maximumReward) <= 0)
  ) {
    return null
  }
  return {
    enabled: draft.enabled,
    inviter_reward_quota: parseQuotaFromCNY(Number(draft.inviterReward)),
    invitee_reward_quota: parseQuotaFromCNY(Number(draft.inviteeReward)),
    registration_reward_trigger: draft.registrationRewardTrigger,
    reward_mode: draft.rewardMode,
    cashback_frequency: draft.cashbackFrequency,
    reward_rate_bps: Math.round(rewardRate * 100),
    fixed_reward_quota: parseQuotaFromCNY(Number(draft.fixedReward)),
    unlimited_reward: draft.unlimitedReward,
    maximum_reward_quota: parseQuotaFromCNY(Number(draft.maximumReward)),
    minimum_topup_cents: Math.round(minimumTopUp * 100),
    hold_seconds: holdDays * SECONDS_PER_DAY,
    minimum_transfer_quota: parseQuotaFromCNY(Number(draft.minimumTransfer)),
    show_invitee_topups: draft.showInviteeTopUps,
  }
}

interface RuleEditorProps {
  draft: RuleDraft
  onChange: (draft: RuleDraft) => void
  disabled?: boolean
  grouped?: boolean
}

function RuleEditor(props: RuleEditorProps) {
  const { t } = useTranslation()
  const update = <K extends keyof RuleDraft>(key: K, value: RuleDraft[K]) =>
    props.onChange({ ...props.draft, [key]: value })
  const isDisabled = () => props.disabled

  function ruleField(
    key: RuleFieldKey,
    label: string,
    content: ReactNode,
    description?: string
  ) {
    return (
      <Field key={key} data-disabled={isDisabled()}>
        <FieldLabel>{label}</FieldLabel>
        {content}
        {description ? (
          <FieldDescription>{description}</FieldDescription>
        ) : null}
      </Field>
    )
  }

  function switchField(
    key: RuleFieldKey,
    label: string,
    checked: boolean,
    onCheckedChange: (checked: boolean) => void,
    description?: string
  ) {
    return (
      <SettingsSwitchField
        key={key}
        checked={checked}
        onCheckedChange={onCheckedChange}
        label={label}
        description={description}
        disabled={isDisabled()}
        aria-label={label}
      />
    )
  }

  const enabledField = ruleField(
    'enabled',
    t('Enable referral cashback'),
    <Switch
      checked={props.draft.enabled}
      onCheckedChange={(checked) => update('enabled', checked)}
      disabled={isDisabled()}
    />,
    t('The invitation code remains valid when rewards are disabled.')
  )
  const registrationRewardTriggerField = ruleField(
    'registration_reward_trigger',
    t('Registration reward timing'),
    <ToggleGroup
      value={[props.draft.registrationRewardTrigger]}
      onValueChange={(values) => {
        const next = values[0] as AffiliateRegistrationRewardTrigger
        if (next) update('registrationRewardTrigger', next)
      }}
      disabled={isDisabled()}
      className='grid h-auto min-h-10 w-full grid-cols-2'
    >
      <ToggleGroupItem
        value='registration_success'
        className='h-auto min-h-10 w-full min-w-0 py-2 text-center leading-tight whitespace-normal'
      >
        {t('After registration')}
      </ToggleGroupItem>
      <ToggleGroupItem
        value='first_qualified_topup'
        className='h-auto min-h-10 w-full min-w-0 py-2 text-center leading-tight whitespace-normal'
      >
        {t('After first qualification')}
      </ToggleGroupItem>
    </ToggleGroup>
  )
  const inviterRewardField = ruleField(
    'inviter_reward_quota',
    t('Inviter reward'),
    <InputGroup>
      <InputGroupInput
        type='number'
        min={0}
        step='0.01'
        value={props.draft.inviterReward}
        onChange={(event) => update('inviterReward', event.target.value)}
        disabled={isDisabled()}
      />
      <InputGroupAddon align='inline-end'>
        <InputGroupText>{t('CNY')}</InputGroupText>
      </InputGroupAddon>
    </InputGroup>,
    t('The inviter reward uses the same hold period as cashback.')
  )
  const inviteeRewardField = ruleField(
    'invitee_reward_quota',
    t('Invitee reward'),
    <InputGroup>
      <InputGroupInput
        type='number'
        min={0}
        step='0.01'
        value={props.draft.inviteeReward}
        onChange={(event) => update('inviteeReward', event.target.value)}
        disabled={isDisabled()}
      />
      <InputGroupAddon align='inline-end'>
        <InputGroupText>{t('CNY')}</InputGroupText>
      </InputGroupAddon>
    </InputGroup>,
    t('The invitee reward is credited directly to account balance.')
  )
  const cashbackFrequencyField = ruleField(
    'cashback_frequency',
    t('Cashback frequency'),
    <ToggleGroup
      value={[props.draft.cashbackFrequency]}
      onValueChange={(values) => {
        const next = values[0] as AffiliateCashbackFrequency
        if (next) update('cashbackFrequency', next)
      }}
      disabled={isDisabled()}
      className='grid h-auto min-h-10 w-full grid-cols-2'
    >
      <ToggleGroupItem
        value='first_qualified'
        className='h-auto min-h-10 w-full min-w-0 py-2 text-center leading-tight whitespace-normal'
      >
        {t('First qualification only')}
      </ToggleGroupItem>
      <ToggleGroupItem
        value='every_topup'
        className='h-auto min-h-10 w-full min-w-0 py-2 text-center leading-tight whitespace-normal'
      >
        {t('Every top-up')}
      </ToggleGroupItem>
    </ToggleGroup>
  )
  const rewardModeField = ruleField(
    'reward_mode',
    t('Cashback method'),
    <ToggleGroup
      value={[props.draft.rewardMode]}
      onValueChange={(values) => {
        const next = values[0] as AffiliateRewardMode
        if (next) update('rewardMode', next)
      }}
      disabled={isDisabled()}
      className='grid h-auto min-h-10 w-full grid-cols-2'
    >
      <ToggleGroupItem
        value='percentage'
        className='h-auto min-h-10 w-full min-w-0 py-2 text-center leading-tight whitespace-normal'
      >
        {t('Percentage')}
      </ToggleGroupItem>
      <ToggleGroupItem
        value='fixed'
        className='h-auto min-h-10 w-full min-w-0 py-2 text-center leading-tight whitespace-normal'
      >
        {t('Fixed amount')}
      </ToggleGroupItem>
    </ToggleGroup>
  )
  const rewardValueField =
    props.draft.rewardMode === 'percentage'
      ? ruleField(
          'reward_rate_bps',
          t('Cashback rate'),
          <InputGroup>
            <InputGroupInput
              type='number'
              min={0}
              max={100}
              step='0.01'
              value={props.draft.rewardRate}
              onChange={(event) => update('rewardRate', event.target.value)}
              disabled={isDisabled()}
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupText>%</InputGroupText>
            </InputGroupAddon>
          </InputGroup>
        )
      : ruleField(
          'fixed_reward_quota',
          t('Fixed cashback'),
          <InputGroup>
            <InputGroupInput
              type='number'
              min={0}
              step='0.01'
              value={props.draft.fixedReward}
              onChange={(event) => update('fixedReward', event.target.value)}
              disabled={isDisabled()}
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupText>{t('CNY')}</InputGroupText>
            </InputGroupAddon>
          </InputGroup>
        )
  const minimumTopUpField = ruleField(
    'minimum_topup_cents',
    t('First qualifying cumulative top-up'),
    <InputGroup>
      <InputGroupInput
        type='number'
        min={0}
        step='0.01'
        value={props.draft.minimumTopUp}
        onChange={(event) => update('minimumTopUp', event.target.value)}
        disabled={isDisabled()}
      />
      <InputGroupAddon align='inline-end'>
        <InputGroupText>{t('CNY')}</InputGroupText>
      </InputGroupAddon>
    </InputGroup>,
    t('After qualification, later top-ups have no minimum amount.')
  )
  const maximumRewardField = props.draft.unlimitedReward
    ? null
    : ruleField(
        'maximum_reward_quota',
        t('Maximum cashback per invitee'),
        <InputGroup>
          <InputGroupInput
            type='number'
            min={0.01}
            step='0.01'
            value={props.draft.maximumReward}
            onChange={(event) => update('maximumReward', event.target.value)}
            disabled={isDisabled()}
          />
          <InputGroupAddon align='inline-end'>
            <InputGroupText>{t('CNY')}</InputGroupText>
          </InputGroupAddon>
        </InputGroup>
      )
  const holdPeriodField = ruleField(
    'hold_seconds',
    t('Hold period'),
    <InputGroup>
      <InputGroupInput
        type='number'
        min={0}
        max={365}
        step={1}
        value={props.draft.holdDays}
        onChange={(event) => update('holdDays', event.target.value)}
        disabled={isDisabled()}
      />
      <InputGroupAddon align='inline-end'>
        <InputGroupText>{t('Days')}</InputGroupText>
      </InputGroupAddon>
    </InputGroup>
  )
  const minimumTransferField = ruleField(
    'minimum_transfer_quota',
    t('Minimum balance transfer'),
    <InputGroup>
      <InputGroupInput
        type='number'
        min={0}
        step='0.01'
        value={props.draft.minimumTransfer}
        onChange={(event) => update('minimumTransfer', event.target.value)}
        disabled={isDisabled()}
      />
      <InputGroupAddon align='inline-end'>
        <InputGroupText>{t('CNY')}</InputGroupText>
      </InputGroupAddon>
    </InputGroup>
  )

  if (!props.grouped) {
    return (
      <FieldGroup className='grid gap-5 md:grid-cols-2'>
        {enabledField}
        {registrationRewardTriggerField}
        {inviterRewardField}
        {inviteeRewardField}
        {cashbackFrequencyField}
        {rewardModeField}
        {rewardValueField}
        {minimumTopUpField}
        {ruleField(
          'unlimited_reward',
          t('Unlimited cashback per invitee'),
          <Switch
            checked={props.draft.unlimitedReward}
            onCheckedChange={(checked) => update('unlimitedReward', checked)}
            disabled={isDisabled()}
          />
        )}
        {maximumRewardField}
        {holdPeriodField}
        {minimumTransferField}
        {ruleField(
          'show_invitee_topups',
          t('Allow invitee top-up records'),
          <Switch
            checked={props.draft.showInviteeTopUps}
            onCheckedChange={(checked) => update('showInviteeTopUps', checked)}
            disabled={isDisabled()}
          />,
          t('Emails are masked before records are returned to the inviter.')
        )}
      </FieldGroup>
    )
  }

  return (
    <div className='space-y-6'>
      <section className='space-y-3'>
        <h4 className='text-sm font-semibold'>{t('Preferences')}</h4>
        <div className='divide-y rounded-lg border px-3'>
          {switchField(
            'enabled',
            t('Enable referral cashback'),
            props.draft.enabled,
            (checked) => update('enabled', checked),
            t('The invitation code remains valid when rewards are disabled.')
          )}
          {switchField(
            'unlimited_reward',
            t('Unlimited cashback per invitee'),
            props.draft.unlimitedReward,
            (checked) => update('unlimitedReward', checked)
          )}
          {switchField(
            'show_invitee_topups',
            t('Allow invitee top-up records'),
            props.draft.showInviteeTopUps,
            (checked) => update('showInviteeTopUps', checked),
            t('Emails are masked before records are returned to the inviter.')
          )}
        </div>
      </section>

      <section className='space-y-3 border-t pt-5'>
        <h4 className='text-sm font-semibold'>{t('Rules')}</h4>
        <div className='grid gap-4 sm:grid-cols-2'>
          {registrationRewardTriggerField}
          {cashbackFrequencyField}
          {rewardModeField}
        </div>
      </section>

      <section className='space-y-3 border-t pt-5'>
        <h4 className='text-sm font-semibold'>{t('Configuration')}</h4>
        <div className='grid gap-4 sm:grid-cols-2'>
          {inviterRewardField}
          {inviteeRewardField}
          {rewardValueField}
          {minimumTopUpField}
          {maximumRewardField}
          {holdPeriodField}
          {minimumTransferField}
        </div>
      </section>
    </div>
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
  const [draft, setDraft] = useState<RuleDraft>()
  const [dirty, setDirty] = useState(false)
  useEffect(() => {
    if (!query.data) return
    setDraft(settingsToDraft(query.data))
    setDirty(false)
  }, [query.data])
  const mutation = useMutation({
    mutationFn: updateAffiliateSettings,
    onSuccess: async (response) => {
      if (!response.success || !response.data) return
      toast.success(t('Referral cashback settings saved'))
      setDraft(settingsToDraft(response.data))
      setDirty(false)
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'affiliate'],
      })
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

  if (!draft) {
    return <Spinner className='mx-auto my-12' />
  }

  return (
    <SettingsForm onSubmit={submit} autoComplete='off'>
      <SettingsPageFormActions
        onSave={() => submit()}
        isSaving={mutation.isPending}
        isSaveDisabled={!dirty}
        saveLabel='Save referral cashback settings'
      />
      <RuleEditor
        draft={draft}
        onChange={(next) => {
          setDraft(next)
          setDirty(true)
        }}
        disabled={mutation.isPending}
        grouped
      />
    </SettingsForm>
  )
}

function PersonalizedAffiliateSettings() {
  const { t } = useTranslation()
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['admin', 'affiliate', 'user-overrides', keyword, page],
    queryFn: () =>
      searchAffiliateUserOverrides({
        keyword,
        page,
        pageSize: AFFILIATE_OVERRIDES_PAGE_SIZE,
      }),
    select: (response) =>
      response.data ?? {
        items: [] as AffiliateUserOverrideView[],
        total: 0,
      },
  })

  const fieldLabels: Record<RuleFieldKey, string> = {
    enabled: t('Enable referral cashback'),
    inviter_reward_quota: t('Inviter reward'),
    invitee_reward_quota: t('Invitee reward'),
    registration_reward_trigger: t('Registration reward timing'),
    reward_mode: t('Cashback method'),
    cashback_frequency: t('Cashback frequency'),
    reward_rate_bps: t('Cashback rate'),
    fixed_reward_quota: t('Fixed cashback'),
    unlimited_reward: t('Unlimited cashback per invitee'),
    maximum_reward_quota: t('Maximum cashback per invitee'),
    minimum_topup_cents: t('First qualifying cumulative top-up'),
    hold_seconds: t('Hold period'),
    minimum_transfer_quota: t('Minimum balance transfer'),
    show_invitee_topups: t('Allow invitee top-up records'),
  }

  function formatValue(key: RuleFieldKey, value: unknown): string {
    if (value === null || value === undefined) return '-'
    switch (key) {
      case 'enabled':
      case 'unlimited_reward':
      case 'show_invitee_topups':
        return t(value ? 'Enabled' : 'Disabled')
      case 'registration_reward_trigger':
        return value === 'first_qualified_topup'
          ? t('After first qualification')
          : t('After registration')
      case 'reward_mode':
        return value === 'fixed' ? t('Fixed amount') : t('Percentage')
      case 'cashback_frequency':
        return value === 'every_topup'
          ? t('Every top-up')
          : t('First qualification only')
      case 'reward_rate_bps':
        return `${Number(value) / 100}%`
      case 'minimum_topup_cents':
        return `${(Number(value) / 100).toFixed(2)} ${t('CNY')}`
      case 'hold_seconds':
        return `${Number(value) / SECONDS_PER_DAY} ${t('Days')}`
      case 'inviter_reward_quota':
      case 'invitee_reward_quota':
      case 'fixed_reward_quota':
      case 'maximum_reward_quota':
      case 'minimum_transfer_quota':
        return formatQuotaAsCNY(Number(value))
      default:
        return String(value)
    }
  }

  function getOverriddenKeys(view: AffiliateUserOverrideView): RuleFieldKey[] {
    if (!view.override) return []
    return ruleFieldKeys.filter((key) => view.override?.[key] != null)
  }

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(
    1,
    Math.ceil(total / AFFILIATE_OVERRIDES_PAGE_SIZE)
  )

  let results: ReactNode
  if (query.isLoading) {
    results = (
      <div className='flex justify-center py-12'>
        <Spinner />
      </div>
    )
  } else if (query.isError) {
    results = (
      <Empty className='min-h-40'>
        <EmptyHeader>
          <EmptyTitle>{t('Loading failed')}</EmptyTitle>
        </EmptyHeader>
      </Empty>
    )
  } else if (items.length === 0) {
    results = (
      <Empty className='min-h-40'>
        <EmptyHeader>
          <EmptyTitle>
            {keyword ? t('No matching users') : t('No users')}
          </EmptyTitle>
          <EmptyDescription>
            {t('No users available. Try adjusting your search or filters.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  } else {
    results = (
      <div className='overflow-x-auto rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('User')}</TableHead>
              <TableHead>{t('Email')}</TableHead>
              <TableHead>{t('Configuration')}</TableHead>
              <TableHead>{t('Last modified at')}</TableHead>
              <TableHead>{t('Change reason')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((view) => {
              const overriddenKeys = getOverriddenKeys(view)
              const override = view.override
              return (
                <TableRow key={view.user_id}>
                  <TableCell className='min-w-44 align-top font-medium'>
                    <div>{view.username}</div>
                    <div className='text-muted-foreground text-xs'>
                      #{view.user_id}
                    </div>
                  </TableCell>
                  <TableCell className='min-w-52 align-top'>
                    {view.email || '-'}
                  </TableCell>
                  <TableCell className='max-w-[42rem] min-w-[24rem] align-top'>
                    <div className='flex flex-wrap gap-1.5'>
                      {overriddenKeys.map((key) => (
                        <Badge
                          key={key}
                          variant='outline'
                          className='max-w-full text-left whitespace-normal'
                        >
                          {fieldLabels[key]}:{' '}
                          {formatValue(key, override?.[key])}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className='min-w-44 align-top'>
                    <div>
                      {formatTimestampToDate(override?.updated_at ?? 0)}
                    </div>
                    <div className='text-muted-foreground mt-1 text-xs'>
                      {view.updated_by_username || t('User')} #
                      {override?.updated_by ?? '-'}
                    </div>
                  </TableCell>
                  <TableCell className='max-w-80 min-w-56 align-top whitespace-normal'>
                    <span className='wrap-break-word'>
                      {override?.change_reason || '-'}
                    </span>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
        {total > 0 ? (
          <div className='border-border flex items-center justify-between gap-3 border-t px-4 py-3'>
            <div className='text-muted-foreground text-xs'>
              {t('Showing')} {(page - 1) * AFFILIATE_OVERRIDES_PAGE_SIZE + 1}-
              {Math.min(page * AFFILIATE_OVERRIDES_PAGE_SIZE, total)} {t('of')}{' '}
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
      <form
        className='max-w-xl'
        onSubmit={(event) => {
          event.preventDefault()
          setPage(1)
          setKeyword(searchInput.trim())
        }}
      >
        <Field>
          <FieldLabel htmlFor='affiliate-user-search'>
            {t('Search user')}
          </FieldLabel>
          <InputGroup>
            <InputGroupInput
              id='affiliate-user-search'
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={t('Username, email, or user ID')}
            />
            <InputGroupAddon align='inline-end'>
              <InputGroupButton type='submit' aria-label={t('Search')}>
                <Search />
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </Field>
      </form>

      {results}
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
            value='personalized'
            className='h-auto min-h-9 py-2 text-center leading-tight whitespace-normal'
          >
            {t('User-specific settings')}
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
        <TabsContent value='personalized' className='pt-5'>
          <PersonalizedAffiliateSettings />
        </TabsContent>
        <TabsContent value='adjustments' className='pt-5'>
          <AffiliateAdjustmentsSection />
        </TabsContent>
      </Tabs>
    </SettingsSection>
  )
}
