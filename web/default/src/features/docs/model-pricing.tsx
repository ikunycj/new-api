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
import { ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import Claude from '@lobehub/icons/es/Claude'
import Gemini from '@lobehub/icons/es/Gemini'
import OpenAI from '@lobehub/icons/es/OpenAI'
import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'

import { DocsShell } from './components/docs-shell'

const MODEL_PRICING_TOC = [
  { id: 'pricing', label: '模型定价' },
  { id: 'providers', label: '有哪些供应商' },
  { id: 'groups', label: '按照分组计费' },
]

export function DocsModelPricing() {
  return (
    <DocsShell
      pageId='model-pricing'
      title='模型定价'
      description='查看当前模型价格，并了解供应商与分组倍率。'
      toc={MODEL_PRICING_TOC}
    >
      <section id='pricing' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>模型定价</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          模型定价请前往模型定价页面查看。
        </p>
        <div className='mt-5'>
          <Button render={<Link to='/pricing' />}>
            打开模型定价
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>

        <div className='border-border mt-7 border-t pt-5'>
          <p className='text-muted-foreground text-sm leading-6'>
            也可以前往模型厂商官网查看官方定价：
          </p>
          <div className='mt-3 grid gap-2 sm:grid-cols-3'>
            <Button
              variant='outline'
              className='justify-start'
              render={
                <a
                  href='https://openai.com/api/pricing/'
                  target='_blank'
                  rel='noreferrer'
                />
              }
            >
              <OpenAI data-icon='inline-start' aria-hidden='true' />
              OpenAI / GPT
            </Button>
            <Button
              variant='outline'
              className='justify-start'
              render={
                <a
                  href='https://www.anthropic.com/pricing#api'
                  target='_blank'
                  rel='noreferrer'
                />
              }
            >
              <Claude.Color data-icon='inline-start' aria-hidden='true' />
              Claude
            </Button>
            <Button
              variant='outline'
              className='justify-start'
              render={
                <a
                  href='https://ai.google.dev/gemini-api/docs/pricing'
                  target='_blank'
                  rel='noreferrer'
                />
              }
            >
              <Gemini.Color data-icon='inline-start' aria-hidden='true' />
              Gemini
            </Button>
          </div>
        </div>
      </section>

      <section id='providers' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>有哪些供应商</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          目前文档中列出的模型厂商有：
        </p>
        <ol className='border-border mt-5 divide-y border-y'>
          <li className='flex items-center gap-4 py-3'>
            <span className='text-muted-foreground w-6 shrink-0 font-mono text-xs'>
              01
            </span>
            <span className='font-medium'>ChatGPT</span>
          </li>
          <li className='flex items-center gap-4 py-3'>
            <span className='text-muted-foreground w-6 shrink-0 font-mono text-xs'>
              02
            </span>
            <span className='font-medium'>Claude</span>
          </li>
        </ol>
      </section>

      <section id='groups' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>按照分组计费</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          同一个供应商会有多个分组。分组中的模型能力相同，但价格、稳定性和速度可能有所差异。
        </p>
        <div className='border-border bg-muted/30 mt-5 border-l-2 px-4 py-3'>
          <p className='text-muted-foreground leading-7'>
            每个分组都会显示一个倍率，例如{' '}
            <code className='bg-background text-foreground rounded border px-1.5 py-0.5 font-semibold'>
              0.05x
            </code>
            ，表示以模型官方价格为标准的 0.05 倍价格。
          </p>
        </div>
      </section>
    </DocsShell>
  )
}
