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

import {
  formatUsageLogQuotaUSD,
  formatUsageLogUSDMicros,
  usageLogQuotaToUSD,
} from './money'

const originalConfig = useSystemConfigStore.getState().config

afterEach(() => {
  useSystemConfigStore.setState({ config: originalConfig })
})

describe('usage log USD formatting', () => {
  test('reverses the frozen billing exchange rate', () => {
    useSystemConfigStore.setState({
      config: {
        ...originalConfig,
        currency: {
          ...originalConfig.currency,
          quotaDisplayType: 'CNY',
          quotaPerUnit: 500_000,
        },
      },
    })

    assert.equal(usageLogQuotaToUSD(3_650_000, { billingUSDToCNYRate: 7.3 }), 1)
    assert.match(
      formatUsageLogQuotaUSD(3_650_000, { billingUSDToCNYRate: 7.3 }),
      /^\$1(?:[.,]0*)?$/
    )
  })

  test('uses the quota unit frozen on the log when available', () => {
    assert.equal(
      usageLogQuotaToUSD(14_600_000, {
        billingUSDToCNYRate: 7.3,
        quotaPerUnit: 1_000_000,
      }),
      2
    )
  })

  test('formats reconciliation micros as USD independently of site currency', () => {
    assert.match(formatUsageLogUSDMicros(1_250_000), /^\$1[.,]25$/)
  })
})
