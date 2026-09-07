import { isAxiosError } from 'axios'

import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  BillingGroupChannel,
  BillingGroupRoute,
  BillingGroupType,
  FailoverConfig,
  FailoverMonitoringSnapshot,
} from './types'

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

async function scopedFailoverRequest<T>(
  request: () => Promise<{ data: ApiResponse<T> }>
): Promise<T> {
  try {
    const response = await request()
    if (!response.data.success || response.data.data === undefined) {
      throw new Error(response.data.message || 'Request failed')
    }
    return response.data.data
  } catch (error) {
    if (isAxiosError<ApiResponse<T>>(error)) {
      throw new Error(error.response?.data.message || 'Request failed')
    }
    throw error
  }
}

export function createBillingGroupRoute(
  route: BillingGroupRoute
): Promise<BillingGroupRoute> {
  return scopedFailoverRequest(() =>
    api.post<ApiResponse<BillingGroupRoute>>(
      '/api/channel/failover/routes',
      route,
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  )
}

export function updateBillingGroupRoute(
  route: BillingGroupRoute
): Promise<BillingGroupRoute> {
  return scopedFailoverRequest(() =>
    api.patch<ApiResponse<BillingGroupRoute>>(
      `/api/channel/failover/routes/${route.id}`,
      route,
      {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig
    )
  )
}

export function deleteBillingGroupRoute(routeID: number): Promise<void> {
  return scopedFailoverRequest(() =>
    api
      .delete<ApiResponse<null>>(`/api/channel/failover/routes/${routeID}`, {
        skipBusinessError: true,
        skipErrorHandler: true,
      } satisfies ApiRequestConfig)
      .then((response) => ({
        data: response.data,
      }))
  ).then(() => undefined)
}

export function saveBillingGroupRouteChannel(
  entry: BillingGroupChannel
): Promise<BillingGroupChannel> {
  const routePath = `/api/channel/failover/routes/${entry.billing_group_route_id}/channels`
  const request =
    entry.id > 0
      ? () =>
          api.patch<ApiResponse<BillingGroupChannel>>(
            `${routePath}/${entry.channel_id}`,
            entry,
            {
              skipBusinessError: true,
              skipErrorHandler: true,
            } satisfies ApiRequestConfig
          )
      : () =>
          api.post<ApiResponse<BillingGroupChannel>>(routePath, entry, {
            skipBusinessError: true,
            skipErrorHandler: true,
          } satisfies ApiRequestConfig)
  return scopedFailoverRequest(request)
}

export function deleteBillingGroupRouteChannel(
  routeID: number,
  channelID: number
): Promise<void> {
  return scopedFailoverRequest(() =>
    api
      .delete<ApiResponse<null>>(
        `/api/channel/failover/routes/${routeID}/channels/${channelID}`,
        {
          skipBusinessError: true,
          skipErrorHandler: true,
        } satisfies ApiRequestConfig
      )
      .then((response) => ({ data: response.data }))
  ).then(() => undefined)
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
