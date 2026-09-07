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
import { useTranslation } from 'react-i18next'

import { toIntlLocale } from '@/i18n/languages'

import { formatApiKeyTokens, formatApiKeyUsageCost } from '../lib/usage-format'
import type { ApiKey } from '../types'

function ApiKeyUsageLine(props: {
  label: string
  quota: number | null | undefined
  tokens: number | null | undefined
  locale: string | undefined
}) {
  return (
    <div className='flex min-w-0 items-baseline whitespace-nowrap'>
      <span className='text-muted-foreground shrink-0'>{props.label}</span>
      <span className='min-w-0 font-medium tabular-nums'>
        {formatApiKeyUsageCost(props.quota, props.locale)}
        <span className='text-muted-foreground font-normal'>/</span>
        {formatApiKeyTokens(props.tokens, props.locale)}
      </span>
    </div>
  )
}

/** Render the daily and cumulative charge/token totals for an API key. */
export function ApiKeyUsageCell(props: { apiKey: ApiKey }) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  return (
    <div className='flex flex-col gap-0.5 text-xs leading-5'>
      <ApiKeyUsageLine
        label={t('Today:')}
        quota={props.apiKey.daily_quota}
        tokens={props.apiKey.daily_tokens}
        locale={locale}
      />
      <ApiKeyUsageLine
        label={t('Cumulative:')}
        quota={props.apiKey.total_quota}
        tokens={props.apiKey.total_tokens}
        locale={locale}
      />
    </div>
  )
}
