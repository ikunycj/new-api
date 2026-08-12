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
import type { ComponentType } from 'react'

import { DocsApiIntegration } from './ai-model-api'
import { DocsApiCompatibility } from './api-compatibility'
import { DocsApiMultimodal } from './api-multimodal'
import { DocsApiTextChat } from './api-text-chat'
import { DocsCcSwitch } from './cc-switch-guide'
import { DocsClaudeCode } from './claude-code-guide'
import { DocsCodex } from './codex-guide'
import type { DocsRoutePath } from './docs-config'
import { DocsGemini } from './gemini-guide'
import { DocsHermes } from './hermes-guide'
import { DocsIntegrationGuide } from './integration-guide'
import { DocsModelPricing } from './model-pricing'
import { DocsOpenClaw } from './openclaw-guide'
import { DocsOpenCode } from './opencode-guide'
import { DocsQuickStart } from './quick-start'

export const DOCS_PRERENDER_SOURCES = {
  '/docs/quick-start': DocsQuickStart,
  '/docs/model-pricing': DocsModelPricing,
  '/docs/integrations': DocsIntegrationGuide,
  '/docs/tools/cc-switch': DocsCcSwitch,
  '/docs/tools/claude-code': DocsClaudeCode,
  '/docs/tools/codex': DocsCodex,
  '/docs/tools/gemini': DocsGemini,
  '/docs/tools/hermes': DocsHermes,
  '/docs/tools/openclaw': DocsOpenClaw,
  '/docs/tools/opencode': DocsOpenCode,
  '/docs/api/integration': DocsApiIntegration,
  '/docs/api/text-chat': DocsApiTextChat,
  '/docs/api/multimodal': DocsApiMultimodal,
  '/docs/api/compatibility': DocsApiCompatibility,
} satisfies Record<DocsRoutePath, ComponentType>
