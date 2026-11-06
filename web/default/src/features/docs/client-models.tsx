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
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { GuideSteps, type GuideScreenshot } from './components/guide-steps'

const DOCS_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const CLIENT_MODELS_TOC: DocsTocItem[] = [
  { id: 'what-it-does', label: '概览' },
  { id: 'setup', label: '设置步骤' },
  { id: 'notes', label: '注意事项' },
]

const CLIENT_MODELS_SCREENSHOTS = {
  modelGroups: {
    largeSrc: '/static/image/docs/client-models/model-groups.webp',
    smallSrc: '/static/image/docs/client-models/model-groups-small.webp',
    width: 1482,
    height: 1118,
    alt: '创建 API Key 时选择多个模型分组',
    caption: '创建密钥时可以选择多个分组，并通过拖动调整分组优先级。',
  },
  modelMapping: {
    largeSrc: '/static/image/docs/client-models/cc-switch-mapping.webp',
    smallSrc: '/static/image/docs/client-models/cc-switch-mapping-small.webp',
    width: 3416,
    height: 2220,
    alt: 'CC Switch 高级选项中的模型映射和第三方模型配置',
    caption:
      '在高级选项中获取模型列表、添加模型，并设置显示名称与实际请求模型。',
  },
} as const satisfies Record<string, GuideScreenshot>

export function DocsClientModels() {
  const { t } = useTranslation()

  return (
    <DocsShell
      pageId='client-models'
      title={t('Configure multi-provider models in your client')}
      description='通过一个 API Key 和 CC Switch，在客户端中同时使用多个模型厂商，并为相同模型设置故障降级顺序。'
      toc={CLIENT_MODELS_TOC}
    >
      <section id='what-it-does' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>概览</h2>
        <p className='mt-3 leading-7'>
          通过一个 API
          密钥配置多个分组，实现一个客户端接入多重厂商模型，具体接口能力可查看{' '}
          <a
            href='http://localhost:5173/docs/api/compatibility'
            className={DOCS_LINK_CLASS}
          >
            接口选择与兼容性
          </a>
          。
        </p>
      </section>

      <section id='setup' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>设置步骤</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  <span className='block'>
                    创建 API Key 时选择多个分组，并按需要调整顺序。
                  </span>
                  <span className='text-muted-foreground mt-1 block text-sm leading-6'>
                    相同模型会按照分组顺序从上到下优先使用；不同模型不互相排斥，可以同时使用。
                  </span>
                </>
              ),
              screenshots: [CLIENT_MODELS_SCREENSHOTS.modelGroups],
            },
            {
              content: (
                <>
                  下载并打开{' '}
                  <a
                    href='https://github.com/farion1231/cc-switch/releases'
                    target='_blank'
                    rel='noopener noreferrer'
                    className={DOCS_LINK_CLASS}
                  >
                    CC Switch
                  </a>
                  ，将包含多个模型分组的 API Key 导入 CC
                  Switch。然后进入高级选项，设置模型映射和第三方模型：先获取模型列表，再添加需要在客户端中显示和使用的模型，并填写对应的实际请求模型。
                </>
              ),
              screenshots: [CLIENT_MODELS_SCREENSHOTS.modelMapping],
            },
            {
              content: '保存设置，然后重启客户端使配置生效。',
            },
          ]}
        />
      </section>

      <section id='notes' className='scroll-mt-28'>
        <Alert className='mt-2'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>重要注意事项</AlertTitle>
          <AlertDescription>
            Codex 和 ChatGPT 桌面客户端需要先登录 ChatGPT。请使用 CC Switch 保存
            ChatGPT 登录态，否则即使 API Key
            和模型映射已保存，客户端也可能无法正常使用。
          </AlertDescription>
        </Alert>
        <p className='text-muted-foreground mt-5 leading-7'>
          如果修改了模型映射、第三方模型或分组顺序，请再次保存并重启对应客户端；仅刷新页面通常不会让已启动的客户端重新读取配置。
        </p>
      </section>
    </DocsShell>
  )
}
