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
  getTieredBillingSummary,
  hasAnyCacheTokens,
} from '@/features/usage-logs/lib/format'

import type { PricingModel } from '../types'
import {
  BILLING_CONDITION_VARS,
  BILLING_EXTRA_VARS,
  formatBillingPriceUnit,
  formatBillingVarLabel,
  parseTiersFromExpr,
} from './billing-expr'
import {
  formatDynamicUnitPrice,
  getDynamicPricingSummary,
  getOfficialDynamicPricingSummary,
} from './dynamic-price'
import {
  formatOfficialPrice,
  formatOfficialRequestPrice,
  formatPrice,
  formatRequestPrice,
} from './price'
import {
  evalExprLocally,
  generateExprFromVisualConfig,
  tryParseVisualConfig,
  type ExtraTokenValues,
} from './tier-expr'

const formulaModel: PricingModel = {
  id: 4,
  model_name: 'billing-formula-model',
  quota_type: 0,
  model_ratio: 0.5,
  completion_ratio: 1,
  cache_ratio: 0.1,
  enable_groups: ['sale'],
  group_ratio: { sale: 0.05 },
}

const requestModel: PricingModel = {
  id: 2,
  model_name: 'discounted-request-model',
  quota_type: 1,
  model_ratio: 0,
  completion_ratio: 0,
  model_price: 0.4,
  enable_groups: ['sale'],
  group_ratio: { sale: 0.25 },
}

const dynamicModel: PricingModel = {
  id: 3,
  model_name: 'discounted-dynamic-model',
  quota_type: 0,
  model_ratio: 0,
  completion_ratio: 0,
  enable_groups: [],
  billing_mode: 'tiered_expr',
  billing_expr: 'tier("standard", p * 5 + c * 30 + cr * 1)',
}

describe('billing exchange rate model-square prices', () => {
  test('applies the billing exchange rate after the token group ratio', () => {
    assert.equal(
      formatPrice(formulaModel, 'input', 'M', false, 1, 7.3, 'sale'),
      '$0.05'
    )
    assert.equal(
      formatPrice(formulaModel, 'input', 'M', true, 1, 7.3, 'sale'),
      '¥0.365'
    )
    assert.equal(formatOfficialPrice(formulaModel, 'input', 'M', 7.3), '¥7.3')
    assert.equal(
      formatOfficialPrice(formulaModel, 'input', 'K', 7.3),
      '¥0.0073'
    )
    assert.equal(formatOfficialPrice(formulaModel, 'cache', 'M', 7.3), '¥0.73')
    assert.equal(
      formatOfficialPrice(formulaModel, 'input', 'M', 7.3, false),
      '$1'
    )
    assert.equal(
      formatPrice(formulaModel, 'cache', 'M', true, 1, 7.3, 'sale'),
      '¥0.0365'
    )
    assert.equal(
      formatPrice(formulaModel, 'input', 'M', true, 1, Number.NaN, 'sale'),
      '¥0.05'
    )
  })

  test('applies the same formula to per-request prices', () => {
    assert.equal(
      formatRequestPrice(requestModel, false, 1, 7.3, 'sale'),
      '$0.1'
    )
    assert.equal(formatOfficialRequestPrice(requestModel, 7.3), '¥2.92')
    assert.equal(formatOfficialRequestPrice(requestModel, 7.3, false), '$0.4')
    assert.equal(
      formatRequestPrice(requestModel, true, 1, 7.3, 'sale'),
      '¥0.73'
    )
  })

  test('applies the same formula to dynamic prices', () => {
    assert.equal(
      formatDynamicUnitPrice(5, {
        tokenUnit: 'M',
        showRechargePrice: true,
        usdExchangeRate: 7.3,
        groupRatioMultiplier: 0.2,
      }),
      '¥7.3'
    )
    assert.equal(
      formatDynamicUnitPrice(5, {
        tokenUnit: 'M',
        showRechargePrice: false,
        usdExchangeRate: 7.3,
        groupRatioMultiplier: 0.2,
      }),
      '$1'
    )
    const officialSummary = getOfficialDynamicPricingSummary(
      dynamicModel,
      'M',
      7.3
    )
    assert.deepEqual(
      officialSummary?.entries.map((entry) => entry.formatted),
      ['¥36.5', '¥219', '¥7.3']
    )
    const officialUsdSummary = getOfficialDynamicPricingSummary(
      dynamicModel,
      'M',
      7.3,
      false
    )
    assert.deepEqual(
      officialUsdSummary?.entries.map((entry) => entry.formatted),
      ['$5', '$30', '$1']
    )
  })
})

const emptyExtras: ExtraTokenValues = {
  cacheReadTokens: 0,
  cacheCreateTokens: 0,
  cacheCreate1hTokens: 0,
  cacheCreate12hTokens: 0,
  imageTokens: 0,
  imageOutputTokens: 0,
  audioInputTokens: 0,
  audioOutputTokens: 0,
}

