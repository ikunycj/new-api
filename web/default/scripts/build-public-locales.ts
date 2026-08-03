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
import { mkdir, readFile, readdir, stat, writeFile } from 'node:fs/promises'
import path from 'node:path'

const projectRoot = path.resolve(import.meta.dir, '..')
const sourceRoot = path.join(projectRoot, 'src')
const localeRoot = path.join(sourceRoot, 'i18n/locales')
const outputRoot = path.join(sourceRoot, 'i18n/public-locales')

const sourceEntries = [
  'components',
  'context',
  'features/auth',
  'features/docs',
  'features/errors',
  'features/home',
  'hooks',
  'lib',
  'routes/__root.tsx',
  'routes/docs',
  'routes/index.tsx',
  'stores',
]

const locales = {
  en: 'en.json',
  fr: 'fr.json',
  ja: 'ja.json',
  ru: 'ru.json',
  vi: 'vi.json',
  zhCN: 'zh.json',
  zhTW: 'zh-TW.json',
} as const

interface LocaleFile {
  translation: Record<string, string>
}

const dynamicPublicKeys = new Set([
  ['footer', 'new' + 'api', 'projectAttributionSuffix'].join('.'),
])

async function collectSourceFiles(input: string): Promise<string[]> {
  const inputStat = await stat(input)
  if (!inputStat.isDirectory()) {
    return /\.(?:ts|tsx)$/.test(input) ? [input] : []
  }

  const entries = await readdir(input, { withFileTypes: true })
  const nested = await Promise.all(
    entries.map((entry) => collectSourceFiles(path.join(input, entry.name)))
  )
  return nested.flat()
}

function sourceContainsKey(source: string, key: string): boolean {
  const singleQuoted = `'${key
    .replaceAll('\\', '\\\\')
    .replaceAll("'", "\\'")}'`
  const templateQuoted = `\`${key
    .replaceAll('\\', '\\\\')
    .replaceAll('`', '\\`')}\``
  return (
    source.includes(JSON.stringify(key)) ||
    source.includes(singleQuoted) ||
    source.includes(templateQuoted)
  )
}

const sourceFiles = (
  await Promise.all(
    sourceEntries.map((entry) =>
      collectSourceFiles(path.join(sourceRoot, entry))
    )
  )
).flat()
const publicSource = (
  await Promise.all(sourceFiles.map((file) => readFile(file, 'utf8')))
).join('\n')

const parsedLocales = new Map<string, LocaleFile>()
for (const [locale, fileName] of Object.entries(locales)) {
  const parsed = JSON.parse(
    await readFile(path.join(localeRoot, fileName), 'utf8')
  ) as LocaleFile
  parsedLocales.set(locale, parsed)
}

const english = parsedLocales.get('en')
if (!english) throw new Error('English locale is missing')
const publicKeys = Object.keys(english.translation).filter((key) =>
  dynamicPublicKeys.has(key) || sourceContainsKey(publicSource, key)
)
if (publicKeys.length === 0) {
  throw new Error('No public translation keys were discovered')
}

await mkdir(outputRoot, { recursive: true })
for (const [locale, parsed] of parsedLocales) {
  const missingKeys = publicKeys.filter(
    (key) => typeof parsed.translation[key] !== 'string'
  )
  if (missingKeys.length > 0) {
    throw new Error(
      `${locale} is missing public translations: ${missingKeys.join(', ')}`
    )
  }

  const translation = Object.fromEntries(
    publicKeys.map((key) => [key, parsed.translation[key]])
  )
  await writeFile(
    path.join(outputRoot, `${locale}.json`),
    `${JSON.stringify({ translation }, null, 2)}\n`
  )
}

console.log(
  `Generated ${publicKeys.length} public translation keys for ${parsedLocales.size} locales.`
)
