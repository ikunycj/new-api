/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'

export type UserGroupSummary = {
  id: number
  name: string
  user_count: number
  created_at: number
  updated_at: number
  topup_ratio: number
  pricing_groups: string[]
  pricing_groups_all?: boolean
}

export type UpdateUserGroupRequest = {
  topup_ratio: number
  pricing_groups: string[]
}

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

function requireData<T>(response: ApiResponse<T>): T {
  if (!response.success || response.data === undefined) {
    throw new Error(response.message || '请求失败')
  }
  return response.data
}

export async function getUserGroupSummaries(): Promise<UserGroupSummary[]> {
  const response =
    await api.get<ApiResponse<UserGroupSummary[]>>('/api/group/manage')
  return requireData(response.data)
}

export async function createUserGroup(name: string): Promise<UserGroupSummary> {
  const response = await api.post<ApiResponse<UserGroupSummary>>(
    '/api/group/',
    { name }
  )
  return requireData(response.data)
}

export async function deleteUserGroup(name: string): Promise<void> {
  const response = await api.delete<ApiResponse<null>>(
    `/api/group/${encodeURIComponent(name)}`
  )
  if (!response.data.success) {
    throw new Error(response.data.message || '请求失败')
  }
}

export async function updateUserGroup(
  name: string,
  request: UpdateUserGroupRequest
): Promise<UserGroupSummary> {
  const response = await api.put<ApiResponse<UserGroupSummary>>(
    `/api/group/${encodeURIComponent(name)}`,
    request
  )
  return requireData(response.data)
}

export async function getPricingGroupNames(): Promise<string[]> {
  const response = await api.get<ApiResponse<string[]>>(
    '/api/group/pricing-groups'
  )
  return requireData(response.data)
}
