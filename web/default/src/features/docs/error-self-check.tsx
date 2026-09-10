/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { TFunction } from 'i18next'
import { Search, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { NumberedSteps } from './components/numbered-steps'

// Keep upstream error strings verbatim so users can search for them in logs.
const OPENAI_CAPACITY_ERROR =
  'Selected model is at capacity. Please try a different model.'
const OPENAI_SERVER_OVERLOADED_ERROR =
  'stream disconnected before completion: Our servers are currently overloaded. Please try again later.'
const OPENAI_PROCESSING_ERROR =
  'stream disconnected before completion: An error occurred while processing your request. You can retry your request, or contact us through our help center at help.openai.com if the error persists.'
const INVALID_API_KEY_ERROR =
  'unexpected status 401 Unauthorized: Incorrect API key provided\n\nauth error: 401, auth error code: invalid_api_key'
const UNAUTHORIZED_ERROR = 'Unauthorized'
const STREAM_DISCONNECTED_ERROR =
  'stream disconnected before completion: stream closed before response.completed'

type ErrorGuide = {
  id: string
  code?: string
  descriptionKey: string
  message: string
  causeKey: string
  solutionKeys: readonly string[]
}

type ErrorCategory = {
  id: string
  titleKey: string
  errors: readonly ErrorGuide[]
}

type FilteredErrorCategory = {
  category: ErrorCategory
  categoryIndex: number
}

const ERROR_CATEGORIES: readonly ErrorCategory[] = [
  {
    id: 'model-official-errors',
    titleKey: 'Official model errors',
    errors: [
      {
        id: 'model-capacity-insufficient',
        descriptionKey: 'Model capacity insufficient',
        message: OPENAI_CAPACITY_ERROR,
        causeKey:
          'OpenAI and other official providers temporarily lack enough compute capacity.',
        solutionKeys: [
          'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.',
          'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.',
        ],
      },
      {
        id: 'openai-server-overloaded',
        descriptionKey: 'OpenAI',
        message: OPENAI_SERVER_OVERLOADED_ERROR,
        causeKey:
          'OpenAI and other model providers are experiencing server overload.',
        solutionKeys: ['Retry'],
      },
      {
        id: 'openai-processing-error',
        descriptionKey: 'Official OpenAI error',
        message: OPENAI_PROCESSING_ERROR,
        causeKey:
          'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.',
        solutionKeys: ['Retry'],
      },
    ],
  },
  {
    id: 'user-client-errors',
    titleKey: 'User client errors',
    errors: [
      {
        id: 'client-authentication-failed',
        code: '401',
        descriptionKey: 'Authentication failed',
        message: UNAUTHORIZED_ERROR,
        causeKey:
          'The API key or authentication header is missing, expired, or invalid.',
        solutionKeys: [
          'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.',
          'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.',
          'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.',
        ],
      },
      {
        id: 'client-invalid-api-key',
        code: '401',
        descriptionKey: 'Invalid API key',
        message: INVALID_API_KEY_ERROR,
        causeKey:
          'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.',
        solutionKeys: [
          'Re-import through CC Switch (simplest).',
          'Confirm that the API key and Base URL match.',
        ],
      },
    ],
  },
  {
    id: 'relay-errors',
    titleKey: 'Relay errors',
    errors: [
      {
        id: 'relay-stream-too-many-pending-requests',
        descriptionKey:
          'Streaming output interrupted due to too many pending requests',
        message:
          'stream disconnected before completion: Too many pending requests, please retry later',
        causeKey:
          'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.',
        solutionKeys: [
          'Retry the request several times or avoid peak hours.',
          'If the problem persists, click About above and contact the administrator.',
        ],
      },
    ],
  },
  {
    id: 'other-errors',
    titleKey: 'Other errors',
    errors: [
      {
        id: 'relay-stream-interrupted',
        descriptionKey: 'Response stream interrupted',
        message: STREAM_DISCONNECTED_ERROR,
        causeKey:
          'The upstream connection or relay path closed before the response completed.',
        solutionKeys: [
          'Retry the request. A transient upstream disconnect can succeed on a later attempt.',
          'If the error persists, try another available model or group and check the channel health and usage logs.',
        ],
      },
    ],
  },
]

type ErrorGuideCardProps = {
  error: ErrorGuide
  t: TFunction
}

function ErrorGuideCard(props: ErrorGuideCardProps) {
  return (
    <AccordionItem
      id={props.error.id}
      value={props.error.id}
      className='border-border bg-card scroll-mt-28 overflow-hidden rounded-lg border px-4 sm:px-5'
    >
      <AccordionTrigger className='min-h-14 gap-3 py-4 hover:no-underline'>
        <span className='flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1 pr-3'>
          {props.error.code ? (
            <code className='text-primary shrink-0 font-mono text-sm font-semibold'>
              {props.error.code}
            </code>
          ) : null}
          <span className='text-foreground min-w-0 text-sm leading-6 font-semibold break-words'>
            {props.t(props.error.descriptionKey)}
          </span>
        </span>
      </AccordionTrigger>

      <AccordionContent className='pb-1'>
        <div className='border-border/70 space-y-5 border-t pt-4'>
          <dl className='grid gap-4 sm:grid-cols-2'>
            {props.error.code ? (
              <div className='min-w-0'>
                <dt className='text-muted-foreground text-xs leading-5 font-semibold'>
                  {props.t('Error code')}
                </dt>
                <dd className='text-foreground mt-1 font-mono text-sm leading-6'>
                  {props.error.code}
                </dd>
              </div>
            ) : null}
            <div className='min-w-0 sm:col-span-2'>
              <dt className='text-muted-foreground text-xs leading-5 font-semibold'>
                {props.t('Error description')}
              </dt>
              <dd className='text-foreground mt-1 text-sm leading-6'>
                {props.t(props.error.descriptionKey)}
              </dd>
            </div>
          </dl>

          <div className='space-y-4'>
            <div>
              <p className='text-muted-foreground text-xs leading-5 font-semibold'>
                {props.t('Error information')}
              </p>
              <p className='bg-muted/50 text-foreground mt-1.5 rounded-lg px-3 py-2.5 font-mono text-xs leading-6 break-words whitespace-pre-wrap'>
                {props.error.message}
              </p>
            </div>

            <div>
              <p className='text-muted-foreground text-xs leading-5 font-semibold'>
                {props.t('Cause')}
              </p>
              <p className='text-foreground mt-1 text-sm leading-7'>
                {props.t(props.error.causeKey)}
              </p>
            </div>

            <div>
              <p className='text-muted-foreground text-xs leading-5 font-semibold'>
                {props.t('Solution')}
              </p>
              <NumberedSteps
                items={props.error.solutionKeys.map((key) => props.t(key))}
              />
            </div>
          </div>
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

export function DocsErrorSelfCheck() {
  const { t } = useTranslation()
  const [searchQuery, setSearchQuery] = useState('')
  const filteredCategories = useMemo<readonly FilteredErrorCategory[]>(() => {
    const query = searchQuery.trim().normalize('NFKC').toLowerCase()
    if (!query) {
      return ERROR_CATEGORIES.map((category, categoryIndex) => ({
        category,
        categoryIndex,
      }))
    }

    const queryTerms = query.split(/\s+/).filter(Boolean)

    return ERROR_CATEGORIES.flatMap((category, categoryIndex) => {
      const errors = category.errors.filter((error) => {
        const searchableText = [
          category.titleKey,
          t(category.titleKey),
          error.code,
          error.descriptionKey,
          t(error.descriptionKey),
          error.message,
          error.causeKey,
          t(error.causeKey),
          ...error.solutionKeys,
          ...error.solutionKeys.map((key) => t(key)),
        ]
          .filter(Boolean)
          .join(' ')
          .normalize('NFKC')
          .toLowerCase()

        return queryTerms.every((term) => searchableText.includes(term))
      })

      return errors.length > 0
        ? [{ category: { ...category, errors }, categoryIndex }]
        : []
    })
  }, [searchQuery, t])

  const toc: DocsTocItem[] = filteredCategories.flatMap(
    ({ category, categoryIndex }) => [
      {
        id: category.id,
        label: `${categoryIndex + 1}. ${t(category.titleKey)}`,
        level: 1 as const,
      },
      ...category.errors.map((error, errorIndex) => ({
        id: error.id,
        label: `${categoryIndex + 1}.${errorIndex + 1} ${error.code ? `${error.code} ` : ''}${t(error.descriptionKey)}`,
        level: 2 as const,
      })),
    ]
  )

  return (
    <DocsShell
      pageId='error-self-check'
      title={t('Error self-check guide')}
      toc={toc}
    >
      <div className='flex flex-col gap-8'>
        <div
          role='search'
          aria-label={t('Search error keywords')}
          className='relative max-w-2xl'
        >
          <Search
            className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2'
            aria-hidden='true'
          />
          <Input
            type='text'
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder={t('Search error keywords...')}
            aria-label={t('Search error keywords')}
            className='h-10 pr-10 pl-9'
          />
          {searchQuery ? (
            <Button
              type='button'
              variant='ghost'
              size='icon-sm'
              onClick={() => setSearchQuery('')}
              aria-label={t('Clear search')}
              className='text-muted-foreground hover:text-foreground absolute top-1/2 right-1 -translate-y-1/2'
            >
              <X className='size-4' aria-hidden='true' />
            </Button>
          ) : null}
        </div>

        {filteredCategories.length === 0 ? (
          <p
            role='status'
            className='border-border bg-muted/20 text-muted-foreground rounded-lg border px-4 py-8 text-center text-sm'
          >
            {t('No matching errors found.')}
          </p>
        ) : (
          filteredCategories.map(({ category, categoryIndex }) => (
            <section
              id={category.id}
              key={category.id}
              className='scroll-mt-28'
            >
              <h2 className='text-2xl leading-8 font-semibold tracking-tight'>
                {categoryIndex + 1}. {t(category.titleKey)}
              </h2>
              <Accordion multiple className='mt-4 flex flex-col gap-3'>
                {category.errors.map((error) => (
                  <ErrorGuideCard key={error.id} error={error} t={t} />
                ))}
              </Accordion>
            </section>
          ))
        )}
      </div>
    </DocsShell>
  )
}
