import { zodResolver } from '@hookform/resolvers/zod'
import {
  Add01Icon,
  Alert02Icon,
  Delete02Icon,
  FloppyDiskIcon,
  Route01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Controller, useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Form } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
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

import { getFailoverConfig, updateFailoverConfig } from './api'
import { ClusterConfigurationPanel } from './components/cluster-configuration-panel'
import { FailoverMonitoring } from './monitoring'
import type { FailoverConfig, FailoverMode } from './types'

const clusterSchema = z.object({
  id: z.coerce.number().int().min(0),
  name: z.string().trim().min(1),
  type: z.string().trim().min(1),
  status: z.coerce.number().int().min(0).max(1),
  billing_group: z.string(),
  policy_id: z.coerce.number().int().min(0),
  failover_priority: z.coerce.number().int().min(0),
  remark: z.string(),
  archived: z.boolean(),
  created_time: z.coerce.number(),
  updated_time: z.coerce.number(),
})

const poolSchema = z.object({
  id: z.coerce.number().int().min(0),
  cluster_id: z.coerce.number().int().positive(),
  tier: z.coerce.number().int().min(1).max(4),
  name: z.string().trim().min(1),
  status: z.coerce.number().int().min(0).max(1),
  cost_factor: z.coerce.number().nonnegative(),
  remark: z.string(),
  created_time: z.coerce.number(),
  updated_time: z.coerce.number(),
})

const policySchema = z
  .object({
    id: z.coerce.number().int().min(0),
    name: z.string().trim().min(1),
    mode: z.enum(['conservative', 'balanced', 'aggressive']),
    enabled: z.boolean(),
    same_pool_retries: z.coerce.number().int().min(0).max(10),
    connect_timeout_ms: z.coerce.number().int().positive(),
    first_byte_timeout_ms: z.coerce.number().int().positive(),
    max_pool_attempts: z.coerce.number().int().positive(),
    max_cluster_attempts: z.coerce.number().int().positive(),
    max_total_attempts: z.coerce.number().int().positive(),
    total_failover_budget_ms: z.coerce.number().int().positive(),
    switch_status_codes: z.string(),
    switch_error_codes: z.string(),
    circuit_failure_threshold: z.coerce.number().int().positive(),
    circuit_window_seconds: z.coerce.number().int().positive(),
    circuit_cooldown_seconds: z.coerce.number().int().positive(),
    circuit_half_open_requests: z.coerce.number().int().positive(),
    allow_paid_escalation: z.boolean(),
    allow_fallback: z.boolean(),
    max_cost_multiplier: z.coerce.number().positive(),
    created_time: z.coerce.number(),
    updated_time: z.coerce.number(),
  })
  .refine((value) => value.max_total_attempts >= value.max_cluster_attempts, {
    path: ['max_total_attempts'],
    message: 'Max total attempts must cover cluster attempts',
  })

const errorMappingSchema = z
  .object({
    id: z.coerce.number().int().min(0),
    cluster_type: z.string().trim().min(1),
    raw_code: z.string().trim(),
    status_code: z.coerce.number().int().min(0).max(599),
    alltoken_code: z.coerce.number().int().min(100000).max(999999),
    category: z.string().trim().min(1),
    failure_scope: z.enum([
      'request',
      'credential',
      'channel',
      'cluster',
      'provider',
    ]),
    action: z.enum(['none', 'failover', 'retry_later', 'abort', 'manual']),
    retryable: z.boolean(),
    enabled: z.boolean(),
  })
  .refine((value) => value.raw_code.length > 0 || value.status_code > 0, {
    path: ['raw_code'],
    message: 'Raw code or HTTP status is required',
  })

const failoverGroupSchema = z.object({
  id: z.coerce.number().int().min(0),
  name: z.string().trim().min(1),
  policy_id: z.coerce.number().int().positive(),
  enabled: z.boolean(),
  created_time: z.coerce.number(),
  updated_time: z.coerce.number(),
})

const groupMemberSchema = z.object({
  id: z.coerce.number().int().min(0),
  failover_group_id: z.coerce.number().int().positive(),
  cluster_id: z.coerce.number().int().positive(),
  priority: z.coerce.number().int(),
  weight: z.coerce.number().int().nonnegative(),
})

