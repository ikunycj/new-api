import {
  ArrowDown01Icon,
  ArrowLeft01Icon,
  ArrowRight01Icon,
  BookOpen01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link, useNavigate } from '@tanstack/react-router'
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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout/components/public-layout'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

import {
  DOCS_NAVIGATION_GROUPS,
  type DocsNavigationGroup,
  type DocsPageId,
  type DocsRoutePath,
} from '../docs-config'

const DOCS_NAVIGATION_STORAGE_KEY = 'new-api-docs-expanded-groups-v2'

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
  const activeGroupId = props.groups.find((group) =>
    group.items.some((item) => item.id === props.currentPageId)
  )?.id
  const [expandedGroups, setExpandedGroups] = useState<string[]>(() =>
    props.groups.map((group) => group.id)
  )
  const [hasHydrated, setHasHydrated] = useState(false)

  useEffect(() => {
    let storedGroups: string[] = []
    let hasStoredGroups = false

    try {
      const stored = window.localStorage.getItem(DOCS_NAVIGATION_STORAGE_KEY)
      const parsed: unknown = stored ? JSON.parse(stored) : undefined
      if (Array.isArray(parsed)) {
        hasStoredGroups = true
        storedGroups = parsed.filter(
          (groupId): groupId is string => typeof groupId === 'string'
        )
      }
    } catch {
      storedGroups = []
    }

    setExpandedGroups(() => {
      const validGroupIds = new Set(props.groups.map((group) => group.id))
      const defaultGroups = props.groups.map((group) => group.id)
      const nextGroups = new Set(
        (hasStoredGroups ? storedGroups : defaultGroups).filter((groupId) =>
          validGroupIds.has(groupId as DocsNavigationGroup['id'])
        )
      )
      if (activeGroupId) {
        nextGroups.add(activeGroupId)
      }
      return [...nextGroups].filter((groupId) =>
        validGroupIds.has(groupId as DocsNavigationGroup['id'])
      )
    })
    setHasHydrated(true)
  }, [activeGroupId, props.groups])

  useEffect(() => {
    if (!hasHydrated) {
      return
    }

    try {
      window.localStorage.setItem(
        DOCS_NAVIGATION_STORAGE_KEY,
        JSON.stringify(expandedGroups)
      )
    } catch {
      // Ignore storage failures; navigation remains usable without persistence.
    }
  }, [expandedGroups, hasHydrated])

  const toggleGroup = (groupId: DocsNavigationGroup['id']) => {
    setExpandedGroups((currentGroups) =>
      currentGroups.includes(groupId)
        ? currentGroups.filter((currentGroupId) => currentGroupId !== groupId)
        : [...currentGroups, groupId]
    )
  }

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
        const isExpanded = expandedGroups.includes(group.id)
        const contentId = `docs-group-${group.id}`

        return (
          <section key={group.id} className='mt-1.5'>
            <button
              type='button'
              aria-expanded={isExpanded}
              aria-controls={contentId}
              onClick={() => toggleGroup(group.id)}
              className='text-foreground hover:bg-muted/50 focus-visible:ring-ring/50 group flex min-h-10 w-full items-center rounded-md px-3 text-left text-sm font-semibold transition-colors outline-none focus-visible:ring-3'
            >
              <span>{group.label}</span>
              <HugeiconsIcon
                icon={ArrowDown01Icon}
                strokeWidth={2}
                aria-hidden='true'
                className={cn(
                  'text-muted-foreground ml-auto size-3.5 transition-transform',
                  !isExpanded && '-rotate-90'
                )}
              />
            </button>

            <div
              id={contentId}
              hidden={!isExpanded}
              className='border-border/70 ml-2 border-l pl-3'
            >
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
          </section>
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
  const navigate = useNavigate()
  const currentItem = props.groups
    .flatMap((group) => group.items)
    .find((item) => item.id === props.currentPageId)

  return (
    <div className='mx-auto flex w-full max-w-[1400px] items-center gap-2'>
      <HugeiconsIcon
        icon={BookOpen01Icon}
        className='text-muted-foreground size-4 shrink-0'
        aria-hidden='true'
      />
      <select
        data-doc-navigation='true'
        value={currentItem?.path ?? '/docs'}
        aria-label={props.ariaLabel}
        onChange={(event) => {
          void navigate({ to: event.target.value as DocsRoutePath })
        }}
        className='border-border bg-background text-foreground focus-visible:ring-ring h-9 min-w-0 flex-1 rounded-md border px-3 text-sm outline-none focus-visible:ring-2'
      >
        {props.groups.map((group) => (
          <optgroup key={group.label} label={group.label}>
            {group.items.map((item) => (
              <option key={item.id} value={item.path}>
                {item.label}
              </option>
            ))}
          </optgroup>
        ))}
      </select>
    </div>
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
        <div className='border-border bg-background/95 sticky top-16 z-30 mt-16 border-y px-4 py-2 backdrop-blur md:hidden'>
          <DocsMobileNavigation
            currentPageId={props.pageId}
            groups={DOCS_NAVIGATION_GROUPS}
            ariaLabel={t('Documentation')}
          />
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
