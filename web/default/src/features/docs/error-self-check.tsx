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
import { Alert02Icon, ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { DocsShell, type DocsTocItem } from './components/docs-shell'

const STREAM_DISCONNECTED_ERROR =
  'stream disconnected before completion: stream closed before response.completed'

export function DocsErrorSelfCheck() {
  const { t } = useTranslation()
  const toc: DocsTocItem[] = [
    { id: 'error-directory', label: `1. ${t('Common error directory')}`, level: 1 },
    { id: 'common-errors', label: `2. ${t('Common errors')}`, level: 1 },
    { id: 'openai-errors', label: `2.1 ${t('Official OpenAI errors')}`, level: 2 },
    { id: 'user-errors', label: `2.2 ${t('User-side errors')}`, level: 2 },
    { id: 'relay-errors', label: `2.3 ${t('Relay errors')}`, level: 2 },
    { id: 'other-errors', label: `2.4 ${t('Other errors')}`, level: 2 },
  ]

  return (
    <DocsShell
      pageId='error-self-check'
      title={t('Error self-check guide')}
      toc={toc}
    >
      <section id='error-directory' className='scroll-mt-28'>
        <h2 className='border-border border-b pb-3 text-2xl leading-8 font-semibold tracking-tight'>
          1. {t('Common error directory')}
        </h2>
        <nav
          aria-label={t('Common error directory')}
          className='border-border bg-muted/20 mt-5 overflow-hidden rounded-lg border'
        >
          <div className='border-border/70 bg-muted/40 flex items-center gap-2 border-b px-4 py-2.5 text-sm font-semibold'>
            <HugeiconsIcon
              icon={Alert02Icon}
              className='text-primary size-4'
              aria-hidden='true'
            />
            <span>{t('Error index')}</span>
          </div>
          <ol className='divide-border divide-y'>
            <li>
              <a
                href='#relay-stream-disconnected'
                className='group focus-visible:ring-ring hover:bg-muted/50 flex min-w-0 items-start gap-3 px-4 py-3.5 transition-colors focus-visible:ring-2 focus-visible:outline-none focus-visible:ring-inset'
              >
                <span className='bg-primary/10 text-primary mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md font-mono text-xs font-semibold'>
                  01
                </span>
                <span className='min-w-0 flex-1'>
                  <span className='text-muted-foreground block text-xs leading-5'>
                    2.3.1 · {t('Relay errors')}
                  </span>
                  <code className='text-foreground mt-0.5 block font-mono text-sm leading-6 break-words'>
                    {STREAM_DISCONNECTED_ERROR}
                  </code>
                </span>
                <HugeiconsIcon
                  icon={ArrowRight01Icon}
                  className='text-muted-foreground mt-1 size-4 shrink-0 transition-transform group-hover:translate-x-0.5'
                  aria-hidden='true'
                />
              </a>
            </li>
          </ol>
        </nav>
      </section>

      <section id='common-errors' className='scroll-mt-28'>
        <h2 className='border-border border-b pb-3 text-2xl leading-8 font-semibold tracking-tight'>
          2. {t('Common errors')}
        </h2>

        <div className='mt-8 flex flex-col gap-8'>
          <section
            id='openai-errors'
            className='border-border/70 scroll-mt-28 border-t pt-6 first:border-t-0 first:pt-0'
          >
            <h3 className='border-primary/50 border-l-2 pl-3 text-lg leading-7 font-semibold tracking-tight'>
              2.1 {t('Official OpenAI errors')}
            </h3>
          </section>

          <section
            id='user-errors'
            className='border-border/70 scroll-mt-28 border-t pt-6 first:border-t-0 first:pt-0'
          >
            <h3 className='border-primary/50 border-l-2 pl-3 text-lg leading-7 font-semibold tracking-tight'>
              2.2 {t('User-side errors')}
            </h3>
          </section>

          <section
            id='relay-errors'
            className='border-border/70 scroll-mt-28 border-t pt-6 first:border-t-0 first:pt-0'
          >
            <h3 className='border-primary/50 border-l-2 pl-3 text-lg leading-7 font-semibold tracking-tight'>
              2.3 {t('Relay errors')}
            </h3>
            <section
              id='relay-stream-disconnected'
              className='border-primary/30 bg-muted/20 mt-5 scroll-mt-28 rounded-r-lg border-l-2 px-4 py-4 sm:px-5'
            >
              <h4 className='text-base leading-6 font-semibold tracking-tight break-words'>
                2.3.1 {STREAM_DISCONNECTED_ERROR}
              </h4>
              <p className='border-border/70 text-muted-foreground mt-3 border-t pt-3 text-sm leading-6'>
                {t('The response stream was interrupted unexpectedly.')}
              </p>
            </section>
          </section>

          <section
            id='other-errors'
            className='border-border/70 scroll-mt-28 border-t pt-6 first:border-t-0 first:pt-0'
          >
            <h3 className='border-primary/50 border-l-2 pl-3 text-lg leading-7 font-semibold tracking-tight'>
              2.4 {t('Other errors')}
            </h3>
          </section>
        </div>
      </section>
    </DocsShell>
  )
}