const failoverRuleSchema = z.object({
  id: z.coerce.number().int().min(0),
  failover_group_id: z.coerce.number().int().positive(),
  model_pattern: z.string().trim().min(1),
  route_pattern: z.string().trim().min(1),
  user_group: z.string().trim().min(1),
  policy_id: z.coerce.number().int().min(0),
  priority: z.coerce.number().int(),
  enabled: z.boolean(),
})

const formSchema = z.object({
  clusters: z.array(clusterSchema),
  pools: z.array(poolSchema),
  policies: z.array(policySchema),
  groups: z.array(failoverGroupSchema),
  group_members: z.array(groupMemberSchema),
  rules: z.array(failoverRuleSchema),
  error_mappings: z.array(errorMappingSchema),
})

type FormValues = z.output<typeof formSchema>
type FormInput = z.input<typeof formSchema>

const modeOrder: FailoverMode[] = ['conservative', 'balanced', 'aggressive']

function defaultPolicy(mode: FailoverMode): FormInput['policies'][number] {
  const policy: FormInput['policies'][number] = {
    id: 0,
    name: mode,
    mode,
    enabled: true,
    same_pool_retries: 0,
    connect_timeout_ms: 1500,
    first_byte_timeout_ms: 3000,
    max_pool_attempts: 4,
    max_cluster_attempts: 3,
    max_total_attempts: 6,
    total_failover_budget_ms: 10000,
    switch_status_codes: '[429,500,502,503,504]',
    switch_error_codes: '["all_pools_exhausted"]',
    circuit_failure_threshold: 5,
    circuit_window_seconds: 60,
    circuit_cooldown_seconds: 60,
    circuit_half_open_requests: 1,
    allow_paid_escalation: true,
    allow_fallback: true,
    max_cost_multiplier: 2,
    created_time: 0,
    updated_time: 0,
  }
  if (mode === 'conservative') {
    policy.same_pool_retries = 1
    policy.connect_timeout_ms = 2500
    policy.first_byte_timeout_ms = 5000
    policy.max_cluster_attempts = 2
    policy.max_total_attempts = 4
    policy.total_failover_budget_ms = 12000
    policy.circuit_failure_threshold = 8
  }
  if (mode === 'aggressive') {
    policy.connect_timeout_ms = 1000
    policy.first_byte_timeout_ms = 2000
    policy.max_cluster_attempts = 4
    policy.max_total_attempts = 8
    policy.total_failover_budget_ms = 6000
    policy.circuit_failure_threshold = 3
    policy.circuit_window_seconds = 30
    policy.circuit_cooldown_seconds = 90
  }
  return policy
}

