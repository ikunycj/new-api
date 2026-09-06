import { isAxiosError } from 'axios'

import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  BillingGroupType,
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

export async function updateBillingGroupType(
  billingGroup: string,
  groupType: BillingGroupType
): Promise<void> {
  let response
  try {
    response = await api.patch<ApiResponse<null>>(
      '/api/channel/failover/config/group-type',
      {
        billing_group: billingGroup,
        group_type: groupType,
      },
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  } catch (error) {
    if (isAxiosError<ApiResponse<null>>(error)) {
      throw new Error(error.response?.data.message || 'Request failed')
    }
    throw error
  }
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
