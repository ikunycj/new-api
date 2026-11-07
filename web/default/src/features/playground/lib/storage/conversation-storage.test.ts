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
import { after, beforeEach, describe, test } from 'node:test'

import { IDBFactory } from 'fake-indexeddb'

import type { PlaygroundConversation } from '../../types'
import { createUserMessage } from '../message/message-utils'
import {
  loadPlaygroundConversations,
  removePlaygroundConversation,
  savePlaygroundConversation,
} from './conversation-storage'

const originalIndexedDb = Object.getOwnPropertyDescriptor(
  globalThis,
  'indexedDB'
)

beforeEach(() => {
  Object.defineProperty(globalThis, 'indexedDB', {
    configurable: true,
    value: new IDBFactory(),
  })
})

after(() => {
  if (originalIndexedDb) {
    Object.defineProperty(globalThis, 'indexedDB', originalIndexedDb)
  } else {
    Reflect.deleteProperty(globalThis, 'indexedDB')
  }
})

describe('playground conversation persistence', () => {
  test('stores, sorts, and removes independent conversations', async () => {
    const older: PlaygroundConversation = {
      id: 'older',
      title: 'Older',
      updatedAt: 1,
      messages: [createUserMessage('older')],
      summary: 'older summary',
    }
    const newer: PlaygroundConversation = {
      id: 'newer',
      title: 'Newer',
      updatedAt: 2,
      messages: [createUserMessage('newer')],
      summary: 'newer summary',
    }

    assert.equal(await savePlaygroundConversation(older), true)
    assert.equal(await savePlaygroundConversation(newer), true)
    assert.deepEqual(
      (await loadPlaygroundConversations()).map((item) => item.id),
      ['newer', 'older']
    )

    assert.equal(await removePlaygroundConversation('older'), true)
    assert.deepEqual(
      (await loadPlaygroundConversations()).map((item) => item.id),
      ['newer']
    )
  })
})
