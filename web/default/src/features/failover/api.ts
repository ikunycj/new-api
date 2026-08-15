import { api } from '@/lib/api'

import type {
  ChannelFailoverBinding,
  ChannelFailoverBindingUpdate,
  FailoverConfig,
  FailoverMonitoringSnapshot,
} from './types'

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export async function getFailoverConfig(): Promise<FailoverConfig> {
  const response = await api.get<ApiResponse<FailoverConfig>>(
    '/api/channel/failover/config'
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Request failed')
  }
  return response.data.data
}

export async function updateFailoverConfig(
  config: FailoverConfig
): Promise<void> {
  const response = await api.put<ApiResponse<null>>(
    '/api/channel/failover/config',
    config
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Request failed')
  }
}

export async function getChannelFailoverBindings(): Promise<
  ChannelFailoverBinding[]
> {
  const response = await api.get<ApiResponse<ChannelFailoverBinding[]>>(
    '/api/channel/failover/bindings'
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Request failed')
  }
  return response.data.data
}

export async function updateChannelFailoverBindings(
  bindings: ChannelFailoverBindingUpdate[]
): Promise<void> {
  const response = await api.put<ApiResponse<null>>(
    '/api/channel/failover/bindings',
    { bindings }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Request failed')
  }
}

export async function getFailoverMonitoring(): Promise<FailoverMonitoringSnapshot> {
  const response = await api.get<ApiResponse<FailoverMonitoringSnapshot>>(
    '/api/channel/failover/monitoring'
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Request failed')
  }
  return response.data.data
}
