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
export interface FeaturedModel {
  modelName: string
  provider: string
  icon: string
  // USD per million tokens, in the same order as the pricing scope label.
  prices: { label: string; usd: readonly number[] }[]
  pricingScope: string
  discountRatio: number
  sourceUrl: string
}

export const FEATURED_USD_TO_CNY_RATE = 7

// Provider-published USD rates, separate from the gateway's model ratios.
export const FEATURED_MODELS: FeaturedModel[] = [
  {
    modelName: 'gpt-6-astra',
    provider: 'OpenAI',
    icon: "OpenAI.Avatar.type={'gpt5'}.shape={'square'}",
    prices: [
      { label: 'Input', usd: [10, 20] },
      { label: 'Output', usd: [50, 75] },
    ],
    pricingScope: 'Input tokens: ≤272K / >272K',
    discountRatio: 0.01,
    sourceUrl: 'https://developers.openai.com/api/docs/models/gpt-6-astra',
  },
  {
    modelName: 'claude-fable-5-1',
    provider: 'Anthropic',
    icon: 'Claude.Color',
    prices: [
      { label: 'Input', usd: [10] },
      { label: 'Output', usd: [50] },
    ],
    pricingScope: 'Standard pricing',
    discountRatio: 0.16,
    sourceUrl: 'https://platform.claude.com/docs/en/about-claude/pricing',
  },
  {
    modelName: 'deepseek-flash',
    provider: 'DeepSeek',
    icon: 'DeepSeek.Color',
    prices: [
      { label: 'Input', usd: [0.15, 0.3] },
      { label: 'Output', usd: [0.6, 1.2] },
    ],
    // deepseek-flash is the DeepSeek-V4.1-Flash pricing row.
    // Peak: Mon–Fri 01:00–04:00 and 06:00–10:00 UTC; otherwise off-peak.
    pricingScope: 'Off-peak / peak pricing (UTC)',
    discountRatio: 1,
    sourceUrl: 'https://api-docs.deepseek.com/quick_start/pricing',
  },
  {
    modelName: 'gpt-image-2.5-sunburst',
    provider: 'OpenAI',
    icon: 'Dalle.Color',
    prices: [
      { label: 'Text input', usd: [5] },
      { label: 'Image input', usd: [8] },
      { label: 'Image output', usd: [30] },
    ],
    pricingScope: 'Text and image tokens are priced separately',
    discountRatio: 0.01,
    sourceUrl:
      'https://developers.openai.com/api/docs/pricing#image-generation',
  },
  {
    modelName: 'grok-4.6',
    provider: 'xAI',
    icon: "Grok.Avatar.shape={'square'}",
    prices: [
      { label: 'Input', usd: [2, 4] },
      { label: 'Output', usd: [6, 12] },
    ],
    pricingScope: 'Input tokens: <200K / ≥200K',
    discountRatio: 0.02,
    sourceUrl: 'https://docs.x.ai/developers/pricing.md',
  },
  {
    modelName: 'claude-sonnet-5',
    provider: 'Anthropic',
    icon: 'Claude.Color',
    prices: [
      { label: 'Input', usd: [2] },
      { label: 'Output', usd: [10] },
    ],
    // The announced September price increase was cancelled; $2/$10 is standard.
    pricingScope: 'Standard pricing',
    discountRatio: 0.02,
    sourceUrl: 'https://platform.claude.com/docs/en/about-claude/pricing',
  },
]

const featuredPriceFormatters = {
  CNY: new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'CNY',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
    useGrouping: false,
  }),
  USD: new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
    useGrouping: false,
  }),
} as const

/** Official USD × the supplied display rate × the showcase multiplier. */
export function getFeaturedPriceDisplay(
  officialPricesUSD: readonly number[],
  discountRatio: number,
  billingUSDToCNYRate?: number
) {
  const hasExchangeRate =
    typeof billingUSDToCNYRate === 'number' &&
    Number.isFinite(billingUSDToCNYRate) &&
    billingUSDToCNYRate > 0
  const currency = hasExchangeRate ? 'CNY' : 'USD'
  const exchangeRate = hasExchangeRate ? billingUSDToCNYRate : 1
  const formatter = featuredPriceFormatters[currency]
  const officialPrices = officialPricesUSD.map((price) => price * exchangeRate)

  return {
    currency,
    officialPrice: officialPrices
      .map((price) => formatter.format(price))
      .join(' / '),
    currentPrice: officialPrices
      .map((price) => formatter.format(price * discountRatio))
      .join(' / '),
  }
}
