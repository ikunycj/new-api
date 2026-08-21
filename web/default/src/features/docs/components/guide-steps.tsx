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
import { ArrowDown01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { ResponsiveDocsImage } from './responsive-docs-image'

export type GuideScreenshot = {
  readonly largeSrc: string
  readonly smallSrc: string
  readonly width: number
  readonly height: number
  readonly alt: string
  readonly caption: string
}

export type GuideStepItem = {
  content: ReactNode
  screenshots?: readonly GuideScreenshot[]
}

function GuideStep(props: {
  number: number
  content: ReactNode
  screenshots?: readonly GuideScreenshot[]
}) {
  const { t } = useTranslation()
  const hasScreenshots = Boolean(props.screenshots?.length)

  return (
    <li className='grid grid-cols-[2rem_minmax(0,1fr)] gap-3'>
      <span className='bg-muted flex size-8 items-center justify-center rounded-lg text-sm font-semibold'>
        {props.number}
      </span>
      <div className='min-w-0 pt-0.5'>
        {hasScreenshots && props.screenshots ? (
          <details className='group/guide-step'>
            <summary className='focus-visible:ring-ring/50 flex cursor-pointer list-none items-start justify-between gap-x-4 gap-y-2 rounded-md outline-none focus-visible:ring-3 [&::-webkit-details-marker]:hidden'>
              <span className='min-w-0 flex-1 leading-7'>{props.content}</span>
              <span className='text-primary inline-flex shrink-0 items-center gap-1 pt-1 text-sm font-medium underline-offset-4 group-hover/guide-step:underline'>
                <span className='group-open/guide-step:hidden'>
                  {t('View screenshot example')}
                </span>
                <span className='hidden group-open/guide-step:inline'>
                  {t('Hide screenshot example')}
                </span>
                <HugeiconsIcon
                  icon={ArrowDown01Icon}
                  aria-hidden='true'
                  className='size-4 transition-transform group-open/guide-step:rotate-180'
                />
              </span>
            </summary>
            <div className='mt-1 w-full max-w-[620px] space-y-4'>
              {props.screenshots.map((screenshot) => (
                <ResponsiveDocsImage
                  key={screenshot.largeSrc}
                  {...screenshot}
                />
              ))}
            </div>
          </details>
        ) : (
          <div className='leading-7'>{props.content}</div>
        )}
      </div>
    </li>
  )
}

export function GuideSteps(props: { items: readonly GuideStepItem[] }) {
  return (
    <ol className='mt-5 flex flex-col gap-4'>
      {props.items.map((item, index) => (
        <GuideStep
          key={`step-${index + 1}`}
          number={index + 1}
          content={item.content}
          screenshots={item.screenshots}
        />
      ))}
    </ol>
  )
}
