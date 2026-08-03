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

export interface CachedDocsPage {
  fileName: string
  html: string
  key: string
  savedAt: number
}

const DATABASE_NAME = 'new-api-documents'
const STORE_NAME = 'documents'

let databasePromise: Promise<IDBDatabase | null> | undefined

function openDocsDatabase(): Promise<IDBDatabase | null> {
  if (databasePromise) return databasePromise
  if (typeof indexedDB === 'undefined') return Promise.resolve(null)

  databasePromise = new Promise((resolve) => {
    const request = indexedDB.open(DATABASE_NAME, 1)
    request.addEventListener('upgradeneeded', () => {
      const database = request.result
      if (!database.objectStoreNames.contains(STORE_NAME)) {
        database.createObjectStore(STORE_NAME, { keyPath: 'key' })
      }
    })
    request.addEventListener('success', () => resolve(request.result))
    request.addEventListener('error', () => resolve(null))
    request.addEventListener('blocked', () => resolve(null))
  })
  return databasePromise
}

export async function getCachedDocsPage(
  key: string
): Promise<CachedDocsPage | undefined> {
  const database = await openDocsDatabase()
  if (!database) return undefined

  return new Promise((resolve) => {
    const request = database
      .transaction(STORE_NAME, 'readonly')
      .objectStore(STORE_NAME)
      .get(key)
    request.addEventListener('success', () =>
      resolve(request.result as CachedDocsPage)
    )
    request.addEventListener('error', () => resolve(undefined))
  })
}

export async function putCachedDocsPage(page: CachedDocsPage): Promise<void> {
  const database = await openDocsDatabase()
  if (!database) return

  await new Promise<void>((resolve) => {
    const transaction = database.transaction(STORE_NAME, 'readwrite')
    transaction.objectStore(STORE_NAME).put(page)
    transaction.addEventListener('complete', () => resolve())
    transaction.addEventListener('error', () => resolve())
    transaction.addEventListener('abort', () => resolve())
  })
}
