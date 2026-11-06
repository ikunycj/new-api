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

import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardAction,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

import {
  FEATURED_MODELS,
  FEATURED_USD_TO_CNY_RATE,
  getFeaturedPriceDisplay,
  type FeaturedModel,
} from '../../lib/featured-models'
import { HomeProviderIcon } from './home-provider-icon'
import { SectionHeading } from './section-heading'

interface FeaturedModelsSectionProps {
  catalogAvailable: boolean
}

function ModelPrice(props: {
  officialPricesUSD: readonly number[]
  discountRatio: number
}) {
  const { t } = useTranslation()
  const price = getFeaturedPriceDisplay(
    props.officialPricesUSD,
    props.discountRatio,
    FEATURED_USD_TO_CNY_RATE
  )

  return (
    <div className='mt-1 space-y-0.5'>
      <p className='text-primary font-mono text-sm font-semibold tabular-nums'>
        {price.currentPrice}
      </p>
      <p className='text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-[11px] leading-4'>
        <del
          className='font-mono tabular-nums decoration-current'
          aria-label={t('Official price: {{price}}', {
            price: price.officialPrice,
          })}
        >
          {price.officialPrice}
        </del>
        <span>
          {t('×{{ratio}} of official price', {
            ratio: props.discountRatio,
            folds: Number((props.discountRatio * 10).toFixed(1)),
          })}
        </span>
      </p>
    </div>
  )
}

function ModelPreviewCard(props: {
  model: FeaturedModel
  targetPath: '/pricing' | '/docs'
}) {
  const { t } = useTranslation()

  return (
    <Card className='h-full min-h-72 rounded-lg' data-card-hover='true'>
      <Link
        to={props.targetPath}
        className='focus-visible:ring-ring/50 flex flex-1 flex-col gap-4 rounded-t-lg focus-visible:ring-[3px] focus-visible:outline-none'
      >
        <CardHeader>
          <div className='border-border/70 bg-background mb-3 flex size-11 items-center justify-center overflow-hidden rounded-lg border shadow-xs'>
            <HomeProviderIcon
              icon={props.model.icon}
              provider={props.model.provider}
              size={30}
            />
          </div>
          <CardTitle className='truncate text-lg' title={props.model.modelName}>
            {props.model.modelName}
          </CardTitle>
          <CardAction>
            <Badge variant='outline'>{props.model.provider}</Badge>
          </CardAction>
        </CardHeader>

        <CardContent className='mt-auto space-y-3'>
          <p className='text-muted-foreground text-xs'>
            {t(props.model.pricingScope)}
          </p>
          <div className='grid grid-cols-2 gap-4'>
            {props.model.prices.map((price) => (
              <div key={price.label}>
                <p className='text-muted-foreground text-xs'>
                  {t(price.label)}
                </p>
                <ModelPrice
                  officialPricesUSD={price.usd}
                  discountRatio={props.model.discountRatio}
                />
              </div>
            ))}
          </div>
        </CardContent>
      </Link>

      <CardFooter className='justify-between gap-3'>
        <span className='text-muted-foreground text-xs'>
          {t('Per 1M tokens')} · CNY
        </span>
        <a
          href={props.model.sourceUrl}
          target='_blank'
          rel='noopener noreferrer'
          className='text-muted-foreground hover:text-primary focus-visible:ring-ring/50 max-w-40 truncate rounded text-xs underline underline-offset-4 focus-visible:ring-2 focus-visible:outline-none'
          aria-label={`${t('Official price')}: ${props.model.modelName}`}
        >
          {t('Official price')} ↗
        </a>
      </CardFooter>
    </Card>
  )
}

export function FeaturedModelsSection(props: FeaturedModelsSectionProps) {
  const { t } = useTranslation()
  const targetPath = props.catalogAvailable ? '/pricing' : '/docs'

  return (
    <section className='px-4 py-16 sm:px-6 sm:py-20 lg:py-24'>
      <div className='mx-auto w-full max-w-6xl'>
        <SectionHeading
          eyebrow={t('Featured models')}
          title={t('Various prices at a glance')}
          action={
            <Link
              to='/pricing'
              className='text-primary shrink-0 text-xs font-medium underline-offset-4 hover:underline'
            >
              {t('View more in Model Square')}
            </Link>
          }
        />

        <div className='grid gap-3 md:grid-cols-2 lg:grid-cols-3'>
          {FEATURED_MODELS.map((model) => (
            <ModelPreviewCard
              key={model.modelName}
              model={model}
              targetPath={targetPath}
            />
          ))}
        </div>
      </div>
    </section>
  )
}
