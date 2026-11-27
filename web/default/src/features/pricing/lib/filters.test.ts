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

import { QUOTA_TYPE_VALUES, SORT_OPTIONS } from '../constants'
import type { PricingModel } from '../types'
import { filterAndSortModels, filterByGroup, sortModels } from './filters'
import { expandModelsByGroup } from './model-helpers'

const catalogModels: PricingModel[] = [
  {
    id: 1,
    model_name: 'zeta-model',
    vendor_name: 'Other',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
  },
  {
    id: 2,
    model_name: 'gpt-5.6-luna',
    vendor_name: 'OpenAI',
    quota_type: 0,
    model_ratio: 2,
    completion_ratio: 1,
    enable_groups: ['default'],
  },
  {
    id: 3,
    model_name: 'claude-fable-5',
    vendor_name: 'Anthropic',
    quota_type: 0,
    model_ratio: 6,
    completion_ratio: 1,
    enable_groups: ['default'],
  },
  {
    id: 4,
    model_name: 'gpt-5.6-terra',
    vendor_name: 'OpenAI',
    quota_type: 0,
    model_ratio: 3,
    completion_ratio: 1,
    enable_groups: ['default'],
  },
  {
    id: 5,
    model_name: 'alpha-model',
    vendor_name: 'Other',
    quota_type: 0,
    model_ratio: 0.5,
    completion_ratio: 1,
    enable_groups: ['default'],
  },
  {
    id: 6,
    model_name: 'gpt-5.6-sol',
    vendor_name: 'OpenAI',
    quota_type: 0,
    model_ratio: 4,
    completion_ratio: 1,
    enable_groups: ['default'],
  },
]

const models = expandModelsByGroup(catalogModels, ['default'], { default: 1 })

