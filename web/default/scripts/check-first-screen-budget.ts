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
import { readFile, readdir, stat } from 'node:fs/promises'
import path from 'node:path'
import { gzipSync } from 'node:zlib'

const projectRoot = path.resolve(import.meta.dir, '..')
const distRoot = path.join(projectRoot, 'dist')
const prerenderRoot = path.join(distRoot, 'prerender')

const limits = {
  criticalCssGzip: 80 * 1024,
  homeHtmlGzip: 50 * 1024,
  hydrationJsGzip: 450 * 1024,
  lcpImageBytes: 80 * 1024,
  maxInitialJsChunkGzip: 250 * 1024,
  publicBootTransfer: 500 * 1024,
  regularDocsHtmlGzip: 100 * 1024,
}

async function gzipFileSize(filePath: string): Promise<number> {
  return gzipSync(await readFile(filePath)).length
}

async function listFiles(directory: string): Promise<string[]> {
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(
    entries.map((entry) => {
      const entryPath = path.join(directory, entry.name)
      return entry.isDirectory() ? listFiles(entryPath) : [entryPath]
    })
  )
  return nested.flat()
}

function collectAssetUrls(html: string, pattern: RegExp): string[] {
  return [...html.matchAll(pattern)].map((match) => match[1])
}

function resolveDistAsset(url: string): string {
  if (!url.startsWith('/')) {
    throw new Error(`Expected an absolute build asset URL, received ${url}`)
  }
  return path.join(distRoot, url.slice(1))
}

function formatKiB(bytes: number): string {
  return `${(bytes / 1024).toFixed(1)} KiB`
}

const prerenderFiles = await listFiles(prerenderRoot)
const homeFiles = prerenderFiles.filter((file) => file.endsWith('home.html'))
const docsFiles = prerenderFiles.filter(
  (file) => file.endsWith('.html') && !file.endsWith('home.html')
)
if (homeFiles.length === 0 || docsFiles.length === 0) {
  throw new Error('Prerendered home or documentation HTML is missing')
}

const homeHtml = await readFile(homeFiles[0], 'utf8')
if (
  !homeHtml.includes('data-prerendered="true"') ||
  !homeHtml.includes('<h1')
) {
  throw new Error(
    'Homepage prerender output does not contain visible SSR content'
  )
}

const initialScriptUrls = collectAssetUrls(
  homeHtml,
  /<script\b[^>]*\bsrc="([^"]+)"[^>]*><\/script>/g
)
const criticalStyleUrls = collectAssetUrls(
  homeHtml,
  /<link\b[^>]*\bhref="([^"]+)"[^>]*\brel="stylesheet"[^>]*>/g
)
if (initialScriptUrls.length === 0 || criticalStyleUrls.length === 0) {
  throw new Error('Initial scripts or stylesheets could not be resolved')
}

const initialScriptSizes = await Promise.all(
  [...new Set(initialScriptUrls)].map(async (url) => ({
    gzip: await gzipFileSize(resolveDistAsset(url)),
    url,
  }))
)
const criticalCssGzip = (
  await Promise.all(
    [...new Set(criticalStyleUrls)].map((url) =>
      gzipFileSize(resolveDistAsset(url))
    )
  )
).reduce((total, size) => total + size, 0)

const localeDirectory = path.join(projectRoot, 'src/i18n/public-locales')
const localeFiles = (await readdir(localeDirectory))
  .filter((file) => file.endsWith('.json') && !file.startsWith('_'))
  .map((file) => path.join(localeDirectory, file))
const localeGzipSizes = await Promise.all(localeFiles.map(gzipFileSize))
const largestLocaleGzip = Math.max(...localeGzipSizes)

const initialJsGzip = initialScriptSizes.reduce(
  (total, asset) => total + asset.gzip,
  0
)
const hydrationJsGzip = initialJsGzip + largestLocaleGzip
const maxInitialJsChunkGzip = Math.max(
  ...initialScriptSizes.map((asset) => asset.gzip)
)

const homeHtmlGzip = Math.max(
  ...(await Promise.all(homeFiles.map(gzipFileSize)))
)
const regularDocsHtmlGzip = Math.max(
  ...(await Promise.all(docsFiles.map(gzipFileSize)))
)
const posterFiles = (
  await listFiles(path.join(distRoot, 'static/image'))
).filter((file) =>
  path.basename(file).startsWith('alltokenapi-smooth-ribbon-poster.')
)
if (posterFiles.length !== 1) {
  throw new Error('Expected exactly one emitted Hero ribbon poster')
}
const lcpImageBytes = (await stat(posterFiles[0])).size
const logoBytes = (await stat(path.join(distRoot, 'logo-56.webp'))).size
const publicBootTransfer =
  homeHtmlGzip + hydrationJsGzip + criticalCssGzip + lcpImageBytes + logoBytes

const measurements = {
  criticalCssGzip,
  homeHtmlGzip,
  hydrationJsGzip,
  lcpImageBytes,
  maxInitialJsChunkGzip,
  publicBootTransfer,
  regularDocsHtmlGzip,
}

const failures = Object.entries(measurements).filter(
  ([name, value]) => value > limits[name as keyof typeof limits]
)

console.table(
  Object.entries(measurements).map(([name, value]) => ({
    budget: formatKiB(limits[name as keyof typeof limits]),
    metric: name,
    result: value <= limits[name as keyof typeof limits] ? 'PASS' : 'FAIL',
    size: formatKiB(value),
  }))
)

if (failures.length > 0) {
  throw new Error(
    `First-screen performance budget exceeded: ${failures
      .map(
        ([name, value]) =>
          `${name}=${formatKiB(value)}>${formatKiB(
            limits[name as keyof typeof limits]
          )}`
      )
      .join(', ')}`
  )
}
