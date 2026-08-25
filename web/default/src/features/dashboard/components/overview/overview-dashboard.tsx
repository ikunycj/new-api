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
import { Link, useNavigate } from '@tanstack/react-router'
import {
  ArrowRight,
  BookOpen,
  Check,
  ChevronDown,
  ChevronUp,
  FileText,
  KeyRound,
  ListChecks,
  LoaderCircle,
  MessageSquareText,
  MonitorDown,
  RadioTower,
  type LucideIcon,
} from 'lucide-react'
import { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { useChatPresets } from '@/features/chat/hooks/use-chat-presets'
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'
import { API_KEY_STATUS } from '@/features/keys/constants'
import type { ApiKey } from '@/features/keys/types'
import { completeOnboarding, getSelf } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { useDashboardContentVisibility } from '../../hooks/use-status-data'
import { AnnouncementsPanel } from './announcements-panel'
import { ApiInfoPanel } from './api-info-panel'
import { FAQPanel } from './faq-panel'
import { NewUserOnboardingDialog } from './new-user-onboarding-dialog'
import { PerformanceHealthPanel } from './performance-health-panel'
import { SummaryCards } from './summary-cards'
import { UptimePanel } from './uptime-panel'

const SETUP_GUIDE_VISIBILITY_STORAGE_KEY =
  'dashboard_overview_setup_guide_expanded:v2'
const SETUP_GUIDE_SKIPPED_STORAGE_KEY =
  'dashboard_overview_setup_guide_skipped_steps:v1'
const ONBOARDING_UI_VERSION = 1

const LazyCCSwitchDialog = lazy(() =>
  import('@/features/keys/components/dialogs/cc-switch-dialog').then(
    (module) => ({
      default: module.CCSwitchDialog,
    })
  )
)

const SETUP_GUIDE_CODE_PATTERN = [
  'const request = await client.responses.create({',
  "  model: 'gpt-4.1-mini',",
  "  input: 'Start routing traffic',",
  '})',
  '',
  'if (request.output_text) {',
  '  console.log(request.output_text)',
  '}',
].join('\n')

type DashboardActionPath =
  | '/keys'
  | '/wallet'
  | '/playground'
  | '/channels'
  | '/usage-logs'
  | '/pricing'

interface StartStep {
  title: string
  description: string
  to: DashboardActionPath
  completed: boolean
}

type StartStepState = 'completed' | 'current' | 'pending'

interface QuickAction {
  title: string
  description: string
  to: DashboardActionPath
  icon: LucideIcon
  adminOnly?: boolean
}

function getSavedSetupGuideExpanded(userId?: number): boolean | null {
  if (typeof window === 'undefined' || !userId) return null
  const saved = window.localStorage.getItem(
    `${SETUP_GUIDE_VISIBILITY_STORAGE_KEY}:${userId}`
  )
  if (saved === 'expanded') return true
  if (saved === 'collapsed') return false
  return null
}

function saveSetupGuideExpanded(
  userId: number | undefined,
  expanded: boolean
): void {
  if (typeof window === 'undefined' || !userId) return
  window.localStorage.setItem(
    `${SETUP_GUIDE_VISIBILITY_STORAGE_KEY}:${userId}`,
    expanded ? 'expanded' : 'collapsed'
  )
}

function getSavedSkippedSetupSteps(userId?: number): number[] {
  if (typeof window === 'undefined' || !userId) return []
  const saved = window.localStorage.getItem(
    `${SETUP_GUIDE_SKIPPED_STORAGE_KEY}:${ONBOARDING_UI_VERSION}:${userId}`
  )
  if (!saved) return []

  try {
    const parsed = JSON.parse(saved)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (value): value is number =>
        typeof value === 'number' && Number.isInteger(value) && value >= 0
    )
  } catch {
    return []
  }
}

