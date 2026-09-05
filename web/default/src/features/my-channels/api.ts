import { api } from '@/lib/api'
import type { SelfChannel, SelfChannelResponse } from './types'

export async function getSelfChannels(): Promise<SelfChannel[]> {
  const response = await api.get<SelfChannelResponse>('/api/user/self/channels')
  return Array.isArray(response.data.data) ? response.data.data : []
}
