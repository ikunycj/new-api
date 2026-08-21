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

import { CC_SWITCH_SCREENSHOTS as screenshots } from './components/cc-switch-screenshots'
import { DocsShell } from './components/docs-shell'
import { GuideSteps } from './components/guide-steps'

const CC_SWITCH_RELEASES_URL =
  'https://github.com/farion1231/cc-switch/releases'
const CC_SWITCH_DOCS_URL = 'https://ccswitch.io/zh/docs'
const CC_SWITCH_INSTALL_URL =
  'https://ccswitch.io/zh/docs?section=getting-started&item=installation'
const CC_SWITCH_QUICKSTART_URL =
  'https://ccswitch.io/zh/docs?section=getting-started&item=quickstart'
const DOCS_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

export function DocsCcSwitch() {
  return (
    <DocsShell
      pageId='cc-switch'
      title='CC Switch'
      description='cc switch是一款主流AI客户端的配置管理软件，帮助您一键配置AI客户端'
      toc={[
        { id: 'download', label: '1. 下载 CC Switch' },
        { id: 'usage', label: '2. 使用和导入' },
        { id: 'one-click-import', label: '2.1 一键导入配置' },
        { id: 'manual-config', label: '2.2 手动配置' },
        { id: 'advanced', label: '高级用法' },
      ]}
    >
      <section id='download' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 下载 CC Switch</h2>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  打开{' '}
                  <a
                    href={CC_SWITCH_RELEASES_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                    className={DOCS_LINK_CLASS}
                  >
                    CC Switch 发布页
                  </a>
                  ，选择最新版本。
                </>
              ),
              screenshots: [screenshots.download],
            },
            {
              content: (
                <>
                  在页面底部的安装包列表中选择与操作系统和 CPU
                  架构匹配的文件；也可以查看{' '}
                  <a
                    href={CC_SWITCH_INSTALL_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                    className={DOCS_LINK_CLASS}
                  >
                    官方安装指南
                  </a>{' '}
                  确认安装方式。
                </>
              ),
            },
          ]}
        />
      </section>

      <section id='usage' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 使用和导入</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用前可以先阅读{' '}
          <a
            href={CC_SWITCH_DOCS_URL}
            target='_blank'
            rel='noopener noreferrer'
            className={DOCS_LINK_CLASS}
          >
            CC Switch 官方文档
          </a>{' '}
          和{' '}
          <a
            href={CC_SWITCH_QUICKSTART_URL}
            target='_blank'
            rel='noopener noreferrer'
            className={DOCS_LINK_CLASS}
          >
            官方快速上手文档
          </a>
          。
        </p>
      </section>

      <section id='one-click-import' className='scroll-mt-28'>
        <h3 className='text-xl font-semibold'>2.1 一键导入 CC Switch 配置</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          目前仅支持 Codex、Claude Code 和 Gemini。
        </p>
        <GuideSteps
          items={[
            {
              content: (
                <>
                  登录本站，进入{' '}
                  <Link to='/keys' className={DOCS_LINK_CLASS}>
                    API Key 页面
                  </Link>{' '}
                  并创建密钥。
                </>
              ),
              screenshots: [screenshots.apiKey],
            },
            {
              content: (
                <>
                  填写密钥名称，选择分组后保存密钥；分组的计费方式可查看{' '}
                  <Link to='/docs/model-pricing' className={DOCS_LINK_CLASS}>
                    模型定价说明
                  </Link>
                  。
                </>
              ),
              screenshots: [screenshots.apiKeyDetails],
            },
            {
              content: '在密钥行的操作菜单中点击 CC Switch 图标。',
              screenshots: [screenshots.importEntry],
            },
            {
              content:
                '确认导入信息无误后，允许浏览器打开 ccswitch:// 链接并点击导入。',
              screenshots: [screenshots.importDialog],
            },
            {
              content:
                '在 CC Switch 中确认并启用导入的供应商，然后重启对应 Agent 客户端使配置生效。',
              screenshots: [screenshots.confirmImport, screenshots.imported],
            },
          ]}
        />
      </section>

      <section id='manual-config' className='scroll-mt-28'>
        <h3 className='text-xl font-semibold'>2.2 手动配置 CC Switch</h3>
        <GuideSteps
          items={[
            {
              content: '打开 CC Switch，点击右上角加号添加模型供应商。',
              screenshots: [screenshots.addProvider],
            },
            {
              content:
                '选择统一供应商或 Codex 供应商，自定义配置，然后下滑填写具体配置。',
              screenshots: [screenshots.providerType],
            },
            {
              content: (
                <>
                  按照图示填写信息，红框标注项为必填；需要 API Key 时可直接前往{' '}
                  <Link to='/keys' className={DOCS_LINK_CLASS}>
                    API Key 页面
                  </Link>{' '}
                  创建密钥，完成后点击添加。
                </>
              ),
              screenshots: [screenshots.providerConfig],
            },
            {
              content: '点击启用，并重启对应的 Agent 客户端。',
              screenshots: [screenshots.enabled],
            },
          ]}
        />
      </section>

      <section id='advanced' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>高级用法</h2>
        <div className='mt-4 space-y-3'>
          <h3 className='text-lg font-semibold'>
            <a
              href='https://ccswitch.io/zh/tutorials/claude-codex-routing-guide'
              target='_blank'
              rel='noopener noreferrer'
              className={DOCS_LINK_CLASS}
            >
              在 Claude Code 中使用 ChatGPT
            </a>
          </h3>
          <h3 className='text-lg font-semibold'>
            <a
              href='https://ccswitch.io/zh/tutorials/codex-claude-routing-guide'
              target='_blank'
              rel='noopener noreferrer'
              className={DOCS_LINK_CLASS}
            >
              在 Codex 中使用 Claude 模型
            </a>
          </h3>
          <h3 className='text-lg font-semibold'>
            <a
              href='https://ccswitch.io/zh/tutorials'
              target='_blank'
              rel='noopener noreferrer'
              className={DOCS_LINK_CLASS}
            >
              更多高级用法请见 CC Switch 官网
            </a>
          </h3>
        </div>
      </section>
    </DocsShell>
  )
}
