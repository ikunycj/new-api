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
import { Radio } from '@base-ui/react/radio'
import {
  AlertCircleIcon,
  ArrowDown01Icon,
  ArrowUpRight01Icon,
  InformationCircleIcon,
  ReloadIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { skipToken, useQuery } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { RadioGroup } from '@/components/ui/radio-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { useStatus } from '@/hooks/use-status'
import { getUserGroups } from '@/lib/api'
import { getPreferredModelOrder } from '@/lib/model-preferences'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { authorizeCCSwitchUsage, type ApiKeyModelsResult } from '../../api'
import { useApiKeyModelCatalog } from '../../hooks/use-api-key-model-catalog'
import {
  buildCCSwitchImportUrl,
  getCCSwitchUsageBaseUrl,
  resolveCCSwitchDefaultName,
} from '../../lib/cc-switch-import'
import { selectSuggestedModel } from '../../lib/cc-switch-model-selection'
import {
  CC_SWITCH_APPS,
  type CCSwitchApp,
  type CCSwitchModelField,
} from '../../lib/model-catalog'
import type { ApiKey } from '../../types'
import {
  ApiKeyModelCombobox,
  type ApiKeyModelSelectionSource,
} from '../api-key-model-combobox'
import { CCSwitchAppIcon } from '../cc-switch-app-icon'

const CCSWITCH_WEBSITE = 'https://ccswitch.io/'
const APP_IDS = Object.keys(CC_SWITCH_APPS) as CCSwitchApp[]

type ModelSelection = {
  source: ApiKeyModelSelectionSource
  value: string
}

type AppDraft = {
  models: Partial<Record<CCSwitchModelField, ModelSelection>>
  name: string
}

type CCSwitchDialogProps = {
  apiBaseUrl: string
  apiKey: ApiKey | null
  onOpenChange: (open: boolean) => void
  open: boolean
  serverAddress: string
  tokenKey: string
}

function getModelFieldLabel(t: TFunction, field: CCSwitchModelField): string {
  switch (field) {
    case 'model':
      return t('Primary Model')
    case 'haikuModel':
      return t('Haiku Model')
    case 'sonnetModel':
      return t('Sonnet Model')
    case 'opusModel':
      return t('Opus Model')
  }
}

function getRoutingLabel(
  t: TFunction,
  apiKey: ApiKey | null,
  groupLabels: Record<string, string>
): string {
  if (!apiKey) return t('Not available')
  if (apiKey.group_candidates.length > 0) {
    return apiKey.group_candidates
      .map((group) => groupLabels[group] || group)
      .join(' -> ')
  }
  if (apiKey.group === 'auto') return t('System routing')
  if (apiKey.group) return groupLabels[apiKey.group] || apiKey.group
  return t('User group')
}

export function CCSwitchDialog(props: CCSwitchDialogProps) {
  const resetKey = `${props.apiKey?.id ?? 'none'}:${props.apiKey?.name ?? 'unnamed'}:${props.open ? 'open' : 'closed'}`
  return <CCSwitchDialogContent key={resetKey} {...props} />
}

function CCSwitchDialogContent(props: CCSwitchDialogProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const preferredModels = useSystemConfigStore(
    (state) => state.config.preferredModels
  )
  const [app, setApp] = useState<CCSwitchApp>('claude')
  const [drafts, setDrafts] = useState<Partial<Record<CCSwitchApp, AppDraft>>>(
    {}
  )
  const [launchState, setLaunchState] = useState<'idle' | 'opening' | 'help'>(
    'idle'
  )
  const [primaryModelError, setPrimaryModelError] = useState(false)
  const primaryModelRef = useRef<HTMLInputElement>(null)
  const launchCleanupRef = useRef<() => void>(() => {})
  const normalizedServerAddress = props.serverAddress.trim().replace(/\/+$/, '')
  const isConfigurationReady = Boolean(
    props.apiKey &&
    props.tokenKey &&
    normalizedServerAddress &&
    props.apiBaseUrl.trim()
  )
  const supportsUsageQuery = Boolean(
    getCCSwitchUsageBaseUrl(normalizedServerAddress)
  )
  const usageTokenId =
    props.open && isConfigurationReady && supportsUsageQuery
      ? props.apiKey?.id
      : undefined
  const usageAuthorization = useQuery({
    queryKey: ['cc-switch-usage-authorization', usageTokenId],
    queryFn: usageTokenId
      ? ({ signal }) => authorizeCCSwitchUsage(usageTokenId, signal)
      : skipToken,
    // Keep the credential in memory for this open dialog only.
    gcTime: 0,
    staleTime: Infinity,
    retry: false,
  })
  const canImport =
    isConfigurationReady && (!supportsUsageQuery || !!usageAuthorization.data)
  const modelsQuery = useApiKeyModelCatalog({
    apiBaseUrl: props.apiBaseUrl,
    apiKey: props.apiKey,
    enabled: props.open && isConfigurationReady,
    tokenKey: props.tokenKey,
  })
  const groupsQuery = useQuery({
    queryKey: ['user-groups'],
    queryFn: getUserGroups,
    enabled: props.open,
    staleTime: 0,
  })
  const groupLabels = Object.fromEntries(
    Object.entries(groupsQuery.data?.data ?? {}).map(([group, info]) => [
      group,
      info.desc || group,
    ])
  )
  const currentConfig = CC_SWITCH_APPS[app]
  const [primaryField, ...optionalFields] = currentConfig.modelFields
  const defaultNames: Partial<Record<CCSwitchApp, string>> = {
    claude: t('My Claude'),
    codex: t('My Codex'),
    gemini: t('My Gemini'),
  }
  const currentDefaultName = resolveCCSwitchDefaultName(
    props.apiKey?.name,
    defaultNames[app] ?? currentConfig.label
  )
  const currentDraft = drafts[app] ?? { name: currentDefaultName, models: {} }
  const primarySelection = currentDraft.models.model
  const primaryModelValue = primarySelection?.value.trim() ?? ''
  const modelsCount = modelsQuery.data?.success
    ? modelsQuery.data.models.length
    : null
  const modelsFailure: Extract<ApiKeyModelsResult, { success: false }> | null =
    modelsQuery.data?.success === false ? modelsQuery.data : null

  useEffect(() => {
    if (!props.open || !modelsQuery.data?.success) return
    if (primaryModelValue) return

    const suggestedModel = selectSuggestedModel(
      modelsQuery.data,
      app,
      getPreferredModelOrder()
    )
    if (!suggestedModel) return

    setDrafts((current) => ({
      ...current,
      [app]: {
        name: current[app]?.name ?? currentDefaultName,
        models: {
          ...current[app]?.models,
          model: { source: 'catalog', value: suggestedModel },
        },
      },
    }))
  }, [
    app,
    currentDefaultName,
    modelsQuery.data,
    primaryModelValue,
    preferredModels,
    props.open,
  ])

  useEffect(() => {
    return () => launchCleanupRef.current()
  }, [])

  const updateName = (name: string) => {
    setDrafts((current) => ({
      ...current,
      [app]: { models: current[app]?.models ?? {}, name },
    }))
  }

  const updateModel = (
    field: CCSwitchModelField,
    value: string,
    source: ApiKeyModelSelectionSource
  ) => {
    setDrafts((current) => ({
      ...current,
      [app]: {
        name: current[app]?.name ?? currentDefaultName,
        models: {
          ...current[app]?.models,
          [field]: { source, value },
        },
      },
    }))
    if (field === 'model' && value.trim()) setPrimaryModelError(false)
  }

  const handleAppChange = (nextApp: CCSwitchApp) => {
    setPrimaryModelError(false)
    setApp(nextApp)
  }

  const handleSubmit = () => {
    if (!canImport || launchState !== 'idle') return
    if (primaryField.required && !primaryModelValue) {
      setPrimaryModelError(true)
      requestAnimationFrame(() => primaryModelRef.current?.focus())
      return
    }
    const models = Object.fromEntries(
      Object.entries(currentDraft.models)
        .map(([field, selection]) => [field, selection?.value.trim() ?? ''])
        .filter(([, value]) => value)
    )
    const url = buildCCSwitchImportUrl({
      app,
      apiKey: props.tokenKey,
      models,
      name: currentDraft.name.trim() || currentDefaultName,
      serverAddress: normalizedServerAddress,
      logoUrl: status?.logo,
      usageAuthorization: usageAuthorization.data,
    })

    setLaunchState('opening')

    const cleanup = () => {
      window.clearTimeout(timerId)
      window.removeEventListener('blur', finishLaunch)
      window.removeEventListener('pagehide', finishLaunch)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      launchCleanupRef.current = () => {}
    }
    const finishLaunch = () => {
      cleanup()
      setLaunchState('idle')
    }
    const handleVisibilityChange = () => {
      if (document.hidden) finishLaunch()
    }
    // Browsers expose focus changes, but no confirmation from the protocol handler.
    const timerId = window.setTimeout(() => {
      cleanup()
      setLaunchState('help')
    }, 2000)
    launchCleanupRef.current = cleanup
    window.addEventListener('blur', finishLaunch)
    window.addEventListener('pagehide', finishLaunch)
    document.addEventListener('visibilitychange', handleVisibilityChange)

    const link = document.createElement('a')
    link.href = url
    link.rel = 'noreferrer'
    link.style.display = 'none'
    document.body.appendChild(link)
    try {
      link.click()
    } catch {
      cleanup()
      setLaunchState('help')
    } finally {
      link.remove()
    }
  }

  const renderModelField = (field: {
    key: CCSwitchModelField
    required: boolean
  }) => {
    const selection = currentDraft.models[field.key]
    const isPrimaryInvalid = field.key === 'model' && primaryModelError
    const fieldId = `cc-switch-${app}-${field.key}`
    const errorId = `${fieldId}-error`

    return (
      <Field key={`${app}-${field.key}`} data-invalid={isPrimaryInvalid}>
        <FieldLabel htmlFor={fieldId}>
          {getModelFieldLabel(t, field.key)}
          {field.required ? (
            <span className='text-destructive' aria-hidden='true'>
              *
            </span>
          ) : null}
        </FieldLabel>
        <ApiKeyModelCombobox
          app={app}
          describedBy={isPrimaryInvalid ? errorId : undefined}
          disabled={launchState === 'opening' || !isConfigurationReady}
          field={field.key}
          id={fieldId}
          inputRef={field.key === 'model' ? primaryModelRef : undefined}
          invalid={isPrimaryInvalid}
          isFetching={modelsQuery.isFetching}
          isPending={modelsQuery.isPending}
          modelsResult={modelsQuery.data}
          source={selection?.source ?? 'custom'}
          value={selection?.value ?? ''}
          onRetry={() => void modelsQuery.refetch()}
          required={field.required}
          onValueChange={(value, source) =>
            updateModel(field.key, value, source)
          }
        />
        {isPrimaryInvalid ? (
          <FieldError id={errorId}>
            {t('Please select a primary model')}
          </FieldError>
        ) : null}
      </Field>
    )
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Import to CC Switch')}
      contentClassName='sm:max-w-2xl'
      contentHeight='auto'
      bodyClassName='space-y-5'
      footer={
        <>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={launchState === 'opening' || !canImport}
          >
            {launchState === 'opening' || usageAuthorization.isFetching ? (
              <Spinner data-icon='inline-start' aria-label={t('Loading')} />
            ) : null}
            {t('One-click import')}
          </Button>
        </>
      }
    >
      <div className='bg-muted/40 flex min-w-0 flex-col gap-2 rounded-lg px-3 py-2.5 sm:flex-row sm:items-center'>
        <div className='min-w-0 flex-1'>
          <div className='truncate text-sm font-medium'>
            {props.apiKey?.name ?? t('API Key')}
          </div>
          <div className='text-muted-foreground mt-0.5 truncate text-xs'>
            {getRoutingLabel(t, props.apiKey, groupLabels)}
          </div>
        </div>
        <div className='flex shrink-0 items-center gap-2' aria-live='polite'>
          {modelsQuery.isPending && isConfigurationReady ? (
            <Skeleton className='h-5 w-20' />
          ) : null}
          {modelsCount !== null ? (
            <Badge variant='secondary'>
              {t('{{count}} models', { count: modelsCount })}
            </Badge>
          ) : null}
          {modelsQuery.isFetching && !modelsQuery.isPending ? (
            <Spinner aria-label={t('Loading')} />
          ) : null}
        </div>
      </div>

      {!isConfigurationReady ? (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={AlertCircleIcon} aria-hidden='true' />
          <AlertTitle>{t('CC Switch import unavailable')}</AlertTitle>
          <AlertDescription>
            {t('The API key or server address is missing.')}
          </AlertDescription>
        </Alert>
      ) : null}

      {usageAuthorization.isError ? (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={AlertCircleIcon} aria-hidden='true' />
          <AlertTitle>
            {t('Failed to prepare the CC Switch usage query')}
          </AlertTitle>
          <AlertAction>
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={t('Retry')}
              disabled={usageAuthorization.isFetching}
              onClick={() => void usageAuthorization.refetch()}
            >
              {usageAuthorization.isFetching ? (
                <Spinner aria-label={t('Loading')} />
              ) : (
                <HugeiconsIcon icon={ReloadIcon} aria-hidden='true' />
              )}
            </Button>
          </AlertAction>
        </Alert>
      ) : null}

      {isConfigurationReady && !supportsUsageQuery ? (
        <Alert>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>
            {t('CC Switch usage queries require an HTTPS server address')}
          </AlertTitle>
        </Alert>
      ) : null}

      {modelsFailure ? (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={AlertCircleIcon} aria-hidden='true' />
          <AlertTitle>{t('Failed to fetch models')}</AlertTitle>
          <AlertDescription>
            {modelsFailure.message || t('Please retry the model request.')}
          </AlertDescription>
          <AlertAction>
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={t('Retry')}
              disabled={modelsQuery.isFetching}
              onClick={() => void modelsQuery.refetch()}
            >
              {modelsQuery.isFetching ? (
                <Spinner aria-label={t('Loading')} />
              ) : (
                <HugeiconsIcon icon={ReloadIcon} aria-hidden='true' />
              )}
            </Button>
          </AlertAction>
        </Alert>
      ) : null}

      {modelsCount === 0 ? (
        <Alert>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>{t('No available models')}</AlertTitle>
          <AlertDescription>
            {t('No available models were returned for this API key.')}
          </AlertDescription>
        </Alert>
      ) : null}

      <FieldGroup>
        <FieldSet disabled={launchState === 'opening'}>
          <FieldLegend variant='label'>{t('Application')}</FieldLegend>
          <RadioGroup
            value={app}
            onValueChange={handleAppChange}
            aria-label={t('Application')}
            disabled={launchState === 'opening'}
            className='flex flex-wrap gap-2'
          >
            {APP_IDS.map((appKey) => (
              <Radio.Root
                key={appKey}
                value={appKey}
                className='border-input bg-background text-foreground hover:bg-muted data-checked:border-primary data-checked:bg-primary/10 focus-visible:ring-ring/50 inline-flex min-h-9 max-w-full items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium transition-colors outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 motion-reduce:transition-none'
              >
                <CCSwitchAppIcon app={appKey} />
                {CC_SWITCH_APPS[appKey].label}
              </Radio.Root>
            ))}
          </RadioGroup>
        </FieldSet>

        <Field data-disabled={launchState === 'opening'}>
          <FieldLabel htmlFor={`cc-switch-${app}-name`}>{t('Name')}</FieldLabel>
          <Input
            id={`cc-switch-${app}-name`}
            disabled={launchState === 'opening'}
            placeholder={currentDefaultName}
            value={currentDraft.name}
            onChange={(event) => updateName(event.target.value)}
          />
        </Field>

        {renderModelField(primaryField)}

        {optionalFields.length > 0 ? (
          <Collapsible className='border-border border-t pt-2'>
            <CollapsibleTrigger
              disabled={launchState === 'opening'}
              render={
                <Button variant='ghost' size='sm' className='group -ml-2' />
              }
            >
              <HugeiconsIcon
                icon={ArrowDown01Icon}
                data-icon='inline-start'
                className='group-data-[panel-open]:rotate-180'
                aria-hidden='true'
              />
              {t('Claude model mapping')}
              <span className='text-muted-foreground font-normal'>
                ({t('Optional')})
              </span>
            </CollapsibleTrigger>
            <CollapsibleContent className='pt-3'>
              <FieldGroup>{optionalFields.map(renderModelField)}</FieldGroup>
            </CollapsibleContent>
          </Collapsible>
        ) : null}
      </FieldGroup>

      <AlertDialog
        open={launchState === 'help'}
        onOpenChange={(open) => !open && setLaunchState('idle')}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('CC Switch did not open')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'CC Switch may not be installed. Download it and try importing again.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Close')}</AlertDialogCancel>
            <a
              className={buttonVariants()}
              href={CCSWITCH_WEBSITE}
              target='_blank'
              rel='noopener noreferrer'
            >
              {t('Download from the CC Switch website')}
              <HugeiconsIcon
                icon={ArrowUpRight01Icon}
                data-icon='inline-end'
                aria-hidden='true'
              />
            </a>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Dialog>
  )
}
