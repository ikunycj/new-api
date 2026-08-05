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
  Download04Icon,
  InformationCircleIcon,
  Key01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import { DocsShell } from './components/docs-shell'
import { NumberedSteps } from './components/numbered-steps'
import { ResponsiveDocsImage } from './components/responsive-docs-image'

const CC_SWITCH_RELEASES_URL =
  'https://github.com/farion1231/cc-switch/releases'
const CC_SWITCH_DOCS_URL = 'https://ccswitch.io/zh/docs'
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
      title='CC Switch 一键导入'
      description='使用 CC Switch 统一管理 Claude Code、Codex、Gemini CLI、OpenCode、OpenClaw 和 Hermes 等工具的中转站配置。'
      toc={[
        { id: 'what-is-cc-switch', label: 'CC Switch 是什么' },
        { id: 'download', label: '下载与安装' },
        { id: 'one-click-import', label: '一键导入配置' },
        { id: 'manual-config', label: '手动配置' },
        { id: 'advanced', label: '高级用法' },
      ]}
    >
      <section id='what-is-cc-switch' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>CC Switch 是什么</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          CC Switch 是一款跨平台桌面应用，专为使用 AI
          编程工具的开发者设计。它可以统一管理 Claude Code、Claude
          Desktop、Codex、Gemini CLI、OpenCode、OpenClaw 和 Hermes
          等受管应用的配置。
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          当你需要在官方服务商和中转服务商之间切换，或需要给不同工具配置不同协议时，CC
          Switch
          可以减少手动编辑配置文件的次数。它也能集中保存多个供应商并快速启用其中一个。
        </p>
        <a
          href={CC_SWITCH_DOCS_URL}
          target='_blank'
          rel='noopener noreferrer'
          className='text-foreground mt-4 inline-block underline underline-offset-4'
        >
          查看 CC Switch 官方文档
        </a>
      </section>

      <section id='download' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>下载与安装</h2>
        <NumberedSteps
          items={[
            '打开 CC Switch 发布页，选择最新版本。',
            '在页面底部的安装包列表中选择与操作系统和 CPU 架构匹配的文件；不确定时可以把设备信息交给 AI 判断。',
            '按系统提示完成安装，并首次启动 CC Switch 以注册 ccswitch:// 链接协议。',
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
                href={CC_SWITCH_DOCS_URL}
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

      <section id='one-click-import' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>一键导入配置</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          一键导入从本站 API Key 页面发起，会把服务商、Base URL、API
          Key、模型和目标客户端字段传递给 CC
          Switch。只在自己的设备上确认导入链接。
        </p>
        <NumberedSteps
          items={[
            '登录本站，进入 API Key 页面并创建密钥。',
            '填写密钥名称，选择分组后保存密钥。',
            '在密钥行的操作菜单中点击 CC Switch 图标。',
            '选择目标 Agent 客户端和对应模型。目前一键导入覆盖 Claude、Codex、Gemini，其他客户端可以使用下面的手动配置。',
            '确认导入信息无误后，允许浏览器打开 ccswitch:// 链接并点击导入。',
            '在 CC Switch 中启用导入的供应商，然后重启对应 Agent 客户端使配置生效。',
          ]}
        />
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
          {...screenshots[4]}
          alt='选择 CC Switch 目标客户端'
          caption='选择需要导入配置的 Agent 客户端。'
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
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
          <AlertTitle>导入链接包含敏感信息</AlertTitle>
          <AlertDescription>
            导入链接可能包含 API Key
            和模型配置。不要把链接转发给他人，也不要在公共聊天或截图中暴露完整密钥；如果链接被不可信设备打开，请立即撤销并重新创建
            API Key。
          </AlertDescription>
        </Alert>
      </section>

      <section id='manual-config' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>手动配置</h2>
        <NumberedSteps
          items={[
            '打开 CC Switch，点击右上角加号添加模型供应商。',
            '选择统一供应商或 Codex 供应商，自定义供应商名称，并填写与模型接口匹配的 Base URL。',
            '填写 API Key、准确模型 ID 和其他必需字段，然后点击添加。',
            '点击启用，并重启对应的 Agent 客户端。',
          ]}
        />
        <p className='text-muted-foreground mt-4 leading-7'>
          Chat Completions 和 Responses 通常使用以 <code>/v1</code> 结尾的 Base
          URL；Claude Messages 和 Gemini
          原生接口的客户端通常使用服务根地址。请以模型定价页面的端点类型为准。
        </p>
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
        <ul className='text-muted-foreground mt-3 list-disc space-y-2 pl-5 leading-7'>
          <li>
            <a
              href='https://ccswitch.io/zh/tutorials/claude-codex-routing-guide'
              target='_blank'
              rel='noopener noreferrer'
              className='text-foreground underline underline-offset-4'
            >
              在 Claude Code 中使用 ChatGPT
            </a>
          </li>
          <li>
            <a
              href='https://ccswitch.io/zh/tutorials/codex-claude-routing-guide'
              target='_blank'
              rel='noopener noreferrer'
              className='text-foreground underline underline-offset-4'
            >
              在 Codex 中使用 Claude 模型
            </a>
          </li>
          <li>
            <a
              href='https://ccswitch.io/zh/tutorials'
              target='_blank'
              rel='noopener noreferrer'
              className='text-foreground underline underline-offset-4'
            >
              更多高级用法请见 CC Switch 官方教程
            </a>
          </li>
        </ul>
      </section>
    </DocsShell>
  )
}
