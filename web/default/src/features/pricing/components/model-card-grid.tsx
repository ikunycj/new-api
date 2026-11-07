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
import { useEffect, useMemo, useRef, useState } from 'react'

import { DEFAULT_TOKEN_UNIT, PRICING_CARD_BATCH_SIZE } from '../constants'
import type { PricingDisplayModel, TokenUnit } from '../types'
import { ModelCard } from './model-card'

export interface ModelCardGridProps {
  models: PricingDisplayModel[]
  onModelClick: (modelName: string) => void
  priceRate?: number
  usdExchangeRate?: number
  tokenUnit?: TokenUnit
  showRechargePrice?: boolean
}

export function ModelCardGrid(props: ModelCardGridProps) {
  const [visibleCount, setVisibleCount] = useState(() =>
    Math.min(PRICING_CARD_BATCH_SIZE, props.models.length)
  )
  const loadMoreMarkerRef = useRef<HTMLDivElement>(null)
  const tokenUnit = props.tokenUnit ?? DEFAULT_TOKEN_UNIT

  useEffect(() => {
    setVisibleCount(Math.min(PRICING_CARD_BATCH_SIZE, props.models.length))
  }, [props.models])

  useEffect(() => {
    const marker = loadMoreMarkerRef.current
    if (!marker || visibleCount >= props.models.length) return

    if (typeof window === 'undefined' || !('IntersectionObserver' in window)) {
      setVisibleCount(props.models.length)
      return
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry.isIntersecting) return
        setVisibleCount((current) =>
          Math.min(current + PRICING_CARD_BATCH_SIZE, props.models.length)
        )
      },
      { rootMargin: '0px 0px 400px 0px' }
    )

    observer.observe(marker)
    return () => observer.disconnect()
  }, [props.models.length, visibleCount])

  const visibleModels = useMemo(
    () => props.models.slice(0, visibleCount),
    [props.models, visibleCount]
  )

  if (props.models.length === 0) {
    return null
  }

  return (
    <div className='space-y-4 sm:space-y-5'>
      <div className='grid grid-cols-1 gap-3 sm:gap-4 md:grid-cols-2 lg:grid-cols-3'>
        {visibleModels.map((model) => (
          <ModelCard
            key={`${model.model_name}::${model.display_group}`}
            model={model}
            tokenUnit={tokenUnit}
            priceRate={props.priceRate}
            usdExchangeRate={props.usdExchangeRate}
            showRechargePrice={props.showRechargePrice}
            onClick={() => props.onModelClick(model.model_name || '')}
          />
        ))}
      </div>

      {visibleCount < props.models.length && (
        <div ref={loadMoreMarkerRef} className='h-px' aria-hidden='true' />
      )}
    </div>
  )
}
