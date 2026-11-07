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
import type { TimeGranularity } from '@/lib/time'

import type {
  AdminConsoleResponse,
  AdminConsoleCacheTrendDimension,
  AdminConsoleCacheTrendPoint,
  AdminConsoleCacheTrendResponse,
  AdminConsoleRealtimeResponse,
  AdminConsoleRealtimeStats,
  AdminConsoleStats,
  AdminConsoleSystemLoad,
  AdminConsoleSystemLoadResponse,
} from './types'

export interface AdminConsoleTimeRange {
  start_timestamp?: number
  end_timestamp?: number
}

export interface AdminConsoleCacheTrendParams extends AdminConsoleTimeRange {
  dimension: AdminConsoleCacheTrendDimension
  granularity: Extract<TimeGranularity, 'hour' | 'day' | 'week'>
  timezone_offset: number
}

export async function getAdminConsoleStats(
  params?: AdminConsoleTimeRange
): Promise<AdminConsoleStats> {
  const response = await api.get<AdminConsoleResponse>('/api/admin/console', {
    params,
  })
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || '管理控制台数据加载失败')
  }
  return response.data.data
}

export async function getAdminConsoleSystemLoad(): Promise<AdminConsoleSystemLoad> {
  const response = await api.get<AdminConsoleSystemLoadResponse>(
    '/api/admin/system-load'
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || '系统负载数据加载失败')
  }
  return response.data.data
}

export async function getAdminConsoleRealtimeStats(
  params?: AdminConsoleTimeRange
): Promise<AdminConsoleRealtimeStats> {
  const response = await api.get<AdminConsoleRealtimeResponse>(
    '/api/admin/realtime',
    { params }
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || '实时统计数据加载失败')
  }
  return response.data.data
}

export async function getAdminConsoleCacheTrend(
  params: AdminConsoleCacheTrendParams
): Promise<AdminConsoleCacheTrendPoint[]> {
  const response = await api.get<AdminConsoleCacheTrendResponse>(
    '/api/log/cache-trend',
    { params }
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || '缓存命中率数据加载失败')
  }
  return response.data.data
}
