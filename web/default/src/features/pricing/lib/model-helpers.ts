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
import { EXCLUDED_GROUPS, FILTER_ALL, QUOTA_TYPE_VALUES } from '../constants'
import type { PricingDisplayModel, PricingModel } from '../types'

// ----------------------------------------------------------------------------
// Model Helper Utilities
// ----------------------------------------------------------------------------

/**
 * Get available groups for a model
 */
export function getAvailableGroups(
  model: PricingModel,
  usableGroup: Record<string, { desc: string; ratio: number }>
): string[] {
  const modelEnableGroups = Array.isArray(model.enable_groups)
    ? model.enable_groups
    : []
  const supportsAllGroups = modelEnableGroups.includes(FILTER_ALL)

  return Object.keys(usableGroup)
    .filter((group) => group !== FILTER_ALL && !EXCLUDED_GROUPS.includes(group))
    .filter((group) => supportsAllGroups || modelEnableGroups.includes(group))
}

/**
 * Read a configured group ratio while preserving valid zero ratios.
 */
export function getConfiguredGroupRatio(
  groupRatio: Record<string, number>,
  group: string
): number {
  const ratio = groupRatio[group]
  return typeof ratio === 'number' && Number.isFinite(ratio) ? ratio : 1
}

export function formatGroupRatio(
  ratio: number | undefined
): string | undefined {
  if (ratio == null || !Number.isFinite(ratio)) return undefined
  return `x${Object.is(ratio, -0) ? 0 : ratio}`
}

export function expandModelsByGroup(
  models: PricingModel[],
  availableGroups: string[],
  groupRatio: Record<string, number>
): PricingDisplayModel[] {
  const selectableGroups = availableGroups.filter(
    (group) => group !== FILTER_ALL && !EXCLUDED_GROUPS.includes(group)
  )
  return models.flatMap((model) => {
    const enabledGroups = Array.isArray(model.enable_groups)
      ? model.enable_groups
      : []
    const supportsAllGroups = enabledGroups.includes(FILTER_ALL)

    const groups = selectableGroups
      .filter((group) => supportsAllGroups || enabledGroups.includes(group))
      .map((group) => ({
        group,
        ratio: getConfiguredGroupRatio(groupRatio, group),
      }))
      .sort((a, b) => a.ratio - b.ratio || a.group.localeCompare(b.group))
      .slice(0, 3)

    if (groups.length === 0) return []
    const lowest = groups[0]
    return [
      {
        ...model,
        key: model.key || model.model_name,
        display_group: lowest.group,
        display_group_ratio: lowest.ratio,
        display_groups: groups,
      },
    ]
  })
}

/**
 * Resolve the group ratio used by model square summary prices.
 *
 * When no specific group is selected, the model square shows the best price
 * available to the viewer. When a group filter is active, it shows that
 * group's price.
 */
export function getDisplayGroupRatio(
  model: PricingModel,
  selectedGroup?: string
): number {
  const modelEnableGroups = Array.isArray(model.enable_groups)
    ? model.enable_groups
    : []
  const supportsAllGroups = modelEnableGroups.includes(FILTER_ALL)
  const groupRatio = model.group_ratio || {}

  if (
    selectedGroup &&
    selectedGroup !== FILTER_ALL &&
    (supportsAllGroups || modelEnableGroups.includes(selectedGroup))
  ) {
    return getConfiguredGroupRatio(groupRatio, selectedGroup)
  }

  if (modelEnableGroups.length === 0) {
    return 1
  }

  let minRatio = Number.POSITIVE_INFINITY

  for (const group of modelEnableGroups) {
    const ratio = groupRatio[group]
    if (
      typeof ratio === 'number' &&
      Number.isFinite(ratio) &&
      ratio < minRatio
    ) {
      minRatio = ratio
    }
  }

  return minRatio === Number.POSITIVE_INFINITY ? 1 : minRatio
}

/**
 * Replace model placeholder in endpoint path
 */
export function replaceModelInPath(path: string, modelName: string): string {
  return path.replaceAll('{model}', modelName)
}

/**
 * Check if model is token-based pricing
 */
export function isTokenBasedModel(model: PricingModel): boolean {
  return model.quota_type === QUOTA_TYPE_VALUES.TOKEN
}
