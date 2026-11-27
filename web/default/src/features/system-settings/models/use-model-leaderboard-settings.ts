/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { useState } from 'react'
import { toast } from 'sonner'

import {
  MODEL_LEADERBOARDS_QUERY_KEY,
  type ModelLeaderboards,
} from '@/hooks/use-model-leaderboards'
import { api } from '@/lib/api'

type LeaderboardSettings = {
  enabled: boolean
  recommended_models: string[]
}

type LeaderboardResponse = {
  success: boolean
  message: string
  data?: ModelLeaderboards & LeaderboardSettings
}

const leaderboardsPath = '/api/option/model_leaderboards'
const settingsQueryKey = ['admin', 'model-leaderboards'] as const
function readLeaderboardResponse(response: LeaderboardResponse) {
  if (!response.success || !response.data) {
    throw new Error(response.message || '无法读取模型排序榜单')
  }
  return response.data
}

export function leaderboardErrorMessage(error: Error, fallback: string) {
  if (isAxiosError<LeaderboardResponse>(error)) {
    return error.response?.data?.message || fallback
  }
  return error.message || fallback
}

export function useModelLeaderboardSettings() {
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<{
    enabled?: boolean
    recommendedModels?: string[]
  }>({})
  const query = useQuery({
    queryKey: settingsQueryKey,
    queryFn: async () => {
      const { data } = await api.get<LeaderboardResponse>(leaderboardsPath, {
        skipBusinessError: true,
        skipErrorHandler: true,
      })
      return readLeaderboardResponse(data)
    },
  })

  function refreshModelOrder(data: ModelLeaderboards & LeaderboardSettings) {
    queryClient.setQueryData(settingsQueryKey, data)
    return Promise.all([
      queryClient.invalidateQueries({ queryKey: MODEL_LEADERBOARDS_QUERY_KEY }),
      queryClient.invalidateQueries({ queryKey: ['status'] }),
    ])
  }

  const save = useMutation({
    mutationFn: async (settings: LeaderboardSettings) => {
      const { data } = await api.put<LeaderboardResponse>(
        leaderboardsPath,
        settings,
        { skipBusinessError: true, skipErrorHandler: true }
      )
      return readLeaderboardResponse(data)
    },
    onSuccess: (data) => {
      setDraft({})
      toast.success('已保存榜单设置')
      return refreshModelOrder(data)
    },
    onError: (error) =>
      toast.error(leaderboardErrorMessage(error, '保存榜单设置失败')),
  })

  const sync = useMutation({
    mutationFn: async () => {
      const { data } = await api.post<LeaderboardResponse>(
        `${leaderboardsPath}/sync`,
        undefined,
        { timeout: 45_000, skipBusinessError: true, skipErrorHandler: true }
      )
      return readLeaderboardResponse(data)
    },
    onSuccess: (data) => {
      toast.success('已同步三个能力榜单')
      return refreshModelOrder(data)
    },
    // Keep the upstream failure visible in the section until the next attempt.
    onError: () => {},
  })

  const saved = query.data
  const enabled = draft.enabled ?? saved?.enabled
  const savedRecommendedModels = saved?.recommended_models ?? []
  const recommendedModels = draft.recommendedModels ?? savedRecommendedModels
  const hasChanges =
    enabled !== saved?.enabled ||
    recommendedModels.length !== savedRecommendedModels.length ||
    recommendedModels.some(
      (model, index) => model !== savedRecommendedModels[index]
    )

  async function saveChanges() {
    if (enabled !== undefined && hasChanges) {
      await save.mutateAsync({ enabled, recommended_models: recommendedModels })
    }
  }

  return {
    query,
    sync,
    enabled,
    recommendedModels,
    setDraft,
    hasChanges,
    saveChanges,
  }
}
