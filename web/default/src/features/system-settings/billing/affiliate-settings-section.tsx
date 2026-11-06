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

import { Button } from '@/components/ui/button'
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
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { parseQuotaFromCNY, quotaUnitsToCNY } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  getAffiliateSettings,
  searchAffiliateUserOverrides,
  updateAffiliateUserOverride,
  updateAffiliateSettings,
  deleteAffiliateUserOverride,
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
  AffiliateUserOverride,
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

type AffiliateConfigDraft = {
  inviterReward: string
  inviteeReward: string
  rewardValue: string
  minimumTopUp: string
  maximumReward: string
  holdDays: string
}

const affiliateConfigFieldKeys = [
  'inviter_reward_quota',
  'invitee_reward_quota',
  'reward_value',
  'minimum_topup_cents',
  'maximum_reward_quota',
  'hold_seconds',
] as const

type AffiliateConfigFieldKey = (typeof affiliateConfigFieldKeys)[number]

const affiliateConfigDraftKeys: Record<
  AffiliateConfigFieldKey,
  keyof AffiliateConfigDraft
> = {
  inviter_reward_quota: 'inviterReward',
  invitee_reward_quota: 'inviteeReward',
  reward_value: 'rewardValue',
  minimum_topup_cents: 'minimumTopUp',
  maximum_reward_quota: 'maximumReward',
  hold_seconds: 'holdDays',
}

type AffiliateConfigFollowing = Record<AffiliateConfigFieldKey, boolean>

type AffiliateUserRowDraft = {
  config: AffiliateConfigDraft
  following: AffiliateConfigFollowing
  note: string
}

function settingsToConfigDraft(
  settings: AffiliateSettings & { reward_mode: AffiliateRewardMode }
): AffiliateConfigDraft {
  return {
    inviterReward: String(quotaUnitsToCNY(settings.inviter_reward_quota)),
    inviteeReward: String(quotaUnitsToCNY(settings.invitee_reward_quota)),
    rewardValue:
      settings.reward_mode === 'percentage'
        ? String(settings.reward_rate_bps / 100)
        : String(quotaUnitsToCNY(settings.fixed_reward_quota)),
    minimumTopUp: String(settings.minimum_topup_cents / 100),
    maximumReward: String(quotaUnitsToCNY(settings.maximum_reward_quota)),
    holdDays: String(settings.hold_seconds / SECONDS_PER_DAY),
  }
}

