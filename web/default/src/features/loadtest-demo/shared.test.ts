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
*/
import assert from 'node:assert/strict'
import test from 'node:test'

import type { LoadTestAgent } from './api'
import {
  getJoinedWorkerLabel,
  getSharedCapacityEstimate,
  parseSharedWorkerCount,
} from './shared'

const agent: LoadTestAgent = {
  id: 'agent-1',
  managed: true,
  name: 'shared',
  platform: 'linux',
  version: '0.5.0',
  cpu_cores: 2,
  memory_bytes: 1024,
  max_rps: 100,
  max_concurrency: 40,
  last_seen_at: 1,
  created_at: 1,
}

test('shared UI accepts only a bounded integer worker count', () => {
  assert.equal(parseSharedWorkerCount('2'), 2)
  assert.equal(parseSharedWorkerCount('256'), 256)
  assert.equal(parseSharedWorkerCount('1'), null)
  assert.equal(parseSharedWorkerCount('257'), null)
  assert.equal(parseSharedWorkerCount('2.5'), null)
})

test('shared UI estimates aggregate capacity from the paired agent', () => {
  assert.deepEqual(getSharedCapacityEstimate(agent, 3, 'shared'), {
    rps: 300,
    concurrency: 120,
  })
  assert.deepEqual(
    getSharedCapacityEstimate(agent, 3, 'shared', {
      max_rps: 250,
      max_concurrency: 100,
    }),
    { rps: 250, concurrency: 100 }
  )
  assert.equal(getSharedCapacityEstimate(agent, 3, 'single'), null)
})

test('shared UI renders joined worker progress only for shared runs', () => {
  assert.equal(getJoinedWorkerLabel('shared', 2, 3), '2/3')
  assert.equal(getJoinedWorkerLabel('single', 1, 1), null)
})
