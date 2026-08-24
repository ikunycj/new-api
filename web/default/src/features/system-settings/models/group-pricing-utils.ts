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
import type { Channel } from '@/features/channels/types'
import type { BillingGroupRoute } from '@/features/failover/types'

export type GroupPricingSnapshot = {
  name: string
  ratio: number
  topupRatio: number | null
  selectable: boolean
  description: string
}

export function reorderBillingGroupChannels<
  T extends {
    id: number
    channel_id: number
    priority: number
  },
>(
  entries: T[],
  channelID: number,
  direction: -1 | 1,
  movableChannelIDs?: ReadonlySet<number>
): T[] {
  const ordered = [...entries].sort(
    (a, b) => b.priority - a.priority || a.id - b.id
  )
  const movableIndexes = ordered.flatMap((entry, index) =>
    !movableChannelIDs || movableChannelIDs.has(entry.channel_id) ? [index] : []
  )
  const movableIndex = movableIndexes.findIndex(
    (index) => ordered[index].channel_id === channelID
  )
  const targetMovableIndex = movableIndex + direction
  if (
    movableIndex < 0 ||
    targetMovableIndex < 0 ||
    targetMovableIndex >= movableIndexes.length
  ) {
    return entries
  }

  const index = movableIndexes[movableIndex]
  const targetIndex = movableIndexes[targetMovableIndex]
  const target = ordered[targetIndex]
  ordered[targetIndex] = ordered[index]
  ordered[index] = target
  return ordered.map((entry, position) => ({
    ...entry,
    priority: ordered.length - position,
  }))
}

function parseNumberMap(value: string): Record<string, number> {
  try {
    const parsed: unknown = JSON.parse(value)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      return {}
    }
    return Object.fromEntries(
      Object.entries(parsed).flatMap(([key, candidate]) => {
        const number = Number(candidate)
        return Number.isFinite(number) ? [[key, number]] : []
      })
    )
  } catch {
    return {}
  }
}

function parseStringMap(value: string): Record<string, string> {
  try {
    const parsed: unknown = JSON.parse(value)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      return {}
    }
    return Object.fromEntries(
      Object.entries(parsed).map(([key, candidate]) => [
        key,
        typeof candidate === 'string' ? candidate : '',
      ])
    )
  } catch {
    return {}
  }
}

export function buildGroupPricingSnapshots(
  groupRatio: string,
  topupGroupRatio: string,
  userUsableGroups: string
): GroupPricingSnapshot[] {
  const ratios = parseNumberMap(groupRatio)
  const topupRatios = parseNumberMap(topupGroupRatio)
  const usableGroups = parseStringMap(userUsableGroups)
  const names = new Set([
    ...Object.keys(ratios),
    ...Object.keys(topupRatios),
    ...Object.keys(usableGroups),
  ])

  return [...names]
    .sort((a, b) => a.localeCompare(b))
    .map((name) => ({
      name,
      ratio: ratios[name] ?? 1,
      topupRatio: Object.hasOwn(topupRatios, name) ? topupRatios[name] : null,
      selectable: Object.hasOwn(usableGroups, name),
      description: usableGroups[name] ?? '',
    }))
}

export function getToBGroupNames(
  routes: BillingGroupRoute[]
): ReadonlySet<string> {
  return new Set(
    routes.map((route) => route.billing_group.trim()).filter(Boolean)
  )
}

export function channelBelongsToGroup(
  channel: Pick<Channel, 'group'>,
  group: string
): boolean {
  const normalizedGroup = group.trim()
  return channel.group
    .split(',')
    .some((candidate) => candidate.trim() === normalizedGroup)
}