function buildAffiliateConfigOverride(
  draft: AffiliateConfigDraft,
  rewardMode: AffiliateRewardMode,
  unlimitedReward: boolean,
  following: AffiliateConfigFollowing,
  note: string
): AffiliateUserOverride | null {
  const hasOverrides = affiliateConfigFieldKeys.some((key) => !following[key])
  if (!hasOverrides) return null

  const rawValues: Record<AffiliateConfigFieldKey, string> = {
    inviter_reward_quota: draft.inviterReward,
    invitee_reward_quota: draft.inviteeReward,
    reward_value: draft.rewardValue,
    minimum_topup_cents: draft.minimumTopUp,
    maximum_reward_quota: draft.maximumReward,
    hold_seconds: draft.holdDays,
  }
  const draftValues: Record<AffiliateConfigFieldKey, number | null> = {
    inviter_reward_quota: following.inviter_reward_quota
      ? null
      : Number(draft.inviterReward),
    invitee_reward_quota: following.invitee_reward_quota
      ? null
      : Number(draft.inviteeReward),
    reward_value: following.reward_value ? null : Number(draft.rewardValue),
    minimum_topup_cents: following.minimum_topup_cents
      ? null
      : Number(draft.minimumTopUp),
    maximum_reward_quota: following.maximum_reward_quota
      ? null
      : Number(draft.maximumReward),
    hold_seconds: following.hold_seconds ? null : Number(draft.holdDays),
  }
  const activeValues = affiliateConfigFieldKeys
    .map((key) => draftValues[key])
    .filter((value): value is number => value !== null)
  const hasBlankValue = affiliateConfigFieldKeys.some((key) => {
    return !following[key] && rawValues[key].trim() === ''
  })
  const rewardValue = draftValues.reward_value
  const maximumReward = draftValues.maximum_reward_quota
  const holdDays = draftValues.hold_seconds
  if (
    hasBlankValue ||
    activeValues.some((value) => !Number.isFinite(value) || value < 0) ||
    (rewardValue !== null &&
      rewardMode === 'percentage' &&
      rewardValue > 100) ||
    (maximumReward !== null && !unlimitedReward && maximumReward <= 0) ||
    (holdDays !== null &&
      (!Number.isInteger(holdDays) || holdDays < 0 || holdDays > 365))
  ) {
    return null
  }
  return {
    enabled: null,
    inviter_reward_quota:
      draftValues.inviter_reward_quota === null
        ? null
        : parseQuotaFromCNY(draftValues.inviter_reward_quota),
    invitee_reward_quota:
      draftValues.invitee_reward_quota === null
        ? null
        : parseQuotaFromCNY(draftValues.invitee_reward_quota),
    registration_reward_trigger: null,
    reward_mode: null,
    cashback_frequency: null,
    reward_rate_bps:
      rewardMode === 'percentage' && rewardValue !== null
        ? Math.round(rewardValue * 100)
        : null,
    fixed_reward_quota:
      rewardMode === 'fixed' && rewardValue !== null
        ? parseQuotaFromCNY(rewardValue)
        : null,
    unlimited_reward: null,
    maximum_reward_quota:
      maximumReward === null ? null : parseQuotaFromCNY(maximumReward),
    minimum_topup_cents:
      draftValues.minimum_topup_cents === null
        ? null
        : Math.round(draftValues.minimum_topup_cents * 100),
    hold_seconds: holdDays === null ? null : holdDays * SECONDS_PER_DAY,
    minimum_transfer_quota: null,
    show_invitee_topups: null,
    change_reason: note.trim(),
  }
}

function getAffiliateConfigFollowing(
  view: AffiliateUserOverrideView
): AffiliateConfigFollowing {
  const override = view.override
  const rewardMode = view.global_rule.reward_mode
  return {
    inviter_reward_quota: override?.inviter_reward_quota == null,
    invitee_reward_quota: override?.invitee_reward_quota == null,
    reward_value:
      rewardMode === 'percentage'
        ? override?.reward_rate_bps == null
        : override?.fixed_reward_quota == null,
    minimum_topup_cents: override?.minimum_topup_cents == null,
    maximum_reward_quota: override?.maximum_reward_quota == null,
    hold_seconds: override?.hold_seconds == null,
  }
}

function affiliateViewToRowDraft(
  view: AffiliateUserOverrideView
): AffiliateUserRowDraft {
  return {
    config: settingsToConfigDraft(view.effective_rule),
    following: getAffiliateConfigFollowing(view),
    note: view.override?.change_reason ?? '',
  }
}

function affiliateViewToGlobalRowDraft(
  view: AffiliateUserOverrideView
): AffiliateUserRowDraft {
  return {
    config: settingsToConfigDraft(view.global_rule),
    following: {
      inviter_reward_quota: true,
      invitee_reward_quota: true,
      reward_value: true,
      minimum_topup_cents: true,
      maximum_reward_quota: true,
      hold_seconds: true,
    },
    note: '',
  }
}

function hasAffiliateConfigurationOverride(
  view: AffiliateUserOverrideView
): boolean {
  const override = view.override
  return Boolean(
    override &&
    (override.inviter_reward_quota != null ||
      override.invitee_reward_quota != null ||
      override.reward_rate_bps != null ||
      override.fixed_reward_quota != null ||
      override.maximum_reward_quota != null ||
      override.minimum_topup_cents != null ||
      override.hold_seconds != null)
  )
}

