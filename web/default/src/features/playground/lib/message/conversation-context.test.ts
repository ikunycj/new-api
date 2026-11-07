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
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../../constants'
import type { Message } from '../../types'
import {
  CONVERSATION_SUMMARY_TOKEN_LIMIT,
  CONVERSATION_WINDOW_ROUNDS,
  prepareConversationContext,
} from './conversation-context'
import { createUserMessage } from './message-utils'

function assistant(content: string): Message {
  return {
    key: `assistant-${content}`,
    from: 'assistant',
    versions: [{ id: `version-${content}`, content }],
    status: 'complete',
  }
}

function conversation(roundCount: number, contentSize = 1): Message[] {
  return Array.from({ length: roundCount }, (_, index) => [
    createUserMessage(
      `User request ${index} ${'details '.repeat(contentSize)}`
    ),
    assistant(`Assistant answer ${index} ${'details '.repeat(contentSize)}`),
  ]).flat()
}

function options(overrides: Partial<typeof DEFAULT_CONFIG> = {}) {
  return {
    model: 'gpt-4o',
    existingSummary: '',
    config: { ...DEFAULT_CONFIG, ...overrides },
    parameterEnabled: DEFAULT_PARAMETER_ENABLED,
    compressSummary: async (_source: string, maxTokens: number) => {
      assert.ok(maxTokens <= CONVERSATION_SUMMARY_TOKEN_LIMIT)
      return 'Compressed conversation summary'
    },
  }
}

describe('conversation context', () => {
  test('keeps the latest five previous exchanges without compression when they fit', async () => {
    const result = await prepareConversationContext(conversation(7), options())

    assert.equal(result.summary, 'Compressed conversation summary')
    assert.match(String(result.messages.at(-1)?.content), /Assistant answer 6/)
    assert.equal(
      result.messages.filter((message) => message.role === 'user').length,
      CONVERSATION_WINDOW_ROUNDS + 1
    )
  })

  test('reduces the sliding window before compressing again when the context is small', async () => {
    const summarySources: string[] = []
    const result = await prepareConversationContext(conversation(7, 80), {
      ...options({ contextWindowTokens: 6000 }),
      compressSummary: async (source, maxTokens) => {
        summarySources.push(source)
        assert.ok(maxTokens <= CONVERSATION_SUMMARY_TOKEN_LIMIT)
        return 'Short summary'
      },
    })

    assert.ok(summarySources.length > 1)
    assert.equal(result.summary, 'Short summary')
    assert.match(String(result.messages.at(-1)?.content), /Assistant answer 6/)
  })

  test('preserves image context and the latest prompt', async () => {
    const result = await prepareConversationContext(conversation(2), options())

    assert.match(result.imagePrompt, /Assistant answer 1/)
    assert.match(
      result.imagePrompt,
      /Create the image requested by the latest user message/
    )
  })
})
