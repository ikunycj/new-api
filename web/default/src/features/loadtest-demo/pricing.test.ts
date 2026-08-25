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

import type { LoadTestPricing } from './api'
import {
  calculateLoadTestUserCharge,
  getLoadTestTotalTokens,
  type LoadTestUsageTotals,
} from './pricing'

const usage: LoadTestUsageTotals = {
  successes: 2,
  inputTokens: 1_000_000,
  outputTokens: 500_000,
  cacheReadTokens: 250_000,
  cacheWriteTokens: 125_000,
}

function pricing(
  overrides: Partial<LoadTestPricing['model']>
): LoadTestPricing {
  return {
    group: 'balanced',
    groupRatio: 0.4,
    model: {
      id: 1,
      model_name: 'test-model',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 2,
      cache_ratio: 0.1,
      create_cache_ratio: 1.25,
      enable_groups: ['balanced'],
      ...overrides,
    },
  }
}

describe('load-test pricing', () => {
  test('calculates token charge with group and cache ratios', () => {
    assert.equal(getLoadTestTotalTokens(usage), 1_875_000)
    assert.ok(
      Math.abs(calculateLoadTestUserCharge(usage, pricing({})) - 1.745) <
        Number.EPSILON
    )
  })

  test('charges only successful requests for request-priced models', () => {
    assert.equal(
      calculateLoadTestUserCharge(
        usage,
        pricing({ quota_type: 1, model_price: 0.25 })
      ),
      0.2
    )
  })
})