interface AffiliateConfigTableCellProps {
  value: string
  following: boolean
  unit: string
  label: string
  inputProps?: {
    min?: number
    max?: number
    step?: number | string
  }
  disabled: boolean
  onValueChange: (value: string) => void
  onFollowingChange: (following: boolean) => void
}

function AffiliateConfigTableCell(props: AffiliateConfigTableCellProps) {
  const { t } = useTranslation()
  return (
    <TableCell className='align-top'>
      <div className='grid min-w-36 gap-1.5'>
        <label className='text-muted-foreground flex items-center gap-1 text-xs'>
          <Switch
            size='sm'
            checked={props.following}
            onCheckedChange={props.onFollowingChange}
            disabled={props.disabled}
            aria-label={`${props.label}: ${t('Follow global')}`}
          />
          <span>{t('Follow global')}</span>
        </label>
        <InputGroup>
          <InputGroupInput
            type='number'
            value={props.value}
            onChange={(event) => props.onValueChange(event.target.value)}
            disabled={props.disabled || props.following}
            aria-label={props.label}
            {...props.inputProps}
          />
          <InputGroupAddon align='inline-end'>
            <InputGroupText>{props.unit}</InputGroupText>
          </InputGroupAddon>
        </InputGroup>
      </div>
    </TableCell>
  )
}

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
  following?: Record<RuleFieldKey, boolean>
  onFollowingChange?: (key: RuleFieldKey, following: boolean) => void
  disabled?: boolean
  grouped?: boolean
}

