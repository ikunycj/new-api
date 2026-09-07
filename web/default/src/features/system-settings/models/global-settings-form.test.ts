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
import { describe, test } from 'node:test'

import {
  DEFAULT_CHAT_COMPLETIONS_TO_RESPONSES_POLICY,
  globalModelSettingsFormSchema,
  parseGlobalModelSettings,
  serializeGlobalModelSettings,
} from './global-settings-form'

describe('global model settings form mapping', () => {
  test('normalizes legacy JSON lists while preserving preferred model order', () => {
    const values = parseGlobalModelSettings({
      'global.pass_through_request_enabled': 'true' as unknown as boolean,
      'global.thinking_model_blacklist':
        '[" kimi-k2-thinking ","",2,"kimi-k2-thinking"]',
      PreferredModels: '[" claude-fable-5 ","gpt-5.6-sol","claude-fable-5"]',
      'global.chat_completions_to_responses_policy': '{}',
      'general_setting.ping_interval_enabled': false,
      'general_setting.ping_interval_seconds': 60,
    })

    assert.equal(values.global.pass_through_request_enabled, true)
    assert.deepEqual(values.global.thinking_model_blacklist, [
      'kimi-k2-thinking',
    ])
    assert.deepEqual(values.PreferredModels, ['claude-fable-5', 'gpt-5.6-sol'])
    assert.deepEqual(
      values.global.chat_completions_to_responses_policy,
      DEFAULT_CHAT_COMPLETIONS_TO_RESPONSES_POLICY
    )
  })

  test('falls back safely when persisted JSON is malformed', () => {
    const values = parseGlobalModelSettings({
      'global.thinking_model_blacklist': '{not-json',
      PreferredModels: 'not-json',
      'global.chat_completions_to_responses_policy': '[1,2,3]',
      'general_setting.ping_interval_seconds': Number.NaN,
    })

    assert.deepEqual(values.global.thinking_model_blacklist, [
      'moonshotai/kimi-k2-thinking',
      'kimi-k2-thinking',
    ])
    assert.deepEqual(values.PreferredModels, ['gpt-5.6-sol', 'claude-fable-5'])
    assert.deepEqual(
      values.global.chat_completions_to_responses_policy,
      DEFAULT_CHAT_COMPLETIONS_TO_RESPONSES_POLICY
    )
    assert.equal(values.general_setting.ping_interval_seconds, 60)
  })

  test('filters invalid channel IDs and canonicalizes policy JSON', () => {
    const values = parseGlobalModelSettings({
      'global.chat_completions_to_responses_policy': JSON.stringify({
        enabled: true,
        all_channels: false,
        channel_ids: [3, 1, 1, 0, -4, 1.5, Number.MAX_SAFE_INTEGER + 1],
        channel_types: [24, '3', null],
        model_patterns: ['^gpt', ' [', '^gpt'],
      }),
    })

    assert.deepEqual(
      values.global.chat_completions_to_responses_policy.channel_ids,
      [1, 3]
    )
    assert.deepEqual(
      values.global.chat_completions_to_responses_policy.channel_types,
      [3, 24]
    )
    assert.deepEqual(
      serializeGlobalModelSettings(values)[
        'global.chat_completions_to_responses_policy'
      ],
      '{"enabled":true,"all_channels":false,"channel_ids":[1,3],"channel_types":[3,24],"model_patterns":["^gpt","["]}'
    )
  })

  test('rejects invalid channel IDs and regular expressions in form values', () => {
    const values = parseGlobalModelSettings({})
    values.global.chat_completions_to_responses_policy.channel_ids = [0]
    values.global.chat_completions_to_responses_policy.model_patterns = ['[']

    const result = globalModelSettingsFormSchema.safeParse(values)
    assert.equal(result.success, false)
    if (!result.success) {
      assert.ok(
        result.error.issues.some((issue) => issue.path.includes('channel_ids'))
      )
      assert.ok(
        result.error.issues.some((issue) =>
          issue.path.includes('model_patterns')
        )
      )
    }
  })

  test('requires matching rules and a channel filter for an enabled scoped policy', () => {
    const values = parseGlobalModelSettings({})
    values.global.chat_completions_to_responses_policy = {
      enabled: true,
      all_channels: false,
      channel_ids: [],
      channel_types: [],
      model_patterns: [],
    }

    const result = globalModelSettingsFormSchema.safeParse(values)
    assert.equal(result.success, false)
    if (!result.success) {
      assert.ok(
        result.error.issues.some((issue) =>
          issue.message.includes('model pattern')
        )
      )
      assert.ok(
        result.error.issues.some((issue) =>
          issue.message.includes('channel or channel type')
        )
      )
    }
  })

  test('serializes semantically equivalent values consistently', () => {
    const first = parseGlobalModelSettings({
      'global.chat_completions_to_responses_policy':
        '{ "all_channels": false, "enabled": true, "channel_ids": [2, 1] }',
    })
    const second = parseGlobalModelSettings({
      'global.chat_completions_to_responses_policy':
        '{"enabled":true,"all_channels":false,"channel_ids":[1,2]}',
    })

    assert.equal(
      serializeGlobalModelSettings(first)[
        'global.chat_completions_to_responses_policy'
      ],
      serializeGlobalModelSettings(second)[
        'global.chat_completions_to_responses_policy'
      ]
    )
  })

  test('omits channel filters when the policy applies to all channels', () => {
    const values = parseGlobalModelSettings({
      'global.chat_completions_to_responses_policy': JSON.stringify({
        enabled: true,
        all_channels: true,
        channel_ids: [8],
        channel_types: [24],
        model_patterns: ['^gpt'],
      }),
    })

    assert.equal(
      serializeGlobalModelSettings(values)[
        'global.chat_completions_to_responses_policy'
      ],
      '{"enabled":true,"all_channels":true,"model_patterns":["^gpt"]}'
    )
  })
})
