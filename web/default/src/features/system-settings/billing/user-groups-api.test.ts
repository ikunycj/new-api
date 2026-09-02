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
import { afterEach, describe, test } from 'node:test'

import { api } from '@/lib/api'

import { getUserGroupSummaries } from './user-groups-api'

const originalApiGet = api.get

afterEach(() => {
  api.get = originalApiGet
})

describe('user group API normalization', () => {
  test('fills fields added after the initial user-group API release', async () => {
    api.get = (async () => ({
      data: {
        success: true,
        data: [
          {
            id: 1,
            name: 'default',
            user_count: 3,
            created_at: 1,
            updated_at: 1,
            topup_ratio: 1,
          },
        ],
      },
    })) as typeof api.get

    const [group] = await getUserGroupSummaries()

    assert.ok(group)
    assert.equal(group.active_today, 0)
    assert.equal(group.active_month, 0)
    assert.deepEqual(group.pricing_groups, ['*'])
    assert.equal(group.pricing_groups_all, true)
  })
})
