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

export interface AdminConsoleKeyStats {
  total: number
  active: number
  enabled: number
}

export interface AdminConsoleChannelStats {
  total: number
  enabled: number
  auto_disabled: number
}

export interface AdminConsolePeriodStats {
  today: number
  month: number
  total: number
}

export interface AdminConsoleUserStats {
  today: number
  total: number
  active_today: number
  active_week: number
  active_month: number
}

export interface AdminConsoleRevenueStats {
  today: number
  month: number
  total: number
}

export interface AdminConsolePerformanceStats {
  average_response_seconds: number
  today_response_p50_seconds: number
  today_response_p90_seconds: number
  today_response_p99_seconds: number
  today_rpm_p50: number
  today_rpm_p90: number
  today_rpm_p99: number
  today_tpm_p50: number
  today_tpm_p90: number
  today_tpm_p99: number
  concurrency_p50: number
  concurrency_p90: number
  concurrency_p99: number
  month_concurrency_p50: number
  month_concurrency_p90: number
  month_concurrency_p95: number
}

export interface AdminConsoleRealtimeStats {
  current_concurrency: number
  response_seconds: number
  rpm: number
  tpm: number
}

export interface AdminConsoleSystemLoad {
  cpu_usage_percent: number
  memory_usage_percent: number
  storage_usage_percent: number
}

export interface AdminConsoleStats {
  api_keys: AdminConsoleKeyStats
  channels: AdminConsoleChannelStats
  requests: AdminConsolePeriodStats
  users: AdminConsoleUserStats
  tokens: AdminConsolePeriodStats
  quota: AdminConsolePeriodStats
  revenue: AdminConsoleRevenueStats
  performance: AdminConsolePerformanceStats
  system_load: AdminConsoleSystemLoad
}

export interface AdminConsoleDataState {
  stats?: AdminConsoleStats
  realtimeStats?: AdminConsoleRealtimeStats
  systemLoad?: AdminConsoleSystemLoad
  statsLoading: boolean
  realtimeStatsLoading: boolean
  systemLoadLoading: boolean
  statsError: boolean
  realtimeStatsError: boolean
  systemLoadError: boolean
}

export interface AdminConsoleResponse {
  success: boolean
  message?: string
  data?: AdminConsoleStats
}

export interface AdminConsoleSystemLoadResponse {
  success: boolean
  message?: string
  data?: AdminConsoleSystemLoad
}

export interface AdminConsoleRealtimeResponse {
  success: boolean
  message?: string
  data?: AdminConsoleRealtimeStats
}

export type AdminConsoleCacheTrendDimension = 'group' | 'channel'

export interface AdminConsoleCacheTrendPoint {
  timestamp: number
  name: string
  channel_id?: number
  cache_input_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  cache_hit_requests: number
  cache_eligible_requests: number
  cache_hit_rate: number
}

export interface AdminConsoleCacheTrendResponse {
  success: boolean
  message?: string
  data?: AdminConsoleCacheTrendPoint[]
}
