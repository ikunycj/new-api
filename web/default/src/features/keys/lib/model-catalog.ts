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
import type { ApiKeyModel, ApiKeyModelEndpoint } from '../api'

export type CCSwitchModelField =
  | 'model'
  | 'haikuModel'
  | 'sonnetModel'
  | 'opusModel'

type CCSwitchAppConfig = {
  label: string
  modelFields: readonly { key: CCSwitchModelField; required: boolean }[]
  apiFormat: 'anthropic' | 'openai_chat' | 'openai_responses' | 'gemini_native'
  endpointPath: '' | '/v1' | '/v1/chat/completions'
}

const PRIMARY_MODEL_FIELDS = [{ key: 'model', required: true }] as const
const CLAUDE_MODEL_FIELDS = [
  ...PRIMARY_MODEL_FIELDS,
  { key: 'haikuModel', required: false },
  { key: 'sonnetModel', required: false },
  { key: 'opusModel', required: false },
] as const

const ANTHROPIC = {
  modelFields: CLAUDE_MODEL_FIELDS,
  apiFormat: 'anthropic',
  endpointPath: '',
} as const
const OPENAI_RESPONSES = {
  modelFields: PRIMARY_MODEL_FIELDS,
  apiFormat: 'openai_responses',
  endpointPath: '/v1',
} as const
const GEMINI_NATIVE = {
  modelFields: PRIMARY_MODEL_FIELDS,
  apiFormat: 'gemini_native',
  endpointPath: '',
} as const
const OPENAI_CHAT = {
  modelFields: PRIMARY_MODEL_FIELDS,
  apiFormat: 'openai_chat',
  endpointPath: '/v1',
} as const

// The CC Switch v4.0.39 provider parser defines the deeplink application IDs.
export const CC_SWITCH_APPS = {
  claude: { label: 'Claude Code', ...ANTHROPIC },
  codex: { label: 'Codex', ...OPENAI_RESPONSES },
  gemini: { label: 'Gemini CLI', ...GEMINI_NATIVE },
  'claude-desktop': { label: 'Claude Desktop', ...ANTHROPIC },
  grokbuild: { label: 'Grok Build', ...OPENAI_RESPONSES },
  agy: { label: 'Antigravity CLI', ...GEMINI_NATIVE },
  copilot: { label: 'Copilot CLI', ...OPENAI_CHAT },
  opencode: { label: 'OpenCode', ...OPENAI_CHAT },
  openclaw: { label: 'OpenClaw', ...OPENAI_CHAT },
  hermes: { label: 'Hermes', ...OPENAI_CHAT },
  pi: { label: 'Pi', ...OPENAI_CHAT },
  mcode: { label: 'MiniMax Code', ...OPENAI_CHAT },
  kimi: { label: 'Kimi Code', ...OPENAI_CHAT },
  dsh: { label: 'DeepSeek Harness', ...OPENAI_CHAT },
  omp: { label: 'Oh My Pi', ...OPENAI_CHAT },
  workbuddy: {
    label: 'WorkBuddy',
    ...OPENAI_CHAT,
    endpointPath: '/v1/chat/completions',
  },
} as const satisfies Record<string, CCSwitchAppConfig>

export type CCSwitchApp = keyof typeof CC_SWITCH_APPS

export type CCSwitchModelGroupKind = 'protocol' | 'gateway' | 'other'

export type CCSwitchModelGroup = {
  kind: CCSwitchModelGroupKind
  models: ApiKeyModel[]
}

const SPECIALIZED_ENDPOINTS = new Set<ApiKeyModelEndpoint>([
  'embeddings',
  'image-generation',
  'jina-rerank',
  'openai-video',
])

const CLAUDE_OWNERS = ['anthropic', 'claude']
const CODEX_OWNERS = ['openai']
const GEMINI_OWNERS = ['gemini', 'google']

const PROTOCOL_ENDPOINTS: Record<
  CCSwitchAppConfig['apiFormat'],
  ApiKeyModelEndpoint
> = {
  anthropic: 'anthropic',
  openai_chat: 'openai',
  openai_responses: 'openai-response',
  gemini_native: 'gemini',
}

export function getCCSwitchModelEndpoint(
  app: CCSwitchApp
): ApiKeyModelEndpoint {
  return PROTOCOL_ENDPOINTS[CC_SWITCH_APPS[app].apiFormat]
}

function includesEndpoint(
  model: ApiKeyModel,
  endpoint: ApiKeyModelEndpoint
): boolean {
  return model.supportedEndpointTypes.includes(endpoint)
}

function hasAnyEndpoint(
  model: ApiKeyModel,
  endpoints: ApiKeyModelEndpoint[]
): boolean {
  return endpoints.some((endpoint) => includesEndpoint(model, endpoint))
}

