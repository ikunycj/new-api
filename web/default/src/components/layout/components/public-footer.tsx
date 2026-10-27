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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { HeaderLogo } from '@/components/layout/components/header-logo'
import { RichContent } from '@/components/rich-content'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { isLikelyHtml } from '@/lib/content-format'
import { cn } from '@/lib/utils'

export function PublicFooter() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { logo, loading, logoLoaded, systemName } = useSystemConfig()
  const navLinks = useTopNavLinks()
  const customerService =
    typeof status?.customer_service === 'string'
      ? status.customer_service.trim()
      : ''
  const footerHtml =
    typeof status?.footer_html === 'string' ? status.footer_html.trim() : ''
  const displayName = status?.system_name || systemName
  const legalLinks = []

  if (status?.user_agreement_enabled) {
    legalLinks.push({ href: '/user-agreement', label: t('User Agreement') })
  }
  if (status?.privacy_policy_enabled) {
    legalLinks.push({ href: '/privacy-policy', label: t('Privacy Policy') })
  }

  return (
    <footer
      aria-label={t('Contact us')}
      className='border-border/70 bg-muted/20 scroll-mt-24 border-t'
    >
      <div className='container mx-auto w-full max-w-7xl px-4 py-12 md:px-6 md:py-14'>
        <div
          className={cn(
            'grid gap-10 sm:grid-cols-2',
            customerService
              ? 'lg:grid-cols-[minmax(0,1.4fr)_minmax(12rem,0.7fr)_minmax(15rem,1fr)]'
              : 'lg:grid-cols-[minmax(0,1.4fr)_minmax(12rem,0.7fr)]'
          )}
        >
          <div className='min-w-0'>
            <Link
              to='/'
              aria-label={t('Go to home')}
              className='inline-flex max-w-full items-center gap-3'
            >
              <HeaderLogo
                src={logo}
                alt=''
                loading={loading}
                logoLoaded={logoLoaded}
                className='size-9 rounded-md object-contain'
              />
              <span className='truncate text-base font-semibold'>
                {displayName}
              </span>
            </Link>
          </div>

          <div>
            <h2 className='text-sm font-semibold'>{t('Explore')}</h2>
            <nav
              aria-label={t('Explore')}
              className='mt-4 flex flex-col items-start gap-3'
            >
              {navLinks.map((link) =>
                link.external ? (
                  <a
                    key={`${link.title}-${link.href}`}
                    href={link.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    aria-disabled={link.disabled}
                    className={cn(
                      'text-muted-foreground hover:text-foreground text-sm transition-colors',
                      link.disabled && 'pointer-events-none opacity-50'
                    )}
                  >
                    {link.title}
                  </a>
                ) : (
                  <Link
                    key={`${link.title}-${link.href}`}
                    to={link.href}
                    disabled={link.disabled}
                    className={cn(
                      'text-muted-foreground hover:text-foreground text-sm transition-colors',
                      link.disabled && 'pointer-events-none opacity-50'
                    )}
                  >
                    {link.title}
                  </Link>
                )
              )}
            </nav>
          </div>

          {customerService && (
            <section aria-labelledby='public-footer-contact-title'>
              <h2 id='public-footer-contact-title' className='text-sm font-semibold'>
                <Link to='/about' className='hover:text-primary transition-colors'>
                  {t('Contact us')}
                </Link>
              </h2>
              <p className='text-muted-foreground mt-4 max-w-sm text-sm leading-6 break-words whitespace-pre-wrap'>
                {customerService}
              </p>
            </section>
          )}
        </div>

        {(footerHtml || legalLinks.length > 0) && (
          <div className='border-border/70 mt-10 flex flex-col gap-4 border-t pt-5 text-sm sm:flex-row sm:items-center sm:justify-between'>
            {footerHtml &&
              (isLikelyHtml(footerHtml) ? (
                <RichContent
                  mode='html'
                  content={footerHtml}
                  className='text-muted-foreground max-w-3xl text-sm leading-6 [&_a]:text-primary [&_a]:underline [&_p]:my-0 [&_p+p]:mt-2'
                />
              ) : (
                <p className='text-muted-foreground max-w-3xl text-sm leading-6 whitespace-pre-wrap'>
                  {footerHtml}
                </p>
              ))}
            {legalLinks.length > 0 && (
              <nav
                aria-label={t('Legal')}
                className='flex flex-wrap items-center gap-x-5 gap-y-2'
              >
                {legalLinks.map((link) => (
                  <Link
                    key={link.href}
                    to={link.href}
                    className='text-muted-foreground hover:text-foreground transition-colors'
                  >
                    {link.label}
                  </Link>
                ))}
              </nav>
            )}
          </div>
        )}
      </div>
    </footer>
  )
}
