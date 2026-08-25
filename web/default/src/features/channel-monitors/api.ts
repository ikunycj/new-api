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

import { isAxiosError } from 'axios'

import { api } from '@/lib/api'

import type {
  ChannelMonitor,
  ChannelMonitorCreatePayload,
  ChannelMonitorRunResponse,
  ChannelMonitorSettingsPayload,
  GroupStatusMonitor,
  GroupStatusTestResponse,
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

export async function getChannelMonitors(): Promise<ChannelMonitor[]> {
  const response = await api.get<ApiResponse<{ items: ChannelMonitor[] }>>(
    '/api/monitor/channel/'
  )
  return requireData(response.data).items
}

export async function getPricingGroupChannelCount(
  pricingGroup: string
): Promise<number> {
  const response = await api.get<
    ApiResponse<{
      total: number
    }>
  >('/api/channel', {
    params: {
      p: 1,
      page_size: 1,
      group: pricingGroup,
    },
  })
  return Math.max(1, requireData(response.data).total)
}

export async function createChannelMonitor(
  payload: ChannelMonitorCreatePayload
): Promise<ChannelMonitor> {
  const response = await api.post<ApiResponse<ChannelMonitor>>(
    '/api/monitor/channel/',
    payload
  )
  return requireData(response.data)
}

export async function updateChannelMonitor(
  id: number,
  payload: ChannelMonitorSettingsPayload
): Promise<ChannelMonitor> {
  const response = await api.put<ApiResponse<ChannelMonitor>>(
    `/api/monitor/channel/${id}`,
    payload
  )
  return requireData(response.data)
}

export async function runChannelMonitor(
  id: number
): Promise<ChannelMonitorRunResponse> {
  const response = await api.post<ApiResponse<ChannelMonitorRunResponse>>(
    `/api/monitor/channel/${id}/run`
  )
  return requireData(response.data)
}

export async function getGroupStatus(): Promise<GroupStatusMonitor[]> {
  const response =
    await api.get<ApiResponse<{ items: GroupStatusMonitor[] }>>(
      '/api/group-status/'
    )
  return requireData(response.data).items
}

export class GroupStatusTestCooldownError extends Error {
  readonly nextTestAt: number

  constructor(nextTestAt: number) {
    super('Availability test is cooling down')
    this.name = 'GroupStatusTestCooldownError'
    this.nextTestAt = nextTestAt
  }
}

export async function runGroupStatusTest(
  id: number
): Promise<GroupStatusTestResponse> {
  try {
    const response = await api.post<ApiResponse<GroupStatusTestResponse>>(
      `/api/group-status/${id}/test`,
      undefined,
      { skipErrorHandler: true }
    )
    return requireData(response.data)
  } catch (error) {
    if (isAxiosError(error) && error.response?.status === 429) {
      const cooldown = error.response.data as ApiResponse<{
        retry_after?: number
        next_test_at?: number
      }>
      const retryAfter = Math.max(1, cooldown.data?.retry_after ?? 1)
      const nextTestAt =
        cooldown.data?.next_test_at ??
        Math.floor(Date.now() / 1000) + retryAfter
      throw new GroupStatusTestCooldownError(nextTestAt)
    }
    throw error
  }
}
