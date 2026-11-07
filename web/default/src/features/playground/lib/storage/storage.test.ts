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
import { after, beforeEach, describe, test } from 'node:test'

import { IDBFactory } from 'fake-indexeddb'

import { STORAGE_KEYS } from '../../constants'
import type { Message } from '../../types'
import {
  clearPlaygroundData,
  loadMessagesWithImages,
  saveMessages,
} from './storage'

const originalIndexedDb = Object.getOwnPropertyDescriptor(
  globalThis,
  'indexedDB'
)
const originalLocalStorage = Object.getOwnPropertyDescriptor(
  globalThis,
  'localStorage'
)

class MemoryStorage implements Storage {
  private readonly values = new Map<string, string>()

  get length(): number {
    return this.values.size
  }

  clear(): void {
    this.values.clear()
  }

  getItem(key: string): string | null {
    return this.values.get(key) ?? null
  }

  key(index: number): string | null {
    return [...this.values.keys()][index] ?? null
  }

  removeItem(key: string): void {
    this.values.delete(key)
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value)
  }
}

function createImageMessage(): Message {
  return {
    key: 'assistant-image',
    from: 'assistant',
    versions: [{ id: 'version-1', content: '' }],
    status: 'complete',
    images: [
      {
        url: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB',
        revisedPrompt: 'A durable image',
      },
    ],
  }
}

function createTextMessage(key: string, content: string): Message {
  return {
    key,
    from: 'user',
    versions: [{ id: `${key}-version`, content }],
  }
}

beforeEach(async () => {
  Object.defineProperty(globalThis, 'indexedDB', {
    configurable: true,
    value: new IDBFactory(),
  })
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: new MemoryStorage(),
  })
})

after(() => {
  if (originalIndexedDb) {
    Object.defineProperty(globalThis, 'indexedDB', originalIndexedDb)
  } else {
    Reflect.deleteProperty(globalThis, 'indexedDB')
  }

  if (originalLocalStorage) {
    Object.defineProperty(globalThis, 'localStorage', originalLocalStorage)
  } else {
    Reflect.deleteProperty(globalThis, 'localStorage')
  }
})

