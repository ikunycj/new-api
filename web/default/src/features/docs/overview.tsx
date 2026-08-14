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
import {
  AiBrain01Icon,
  AiChat01Icon,
  ApiIcon,
  ArrowRight01Icon,
  Image01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { DocsShell, type DocsTocItem } from './components/docs-shell'

const OVERVIEW_TOC: DocsTocItem[] = [
  { id: 'usage-paths', label: '从这里开始使用 AI' },
]

const USAGE_PATHS = [
  {
    id: 'chat',
    icon: AiChat01Icon,
    title: '像豆包或 ChatGPT 一样聊天、写作',
    description: '直接和 AI 对话，写文章、翻译和总结内容。',
    to: '/playground' as const,
  },
  {
    id: 'image',
    icon: Image01Icon,
    title: '让 AI 帮你生成图片（多模态）',
    description: '自动切换到 gpt-image-2，并生成一只小猫。',
    to: '/playground' as const,
  },
  {
    id: 'tools',
    icon: AiBrain01Icon,
    title: '使用 AI Agent 工具（Codex、Claude Code）',
    description: '让 Codex 或 Claude Code 读取项目文件、编写代码并完成任务。',
    to: '/docs/tools/cc-switch' as const,
  },
  {
    id: 'api',
    icon: ApiIcon,
    title: '将 AI 接入自己的网站或业务系统（API）',
    description: '让你自己的程序调用 AI，适合开发者和企业使用。',
    to: '/docs/api/integration' as const,
  },
]

export function DocsOverview() {
  const { t } = useTranslation()

  return (
    <DocsShell
      pageId='introduction'
      title={t('AI 使用指南')}
      description={t(
        '从网页聊天、图片生成、AI Agent 工具或 API 接入中，选择适合你的开始方式。'
      )}
      toc={OVERVIEW_TOC}
    >
      <section id='usage-paths' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>{t('从这里开始使用 AI')}</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            '如果你平时只用过豆包或 ChatGPT 网页版，可以先按自己的目标选择一种方式。'
          )}
        </p>
        <div className='mt-6 grid gap-4 sm:grid-cols-2'>
          {USAGE_PATHS.map((path) => {
            const cardContent = (
              <div className='flex items-start gap-3'>
                <HugeiconsIcon
                  icon={path.icon}
                  className='text-primary mt-0.5 size-5 shrink-0'
                  aria-hidden='true'
                />
                <div className='min-w-0'>
                  <h3 className='font-semibold'>{t(path.title)}</h3>
                  <p className='text-muted-foreground mt-2 text-sm leading-6'>
                    {t(path.description)}
                  </p>
                  <span className='text-primary mt-3 inline-flex items-center gap-1 text-sm font-medium'>
                    {t('了解并开始')}
                    <HugeiconsIcon
                      icon={ArrowRight01Icon}
                      className='size-4 transition-transform group-hover:translate-x-0.5'
                      aria-hidden='true'
                    />
                  </span>
                </div>
              </div>
            )

            if (path.id === 'image') {
              return (
                <Link
                  key={path.id}
                  to='/playground'
                  search={{
                    model: 'gpt-image-2',
                    prompt: '生成一只小猫',
                    autoSend: true,
                  }}
                  className='group border-border bg-card hover:border-primary/50 hover:bg-muted/30 focus-visible:ring-ring rounded-lg border p-5 transition-colors focus-visible:ring-2 focus-visible:outline-none'
                >
                  {cardContent}
                </Link>
              )
            }

            return (
              <Link
                key={path.id}
                to={path.to}
                className='group border-border bg-card hover:border-primary/50 hover:bg-muted/30 focus-visible:ring-ring rounded-lg border p-5 transition-colors focus-visible:ring-2 focus-visible:outline-none'
              >
                {cardContent}
              </Link>
            )
          })}
        </div>
      </section>
    </DocsShell>
  )
}
