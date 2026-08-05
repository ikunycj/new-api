import {
  deleteCachedDocsPage,
  getAllCachedDocsPages,
  getCachedDocsPage,
  putCachedDocsPage,
  requestDocsStoragePersistence,
  type CachedDocsPage,
} from './docs-cache'
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
import { DOCS_LOCALE, DOCS_ROUTES, type DocsRoutePath } from './docs-config'

export interface DocsPayload {
  html: string
}

export interface DocsManifest {
  locales: Record<string, Partial<Record<DocsRoutePath, string>>>
  version: number
}

const MANIFEST_CACHE_TTL_MS = 5 * 60 * 1000
const MAX_BACKGROUND_ROUTES = 12

let manifestPromise: Promise<DocsManifest> | undefined
let manifestFetchedAt = 0
const inFlightDocs = new Map<string, Promise<CachedDocsPage>>()
const queuedBackgroundRoutes = new Set<DocsRoutePath>()
const backgroundQueue: DocsRoutePath[] = []
let idleHandle: number | undefined
let backgroundQueueRunning = false
let persistenceRequested = false
let visibilityListenerInstalled = false
let lastPrunedManifestSignature = ''

function getDocsCacheKey(route: DocsRoutePath): string {
  return `${DOCS_LOCALE}:${route}`
}

function isSlowConnection(): boolean {
  if (typeof navigator === 'undefined') return false
  const connection = (
    navigator as Navigator & {
      connection?: {
        effectiveType?: string
        saveData?: boolean
      }
    }
  ).connection
  return (
    connection?.saveData === true ||
    connection?.effectiveType === 'slow-2g' ||
    connection?.effectiveType === '2g'
  )
}

function isDocsRoute(pathname: string): pathname is DocsRoutePath {
  return DOCS_ROUTES.some((route) => route.path === pathname)
}

function getAdjacentRoutes(route: DocsRoutePath): DocsRoutePath[] {
  const currentIndex = DOCS_ROUTES.findIndex((item) => item.path === route)
  if (currentIndex < 0) return []

  const adjacent = [
    DOCS_ROUTES[currentIndex - 1]?.path,
    DOCS_ROUTES[currentIndex + 1]?.path,
  ].filter((path): path is DocsRoutePath => Boolean(path))
  return adjacent
}

function scheduleIdle(callback: () => void): number {
  const idleWindow = window as Window & {
    requestIdleCallback?: (
      callback: IdleRequestCallback,
      options?: IdleRequestOptions
    ) => number
  }
  if (idleWindow.requestIdleCallback) {
    return idleWindow.requestIdleCallback(callback, { timeout: 2000 })
  }
  return window.setTimeout(callback, 750)
}

export function getDocsRouteFromPathname(
  pathname: string
): DocsRoutePath | undefined {
  return isDocsRoute(pathname) ? pathname : undefined
}

export function getDocsManifest(options?: {
  force?: boolean
}): Promise<DocsManifest> {
  const force = options?.force ?? false
  const now = Date.now()
  if (
    !force &&
    manifestPromise &&
    now - manifestFetchedAt < MANIFEST_CACHE_TTL_MS
  ) {
    return manifestPromise
  }

  manifestFetchedAt = now
  manifestPromise = fetch('/static/docs/manifest.json', {
    cache: 'no-cache',
    credentials: 'same-origin',
  })
    .then(async (response) => {
      if (!response.ok) throw new Error('Documentation manifest unavailable')
      return (await response.json()) as DocsManifest
    })
    .catch((error) => {
      manifestPromise = undefined
      manifestFetchedAt = 0
      throw error
    })

  return manifestPromise
}

async function fetchDocsPayload(fileName: string): Promise<DocsPayload> {
  const response = await fetch(`/static/docs/${DOCS_LOCALE}/${fileName}`, {
    cache: 'force-cache',
    credentials: 'same-origin',
  })
  if (!response.ok) throw new Error('Documentation payload unavailable')
  const payload = (await response.json()) as DocsPayload
  if (typeof payload.html !== 'string') {
    throw new Error('Documentation payload is invalid')
  }
  return payload
}

