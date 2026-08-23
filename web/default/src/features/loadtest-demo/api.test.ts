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

import { sendLoadTestRequest, type LoadTestKey } from './api'

const originalFetch = globalThis.fetch
const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')

afterEach(() => {
  globalThis.fetch = originalFetch
  if (originalWindow) {
    Object.defineProperty(globalThis, 'window', originalWindow)
  } else {
    Reflect.deleteProperty(globalThis, 'window')
  }
})

describe('load test request identity', () => {
  test('uses the gateway request ID when the upstream also returns an ID', async () => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: { clearTimeout, setTimeout },
    })
    globalThis.fetch = (async () =>
      new Response(JSON.stringify({ usage: {} }), {
        status: 200,
        headers: {
          'X-Oneapi-Request-Id': 'gateway-request-id',
          'X-Request-Id': 'upstream-request-id',
        },
      })) as typeof fetch

    const result = await sendLoadTestRequest(
      'https://api.example.com',
      { name: 'test-key', secret: 'test-secret' } as LoadTestKey,
      'gpt-5.6-sol',
      'load-test-run',
      false
    )

    assert.equal(result.requestId, 'gateway-request-id')
    assert.equal(result.success, true)
  })
})
