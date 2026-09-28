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

import { api } from '@/lib/api'

import type {
  GroupBenchArtifact,
  GroupBenchConfig,
  GroupBenchConfigPayload,
  GroupBenchRun,
} from './types'

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

function requireData<T>(response: ApiResponse<T>): T {
  if (!response.success || response.data === undefined) {
    throw new Error(response.message || 'Request failed')
  }
  return response.data
}

export async function getGroupBenchConfig(
  group: string
): Promise<GroupBenchConfig> {
  const response = await api.get<ApiResponse<GroupBenchConfig>>(
    '/api/group/bench/config',
    { params: { group } }
  )
  return requireData(response.data)
}

export async function updateGroupBenchConfig(
  payload: GroupBenchConfigPayload
): Promise<{ next_run_at: number }> {
  const response = await api.put<ApiResponse<{ next_run_at: number }>>(
    '/api/group/bench/config',
    payload
  )
  return requireData(response.data)
}

export async function runGroupBenchNow(
  group: string
): Promise<{ task_id: string; created: boolean }> {
  const response = await api.post<
    ApiResponse<{ task_id: string; created: boolean }>
  >('/api/group/bench/run', null, { params: { group } })
  return requireData(response.data)
}

export async function getGroupBenchRuns(params: {
  group: string
  channel_id?: number
  model?: string
  since?: number
  limit?: number
}): Promise<GroupBenchRun[]> {
  const response = await api.get<ApiResponse<{ items: GroupBenchRun[] }>>(
    '/api/group/bench/runs',
    { params }
  )
  return requireData(response.data).items
}

export async function getGroupBenchArtifact(
  runId: number
): Promise<GroupBenchArtifact> {
  const response = await api.get<ApiResponse<GroupBenchArtifact>>(
    `/api/group/bench/runs/${runId}/artifact`,
    { skipErrorHandler: true }
  )
  return requireData(response.data)
}
