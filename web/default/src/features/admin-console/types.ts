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

export interface AdminConsoleAccountStats {
  total: number
  enabled: number
  auto_disabled: number
}

export interface AdminConsolePeriodStats {
  today: number
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
  rpm: number
  tpm: number
  average_response_seconds: number
}

export interface AdminConsoleStats {
  api_keys: AdminConsoleKeyStats
  accounts: AdminConsoleAccountStats
  requests: AdminConsolePeriodStats
  users: AdminConsoleUserStats
  tokens: AdminConsolePeriodStats
  quota: AdminConsolePeriodStats
  revenue: AdminConsoleRevenueStats
  performance: AdminConsolePerformanceStats
}

export interface AdminConsoleResponse {
  success: boolean
  message?: string
  data?: AdminConsoleStats
}