function saveSkippedSetupSteps(userId: number | undefined, steps: number[]) {
  if (typeof window === 'undefined' || !userId) return
  window.localStorage.setItem(
    `${SETUP_GUIDE_SKIPPED_STORAGE_KEY}:${ONBOARDING_UI_VERSION}:${userId}`,
    JSON.stringify(steps)
  )
}

function getAnyKey(keys: ApiKey[]): ApiKey | null {
  return keys[0] ?? null
}

function getPreferredEnabledKey(keys: ApiKey[]): ApiKey | null {
  return keys.find((item) => item.status === API_KEY_STATUS.ENABLED) ?? null
}

function SetupGuideBackdrop(props: { compact?: boolean }) {
  return (
    <>
      <div
        className={cn(
          'pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_48%_120%_at_78%_0%,color-mix(in_oklch,var(--overview-accent-1)_14%,transparent)_0%,transparent_62%),linear-gradient(112deg,color-mix(in_oklch,var(--card)_94%,var(--overview-accent-2)_6%)_0%,color-mix(in_oklch,var(--card)_94%,var(--overview-accent-3)_6%)_48%,color-mix(in_oklch,var(--background)_90%,var(--overview-accent-1)_10%)_100%)] dark:opacity-60',
          props.compact
            ? '[mask-image:linear-gradient(90deg,black_0%,black_48%,transparent_74%)] opacity-55'
            : 'opacity-85'
        )}
        aria-hidden='true'
      />
      <div
        className={cn(
          'text-foreground/5 dark:text-foreground/8 pointer-events-none absolute inset-y-0 right-0 hidden overflow-hidden font-mono sm:block',
          props.compact ? 'w-1/2 opacity-45' : 'w-[58%] opacity-75'
        )}
        aria-hidden='true'
      >
        <pre
          className={cn(
            'absolute right-3 [mask-image:linear-gradient(90deg,transparent_0%,black_30%,black_82%,transparent_100%)] text-right tracking-[0.38em] whitespace-pre',
            props.compact
              ? '-top-6 text-[9px] leading-4'
              : 'top-1 text-[11px] leading-5'
          )}
        >
          {SETUP_GUIDE_CODE_PATTERN}
        </pre>
      </div>
      <div
        className='from-background/35 to-background/70 dark:from-background/20 dark:to-background/80 pointer-events-none absolute inset-0 bg-linear-to-b via-transparent'
        aria-hidden='true'
      />
    </>
  )
}

function StartStepItem(props: {
  step: StartStep
  index: number
  state: StartStepState
  onSkip: () => void
  onOpen: () => void
}) {
  const { t } = useTranslation()
  const isCurrent = props.state === 'current'
  const isCompleted = props.state === 'completed'
  const isRequestStep = props.index === 2
  let stateLabel = t('Pending')
  if (isCompleted) stateLabel = t('Completed')
  if (isCurrent) stateLabel = t('Processing')

  const stepContent = <StepContent index={props.index} step={props.step} />
  const stepControlClassName =
    'focus-visible:ring-ring flex min-w-0 flex-1 items-center gap-3 rounded-lg text-left outline-none focus-visible:ring-2'
  const arrowClassName = cn(
    'size-4',
    isCompleted ? 'text-success' : 'text-muted-foreground'
  )
  const arrowControlClassName =
    'hover:bg-muted focus-visible:ring-ring flex size-8 shrink-0 items-center justify-center rounded-md outline-none focus-visible:ring-2'

  return (
    <li className='relative pb-2.5 last:pb-0'>
      <div
        className={cn(
          'flex min-h-16 items-center gap-2 rounded-xl border px-3 py-2.5 shadow-xs transition-transform duration-300 sm:gap-3',
          isCompleted && 'border-success/35 bg-success/12',
          isCurrent &&
            'scale-[1.03] border-warning/90 bg-warning/20 shadow-[0_0_0_4px_color-mix(in_oklch,var(--warning)_28%,transparent)] motion-safe:animate-[pulse_2s_ease-in-out_infinite]',
          !isCompleted && !isCurrent && 'border-border bg-background'
        )}
      >
        <span className='sr-only'>{stateLabel}</span>
        {isRequestStep ? (
          <button
            type='button'
            onClick={props.onOpen}
            className={stepControlClassName}
            aria-current={isCurrent ? 'step' : undefined}
          >
            {stepContent}
          </button>
        ) : (
          <Link
            to={props.step.to}
            className={stepControlClassName}
            aria-current={isCurrent ? 'step' : undefined}
          >
            {stepContent}
          </Link>
        )}
        <div className='flex shrink-0 items-center gap-1'>
          {isCurrent ? (
            <Button
              type='button'
              variant='ghost'
              size='sm'
              className='h-7 shrink-0 px-2 text-xs'
              onClick={props.onSkip}
            >
              {t('Skip')}
            </Button>
          ) : null}
          {isRequestStep ? (
            <button
              type='button'
              onClick={props.onOpen}
              className={arrowControlClassName}
              aria-label={props.step.title}
            >
              <ArrowRight className={arrowClassName} aria-hidden='true' />
            </button>
          ) : (
            <Link
              to={props.step.to}
              className={arrowControlClassName}
              aria-label={props.step.title}
            >
              <ArrowRight className={arrowClassName} aria-hidden='true' />
            </Link>
          )}
        </div>
      </div>
    </li>
  )
}

