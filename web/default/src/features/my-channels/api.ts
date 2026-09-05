import { api } from '@/lib/api'

import type { ChannelTestResponse } from '../channels/types'
import type { SelfChannel, SelfChannelResponse } from './types'

export async function getSelfChannels(): Promise<SelfChannel[]> {
  const response = await api.get<SelfChannelResponse>('/api/user/self/channels')
  return Array.isArray(response.data.data) ? response.data.data : []
}

export async function testSelfChannel(
  id: number,
  params?: { model?: string; endpoint_type?: string; stream?: boolean }
): Promise<ChannelTestResponse> {
  const response = await api.get<ChannelTestResponse>(
    `/api/user/self/channels/${id}/test`,
    { params }
  )
  return response.data
}
