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
export type PlaygroundMessagesRecord = {
  version: number
  data: unknown
  revision?: number
  cleared?: boolean
}

export type PlaygroundMessagesReadResult =
  | { status: 'success'; record: PlaygroundMessagesRecord }
  | { status: 'empty' | 'error' | 'unavailable' | 'unsupported' }

const DATABASE_NAME = 'new-api-playground'
const DATABASE_VERSION = 1
const MESSAGES_STORE_NAME = 'messages'
const CURRENT_MESSAGES_KEY = 'current'

function openPlaygroundDatabase(): Promise<IDBDatabase | null> {
  if (typeof indexedDB === 'undefined') return Promise.resolve(null)

  return new Promise((resolve) => {
    let settled = false
    const finish = (database: IDBDatabase | null) => {
      if (settled) {
        database?.close()
        return
      }

      settled = true
      resolve(database)
    }

    try {
      const request = indexedDB.open(DATABASE_NAME, DATABASE_VERSION)

      request.addEventListener('upgradeneeded', () => {
        const database = request.result
        if (!database.objectStoreNames.contains(MESSAGES_STORE_NAME)) {
          database.createObjectStore(MESSAGES_STORE_NAME)
        }
      })
      request.addEventListener('success', () => finish(request.result))
      request.addEventListener('error', () => finish(null))
      request.addEventListener('blocked', () => finish(null))
    } catch {
      finish(null)
    }
  })
}

export async function readPlaygroundMessages(): Promise<PlaygroundMessagesReadResult> {
  if (typeof indexedDB === 'undefined') {
    return { status: 'unsupported' }
  }

  const database = await openPlaygroundDatabase()
  if (!database) return { status: 'unavailable' }

  try {
    const result = await new Promise<
      | { status: 'success'; record: PlaygroundMessagesRecord }
      | { status: 'empty' | 'error' }
    >((resolve) => {
      try {
        const transaction = database.transaction(
          MESSAGES_STORE_NAME,
          'readonly'
        )
        const request = transaction
          .objectStore(MESSAGES_STORE_NAME)
          .get(CURRENT_MESSAGES_KEY)
        let settled = false
        const finish = (
          value:
            | { status: 'success'; record: PlaygroundMessagesRecord }
            | { status: 'empty' | 'error' }
        ) => {
          if (settled) return
          settled = true
          resolve(value)
        }

        request.addEventListener('success', () => {
          const record = request.result as PlaygroundMessagesRecord | undefined
          finish(record ? { status: 'success', record } : { status: 'empty' })
        })
        request.addEventListener('error', () => finish({ status: 'error' }))
        transaction.addEventListener('error', () => finish({ status: 'error' }))
        transaction.addEventListener('abort', () => finish({ status: 'error' }))
      } catch {
        resolve({ status: 'error' })
      }
    })

    return result
  } finally {
    database.close()
  }
}

async function mutatePlaygroundMessages(
  operation: (store: IDBObjectStore) => void
): Promise<boolean> {
  const database = await openPlaygroundDatabase()
  if (!database) return false

  try {
    return await new Promise<boolean>((resolve) => {
      try {
        const transaction = database.transaction(
          MESSAGES_STORE_NAME,
          'readwrite'
        )
        operation(transaction.objectStore(MESSAGES_STORE_NAME))
        transaction.addEventListener('complete', () => resolve(true))
        transaction.addEventListener('error', () => resolve(false))
        transaction.addEventListener('abort', () => resolve(false))
      } catch {
        resolve(false)
      }
    })
  } finally {
    database.close()
  }
}

export function writePlaygroundMessages(
  record: PlaygroundMessagesRecord
): Promise<boolean> {
  return mutatePlaygroundMessages((store) => {
    store.put(record, CURRENT_MESSAGES_KEY)
  })
}

export function clearPlaygroundMessages(
  record: PlaygroundMessagesRecord
): Promise<boolean> {
  // Keep a small tombstone instead of deleting the key. If the delete/write
  // races with a reload, the cleared state cannot resurrect an older snapshot.
  return mutatePlaygroundMessages((store) => {
    store.put(record, CURRENT_MESSAGES_KEY)
  })
}
