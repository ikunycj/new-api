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

import {
  buildLoadTestRequestBody,
  getLoadTestApiBaseUrl,
  getLoadTestModels,
  sendLoadTestRequest,
  type LoadTestKey,
} from './api'

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
      'Return a short answer.',
      false
    )

    assert.equal(result.requestId, 'gateway-request-id')
    assert.equal(result.success, true)
  })

  test('uses the model endpoint reported by the gateway', async () => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: { clearTimeout, setTimeout },
    })
    let requestedUrl = ''
    let requestedBody = ''
    globalThis.fetch = (async (input, init) => {
      requestedUrl = String(input)
      requestedBody = String(init?.body ?? '')
      return new Response(JSON.stringify({ usage: {} }), { status: 200 })
    }) as typeof fetch

    await sendLoadTestRequest(
      'https://api.example.com',
      { name: 'test-key', secret: 'test-secret' } as LoadTestKey,
      'o4-mini',
      'load-test-run',
      'Describe this request.',
      false,
      undefined,
      'openai',
      'openai-response'
    )

    assert.equal(requestedUrl, 'https://api.example.com/v1/responses')
    assert.deepEqual(JSON.parse(requestedBody), {
      model: 'o4-mini',
      input: [{ role: 'user', content: 'Describe this request.' }],
      max_output_tokens: 32,
      stream: false,
    })
  })

  test('builds the Anthropic preview from the same custom prompt', () => {
    assert.deepEqual(
      buildLoadTestRequestBody(
        'claude-opus-4-8',
        'Summarize the attached trajectory.',
        false,
        'anthropic'
      ),
      {
        model: 'claude-opus-4-8',
        max_tokens: 32,
        messages: [
          { role: 'user', content: 'Summarize the attached trajectory.' },
        ],
      }
    )
  })
})

describe('load test model catalog', () => {
  test('keeps every OpenAI and Anthropic model returned for the selected key', () => {
    const models = getLoadTestModels([
      {
        id: 'gpt-5.6-sol',
        ownedBy: 'openai',
        supportedEndpointTypes: ['openai'],
      },
      {
        id: 'claude-opus-4-8',
        ownedBy: 'anthropic',
        supportedEndpointTypes: ['anthropic', 'openai'],
      },
      {
        id: 'embedding-model',
        ownedBy: 'openai',
        supportedEndpointTypes: ['embeddings'],
      },
      {
        id: 'o4-mini',
        ownedBy: 'openai',
        supportedEndpointTypes: ['openai-response'],
      },
    ])

    assert.deepEqual(models, [
      { id: 'gpt-5.6-sol', provider: 'openai', endpoint: 'openai' },
      {
        id: 'claude-opus-4-8',
        provider: 'claude',
        endpoint: 'anthropic',
      },
      { id: 'o4-mini', provider: 'openai', endpoint: 'openai-response' },
    ])
  })

  test('normalizes the gateway model endpoint once', () => {
    assert.equal(
      getLoadTestApiBaseUrl('https://api.example.com/'),
      'https://api.example.com/v1'
    )
    assert.equal(
      getLoadTestApiBaseUrl('https://api.example.com/v1/'),
      'https://api.example.com/v1'
    )
  })
})
