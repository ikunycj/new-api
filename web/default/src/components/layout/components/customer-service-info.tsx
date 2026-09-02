/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'

type CustomerServiceInfoProps = {
  compact?: boolean
  inline?: boolean
}

export function CustomerServiceInfo(props: CustomerServiceInfoProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const content =
    typeof status?.customer_service === 'string'
      ? status.customer_service.trim()
      : ''

  if (!content) return null

  if (props.inline) {
    return (
      <p
        aria-label={t('Customer Service')}
        className='text-muted-foreground w-full min-w-0 text-left text-sm leading-5 break-words whitespace-pre-wrap sm:w-auto sm:flex-1'
      >
        {content}
      </p>
    )
  }

  if (props.compact) {
    return (
      <section aria-label={t('Customer Service')} className='mt-1 pt-3'>
        <p className='text-muted-foreground text-sm leading-5 break-words whitespace-pre-wrap'>
          {content}
        </p>
      </section>
    )
  }

  return (
    <section
      aria-label={t('Customer Service')}
      className='border-border/60 border-t'
    >
      <div className='container px-4 py-5 md:px-6'>
        <p className='text-muted-foreground max-w-3xl text-sm leading-6 whitespace-pre-wrap'>
          {content}
        </p>
      </div>
    </section>
  )
}