function isSpecializedModel(model: ApiKeyModel): boolean {
  return model.supportedEndpointTypes.some((endpoint) =>
    SPECIALIZED_ENDPOINTS.has(endpoint)
  )
}

function isCompactOnlyModel(model: ApiKeyModel): boolean {
  return (
    model.supportedEndpointTypes.length > 0 &&
    model.supportedEndpointTypes.every(
      (endpoint) => endpoint === 'openai-response-compact'
    )
  )
}

function getModelGroupKind(
  model: ApiKeyModel,
  app: CCSwitchApp
): CCSwitchModelGroupKind {
  if (isSpecializedModel(model) || isCompactOnlyModel(model)) return 'other'

  const endpoint = getCCSwitchModelEndpoint(app)
  if (includesEndpoint(model, endpoint)) return 'protocol'

  if (hasAnyEndpoint(model, ['openai', 'anthropic', 'gemini'])) {
    return 'gateway'
  }
  if (endpoint === 'openai' && includesEndpoint(model, 'openai-response')) {
    return 'gateway'
  }
  return 'other'
}

function matchesOwner(model: ApiKeyModel, owners: string[]): boolean {
  const owner = model.ownedBy?.toLowerCase() ?? ''
  return owners.some((candidate) => owner.includes(candidate))
}

export function matchesCCSwitchModelFamily(
  model: ApiKeyModel,
  app: CCSwitchApp
): boolean {
  const modelId = model.id.toLowerCase()

  if (app === 'claude' || app === 'claude-desktop') {
    return modelId.includes('claude') || matchesOwner(model, CLAUDE_OWNERS)
  }

  if (app === 'codex') {
    return (
      modelId.startsWith('gpt-') ||
      modelId.startsWith('codex') ||
      /^o[134](?:-|$)/.test(modelId) ||
      matchesOwner(model, CODEX_OWNERS)
    )
  }

  switch (app) {
    case 'gemini':
    case 'agy':
      return modelId.includes('gemini') || matchesOwner(model, GEMINI_OWNERS)
    case 'grokbuild':
      return modelId.includes('grok') || matchesOwner(model, ['xai'])
    case 'mcode':
      return modelId.includes('minimax') || matchesOwner(model, ['minimax'])
    case 'kimi':
      return (
        modelId.includes('kimi') || matchesOwner(model, ['kimi', 'moonshot'])
      )
    case 'dsh':
      return modelId.includes('deepseek') || matchesOwner(model, ['deepseek'])
    default:
      return false
  }
}

function matchesFieldFamily(
  model: ApiKeyModel,
  field: CCSwitchModelField
): boolean {
  if (field === 'model') return false
  const family = field.replace('Model', '').toLowerCase()
  return model.id.toLowerCase().includes(family)
}

function getSearchRank(model: ApiKeyModel, search: string): number {
  if (!search) return 0

  const searchableValues = [
    model.id,
    model.ownedBy ?? '',
    ...model.supportedEndpointTypes,
  ].map((value) => value.toLowerCase())

  if (searchableValues.some((value) => value === search)) return 0
  if (searchableValues.some((value) => value.startsWith(search))) return 1
  if (searchableValues.some((value) => value.includes(search))) return 2
  return Number.POSITIVE_INFINITY
}

export function groupCCSwitchModels(
  models: ApiKeyModel[],
  app: CCSwitchApp,
  field: CCSwitchModelField,
  search = ''
): CCSwitchModelGroup[] {
  const normalizedSearch = search.trim().toLowerCase()
  const indexedModels = models.map((model, index) => ({ model, index }))
  const kinds: CCSwitchModelGroupKind[] = ['protocol', 'gateway', 'other']

  return kinds.map((kind) => {
    const groupedModels = indexedModels
      .filter(({ model }) => getModelGroupKind(model, app) === kind)
      .map(({ model, index }) => ({
        appRank: matchesCCSwitchModelFamily(model, app) ? 0 : 1,
        fieldRank: matchesFieldFamily(model, field) ? 0 : 1,
        index,
        model,
        searchRank: getSearchRank(model, normalizedSearch),
      }))
      .filter(({ searchRank }) => Number.isFinite(searchRank))
      .sort((left, right) => {
        if (left.searchRank !== right.searchRank) {
          return left.searchRank - right.searchRank
        }
        if (left.fieldRank !== right.fieldRank) {
          return left.fieldRank - right.fieldRank
        }
        if (left.appRank !== right.appRank) {
          return left.appRank - right.appRank
        }
        return left.index - right.index
      })
      .map(({ model }) => model)

    return { kind, models: groupedModels }
  })
}
