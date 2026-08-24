import type { Channel } from '@/features/channels/types'
import type { BillingGroupRoute } from '@/features/failover/types'

export type GroupPricingSnapshot = {
  name: string
  ratio: number
  topupRatio: number | null
  selectable: boolean
  description: string
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
