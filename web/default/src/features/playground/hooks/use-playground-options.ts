import { useQuery } from '@tanstack/react-query'
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
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { getUserGroups, getUserModels } from '../api'
import {
  getGroupFallback,
  getModelFallback,
  getOptionLoadErrorMessage,
  shouldClearModelForGroup,
} from '../lib'
import type { GroupOption, ModelOption, PlaygroundConfig } from '../types'

type UsePlaygroundOptionsParams = {
  currentGroup: string
  currentModel: string
  preferredModel?: string
  setGroups: (groups: GroupOption[]) => void
  setModels: (models: ModelOption[]) => void
  updateConfig: <K extends keyof PlaygroundConfig>(
    key: K,
    value: PlaygroundConfig[K]
  ) => void
}

export function usePlaygroundOptions({
  currentGroup,
  currentModel,
  preferredModel,
  setGroups,
  setModels,
  updateConfig,
}: UsePlaygroundOptionsParams) {
  const { t } = useTranslation()
  const preferredModels = useSystemConfigStore(
    (state) => state.config.preferredModels
  )

  const {
    data: modelsData,
    error: modelsError,
    isError: isModelsError,
    isLoading: isLoadingModels,
  } = useQuery({
    queryKey: ['playground-models', currentGroup],
    queryFn: () => getUserModels(currentGroup),
    enabled: currentGroup !== '',
  })

  const {
    data: groupsData,
    error: groupsError,
    isError: isGroupsError,
  } = useQuery({
    queryKey: ['playground-groups'],
    queryFn: getUserGroups,
  })

  const shouldResolvePreferredGroup = Boolean(
    preferredModel &&
    currentModel === preferredModel &&
    modelsData &&
    !modelsData.some((model) => model.value === preferredModel)
  )
  const hasLoadedGroups = groupsData !== undefined
  const candidateGroups =
    groupsData?.filter((group) => group.value !== currentGroup) ?? []
  const {
    data: preferredGroup,
    isFetched: hasResolvedPreferredGroup,
    isFetching: isResolvingPreferredGroup,
  } = useQuery({
    queryKey: [
      'playground-preferred-model-group',
      preferredModel,
      currentGroup,
      candidateGroups.map((group) => group.value),
    ],
    queryFn: async () => {
      if (!preferredModel) return null

      const modelsByGroup = await Promise.all(
        candidateGroups.map(async (group) => ({
          group: group.value,
          models: await getUserModels(group.value),
        }))
      )

      return (
        modelsByGroup.find(({ models }) =>
          models.some((model) => model.value === preferredModel)
        )?.group ?? null
      )
    },
    enabled: shouldResolvePreferredGroup && candidateGroups.length > 0,
  })

  useEffect(() => {
    if (!isModelsError) return

    toast.error(
      getOptionLoadErrorMessage(
        modelsError,
        t('Failed to load playground models')
      )
    )
  }, [isModelsError, modelsError, t])

  useEffect(() => {
    if (!isGroupsError) return

    toast.error(
      getOptionLoadErrorMessage(
        groupsError,
        t('Failed to load playground groups')
      )
    )
  }, [isGroupsError, groupsError, t])

  useEffect(() => {
    if (!modelsData) return

    setModels(modelsData)

    if (shouldResolvePreferredGroup) {
      if (!hasLoadedGroups) return

      const hasGroupsToCheck = candidateGroups.length > 0
      if (
        preferredGroup ||
        (hasGroupsToCheck &&
          (!hasResolvedPreferredGroup || isResolvingPreferredGroup))
      ) {
        return
      }
    }

    const fallback = getModelFallback(modelsData, currentModel)

    if (fallback) {
      updateConfig('model', fallback)
      return
    }

    if (shouldClearModelForGroup(modelsData, currentModel)) {
      updateConfig('model', '')
    }
  }, [
    candidateGroups.length,
    currentModel,
    hasLoadedGroups,
    hasResolvedPreferredGroup,
    isResolvingPreferredGroup,
    modelsData,
    preferredGroup,
    setModels,
    shouldResolvePreferredGroup,
    updateConfig,
    preferredModels,
  ])

  useEffect(() => {
    if (
      !preferredGroup ||
      preferredGroup === currentGroup ||
      currentModel !== preferredModel
    ) {
      return
    }

    updateConfig('group', preferredGroup)
  }, [currentGroup, currentModel, preferredGroup, preferredModel, updateConfig])

  useEffect(() => {
    if (!groupsData) return

    setGroups(groupsData)
    const fallback = getGroupFallback(groupsData, currentGroup)

    if (fallback) {
      updateConfig('group', fallback)
    }
  }, [groupsData, currentGroup, setGroups, updateConfig])

  return {
    isLoadingModels: isLoadingModels || isResolvingPreferredGroup,
  }
}
