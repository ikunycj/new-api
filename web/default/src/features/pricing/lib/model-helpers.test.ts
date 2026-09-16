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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { PricingModel } from '../types'
import { filterByGroup } from './filters'
import {
  expandModelsByGroup,
  formatGroupRatio,
  getDisplayGroupRatio,
} from './model-helpers'

const models: PricingModel[] = [
  {
    id: 1,
    model_name: 'grouped-model',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default', 'vip', 'enterprise', 'pro', 'hidden'],
  },
  {
    id: 2,
    model_name: 'wildcard-model',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['all'],
  },
]

describe('model group display entries', () => {
  test('returns each model with its enabled catalog groups ordered by price', () => {
    const result = expandModelsByGroup(
      models,
      ['default', 'vip', 'enterprise', 'pro'],
      {
        default: 1,
        vip: 0.8,
        enterprise: 1.2,
        pro: 1.5,
      }
    )

    assert.deepEqual(
      result.map((model) => [
        model.model_name,
        model.display_group,
        model.display_group_ratio,
        model.key,
        model.display_groups,
      ]),
      [
        [
          'grouped-model',
          'vip',
          0.8,
          'grouped-model',
          [
            { group: 'vip', ratio: 0.8 },
            { group: 'default', ratio: 1 },
            { group: 'enterprise', ratio: 1.2 },
            { group: 'pro', ratio: 1.5 },
          ],
        ],
      ]
    )
  })

  test('uses the lowest available group when ratios are missing', () => {
    const result = expandModelsByGroup(models, ['default', 'vip'], {})

    assert.deepEqual(
      result.map((model) => [
        model.model_name,
        model.display_group,
        model.display_group_ratio,
      ]),
      [['grouped-model', 'default', 1]]
    )
  })

  test('keeps a literal all group scoped to models that enable it', () => {
    const result = expandModelsByGroup(models, ['default', 'auto', 'all'], {
      default: 1,
      auto: 0.5,
      all: 0.1,
    })

    assert.deepEqual(
      result.map((model) => model.display_groups),
      [[{ group: 'default', ratio: 1 }], [{ group: 'all', ratio: 0.1 }]]
    )
  })

  test('omits models with no enabled catalog groups', () => {
    const result = expandModelsByGroup(
      [
        {
          id: 3,
          model_name: 'catalog-only-model',
          quota_type: 0,
          model_ratio: 1,
          completion_ratio: 1,
          enable_groups: [],
        },
      ],
      ['default', 'vip'],
      { default: 1, vip: 0.8 }
    )

    assert.deepEqual(result, [])
  })

  test('ignores a selected group absent from model abilities', () => {
    const model = {
      ...models[0],
      enable_groups: ['default'],
      group_ratio: { default: 1, vip: 0.8 },
    }

    assert.equal(getDisplayGroupRatio(model, 'vip'), 1)
  })

  test('uses the lowest ratio among enabled model groups', () => {
    const model = {
      ...models[0],
      enable_groups: ['default'],
      group_ratio: { default: 1, vip: 0.8 },
    }

    assert.equal(getDisplayGroupRatio(model), 1)
  })

  test('filters models by their enabled groups', () => {
    assert.deepEqual(
      filterByGroup(models, 'vip').map((model) => model.model_name),
      ['grouped-model']
    )
  })

  test('formats finite group multipliers', () => {
    assert.equal(formatGroupRatio(1), 'x1')
    assert.equal(formatGroupRatio(0.125), 'x0.125')
    assert.equal(formatGroupRatio(Number.NaN), undefined)
  })
})
