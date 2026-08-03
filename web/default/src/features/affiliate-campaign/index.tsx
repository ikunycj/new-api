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
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowRight, CalendarClock, Gift, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import { getAffiliateCampaign } from '../wallet/api'

function useAffiliateCampaign() {
  return useQuery({
    queryKey: ['affiliate', 'campaign'],
    queryFn: getAffiliateCampaign,
    select: (response) => response.data,
    staleTime: 5 * 60 * 1000,
  })
}

function isCampaignVisible(enabled: boolean, endsAt: number): boolean {
  return enabled && endsAt > Math.floor(Date.now() / 1000)
}

export function AffiliateCampaignPage() {
  const { t } = useTranslation()
  const query = useAffiliateCampaign()
  const isAuthenticated = useAuthStore((state) => !!state.auth.user)
  const campaign = query.data
  const visible = campaign
    ? isCampaignVisible(campaign.enabled, campaign.ends_at)
    : false

  return (
    <PublicLayout showMainContainer={false}>
      <main>
        <section className='min-h-[min(680px,82vh)] bg-neutral-950 text-white'>
          <div className='mx-auto flex min-h-[min(680px,82vh)] max-w-7xl flex-col justify-end px-5 py-14 sm:px-8 sm:py-20'>
            <div className='max-w-3xl'>
              <p className='text-sm font-semibold tracking-normal text-emerald-300 uppercase'>
                {visible
                  ? t('Limited-time referral campaign')
                  : t('Referral campaign')}
              </p>
              <h1 className='mt-3 text-4xl font-semibold sm:text-5xl'>
                {campaign?.name || t('Invite rewards campaign')}
              </h1>
              <p className='mt-5 max-w-2xl text-base leading-7 text-white/80 sm:text-lg'>
                {t(
                  'Invite a new user. You receive 25% of every eligible top-up as CNY cashback, and the invited user receives 20% extra quota.'
                )}
              </p>
              <div className='mt-7 flex flex-wrap gap-3'>
                <Button
                  size='lg'
                  render={
                    <Link to={isAuthenticated ? '/wallet' : '/sign-up'} />
                  }
                >
                  {isAuthenticated ? t('Open wallet') : t('Create account')}
                  <ArrowRight data-icon='inline-end' />
                </Button>
                {campaign?.ends_at ? (
                  <div className='flex items-center gap-2 rounded-md border border-white/25 bg-black/25 px-3 text-sm text-white/80'>
                    <CalendarClock className='size-4' />
                    {t('Ends {{time}}', {
                      time: formatTimestampToDate(campaign.ends_at),
                    })}
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        </section>

        <section className='bg-background'>
          <div className='mx-auto grid max-w-6xl gap-10 px-5 py-14 sm:px-8 md:grid-cols-2 md:py-20'>
            <div className='flex gap-4'>
              <WalletCards className='text-primary mt-1 size-6 shrink-0' />
              <div>
                <h2 className='text-xl font-semibold'>
                  {t('25% CNY cashback')}
                </h2>
                <p className='text-muted-foreground mt-2 text-sm leading-6'>
                  {t(
                    'Cashback is recorded in your wallet after each eligible payment and can be transferred to your account balance after the hold period.'
                  )}
                </p>
              </div>
            </div>
            <div className='flex gap-4'>
              <Gift className='text-primary mt-1 size-6 shrink-0' />
              <div>
                <h2 className='text-xl font-semibold'>
                  {t('20% extra quota')}
                </h2>
                <p className='text-muted-foreground mt-2 text-sm leading-6'>
                  {t(
                    'The invited user receives an additional 20% quota automatically when an eligible online top-up succeeds.'
                  )}
                </p>
              </div>
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </PublicLayout>
  )
}
