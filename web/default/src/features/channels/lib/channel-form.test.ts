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
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
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
  test('normalizes an invalid zero probe period and preserves zero routing values', () => {
    const defaults = transformChannelToFormDefaults(
      channel({
        probe_period_minutes: 0,
        price_multiplier: 0,
      })
    )

    assert.equal(defaults.probe_period_minutes, 5)
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

  test('uses the default channel concurrency when it is not configured', () => {
    const defaults = transformChannelToFormDefaults(
      channel({ max_concurrency: null })
    )

    assert.equal(defaults.max_concurrency, 1000)
    const parsed = channelFormSchema.parse(defaults)
    const payload = transformFormDataToUpdatePayload(parsed, 42)
    assert.equal(payload.max_concurrency, 1000)
  })

  test('normalizes an invalid zero channel concurrency to the default', () => {
    const defaults = transformChannelToFormDefaults(
      channel({ max_concurrency: 0 })
    )

    assert.equal(defaults.max_concurrency, 1000)
  })

  test('uses the updated routing strategy defaults for new channels', () => {
    assert.deepEqual(
      {
        auto_ban: CHANNEL_FORM_DEFAULT_VALUES.auto_ban,
        auto_probe_enabled: CHANNEL_FORM_DEFAULT_VALUES.auto_probe_enabled,
        probe_period_minutes: CHANNEL_FORM_DEFAULT_VALUES.probe_period_minutes,
        probe_random_delay_enabled:
          CHANNEL_FORM_DEFAULT_VALUES.probe_random_delay_enabled,
        probe_stream_enabled: CHANNEL_FORM_DEFAULT_VALUES.probe_stream_enabled,
        upstream_max_retries: CHANNEL_FORM_DEFAULT_VALUES.upstream_max_retries,
        max_concurrency: CHANNEL_FORM_DEFAULT_VALUES.max_concurrency,
        price_multiplier: CHANNEL_FORM_DEFAULT_VALUES.price_multiplier,
      },
      {
        auto_ban: 0,
        auto_probe_enabled: false,
        probe_period_minutes: 5,
        probe_random_delay_enabled: false,
        probe_stream_enabled: false,
        upstream_max_retries: 0,
        max_concurrency: 1000,
        price_multiplier: 1,
      }
    )

    const defaults = transformChannelToFormDefaults(
      channel({
        auto_ban: null,
        probe_period_minutes: undefined,
        probe_random_delay_enabled: undefined,
        probe_stream_enabled: undefined,
        max_concurrency: null,
      })
    )

    assert.equal(defaults.auto_ban, 0)
    assert.equal(defaults.auto_probe_enabled, false)
    assert.equal(defaults.probe_period_minutes, 5)
    assert.equal(defaults.probe_random_delay_enabled, false)
    assert.equal(defaults.probe_stream_enabled, false)
    assert.equal(defaults.upstream_max_retries, 0)
    assert.equal(defaults.max_concurrency, 1000)
    assert.equal(defaults.price_multiplier, 1)
  })

  test('accepts channel responses that omit credentials', () => {
    const parsed = channel()

    assert.equal(parsed.key, undefined)
    assert.equal(transformChannelToFormDefaults(parsed).key, '')
  })

  for (const enabled of [true, false]) {
    test(`自动探测为 ${enabled} 时使用单一开关保存且不改变业务自动禁用`, () => {
      const defaults = transformChannelToFormDefaults(
        channel({
          auto_probe_enabled: enabled,
          auto_ban: 0,
          probe_period_minutes: 3,
          probe_random_delay_enabled: true,
          probe_stream_enabled: true,
        })
      )
      const parsed = channelFormSchema.parse(defaults)
      const created = transformFormDataToCreatePayload(parsed).channel
      const updated = transformFormDataToUpdatePayload(parsed, 42)

      assert.equal(defaults.auto_probe_enabled, enabled)
      for (const payload of [created, updated]) {
        assert.equal(payload.auto_probe_enabled, enabled)
        assert.equal(payload.auto_ban, 0)
        assert.equal(Object.hasOwn(payload, 'probe_failure_auto_ban'), false)
        assert.equal(Object.hasOwn(payload, 'probe_success_auto_enable'), false)
        assert.equal(payload.probe_period_minutes, 3)
        assert.equal(payload.probe_random_delay_enabled, true)
        assert.equal(payload.probe_stream_enabled, true)
      }
    })
  }

  test('requires a configured test model', () => {
    const defaults = transformChannelToFormDefaults(channel())
    defaults.test_model = ''

    assert.throws(
      () => channelFormSchema.parse(defaults),
      /Test model is required/
    )
  })
})
