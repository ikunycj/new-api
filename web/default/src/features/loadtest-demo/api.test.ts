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
  supportsLoadTestStreaming,
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

  test('omits prompt-cache input when the cache switch is off', () => {
    assert.deepEqual(
      buildLoadTestRequestBody(
        'gpt-5.6-sol',
        'Do not add a cache prefix.',
        false,
        'openai'
      ),
      {
        model: 'gpt-5.6-sol',
        max_tokens: 32,
        messages: [{ role: 'user', content: 'Do not add a cache prefix.' }],
        temperature: 0,
        stream: false,
      }
    )
  })

  test('requests usage in OpenAI streaming responses', () => {
    assert.deepEqual(
      buildLoadTestRequestBody(
        'gpt-5.6-sol',
        'Stream this response.',
        false,
        'openai',
        true
      ),
      {
        model: 'gpt-5.6-sol',
        max_tokens: 32,
        messages: [{ role: 'user', content: 'Stream this response.' }],
        temperature: 0,
        stream: true,
        stream_options: { include_usage: true },
      }
    )
  })

  test('does not mark responses compaction as streaming', () => {
    assert.equal(supportsLoadTestStreaming('openai-response-compact'), false)
    assert.deepEqual(
      buildLoadTestRequestBody(
        'gpt-5.6-sol-compact',
        'Compact this response.',
        false,
        'openai-response-compact',
        true
      ),
      {
        model: 'gpt-5.6-sol-compact',
        input: [{ role: 'user', content: 'Compact this response.' }],
        stream: false,
      }
    )
  })

  test('reads streaming SSE responses and reports TTFT and output speed', async () => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: { clearTimeout, setTimeout },
    })
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(
          new TextEncoder().encode(
            'data: {"choices":[{"delta":{"content":"OK"}}]}\n\n'
          )
        )
        controller.enqueue(
          new TextEncoder().encode(
            'data: {"usage":{"prompt_tokens":4,"completion_tokens":2}}\n\ndata: [DONE]\n\n'
          )
        )
        controller.close()
      },
    })
    globalThis.fetch = (async () =>
      new Response(stream, { status: 200 })) as typeof fetch

    const result = await sendLoadTestRequest(
      'https://api.example.com',
      { name: 'test-key', secret: 'test-secret' } as LoadTestKey,
      'gpt-5.6-sol',
      'load-test-run',
      'Stream this.',
      false,
      undefined,
      'openai',
      'openai',
      true
    )

    assert.equal(result.success, true)
    assert.equal(result.usage?.outputTokens, 2)
    assert.ok((result.firstTokenLatency ?? 0) >= 0)
    assert.ok((result.outputTokensPerSecond ?? 0) >= 0)
  })

  test('preserves explicit zero usage fields instead of falling back', async () => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: { clearTimeout, setTimeout },
    })
    globalThis.fetch = (async () =>
      new Response(
        JSON.stringify({
          usage: {
            input_tokens: 0,
            prompt_tokens: 99,
            output_tokens: 0,
            completion_tokens: 88,
          },
        }),
        { status: 200 }
      )) as typeof fetch

    const result = await sendLoadTestRequest(
      'https://api.example.com',
      { name: 'test-key', secret: 'test-secret' } as LoadTestKey,
      'gpt-5.6-sol',
      'load-test-run',
      'Zero usage.',
      false
    )

    assert.deepEqual(result.usage, {
      inputTokens: 0,
      outputTokens: 0,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    })
  })

  test('merges split Anthropic streaming usage events', async () => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: { clearTimeout, setTimeout },
    })
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(
          new TextEncoder().encode(
            'event: message_start\ndata: {"message":{"usage":{"input_tokens":17}}}\n\n'
          )
        )
        controller.enqueue(
          new TextEncoder().encode(
            'event: message_delta\ndata: {"usage":{"output_tokens":3}}\n\n'
          )
        )
        controller.close()
      },
    })
    globalThis.fetch = (async () =>
      new Response(stream, { status: 200 })) as typeof fetch

    const result = await sendLoadTestRequest(
      'https://api.example.com',
      { name: 'test-key', secret: 'test-secret' } as LoadTestKey,
      'claude-opus-4-8',
      'load-test-run',
      'Stream this.',
      false,
      undefined,
      'claude',
      'anthropic',
      true
    )

    assert.deepEqual(result.usage, {
      inputTokens: 17,
      outputTokens: 3,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    })
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
