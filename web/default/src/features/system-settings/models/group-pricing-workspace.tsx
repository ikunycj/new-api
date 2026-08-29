import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getChannels } from '@/features/channels/api'
import { getFailoverConfig } from '@/features/failover/api'

import {
  buildGroupPricingSnapshots,
  getToBGroupNames,
} from './group-pricing-utils'
import { GroupRatioForm } from './group-ratio-form'
import { ToBRoutingSection } from './tob-routing-section'
import { ToCGroupsSection } from './toc-groups-section'

export type GroupPricingValues = {
  GroupRatio: string
  TopupGroupRatio: string
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
  const configQuery = useQuery({
    queryKey: ['channel-routing-config'],
    queryFn: getFailoverConfig,
  })
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
  const groupRatio = props.form.watch('GroupRatio')
  const userUsableGroups = props.form.watch('UserUsableGroups')
  const groups = useMemo(
    () =>
      buildGroupPricingSnapshots(groupRatio, userUsableGroups),
    [groupRatio, userUsableGroups]
  )
  const classifiedGroups = useMemo(() => {
    const byName = new Map(groups.map((group) => [group.name, group]))
    for (const name of toBGroupNames) {
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
  }, [groups, toBGroupNames])
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
        />
      </TabsContent>
      <TabsContent value='tob'>
        <ToBRoutingSection
          config={configQuery.data}
          channels={channels}
          groupNames={groupNames}
          groupRatios={groupRatios}
          isLoading={configQuery.isLoading || channelsQuery.isLoading}
        />
      </TabsContent>
      <TabsContent value='toc'>
        <ToCGroupsSection
          groups={groups}
          channels={channels}
          toBGroupNames={toBGroupNames}
        />
      </TabsContent>
    </Tabs>
  )
}
