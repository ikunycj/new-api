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

const localeBackend: BackendModule = {
  type: 'backend',
  init: () => undefined,
  read(language, _namespace, callback) {
    const locale = normalizeInterfaceLanguage(language) as InterfaceLanguageCode
    return fullLocaleLoaders[locale]().then(
      (module) => {
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