export function loadDocsRoute(
  route: DocsRoutePath,
  cached?: CachedDocsPage
): Promise<CachedDocsPage> {
  const key = getDocsCacheKey(route)
  const existing = inFlightDocs.get(key)
  if (existing) return existing

  const request = (async () => {
    const manifest = await getDocsManifest()
    const fileName = manifest.locales?.[DOCS_LOCALE]?.[route]
    if (!fileName) throw new Error('Documentation route missing from manifest')

    const local = cached ?? (await getCachedDocsPage(key))
    if (local?.fileName === fileName && local.html) return local

    const payload = await fetchDocsPayload(fileName)
    const page: CachedDocsPage = {
      fileName,
      html: payload.html,
      key,
      savedAt: Date.now(),
    }
    await putCachedDocsPage(page)
    return page
  })()

  inFlightDocs.set(key, request)
  void request.then(
    () => inFlightDocs.delete(key),
    () => inFlightDocs.delete(key)
  )
  return request
}

async function pruneDocsCache(manifest: DocsManifest): Promise<void> {
  const validKeys = new Set(
    Object.keys(manifest.locales?.[DOCS_LOCALE] ?? {}).map(
      (route) => `${DOCS_LOCALE}:${route}`
    )
  )
  const cachedPages = await getAllCachedDocsPages()
  await Promise.all(
    cachedPages
      .filter((page) => !validKeys.has(page.key))
      .map((page) => deleteCachedDocsPage(page.key))
  )
}

function queueBackgroundRoute(route: DocsRoutePath): void {
  if (queuedBackgroundRoutes.has(route)) return
  queuedBackgroundRoutes.add(route)
  backgroundQueue.push(route)
}

function scheduleNextBackgroundRoute(): void {
  if (
    backgroundQueueRunning ||
    backgroundQueue.length === 0 ||
    idleHandle !== undefined ||
    typeof window === 'undefined' ||
    document.visibilityState !== 'visible'
  ) {
    return
  }

  idleHandle = scheduleIdle(() => {
    idleHandle = undefined
    const route = backgroundQueue.shift()
    if (!route) return

    backgroundQueueRunning = true
    void loadDocsRoute(route)
      .catch(() => undefined)
      .finally(() => {
        queuedBackgroundRoutes.delete(route)
        backgroundQueueRunning = false
        scheduleNextBackgroundRoute()
      })
  })
}

export function startDocsBackgroundWarmup(currentRoute: DocsRoutePath): void {
  if (typeof window === 'undefined' || isSlowConnection()) return

  if (!persistenceRequested) {
    persistenceRequested = true
    void requestDocsStoragePersistence()
  }

  const routes = [
    ...getAdjacentRoutes(currentRoute),
    ...DOCS_ROUTES.map((route) => route.path),
  ].slice(0, MAX_BACKGROUND_ROUTES)
  routes.forEach(queueBackgroundRoute)
  void getDocsManifest()
    .then((manifest) => {
      const signature = JSON.stringify(manifest.locales?.[DOCS_LOCALE] ?? {})
      if (signature === lastPrunedManifestSignature) return
      lastPrunedManifestSignature = signature
      return pruneDocsCache(manifest)
    })
    .catch(() => undefined)
  scheduleNextBackgroundRoute()
}

export function preloadDocsRoute(route: DocsRoutePath): void {
  void loadDocsRoute(route).catch(() => undefined)
}

export function installDocsVisibilityResume(): () => void {
  if (typeof document === 'undefined' || visibilityListenerInstalled) {
    return () => undefined
  }
  visibilityListenerInstalled = true

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      scheduleNextBackgroundRoute()
    }
  }
  document.addEventListener('visibilitychange', handleVisibilityChange)
  return () => undefined
}
