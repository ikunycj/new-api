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

import { channelSchema, type Channel } from '../types'
import {
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToUpdatePayload,
} from './channel-form'

function channel(overrides: Partial<Channel> = {}): Channel {
  return channelSchema.parse({
    id: 42,
    type: 1,
    status: 1,
    name: 'primary',
    created_time: 1,
    test_time: 0,
    response_time: 0,
    balance_updated_time: 0,
    models: 'gpt-5.6-sol',
    test_model: 'gpt-5.6-sol',
    group: 'default',
    ...overrides,
  })
}

describe('channel form API mapping', () => {
  test('preserves explicit zero routing and probe values', () => {
    const defaults = transformChannelToFormDefaults(
      channel({
        probe_interval_seconds: 0,
        auto_disabled_probe_interval_seconds: 0,
        price_multiplier: 0,
      })
    )

    assert.equal(defaults.probe_interval_seconds, 0)
    assert.equal(defaults.auto_disabled_probe_interval_seconds, 0)
    assert.equal(defaults.price_multiplier, 0)
  })

  test('round-trips an inherited upstream retry limit as null', () => {
    const defaults = transformChannelToFormDefaults(
      channel({ upstream_max_retries: null })
    )
    const parsed = channelFormSchema.parse(defaults)
    const payload = transformFormDataToUpdatePayload(parsed, 42)

    assert.equal(defaults.upstream_max_retries, null)
    assert.equal(payload.upstream_max_retries, null)
  })

  test('accepts channel responses that omit credentials', () => {
    const parsed = channel()

    assert.equal(parsed.key, undefined)
    assert.equal(transformChannelToFormDefaults(parsed).key, '')
  })

  test('requires a configured test model', () => {
    const defaults = transformChannelToFormDefaults(channel())
    defaults.test_model = ''

    assert.throws(() => channelFormSchema.parse(defaults), /Test model is required/)
  })
})
