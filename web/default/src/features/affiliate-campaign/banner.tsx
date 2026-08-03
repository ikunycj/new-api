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
import { Link } from '@tanstack/react-router'
import { ArrowRight, Gift, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function AffiliateCampaignBanner() {
  const { t } = useTranslation()

  return (
    <section className='border-border bg-muted/25 border-y [contain-intrinsic-size:560px] [content-visibility:auto]'>
      <div className='mx-auto max-w-7xl px-5 py-12 sm:px-8 sm:py-16 lg:px-12 lg:py-20'>
        <div className='text-primary text-xs font-semibold tracking-normal uppercase'>
          {t('Referral campaign')}
        </div>
        <h2 className='mt-3 max-w-2xl text-3xl font-semibold sm:text-4xl'>
          {t('Invite friends and earn 25% cashback')}
        </h2>
        <p className='text-muted-foreground mt-4 max-w-2xl text-sm leading-6 sm:text-base sm:leading-7'>
          {t(
            'Every invited top-up earns you 25% CNY cashback, while your friend receives 20% extra quota.'
          )}
        </p>

        <div className='border-border mt-8 grid max-w-2xl gap-6 border-y py-6 sm:grid-cols-2 sm:gap-0'>
          <div className='flex gap-3 sm:pr-6'>
            <WalletCards className='text-primary mt-1 size-5 shrink-0' />
            <div>
              <div className='text-2xl font-semibold'>25%</div>
              <div className='text-muted-foreground mt-1 text-sm'>
                {t('25% CNY cashback')}
              </div>
            </div>
          </div>
          <div className='flex gap-3 sm:border-l sm:pl-6'>
            <Gift className='text-primary mt-1 size-5 shrink-0' />
            <div>
              <div className='text-2xl font-semibold'>20%</div>
              <div className='text-muted-foreground mt-1 text-sm'>
                {t('20% extra quota')}
              </div>
            </div>
          </div>
        </div>

        <div className='mt-7'>
          <Button render={<Link to='/campaign/referral' />}>
            {t('View campaign')}
            <ArrowRight data-icon='inline-end' />
          </Button>
        </div>
      </div>
    </section>
  )
}
