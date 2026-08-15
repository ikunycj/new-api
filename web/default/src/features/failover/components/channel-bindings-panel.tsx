import { Alert02Icon, FloppyDiskIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
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
  getChannelFailoverBindings,
  updateChannelFailoverBindings,
} from '../api'
import type { ChannelFailoverBinding, Cluster, ClusterPool } from '../types'

type ChannelBindingsEditorProps = {
  bindings: ChannelFailoverBinding[]
  clusters: Cluster[]
  pools: ClusterPool[]
}

function ChannelBindingsEditor(props: ChannelBindingsEditorProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [bindings, setBindings] = useState(props.bindings)
  const activeClusters = props.clusters.filter((cluster) => !cluster.archived)
  const hasIncompleteBinding = bindings.some(
    (binding) => (binding.cluster_id === 0) !== (binding.cluster_pool_id === 0)
  )
  const mutation = useMutation({
    mutationFn: () =>
      updateChannelFailoverBindings(
        bindings.map((binding) => ({
          channel_id: binding.channel_id,
          cluster_id: binding.cluster_id,
          cluster_pool_id: binding.cluster_pool_id,
        }))
      ),
    onSuccess: async () => {
      toast.success(t('Channel bindings saved'))
      await queryClient.invalidateQueries({
        queryKey: ['channel-failover-bindings'],
      })
    },
    onError: (error) => toast.error(error.message || t('Save failed')),
  })

  return (
    <div className='space-y-4'>
      {activeClusters.length === 0 && (
        <Alert>
          <HugeiconsIcon icon={Alert02Icon} />
          <AlertTitle>{t('No clusters available')}</AlertTitle>
          <AlertDescription>
            {t('Save a cluster and its pools before binding channels.')}
          </AlertDescription>
        </Alert>
      )}
      {hasIncompleteBinding && (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={Alert02Icon} />
          <AlertTitle>{t('Channel binding is incomplete')}</AlertTitle>
          <AlertDescription>
            {t('Select a pool for every assigned cluster.')}
          </AlertDescription>
        </Alert>
      )}
      <div className='overflow-x-auto rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Channel')}</TableHead>
              <TableHead>{t('Base URL')}</TableHead>
              <TableHead>{t('Cluster')}</TableHead>
              <TableHead>{t('Pool')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {bindings.map((binding) => {
              const clusterPools = props.pools.filter(
                (pool) => pool.cluster_id === binding.cluster_id
              )
              return (
                <TableRow key={binding.channel_id}>
                  <TableCell className='min-w-44 font-medium'>
                    <div>{binding.channel_name}</div>
                    <div className='text-muted-foreground text-xs tabular-nums'>
                      #{binding.channel_id}
                    </div>
                  </TableCell>
                  <TableCell className='text-muted-foreground max-w-80 min-w-52 truncate font-mono text-xs'>
                    {binding.base_url || t('Default')}
                  </TableCell>
                  <TableCell className='min-w-56'>
                    <NativeSelect
                      className='w-full'
                      aria-label={`${t('Cluster')}: ${binding.channel_name}`}
                      value={String(binding.cluster_id)}
                      onChange={(event) => {
                        const clusterId = Number(event.target.value)
                        setBindings((current) =>
                          current.map((item) => {
                            if (item.channel_id !== binding.channel_id) {
                              return item
                            }
                            const selectedPool = props.pools.find(
                              (pool) => pool.id === item.cluster_pool_id
                            )
                            return {
                              ...item,
                              cluster_id: clusterId,
                              cluster_pool_id:
                                selectedPool?.cluster_id === clusterId
                                  ? item.cluster_pool_id
                                  : 0,
                            }
                          })
                        )
                      }}
                    >
                      <NativeSelectOption value='0'>
                        {t('Unassigned')}
                      </NativeSelectOption>
                      {activeClusters.map((cluster) => (
                        <NativeSelectOption
                          key={cluster.id}
                          value={String(cluster.id)}
                        >
                          C{cluster.id} · {cluster.name}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </TableCell>
                  <TableCell className='min-w-56'>
                    <NativeSelect
                      className='w-full'
                      aria-label={`${t('Pool')}: ${binding.channel_name}`}
                      value={String(binding.cluster_pool_id)}
                      disabled={binding.cluster_id === 0}
                      onChange={(event) => {
                        const poolId = Number(event.target.value)
                        setBindings((current) =>
                          current.map((item) =>
                            item.channel_id === binding.channel_id
                              ? { ...item, cluster_pool_id: poolId }
                              : item
                          )
                        )
                      }}
                    >
                      <NativeSelectOption value='0'>
                        {binding.cluster_id === 0
                          ? t('Unassigned')
                          : t('Select pool')}
                      </NativeSelectOption>
                      {clusterPools.map((pool) => (
                        <NativeSelectOption
                          key={pool.id}
                          value={String(pool.id)}
                        >
                          P{pool.tier} · {pool.name}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={binding.status === 1 ? 'default' : 'secondary'}
                    >
                      {binding.status === 1 ? t('Enabled') : t('Disabled')}
                    </Badge>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>
      <div className='flex justify-end'>
        <Button
          type='button'
          disabled={
            mutation.isPending || hasIncompleteBinding || bindings.length === 0
          }
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? (
            <Spinner data-icon='inline-start' />
          ) : (
            <HugeiconsIcon icon={FloppyDiskIcon} data-icon='inline-start' />
          )}
          {mutation.isPending ? t('Saving...') : t('Save bindings')}
        </Button>
      </div>
    </div>
  )
}

type ChannelBindingsPanelProps = {
  clusters: Cluster[]
  pools: ClusterPool[]
}

export function ChannelBindingsPanel(props: ChannelBindingsPanelProps) {
  const { t } = useTranslation()
  const bindingsQuery = useQuery({
    queryKey: ['channel-failover-bindings'],
    queryFn: getChannelFailoverBindings,
  })

  return (
    <section className='space-y-4'>
      <div>
        <h2 className='text-base font-semibold'>{t('Channel bindings')}</h2>
        <p className='text-muted-foreground text-sm'>
          {t('Link each channel to a cluster and one of its P1-P3 pools.')}
        </p>
      </div>
      {bindingsQuery.isLoading && (
        <div className='space-y-3'>
          <Skeleton className='h-10 w-full' />
          <Skeleton className='h-40 w-full' />
        </div>
      )}
      {bindingsQuery.isError && (
        <Alert variant='destructive'>
          <HugeiconsIcon icon={Alert02Icon} />
          <AlertTitle>{t('Failed to load channel bindings')}</AlertTitle>
          <AlertDescription>{bindingsQuery.error.message}</AlertDescription>
        </Alert>
      )}
      {bindingsQuery.data && (
        <ChannelBindingsEditor
          key={bindingsQuery.dataUpdatedAt}
          bindings={bindingsQuery.data}
          clusters={props.clusters}
          pools={props.pools}
        />
      )}
    </section>
  )
}
