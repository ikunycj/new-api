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
import { afterEach, describe, test } from 'node:test'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { formatApiKeyTokens, formatApiKeyUsageCost } from './usage-format'

const originalConfig = useSystemConfigStore.getState().config

afterEach(() => {
  useSystemConfigStore.setState({ config: originalConfig })
})

describe('API key usage cost formatting', () => {
  test('converts raw quota to the configured monetary display', () => {
    useSystemConfigStore.setState({
      config: {
        ...originalConfig,
        currency: {
          ...originalConfig.currency,
          quotaDisplayType: 'CNY',
          quotaPerUnit: 500_000,
          usdExchangeRate: 7.3,
        },
      },
    })

    assert.equal(formatApiKeyUsageCost(3_000_000, 'en-US'), '¥43.8')
  })

  test('keeps cost monetary when general quota display uses tokens', () => {
    useSystemConfigStore.setState({
      config: {
        ...originalConfig,
        currency: {
          ...originalConfig.currency,
          quotaDisplayType: 'TOKENS',
          quotaPerUnit: 500_000,
        },
      },
    })

    assert.equal(formatApiKeyUsageCost(3_000_000, 'en-US'), '$6')
  })
})

describe('API key token usage formatting', () => {
  test('uses compact K, M, and B units without trailing zeroes', () => {
    assert.equal(formatApiKeyTokens(0, 'en-US'), '0')
    assert.equal(formatApiKeyTokens(999, 'en-US'), '999')
    assert.equal(formatApiKeyTokens(1_500, 'en-US'), '1.5K')
    assert.equal(formatApiKeyTokens(34_670_000, 'en-US'), '34.67M')
    assert.equal(formatApiKeyTokens(2_110_000_000, 'en-US'), '2.11B')
  })

  test('normalizes missing, negative, and non-finite usage to zero', () => {
    assert.equal(formatApiKeyTokens(undefined, 'en-US'), '0')
    assert.equal(formatApiKeyTokens(-1, 'en-US'), '0')
    assert.equal(formatApiKeyTokens(Number.POSITIVE_INFINITY, 'en-US'), '0')
  })
})
