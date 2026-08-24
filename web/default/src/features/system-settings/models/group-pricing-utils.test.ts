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

import { reorderBillingGroupChannels } from './group-pricing-utils'

const entries = [
  { id: 11, channel_id: 35, priority: 300 },
  { id: 12, channel_id: 38, priority: 300 },
  { id: 13, channel_id: 36, priority: 200 },
]

describe('billing group channel ordering', () => {
  test('moves a channel down when adjacent priorities are equal', () => {
    const reordered = reorderBillingGroupChannels(entries, 35, 1)

    assert.deepEqual(reordered, [
      { id: 12, channel_id: 38, priority: 3 },
      { id: 11, channel_id: 35, priority: 2 },
      { id: 13, channel_id: 36, priority: 1 },
    ])
  })

  test('moves a channel up and keeps unrelated entries in route order', () => {
    const reordered = reorderBillingGroupChannels(entries, 36, -1)

    assert.deepEqual(reordered, [
      { id: 11, channel_id: 35, priority: 3 },
      { id: 13, channel_id: 36, priority: 2 },
      { id: 12, channel_id: 38, priority: 1 },
    ])
  })

  test('moves a channel only within its protocol while preserving global slots', () => {
    const protocolChannelIDs = new Set([35, 36])
    const reordered = reorderBillingGroupChannels(
      entries,
      36,
      -1,
      protocolChannelIDs
    )

    assert.deepEqual(reordered, [
      { id: 13, channel_id: 36, priority: 3 },
      { id: 12, channel_id: 38, priority: 2 },
      { id: 11, channel_id: 35, priority: 1 },
    ])
  })

  test('does not move beyond the current protocol boundary', () => {
    const protocolChannelIDs = new Set([35, 36])

    assert.strictEqual(
      reorderBillingGroupChannels(entries, 35, -1, protocolChannelIDs),
      entries
    )
  })

  test('returns the original entries when movement is out of bounds', () => {
    assert.strictEqual(reorderBillingGroupChannels(entries, 35, -1), entries)
    assert.strictEqual(reorderBillingGroupChannels(entries, 999, 1), entries)
  })
})
