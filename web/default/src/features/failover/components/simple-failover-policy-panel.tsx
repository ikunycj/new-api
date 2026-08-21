import {
  Add01Icon,
  Alert02Icon,
  Delete02Icon,
  FloppyDiskIcon,
  Route01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'

import { updateFailoverConfig } from '../api'
import type {
  FailoverConfig,
  FailoverPolicy,
  FailoverPolicyStep,
  FailoverChannelOption,
  RoutingStrategy,
} from '../types'

type SimplePolicyFormValues = {
  policies: FailoverPolicy[]
  policy_steps: FailoverPolicyStep[]
}

const strategyOrder: RoutingStrategy[] = [
  'cost_first',
  'balanced',
  'stability_first',
  'pro_cost_first',
  'pro_stability_first',
]

const defaultPoolOrder: Record<RoutingStrategy, number[]> = {
  cost_first: [1, 2, 3, 4],
  balanced: [2, 1, 3, 4],
  stability_first: [3, 4, 2, 1],
  pro_cost_first: [2, 3],
  pro_stability_first: [2, 3],
}

export function SimpleFailoverPolicyPanel(props: { config: FailoverConfig }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<SimplePolicyFormValues>({
    defaultValues: {
      policies: props.config.policies,
      policy_steps: props.config.policy_steps,
    },
  })
  const policyValues = form.watch('policies')
  const channelOptions = useMemo(
    () => props.config.channels,
    [props.config.channels]
  )
  const clusterNames = useMemo(
    () => new Map(props.config.clusters.map((cluster) => [cluster.id, cluster.name])),
    [props.config.clusters]
  )

  const policyAssignments = useMemo(() => {
    const assignments = new Map<number, string[]>()
    for (const cluster of props.config.clusters) {
      if (cluster.policy_id <= 0) continue
      const current = assignments.get(cluster.policy_id) ?? []
      current.push(`C${cluster.id} ${cluster.name}`)
      assignments.set(cluster.policy_id, current)
    }
    for (const group of props.config.groups) {
      if (group.policy_id <= 0) continue
      const current = assignments.get(group.policy_id) ?? []
      current.push(group.name)
      assignments.set(group.policy_id, current)
    }
    return assignments
  }, [props.config.clusters, props.config.groups])

  const updateMutation = useMutation({
    mutationFn: (values: SimplePolicyFormValues) =>
      updateFailoverConfig({
        ...props.config,
        policies: values.policies,
        policy_steps: values.policy_steps,
      }),
    onSuccess: async () => {
      toast.success(t('Failover configuration saved'))
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['failover-config'] }),
        queryClient.invalidateQueries({ queryKey: ['cluster-configuration'] }),
      ])
    },
    onError: (error) => toast.error(error.message || t('Save failed')),
  })

  const updateStrategy = (policyIndex: number, strategy: RoutingStrategy) => {
    const policy = form.getValues(`policies.${policyIndex}`)
    form.setValue(`policies.${policyIndex}.strategy`, strategy, {
      shouldDirty: true,
    })

    const allSteps = form.getValues('policy_steps')
    const policySteps = allSteps.filter((step) => step.policy_id === policy.id)
    if (policySteps.length === 0) return

    if (policySteps.some((step) => step.channel_id > 0)) return

    const stepByTier = new Map(
      policySteps.map((step) => [step.pool_tier, step] as const)
    )
    const configuredTiers = new Set(stepByTier.keys())
    const reorderedTiers = defaultPoolOrder[strategy].filter((tier) =>
      configuredTiers.has(tier)
    )
    for (const step of policySteps) {
      if (!reorderedTiers.includes(step.pool_tier)) {
        reorderedTiers.push(step.pool_tier)
      }
    }

    const reorderedSteps = reorderedTiers.flatMap((tier, index) => {
      const step = stepByTier.get(tier)
      if (!step) return []
      return [{ ...step, step_order: index + 1 }]
    })
    form.setValue(
      'policy_steps',
      [
        ...allSteps.filter((step) => step.policy_id !== policy.id),
        ...reorderedSteps,
      ],
      { shouldDirty: true }
    )
  }

  const directStepsForPolicy = (policyId: number) =>
    form
      .getValues('policy_steps')
      .map((step, index) => ({ step, index }))
      .filter(({ step }) => step.policy_id === policyId && step.channel_id > 0)
      .sort((left, right) => left.step.step_order - right.step.step_order)

  const addChannelRoute = (policyId: number) => {
    const values = form.getValues('policy_steps')
    const existing = directStepsForPolicy(policyId)
    const used = new Set(existing.map(({ step }) => step.channel_id))
    const next = channelOptions.find((channel) => !used.has(channel.id))
    const retained = values.filter((step) => step.policy_id !== policyId)
    const direct = existing.map(({ step }, index) => ({
      ...step,
      step_order: index + 1,
      pool_tier: 0,
    }))
    direct.push({
      id: 0,
      policy_id: policyId,
      step_order: direct.length + 1,
      channel_id: next?.id ?? 0,
      pool_tier: 0,
      max_attempts: 1,
    })
    form.setValue('policy_steps', [...retained, ...direct], {
      shouldDirty: true,
      shouldValidate: true,
    })
  }

  const updateChannelRoute = (policyId: number, stepIndex: number, channelId: number) => {
    const values = form.getValues('policy_steps')
    const policySteps = values
      .filter((step) => step.policy_id === policyId && step.channel_id > 0)
      .sort((left, right) => left.step_order - right.step_order)
    const target = policySteps[stepIndex]
    if (!target) return
    target.channel_id = channelId
    target.pool_tier = 0
    form.setValue(
      'policy_steps',
      [
        ...values.filter((step) => step.policy_id !== policyId),
        ...policySteps.map((step, index) => ({ ...step, step_order: index + 1 })),
      ],
      { shouldDirty: true, shouldValidate: true }
    )
  }

  const updateChannelAttempts = (policyId: number, stepIndex: number, maxAttempts: number) => {
    const values = form.getValues('policy_steps')
    const policySteps = values
      .filter((step) => step.policy_id === policyId && step.channel_id > 0)
      .sort((left, right) => left.step_order - right.step_order)
    const target = policySteps[stepIndex]
    if (!target) return
    target.max_attempts = Math.max(1, Math.min(10, maxAttempts || 1))
    form.setValue(
      'policy_steps',
      [
        ...values.filter((step) => step.policy_id !== policyId),
        ...policySteps.map((step, index) => ({ ...step, step_order: index + 1 })),
      ],
      { shouldDirty: true, shouldValidate: true }
    )
  }

  const removeChannelRoute = (policyId: number, stepIndex: number) => {
    const values = form.getValues('policy_steps')
    const policySteps = values
      .filter((step) => step.policy_id === policyId && step.channel_id > 0)
      .sort((left, right) => left.step_order - right.step_order)
      .filter((_, index) => index !== stepIndex)
      .map((step, index) => ({ ...step, step_order: index + 1 }))
    form.setValue(
      'policy_steps',
      [
        ...values.filter((step) => step.policy_id !== policyId),
        ...policySteps,
      ],
      { shouldDirty: true, shouldValidate: true }
    )
  }

  return (
    <form
      className='flex min-w-0 flex-col gap-5'
      onSubmit={form.handleSubmit(
        (values) => updateMutation.mutate(values),
        () => toast.error(t('Configuration validation failed'))
      )}
    >
      <Alert>
        <HugeiconsIcon icon={Route01Icon} />
        <AlertTitle>{t('Channel routing')}</AlertTitle>
        <AlertDescription>
          {t(
            'Routing policies use the configured channel order. Cluster and billing settings remain separate.'
          )}
        </AlertDescription>
      </Alert>

      <div className='grid gap-4 xl:grid-cols-2'>
        {policyValues.map((policy, index) => {
          const directSteps = directStepsForPolicy(policy.id)
          const assignments = policyAssignments.get(policy.id) ?? []
          return (
            <Card key={policy.id} className='rounded-lg'>
              <CardHeader>
                <CardTitle>
                  <Input
                    aria-label={t('Name')}
                    className='max-w-sm font-medium'
                    {...form.register(`policies.${index}.name`, {
                      required: true,
                    })}
                  />
                </CardTitle>
                <CardDescription className='flex flex-wrap items-center gap-2'>
                  {assignments.map((assignment) => (
                    <Badge key={assignment} variant='secondary'>
                      {assignment}
                    </Badge>
                  ))}
                </CardDescription>
                <CardAction>
                  <Controller
                    control={form.control}
                    name={`policies.${index}.enabled`}
                    render={({ field }) => (
                      <Switch
                        aria-label={t('Enabled')}
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    )}
                  />
                </CardAction>
              </CardHeader>

              <CardContent className='space-y-5'>
                <Field>
                  <FieldLabel>{t('Routing objective')}</FieldLabel>
                  <Controller
                    control={form.control}
                    name={`policies.${index}.strategy`}
                    render={({ field }) => (
                      <Select
                        value={field.value}
                        onValueChange={(value) =>
                          updateStrategy(index, value as RoutingStrategy)
                        }
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            {strategyOrder.map((strategy) => (
                              <SelectItem key={strategy} value={strategy}>
                                {t(strategy)}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    )}
                  />
                </Field>

                <div className='space-y-3'>
                  <div className='flex items-center justify-between gap-2'>
                    <div>
                      <p className='text-sm font-medium'>{t('Channel order')}</p>
                      <p className='text-muted-foreground text-xs'>
                        {directSteps.length > 0
                          ? t('Requests try these channels from left to right.')
                          : t('This policy still uses the legacy compatibility route.')}
                      </p>
                    </div>
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      onClick={() => addChannelRoute(policy.id)}
                      disabled={
                        channelOptions.length === 0 ||
                        directSteps.length >= channelOptions.length
                      }
                    >
                      <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
                      {t('Add channel')}
                    </Button>
                  </div>
                  {directSteps.length > 0 ? (
                    <div className='space-y-2'>
                      {directSteps.map(({ step }, stepIndex) => (
                        <div key={`${policy.id}-${step.channel_id}`} className='flex items-center gap-2'>
                          <span className='text-muted-foreground w-6 text-center text-xs'>
                            {stepIndex + 1}
                          </span>
                          <Select
                            value={String(step.channel_id)}
                            onValueChange={(value) =>
                              updateChannelRoute(policy.id, stepIndex, Number(value))
                            }
                          >
                            <SelectTrigger className='min-w-0 flex-1'>
                              <SelectValue placeholder={t('Select channel')} />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectGroup>
                                {channelOptions.map((channel: FailoverChannelOption) => (
                                  <SelectItem key={channel.id} value={String(channel.id)}>
                                    {channel.name || `Channel ${channel.id}`} · ID {channel.id}
                                    {channel.cluster_id > 0
                                      ? ` · C${channel.cluster_id}${clusterNames.get(channel.cluster_id) ? ` ${clusterNames.get(channel.cluster_id)}` : ''}`
                                      : ''}
                                  </SelectItem>
                                ))}
                              </SelectGroup>
                            </SelectContent>
                          </Select>
                          <Input
                            className='w-20'
                            type='number'
                            min='1'
                            max='10'
                            aria-label={t('Maximum attempts')}
                            value={step.max_attempts}
                            onChange={(event) =>
                              updateChannelAttempts(
                                policy.id,
                                stepIndex,
                                Number(event.target.value)
                              )
                            }
                          />
                          <Button
                            type='button'
                            size='icon'
                            variant='ghost'
                            aria-label={t('Remove channel')}
                            onClick={() => removeChannelRoute(policy.id, stepIndex)}
                          >
                            <HugeiconsIcon icon={Delete02Icon} />
                          </Button>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className='bg-muted/40 rounded-md p-3 text-xs'>
                      {t('Uses each cluster channel chain')}
                    </div>
                  )}
                </div>

                <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-2 2xl:grid-cols-4'>
                  <Field>
                    <FieldLabel>{t('Retry count')}</FieldLabel>
                    <Input
                      type='number'
                      min='0'
                      max='10'
                      {...form.register(`policies.${index}.same_pool_retries`, {
                        valueAsNumber: true,
                        min: 0,
                        max: 10,
                      })}
                    />
                  </Field>
                  <Field>
                    <FieldLabel>{t('Cluster attempts')}</FieldLabel>
                    <Input
                      type='number'
                      min='1'
                      {...form.register(
                        `policies.${index}.max_cluster_attempts`,
                        { valueAsNumber: true, min: 1 }
                      )}
                    />
                  </Field>
                  <Field>
                    <FieldLabel>{t('Total attempts')}</FieldLabel>
                    <Input
                      type='number'
                      min='1'
                      {...form.register(
                        `policies.${index}.max_total_attempts`,
                        {
                          valueAsNumber: true,
                          min: 1,
                        }
                      )}
                    />
                  </Field>
                  <Field>
                    <FieldLabel>{t('Budget (ms)')}</FieldLabel>
                    <Input
                      type='number'
                      min='1000'
                      step='1000'
                      {...form.register(
                        `policies.${index}.total_failover_budget_ms`,
                        { valueAsNumber: true, min: 1000 }
                      )}
                    />
                  </Field>
                </div>

                <div className='bg-muted/40 grid gap-3 rounded-md p-3 sm:grid-cols-3'>
                  <Field orientation='horizontal'>
                    <FieldLabel>{t('Paid escalation')}</FieldLabel>
                    <Controller
                      control={form.control}
                      name={`policies.${index}.allow_paid_escalation`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </Field>
                  <Field orientation='horizontal'>
                    <FieldLabel>{t('Fallback')}</FieldLabel>
                    <Controller
                      control={form.control}
                      name={`policies.${index}.allow_fallback`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </Field>
                  <Field>
                    <FieldLabel>{t('Max cost multiplier')}</FieldLabel>
                    <Input
                      type='number'
                      min='0.01'
                      step='0.01'
                      {...form.register(
                        `policies.${index}.max_cost_multiplier`,
                        { valueAsNumber: true, min: 0.01 }
                      )}
                    />
                  </Field>
                </div>

                <p className='text-muted-foreground text-xs'>
                  {t('Circuit threshold')}: {policy.circuit_failure_threshold} ·{' '}
                  {t('Window (s)')}: {policy.circuit_window_seconds} ·{' '}
                  {t('Cooldown (s)')}: {policy.circuit_cooldown_seconds}
                </p>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {Object.keys(form.formState.errors).length > 0 && (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={Alert02Icon} />
          <AlertTitle>{t('Configuration validation failed')}</AlertTitle>
          <AlertDescription>
            {t('Check required fields and attempt limits before saving.')}
          </AlertDescription>
        </Alert>
      )}

      <div className='flex justify-end'>
        <Button type='submit' disabled={updateMutation.isPending}>
          {updateMutation.isPending ? (
            <Spinner data-icon='inline-start' />
          ) : (
            <HugeiconsIcon icon={FloppyDiskIcon} data-icon='inline-start' />
          )}
          {updateMutation.isPending ? t('Saving...') : t('Save changes')}
        </Button>
      </div>
    </form>
  )
}
