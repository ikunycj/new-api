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

import type {
  AdminConsoleResponse,
  AdminConsoleStats,
  AdminConsoleSystemLoad,
  AdminConsoleSystemLoadResponse,
} from './types'

export async function getAdminConsoleStats(): Promise<AdminConsoleStats> {
  const response = await api.get<AdminConsoleResponse>('/api/admin/console')
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
