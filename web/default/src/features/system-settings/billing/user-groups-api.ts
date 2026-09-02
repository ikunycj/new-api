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
  active_today: number
  active_month: number
  created_at: number
  updated_at: number
  topup_ratio: number
  pricing_groups: string[]
  pricing_groups_all?: boolean
}

export type UpdateUserGroupRequest = {
  name: string
  topup_ratio: number
  pricing_groups: string[]
  pricing_groups_all: boolean
}

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

function normalizeCount(value: unknown): number {
  const count = Number(value)
  return Number.isFinite(count) ? count : 0
}

function normalizeUserGroupSummary(
  summary: UserGroupSummary
): UserGroupSummary {
  const pricingGroups = Array.isArray(summary.pricing_groups)
    ? summary.pricing_groups.filter(
        (group): group is string => typeof group === 'string'
      )
    : []
  const pricingGroupsAll =
    summary.pricing_groups_all === true || pricingGroups.length === 0

  return {
    ...summary,
    active_today: normalizeCount(summary.active_today),
    active_month: normalizeCount(summary.active_month),
    pricing_groups: pricingGroupsAll ? ['*'] : pricingGroups,
    pricing_groups_all: pricingGroupsAll,
  }
}

function requireData<T>(response: ApiResponse<T>): T {
  if (!response.success || response.data === undefined) {
    throw new Error(response.message || '请求失败')
  }
  return response.data
}

export async function getUserGroupSummaries(): Promise<UserGroupSummary[]> {
  const response = await api.get<ApiResponse<UserGroupSummary[]>>(
    '/api/group/manage',
    {
      skipErrorHandler: true,
    }
  )
  return requireData(response.data).map(normalizeUserGroupSummary)
}

export async function createUserGroup(name: string): Promise<UserGroupSummary> {
  const response = await api.post<ApiResponse<UserGroupSummary>>(
    '/api/group/',
    { name }
  )
  return normalizeUserGroupSummary(requireData(response.data))
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
  return normalizeUserGroupSummary(requireData(response.data))
}

export async function getPricingGroupNames(): Promise<string[]> {
  const response = await api.get<ApiResponse<string[]>>(
    '/api/group/pricing-groups',
    { skipErrorHandler: true }
  )
  return requireData(response.data)
}
