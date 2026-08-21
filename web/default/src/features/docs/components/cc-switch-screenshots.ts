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
import type { GuideScreenshot } from './guide-steps'

const IMAGE_BASE = '/static/image/docs/cc-switch'

export const CC_SWITCH_SCREENSHOTS = {
  download: {
    largeSrc: `${IMAGE_BASE}/step-01-1520.f704c73a6eb1.webp`,
    smallSrc: `${IMAGE_BASE}/step-01-760.8f5ed2222c27.webp`,
    width: 1859,
    height: 1370,
    alt: 'CC Switch 发布页安装包列表',
    caption: '在发布页选择适合自己设备的安装包。',
  },
  apiKey: {
    largeSrc: `${IMAGE_BASE}/step-02-1520.ebee24073ddb.webp`,
    smallSrc: `${IMAGE_BASE}/step-02-760.74e196f40b56.webp`,
    width: 3420,
    height: 1902,
    alt: '创建 API Key',
    caption: '在 API Key 页面创建用于导入的密钥。',
  },
  apiKeyDetails: {
    largeSrc: `${IMAGE_BASE}/step-03-1520.5e3bd8e9b8b5.webp`,
    smallSrc: `${IMAGE_BASE}/step-03-760.877222304998.webp`,
    width: 3420,
    height: 1902,
    alt: '填写 API Key 名称和分组',
    caption: '填写密钥名称并选择分组。',
  },
  importEntry: {
    largeSrc: `${IMAGE_BASE}/step-04-1520.03c352daa9ff.webp`,
    smallSrc: `${IMAGE_BASE}/step-04-760.bcbb3eb9f9c3.webp`,
    width: 3420,
    height: 1902,
    alt: '打开 CC Switch 导入入口',
    caption: '创建密钥后打开 CC Switch 导入入口。',
  },
  importDialog: {
    largeSrc: `${IMAGE_BASE}/step-05-1520.0d6281dc4d60.webp`,
    smallSrc: `${IMAGE_BASE}/step-05-760.4d0ea61f3015.webp`,
    width: 3420,
    height: 1902,
    alt: '打开 CC Switch 导入对话框',
    caption: '允许浏览器打开 ccswitch:// 链接后查看导入对话框。',
  },
  confirmImport: {
    largeSrc: `${IMAGE_BASE}/step-06-1520.f89c84dae388.webp`,
    smallSrc: `${IMAGE_BASE}/step-06-760.d86b028eb18a.webp`,
    width: 2000,
    height: 1302,
    alt: '确认 CC Switch 导入信息',
    caption: '导入前检查服务商、模型和端点信息。',
  },
  imported: {
    largeSrc: `${IMAGE_BASE}/step-07-1520.24a51ac729c8.webp`,
    smallSrc: `${IMAGE_BASE}/step-07-760.ae74559dcaff.webp`,
    width: 2000,
    height: 1302,
    alt: '启用导入后的 CC Switch 配置',
    caption: '导入成功后启用配置并重启 Agent。',
  },
  addProvider: {
    largeSrc: `${IMAGE_BASE}/step-08-1520.e53705f90a86.webp`,
    smallSrc: `${IMAGE_BASE}/step-08-760.8941d8843074.webp`,
    width: 2000,
    height: 1302,
    alt: '在 CC Switch 中添加供应商',
    caption: '点击加号添加模型供应商。',
  },
  providerType: {
    largeSrc: `${IMAGE_BASE}/step-09-1520.786f116444c5.webp`,
    smallSrc: `${IMAGE_BASE}/step-09-760.e196babd0bd0.webp`,
    width: 2000,
    height: 1302,
    alt: '选择 CC Switch 供应商类型',
    caption: '选择与目标客户端匹配的供应商类型。',
  },
  providerConfig: {
    largeSrc: `${IMAGE_BASE}/step-10-1520.ba6f0311bcdd.webp`,
    smallSrc: `${IMAGE_BASE}/step-10-760.e97e4ffcffde.webp`,
    width: 2000,
    height: 1302,
    alt: '填写 CC Switch 模型配置',
    caption: '填写红框标注的必需配置。',
  },
  enabled: {
    largeSrc: `${IMAGE_BASE}/step-11-1520.0283546e75b8.webp`,
    smallSrc: `${IMAGE_BASE}/step-11-760.56feb3ad9e90.webp`,
    width: 2000,
    height: 1302,
    alt: '启用 CC Switch 模型配置',
    caption: '保存后启用配置并重启客户端。',
  },
} as const satisfies Record<string, GuideScreenshot>
