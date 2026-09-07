import { isAxiosError } from 'axios'

import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  BillingGroupChannel,
  BillingGroupRoute,
  BillingGroupRouteConfig,
  BillingGroupType,
  FailoverConfig,
  FailoverMonitoringSnapshot,
  StaleRouteCleanupResult,
  UpstreamErrorMapping,
} from './types'

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

function normalizeRequestError(error: unknown): Error {
  if (isAxiosError<ApiResponse<unknown>>(error)) {
    const responseData = error.response?.data
    let responseMessage: string | undefined
    if (typeof responseData === 'string') {
      responseMessage = responseData
    } else if (responseData && typeof responseData === 'object') {
      responseMessage = (responseData as ApiResponse<unknown>).message
    }
    return new Error(responseMessage || error.message || 'Request failed')
  }
  return error instanceof Error ? error : new Error('Request failed')
}

export async function getFailoverConfig(): Promise<FailoverConfig> {
  const response = await api.get<ApiResponse<FailoverConfig>>(
    '/api/channel/failover/config',
    { disableDuplicate: true } satisfies ApiRequestConfig
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Request failed')
  }
  return response.data.data
}

export async function updateFailoverConfig(
  config: FailoverConfig
): Promise<void> {
  let response
  try {
    response = await api.put<ApiResponse<null>>(
      '/api/channel/failover/config',
      config,
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  } catch (error) {
    throw normalizeRequestError(error)
  }
  if (!response.data.success) {
    throw new Error(response.data.message || 'Request failed')
  }
}

export async function updateFailoverErrorMappings(
  errorMappings: UpstreamErrorMapping[]
): Promise<void> {
  let response
  try {
    response = await api.put<ApiResponse<null>>(
      '/api/channel/failover/config/error-mappings',
      { error_mappings: errorMappings },
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  } catch (error) {
    throw normalizeRequestError(error)
  }
  if (!response.data.success) {
    throw new Error(response.data.message || 'Request failed')
  }
}

export async function updateBillingGroupRoute(
  route: BillingGroupRoute,
  routeChannels: BillingGroupChannel[]
): Promise<BillingGroupRouteConfig> {
  let response
  try {
    response = await api.put<ApiResponse<BillingGroupRouteConfig>>(
      '/api/channel/failover/config/route',
      { route, route_channels: routeChannels },
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  } catch (error) {
    throw normalizeRequestError(error)
  }
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Request failed')
  }
  return response.data.data
}

export async function deleteBillingGroupRoute(routeID: number): Promise<void> {
  let response
  try {
    response = await api.delete<ApiResponse<null>>(
      `/api/channel/failover/config/route/${routeID}`,
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  } catch (error) {
    throw normalizeRequestError(error)
  }
  if (!response.data.success) {
    throw new Error(response.data.message || 'Request failed')
  }
}

export async function cleanupStaleBillingGroupRoutes(
  routeID?: number
): Promise<StaleRouteCleanupResult> {
  let response
  try {
    response = await api.post<ApiResponse<StaleRouteCleanupResult>>(
      '/api/channel/failover/config/cleanup-stale',
      routeID && routeID > 0 ? { route_id: routeID } : {},
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  } catch (error) {
    throw normalizeRequestError(error)
  }
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Request failed')
  }
  return response.data.data
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
    throw normalizeRequestError(error)
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
