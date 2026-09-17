/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

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
  FEATURED_MODELS,
  FEATURED_USD_TO_CNY_RATE,
  getFeaturedPriceDisplay,
} from './featured-models'

describe('featured official price conversion', () => {
  // Provider Standard API USD/1M snapshots verified on 2026-09-17.
  const cases = [
    {
      model: 'gpt-6-astra',
      original: ['¥70.00 / ¥140.00', '¥350.00 / ¥525.00'],
      current: ['¥0.70 / ¥1.40', '¥3.50 / ¥5.25'],
    },
    {
      model: 'claude-fable-5-1',
      original: ['¥70.00', '¥350.00'],
      current: ['¥11.20', '¥56.00'],
    },
    {
      model: 'deepseek-flash',
      original: ['¥1.05 / ¥2.10', '¥4.20 / ¥8.40'],
      current: ['¥1.05 / ¥2.10', '¥4.20 / ¥8.40'],
    },
    {
      model: 'gpt-image-2.5-sunburst',
      original: ['¥35.00', '¥56.00', '¥210.00'],
      current: ['¥0.35', '¥0.56', '¥2.10'],
    },
    {
      model: 'grok-4.6',
      original: ['¥14.00 / ¥28.00', '¥42.00 / ¥84.00'],
      current: ['¥0.28 / ¥0.56', '¥0.84 / ¥1.68'],
    },
    {
      model: 'claude-sonnet-5',
      original: ['¥14.00', '¥70.00'],
      current: ['¥0.28', '¥1.40'],
    },
  ]

  for (const expected of cases) {
    test(`${expected.model}: converts official USD before applying its discount`, () => {
      const model = FEATURED_MODELS.find(
        (item) => item.modelName === expected.model
      )
      assert.ok(model)
      const prices = model.prices.map((price) =>
        getFeaturedPriceDisplay(
          price.usd,
          model.discountRatio,
          FEATURED_USD_TO_CNY_RATE
        )
      )
      assert.deepEqual(
        prices.map((price) => price.officialPrice),
        expected.original
      )
      assert.deepEqual(
        prices.map((price) => price.currentPrice),
        expected.current
      )
      assert.ok(prices.every((price) => price.currency === 'CNY'))
    })
  }

  test('uses the supplied exchange rate rather than a hardcoded rate', () => {
    assert.deepEqual(getFeaturedPriceDisplay([10, 20], 0.01, 7.2), {
      currency: 'CNY',
      officialPrice: '¥72.00 / ¥144.00',
      currentPrice: '¥0.72 / ¥1.44',
    })
  })

  test('missing, zero, negative or non-finite rates remain explicitly USD', () => {
    for (const rate of [
      undefined,
      0,
      -7,
      Number.NaN,
      Number.POSITIVE_INFINITY,
    ]) {
      assert.deepEqual(getFeaturedPriceDisplay([5], 0.01, rate), {
        currency: 'USD',
        officialPrice: '$5.00',
        currentPrice: '$0.05',
      })
    }
  })

  test('keeps precision after conversion for low token prices', () => {
    assert.equal(
      getFeaturedPriceDisplay([0.15], 0.01, 7.21).currentPrice,
      '¥0.0108'
    )
  })

  test('labels image input/output separately instead of inventing text output', () => {
    const imageModel = FEATURED_MODELS.find(
      (model) => model.modelName === 'gpt-image-2.5-sunburst'
    )
    assert.deepEqual(
      imageModel?.prices.map((price) => price.label),
      ['Text input', 'Image input', 'Image output']
    )
  })
})