function StepContent(props: { index: number; step: StartStep }) {
  return (
    <span className='flex min-w-0 items-center gap-3'>
      <span className='text-muted-foreground w-8 shrink-0 font-mono text-xs font-semibold tabular-nums'>
        {String(props.index + 1).padStart(2, '0')}
      </span>
      <span className='flex min-w-0 flex-col gap-0.5'>
        <span className='truncate text-sm font-medium'>{props.step.title}</span>
        <span className='text-muted-foreground line-clamp-1 text-xs'>
          {props.step.description}
        </span>
      </span>
    </span>
  )
}

function StartRequestChoiceDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onWebsiteChat: () => void
  onCCSwitch: () => void
  isResolvingCCSwitchKey: boolean
}) {
  const { t } = useTranslation()

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Choose how to start')}
      description={t('Pick the way you want to send your first request.')}
      contentClassName='sm:max-w-2xl'
      bodyClassName='p-0'
    >
      <div className='grid gap-3 sm:grid-cols-2'>
        <button
          type='button'
          onClick={props.onWebsiteChat}
          className='group border-success/30 bg-success/5 hover:border-success/60 hover:bg-success/10 focus-visible:ring-ring flex min-h-48 flex-col items-center justify-between gap-5 rounded-xl border p-5 text-center transition-colors focus-visible:ring-2 focus-visible:outline-none'
        >
          <MessageSquareText
            className='text-success size-10'
            aria-hidden='true'
          />
          <span className='flex flex-col gap-1'>
            <span className='text-sm font-semibold'>
              {t('Start chatting directly on the website')}
            </span>
            <span className='text-muted-foreground text-xs'>
              {t('Use the built-in playground to try a live conversation.')}
            </span>
          </span>
          <span className='text-success text-xs font-medium'>
            {t('Continue')}
          </span>
        </button>

        <button
          type='button'
          onClick={props.onCCSwitch}
          disabled={props.isResolvingCCSwitchKey}
          aria-busy={props.isResolvingCCSwitchKey}
          className='group bg-background hover:border-primary/50 hover:bg-muted/50 focus-visible:ring-ring flex min-h-48 flex-col items-center justify-between gap-5 rounded-xl border p-5 text-center transition-colors focus-visible:ring-2 focus-visible:outline-none disabled:cursor-wait disabled:opacity-70'
        >
          <MonitorDown className='text-primary size-10' aria-hidden='true' />
          <span className='flex flex-col gap-1'>
            <span className='text-sm font-semibold'>
              {t('Import your Agent client with CC Switch')}
            </span>
            <span className='text-muted-foreground text-xs'>
              {t(
                'Bring your API key and model settings into a desktop client.'
              )}
            </span>
          </span>
          <span className='text-primary text-xs font-medium'>
            {props.isResolvingCCSwitchKey ? (
              <LoaderCircle
                className='mx-auto size-4 animate-spin'
                aria-label={t('Loading')}
              />
            ) : (
              t('Continue')
            )}
          </span>
        </button>
      </div>
    </Dialog>
  )
}

