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
/* eslint-disable react/no-danger -- Documentation HTML is generated from repository-owned TSX during the build. */
import { useNavigate } from '@tanstack/react-router'
import {
  lazy,
  Suspense,
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type LazyExoticComponent,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { PublicLayout } from '@/components/layout/components/public-layout'
import { Button } from '@/components/ui/button'
import { normalizeInterfaceLanguage } from '@/i18n/languages'
import { getPublicBootstrap } from '@/lib/public-bootstrap'

import {
  getCachedDocsPage,
  putCachedDocsPage,
  type CachedDocsPage,
} from './docs-cache'
import { getPrerenderDocsPayload } from './docs-prerender-state'

export type DocsRoutePath =
  | '/docs'
  | '/docs/payment'
  | '/docs/model-pricing'
  | '/docs/tools/cc-switch'
  | '/docs/tools/codex'
  | '/docs/tools/claude-code'
  | '/docs/tools/openclaw'
  | '/docs/tools/hermes'
  | '/docs/tools/opencode'
  | '/docs/tools/gemini'
  | '/docs/api/integration'

export interface DocsPayload {
  html: string
}

interface DocsManifest {
  locales: Record<string, Partial<Record<DocsRoutePath, string>>>
  version: number
}

interface DocsPageState {
  cacheKey: string
  error: boolean
  fileName: string
  rawHtml: string
}

const developmentDocsLoaders = import.meta.env.DEV
  ? {
      '/docs': () =>
        import('./overview').then((module) => ({
          default: module.DocsOverview,
        })),
      '/docs/api/integration': () =>
        import('./ai-model-api').then((module) => ({
          default: module.DocsApiIntegration,
        })),
      '/docs/model-pricing': () =>
        import('./model-pricing').then((module) => ({
          default: module.DocsModelPricing,
        })),
      '/docs/payment': () =>
        import('./payment').then((module) => ({ default: module.DocsPayment })),
      '/docs/tools/cc-switch': () =>
        import('./cc-switch-guide').then((module) => ({
          default: module.DocsCcSwitch,
        })),
      '/docs/tools/claude-code': () =>
        import('./claude-code-guide').then((module) => ({
          default: module.DocsClaudeCode,
        })),
      '/docs/tools/codex': () =>
        import('./codex-guide').then((module) => ({
          default: module.DocsCodex,
        })),
      '/docs/tools/gemini': () =>
        import('./gemini-guide').then((module) => ({
          default: module.DocsGemini,
        })),
      '/docs/tools/hermes': () =>
        import('./hermes-guide').then((module) => ({
          default: module.DocsHermes,
        })),
      '/docs/tools/openclaw': () =>
        import('./openclaw-guide').then((module) => ({
          default: module.DocsOpenClaw,
        })),
      '/docs/tools/opencode': () =>
        import('./opencode-guide').then((module) => ({
          default: module.DocsOpenCode,
        })),
    }
  : undefined

const developmentDocsComponents = new Map<
  DocsRoutePath,
  LazyExoticComponent<ComponentType>
>()

function DevelopmentDocsPage(props: { route: DocsRoutePath }) {
  const loader = developmentDocsLoaders?.[props.route]
  if (!loader) return null

  let Component = developmentDocsComponents.get(props.route)
  if (!Component) {
    Component = lazy(loader)
    developmentDocsComponents.set(props.route, Component)
  }
  return (
    <Suspense fallback={null}>
      <Component />
    </Suspense>
  )
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function getServerAddress(): string {
  const configured = getPublicBootstrap()?.status.server_address
  if (typeof configured === 'string' && configured.trim()) {
    return configured.trim().replace(/\/+$/, '')
  }
  return typeof window === 'undefined' ? '' : window.location.origin
}

function resolveDocsHtml(rawHtml: string): string {
  return rawHtml.replaceAll(
    '__NEW_API_SERVER_ADDRESS__',
    escapeHtml(getServerAddress())
  )
}

function restoreServerAddressMarker(html: string): string {
  const serverAddress = getServerAddress()
  return serverAddress
    ? html.replaceAll(escapeHtml(serverAddress), '__NEW_API_SERVER_ADDRESS__')
    : html
}

function getInitialState(route: DocsRoutePath, locale: string): DocsPageState {
  const cacheKey = `${locale}:${route}`
  const prerenderDocsPayload = getPrerenderDocsPayload()
  if (
    prerenderDocsPayload?.route === route &&
    prerenderDocsPayload.locale === locale
  ) {
    return {
      cacheKey,
      error: false,
      fileName: prerenderDocsPayload.fileName,
      rawHtml: prerenderDocsPayload.html,
    }
  }

  if (typeof document !== 'undefined') {
    const bootstrap = getPublicBootstrap()?.docs
    const host = document.querySelector<HTMLElement>('[data-doc-content-host]')
    if (bootstrap?.route === route && host?.dataset.docContentHost === route) {
      return {
        cacheKey,
        error: false,
        fileName: bootstrap.file_name,
        rawHtml: restoreServerAddressMarker(host.innerHTML),
      }
    }
  }

  return { cacheKey, error: false, fileName: '', rawHtml: '' }
}

async function getManifestFileName(
  locale: string,
  route: DocsRoutePath
): Promise<string> {
  const response = await fetch('/static/docs/manifest.json', {
    cache: 'no-cache',
    credentials: 'same-origin',
  })
  if (!response.ok) throw new Error('Documentation manifest unavailable')
  const manifest = (await response.json()) as DocsManifest
  const fileName = manifest.locales?.[locale]?.[route]
  if (!fileName) throw new Error('Documentation route missing from manifest')
  return fileName
}

async function fetchDocsPayload(
  locale: string,
  fileName: string
): Promise<DocsPayload> {
  const response = await fetch(`/static/docs/${locale}/${fileName}`, {
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

function CachedDocsPage(props: { route: DocsRoutePath }) {
  const { i18n, t } = useTranslation()
  const navigate = useNavigate()
  const locale = normalizeInterfaceLanguage(
    i18n.resolvedLanguage || i18n.language
  )
  const cacheKey = `${locale}:${props.route}`
  const [retryKey, setRetryKey] = useState(0)
  const [page, setPage] = useState<DocsPageState>(() =>
    getInitialState(props.route, locale)
  )
  const pageRef = useRef(page)
  pageRef.current = page

  useEffect(() => {
    let active = true
    let idleId: number | undefined

    const refresh = async (cached: CachedDocsPage | undefined) => {
      try {
        const currentFileName = await getManifestFileName(locale, props.route)
        if (!active) return

        const visible = pageRef.current
        if (
          cached?.fileName === currentFileName ||
          (visible.cacheKey === cacheKey &&
            visible.fileName === currentFileName &&
            visible.rawHtml)
        ) {
          return
        }

        const payload = await fetchDocsPayload(locale, currentFileName)
        if (!active) return
        const nextPage: CachedDocsPage = {
          fileName: currentFileName,
          html: payload.html,
          key: cacheKey,
          savedAt: Date.now(),
        }
        setPage({
          cacheKey,
          error: false,
          fileName: nextPage.fileName,
          rawHtml: nextPage.html,
        })
        await putCachedDocsPage(nextPage)
      } catch {
        if (active && !pageRef.current.rawHtml) {
          setPage({ cacheKey, error: true, fileName: '', rawHtml: '' })
        }
      }
    }

    const load = async () => {
      const initial =
        pageRef.current.cacheKey === cacheKey ? pageRef.current : undefined
      if (initial?.rawHtml && initial.fileName) {
        await putCachedDocsPage({
          fileName: initial.fileName,
          html: initial.rawHtml,
          key: cacheKey,
          savedAt: Date.now(),
        })
      }

      const cached = await getCachedDocsPage(cacheKey)
      if (!active) return
      if (!initial?.rawHtml && cached) {
        setPage({
          cacheKey,
          error: false,
          fileName: cached.fileName,
          rawHtml: cached.html,
        })
      }

      if (initial?.rawHtml || cached?.html) {
        const idleWindow = window as Window & {
          requestIdleCallback?: (
            callback: IdleRequestCallback,
            options?: IdleRequestOptions
          ) => number
        }
        if (idleWindow.requestIdleCallback) {
          idleId = idleWindow.requestIdleCallback(
            () => {
              void refresh(cached)
            },
            { timeout: 2000 }
          )
        } else {
          idleId = window.setTimeout(() => {
            void refresh(cached)
          }, 500)
        }
        return
      }

      await refresh(cached)
    }

    void load()
    return () => {
      active = false
      if (idleId === undefined) return
      const idleWindow = window as Window & {
        cancelIdleCallback?: (handle: number) => void
      }
      if (idleWindow.cancelIdleCallback) {
        idleWindow.cancelIdleCallback(idleId)
      } else {
        window.clearTimeout(idleId)
      }
    }
  }, [cacheKey, locale, props.route, retryKey])

  const handleClick = async (event: React.MouseEvent<HTMLDivElement>) => {
    const target = event.target
    if (!(target instanceof Element)) return

    const copyButton = target.closest<HTMLElement>('[data-doc-copy="true"]')
    if (copyButton) {
      event.preventDefault()
      const code = copyButton
        .closest('[data-doc-code-block="true"]')
        ?.querySelector('pre code')?.textContent
      try {
        await navigator.clipboard.writeText(code ?? '')
        toast.success(t('Code copied'))
      } catch {
        toast.error(t('Could not copy code'))
      }
      return
    }

    const anchor = target.closest<HTMLAnchorElement>('a[href]')
    if (anchor?.getAttribute('href')?.startsWith('#')) return
    if (
      !anchor ||
      anchor.target ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return
    }
    const url = new URL(anchor.href, window.location.origin)
    if (
      url.origin !== window.location.origin ||
      !url.pathname.startsWith('/docs')
    ) {
      return
    }
    event.preventDefault()
    await navigate({ to: url.pathname as DocsRoutePath, hash: url.hash })
  }

  const handleChange = (event: React.FormEvent<HTMLDivElement>) => {
    const target = event.target
    if (
      !(target instanceof HTMLSelectElement) ||
      target.dataset.docNavigation !== 'true'
    ) {
      return
    }
    void navigate({ to: target.value as DocsRoutePath })
  }

  const visiblePage = page.cacheKey === cacheKey ? page : undefined
  if (!visiblePage?.rawHtml) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='flex min-h-svh items-center justify-center pt-16'>
          {visiblePage?.error ? (
            <div className='flex flex-col items-center gap-3'>
              <p className='text-muted-foreground text-sm'>
                {t('Failed to load')}
              </p>
              <Button
                size='sm'
                onClick={() => setRetryKey((value) => value + 1)}
              >
                {t('Retry')}
              </Button>
            </div>
          ) : (
            <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
          )}
        </main>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <div
        data-doc-content-host={props.route}
        onClick={(event) => void handleClick(event)}
        onChange={handleChange}
        dangerouslySetInnerHTML={{
          __html: resolveDocsHtml(visiblePage.rawHtml),
        }}
      />
    </PublicLayout>
  )
}

export function DocsPage(props: { route: DocsRoutePath }) {
  return import.meta.env.DEV ? (
    <DevelopmentDocsPage route={props.route} />
  ) : (
    <CachedDocsPage route={props.route} />
  )
}