describe('playground message persistence', () => {
  test('restores generated images without putting base64 data in localStorage', async () => {
    const messages = [createImageMessage()]

    await saveMessages(messages)

    const localEnvelope = JSON.parse(
      localStorage.getItem(STORAGE_KEYS.MESSAGES) ?? '{}'
    ) as { data?: Array<{ images?: unknown }> }
    assert.equal(localEnvelope.data?.[0]?.images, undefined)
    assert.deepEqual(await loadMessagesWithImages(), messages)
  })

  test('persists image removal when the conversation is cleared', async () => {
    await saveMessages([createImageMessage()])
    await saveMessages([])
    localStorage.removeItem(STORAGE_KEYS.MESSAGES)

    assert.deepEqual(await loadMessagesWithImages(), [])
  })

  test('migrates existing localStorage messages into IndexedDB', async () => {
    const legacyMessages: Message[] = [
      {
        key: 'legacy-user',
        from: 'user',
        versions: [{ id: 'legacy-version', content: 'hello' }],
      },
    ]
    localStorage.setItem(
      STORAGE_KEYS.MESSAGES,
      JSON.stringify({ version: 1, data: legacyMessages })
    )

    assert.deepEqual(await loadMessagesWithImages(), legacyMessages)

    localStorage.removeItem(STORAGE_KEYS.MESSAGES)
    assert.deepEqual(await loadMessagesWithImages(), legacyMessages)
  })

  test('migrates legacy localStorage snapshots that already contain images', async () => {
    const legacyMessages = [createImageMessage()]
    localStorage.setItem(
      STORAGE_KEYS.MESSAGES,
      JSON.stringify({ version: 1, data: legacyMessages })
    )

    assert.deepEqual(await loadMessagesWithImages(), legacyMessages)
    localStorage.removeItem(STORAGE_KEYS.MESSAGES)
    assert.deepEqual(await loadMessagesWithImages(), legacyMessages)
  })

  test('does not restore an older IndexedDB snapshot over a newer local backup', async () => {
    await saveMessages([createImageMessage()])
    const newerMessages = [createTextMessage('newer', 'newer message')]
    localStorage.setItem(
      STORAGE_KEYS.MESSAGES,
      JSON.stringify({
        version: 2,
        revision: Date.now() + 1_000_000,
        data: newerMessages,
      })
    )

    assert.deepEqual(await loadMessagesWithImages(), newerMessages)
    localStorage.removeItem(STORAGE_KEYS.MESSAGES)
    assert.deepEqual(await loadMessagesWithImages(), newerMessages)
  })

  test('keeps durable images when a newer local text backup is recovered', async () => {
    await saveMessages([createImageMessage()])
    const newerMessages: Message[] = [
      {
        ...createImageMessage(),
        versions: [{ id: 'version-1', content: 'updated prompt' }],
      },
    ]
    const localBackupMessages = newerMessages.map(
      ({ images: _images, ...message }) => message
    )
    const currentEnvelope = JSON.parse(
      localStorage.getItem(STORAGE_KEYS.MESSAGES) ?? '{}'
    ) as { revision?: number }
    localStorage.setItem(
      STORAGE_KEYS.MESSAGES,
      JSON.stringify({
        version: 2,
        revision: (currentEnvelope.revision ?? 0) + 1,
        data: localBackupMessages,
      })
    )

    assert.deepEqual(await loadMessagesWithImages(), newerMessages)
    localStorage.removeItem(STORAGE_KEYS.MESSAGES)
    assert.deepEqual(await loadMessagesWithImages(), newerMessages)
  })

  test('honors a clear marker when the durable delete cannot be observed', async () => {
    await saveMessages([createImageMessage()])
    await clearPlaygroundData()

    assert.equal(localStorage.getItem(STORAGE_KEYS.MESSAGES), null)
    assert.equal(
      localStorage.getItem(STORAGE_KEYS.MESSAGES_CLEARED) !== null,
      true
    )
    assert.equal(await loadMessagesWithImages(), null)
  })

  test('keeps a newer clear marker ahead of an intermediate local backup', async () => {
    await saveMessages([createImageMessage()])
    const savedEnvelope = JSON.parse(
      localStorage.getItem(STORAGE_KEYS.MESSAGES) ?? '{}'
    ) as { revision?: number }
    const durableRevision = savedEnvelope.revision ?? 0

    localStorage.setItem(
      STORAGE_KEYS.MESSAGES,
      JSON.stringify({
        version: 2,
        revision: durableRevision + 1,
        data: [createTextMessage('intermediate', 'intermediate message')],
      })
    )
    localStorage.setItem(
      STORAGE_KEYS.MESSAGES_CLEARED,
      JSON.stringify({
        version: 2,
        revision: durableRevision + 2,
        cleared: true,
        data: null,
      })
    )

    assert.equal(await loadMessagesWithImages(), null)
  })

  test('allows a newer local save to supersede an older durable clear marker', async () => {
    await saveMessages([createImageMessage()])
    await clearPlaygroundData()
    const clearEnvelope = JSON.parse(
      localStorage.getItem(STORAGE_KEYS.MESSAGES_CLEARED) ?? '{}'
    ) as { revision?: number }
    const newerMessages = [createTextMessage('newer', 'newer message')]

    localStorage.setItem(
      STORAGE_KEYS.MESSAGES,
      JSON.stringify({
        version: 2,
        revision: (clearEnvelope.revision ?? 0) + 1,
        data: newerMessages,
      })
    )

    assert.deepEqual(await loadMessagesWithImages(), newerMessages)
  })

  test('accepts a null revised prompt from an upstream image response', async () => {
    const messages = [
      {
        ...createImageMessage(),
        images: [{ url: 'data:image/png;base64,abc', revisedPrompt: null }],
      } as unknown as Message,
    ]

    await saveMessages(messages)

    assert.deepEqual(await loadMessagesWithImages(), [
      {
        ...createImageMessage(),
        images: [
          { url: 'data:image/png;base64,abc', revisedPrompt: undefined },
        ],
      },
    ])
  })

  test('clears both durable and fallback message storage', async () => {
    await saveMessages([createImageMessage()])
    await clearPlaygroundData()

    assert.equal(localStorage.getItem(STORAGE_KEYS.MESSAGES), null)
    assert.equal(await loadMessagesWithImages(), null)
  })
})
