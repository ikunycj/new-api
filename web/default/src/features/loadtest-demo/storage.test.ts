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
  clearPersistedLoadTestRuns,
  loadPersistedLoadTestRun,
  loadPersistedLoadTestRuns,
  savePersistedLoadTestRun,
  type LoadTestRunResult,
} from './storage'

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
const originalDateNow = Date.now

function installLocalStorage() {
  const values = new Map<string, string>()
  const localStorage = {
    getItem: (key: string) => values.get(key) ?? null,
    removeItem: (key: string) => values.delete(key),
    setItem: (key: string, value: string) => values.set(key, value),
  }
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: { localStorage },
  })
  return values
}

const result: LoadTestRunResult = {
  model: 'gpt-5.6-sol',
  runId: 'demo-result-1',
  durationSeconds: 60,
  requestsPerSecond: 2,
  estimatedCost: 0.12,
  requestIds: ['request-1'],
  stats: {
    completed: 1,
    failures: 0,
    latencies: [120],
    successes: 1,
    statusCodes: { '200': 1 },
    errorCodes: {},
    keyCounts: { primary: 1 },
    inputTokens: 10,
    outputTokens: 2,
    cacheReadTokens: 5,
    cacheWriteTokens: 0,
  },
  channelStats: [
    {
      channel_id: 38,
      channel_name: 'primary',
      billing_group: 'default',
      cost_factor: 0.2,
      requests: 1,
      input_tokens: 10,
      input_tokens_total: 15,
      output_tokens: 2,
      cache_read_tokens: 5,
      cache_write_tokens: 0,
    },
  ],
}

afterEach(() => {
  Date.now = originalDateNow
  if (originalWindow) {
    Object.defineProperty(globalThis, 'window', originalWindow)
  } else {
    Reflect.deleteProperty(globalThis, 'window')
  }
})

describe('load test result storage', () => {
  test('restores channel results only for the same user', () => {
    installLocalStorage()
    Date.now = () => 1_000_000

    savePersistedLoadTestRun(42, result)

    assert.deepEqual(loadPersistedLoadTestRun(42), {
      version: 2,
      savedAt: 1_000_000,
      completedAt: 1_000_000,
      ...result,
    })
    assert.equal(loadPersistedLoadTestRun(43), null)
  })

  test('removes results after seven days', () => {
    const values = installLocalStorage()
    Date.now = () => 1_000_000
    savePersistedLoadTestRun(42, result)
    Date.now = () => 1_000_000 + 7 * 24 * 60 * 60 * 1000 + 1

    assert.equal(loadPersistedLoadTestRun(42), null)
    assert.equal(values.size, 0)
  })

  test('does not persist an API key or cluster fields', () => {
    const values = installLocalStorage()

    savePersistedLoadTestRun(42, result)

    const raw = [...values.values()][0] ?? ''
    assert.equal(raw.includes('sk-'), false)
    assert.equal(raw.includes('cluster_id'), false)
    assert.equal(raw.includes('pool_name'), false)
    assert.equal(raw.includes('billing_group'), true)
  })

  test('keeps multiple runs and supports clearing them', () => {
    const values = installLocalStorage()
    Date.now = () => 1_000_000
    savePersistedLoadTestRun(42, result)
    Date.now = () => 1_000_100
    savePersistedLoadTestRun(42, { ...result, runId: 'demo-result-2' })

    assert.deepEqual(
      loadPersistedLoadTestRuns(42).map((run) => run.runId),
      ['demo-result-2', 'demo-result-1']
    )
    clearPersistedLoadTestRuns(42)
    assert.deepEqual(loadPersistedLoadTestRuns(42), [])
    assert.equal(values.size, 0)
  })
})
