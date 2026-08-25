/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { SSE } from 'sse.js'

import { getCommonHeaders } from '@/lib/api'

import { channelsQueryKeys } from '../lib'

const CHANNEL_PROBE_REFRESH_DEBOUNCE_MS = 250

export function useChannelProbeRefresh() {
  const queryClient = useQueryClient()

  useEffect(() => {
    let refreshTimer: ReturnType<typeof setTimeout> | undefined
    const source = new SSE('/api/channel/probe/events', {
      autoReconnect: true,
      headers: {
        ...getCommonHeaders(),
        Accept: 'text/event-stream',
      },
      maxRetries: null,
      reconnectDelay: 3_000,
      start: false,
      useLastEventId: false,
      withCredentials: true,
    })

    source.addEventListener('channel-probe-completed', () => {
      if (refreshTimer) clearTimeout(refreshTimer)
      refreshTimer = setTimeout(() => {
        void queryClient.invalidateQueries({
          queryKey: channelsQueryKeys.lists(),
          refetchType: 'active',
        })
      }, CHANNEL_PROBE_REFRESH_DEBOUNCE_MS)
    })
    source.stream()

    return () => {
      if (refreshTimer) clearTimeout(refreshTimer)
      source.close()
    }
  }, [queryClient])
}
