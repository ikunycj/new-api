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
import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'

import { DocsShell } from './components/docs-shell'

const QUICK_START_TOC = [
  { id: 'cc-switch', label: 'CC Switch 一键导入' },
  { id: 'playground', label: '网页端对话' },
  { id: 'manual', label: '手动配置客户端' },
]

export function DocsQuickStart() {
  return (
    <DocsShell
      pageId='quick-start'
      title='快速开始'
      description='选择一种方式开始使用 AllTokenAPI。'
      toc={QUICK_START_TOC}
    >
      <section id='cc-switch' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          1. 使用 CC Switch 一键导入客户端
        </h2>
        <Button className='mt-5' render={<Link to='/docs/tools/cc-switch' />}>
          查看 CC Switch 配置
          <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
        </Button>
      </section>

      <section id='playground' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 直接在网页端开启对话</h2>
        <Button
          className='mt-5'
          variant='outline'
          render={<Link to='/playground' />}
        >
          打开网页对话
          <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
        </Button>
      </section>

      <section id='manual' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          3. 自行修改配置文件接入 AI 客户端
        </h2>
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button render={<Link to='/docs/integrations' />}>
            查看集成指南
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button
            variant='outline'
            render={<Link to='/docs/api/integration' />}
          >
            查看 API 模型接口
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>
      </section>
    </DocsShell>
  )
}
