import { zodResolver } from '@hookform/resolvers/zod'
import {
  Add01Icon,
  Alert02Icon,
  Delete02Icon,
  FloppyDiskIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as React from 'react'
import { Controller, useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Form } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
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

import {
  deleteClusterConfiguration,
  getClusterConfiguration,
  updateClusterConfiguration,
} from '../api'
import type {
  ClusterConfiguration,
  ClusterConfigurationSnapshot,
} from '../types'

const routeSchema = z.object({
  channel_id: z.coerce.number().int().positive(),
  pool_tier: z.coerce.number().int().min(1).max(4),
  pool_name: z.string().trim().min(1),
  route_order: z.coerce.number().int().min(1).max(4),
  weight: z.coerce.number().int().min(0).max(1000),
  cost_factor: z.coerce.number().min(0),
})

const clusterConfigurationSchema = z
  .object({
    id: z.coerce.number().int().min(0),
    name: z.string().trim().min(1),
    type: z.string().trim().min(1),
    status: z.coerce.number().int().min(0).max(1),
    archived: z.boolean(),
    billing_group: z.string().trim().min(1).max(64),
    billing_group_description: z.string().trim().min(1),
    billing_group_ratio: z.coerce.number().min(0),
    policy_id: z.coerce.number().int().min(0),
    failover_priority: z.coerce.number().int().min(0).max(100000),
    remark: z.string(),
    routes: z.array(routeSchema).min(1).max(4),
  })
  .superRefine((value, context) => {
    const channelIds = value.routes.map((route) => route.channel_id)
    if (new Set(channelIds).size !== channelIds.length) {
      context.addIssue({
        code: 'custom',
        path: ['routes'],
        message: 'A channel can only appear once in a cluster',
      })
    }
  })

type ClusterConfigurationInput = z.input<typeof clusterConfigurationSchema>
type ClusterConfigurationValues = z.output<typeof clusterConfigurationSchema>

const defaultRoutes: ClusterConfiguration['routes'] = [
  {
    channel_id: 0,
    pool_tier: 1,
    pool_name: 'Free',
    route_order: 1,
    weight: 100,
    cost_factor: 0,
  },
  {
    channel_id: 0,
    pool_tier: 2,
    pool_name: 'Pro/Plus',
    route_order: 2,
    weight: 100,
    cost_factor: 1,
  },
  {
    channel_id: 0,
    pool_tier: 3,
    pool_name: 'Fallback',
    route_order: 3,
    weight: 100,
    cost_factor: 1.5,
  },
  {
    channel_id: 0,
    pool_tier: 4,
    pool_name: 'Emergency',
    route_order: 4,
    weight: 100,
    cost_factor: 2,
  },
]

const unifiedBillingGroup = 'cluster'

function newClusterConfiguration(): ClusterConfiguration {
  return {
    id: 0,
    name: '',
    type: 'ikun',
    status: 1,
    archived: false,
    billing_group: unifiedBillingGroup,
    billing_group_description: '',
    billing_group_ratio: 1,
    policy_id: 0,
    failover_priority: 100,
    remark: '',
    routes: defaultRoutes.slice(0, 3).map((route) => ({ ...route })),
  }
}

function ClusterConfigurationWorkspace(props: {
  snapshot: ClusterConfigurationSnapshot
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const initialCluster = props.snapshot.clusters[0] ?? newClusterConfiguration()
  const [selectedClusterId, setSelectedClusterId] = React.useState(
    initialCluster.id
  )
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const form = useForm<
    ClusterConfigurationInput,
    unknown,
    ClusterConfigurationValues
  >({
    resolver: zodResolver(clusterConfigurationSchema),
    defaultValues: { ...initialCluster, billing_group: unifiedBillingGroup },
  })
  const routes = useFieldArray({ control: form.control, name: 'routes' })

  const mutation = useMutation({
    mutationFn: (values: ClusterConfigurationValues) =>
      updateClusterConfiguration(values),
    onSuccess: async () => {
      toast.success(t('Cluster configuration saved'))
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cluster-configuration'] }),
        queryClient.invalidateQueries({ queryKey: ['failover-config'] }),
        queryClient.invalidateQueries({
          queryKey: ['channel-failover-bindings'],
        }),
      ])
    },
    onError: (error) => toast.error(error.message || t('Save failed')),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteClusterConfiguration,
    onSuccess: async () => {
      setDeleteDialogOpen(false)
      setSelectedClusterId(0)
      form.reset(newClusterConfiguration())
      toast.success(t('Deleted successfully'))
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cluster-configuration'] }),
        queryClient.invalidateQueries({ queryKey: ['failover-config'] }),
        queryClient.invalidateQueries({
          queryKey: ['channel-failover-bindings'],
        }),
      ])
    },
    onError: (error) => toast.error(error.message || t('Delete failed')),
  })

  const selectCluster = (cluster: ClusterConfiguration) => {
    setSelectedClusterId(cluster.id)
    form.reset({ ...cluster, billing_group: unifiedBillingGroup })
  }

  const createCluster = () => {
    setSelectedClusterId(0)
    form.reset(newClusterConfiguration())
  }

  const currentClusterId = Number(form.watch('id'))
  const archivedClusterIds = new Set(
    props.snapshot.clusters
      .filter((cluster) => cluster.archived)
      .map((cluster) => cluster.id)
  )
  const selectableChannels = props.snapshot.channels.filter(
    (channel) =>
      channel.cluster_id === 0 ||
      channel.cluster_id === currentClusterId ||
      archivedClusterIds.has(channel.cluster_id)
  )

  return (
    <div className='grid min-w-0 gap-6 xl:grid-cols-[320px_minmax(0,1fr)]'>
      <aside className='min-w-0 space-y-3 border-b pb-5 xl:border-r xl:border-b-0 xl:pr-5 xl:pb-0'>
        <div className='flex items-center justify-between gap-2'>
          <div>
            <h2 className='text-base font-semibold'>{t('Clusters')}</h2>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Each cluster owns one billing group and an ordered route chain.'
              )}
            </p>
          </div>
          <Button
            type='button'
            size='icon'
            variant='outline'
            aria-label={t('Add cluster')}
            onClick={createCluster}
          >
            <HugeiconsIcon icon={Add01Icon} />
          </Button>
        </div>
        <div className='divide-y rounded-md border'>
          {props.snapshot.clusters.map((cluster) => (
            <button
              key={cluster.id}
              type='button'
              className='hover:bg-muted/50 focus-visible:ring-ring flex w-full items-center justify-between gap-3 px-3 py-3 text-left outline-none focus-visible:ring-2'
              aria-pressed={selectedClusterId === cluster.id}
              onClick={() => selectCluster(cluster)}
            >
              <span className='min-w-0'>
                <span className='block truncate text-sm font-medium'>
                  C{cluster.id} · {cluster.name}
                </span>
                <span className='text-muted-foreground block truncate text-xs'>
                  {t('Unified package')}
                </span>
              </span>
              <Badge variant={cluster.status === 1 ? 'default' : 'secondary'}>
                {cluster.status === 1 ? t('Enabled') : t('Disabled')}
              </Badge>
            </button>
          ))}
          {props.snapshot.clusters.length === 0 && (
            <p className='text-muted-foreground px-3 py-6 text-center text-sm'>
              {t('No clusters configured')}
            </p>
          )}
        </div>
      </aside>

      <Form {...form}>
        <form
          className='min-w-0 space-y-6'
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <div>
              <h2 className='text-base font-semibold'>
                {currentClusterId > 0
                  ? t('Edit cluster C{{id}}', { id: currentClusterId })
                  : t('Create cluster')}
              </h2>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Saving updates billing, channel ownership, pool order, and runtime routing together.'
                )}
              </p>
            </div>
            <div className='flex items-center gap-2'>
              {currentClusterId > 0 && (
                <Button
                  type='button'
                  size='icon'
                  variant='outline'
                  aria-label={t('Delete')}
                  onClick={() => setDeleteDialogOpen(true)}
                >
                  <HugeiconsIcon icon={Delete02Icon} />
                </Button>
              )}
              <FieldLabel htmlFor='cluster-enabled'>{t('Enabled')}</FieldLabel>
              <Controller
                control={form.control}
                name='status'
                render={({ field }) => (
                  <Switch
                    id='cluster-enabled'
                    checked={field.value === 1}
                    onCheckedChange={(checked) =>
                      field.onChange(checked ? 1 : 0)
                    }
                  />
                )}
              />
            </div>
          </div>

          <FieldGroup className='grid gap-4 md:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='cluster-name'>
                {t('Cluster name')}
              </FieldLabel>
              <Input id='cluster-name' {...form.register('name')} />
            </Field>
            <Field>
              <FieldLabel htmlFor='cluster-type'>
                {t('Cluster type')}
              </FieldLabel>
              <Input id='cluster-type' {...form.register('type')} />
              <FieldDescription>
                {t('Used to classify upstream errors, for example ikun.')}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor='cluster-priority'>
                {t('Cluster failover priority')}
              </FieldLabel>
              <Input
                id='cluster-priority'
                type='number'
                min='0'
                {...form.register('failover_priority')}
              />
              <FieldDescription>
                {t(
                  'Higher priority clusters are attempted first within the same billing group.'
                )}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor='cluster-policy'>
                {t('Failover policy')}
              </FieldLabel>
              <NativeSelect
                id='cluster-policy'
                className='w-full'
                {...form.register('policy_id')}
              >
                <NativeSelectOption value='0'>
                  {t('Default policy')}
                </NativeSelectOption>
                {props.snapshot.policies.map((policy) => (
                  <NativeSelectOption key={policy.id} value={policy.id}>
                    {policy.name} · {t(policy.mode)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
          </FieldGroup>

          <section className='space-y-4 border-t pt-5'>
            <div>
              <h3 className='text-sm font-semibold'>{t('Billing package')}</h3>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Requests keep this billing group while switching between clusters.'
                )}
              </p>
            </div>
            <FieldGroup className='grid gap-4 md:grid-cols-3'>
              <Field>
                <FieldLabel htmlFor='billing-group'>
                  {t('Billing package')}
                </FieldLabel>
                <Input
                  id='billing-group'
                  value={t('Unified package')}
                  readOnly
                  aria-readonly='true'
                />
                <input
                  type='hidden'
                  value={unifiedBillingGroup}
                  {...form.register('billing_group')}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='billing-description'>
                  {t('Group description')}
                </FieldLabel>
                <Input
                  id='billing-description'
                  {...form.register('billing_group_description')}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='billing-ratio'>
                  {t('Billing ratio')}
                </FieldLabel>
                <Input
                  id='billing-ratio'
                  type='number'
                  min='0'
                  step='0.01'
                  {...form.register('billing_group_ratio')}
                />
              </Field>
            </FieldGroup>
          </section>

          <section className='space-y-4 border-t pt-5'>
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <div>
                <h3 className='text-sm font-semibold'>
                  {t('Degradation route')}
                </h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'P1-P4 are attempted in order. One channel represents one key.'
                  )}
                </p>
              </div>
              {routes.fields.length < 4 && (
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => {
                    const route = defaultRoutes[routes.fields.length]
                    if (route) routes.append({ ...route })
                  }}
                >
                  <HugeiconsIcon icon={Add01Icon} data-icon='inline-start' />
                  {t('Add route')}
                </Button>
              )}
            </div>
            <div className='overflow-x-auto rounded-md border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Order')}</TableHead>
                    <TableHead>{t('Pool')}</TableHead>
                    <TableHead>{t('Channel / key')}</TableHead>
                    <TableHead>{t('Weight')}</TableHead>
                    <TableHead>{t('Cost factor')}</TableHead>
                    <TableHead className='w-12' />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {routes.fields.map((route, index) => (
                    <TableRow key={route.id}>
                      <TableCell className='font-medium'>
                        P{index + 1}
                      </TableCell>
                      <TableCell>
                        <Input
                          className='w-36'
                          {...form.register(`routes.${index}.pool_name`)}
                        />
                        <input
                          type='hidden'
                          {...form.register(`routes.${index}.pool_tier`)}
                        />
                        <input
                          type='hidden'
                          {...form.register(`routes.${index}.route_order`)}
                        />
                      </TableCell>
                      <TableCell className='min-w-64'>
                        <NativeSelect
                          className='w-full'
                          aria-label={`${t('Channel / key')} P${index + 1}`}
                          {...form.register(`routes.${index}.channel_id`)}
                        >
                          <NativeSelectOption value='0'>
                            {t('Select channel')}
                          </NativeSelectOption>
                          {selectableChannels.map((channel) => (
                            <NativeSelectOption
                              key={channel.id}
                              value={channel.id}
                              disabled={channel.is_multi_key}
                            >
                              #{channel.id} · {channel.name}
                              {channel.cluster_id > 0
                                ? ` · C${channel.cluster_id}`
                                : ''}
                              {channel.is_multi_key
                                ? ` · ${t('Multi-key not supported')}`
                                : ''}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                      </TableCell>
                      <TableCell>
                        <Input
                          className='w-24'
                          type='number'
                          min='0'
                          max='1000'
                          {...form.register(`routes.${index}.weight`)}
                        />
                      </TableCell>
                      <TableCell>
                        <Input
                          className='w-28'
                          type='number'
                          min='0'
                          step='0.01'
                          {...form.register(`routes.${index}.cost_factor`)}
                        />
                      </TableCell>
                      <TableCell>
                        {index === routes.fields.length - 1 &&
                          routes.fields.length > 1 && (
                            <Button
                              type='button'
                              size='icon'
                              variant='ghost'
                              aria-label={t('Delete')}
                              onClick={() => routes.remove(index)}
                            >
                              <HugeiconsIcon icon={Delete02Icon} />
                            </Button>
                          )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </section>

          <Field>
            <FieldLabel htmlFor='cluster-remark'>{t('Remark')}</FieldLabel>
            <Input id='cluster-remark' {...form.register('remark')} />
          </Field>

          {Object.keys(form.formState.errors).length > 0 && (
            <Alert variant='destructive'>
              <HugeiconsIcon icon={Alert02Icon} />
              <AlertTitle>{t('Configuration validation failed')}</AlertTitle>
              <AlertDescription>
                {t('Channel binding is incomplete')}
              </AlertDescription>
            </Alert>
          )}

          <div className='flex justify-end'>
            <Button
              type='submit'
              disabled={mutation.isPending || deleteMutation.isPending}
            >
              {mutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : (
                <HugeiconsIcon icon={FloppyDiskIcon} data-icon='inline-start' />
              )}
              {mutation.isPending ? t('Saving...') : t('Save cluster')}
            </Button>
          </div>
        </form>
        <ConfirmDialog
          destructive
          open={deleteDialogOpen}
          onOpenChange={setDeleteDialogOpen}
          title={t('Are you sure?')}
          desc={t('This action cannot be undone.')}
          confirmText={
            deleteMutation.isPending ? t('Deleting...') : t('Delete')
          }
          isLoading={deleteMutation.isPending}
          handleConfirm={() => deleteMutation.mutate(currentClusterId)}
        />
      </Form>
    </div>
  )
}

export function ClusterConfigurationPanel() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['cluster-configuration'],
    queryFn: getClusterConfiguration,
  })

  if (query.isLoading) {
    return (
      <div className='space-y-3'>
        <Skeleton className='h-10 w-full' />
        <Skeleton className='h-72 w-full' />
      </div>
    )
  }
  if (query.isError) {
    return (
      <Alert variant='destructive'>
        <HugeiconsIcon icon={Alert02Icon} />
        <AlertTitle>{t('Failed to load cluster configuration')}</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
      </Alert>
    )
  }
  if (!query.data) return null

  return (
    <ClusterConfigurationWorkspace
      key={query.dataUpdatedAt}
      snapshot={query.data}
    />
  )
}
