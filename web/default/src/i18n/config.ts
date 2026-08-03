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
import i18n, { type BackendModule } from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import {
  convertDetectedLanguage,
  normalizeInterfaceLanguage,
  type InterfaceLanguageCode,
} from './languages'

interface LocaleModule {
  default: {
    translation: Record<string, string>
  }
}

const fullLocaleLoaders: Record<
  InterfaceLanguageCode,
  () => Promise<LocaleModule>
> = {
  en: () => import('./locales/en.json'),
  fr: () => import('./locales/fr.json'),
  ja: () => import('./locales/ja.json'),
  ru: () => import('./locales/ru.json'),
  vi: () => import('./locales/vi.json'),
  zhCN: () => import('./locales/zh.json'),
  zhTW: () => import('./locales/zh-TW.json'),
}

const publicLocaleLoaders: Record<
  InterfaceLanguageCode,
  () => Promise<LocaleModule>
> = {
  en: () => import('./public-locales/en.json'),
  fr: () => import('./public-locales/fr.json'),
  ja: () => import('./public-locales/ja.json'),
  ru: () => import('./public-locales/ru.json'),
  vi: () => import('./public-locales/vi.json'),
  zhCN: () => import('./public-locales/zhCN.json'),
  zhTW: () => import('./public-locales/zhTW.json'),
}

const fullLocales = new Set<InterfaceLanguageCode>()

function isPublicRoute(): boolean {
  if (typeof window === 'undefined') return true
  const pathname = window.location.pathname.replace(/\/+$/, '') || '/'
  return (
    pathname === '/' || pathname === '/docs' || pathname.startsWith('/docs/')
  )
}

const localeBackend: BackendModule = {
  type: 'backend',
  init: () => undefined,
  read(language, _namespace, callback) {
    const locale = normalizeInterfaceLanguage(language) as InterfaceLanguageCode
    const usePublicLocale = isPublicRoute()
    const loader = usePublicLocale
      ? publicLocaleLoaders[locale]
      : fullLocaleLoaders[locale]
    return loader().then(
      (module) => {
        if (!usePublicLocale) fullLocales.add(locale)
        // The i18next backend contract is callback-based around an async loader.
        // eslint-disable-next-line promise/no-callback-in-promise
        callback(null, module.default.translation)
      },
      (error: unknown) => {
        // eslint-disable-next-line promise/no-callback-in-promise
        callback(
          error instanceof Error ? error : new Error(String(error)),
          false
        )
      }
    )
  },
}

export async function ensureFullLocale(language?: string): Promise<void> {
  const locale = normalizeInterfaceLanguage(
    language || i18n.resolvedLanguage || i18n.language
  ) as InterfaceLanguageCode
  if (fullLocales.has(locale)) return

  const module = await fullLocaleLoaders[locale]()
  i18n.addResourceBundle(
    locale,
    'translation',
    module.default.translation,
    true,
    true
  )
  fullLocales.add(locale)
}

export const i18nReady = i18n
  .use(LanguageDetector)
  .use(localeBackend)
  .use(initReactI18next)
  .init({
    // Translation keys are English source text and locale parity is enforced by
    // i18n:sync, so loading a second full English catalog is unnecessary.
    fallbackLng: false,
    supportedLngs: ['en', 'zhCN', 'fr', 'ru', 'ja', 'vi', 'zhTW'],
    load: 'currentOnly',
    nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // not needed for react as it escapes by default
    },
    detection: {
      order: ['localStorage', 'cookie', 'navigator'],
      caches: ['localStorage', 'cookie'],
      // Browsers report `zh-CN`/`zh-TW`/`zh`; map them onto our `zhCN`/`zhTW`
      // codes (non-Chinese codes pass through for normal supportedLngs matching).
      convertDetectedLanguage,
    },
  })

export default i18n
