import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getChannels } from '@/features/channels/api'
import {
  getFailoverConfig,
  updateFailoverConfig,
} from '@/features/failover/api'
import type { BillingGroupRoute } from '@/features/failover/types'

import { getOptionValue, useSystemOptions } from '../hooks/use-system-options'
import { safeJsonParse } from '../utils/json-parser'
import {
  buildGroupPricingSnapshots,
  getToBGroupNames,
} from './group-pricing-utils'
import { GroupRatioForm } from './group-ratio-form'
import { ToBRoutingSection } from './tob-routing-section'
import { ToCGroupsSection } from './toc-groups-section'

export type GroupPricingValues = {
  GroupRatio: string
  UserUsableGroups: string
  GroupGroupRatio: string
  AutoGroups: string
  DefaultUseAutoGroup: boolean
  GroupSpecialUsableGroup: string
}

type GroupPricingWorkspaceProps = {
  form: UseFormReturn<GroupPricingValues>
  onSave: (values: GroupPricingValues) => Promise<void>
  isSaving: boolean
}

export function GroupPricingWorkspace(props: GroupPricingWorkspaceProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const configQuery = useQuery({
    queryKey: ['channel-routing-config'],
    queryFn: getFailoverConfig,
  })
  const systemOptionsQuery = useSystemOptions()
  const circuitEnabled = useMemo(
    () =>
      getOptionValue(systemOptionsQuery.data?.data, {
        ChannelCircuitEnabled: false,
      }).ChannelCircuitEnabled,
    [systemOptionsQuery.data?.data]
  )
  const channelsQuery = useQuery({
    queryKey: ['channel-routing-channels'],
    queryFn: () => getChannels({ page_size: 1000, id_sort: true }),
  })
  const channels = useMemo(
    () => channelsQuery.data?.data?.items ?? [],
    [channelsQuery.data]
  )
  const toBGroupNames = useMemo(
    () => getToBGroupNames(configQuery.data?.routes ?? []),
    [configQuery.data?.routes]
  )
  const routeGroupNames = useMemo(
    () =>
      new Set(
        (configQuery.data?.routes ?? [])
          .map((route) => route.billing_group.trim())
          .filter(Boolean)
      ),
    [configQuery.data?.routes]
  )
  const routeGroupNameList = useMemo(
    () => [...routeGroupNames],
    [routeGroupNames]
  )
  const groupRatio = props.form.watch('GroupRatio')
  const userUsableGroups = props.form.watch('UserUsableGroups')
  useEffect(() => {
    const ratioMap = safeJsonParse<Record<string, number>>(groupRatio, {
      fallback: {},
      silent: true,
    })
    const missingRouteGroups = routeGroupNameList.filter(
      (name) => !Object.hasOwn(ratioMap, name)
    )
    if (missingRouteGroups.length === 0) return
    const nextRatioMap = { ...ratioMap }
    for (const name of missingRouteGroups) nextRatioMap[name] = 1
    props.form.setValue('GroupRatio', JSON.stringify(nextRatioMap, null, 2), {
      shouldDirty: true,
      shouldValidate: true,
    })
  }, [groupRatio, props.form, routeGroupNameList])
  const groups = useMemo(
    () => buildGroupPricingSnapshots(groupRatio, userUsableGroups),
    [groupRatio, userUsableGroups]
  )
  const classifiedGroups = useMemo(() => {
    const byName = new Map(groups.map((group) => [group.name, group]))
    for (const name of routeGroupNames) {
      if (!byName.has(name)) {
        byName.set(name, {
          name,
          ratio: 1,
          topupRatio: null,
          selectable: false,
          description: '',
        })
      }
    }
    return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name))
  }, [groups, routeGroupNames])
  const groupNames = useMemo(
    () => classifiedGroups.map((group) => group.name),
    [classifiedGroups]
  )
  const groupRatios = useMemo(
    () => new Map(groups.map((group) => [group.name, group.ratio])),
    [groups]
  )

  const groupTypes = useMemo(
    () =>
      new Map(
        classifiedGroups.map(
          (group) =>
            [group.name, toBGroupNames.has(group.name) ? 'ToB' : 'ToC'] as const
        )
      ),
    [classifiedGroups, toBGroupNames]
  )
  const groupTypeByName = useMemo(
    (): ReadonlyMap<string, 'toB' | 'toC'> =>
      new Map<string, 'toB' | 'toC'>(
        classifiedGroups.map((group) => [
          group.name,
          (groupTypes.get(group.name) ?? 'ToC') === 'ToB' ? 'toB' : 'toC',
        ])
      ),
    [classifiedGroups, groupTypes]
  )
  const groupTypeMutation = useMutation({
    mutationFn: updateFailoverConfig,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['channel-routing-config'],
      })
    },
  })
  const handleGroupTypeChange = (name: string, type: 'toB' | 'toC') => {
    if (!name || !configQuery.data || groupTypeMutation.isPending) return
    const routes = structuredClone(configQuery.data.routes)
    const existing = routes.find((route) => route.billing_group === name)
    if (existing) {
      if (existing.group_type === type) return
      existing.group_type = type
    } else if (type === 'toB') {
      if (!configQuery.data.circuit_defaults) return
      const route: BillingGroupRoute = {
        id: -Date.now(),
        billing_group: name,
        name,
        mode: 'balanced',
        group_type: type,
        strategy_config: JSON.stringify({ type: 'priority' }),
        enabled: false,
        max_total_attempts: 4,
        total_timeout_ms: 30000,
        circuit_failure_threshold:
          configQuery.data.circuit_defaults.failure_threshold,
        circuit_window_seconds: configQuery.data.circuit_defaults.window_seconds,
        circuit_cooldown_seconds:
          configQuery.data.circuit_defaults.cooldown_seconds,
        circuit_half_open_requests:
          configQuery.data.circuit_defaults.half_open_requests,
        profit_guard_mode: 'off',
        minimum_profit_margin: 0,
        created_time: 0,
        updated_time: 0,
      }
      routes.push(route)
    } else {
      return
    }
    groupTypeMutation.mutate({ ...configQuery.data, routes })
  }
  const handleGroupRename = (previousName: string, nextName: string) => {
    if (
      !previousName ||
      !nextName ||
      previousName === nextName ||
      !configQuery.data ||
      groupTypeMutation.isPending
    ) {
      return
    }
    const routes = structuredClone(configQuery.data.routes)
    const route = routes.find(
      (candidate) => candidate.billing_group === previousName
    )
    if (
      !route ||
      routes.some((candidate) => candidate.billing_group === nextName)
    ) {
      return
    }
    route.billing_group = nextName
    route.name = nextName
    groupTypeMutation.mutate({ ...configQuery.data, routes })
  }

  return (
    <Tabs defaultValue='basic' className='gap-5'>
      <TabsList>
        <TabsTrigger value='basic'>{t('Basic information')}</TabsTrigger>
        <TabsTrigger value='tob'>{t('ToB')}</TabsTrigger>
        <TabsTrigger value='toc'>{t('ToC')}</TabsTrigger>
      </TabsList>
      <TabsContent value='basic' className='space-y-5'>
        <section className='space-y-3'>
          <div>
            <h3 className='font-medium'>{t('Group type')}</h3>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Groups with channel routing are ToB; other billing groups are ToC.'
              )}
            </p>
          </div>
          <div className='overflow-x-auto rounded-md border'>
            <table className='w-full text-sm'>
              <thead className='bg-muted/50 text-left'>
                <tr>
                  <th className='px-3 py-2 font-medium'>{t('Group name')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Group type')}</th>
                </tr>
              </thead>
              <tbody>
                {classifiedGroups.map((group) => (
                  <tr key={group.name} className='border-t'>
                    <td className='px-3 py-2'>{group.name}</td>
                    <td className='px-3 py-2'>
                      {t(groupTypes.get(group.name) ?? 'ToC')}
                    </td>
                  </tr>
                ))}
                {classifiedGroups.length === 0 ? (
                  <tr>
                    <td className='text-muted-foreground px-3 py-4' colSpan={2}>
                      {t('No billing groups configured')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>
        <GroupRatioForm
          form={props.form}
          onSave={props.onSave}
          isSaving={props.isSaving}
          additionalGroupNames={routeGroupNameList}
          groupTypeByName={groupTypeByName}
          onGroupTypeChange={handleGroupTypeChange}
          onGroupRename={handleGroupRename}
        />
      </TabsContent>
      <TabsContent value='tob'>
        <ToBRoutingSection
          config={configQuery.data}
          circuitDefaults={configQuery.data?.circuit_defaults}
          circuitPresets={configQuery.data?.circuit_presets}
          circuitEnabled={circuitEnabled}
          channels={channels}
          groupNames={groupNames}
          groupRatios={groupRatios}
          isLoading={configQuery.isLoading || channelsQuery.isLoading}
        />
      </TabsContent>
      <TabsContent value='toc'>
        <ToCGroupsSection
          groups={classifiedGroups}
          channels={channels}
          toBGroupNames={toBGroupNames}
        />
      </TabsContent>
    </Tabs>
  )
}
