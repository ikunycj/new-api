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
import { afterEach, test } from 'node:test'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { channelSchema, type Channel } from '../types'
import { formatChannelCostCNY } from './channel-cost'
import { aggregateChannelsByTag } from './channel-utils'

const originalConfig = useSystemConfigStore.getState().config

afterEach(() => {
  useSystemConfigStore.setState({ config: originalConfig })
})

function channel(overrides: Partial<Channel>): Channel {
  return channelSchema.parse({
    id: 1,
    type: 1,
    status: 1,
    name: 'upstream',
    created_time: 1,
    test_time: 0,
    response_time: 0,
    balance_updated_time: 0,
    tag: 'provider',
    ...overrides,
  })
}

test('procurement cost stays in CNY when wallet display settings change', () => {
  for (const quotaDisplayType of ['CNY', 'USD', 'TOKENS', 'CUSTOM'] as const) {
    useSystemConfigStore.setState({
      config: {
        ...originalConfig,
        currency: {
          ...originalConfig.currency,
          quotaDisplayType,
          usdExchangeRate: 7,
          customCurrencyExchangeRate: 0.9,
        },
      },
    })
    assert.equal(formatChannelCostCNY(5.329720742857143, 'en-US'), '¥5.33')
  }
})

test('missing or invalid cost is distinct from a confirmed zero cost', () => {
  assert.equal(formatChannelCostCNY(0, 'en-US'), '¥0')
  for (const amount of [null, Number.NaN, Infinity, -1]) {
    assert.equal(formatChannelCostCNY(amount, 'en-US'), '无法估算')
  }
})

test('tags sum procurement cost across both credit modes without converting again', () => {
  const rows = aggregateChannelsByTag([
    channel({
      daily_cost_cny: 5,
      total_cost_cny: 10,
      daily_tokens: 100,
      total_tokens: 200,
      price_multiplier_mode: 'usd',
    }),
    channel({
      id: 2,
      daily_cost_cny: 35,
      total_cost_cny: 70,
      daily_tokens: 300,
      total_tokens: 600,
      price_multiplier_mode: 'cny',
    }),
    channel({ id: 3, tag: 'another provider', daily_cost_cny: 99 }),
  ])
  assert.equal(rows.length, 2)
  assert.equal(rows[0].daily_cost_cny, 40)
  assert.equal(rows[0].total_cost_cny, 80)
  assert.equal(rows[0].daily_tokens, 400)
  assert.equal(rows[0].total_tokens, 800)
})

test('incomplete history makes only the affected period of a tag unknown', () => {
  const complete = channel({ daily_cost_cny: 5, total_cost_cny: 10 })
  const incomplete = channel({ id: 2, daily_cost_cny: 0 })
  for (const children of [
    [complete, incomplete],
    [incomplete, complete],
  ]) {
    const rows = aggregateChannelsByTag(children)
    assert.equal(rows[0].daily_cost_cny, 5)
    assert.equal(rows[0].total_cost_cny, null)
  }
  const rows = aggregateChannelsByTag([complete, channel({ id: 3 })])
  assert.equal(rows[0].daily_cost_cny, null)
  assert.equal(rows[0].total_cost_cny, null)
})
