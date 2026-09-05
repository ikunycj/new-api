import { api } from '@/lib/api'

import type { SelfChannel, SelfChannelRequest, SelfChannelResponse } from './types'

export async function getSelfChannels(): Promise<SelfChannel[]> {
  const response = await api.get<SelfChannelResponse>('/api/user/self/channels')
  return Array.isArray(response.data.data) ? response.data.data : []
}

export async function createSelfChannel(data: SelfChannelRequest) {
  const response = await api.post<SelfChannelResponse>('/api/user/self/channels', data)
  return response.data
}

export async function updateSelfChannel(id: number, data: SelfChannelRequest) {
  const response = await api.put<SelfChannelResponse>(`/api/user/self/channels/${id}`, data)
  return response.data
}

export async function deleteSelfChannel(id: number) {
  const response = await api.delete<SelfChannelResponse>(`/api/user/self/channels/${id}`)
  return response.data
}

export function getSelfChannelFormData(channel?: SelfChannel): SelfChannelRequest {
  return {
    type: channel?.type ?? 1,
    name: channel?.name ?? '',
    key: channel?.key ?? '',
    base_url: channel?.base_url ?? '',
    models: channel?.models ?? '',
    model_mapping: channel?.model_mapping ?? '',
    openai_organization: channel?.openai_organization ?? '',
    remark: channel?.remark ?? '',
  }
}
