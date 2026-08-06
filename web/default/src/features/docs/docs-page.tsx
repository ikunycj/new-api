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
  useState,
  type ComponentType,
  type LazyExoticComponent,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { PublicLayout } from '@/components/layout/components/public-layout'
import { Button } from '@/components/ui/button'
import { getPublicBootstrap } from '@/lib/public-bootstrap'

import {
  getCachedDocsPage,
  putCachedDocsPage,
  type CachedDocsPage,
} from './docs-cache'
import { DOCS_LOCALE, type DocsRoutePath } from './docs-config'
import {
  getDocsRouteFromPathname,
  installDocsVisibilityResume,
  loadDocsRoute,
  preloadDocsRoute,
  startDocsBackgroundWarmup,
} from './docs-loader'
import { getPrerenderDocsPayload } from './docs-prerender-state'

export type { DocsRoutePath } from './docs-config'
export type { DocsPayload } from './docs-loader'

interface DocsPageState {
  cacheKey: string
  error: boolean
  fileName: string
  rawHtml: string
}

let lastVisibleDocsPage: DocsPageState | undefined

function DocsPendingOverlay(props: { error?: boolean; onRetry?: () => void }) {
  const { t } = useTranslation()

  return (
    <div className='bg-background/60 absolute inset-0 z-10 flex items-start justify-center pt-3 backdrop-blur-[1px]'>
      {props.error && props.onRetry ? (
        <div className='border-border bg-card/95 text-muted-foreground flex items-center gap-3 rounded-md border px-3 py-2 text-sm shadow-sm'>
          <span>{t('Failed to load')}</span>
          <Button size='sm' variant='outline' onClick={props.onRetry}>
            {t('Retry')}
          </Button>
        </div>
      ) : (
        <div className='bg-primary/70 h-0.5 w-32 animate-pulse rounded-full' />
      )}
    </div>
  )
}

function DocsLoadingSkeleton(props: { error?: boolean; onRetry?: () => void }) {
  const { t } = useTranslation()

  if (props.error && props.onRetry) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='flex min-h-svh items-center justify-center px-4 pt-16'>
          <div className='flex flex-col items-center gap-3'>
            <p className='text-muted-foreground text-sm'>
              {t('Failed to load')}
            </p>
            <Button size='sm' onClick={props.onRetry}>
              {t('Retry')}
            </Button>
          </div>
        </main>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <main
        aria-busy='true'
        aria-label={t('Loading...')}
        className='mx-auto grid w-full max-w-[1400px] grid-cols-1 gap-8 px-4 pt-24 md:grid-cols-[256px_minmax(0,760px)] md:px-6 xl:grid-cols-[256px_minmax(0,760px)_190px] xl:gap-12'
      >
        <aside className='hidden md:block'>
          <div className='border-border bg-card h-96 animate-pulse rounded-lg border p-4' />
        </aside>
        <section className='min-w-0 animate-pulse space-y-6'>
          <div className='bg-muted h-4 w-28 rounded' />
          <div className='bg-muted h-10 w-2/3 rounded' />
          <div className='space-y-3'>
            <div className='bg-muted h-4 w-full rounded' />
            <div className='bg-muted h-4 w-5/6 rounded' />
            <div className='bg-muted h-4 w-3/4 rounded' />
          </div>
          <div className='bg-muted h-56 w-full rounded-lg' />
        </section>
        <aside className='hidden xl:block'>
          <div className='border-border h-48 animate-pulse border-l pl-5' />
        </aside>
      </main>
    </PublicLayout>
  )
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
      '/docs/api/text-chat': () =>
        import('./api-text-chat').then((module) => ({
          default: module.DocsApiTextChat,
        })),
      '/docs/api/multimodal': () =>
        import('./api-multimodal').then((module) => ({
          default: module.DocsApiMultimodal,
        })),
      '/docs/api/compatibility': () =>
        import('./api-compatibility').then((module) => ({
          default: module.DocsApiCompatibility,
        })),
      '/docs/model-pricing': () =>
        import('./model-pricing').then((module) => ({
          default: module.DocsModelPricing,
        })),
      '/docs/payment': () =>
        import('./payment').then((module) => ({ default: module.DocsPayment })),
      '/docs/referral-rewards': () =>
        import('./referral-rewards').then((module) => ({
          default: module.DocsReferralRewards,
        })),
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
let lastDevelopmentComponent: LazyExoticComponent<ComponentType> | undefined

function LoadedDevelopmentDocs(props: {
  component: LazyExoticComponent<ComponentType>
}) {
  useEffect(() => {
    lastDevelopmentComponent = props.component
  }, [props.component])
  const Component = props.component
  return <Component />
}