function FailoverConfigForm(props: { config: FailoverConfig }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<FormInput, unknown, FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      clusters: props.config.clusters,
      pools: props.config.pools,
      policies: props.config.policies,
      groups: props.config.groups,
      group_members: props.config.group_members,
      rules: props.config.rules,
      error_mappings: props.config.error_mappings,
    },
  })
  const clusters = useFieldArray({ control: form.control, name: 'clusters' })
  const pools = useFieldArray({ control: form.control, name: 'pools' })
  const policies = useFieldArray({ control: form.control, name: 'policies' })
  const errorMappings = useFieldArray({
    control: form.control,
    name: 'error_mappings',
  })
  const groups = useFieldArray({ control: form.control, name: 'groups' })
  const groupMembers = useFieldArray({
    control: form.control,
    name: 'group_members',
  })
  const rules = useFieldArray({ control: form.control, name: 'rules' })
  const clusterValues = form.watch('clusters') as FormValues['clusters']
  const policyValues = form.watch('policies') as FormValues['policies']
  const groupValues = form.watch('groups') as FormValues['groups']
  const clusterSelectItems = useMemo(
    () =>
      clusterValues
        .filter((cluster) => cluster.id > 0)
        .map((cluster) => ({
          label: `${cluster.name} · ${t('ID')} ${cluster.id}`,
          value: String(cluster.id),
        })),
    [clusterValues, t]
  )
  const policySelectItems = useMemo(
    () =>
      policyValues
        .filter((policy) => policy.id > 0)
        .map((policy) => ({
          label: `${policy.name} · ${t('ID')} ${policy.id}`,
          value: String(policy.id),
        })),
    [policyValues, t]
  )
  const groupSelectItems = useMemo(
    () =>
      groupValues
        .filter((group) => group.id > 0)
        .map((group) => ({
          label: `${group.name} · ${t('ID')} ${group.id}`,
          value: String(group.id),
        })),
    [groupValues, t]
  )
  const rulePolicySelectItems = useMemo(
    () => [
      { label: t('Inherit group policy'), value: '0' },
      ...policySelectItems,
    ],
    [policySelectItems, t]
  )

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      updateFailoverConfig({
        ...props.config,
        clusters: values.clusters,
        pools: values.pools,
        policies: values.policies,
        groups: values.groups,
        group_members: values.group_members,
        rules: values.rules,
        error_mappings: values.error_mappings,
      }),
    onSuccess: async () => {
      toast.success(t('Failover configuration saved'))
      await queryClient.invalidateQueries({ queryKey: ['failover-config'] })
    },
    onError: (error) => toast.error(error.message || t('Save failed')),
  })

  const addPolicy = () => {
    const existing = new Set(
      form.getValues('policies').map((item) => item.mode)
    )
    const mode = modeOrder.find((candidate) => !existing.has(candidate))
    if (mode) policies.append(defaultPolicy(mode))
  }

  return (
    <Form {...form}>
      <form
        className='flex min-w-0 flex-col gap-6'
        onSubmit={form.handleSubmit((values) => updateMutation.mutate(values))}
      >
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h2 className='text-base font-semibold'>
              {t('Failover policies')}
            </h2>
            <p className='text-muted-foreground text-sm'>
              {t('Aggressive mode switches earlier; it does not retry more.')}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            onClick={addPolicy}
            disabled={policies.fields.length >= modeOrder.length}
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add policy')}
          </Button>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Mode')}</TableHead>
                <TableHead>{t('Same-pool retries')}</TableHead>
                <TableHead>{t('Pool attempts')}</TableHead>
                <TableHead>{t('Cluster attempts')}</TableHead>
                <TableHead>{t('Total attempts')}</TableHead>
                <TableHead>{t('Budget (ms)')}</TableHead>
                <TableHead>{t('Circuit threshold')}</TableHead>
                <TableHead>{t('Window (s)')}</TableHead>
                <TableHead>{t('Cooldown (s)')}</TableHead>
                <TableHead>{t('Half-open probes')}</TableHead>
                <TableHead>{t('Max cost multiplier')}</TableHead>
                <TableHead>{t('Paid escalation')}</TableHead>
                <TableHead>{t('Fallback')}</TableHead>
                <TableHead className='w-10' />
              </TableRow>
            </TableHeader>
            <TableBody>
              {policies.fields.map((policy, index) => (
                <TableRow key={policy.id}>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`policies.${index}.mode`}
                      render={({ field }) => (
                        <Select
                          value={field.value}
                          onValueChange={field.onChange}
                        >
                          <SelectTrigger className='w-36'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {modeOrder.map((mode) => (
                                <SelectItem key={mode} value={mode}>
                                  {t(mode)}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  {[
                    'same_pool_retries',
                    'max_pool_attempts',
                    'max_cluster_attempts',
                    'max_total_attempts',
                    'total_failover_budget_ms',
                    'circuit_failure_threshold',
                    'circuit_window_seconds',
                    'circuit_cooldown_seconds',
                    'circuit_half_open_requests',
                    'max_cost_multiplier',
                  ].map((name) => (
                    <TableCell key={name}>
                      <Input
                        type='number'
                        min={name === 'same_pool_retries' ? '0' : '1'}
                        step={name === 'max_cost_multiplier' ? '0.01' : '1'}
                        className='w-28'
                        {...form.register(
                          `policies.${index}.${name}` as
                            | `policies.${number}.same_pool_retries`
                            | `policies.${number}.max_pool_attempts`
                            | `policies.${number}.max_cluster_attempts`
                            | `policies.${number}.max_total_attempts`
                            | `policies.${number}.total_failover_budget_ms`
                            | `policies.${number}.circuit_failure_threshold`
                            | `policies.${number}.circuit_window_seconds`
                            | `policies.${number}.circuit_cooldown_seconds`
                            | `policies.${number}.circuit_half_open_requests`
                            | `policies.${number}.max_cost_multiplier`
                        )}
                      />
                    </TableCell>
                  ))}
                  <TableCell>
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
                  </TableCell>
                  <TableCell>
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
                  </TableCell>
                  <TableCell>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      aria-label={t('Delete policy')}
                      onClick={() => policies.remove(index)}
                    >
                      <HugeiconsIcon icon={Delete02Icon} />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <Separator />

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h2 className='text-base font-semibold'>{t('Clusters')}</h2>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Cluster IDs are permanent and are shown in error references.'
              )}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              clusters.append({
                id: 0,
                name: '',
                type: 'custom',
                status: 1,
                billing_group: '',
                policy_id: 0,
                failover_priority: 100,
                remark: '',
                archived: false,
                created_time: 0,
                updated_time: 0,
              })
            }
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add cluster')}
          </Button>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Code')}</TableHead>
                <TableHead>{t('Name')}</TableHead>
                <TableHead>{t('Type')}</TableHead>
                <TableHead>{t('Enabled')}</TableHead>
                <TableHead>{t('Remark')}</TableHead>
                <TableHead className='w-10' />
              </TableRow>
            </TableHeader>
            <TableBody>
              {clusters.fields.map((cluster, index) => (
                <TableRow key={cluster.id}>
                  <TableCell>
                    <Badge variant='outline'>
                      C{clusterValues[index]?.id || t('New')}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Input
                      className='min-w-40'
                      {...form.register(`clusters.${index}.name`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      className='min-w-28'
                      {...form.register(`clusters.${index}.type`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`clusters.${index}.status`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value === 1}
                          onCheckedChange={(checked) =>
                            field.onChange(checked ? 1 : 0)
                          }
                        />
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      className='min-w-48'
                      {...form.register(`clusters.${index}.remark`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      aria-label={t('Remove cluster')}
                      onClick={() => clusters.remove(index)}
                    >
                      <HugeiconsIcon icon={Delete02Icon} />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <Separator />

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h2 className='text-base font-semibold'>{t('Account pools')}</h2>
            <p className='text-muted-foreground text-sm'>
              {t('P1 is free, P2 is Pro/Plus, and P3 is the fallback path.')}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            disabled={!clusterValues.some((cluster) => cluster.id > 0)}
            onClick={() => {
              const cluster = clusterValues.find((item) => item.id > 0)
              if (!cluster) return
              pools.append({
                id: 0,
                cluster_id: cluster.id,
                tier: 1,
                name: 'Free',
                status: 1,
                cost_factor: 1,
                remark: '',
                created_time: 0,
                updated_time: 0,
              })
            }}
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add pool')}
          </Button>
        </div>

        {pools.fields.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <HugeiconsIcon icon={Route01Icon} />
              </EmptyMedia>
              <EmptyTitle>{t('No account pools')}</EmptyTitle>
              <EmptyDescription>
                {t('Add a cluster first, then define its P1-P4 pools.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Cluster')}</TableHead>
                  <TableHead>{t('Pool tier')}</TableHead>
                  <TableHead>{t('Name')}</TableHead>
                  <TableHead>{t('Cost multiplier')}</TableHead>
                  <TableHead>{t('Enabled')}</TableHead>
                  <TableHead className='w-10' />
                </TableRow>
              </TableHeader>
              <TableBody>
                {pools.fields.map((pool, index) => (
                  <TableRow key={pool.id}>
                    <TableCell>
                      <Controller
                        control={form.control}
                        name={`pools.${index}.cluster_id`}
                        render={({ field }) => (
                          <Select
                            items={clusterSelectItems}
                            value={String(field.value)}
                            onValueChange={(value) =>
                              field.onChange(Number(value))
                            }
                          >
                            <SelectTrigger className='w-52'>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectGroup>
                                {clusterSelectItems.map((item) => (
                                  <SelectItem
                                    key={item.value}
                                    value={item.value}
                                  >
                                    {item.label}
                                  </SelectItem>
                                ))}
                              </SelectGroup>
                            </SelectContent>
                          </Select>
                        )}
                      />
                    </TableCell>
                    <TableCell>
                      <Controller
                        control={form.control}
                        name={`pools.${index}.tier`}
                        render={({ field }) => (
                          <Select
                            value={String(field.value)}
                            onValueChange={(value) =>
                              field.onChange(Number(value))
                            }
                          >
                            <SelectTrigger className='w-24'>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectGroup>
                                {[1, 2, 3, 4].map((tier) => (
                                  <SelectItem key={tier} value={String(tier)}>
                                    P{tier}
                                  </SelectItem>
                                ))}
                              </SelectGroup>
                            </SelectContent>
                          </Select>
                        )}
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        className='min-w-36'
                        {...form.register(`pools.${index}.name`)}
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        type='number'
                        min='0'
                        step='0.01'
                        className='w-28'
                        {...form.register(`pools.${index}.cost_factor`)}
                      />
                    </TableCell>
                    <TableCell>
                      <Controller
                        control={form.control}
                        name={`pools.${index}.status`}
                        render={({ field }) => (
                          <Switch
                            checked={field.value === 1}
                            onCheckedChange={(checked) =>
                              field.onChange(checked ? 1 : 0)
                            }
                          />
                        )}
                      />
                    </TableCell>
                    <TableCell>
                      <Button
                        type='button'
                        size='icon'
                        variant='ghost'
                        aria-label={t('Remove pool')}
                        onClick={() => pools.remove(index)}
                      >
                        <HugeiconsIcon icon={Delete02Icon} />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        <Separator />

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h2 className='text-base font-semibold'>{t('Failover groups')}</h2>
            <p className='text-muted-foreground text-sm'>
              {t('Limit cross-cluster routing to an explicit cluster set.')}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            disabled={!policyValues.some((policy) => policy.id > 0)}
            onClick={() => {
              const policy = policyValues.find((item) => item.id > 0)
              if (!policy) return
              groups.append({
                id: 0,
                name: '',
                policy_id: policy.id,
                enabled: true,
                created_time: 0,
                updated_time: 0,
              })
            }}
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add group')}
          </Button>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Name')}</TableHead>
                <TableHead>{t('Policy')}</TableHead>
                <TableHead>{t('Enabled')}</TableHead>
                <TableHead className='w-10' />
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups.fields.map((group, index) => (
                <TableRow key={group.id}>
                  <TableCell>
                    <Input
                      className='min-w-40'
                      {...form.register(`groups.${index}.name`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`groups.${index}.policy_id`}
                      render={({ field }) => (
                        <Select
                          items={policySelectItems}
                          value={String(field.value)}
                          onValueChange={(value) =>
                            field.onChange(Number(value))
                          }
                        >
                          <SelectTrigger className='w-52'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {policySelectItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`groups.${index}.enabled`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      aria-label={t('Remove group')}
                      onClick={() => groups.remove(index)}
                    >
                      <HugeiconsIcon icon={Delete02Icon} />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <h3 className='text-sm font-semibold'>{t('Group members')}</h3>
          <Button
            type='button'
            variant='outline'
            disabled={
              !groupValues.some((group) => group.id > 0) ||
              !clusterValues.some((cluster) => cluster.id > 0)
            }
            onClick={() => {
              const group = groupValues.find((item) => item.id > 0)
              const cluster = clusterValues.find((item) => item.id > 0)
              if (!group || !cluster) return
              groupMembers.append({
                id: 0,
                failover_group_id: group.id,
                cluster_id: cluster.id,
                priority: 100,
                weight: 100,
              })
            }}
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add member')}
          </Button>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Group')}</TableHead>
                <TableHead>{t('Cluster')}</TableHead>
                <TableHead>{t('Priority')}</TableHead>
                <TableHead>{t('Weight')}</TableHead>
                <TableHead className='w-10' />
              </TableRow>
            </TableHeader>
            <TableBody>
              {groupMembers.fields.map((member, index) => (
                <TableRow key={member.id}>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`group_members.${index}.failover_group_id`}
                      render={({ field }) => (
                        <Select
                          items={groupSelectItems}
                          value={String(field.value)}
                          onValueChange={(value) =>
                            field.onChange(Number(value))
                          }
                        >
                          <SelectTrigger className='w-60'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {groupSelectItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`group_members.${index}.cluster_id`}
                      render={({ field }) => (
                        <Select
                          items={clusterSelectItems}
                          value={String(field.value)}
                          onValueChange={(value) =>
                            field.onChange(Number(value))
                          }
                        >
                          <SelectTrigger className='w-52'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {clusterSelectItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  {(['priority', 'weight'] as const).map((fieldName) => (
                    <TableCell key={fieldName}>
                      <Input
                        type='number'
                        min={fieldName === 'weight' ? '0' : undefined}
                        className='w-24'
                        {...form.register(
                          `group_members.${index}.${fieldName}`
                        )}
                      />
                    </TableCell>
                  ))}
                  <TableCell>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      aria-label={t('Remove member')}
                      onClick={() => groupMembers.remove(index)}
                    >
                      <HugeiconsIcon icon={Delete02Icon} />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <h3 className='text-sm font-semibold'>{t('Routing rules')}</h3>
          <Button
            type='button'
            variant='outline'
            disabled={!groupValues.some((group) => group.id > 0)}
            onClick={() => {
              const group = groupValues.find((item) => item.id > 0)
              if (!group) return
              rules.append({
                id: 0,
                failover_group_id: group.id,
                model_pattern: '*',
                route_pattern: '*',
                user_group: '*',
                policy_id: 0,
                priority: 100,
                enabled: true,
              })
            }}
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add rule')}
          </Button>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Group')}</TableHead>
                <TableHead>{t('Model pattern')}</TableHead>
                <TableHead>{t('Route pattern')}</TableHead>
                <TableHead>{t('User group')}</TableHead>
                <TableHead>{t('Policy')}</TableHead>
                <TableHead>{t('Priority')}</TableHead>
                <TableHead>{t('Enabled')}</TableHead>
                <TableHead className='w-10' />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.fields.map((rule, index) => (
                <TableRow key={rule.id}>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`rules.${index}.failover_group_id`}
                      render={({ field }) => (
                        <Select
                          items={groupSelectItems}
                          value={String(field.value)}
                          onValueChange={(value) =>
                            field.onChange(Number(value))
                          }
                        >
                          <SelectTrigger className='w-60'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {groupSelectItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  {(
                    ['model_pattern', 'route_pattern', 'user_group'] as const
                  ).map((fieldName) => (
                    <TableCell key={fieldName}>
                      <Input
                        className='min-w-32'
                        {...form.register(`rules.${index}.${fieldName}`)}
                      />
                    </TableCell>
                  ))}
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`rules.${index}.policy_id`}
                      render={({ field }) => (
                        <Select
                          items={rulePolicySelectItems}
                          value={String(field.value)}
                          onValueChange={(value) =>
                            field.onChange(Number(value))
                          }
                        >
                          <SelectTrigger className='w-52'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {rulePolicySelectItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type='number'
                      className='w-24'
                      {...form.register(`rules.${index}.priority`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`rules.${index}.enabled`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      aria-label={t('Remove rule')}
                      onClick={() => rules.remove(index)}
                    >
                      <HugeiconsIcon icon={Delete02Icon} />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <Separator />

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h2 className='text-base font-semibold'>{t('Error mappings')}</h2>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Map vendor-specific errors to stable AllToken codes and failover actions.'
              )}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              errorMappings.append({
                id: 0,
                cluster_type: '*',
                raw_code: '',
                status_code: 0,
                alltoken_code: 200001,
                category: 'unknown',
                failure_scope: 'cluster',
                action: 'failover',
                retryable: true,
                enabled: true,
              })
            }
          >
            <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
            {t('Add mapping')}
          </Button>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Cluster type')}</TableHead>
                <TableHead>{t('Raw code')}</TableHead>
                <TableHead>{t('HTTP status')}</TableHead>
                <TableHead>{t('AllToken code')}</TableHead>
                <TableHead>{t('Category')}</TableHead>
                <TableHead>{t('Failure scope')}</TableHead>
                <TableHead>{t('Action')}</TableHead>
                <TableHead>{t('Retryable')}</TableHead>
                <TableHead>{t('Enabled')}</TableHead>
                <TableHead className='w-10' />
              </TableRow>
            </TableHeader>
            <TableBody>
              {errorMappings.fields.map((mapping, index) => (
                <TableRow key={mapping.id}>
                  <TableCell>
                    <Input
                      className='w-28'
                      placeholder='ikun'
                      {...form.register(`error_mappings.${index}.cluster_type`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      className='min-w-40'
                      placeholder='pool_exhausted'
                      {...form.register(`error_mappings.${index}.raw_code`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type='number'
                      min='0'
                      max='599'
                      className='w-24'
                      {...form.register(`error_mappings.${index}.status_code`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type='number'
                      min='100000'
                      max='999999'
                      className='w-28'
                      {...form.register(
                        `error_mappings.${index}.alltoken_code`
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      className='w-28'
                      {...form.register(`error_mappings.${index}.category`)}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`error_mappings.${index}.failure_scope`}
                      render={({ field }) => (
                        <Select
                          value={field.value}
                          onValueChange={field.onChange}
                        >
                          <SelectTrigger className='w-32'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {[
                                'request',
                                'credential',
                                'channel',
                                'cluster',
                                'provider',
                              ].map((scope) => (
                                <SelectItem key={scope} value={scope}>
                                  {scope}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`error_mappings.${index}.action`}
                      render={({ field }) => (
                        <Select
                          value={field.value}
                          onValueChange={field.onChange}
                        >
                          <SelectTrigger className='w-32'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {[
                                'none',
                                'failover',
                                'retry_later',
                                'abort',
                                'manual',
                              ].map((action) => (
                                <SelectItem key={action} value={action}>
                                  {action}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`error_mappings.${index}.retryable`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Controller
                      control={form.control}
                      name={`error_mappings.${index}.enabled`}
                      render={({ field }) => (
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </TableCell>
                  <TableCell>
                    <Button
                      type='button'
                      size='icon'
                      variant='ghost'
                      aria-label={t('Remove mapping')}
                      onClick={() => errorMappings.remove(index)}
                    >
                      <HugeiconsIcon icon={Delete02Icon} />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
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

        <FieldGroup>
          <Field orientation='horizontal'>
            <div>
              <FieldLabel>{t('Apply routing configuration')}</FieldLabel>
              <FieldDescription>
                {t(
                  'Changes take effect for new requests after the channel cache refreshes.'
                )}
              </FieldDescription>
            </div>
            <Button type='submit' disabled={updateMutation.isPending}>
              {updateMutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : (
                <HugeiconsIcon icon={FloppyDiskIcon} data-icon='inline-start' />
              )}
              {updateMutation.isPending ? t('Saving...') : t('Save changes')}
            </Button>
          </Field>
        </FieldGroup>
      </form>
    </Form>
  )
}

function AdvancedFailoverConfiguration() {
  const { t } = useTranslation()
  const configQuery = useQuery({
    queryKey: ['failover-config'],
    queryFn: getFailoverConfig,
  })

  return (
    <div className='space-y-6'>
      {configQuery.isLoading && (
        <div className='flex flex-col gap-3'>
          <Skeleton className='h-10 w-full' />
          <Skeleton className='h-48 w-full' />
          <Skeleton className='h-48 w-full' />
        </div>
      )}
      {configQuery.isError && (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={Alert02Icon} />
          <AlertTitle>{t('Failed to load failover configuration')}</AlertTitle>
          <AlertDescription>{configQuery.error.message}</AlertDescription>
        </Alert>
      )}
      {configQuery.data && (
        <FailoverConfigForm
          key={configQuery.dataUpdatedAt}
          config={configQuery.data}
        />
      )}
    </div>
  )
}

function FailoverConfigurationPanel() {
  const { t } = useTranslation()
  return (
    <div className='space-y-6'>
      <Alert>
        <HugeiconsIcon icon={Route01Icon} />
        <AlertTitle>{t('Two-level failover')}</AlertTitle>
        <AlertDescription>
          {t(
            'Clusters exhaust P1-P4 internally; AllToken then switches to another cluster in the same billing group.'
          )}
        </AlertDescription>
      </Alert>
      <Tabs defaultValue='clusters' className='gap-5'>
        <TabsList>
          <TabsTrigger value='clusters'>{t('Cluster setup')}</TabsTrigger>
          <TabsTrigger value='advanced'>{t('Advanced rules')}</TabsTrigger>
        </TabsList>
        <TabsContent value='clusters'>
          <ClusterConfigurationPanel />
        </TabsContent>
        <TabsContent value='advanced'>
          <AdvancedFailoverConfiguration />
        </TabsContent>
      </Tabs>
    </div>
  )
}

export function FailoverConfiguration() {
  const { t } = useTranslation()
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Monitoring & Alerts')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto w-full max-w-[1600px] px-4 py-6 sm:px-6'>
          <Tabs defaultValue='monitoring' className='gap-5'>
            <TabsList>
              <TabsTrigger value='monitoring'>
                {t('Monitoring & Alerts')}
              </TabsTrigger>
              <TabsTrigger value='configuration'>
                {t('Routing Configuration')}
              </TabsTrigger>
            </TabsList>
            <TabsContent value='monitoring'>
              <FailoverMonitoring />
            </TabsContent>
            <TabsContent value='configuration'>
              <FailoverConfigurationPanel />
            </TabsContent>
          </Tabs>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
