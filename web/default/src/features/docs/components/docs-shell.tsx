import {
  ArrowDown01Icon,
  ArrowLeft01Icon,
  ArrowRight01Icon,
  BookOpen01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
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

import { PublicLayout } from '@/components/layout/components/public-layout'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

import {
  DOCS_NAVIGATION_GROUPS,
  type DocsNavigationGroup,
  type DocsPageId,
} from '../docs-config'

export type DocsTocItem = {
  id: string
  label: string
}

type DocsShellProps = {
  pageId: DocsPageId
  title: string
  description: string
  toc: DocsTocItem[]
  children: React.ReactNode
}

function DocsNavigation(props: {
  currentPageId: DocsPageId
  groups: DocsNavigationGroup[]
  ariaLabel: string
}) {
  return (
    <nav aria-label={props.ariaLabel} className='flex flex-col'>
      <Link
        to='/docs'
        activeOptions={{ exact: true }}
        aria-current={
          props.currentPageId === 'introduction' ? 'page' : undefined
        }
        className={cn(
          'rounded-md border px-3 py-2 text-sm font-semibold transition-colors',
          props.currentPageId === 'introduction'
            ? 'border-border bg-gradient-to-br from-background to-muted/80 text-foreground'
            : 'border-transparent text-foreground hover:bg-muted/50'
        )}
      >
        {props.ariaLabel}
      </Link>

      {props.groups.map((group) => {
        const contentId = `docs-group-${group.id}`

        return (
          <details key={group.id} open className='group/docs-section mt-1.5'>
            <summary
              aria-controls={contentId}
              className='text-foreground hover:bg-muted/50 focus-visible:ring-ring/50 flex min-h-10 cursor-pointer list-none items-center rounded-md px-3 text-left text-sm font-semibold transition-colors outline-none focus-visible:ring-3 [&::-webkit-details-marker]:hidden'
            >
              <span>{group.label}</span>
              <HugeiconsIcon
                icon={ArrowDown01Icon}
                strokeWidth={2}
                aria-hidden='true'
                className='text-muted-foreground ml-auto size-3.5 -rotate-90 transition-transform group-open/docs-section:rotate-0'
              />
            </summary>

            <div id={contentId} className='border-border/70 ml-2 border-l pl-3'>
              <div className='flex flex-col gap-px'>
                {group.items.map((item) => {
                  const isActive = item.id === props.currentPageId
                  return (
                    <Link
                      key={item.id}
                      to={item.path}
                      activeOptions={{ exact: true }}
                      aria-current={isActive ? 'page' : undefined}
                      className={cn(
                        'rounded-md border px-2.5 py-2 text-[13px] font-medium leading-5 transition-colors',
                        isActive
                          ? 'border-border bg-muted text-foreground font-semibold'
                          : 'border-transparent text-muted-foreground hover:bg-muted/50 hover:text-foreground'
                      )}
                    >
                      {item.label}
                    </Link>
                  )
                })}
              </div>
            </div>
          </details>
        )
      })}
    </nav>
  )
}

function DocsMobileNavigation(props: {
  currentPageId: DocsPageId
  groups: DocsNavigationGroup[]
  ariaLabel: string
}) {
  const currentItem = props.groups
    .flatMap((group) => group.items)
    .find((item) => item.id === props.currentPageId)

  return (
    <details className='group/docs-mobile relative mx-auto w-full max-w-[1400px]'>
      <summary
        aria-label={props.ariaLabel}
        className='border-border bg-background/90 focus-visible:ring-ring/50 flex min-h-11 cursor-pointer list-none items-center gap-3 rounded-lg border px-2.5 py-1.5 shadow-sm backdrop-blur-xl outline-none focus-visible:ring-3 [&::-webkit-details-marker]:hidden'
      >
        <span className='bg-muted flex size-8 shrink-0 items-center justify-center rounded-md'>
          <HugeiconsIcon
            icon={BookOpen01Icon}
            className='text-muted-foreground size-4'
            aria-hidden='true'
          />
        </span>
        <span className='flex min-w-0 flex-1 flex-col gap-0.5 text-left'>
          <span className='text-muted-foreground text-[11px] leading-none font-medium'>
            {props.ariaLabel}
          </span>
          <span className='truncate text-sm leading-5 font-semibold'>
            {currentItem?.label ?? props.ariaLabel}
          </span>
        </span>
        <HugeiconsIcon
          icon={ArrowDown01Icon}
          strokeWidth={2}
          aria-hidden='true'
          className='text-muted-foreground mr-1 size-4 transition-transform group-open/docs-mobile:rotate-180'
        />
      </summary>

      <nav
        aria-label={props.ariaLabel}
        className='border-border bg-popover absolute inset-x-0 top-[calc(100%+0.375rem)] max-h-[min(68svh,36rem)] overflow-y-auto rounded-lg border p-2 shadow-lg'
      >
        {props.groups.map((group) => (
          <section key={group.id} className='py-1'>
            <p className='text-muted-foreground px-2.5 py-1.5 text-xs font-semibold'>
              {group.label}
            </p>
            <div className='flex flex-col gap-px'>
              {group.items.map((item) => {
                const isActive = item.id === props.currentPageId
                return (
                  <Link
                    key={item.id}
                    to={item.path}
                    activeOptions={{ exact: true }}
                    aria-current={isActive ? 'page' : undefined}
                    className={cn(
                      'rounded-md px-2.5 py-2.5 text-sm leading-5 transition-colors',
                      isActive
                        ? 'bg-muted text-foreground font-semibold'
                        : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
                    )}
                  >
                    {item.label}
                  </Link>
                )
              })}
            </div>
          </section>
        ))}
      </nav>
    </details>
  )
}

export function DocsShell(props: DocsShellProps) {
  const { t } = useTranslation()
  const navigation = DOCS_NAVIGATION_GROUPS.flatMap((group) => group.items)
  const currentIndex = navigation.findIndex((item) => item.id === props.pageId)
  const previous = currentIndex > 0 ? navigation[currentIndex - 1] : undefined
  const next =
    currentIndex < navigation.length - 1
      ? navigation[currentIndex + 1]
      : undefined

  return (
    <PublicLayout showMainContainer={false}>
      <div data-doc-page-source='true'>
        <div className='pointer-events-none sticky top-16 z-30 mt-16 px-3 pt-2 md:hidden'>
          <div className='pointer-events-auto'>
            <DocsMobileNavigation
              currentPageId={props.pageId}
              groups={DOCS_NAVIGATION_GROUPS}
              ariaLabel={t('Documentation')}
            />
          </div>
        </div>

        <div className='mx-auto grid w-full max-w-[1400px] grid-cols-1 px-4 md:grid-cols-[256px_minmax(0,1fr)] md:gap-10 md:px-6 md:pt-16 xl:grid-cols-[256px_minmax(0,760px)_190px] xl:gap-12'>
          <aside className='hidden md:block'>
            <div className='border-border bg-card sticky top-24 mt-8 max-h-[calc(100svh-7rem)] overflow-y-auto rounded-lg border p-3'>
              <DocsNavigation
                currentPageId={props.pageId}
                groups={DOCS_NAVIGATION_GROUPS}
                ariaLabel={t('Documentation')}
              />
            </div>
          </aside>

          <main className='min-w-0 py-8 md:py-12'>
            <nav
              aria-label={t('Breadcrumb')}
              className='text-muted-foreground mb-8 flex items-center gap-2 text-sm'
            >
              <Link to='/' className='hover:text-foreground transition-colors'>
                {t('Home')}
              </Link>
              <span aria-hidden='true'>/</span>
              <Link
                to='/docs'
                className='hover:text-foreground transition-colors'
              >
                {t('Documentation')}
              </Link>
              {props.pageId !== 'introduction' && (
                <>
                  <span aria-hidden='true'>/</span>
                  <span className='text-foreground truncate'>
                    {props.title}
                  </span>
                </>
              )}
            </nav>

            <header>
              <h1 className='text-3xl font-semibold'>{props.title}</h1>
              <p className='text-muted-foreground mt-3 max-w-2xl text-base leading-7'>
                {props.description}
              </p>
            </header>

            <Separator className='my-8' />

            <article className='flex flex-col gap-12'>{props.children}</article>

            <Separator className='mt-12 mb-6' />
            <nav
              aria-label={t('Document pagination')}
              className='grid min-h-16 grid-cols-2 gap-4'
            >
              <div>
                {previous && (
                  <Link
                    to={previous.path}
                    className='group text-muted-foreground hover:text-foreground inline-flex items-center gap-2 text-sm transition-colors'
                  >
                    <HugeiconsIcon
                      icon={ArrowLeft01Icon}
                      className='size-4 transition-transform group-hover:-translate-x-0.5'
                      aria-hidden='true'
                    />
                    <span>
                      <span className='block text-xs'>{t('Previous')}</span>
                      <span className='text-foreground font-medium'>
                        {previous.label}
                      </span>
                    </span>
                  </Link>
                )}
              </div>
              <div className='text-right'>
                {next && (
                  <Link
                    to={next.path}
                    className='group text-muted-foreground hover:text-foreground inline-flex items-center gap-2 text-left text-sm transition-colors'
                  >
                    <span>
                      <span className='block text-xs'>{t('Next')}</span>
                      <span className='text-foreground font-medium'>
                        {next.label}
                      </span>
                    </span>
                    <HugeiconsIcon
                      icon={ArrowRight01Icon}
                      className='size-4 transition-transform group-hover:translate-x-0.5'
                      aria-hidden='true'
                    />
                  </Link>
                )}
              </div>
            </nav>
          </main>

          <aside className='hidden xl:block'>
            <nav
              aria-label={t('On this page')}
              className='border-border sticky top-24 mt-12 border-l pl-5'
            >
              <p className='text-muted-foreground mb-3 text-xs font-semibold'>
                {t('On this page')}
              </p>
              <div className='flex flex-col gap-2.5'>
                {props.toc.map((item) => (
                  <a
                    key={item.id}
                    href={`#${item.id}`}
                    className='text-muted-foreground hover:text-foreground text-sm transition-colors'
                  >
                    {item.label}
                  </a>
                ))}
              </div>
            </nav>
          </aside>
        </div>
      </div>
    </PublicLayout>
  )
}