function CCSwitchDialogFallback(props: {
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()

  return (
    <Dialog open onOpenChange={props.onOpenChange} title={t('Loading')}>
      <div
        className='text-muted-foreground flex items-center justify-center gap-2 p-6 text-sm'
        role='status'
      >
        <LoaderCircle className='size-4 animate-spin' aria-hidden='true' />
        {t('Loading')}
      </div>
    </Dialog>
  )
}

function QuickActionItem(props: { action: QuickAction }) {
  const Icon = props.action.icon

  return (
    <Button
      variant='outline'
      className='h-auto justify-start rounded-xl px-3 py-3 text-left'
      render={<Link to={props.action.to} />}
    >
      <span className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-lg'>
        <Icon className='size-4' aria-hidden='true' />
      </span>
      <span className='flex min-w-0 flex-1 flex-col gap-0.5'>
        <span className='truncate text-sm font-medium'>
          {props.action.title}
        </span>
        <span className='text-muted-foreground line-clamp-2 text-xs leading-relaxed'>
          {props.action.description}
        </span>
      </span>
    </Button>
  )
}

function CompactQuickAction(props: { action: QuickAction }) {
  const Icon = props.action.icon

  return (
    <Button
      variant='outline'
      size='sm'
      className='bg-background/70 h-8 min-w-24 gap-1.5 px-2.5'
      render={<Link to={props.action.to} />}
    >
      <Icon data-icon='inline-start' />
      <span>{props.action.title}</span>
    </Button>
  )
}

export function OverviewDashboard() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.auth.user)
  const setUser = useAuthStore((state) => state.auth.setUser)
  const { serverAddress } = useChatPresets()
  const {
    apiInfo: showApiInfoPanel,
    announcements: showAnnouncementsPanel,
    faq: showFAQPanel,
    uptimeKuma: showUptimePanel,
  } = useDashboardContentVisibility()
  const [manualSetupGuideExpanded, setManualSetupGuideExpanded] = useState<
    boolean | null
  >(() => getSavedSetupGuideExpanded(user?.id))
  const [skippedSteps, setSkippedSteps] = useState<number[]>(() =>
    getSavedSkippedSetupSteps(user?.id)
  )
  const [isCompletingOnboarding, setIsCompletingOnboarding] = useState(false)
  const [requestChoiceOpen, setRequestChoiceOpen] = useState(false)
  const [ccSwitchOpen, setCCSwitchOpen] = useState(false)
  const [ccSwitchKey, setCCSwitchKey] = useState('')
  const [isResolvingCCSwitchKey, setIsResolvingCCSwitchKey] = useState(false)
  const requestChoiceOpenRef = useRef(false)
  const selfRefreshInFlightRef = useRef(false)
  const onboardingMutationGenerationRef = useRef(0)

  useEffect(() => {
    setManualSetupGuideExpanded(getSavedSetupGuideExpanded(user?.id))
    setSkippedSteps(getSavedSkippedSetupSteps(user?.id))
  }, [user?.id])

  useEffect(() => {
    if (!user?.id) return

    let disposed = false

    const refreshSelf = async () => {
      if (disposed || selfRefreshInFlightRef.current) return

      const currentUser = useAuthStore.getState().auth.user
      if (!currentUser || currentUser.id !== user.id) return

      selfRefreshInFlightRef.current = true
      const refreshGeneration = onboardingMutationGenerationRef.current
      try {
        const result = await getSelf()
        if (disposed || !result?.success || !result.data) return

        const latestUser = useAuthStore.getState().auth.user
        if (!latestUser || latestUser.id !== user.id) return

        const refreshedUser = { ...latestUser, ...result.data }
        if (onboardingMutationGenerationRef.current !== refreshGeneration) {
          refreshedUser.onboarding_required = latestUser.onboarding_required
          refreshedUser.onboarding_version = latestUser.onboarding_version
        }
        const localRequestCount = Number(latestUser.request_count ?? 0)
        const serverRequestCount = Number(result.data.request_count ?? 0)
        if (
          Number.isFinite(localRequestCount) &&
          Number.isFinite(serverRequestCount)
        ) {
          // Server-side batch updates may briefly lag the successful request.
          refreshedUser.request_count = Math.max(
            localRequestCount,
            serverRequestCount
          )
        }
        setUser(refreshedUser)
      } catch {
        // Keep the cached snapshot when the refresh is unavailable.
      } finally {
        selfRefreshInFlightRef.current = false
      }
    }

    void refreshSelf()
    const handleWindowFocus = () => void refreshSelf()
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') void refreshSelf()
    }
    window.addEventListener('focus', handleWindowFocus)
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      disposed = true
      window.removeEventListener('focus', handleWindowFocus)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [setUser, user?.id])

  const requestCount = Number(user?.request_count ?? 0)
  const remainQuota = Number(user?.quota ?? 0)
  const usedQuota = Number(user?.used_quota ?? 0)
  const isAdmin = Boolean(user?.role && user.role >= ROLE.ADMIN)

  const apiKeysQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'api-keys', user?.id],
    queryFn: async () => {
      const result = await getApiKeys({ p: 1, size: 100 })
      return result.success ? (result.data?.items ?? []) : []
    },
    enabled: Boolean(user?.id),
    staleTime: 0,
    refetchOnMount: 'always',
  })

  const anyKey = useMemo(
    () => getAnyKey(apiKeysQuery.data ?? []),
    [apiKeysQuery.data]
  )
  const preferredKey = useMemo(
    () => getPreferredEnabledKey(apiKeysQuery.data ?? []),
    [apiKeysQuery.data]
  )

  const startSteps = useMemo<StartStep[]>(
    () => [
      {
        title: t('Create API Key'),
        description: t('Create a key for your app or service'),
        to: '/keys',
        completed: Boolean(anyKey),
      },
      {
        title: t('Add credits'),
        description: t('Keep enough balance before production traffic'),
        to: '/wallet',
        completed: remainQuota > 0 || usedQuota > 0,
      },
      {
        title: t('Send a request'),
        description: t('Verify routing with Playground or your client'),
        to: '/playground',
        completed: requestCount > 0,
      },
    ],
    [anyKey, remainQuota, requestCount, t, usedQuota]
  )

  const quickActions = useMemo<QuickAction[]>(
    () => [
      {
        title: t('API Keys'),
        description: t('Create a key for your app or service'),
        to: '/keys',
        icon: KeyRound,
      },
      {
        title: t('Channels'),
        description: t('Configure upstream providers and routing.'),
        to: '/channels',
        icon: RadioTower,
        adminOnly: true,
      },
      {
        title: t('Usage Logs'),
        description: t('Inspect requests, errors, and billing details'),
        to: '/usage-logs',
        icon: FileText,
      },
      {
        title: t('Pricing'),
        description: t('Review model rates before scaling traffic'),
        to: '/pricing',
        icon: BookOpen,
      },
    ],
    [t]
  )

  const visibleQuickActions = useMemo(
    () => quickActions.filter((action) => !action.adminOnly || isAdmin),
    [isAdmin, quickActions]
  )

  const setupStatusReady = apiKeysQuery.isFetched && Boolean(user)
  const stepStates = useMemo<StartStepState[]>(() => {
    if (!setupStatusReady) return startSteps.map(() => 'pending')

    const currentStepIndex = startSteps.findIndex(
      (step, index) => !step.completed && !skippedSteps.includes(index)
    )

    return startSteps.map((step, index) => {
      if (step.completed || skippedSteps.includes(index)) return 'completed'
      if (index === currentStepIndex) return 'current'
      return 'pending'
    })
  }, [setupStatusReady, skippedSteps, startSteps])
  const completedStepCount = stepStates.filter(
    (state) => state === 'completed'
  ).length
  const setupComplete = completedStepCount === stepStates.length
  const onboardingRequired = user?.onboarding_required === true
  const setupGuideExpanded =
    manualSetupGuideExpanded ?? (setupStatusReady && !setupComplete)
  const showLeftContentPanels =
    isAdmin || showApiInfoPanel || showAnnouncementsPanel || showFAQPanel
  const showContentPanels = showLeftContentPanels || showUptimePanel

  const handleSetupGuideToggle = () => {
    const nextExpanded = !setupGuideExpanded
    setManualSetupGuideExpanded(nextExpanded)
    saveSetupGuideExpanded(user?.id, nextExpanded)
  }

  const handleCompleteOnboarding = async (expandGuide: boolean) => {
    if (isCompletingOnboarding || user?.onboarding_required !== true) return

    setIsCompletingOnboarding(true)
    onboardingMutationGenerationRef.current += 1
    try {
      const result = await completeOnboarding()
      if (!result.success || !result.data) {
        toast.error(result.message || t('Request failed'))
        return
      }

      setManualSetupGuideExpanded(expandGuide)
      saveSetupGuideExpanded(user.id, expandGuide)
      const latestUser = useAuthStore.getState().auth.user
      if (!latestUser || latestUser.id !== user.id) return
      setUser({
        ...latestUser,
        onboarding_required: result.data.onboarding_required,
        onboarding_version: result.data.onboarding_version,
      })
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setIsCompletingOnboarding(false)
    }
  }

  const handleSkipStep = (index: number) => {
    if (skippedSteps.includes(index)) return

    const nextSkippedSteps = [...skippedSteps, index]
    setSkippedSteps(nextSkippedSteps)
    saveSkippedSetupSteps(user?.id, nextSkippedSteps)

    const allStepsResolved = startSteps.every(
      (step, stepIndex) =>
        step.completed || nextSkippedSteps.includes(stepIndex)
    )
    if (allStepsResolved && user?.onboarding_required === true) {
      void handleCompleteOnboarding(false)
    }
  }

  const handleWebsiteChat = () => {
    requestChoiceOpenRef.current = false
    setRequestChoiceOpen(false)
    void navigate({ to: '/playground' })
  }

  const handleOpenRequestChoice = () => {
    requestChoiceOpenRef.current = true
    setRequestChoiceOpen(true)
  }

  const handleRequestChoiceOpenChange = (open: boolean) => {
    requestChoiceOpenRef.current = open
    setRequestChoiceOpen(open)
  }

  const handleCCSwitch = async () => {
    if (isResolvingCCSwitchKey) return

    if (!preferredKey?.id) {
      toast.error(
        anyKey
          ? t(
              'Unable to prepare chat link. Please ensure you have an enabled API key.'
            )
          : t(
              'No API keys available. Create your first API key to get started.'
            )
      )
      return
    }

    setIsResolvingCCSwitchKey(true)
    try {
      const result = await fetchTokenKey(preferredKey.id)
      const tokenKey = result.success ? (result.data?.key ?? '') : ''
      if (!tokenKey) {
        toast.error(
          result.message || t('Unable to resolve the selected API key.')
        )
        return
      }

      if (!requestChoiceOpenRef.current) return

      setCCSwitchKey(tokenKey.startsWith('sk-') ? tokenKey : `sk-${tokenKey}`)
      requestChoiceOpenRef.current = false
      setRequestChoiceOpen(false)
      setCCSwitchOpen(true)
    } catch {
      toast.error(t('Unable to resolve the selected API key.'))
    } finally {
      setIsResolvingCCSwitchKey(false)
    }
  }

  const normalizedServerAddress = serverAddress.trim().replace(/\/+$/, '')
  const apiBaseUrl = normalizedServerAddress
    ? `${normalizedServerAddress}/v1`
    : ''
  const handleCCSwitchOpenChange = (open: boolean) => {
    setCCSwitchOpen(open)
    if (!open) setCCSwitchKey('')
  }

  return (
    <div className='flex flex-col gap-4'>
      <NewUserOnboardingDialog
        open={onboardingRequired}
        steps={startSteps}
        isPending={isCompletingOnboarding}
        onStart={() => void handleCompleteOnboarding(true)}
        onSkip={() => void handleCompleteOnboarding(false)}
      />

      {setupGuideExpanded ? (
        <CardStaggerContainer className='grid items-stretch gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]'>
          <CardStaggerItem className='bg-card h-full overflow-hidden rounded-2xl border shadow-xs'>
            <div className='relative h-full overflow-hidden p-4 sm:p-5'>
              <SetupGuideBackdrop />
              <div className='relative flex min-w-0 flex-col gap-5'>
                <div className='flex flex-wrap items-start justify-between gap-3'>
                  <div className='flex max-w-2xl flex-col gap-1'>
                    <div className='text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wider uppercase'>
                      <ListChecks className='size-3.5' aria-hidden='true' />
                      {t('Get started')}
                    </div>
                    <h3 className='text-xl font-semibold tracking-tight sm:text-2xl'>
                      {t('Build on your API gateway in minutes')}
                    </h3>
                    <p className='text-muted-foreground max-w-xl text-sm leading-relaxed'>
                      {t(
                        'A focused home for keys, balance, routing, and service health.'
                      )}
                    </p>
                  </div>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={handleSetupGuideToggle}
                    >
                      <ChevronUp data-icon='inline-start' />
                      {t('Hide setup guide')}
                    </Button>
                    <Button size='sm' render={<Link to='/keys' />}>
                      <KeyRound data-icon='inline-start' />
                      {t('Create API Key')}
                    </Button>
                  </div>
                </div>

                <ol className='bg-background/45 rounded-2xl border p-2 backdrop-blur'>
                  {startSteps.map((step, index) => (
                    <StartStepItem
                      key={step.title}
                      step={step}
                      index={index}
                      state={stepStates[index]}
                      onSkip={() => handleSkipStep(index)}
                      onOpen={handleOpenRequestChoice}
                    />
                  ))}
                </ol>
              </div>
            </div>
          </CardStaggerItem>

          <CardStaggerItem className='bg-card h-full rounded-2xl border p-4 shadow-xs sm:p-5'>
            <div className='flex h-full flex-col gap-4'>
              <div className='flex flex-col gap-1'>
                <div className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                  {t('Recommended actions')}
                </div>
                <h3 className='text-lg font-semibold tracking-tight'>
                  {t('Keep the platform ready')}
                </h3>
              </div>
              <div className='grid gap-2'>
                {visibleQuickActions.map((action) => (
                  <QuickActionItem key={action.title} action={action} />
                ))}
              </div>
            </div>
          </CardStaggerItem>
        </CardStaggerContainer>
      ) : (
        <CardStaggerContainer>
          <CardStaggerItem className='bg-card overflow-hidden rounded-2xl border shadow-xs'>
            <div className='relative overflow-hidden px-4 py-3 sm:px-5'>
              <SetupGuideBackdrop compact />
              <div className='relative flex flex-wrap items-center justify-between gap-3'>
                <div className='flex min-w-0 items-center gap-3'>
                  <span className='bg-background/70 flex size-9 shrink-0 items-center justify-center rounded-xl border shadow-xs'>
                    <Check className='text-success size-4' aria-hidden='true' />
                  </span>
                  <div className='min-w-0'>
                    <div className='flex items-center gap-2'>
                      <h3 className='truncate text-sm font-semibold'>
                        {setupComplete
                          ? t('Setup guide complete')
                          : t('Setup guide')}
                      </h3>
                      <span className='text-muted-foreground bg-background/60 rounded-md border px-2 py-0.5 text-xs'>
                        {t('Setup progress: {{completed}}/{{total}}', {
                          completed: completedStepCount,
                          total: startSteps.length,
                        })}
                      </span>
                    </div>
                    <p className='text-muted-foreground line-clamp-1 text-xs'>
                      {setupComplete
                        ? t(
                            'Your setup guide is collapsed so usage stays in focus.'
                          )
                        : t('Setup guide is collapsed. Expand it anytime.')}
                    </p>
                  </div>
                </div>

                <div className='flex flex-wrap items-center gap-2'>
                  {visibleQuickActions.map((action) => (
                    <CompactQuickAction key={action.title} action={action} />
                  ))}
                  <Button
                    variant='outline'
                    size='sm'
                    className='bg-background/70 h-8 min-w-28'
                    onClick={handleSetupGuideToggle}
                  >
                    <ChevronDown data-icon='inline-start' />
                    {t('Show setup guide')}
                  </Button>
                </div>
              </div>
            </div>
          </CardStaggerItem>
        </CardStaggerContainer>
      )}

      <StartRequestChoiceDialog
        open={requestChoiceOpen}
        onOpenChange={handleRequestChoiceOpenChange}
        onWebsiteChat={handleWebsiteChat}
        onCCSwitch={() => void handleCCSwitch()}
        isResolvingCCSwitchKey={isResolvingCCSwitchKey}
      />

      {ccSwitchOpen && (
        <Suspense
          fallback={
            <CCSwitchDialogFallback onOpenChange={handleCCSwitchOpenChange} />
          }
        >
          <LazyCCSwitchDialog
            open={ccSwitchOpen}
            onOpenChange={handleCCSwitchOpenChange}
            apiKey={preferredKey}
            apiBaseUrl={apiBaseUrl}
            serverAddress={normalizedServerAddress}
            tokenKey={ccSwitchKey}
          />
        </Suspense>
      )}

      <SummaryCards />

      {showContentPanels && (
        <CardStaggerContainer
          className={cn(
            'grid grid-cols-1 gap-4',
            showLeftContentPanels &&
              showUptimePanel &&
              'xl:grid-cols-[minmax(0,1fr)_22rem]'
          )}
        >
          {showLeftContentPanels && (
            <div
              className={cn(
                'grid min-w-0 grid-cols-1 gap-4',
                (showApiInfoPanel || showAnnouncementsPanel || showFAQPanel) &&
                  'lg:grid-cols-2'
              )}
            >
              {isAdmin && (
                <CardStaggerItem className='lg:col-span-2'>
                  <PerformanceHealthPanel />
                </CardStaggerItem>
              )}
              {showApiInfoPanel && (
                <CardStaggerItem>
                  <ApiInfoPanel />
                </CardStaggerItem>
              )}
              {showAnnouncementsPanel && (
                <CardStaggerItem>
                  <AnnouncementsPanel />
                </CardStaggerItem>
              )}
              {showFAQPanel && (
                <CardStaggerItem>
                  <FAQPanel />
                </CardStaggerItem>
              )}
            </div>
          )}
          {showUptimePanel && (
            <CardStaggerItem>
              <UptimePanel />
            </CardStaggerItem>
          )}
        </CardStaggerContainer>
      )}
    </div>
  )
}
