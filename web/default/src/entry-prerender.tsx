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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRouter,
  RouterContextProvider,
  RouterProvider,
} from '@tanstack/react-router'
import { renderToString } from 'react-dom/server'

import { DirectionProvider } from './context/direction-provider'
import { FontProvider } from './context/font-provider'
import { ThemeProvider } from './context/theme-provider'
import { DocsApiIntegration } from './features/docs/ai-model-api'
import { DocsCcSwitch } from './features/docs/cc-switch-guide'
import { DocsClaudeCode } from './features/docs/claude-code-guide'
import { DocsCodex } from './features/docs/codex-guide'
import type { DocsPayload, DocsRoutePath } from './features/docs/docs-page'
import {
  setPrerenderDocsPayload,
  type PrerenderDocsPayload,
} from './features/docs/docs-prerender-state'
import { DocsGemini } from './features/docs/gemini-guide'
import { DocsHermes } from './features/docs/hermes-guide'
import { DocsModelPricing } from './features/docs/model-pricing'
import { DocsOpenClaw } from './features/docs/openclaw-guide'
import { DocsOpenCode } from './features/docs/opencode-guide'
import { DocsOverview } from './features/docs/overview'
import { DocsPayment } from './features/docs/payment'
import i18n, { i18nReady } from './i18n/config'
import {
  initializePublicBootstrap,
  setPrerenderBootstrap,
  type PublicBootstrap,
} from './lib/public-bootstrap'
import { routeTree } from './routeTree.gen'

const PRERENDER_STATUS = {
  HeaderNavModules: '',
  logo: '__NEW_API_LOGO__',
  server_address: '__NEW_API_SERVER_ADDRESS__',
  system_name: '__NEW_API_SYSTEM_NAME__',
}

const DOCS_SOURCE_COMPONENTS: Record<DocsRoutePath, React.ComponentType> = {
  '/docs': DocsOverview,
  '/docs/api/integration': DocsApiIntegration,
  '/docs/model-pricing': DocsModelPricing,
  '/docs/payment': DocsPayment,
  '/docs/tools/cc-switch': DocsCcSwitch,
  '/docs/tools/claude-code': DocsClaudeCode,
  '/docs/tools/codex': DocsCodex,
  '/docs/tools/gemini': DocsGemini,
  '/docs/tools/hermes': DocsHermes,
  '/docs/tools/openclaw': DocsOpenClaw,
  '/docs/tools/opencode': DocsOpenCode,
}

async function initializeRender(
  pathname: string,
  locale: string,
  docsPayload?: PrerenderDocsPayload
) {
  await i18nReady
  await i18n.changeLanguage(locale)

  const bootstrap: PublicBootstrap = {
    docs: docsPayload
      ? { file_name: docsPayload.fileName, route: docsPayload.route }
      : undefined,
    home_page_content: '',
    home_page_content_loaded: pathname === '/',
    locale,
    setup: true,
    status: PRERENDER_STATUS,
  }
  setPrerenderBootstrap(bootstrap)
  setPrerenderDocsPayload(docsPayload)
  initializePublicBootstrap()

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: false,
        staleTime: 5 * 60 * 1000,
      },
    },
  })
  queryClient.setQueryData(['status'], PRERENDER_STATUS)

  const router = createRouter({
    routeTree,
    context: { queryClient },
    history: createMemoryHistory({ initialEntries: [pathname] }),
    isServer: true,
    defaultPreload: 'intent',
    defaultPreloadStaleTime: 0,
  })
  await router.load()
  initializePublicBootstrap()

  return { queryClient, router }
}

function withPrerenderProviders(
  children: React.ReactNode,
  queryClient: QueryClient
) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <FontProvider>
          <DirectionProvider>{children}</DirectionProvider>
        </FontProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

function extractDocsPage(markup: string): string {
  const marker = '<div data-doc-page-source="true">'
  const markerIndex = markup.indexOf(marker)
  if (markerIndex < 0) throw new Error('Documentation source marker is missing')

  const tagPattern = /<div\b[^>]*>|<\/div>/g
  tagPattern.lastIndex = markerIndex
  let depth = 0
  let contentStart = -1
  for (
    let match = tagPattern.exec(markup);
    match;
    match = tagPattern.exec(markup)
  ) {
    if (match[0] === '</div>') {
      depth -= 1
      if (depth === 0 && contentStart >= 0) {
        return markup.slice(contentStart, match.index)
      }
      continue
    }

    depth += 1
    if (contentStart < 0) contentStart = tagPattern.lastIndex
  }
  throw new Error('Documentation source marker is not balanced')
}

export async function prepareDocsRoute(
  pathname: DocsRoutePath,
  locale: string
): Promise<DocsPayload> {
  const source = DOCS_SOURCE_COMPONENTS[pathname]
  const render = await initializeRender(pathname, locale)
  const Source = source
  const markup = renderToString(
    withPrerenderProviders(
      <RouterContextProvider router={render.router}>
        <Source />
      </RouterContextProvider>,
      render.queryClient
    )
  )
  return { html: extractDocsPage(markup) }
}

export async function renderRoute(
  pathname: string,
  locale: string,
  docsPayload?: PrerenderDocsPayload
) {
  const render = await initializeRender(pathname, locale, docsPayload)

  return renderToString(
    withPrerenderProviders(
      <RouterProvider router={render.router} />,
      render.queryClient
    )
  )
}
