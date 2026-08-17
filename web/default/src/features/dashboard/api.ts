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
import type { PlanRecord } from '@/features/subscriptions/types'
import { api } from '@/lib/api'

import type {
  FlowQuotaDataItem,
  QuotaDataItem,
  UptimeGroupResult,
} from './types'

export interface PackageComparisonStat {
  plan_id: number
  plan_title: string
  plan_price: number
  currency: string
  routing_strategy: string
  plan_quota: number
  requests: number
  success_requests: number
  error_requests: number
  success_rate: number
  failure_rate: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  quota: number
  average_latency_ms: number
  channel_hit_rate: number
  quota_per_million_tokens: number
}

export async function getAdminPlans(): Promise<{
  success: boolean
  data?: PlanRecord[]
}> {
  const res = await api.get<{ success: boolean; data?: PlanRecord[] }>(
    '/api/subscription/admin/plans'
  )
  return res.data
}

export async function getPackageComparison(params: {
  plan_ids: number[]
  start_timestamp: number
  end_timestamp: number
  model_name?: string
  group?: string
}) {
  const res = await api.get<{
    success: boolean
    data?: { plans: PackageComparisonStat[] }
  }>('/api/admin/analytics/package-comparison', {
    params: { ...params, plan_ids: params.plan_ids.join(',') },
  })
  return res.data
}

// ============================================================================
// Dashboard APIs
// ============================================================================

// ----------------------------------------------------------------------------
// Quota & Usage Data
// ----------------------------------------------------------------------------

// Get user quota data within a time range
// Admin users get all users' data by default.
export async function getUserQuotaDates(
  params: {
    start_timestamp: number
    end_timestamp: number
    default_time?: string
    username?: string
  },
  isAdmin = false
) {
  const endpoint = isAdmin ? '/api/data' : '/api/data/self'
  const res = await api.get<{ success: boolean; data: QuotaDataItem[] }>(
    endpoint,
    { params }
  )
  return res.data
}

// ----------------------------------------------------------------------------
// System Monitoring
// ----------------------------------------------------------------------------

export async function getUserQuotaDataByUsers(params: {
  start_timestamp: number
  end_timestamp: number
}) {
  const res = await api.get<{ success: boolean; data: QuotaDataItem[] }>(
    '/api/data/users',
    { params }
  )
  return res.data
}

export async function getFlowQuotaDates(
  params: {
    start_timestamp: number
    end_timestamp: number
    default_time?: string
    username?: string
  },
  isAdmin = false
) {
  const endpoint = isAdmin ? '/api/data/flow' : '/api/data/flow/self'
  const res = await api.get<{
    success: boolean
    data?: FlowQuotaDataItem[]
    message?: string
  }>(endpoint, { params })
  return res.data
}

// Get uptime monitoring status for all services
export async function getUptimeStatus() {
  const res = await api.get<{ success: boolean; data: UptimeGroupResult[] }>(
    '/api/uptime/status'
  )
  return res.data
}