function DevelopmentDocsPage(props: { route: DocsRoutePath }) {
  const loader = developmentDocsLoaders?.[props.route]
  if (!loader) return null

  let Component = developmentDocsComponents.get(props.route)
  if (!Component) {
    Component = lazy(loader)
    developmentDocsComponents.set(props.route, Component)
  }

  const PreviousComponent = lastDevelopmentComponent
  return (
    <Suspense
      fallback={
        PreviousComponent ? (
          <div className='relative'>
            <PreviousComponent />
            <DocsPendingOverlay />
          </div>
        ) : (
          <DocsLoadingSkeleton />
        )
      }
    >
      <LoadedDevelopmentDocs component={Component} />
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

function getInitialState(route: DocsRoutePath): DocsPageState {
  const cacheKey = `${DOCS_LOCALE}:${route}`
  const prerenderDocsPayload = getPrerenderDocsPayload()
  if (
    prerenderDocsPayload?.route === route &&
    prerenderDocsPayload.locale === DOCS_LOCALE
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

function CachedDocsPage(props: { route: DocsRoutePath }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const cacheKey = `${DOCS_LOCALE}:${props.route}`
  const [retryKey, setRetryKey] = useState(0)
  const [initialPage] = useState(() => getInitialState(props.route))
  const [page, setPage] = useState<DocsPageState>(initialPage)
  const [isLoading, setIsLoading] = useState(!initialPage.rawHtml)
  const [hasError, setHasError] = useState(false)

  useEffect(() => {
    let active = true

    const load = async () => {
      if (!initialPage.rawHtml) setIsLoading(true)
      setHasError(false)
      const initial = initialPage.rawHtml ? initialPage : undefined
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
      const initialCachedPage: CachedDocsPage | undefined = initial
        ? {
            key: cacheKey,
            fileName: initial.fileName,
            html: initial.rawHtml,
            savedAt: Date.now(),
          }
        : undefined
      const visibleCachedPage = initialCachedPage ?? cached
      if (visibleCachedPage?.html) {
        setPage({
          cacheKey,
          error: false,
          fileName: visibleCachedPage.fileName,
          rawHtml: visibleCachedPage.html,
        })
        setIsLoading(false)
      }

      try {
        const loaded = await loadDocsRoute(props.route, visibleCachedPage)
        if (!active) return
        setPage({
          cacheKey,
          error: false,
          fileName: loaded.fileName,
          rawHtml: loaded.html,
        })
        setIsLoading(false)
        setHasError(false)
        startDocsBackgroundWarmup(props.route)
      } catch {
        if (!active) return
        setIsLoading(false)
        setHasError(true)
      }
    }

    void load()
    return () => {
      active = false
    }
  }, [cacheKey, initialPage, props.route, retryKey])

  useEffect(() => {
    installDocsVisibilityResume()
    if (
      typeof window !== 'undefined' &&
      page.cacheKey === cacheKey &&
      page.rawHtml
    ) {
      lastVisibleDocsPage = page
    }
  }, [cacheKey, page])

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
      url.pathname.startsWith('/api/') ||
      url.pathname.startsWith('/v1/') ||
      url.pathname.startsWith('/static/')
    ) {
      return
    }
    const targetRoute = getDocsRouteFromPathname(url.pathname)
    if (targetRoute && targetRoute !== props.route) {
      preloadDocsRoute(targetRoute)
    }
    event.preventDefault()
    const search = Object.fromEntries(url.searchParams.entries())
    await navigate({
      to: url.pathname as never,
      ...(Object.keys(search).length > 0 ? { search: search as never } : {}),
      hash: url.hash,
    })
  }

  const handleIntent = (
    event: React.MouseEvent<HTMLDivElement> | React.FocusEvent<HTMLDivElement>
  ) => {
    const target = event.target
    if (!(target instanceof Element)) return
    const anchor = target.closest<HTMLAnchorElement>('a[href]')
    if (!anchor) return
    const route = getDocsRouteFromPathname(
      new URL(anchor.href, window.location.origin).pathname
    )
    if (route && route !== props.route) preloadDocsRoute(route)
  }

  const currentPage =
    page.cacheKey === cacheKey && page.rawHtml ? page : undefined
  const visiblePage = currentPage ?? lastVisibleDocsPage
  if (!visiblePage?.rawHtml) {
    return hasError ? (
      <DocsLoadingSkeleton
        error
        onRetry={() => setRetryKey((value) => value + 1)}
      />
    ) : (
      <DocsLoadingSkeleton />
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <div className='relative'>
        <div
          aria-busy={isLoading || hasError}
          data-doc-content-host={props.route}
          onClick={(event) => void handleClick(event)}
          onMouseOver={handleIntent}
          onFocusCapture={handleIntent}
          dangerouslySetInnerHTML={{
            __html: resolveDocsHtml(visiblePage.rawHtml),
          }}
        />
        {(isLoading || hasError) && (
          <DocsPendingOverlay
            error={hasError}
            onRetry={() => setRetryKey((value) => value + 1)}
          />
        )}
      </div>
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