const imageExpression = 'tier("image", fixed(0.0035)) * image_count'

// Labels and units receive the same translator used by React callers.
const translate = (key: string) => `translated:${key}`

describe('upstream billing expression protocol', () => {
  test('prices cc12h separately and includes it in full input length', () => {
    const expression =
      'len <= 100 ? tier("short", p * 2 + c * 4 + cc * 3 + cc1h * 6 + cc12h * 5) : tier("long", p * 4 + c * 8 + cc * 6 + cc1h * 12 + cc12h * 10)'
    const extras = { ...emptyExtras, cacheCreate12hTokens: 90 }
    assert.deepEqual(evalExprLocally(expression, 10, 2, extras), {
      cost: 478,
      matchedTier: 'short',
      error: null,
    })
    assert.deepEqual(evalExprLocally(expression, 11, 2, extras), {
      cost: 960,
      matchedTier: 'long',
      error: null,
    })
    const model = { ...dynamicModel, billing_expr: expression }
    const summary = getDynamicPricingSummary(model, {
      tokenUnit: 'K',
      groupRatioMultiplier: 0.2,
      showRechargePrice: true,
      usdExchangeRate: 7.3,
    })
    const cc12h = summary?.entries.find((entry) => entry.key === 'cc12h')
    assert.equal(cc12h?.formatted, '¥0.0073')
    assert.ok(cc12h)
    assert.equal(
      formatBillingVarLabel(cc12h.variable, translate),
      'translated:Cache Write (cc12h)'
    )
    assert.equal(
      formatBillingVarLabel(cc12h.variable, translate, false),
      'translated:Cache create price (cc12h)'
    )
  })

  for (const tokenUnit of ['K', 'M'] as const) {
    test(`keeps fixed prices independent of ${tokenUnit} token units`, () => {
      const model = { ...dynamicModel, billing_expr: imageExpression }
      const summary = getDynamicPricingSummary(model, {
        tokenUnit,
        groupRatioMultiplier: 0.2,
        showRechargePrice: true,
        usdExchangeRate: 7.3,
      })
      assert.equal(summary?.isSpecialExpression, false)
      assert.equal(summary?.entries.length, 1)
      assert.equal(summary?.primaryEntries[0]?.formatted, '¥0.00511')
      assert.equal(summary?.primaryEntries[0]?.billingUnit, 'request')
      assert.equal(summary?.primaryEntries[0]?.imageCount, true)
      assert.equal(
        formatBillingPriceUnit('request', true, translate, tokenUnit),
        'translated:Image'
      )
      assert.equal(
        formatBillingPriceUnit('request', false, translate, tokenUnit),
        'translated:request'
      )
      const official = getOfficialDynamicPricingSummary(model, tokenUnit, 7.3)
      assert.equal(official?.entries[0]?.formatted, '¥0.02555')
      const usd = getDynamicPricingSummary(model, {
        tokenUnit,
        groupRatioMultiplier: 0.2,
        usdExchangeRate: 7.3,
      })
      assert.equal(usd?.entries[0]?.formatted, '$0.0007')
    })
  }

  test('keeps plain fixed requests separate from image quantities', () => {
    const model = {
      ...dynamicModel,
      billing_expr: 'tier("request", fixed(0.0035))',
    }
    const summary = getDynamicPricingSummary(model, { tokenUnit: 'K' })
    assert.equal(summary?.primaryEntries[0]?.formatted, '$0.0035')
    assert.equal(summary?.primaryEntries[0]?.imageCount, false)
    const result = evalExprLocally(model.billing_expr, 0, 0, emptyExtras, 0)
    assert.deepEqual(result, {
      cost: 3500,
      matchedTier: 'request',
      error: null,
    })
  })

  test('retains explicit zero fixed prices for free images', () => {
    const model = {
      ...dynamicModel,
      billing_expr: 'tier("free", fixed(0)) * image_count',
    }
    const summary = getDynamicPricingSummary(model, { tokenUnit: 'M' })
    assert.equal(summary?.primaryEntries[0]?.formatted, '$0')
    assert.equal(summary?.primaryEntries[0]?.value, 0)
    const result = evalExprLocally(model.billing_expr, 0, 0, emptyExtras, 128)
    assert.deepEqual(result, { cost: 0, matchedTier: 'free', error: null })
  })

  test('parses conditional fixed leaves with request multipliers', () => {
    const expression =
      '(len <= 100 ? tier("small", fixed(0.0035)) : tier("large", fixed(0.007))) * image_count * (param("fast") == true ? 2 : 1)'
    const summary = getDynamicPricingSummary(
      { ...dynamicModel, billing_expr: expression },
      { tokenUnit: 'M' }
    )
    assert.equal(summary?.hasRequestRules, true)
    assert.equal(summary?.tierCount, 2)
    assert.equal(summary?.tiers[0]?.label, 'small')
    assert.equal(summary?.tiers[0]?.fixedPrice, 0.0035)
    assert.equal(summary?.tiers[1]?.label, 'large')
    assert.equal(summary?.tiers[1]?.fixedPrice, 0.007)
    assert.ok(summary?.tiers.every((tier) => tier.imageCount))
    assert.deepEqual(summary?.tiers[0].conditions, [
      { var: 'len', op: '<=', value: 100 },
    ])
    assert.deepEqual(
      evalExprLocally(
        'len <= 100 ? tier("small", fixed(0.0035)) : tier("large", fixed(0.007))',
        101,
        0,
        emptyExtras
      ),
      { cost: 7000, matchedTier: 'large', error: null }
    )
  })

  test('keeps unsupported arithmetic and fixed/token mixtures raw', () => {
    for (const expression of [
      'tier("bad", fixed(0.01) + p * 2)',
      'tier("bad", fixed(param("price")))',
      'tier("bad", fixed(0.01 + 0.02))',
      'tier("bad", fixed(-0.01))',
      'tier("bad", fixed(1e309))',
      'tier("bad", fixed(0.01)) * p',
      'tier("bad", fixed(0.01)) * 2',
      'tier("bad", fixed(0.01)) * (param("fast") == true ? -2 : 1)',
      'tier("bad", fixed(0.01)) * (param("fast") == true ? 1e309 : 1)',
      'tier("a", fixed(0.01)) + tier("b", fixed(0.02))',
      'tier("bad", p * 2 + unknown * 3)',
    ]) {
      assert.deepEqual(parseTiersFromExpr(expression), [], expression)
      assert.equal(tryParseVisualConfig(expression), null, expression)
    }
  })

  test('visual round trips preserve cache/media prices', () => {
    const expression =
      'tier("base", p * 2 + c * 4 + cr * 0.2 + cc * 3 + cc1h * 6 + cc12h * 5 + img * 2 + img_o * 8 + ai * 4 + ao * 12)'
    const config = tryParseVisualConfig(expression)
    assert.ok(config)
    assert.equal(config.tiers[0].cache_mode, 'timed')
    assert.equal(config.tiers[0].cache_create_12h_unit_cost, 5)
    assert.equal(generateExprFromVisualConfig(config), expression)
    assert.equal(tryParseVisualConfig(imageExpression), null)
    assert.equal(
      tryParseVisualConfig('tier("base", p * 2 + c * 4 + cc12h * 0)'),
      null,
      'an explicit zero cache coefficient must not be silently discarded'
    )
    assert.deepEqual(BILLING_CONDITION_VARS, ['p', 'c', 'len'])
    assert.ok(BILLING_EXTRA_VARS.every((variable) => variable.key !== 'fixed'))
  })

  test('previews fixed prices with bounded actual image quantities', () => {
    assert.deepEqual(evalExprLocally(imageExpression, 0, 0, emptyExtras), {
      cost: 3500,
      matchedTier: 'image',
      error: null,
    })
    assert.deepEqual(evalExprLocally(imageExpression, 0, 0, emptyExtras, 128), {
      cost: 448000,
      matchedTier: 'image',
      error: null,
    })
    for (const count of [0, -1, 129, 1.5, Number.NaN]) {
      const result = evalExprLocally(imageExpression, 0, 0, emptyExtras, count)
      assert.equal(result.cost, 0)
      assert.equal(result.error, 'image_count: 1..128')
    }
    assert.deepEqual(
      evalExprLocally('tier("base", p * 2 + c * 4)', 10, 2, emptyExtras, 0),
      { cost: 28, matchedTier: 'base', error: null }
    )
  })

  test('log summaries preserve frozen fixed prices and image units', () => {
    const summary = getTieredBillingSummary({
      billing_mode: 'tiered_expr',
      expr_b64: Buffer.from(imageExpression).toString('base64'),
      matched_tier: 'image',
      billing_unit: 'request',
      fixed_price: 0,
      image_count: 3,
    })
    assert.equal(summary?.priceEntries.length, 1)
    assert.equal(summary?.priceEntries[0].price, 0)
    assert.equal(summary?.priceEntries[0].billingUnit, 'request')
    assert.equal(summary?.priceEntries[0].imageCount, true)
  })

  test('cc12h-only usage keeps log cache pricing visible', () => {
    const expression = 'tier("base", p * 2 + c * 4 + cc12h * 5)'
    const other = {
      billing_mode: 'tiered_expr',
      expr_b64: Buffer.from(expression).toString('base64'),
      matched_tier: 'base',
      cache_creation_tokens_12h: 90,
    }
    assert.equal(hasAnyCacheTokens(other), true)
    const summary = getTieredBillingSummary(other)
    assert.equal(
      summary?.priceEntries.find((entry) => entry.key === 'cc12h')?.price,
      5
    )
    const withoutCache = getTieredBillingSummary({
      ...other,
      cache_creation_tokens_12h: 0,
    })
    assert.equal(
      withoutCache?.priceEntries.some((entry) => entry.key === 'cc12h'),
      false
    )
  })
})
