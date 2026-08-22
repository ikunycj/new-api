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

export const DOCS_LOCALE = 'zhCN'

const DOCS_GROUPS = [
  { id: 'overview', label: '概览' },
  { id: 'tools', label: '工具使用与接入' },
  { id: 'api-start', label: 'API 集成' },
] as const

export const DOCS_ROUTES = [
  {
    id: 'introduction',
    group: 'overview',
    label: 'AI 使用指南',
    path: '/docs',
    file: 'docs/index.html',
  },
  {
    id: 'quick-start',
    group: 'overview',
    label: '快速开始',
    path: '/docs/quick-start',
    file: 'docs/quick-start.html',
  },
  {
    id: 'model-pricing',
    group: 'overview',
    label: '模型定价',
    path: '/docs/model-pricing',
    file: 'docs/model-pricing.html',
  },
  {
    id: 'referral-rewards',
    group: 'overview',
    label: '推荐奖励',
    path: '/docs/referral-rewards',
    file: 'docs/referral-rewards.html',
  },
  {
    id: 'error-self-check',
    group: 'overview',
    label: '错误自查指南',
    path: '/docs/error-self-check',
    file: 'docs/error-self-check.html',
  },
  {
    id: 'integration-guide',
    group: 'tools',
    label: '集成指南',
    path: '/docs/integrations',
    file: 'docs/integrations.html',
  },
  {
    id: 'cc-switch',
    group: 'tools',
    label: 'AI Agent 与 CC Switch',
    path: '/docs/tools/cc-switch',
    file: 'docs/tools/cc-switch.html',
  },
  {
    id: 'codex',
    group: 'tools',
    label: 'Codex',
    path: '/docs/tools/codex',
    file: 'docs/tools/codex.html',
  },
  {
    id: 'claude-code',
    group: 'tools',
    label: 'Claude Code',
    path: '/docs/tools/claude-code',
    file: 'docs/tools/claude-code.html',
  },
  {
    id: 'openclaw',
    group: 'tools',
    label: 'OpenClaw',
    path: '/docs/tools/openclaw',
    file: 'docs/tools/openclaw.html',
  },
  {
    id: 'hermes',
    group: 'tools',
    label: 'Hermes',
    path: '/docs/tools/hermes',
    file: 'docs/tools/hermes.html',
  },
  {
    id: 'opencode',
    group: 'tools',
    label: 'OpenCode',
    path: '/docs/tools/opencode',
    file: 'docs/tools/opencode.html',
  },
  {
    id: 'gemini',
    group: 'tools',
    label: 'Gemini CLI',
    path: '/docs/tools/gemini',
    file: 'docs/tools/gemini.html',
  },
  {
    id: 'api-integration',
    group: 'api-start',
    label: 'API 模型接口',
    path: '/docs/api/integration',
    file: 'docs/api/integration.html',
  },
  {
    id: 'api-text-chat',
    group: 'api-start',
    label: '文本与对话',
    path: '/docs/api/text-chat',
    file: 'docs/api/text-chat.html',
  },
  {
    id: 'api-multimodal',
    group: 'api-start',
    label: '多模态接口',
    path: '/docs/api/multimodal',
    file: 'docs/api/multimodal.html',
  },
  {
    id: 'api-compatibility',
    group: 'api-start',
    label: '兼容性与限制',
    path: '/docs/api/compatibility',
    file: 'docs/api/compatibility.html',
  },
] as const

export type DocsPageId = (typeof DOCS_ROUTES)[number]['id']
export type DocsRoutePath = (typeof DOCS_ROUTES)[number]['path']
export type DocsNavigationItem = (typeof DOCS_ROUTES)[number]

export type DocsNavigationGroup = {
  id: (typeof DOCS_GROUPS)[number]['id']
  label: string
  items: DocsNavigationItem[]
}

export const DOCS_NAVIGATION_GROUPS: DocsNavigationGroup[] = DOCS_GROUPS.map(
  (group) => ({
    ...group,
    items: DOCS_ROUTES.filter((route) => route.group === group.id),
  })
)
