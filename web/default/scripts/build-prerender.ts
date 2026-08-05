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
import { createHash } from 'node:crypto'
import {
  mkdtemp,
  mkdir,
  readFile,
  readdir,
  rm,
  writeFile,
} from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'

import { DOCS_LOCALE, DOCS_ROUTES } from '../src/features/docs/docs-config'

const LANGUAGES = ['en', 'zhCN', 'zhTW', 'fr', 'ja', 'ru', 'vi'] as const
const ROUTES = [
  { file: 'home.html', path: '/' },
  ...DOCS_ROUTES.map((route) => ({
    docId: route.id,
    file: route.file,
    path: route.path,
  })),
] as const

interface DocsPayload {
  html: string
}

interface PrerenderDocsPayload extends DocsPayload {
  fileName: string
  locale: string
  route: string
}

const projectRoot = path.resolve(import.meta.dir, '..')
const distRoot = path.join(projectRoot, 'dist')
const tempRoot = await mkdtemp(path.join(tmpdir(), 'new-api-prerender-'))

const memoryStorage = new Map<string, string>()
globalThis.localStorage = {
  clear: () => memoryStorage.clear(),
  getItem: (key) => memoryStorage.get(key) ?? null,
  key: (index) => [...memoryStorage.keys()][index] ?? null,
  get length() {
    return memoryStorage.size
  },
  removeItem: (key) => memoryStorage.delete(key),
  setItem: (key, value) => memoryStorage.set(key, value),
}

try {
  const build = await Bun.build({
    entrypoints: [path.join(projectRoot, 'src/entry-prerender.tsx')],
    outdir: tempRoot,
    target: 'bun',
    format: 'esm',
    minify: false,
    define: {
      'import.meta.env.DEV': 'false',
      'import.meta.env.MODE': '"production"',
      'import.meta.env.PROD': 'true',
      'import.meta.env.SSR': 'true',
    },
  })
  if (!build.success) {
    throw new Error(build.logs.map((log) => log.message).join('\n'))
  }

  const entry = build.outputs.find((output) => output.kind === 'entry-point')
  if (!entry) throw new Error('Prerender entry bundle was not generated')

  const renderer = (await import(
    `${pathToFileURL(entry.path).href}?v=${Date.now()}`
  )) as {
    prepareDocsRoute: (pathname: string, locale: string) => Promise<DocsPayload>
    renderRoute: (
      pathname: string,
      locale: string,
      docsPayload?: PrerenderDocsPayload
    ) => Promise<string>
  }
  const indexTemplate = await readFile(
    path.join(distRoot, 'index.html'),
    'utf8'
  )
  const imageFiles = await readdir(path.join(distRoot, 'static/image'))
  const posterFile = imageFiles.find((file) =>
    file.startsWith('alltokenapi-smooth-ribbon-poster.')
  )
  if (!posterFile) throw new Error('Hero ribbon poster was not emitted')
  const posterUrl = `/static/image/${posterFile}`

  await rm(path.join(distRoot, 'prerender'), { recursive: true, force: true })
  await rm(path.join(distRoot, 'static/docs'), {
    recursive: true,
    force: true,
  })
  const manifest: {
    locales: Record<string, Record<string, string>>
    version: number
  } = { locales: { [DOCS_LOCALE]: {} }, version: 1 }

  for (const language of LANGUAGES) {
    for (const route of ROUTES) {
      const isDocsRoute = 'docId' in route
      if (isDocsRoute && language !== DOCS_LOCALE) continue

      let docsPayload: PrerenderDocsPayload | undefined
      if (isDocsRoute) {
        const payload = await renderer.prepareDocsRoute(route.path, language)
        const serializedPayload = JSON.stringify(payload)
        const hash = createHash('sha256')
          .update(serializedPayload)
          .digest('hex')
          .slice(0, 16)
        const fileName = `${route.docId}.${hash}.json`
        docsPayload = {
          ...payload,
          fileName,
          locale: language,
          route: route.path,
        }
        manifest.locales[DOCS_LOCALE][route.path] = fileName

        const payloadPath = path.join(
          distRoot,
          'static/docs',
          language,
          fileName
        )
        await mkdir(path.dirname(payloadPath), { recursive: true })
        await writeFile(payloadPath, serializedPayload)
      }

      const markup = (
        await renderer.renderRoute(route.path, language, docsPayload)
      ).replaceAll('__NEW_API_RIBBON_POSTER__', posterUrl)
      const html = indexTemplate
        .replace('<html lang="en">', `<html lang="${language}">`)
        .replace(
          '<div id="root" translate="no" class="notranslate"></div>',
          `<div id="root" translate="no" class="notranslate" data-prerendered="true">${markup}</div>`
        )
      const outputPath = path.join(distRoot, 'prerender', language, route.file)
      await mkdir(path.dirname(outputPath), { recursive: true })
      await writeFile(outputPath, html)
    }
  }

  await writeFile(
    path.join(distRoot, 'static/docs/manifest.json'),
    JSON.stringify(manifest)
  )
} finally {
  await rm(tempRoot, { recursive: true, force: true })
}
