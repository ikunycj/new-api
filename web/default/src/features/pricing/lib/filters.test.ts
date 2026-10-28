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

import {
  ENDPOINT_TYPES,
  FILTER_ALL,
  QUOTA_TYPES,
  SORT_OPTIONS,
} from '../constants'
import type { PricingModel } from '../types'
import { filterAndSortModels, filterByGroup, sortModels } from './filters'

const models: PricingModel[] = [
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

describe('pricing model sorting', () => {
  test('promotes configured models in merchandising order', () => {
    const preferredOrder = [
      'claude-fable-5',
      'gpt-5.6-sol',
      'gpt-5.6-terra',
      'gpt-5.6-luna',
    ]
    const result = sortModels(models, SORT_OPTIONS.RECOMMENDED, preferredOrder)

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
        quotaType: QUOTA_TYPES.ALL,
        endpointType: ENDPOINT_TYPES.ALL,
        tag: FILTER_ALL,
        sortBy: SORT_OPTIONS.RECOMMENDED,
      },
      ['claude-fable-5', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']
    )

    assert.deepEqual(
      result.map((model) => model.model_name),
      ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']
    )
  })

  test('searches model names, providers, and pricing groups', () => {
    const byModel = filterAndSortModels(models, {
      search: 'fable',
      vendors: [],
      groups: [],
      quotaType: QUOTA_TYPES.ALL,
      endpointType: ENDPOINT_TYPES.ALL,
      tag: FILTER_ALL,
      sortBy: SORT_OPTIONS.NAME,
    })
    const byVendor = filterAndSortModels(models, {
      search: 'anthro',
      vendors: [],
      groups: [],
      quotaType: QUOTA_TYPES.ALL,
      endpointType: ENDPOINT_TYPES.ALL,
      tag: FILTER_ALL,
      sortBy: SORT_OPTIONS.NAME,
    })
    const byGroup = filterAndSortModels(
      [{ ...models[0], enable_groups: ['ChatGPT Plus'] }],
      {
        search: 'plus',
        vendors: [],
        groups: [],
        quotaType: QUOTA_TYPES.ALL,
        endpointType: ENDPOINT_TYPES.ALL,
        tag: FILTER_ALL,
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
    const typedModels: PricingModel[] = [
      models[0],
      { ...models[1], audio_ratio: 8 },
      {
        ...models[2],
        supported_endpoint_types: [ENDPOINT_TYPES.IMAGE_GENERATION],
      },
      {
        ...models[3],
        supported_endpoint_types: [ENDPOINT_TYPES.OPENAI_VIDEO],
      },
    ]

    const result = filterAndSortModels(typedModels, {
      search: '',
      vendors: [],
      groups: [],
      quotaType: QUOTA_TYPES.ALL,
      endpointType: ENDPOINT_TYPES.ALL,
      tag: FILTER_ALL,
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
  })
})

describe('pricing model group filtering', () => {
  test('only keeps models that enable the selected catalog group', () => {
    const model: PricingModel = {
      ...models[0],
      enable_groups: ['default'],
      group_ratio: { default: 1, vip: 0.8 },
    }

    assert.deepEqual(filterByGroup([model], ['vip']), [])
    assert.deepEqual(
      filterByGroup([{ ...models[0], enable_groups: ['vip'] }], ['vip']).map(
        (item) => item.model_name
      ),
      ['zeta-model']
    )
  })

  test('matches any selected pricing group within the selected vendors', () => {
    const candidates: PricingModel[] = [
      { ...models[0], vendor_name: 'OpenAI', enable_groups: ['standard'] },
      { ...models[1], vendor_name: 'Anthropic', enable_groups: ['vip'] },
      { ...models[2], vendor_name: 'Google', enable_groups: ['standard'] },
    ]
    const result = filterAndSortModels(candidates, {
      search: '',
      vendors: ['OpenAI', 'Anthropic'],
      groups: ['vip', 'enterprise'],
      quotaType: QUOTA_TYPES.ALL,
      endpointType: ENDPOINT_TYPES.ALL,
      tag: FILTER_ALL,
      sortBy: SORT_OPTIONS.NAME,
    })

    assert.deepEqual(
      result.map((model) => model.model_name),
      ['gpt-5.6-luna']
    )
  })
})
