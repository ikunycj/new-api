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
  ArrowRight01Icon,
  Download04Icon,
  Key01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'

import { DocsShell } from './components/docs-shell'
import { NumberedSteps } from './components/numbered-steps'
import { ResponsiveDocsImage } from './components/responsive-docs-image'

const CC_SWITCH_RELEASES_URL =
  'https://github.com/farion1231/cc-switch/releases'
const CC_SWITCH_DOCS_URL = 'https://ccswitch.io/zh/docs'
const CC_SWITCH_INSTALL_URL =
  'https://ccswitch.io/zh/docs?section=getting-started&item=installation'
const CC_SWITCH_QUICKSTART_URL =
  'https://ccswitch.io/zh/docs?section=getting-started&item=quickstart'
const IMAGE_BASE = '/static/image/docs/cc-switch'

const screenshots = [
  {
    largeSrc: `${IMAGE_BASE}/step-01-1520.f704c73a6eb1.webp`,
    smallSrc: `${IMAGE_BASE}/step-01-760.8f5ed2222c27.webp`,
    width: 1859,
    height: 1370,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-02-1520.ebee24073ddb.webp`,
    smallSrc: `${IMAGE_BASE}/step-02-760.74e196f40b56.webp`,
    width: 3420,
    height: 1902,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-03-1520.5e3bd8e9b8b5.webp`,
    smallSrc: `${IMAGE_BASE}/step-03-760.877222304998.webp`,
    width: 3420,
    height: 1902,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-04-1520.03c352daa9ff.webp`,
    smallSrc: `${IMAGE_BASE}/step-04-760.bcbb3eb9f9c3.webp`,
    width: 3420,
    height: 1902,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-05-1520.0d6281dc4d60.webp`,
    smallSrc: `${IMAGE_BASE}/step-05-760.4d0ea61f3015.webp`,
    width: 3420,
    height: 1902,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-06-1520.f89c84dae388.webp`,
    smallSrc: `${IMAGE_BASE}/step-06-760.d86b028eb18a.webp`,
    width: 2000,
    height: 1302,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-07-1520.24a51ac729c8.webp`,
    smallSrc: `${IMAGE_BASE}/step-07-760.ae74559dcaff.webp`,
    width: 2000,
    height: 1302,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-08-1520.e53705f90a86.webp`,
    smallSrc: `${IMAGE_BASE}/step-08-760.8941d8843074.webp`,
    width: 2000,
    height: 1302,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-09-1520.786f116444c5.webp`,
    smallSrc: `${IMAGE_BASE}/step-09-760.e196babd0bd0.webp`,
    width: 2000,
    height: 1302,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-10-1520.ba6f0311bcdd.webp`,
    smallSrc: `${IMAGE_BASE}/step-10-760.e97e4ffcffde.webp`,
    width: 2000,
    height: 1302,
  },
  {
    largeSrc: `${IMAGE_BASE}/step-11-1520.0283546e75b8.webp`,
    smallSrc: `${IMAGE_BASE}/step-11-760.56feb3ad9e90.webp`,
    width: 2000,
    height: 1302,
  },
] as const

export function DocsCcSwitch() {
  return (
    <DocsShell
      pageId='cc-switch'
      title='AI Agent 工具与 CC Switch'
      description='先了解 Codex、Claude Code 这类 AI Agent 工具，再通过 CC Switch 一键导入 API Key、模型和服务地址。'
      toc={[
        { id: 'ai-agent-tools', label: 'Codex 与 Claude Code' },
        { id: 'why-cc-switch', label: '为什么推荐 CC Switch' },
        { id: 'download', label: '1. 下载 CC Switch' },
        { id: 'usage', label: '2. 使用和导入' },
        { id: 'one-click-import', label: '2.1 一键导入配置' },
        { id: 'manual-config', label: '2.2 手动配置' },
        { id: 'advanced', label: '高级用法' },
      ]}
    >
      <section id='ai-agent-tools' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Codex 与 Claude Code 是什么</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          Codex 和 Claude Code 都是 AI Agent
          工具。与只在网页里聊天不同，它们可以在你的电脑上读取项目文件、修改代码、运行命令，并根据结果继续完成后续步骤。
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          Codex 由 OpenAI 提供，Claude Code 由 Anthropic
          提供。两者都适合编程、排查问题和处理项目文件，但使用的配置文件和接入方式不同。
        </p>
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button nativeButton={false} render={<Link to='/docs/tools/codex' />}>
            查看 Codex 接入指南
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
          <Button
            nativeButton={false}
            variant='outline'
            render={<Link to='/docs/tools/claude-code' />}
          >
            查看 Claude Code 接入指南
            <HugeiconsIcon icon={ArrowRight01Icon} data-icon='inline-end' />
          </Button>
        </div>
      </section>

      <section id='why-cc-switch' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>为什么推荐 CC Switch</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          Agent 工具需要知道三项信息：使用哪个 API Key、调用哪个模型，以及向哪个
          Base URL 发送请求。Codex 通常读取 config.toml，Claude Code 则使用
          settings.json 或环境变量，新手手动填写时很容易漏字段或重复添加 /v1。
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          CC Switch 是一款跨平台配置管理工具。通过本站 API Key
          页面的一键导入，它会按 Codex 或 Claude Code
          的要求自动带入密钥、模型和服务地址，不需要手动查找并编辑配置文件；以后切换模型或服务商时，也可以直接切换已保存的配置。
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          CC Switch 不是使用 Agent
          的必需软件，但对第一次接入的用户更省步骤，也更不容易填错。熟悉配置后，仍可以按照对应工具文档手动设置。
        </p>
        <a
          href='https://ccswitch.io/zh/'
          target='_blank'
          rel='noopener noreferrer'
          className='text-primary mt-4 inline-flex font-medium underline-offset-4 hover:underline'
        >
          前往 CC Switch 官网
        </a>
      </section>

      <section id='download' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>1. 下载 CC Switch</h2>
        <NumberedSteps
          items={[
            '打开 CC Switch 发布页，选择最新版本。',
            '在页面底部的安装包列表中选择与操作系统和 CPU 架构匹配的文件；不确定时可以把设备信息交给 AI 判断。',
          ]}
        />
        <div className='mt-5 flex flex-wrap gap-3'>
          <Button
            nativeButton={false}
            variant='outline'
            render={
              <a
                href={CC_SWITCH_RELEASES_URL}
                target='_blank'
                rel='noopener noreferrer'
              />
            }
          >
            <HugeiconsIcon icon={Download04Icon} data-icon='inline-start' />
            下载 CC Switch
          </Button>
          <Button
            nativeButton={false}
            variant='ghost'
            render={
              <a
                href={CC_SWITCH_INSTALL_URL}
                target='_blank'
                rel='noopener noreferrer'
              />
            }
          >
            官方安装指南
          </Button>
        </div>
        <ResponsiveDocsImage
          {...screenshots[0]}
          alt='CC Switch 发布页安装包列表'
          caption='在发布页选择适合自己设备的安装包。'
        />
      </section>

      <section id='usage' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2. 使用和导入</h2>
        <div className='mt-4 flex flex-wrap gap-x-6 gap-y-2'>
          <a
            href={CC_SWITCH_DOCS_URL}
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary font-medium underline-offset-4 hover:underline'
          >
            CC Switch 官方文档
          </a>
          <a
            href={CC_SWITCH_QUICKSTART_URL}
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary font-medium underline-offset-4 hover:underline'
          >
            CC Switch 官方快速上手文档
          </a>
        </div>
      </section>

      <section id='one-click-import' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2.1 一键导入 CC Switch 配置</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          目前仅支持 Codex、Claude Code 和 Gemini。
        </p>
        <NumberedSteps
          items={[
            '登录本站，进入 API Key 页面并创建密钥。',
            '填写密钥名称，选择分组后保存密钥。',
            '在密钥行的操作菜单中点击 CC Switch 图标。',
            '确认导入信息无误后，允许浏览器打开 ccswitch:// 链接并点击导入。',
            '在 CC Switch 中启用导入的供应商，然后重启对应 Agent 客户端使配置生效。',
          ]}
        />
        <p className='text-muted-foreground mt-4 leading-7'>
          <Link
            to='/docs/model-pricing'
            className='text-primary font-medium underline-offset-4 hover:underline'
          >
            点击了解更多分组与计费信息
          </Link>
          。
        </p>
        <div className='mt-5'>
          <Button nativeButton={false} render={<Link to='/keys' />}>
            <HugeiconsIcon icon={Key01Icon} data-icon='inline-start' />
            打开 API Key 页面
          </Button>
        </div>
        <ResponsiveDocsImage
          {...screenshots[1]}
          alt='创建 API Key'
          caption='在 API Key 页面创建用于导入的密钥。'
        />
        <ResponsiveDocsImage
          {...screenshots[2]}
          alt='填写 API Key 名称和分组'
          caption='填写密钥名称并选择分组。'
        />
        <ResponsiveDocsImage
          {...screenshots[3]}
          alt='打开 CC Switch 导入入口'
          caption='创建密钥后打开 CC Switch 导入入口。'
        />
        <ResponsiveDocsImage
          {...screenshots[5]}
          alt='确认 CC Switch 导入信息'
          caption='导入前检查服务商、模型和端点信息。'
        />
        <ResponsiveDocsImage
          {...screenshots[6]}
          alt='启用导入后的 CC Switch 配置'
          caption='导入成功后启用配置并重启 Agent。'
        />
      </section>

      <section id='manual-config' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>2.2 手动配置 CC Switch</h2>
        <NumberedSteps
          items={[
            '打开 CC Switch，点击右上角加号添加模型供应商。',
            '选择统一供应商或 Codex 供应商，自定义配置，然后下滑填写具体配置。',
            '按照图示填写信息，红框标注项为必填，完成后点击添加。',
            '点击启用，并重启对应的 Agent 客户端。',
          ]}
        />
        <ResponsiveDocsImage
          {...screenshots[7]}
          alt='在 CC Switch 中添加供应商'
          caption='点击加号添加模型供应商。'
        />
        <ResponsiveDocsImage
          {...screenshots[8]}
          alt='选择 CC Switch 供应商类型'
          caption='选择与目标客户端匹配的供应商类型。'
        />
        <ResponsiveDocsImage
          {...screenshots[9]}
          alt='填写 CC Switch 模型配置'
          caption='填写红框标注的必需配置。'
        />
        <ResponsiveDocsImage
          {...screenshots[10]}
          alt='启用 CC Switch 模型配置'
          caption='保存后启用配置并重启客户端。'
        />
        <div className='mt-5'>
          <Button
            nativeButton={false}
            variant='outline'
            render={<Link to='/keys' />}
          >
            不知道 API Key？打开 API Key 页面
          </Button>
        </div>
      </section>

      <section id='advanced' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>高级用法</h2>
        <div className='mt-4 space-y-3'>
          <h3 className='text-lg font-semibold'>
            <a
              href='https://ccswitch.io/zh/tutorials/claude-codex-routing-guide'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary underline-offset-4 hover:underline'
            >
              在 Claude Code 中使用 ChatGPT
            </a>
          </h3>
          <h3 className='text-lg font-semibold'>
            <a
              href='https://ccswitch.io/zh/tutorials/codex-claude-routing-guide'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary underline-offset-4 hover:underline'
            >
              在 Codex 中使用 Claude 模型
            </a>
          </h3>
          <h3 className='text-lg font-semibold'>
            <a
              href='https://ccswitch.io/zh/tutorials'
              target='_blank'
              rel='noopener noreferrer'
              className='text-primary underline-offset-4 hover:underline'
            >
              更多高级用法请见 CC Switch 官网
            </a>
          </h3>
        </div>
      </section>
    </DocsShell>
  )
}