function RuleEditor(props: RuleEditorProps) {
  const { t } = useTranslation()
  const update = <K extends keyof RuleDraft>(key: K, value: RuleDraft[K]) =>
    props.onChange({ ...props.draft, [key]: value })
  const isDisabled = (key: RuleFieldKey) =>
    props.disabled || props.following?.[key] === true

  function ruleField(
    key: RuleFieldKey,
    label: string,
    content: ReactNode,
    description?: string
  ) {
    const following = props.following?.[key]
    return (
      <Field key={key} data-disabled={isDisabled(key)}>
        <div className='flex min-h-5 items-center justify-between gap-3'>
          <FieldLabel>{label}</FieldLabel>
          {following !== undefined ? (
            <label className='text-muted-foreground flex items-center gap-2 text-xs'>
              {t('Follow global')}
              <Switch
                checked={following}
                onCheckedChange={(checked) =>
                  props.onFollowingChange?.(key, checked)
                }
                disabled={props.disabled}
              />
            </label>
          ) : null}
        </div>
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
    const following = props.following?.[key]
    return (
      <div key={key} className='flex items-center justify-between gap-3'>
        <SettingsSwitchField
          checked={checked}
          onCheckedChange={onCheckedChange}
          label={label}
          description={description}
          disabled={isDisabled(key)}
          aria-label={label}
          className='min-w-0 flex-1'
        />
        {following !== undefined ? (
          <label className='text-muted-foreground flex shrink-0 items-center gap-2 text-xs'>
            {t('Follow global')}
            <Switch
              checked={following}
              onCheckedChange={(next) => props.onFollowingChange?.(key, next)}
              disabled={props.disabled}
            />
          </label>
        ) : null}
      </div>
    )
  }

  const enabledField = ruleField(
    'enabled',
    t('Enable referral cashback'),
    <Switch
      checked={props.draft.enabled}
      onCheckedChange={(checked) => update('enabled', checked)}
      disabled={isDisabled('enabled')}
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
      disabled={isDisabled('registration_reward_trigger')}
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
        disabled={isDisabled('inviter_reward_quota')}
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
        disabled={isDisabled('invitee_reward_quota')}
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
      disabled={isDisabled('cashback_frequency')}
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
      disabled={isDisabled('reward_mode')}
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
              disabled={isDisabled('reward_rate_bps')}
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
              disabled={isDisabled('fixed_reward_quota')}
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
        disabled={isDisabled('minimum_topup_cents')}
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
            disabled={isDisabled('maximum_reward_quota')}
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
        disabled={isDisabled('hold_seconds')}
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
        disabled={isDisabled('minimum_transfer_quota')}
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
            disabled={isDisabled('unlimited_reward')}
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
            disabled={isDisabled('show_invitee_topups')}
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
  const queryClient = useQueryClient()
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [rowDrafts, setRowDrafts] = useState<
    Record<number, AffiliateUserRowDraft>
  >({})
  const globalSettingsQuery = useQuery({
    queryKey: ['admin', 'affiliate', 'settings'],
    queryFn: getAffiliateSettings,
    select: (response) => response.data,
  })
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

  const items = useMemo(() => query.data?.items ?? [], [query.data])
  const total = query.data?.total ?? 0
  const rewardMode =
    globalSettingsQuery.data?.reward_mode ??
    items[0]?.global_rule.reward_mode ??
    'percentage'

  useEffect(() => {
    if (!query.data) return
    const next: Record<number, AffiliateUserRowDraft> = {}
    for (const view of items) {
      next[view.user_id] = affiliateViewToRowDraft(view)
    }
    setRowDrafts(next)
  }, [items, query.data])

  function updateRowConfig(
    view: AffiliateUserOverrideView,
    key: AffiliateConfigFieldKey,
    value: string
  ) {
    const draftKey = affiliateConfigDraftKeys[key]
    setRowDrafts((current) => {
      const row = current[view.user_id] ?? affiliateViewToRowDraft(view)
      return {
        ...current,
        [view.user_id]: {
          ...row,
          config: { ...row.config, [draftKey]: value },
          following: { ...row.following, [key]: false },
        },
      }
    })
  }

  function updateRowFollowing(
    view: AffiliateUserOverrideView,
    key: AffiliateConfigFieldKey,
    following: boolean
  ) {
    const draftKey = affiliateConfigDraftKeys[key]
    const globalConfig = settingsToConfigDraft(view.global_rule)
    setRowDrafts((current) => {
      const row = current[view.user_id] ?? affiliateViewToRowDraft(view)
      return {
        ...current,
        [view.user_id]: {
          ...row,
          config: following
            ? { ...row.config, [draftKey]: globalConfig[draftKey] }
            : row.config,
          following: { ...row.following, [key]: following },
        },
      }
    })
  }

  function updateRowNote(view: AffiliateUserOverrideView, note: string) {
    setRowDrafts((current) => {
      const row = current[view.user_id] ?? affiliateViewToRowDraft(view)
      return { ...current, [view.user_id]: { ...row, note } }
    })
  }

  const saveMutation = useMutation({
    mutationFn: (request: {
      userId: number
      override: AffiliateUserOverride
    }) => updateAffiliateUserOverride(request.userId, request.override),
    onSuccess: async (response, request) => {
      if (!response.success || !response.data) {
        toast.error(response.message || t('Failed to save settings'))
        return
      }
      const savedView = response.data
      setRowDrafts((current) => ({
        ...current,
        [request.userId]: affiliateViewToRowDraft(savedView),
      }))
      toast.success(t('User-specific cashback settings saved'))
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'affiliate', 'user-overrides'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Save failed'))
    },
  })

  const resetMutation = useMutation({
    mutationFn: (userId: number) => deleteAffiliateUserOverride(userId),
    onSuccess: async (response, userId) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to reset settings'))
        return
      }
      const view = items.find((item) => item.user_id === userId)
      if (view) {
        setRowDrafts((current) => ({
          ...current,
          [userId]: affiliateViewToGlobalRowDraft(view),
        }))
      }
      toast.success(t('User now follows global cashback settings'))
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'affiliate', 'user-overrides'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Reset failed'))
    },
  })

  function saveRow(view: AffiliateUserOverrideView) {
    const row = rowDrafts[view.user_id] ?? affiliateViewToRowDraft(view)
    const override = buildAffiliateConfigOverride(
      row.config,
      view.global_rule.reward_mode,
      view.global_rule.unlimited_reward,
      row.following,
      row.note
    )
    if (!override) {
      toast.error(t('Check the cashback settings and try again'))
      return
    }
    saveMutation.mutate({ userId: view.user_id, override })
  }

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
  } else {
    results = (
      <div className='min-w-0 rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className='min-w-44'>{t('User')}</TableHead>
              <TableHead className='min-w-52'>{t('Email')}</TableHead>
              <TableHead className='min-w-44'>{t('Inviter reward')}</TableHead>
              <TableHead className='min-w-44'>{t('Invitee reward')}</TableHead>
              <TableHead className='min-w-44'>
                {rewardMode === 'fixed'
                  ? t('Fixed cashback')
                  : t('Cashback rate')}
              </TableHead>
              <TableHead className='min-w-48'>
                {t('First qualifying cumulative top-up')}
              </TableHead>
              <TableHead className='min-w-48'>
                {t('Maximum cashback per invitee')}
              </TableHead>
              <TableHead className='min-w-36'>{t('Hold period')}</TableHead>
              <TableHead className='min-w-52'>
                {t('Note')} ({t('Optional')})
              </TableHead>
              <TableHead className='sticky right-0 z-30 min-w-44 bg-[var(--table-header)] text-right shadow-[-8px_0_10px_-10px_hsl(var(--foreground))]'>
                {t('Action')}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={10} className='h-36 text-center'>
                  <div className='mx-auto flex max-w-md flex-col items-center gap-1'>
                    <p className='font-medium'>
                      {keyword ? t('No matching users') : t('No users')}
                    </p>
                    <p className='text-muted-foreground text-sm'>
                      {t(
                        'No users available. Try adjusting your search or filters.'
                      )}
                    </p>
                  </div>
                </TableCell>
              </TableRow>
            ) : null}
            {items.map((view) => {
              const row =
                rowDrafts[view.user_id] ?? affiliateViewToRowDraft(view)
              const customized = hasAffiliateConfigurationOverride(view)
              const hasOverride = Boolean(view.override)
              const rowClassName =
                keyword && !customized
                  ? 'bg-muted/30 text-muted-foreground hover:bg-muted/40'
                  : undefined
              const disabled = saveMutation.isPending || resetMutation.isPending
              return (
                <TableRow key={view.user_id} className={rowClassName}>
                  <TableCell className='align-top font-medium'>
                    <div>{view.username}</div>
                    <div className='text-muted-foreground text-xs'>
                      #{view.user_id}
                    </div>
                  </TableCell>
                  <TableCell className='align-top'>
                    {view.email || '-'}
                  </TableCell>
                  <AffiliateConfigTableCell
                    value={row.config.inviterReward}
                    following={row.following.inviter_reward_quota}
                    unit={t('CNY')}
                    label={t('Inviter reward')}
                    disabled={disabled}
                    onValueChange={(value) =>
                      updateRowConfig(view, 'inviter_reward_quota', value)
                    }
                    onFollowingChange={(following) =>
                      updateRowFollowing(
                        view,
                        'inviter_reward_quota',
                        following
                      )
                    }
                  />
                  <AffiliateConfigTableCell
                    value={row.config.inviteeReward}
                    following={row.following.invitee_reward_quota}
                    unit={t('CNY')}
                    label={t('Invitee reward')}
                    disabled={disabled}
                    onValueChange={(value) =>
                      updateRowConfig(view, 'invitee_reward_quota', value)
                    }
                    onFollowingChange={(following) =>
                      updateRowFollowing(
                        view,
                        'invitee_reward_quota',
                        following
                      )
                    }
                  />
                  <AffiliateConfigTableCell
                    value={row.config.rewardValue}
                    following={row.following.reward_value}
                    unit={
                      view.global_rule.reward_mode === 'percentage'
                        ? '%'
                        : t('CNY')
                    }
                    label={
                      view.global_rule.reward_mode === 'percentage'
                        ? t('Cashback rate')
                        : t('Fixed cashback')
                    }
                    inputProps={
                      view.global_rule.reward_mode === 'percentage'
                        ? { min: 0, max: 100, step: '0.01' }
                        : { min: 0, step: '0.01' }
                    }
                    disabled={disabled}
                    onValueChange={(value) =>
                      updateRowConfig(view, 'reward_value', value)
                    }
                    onFollowingChange={(following) =>
                      updateRowFollowing(view, 'reward_value', following)
                    }
                  />
                  <AffiliateConfigTableCell
                    value={row.config.minimumTopUp}
                    following={row.following.minimum_topup_cents}
                    unit={t('CNY')}
                    label={t('First qualifying cumulative top-up')}
                    disabled={disabled}
                    onValueChange={(value) =>
                      updateRowConfig(view, 'minimum_topup_cents', value)
                    }
                    onFollowingChange={(following) =>
                      updateRowFollowing(view, 'minimum_topup_cents', following)
                    }
                  />
                  <AffiliateConfigTableCell
                    value={row.config.maximumReward}
                    following={row.following.maximum_reward_quota}
                    unit={t('CNY')}
                    label={t('Maximum cashback per invitee')}
                    inputProps={{
                      min: view.global_rule.unlimited_reward ? 0 : 0.01,
                      step: '0.01',
                    }}
                    disabled={disabled}
                    onValueChange={(value) =>
                      updateRowConfig(view, 'maximum_reward_quota', value)
                    }
                    onFollowingChange={(following) =>
                      updateRowFollowing(
                        view,
                        'maximum_reward_quota',
                        following
                      )
                    }
                  />
                  <AffiliateConfigTableCell
                    value={row.config.holdDays}
                    following={row.following.hold_seconds}
                    unit={t('Days')}
                    label={t('Hold period')}
                    inputProps={{ min: 0, max: 365, step: 1 }}
                    disabled={disabled}
                    onValueChange={(value) =>
                      updateRowConfig(view, 'hold_seconds', value)
                    }
                    onFollowingChange={(following) =>
                      updateRowFollowing(view, 'hold_seconds', following)
                    }
                  />
                  <TableCell className='align-top'>
                    <Textarea
                      aria-label={`${t('Note')} (${t('Optional')})`}
                      className='min-h-16 min-w-48 resize-y'
                      maxLength={500}
                      value={row.note}
                      placeholder={`${t('Note')} (${t('Optional')})`}
                      onChange={(event) =>
                        updateRowNote(view, event.target.value)
                      }
                      disabled={disabled}
                    />
                  </TableCell>
                  <TableCell
                    className={cn(
                      'sticky right-0 z-10 min-w-44 bg-background align-top text-right shadow-[-8px_0_10px_-10px_hsl(var(--foreground))] group-hover:[background-color:color-mix(in_oklch,var(--muted)_50%,var(--background))]',
                      rowClassName && 'bg-muted/30 group-hover:bg-muted/40'
                    )}
                  >
                    <div className='flex flex-wrap justify-end gap-2'>
                      <Button
                        type='button'
                        size='sm'
                        disabled={
                          disabled ||
                          affiliateConfigFieldKeys.every(
                            (key) => row.following[key]
                          )
                        }
                        onClick={() => saveRow(view)}
                      >
                        {saveMutation.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : null}
                        {t('Save')}
                      </Button>
                      {hasOverride ? (
                        <Button
                          type='button'
                          size='sm'
                          variant='outline'
                          disabled={disabled}
                          onClick={() => resetMutation.mutate(view.user_id)}
                          aria-label={t('Follow all global settings')}
                          title={t('Follow all global settings')}
                        >
                          <RotateCcw data-icon='inline-start' />
                          {t('Reset')}
                        </Button>
                      ) : null}
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
