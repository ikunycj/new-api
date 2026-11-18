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
import { test } from 'node:test'

import {
  strategyWeightPercentages,
  type StrategyWeightDraft,
} from './routing-strategy-weights'

const zeroWeights: StrategyWeightDraft = {
  priceWeight: '0',
  availabilityWeight: '0',
  loadWeight: '0',
  ttftWeight: '0',
  recentTestTTFTWeight: '0',
  iqTestConfidenceWeight: '0',
}

test('shows the same scoring percentages regardless of the input scale', () => {
  for (const [price, confidence] of [
    ['0.25', '0.75'],
    ['25', '75'],
    ['200', '600'],
    ['5e307', '1.5e308'],
  ]) {
    assert.deepEqual(
      strategyWeightPercentages({
        ...zeroWeights,
        priceWeight: price,
        iqTestConfidenceWeight: confidence,
      }),
      [25, 0, 0, 0, 0, 75]
    )
  }
})

test('accepts a single positive weight and rejects invalid drafts', () => {
  const iqOnly = { ...zeroWeights, iqTestConfidenceWeight: '0.5' }
  assert.deepEqual(strategyWeightPercentages(iqOnly), [0, 0, 0, 0, 0, 100])
  assert.equal(strategyWeightPercentages(zeroWeights), null)
  for (const priceWeight of ['', ' ', '-1', 'invalid', '1e309']) {
    assert.equal(strategyWeightPercentages({ ...iqOnly, priceWeight }), null)
  }
})