describe('pricing model sorting', () => {
  test('prioritizes models in the selected capability leaderboard order', () => {
    const preferredOrder = [
      'claude-fable-5',
      'gpt-5.6-sol',
      'gpt-5.6-terra',
      'gpt-5.6-luna',
    ]
    const result = sortModels(models, SORT_OPTIONS.CODING, preferredOrder)

    assert.deepEqual(
      result.map((model) => model.model_name),
      [
        'claude-fable-5',
        'gpt-5.6-sol',
        'gpt-5.6-terra',
        'gpt-5.6-luna',
        'alpha-model',
        'zeta-model',
      ]
    )
  })

  test('filters before applying promoted model priority', () => {
    const result = filterAndSortModels(
      models,
      {
        search: '',
        vendors: ['OpenAI'],
        groups: [],
        sortBy: SORT_OPTIONS.AGENT,
      },
      ['claude-fable-5', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']
    )

    assert.deepEqual(
      result.map((model) => model.model_name),
      ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']
    )
  })

  test('prioritizes the configured best-value models within the current results', () => {
    const result = filterAndSortModels(
      models,
      {
        search: '',
        vendors: ['OpenAI'],
        groups: [],
        sortBy: SORT_OPTIONS.BEST_VALUE,
      },
      ['claude-fable-5', 'gpt-5.6-terra', 'gpt-5.6-sol']
    )

    assert.deepEqual(
      result.map((model) => model.model_name),
      ['gpt-5.6-terra', 'gpt-5.6-sol', 'gpt-5.6-luna']
    )
  })

  test('searches model names, providers, and pricing groups', () => {
    const byModel = filterAndSortModels(models, {
      search: 'fable',
      vendors: [],
      groups: [],
      sortBy: SORT_OPTIONS.NAME,
    })
    const byVendor = filterAndSortModels(models, {
      search: 'anthro',
      vendors: [],
      groups: [],
      sortBy: SORT_OPTIONS.NAME,
    })
    const byGroup = filterAndSortModels(
      expandModelsByGroup(
        [{ ...catalogModels[0], enable_groups: ['ChatGPT Plus'] }],
        ['ChatGPT Plus'],
        { 'ChatGPT Plus': 1 }
      ),
      {
        search: 'plus',
        vendors: [],
        groups: [],
        sortBy: SORT_OPTIONS.NAME,
      }
    )

    assert.deepEqual(
      byModel.map((model) => model.model_name),
      ['claude-fable-5']
    )
    assert.deepEqual(
      byVendor.map((model) => model.model_name),
      ['claude-fable-5']
    )
    assert.equal(byGroup.length, 1)
  })

  test('does not hide models based on model type', () => {
    const typedModels = [
      models[0],
      { ...models[1], audio_ratio: 8 },
      {
        ...models[2],
        supported_endpoint_types: ['image-generation'],
      },
      {
        ...models[3],
        supported_endpoint_types: ['openai-video'],
      },
    ]

    const result = filterAndSortModels(typedModels, {
      search: '',
      vendors: [],
      groups: [],
      sortBy: SORT_OPTIONS.NAME,
    })

    assert.equal(result.length, typedModels.length)
  })

  test('keeps explicit name and price sorting semantics', () => {
    assert.deepEqual(
      sortModels(models, SORT_OPTIONS.NAME).map((model) => model.model_name),
      [
        'alpha-model',
        'claude-fable-5',
        'gpt-5.6-luna',
        'gpt-5.6-sol',
        'gpt-5.6-terra',
        'zeta-model',
      ]
    )
    assert.equal(
      sortModels(models, SORT_OPTIONS.PRICE_LOW)[0]?.model_name,
      'alpha-model'
    )
    assert.equal(
      sortModels(models, SORT_OPTIONS.PRICE_HIGH)[0]?.model_name,
      'claude-fable-5'
    )
  })

  test('uses the lowest visible pricing group even after filtering by another group', () => {
    const candidates = expandModelsByGroup(
      [
        { ...catalogModels[0], enable_groups: ['default', 'hidden'] },
        { ...catalogModels[1], enable_groups: ['default', 'vip'] },
        { ...catalogModels[2], enable_groups: ['vip'] },
      ],
      ['default', 'vip'],
      { default: 1, vip: 0.1, hidden: 0.01 }
    )

    assert.deepEqual(
      sortModels(candidates, SORT_OPTIONS.PRICE_LOW).map(
        (model) => model.model_name
      ),
      ['gpt-5.6-luna', 'claude-fable-5', 'zeta-model']
    )
    assert.deepEqual(
      filterAndSortModels(candidates, {
        search: '',
        vendors: [],
        groups: ['default'],
        sortBy: SORT_OPTIONS.PRICE_LOW,
      }).map((model) => model.model_name),
      ['gpt-5.6-luna', 'zeta-model']
    )
  })

  test('uses the first displayed expression input price with the group multiplier', () => {
    const candidates = expandModelsByGroup(
      [
        catalogModels[0],
        {
          ...catalogModels[2],
          model_name: 'tiered-model',
          enable_groups: ['vip'],
          billing_mode: 'tiered_expr',
          billing_expr:
            'len <= 100 ? tier("output_only", c * 20) : len <= 200 ? tier("standard", p * 8) : tier("long", p * 1)',
        },
        catalogModels[3],
      ],
      ['default', 'vip'],
      { default: 1, vip: 0.5 }
    )

    assert.deepEqual(
      sortModels(candidates, SORT_OPTIONS.PRICE_LOW).map(
        (model) => model.model_name
      ),
      ['zeta-model', 'tiered-model', 'gpt-5.6-terra']
    )
  })

  test('keeps models without comparable input prices after priced models in both directions', () => {
    const candidates = expandModelsByGroup(
      [
        {
          ...catalogModels[0],
          model_name: 'per-request-model',
          quota_type: QUOTA_TYPE_VALUES.REQUEST,
          model_price: 0.01,
        },
        {
          ...catalogModels[1],
          model_name: 'expression-per-request-model',
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("request", fixed(0.01))',
        },
        {
          ...catalogModels[2],
          model_name: 'unparsed-expression-model',
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("base", p * max(2, 3))',
        },
        {
          ...catalogModels[3],
          model_name: 'output-only-model',
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("output", c * 2)',
        },
        { ...catalogModels[4], model_name: 'free-input-model', model_ratio: 0 },
        catalogModels[5],
      ],
      ['default'],
      { default: 1 }
    )

    const ascending = sortModels(candidates, SORT_OPTIONS.PRICE_LOW).map(
      (model) => model.model_name
    )
    const descending = sortModels(candidates, SORT_OPTIONS.PRICE_HIGH).map(
      (model) => model.model_name
    )

    assert.deepEqual(ascending.slice(0, 2), ['free-input-model', 'gpt-5.6-sol'])
    assert.deepEqual(descending.slice(0, 2), [
      'gpt-5.6-sol',
      'free-input-model',
    ])
    const withoutComparablePrice = [
      'expression-per-request-model',
      'output-only-model',
      'per-request-model',
      'unparsed-expression-model',
    ]
    assert.deepEqual(ascending.slice(2).sort(), withoutComparablePrice)
    assert.deepEqual(descending.slice(2).sort(), withoutComparablePrice)
  })
})

describe('pricing model group filtering', () => {
  test('only keeps models that enable the selected catalog group', () => {
    const model: PricingModel = {
      ...catalogModels[0],
      enable_groups: ['default'],
      group_ratio: { default: 1, vip: 0.8 },
    }

    assert.deepEqual(filterByGroup([model], ['vip']), [])
    assert.deepEqual(
      filterByGroup(
        [{ ...catalogModels[0], enable_groups: ['vip'] }],
        ['vip']
      ).map((item) => item.model_name),
      ['zeta-model']
    )
  })

  test('matches any selected pricing group within the selected vendors', () => {
    const candidates = expandModelsByGroup(
      [
        {
          ...catalogModels[0],
          vendor_name: 'OpenAI',
          enable_groups: ['standard'],
        },
        {
          ...catalogModels[1],
          vendor_name: 'Anthropic',
          enable_groups: ['vip'],
        },
        {
          ...catalogModels[2],
          vendor_name: 'Google',
          enable_groups: ['standard'],
        },
      ],
      ['standard', 'vip'],
      { standard: 1, vip: 1 }
    )
    const result = filterAndSortModels(candidates, {
      search: '',
      vendors: ['OpenAI', 'Anthropic'],
      groups: ['vip', 'enterprise'],
      sortBy: SORT_OPTIONS.NAME,
    })

    assert.deepEqual(
      result.map((model) => model.model_name),
      ['gpt-5.6-luna']
    )
  })
})
