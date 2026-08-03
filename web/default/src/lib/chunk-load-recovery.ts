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
const CHUNK_ERROR_PATTERN =
  /ChunkLoadError|Loading chunk .+ failed|Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module/i

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return `${error.name}: ${error.message}`
  if (typeof error === 'string') return error
  if (error && typeof error === 'object' && 'message' in error) {
    return String(error.message)
  }
  return ''
}

export function isChunkLoadError(error: unknown): boolean {
  return CHUNK_ERROR_PATTERN.test(getErrorMessage(error))
}

function getClientBuildId(): string {
  for (const script of document.scripts) {
    const match = script.src.match(/\/static\/js\/index\.([a-f0-9]+)\.js$/i)
    if (match) return match[1]
  }
  return document.documentElement.dataset.buildRev || 'unknown'
}

export function installChunkLoadRecovery(): void {
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  const reloadOnce = () => {
    const storageKey = `chunk-reload:${getClientBuildId()}`
    try {
      if (window.sessionStorage.getItem(storageKey) === '1') return
      window.sessionStorage.setItem(storageKey, '1')
    } catch {
      return
    }
    window.location.reload()
  }

  window.addEventListener('unhandledrejection', (event) => {
    if (isChunkLoadError(event.reason)) reloadOnce()
  })
  window.addEventListener(
    'error',
    (event) => {
      const target = event.target
      if (
        target instanceof HTMLScriptElement &&
        target.src.includes('/static/js/')
      ) {
        reloadOnce()
        return
      }
      if (
        event instanceof ErrorEvent &&
        isChunkLoadError(event.error || event.message)
      ) {
        reloadOnce()
      }
    },
    true
  )
}
