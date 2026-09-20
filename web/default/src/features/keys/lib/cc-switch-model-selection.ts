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
import type {
  ApiKeyModel,
  ApiKeyModelEndpoint,
  ApiKeyModelsResult,
} from '../api'
import type { CCSwitchApp } from './model-catalog'

const APP_ENDPOINT_TYPES: Record<CCSwitchApp, readonly ApiKeyModelEndpoint[]> =
  {
    claude: ['anthropic'],
    // CC Switch Codex imports can target both native Responses channels and
    // regular OpenAI-compatible channels.
    codex: ['openai-response', 'openai'],
    gemini: ['gemini'],
  }

function supportsAppEndpoint(model: ApiKeyModel, app: CCSwitchApp): boolean {
  return APP_ENDPOINT_TYPES[app].some((endpoint) =>
    model.supportedEndpointTypes.includes(endpoint)
  )
}

export function selectSuggestedModel(
  models: ApiKeyModelsResult | undefined,
  app: CCSwitchApp,
  preferredModelOrder: readonly string[]
): string {
  if (!models?.success) return ''

  // The catalog is already scoped to the API key's groups. Preference order
  // must therefore be applied to the returned model IDs independently of
  // which channel endpoint exposed each model.
  const preferredModel = preferredModelOrder.find((preferred) =>
    models.models.some((model) => model.id === preferred)
  )
  if (preferredModel) return preferredModel

  const preferredFamilies: Record<CCSwitchApp, string[]> = {
    claude: ['claude', 'anthropic'],
    codex: ['gpt-', 'codex', 'o1', 'o3', 'o4'],
    gemini: ['gemini', 'google'],
  }
  const family = preferredFamilies[app]
  const appModel = models.models.find((model) => {
    const id = model.id.toLowerCase()
    const owner = model.ownedBy?.toLowerCase() ?? ''
    return (
      supportsAppEndpoint(model, app) &&
      family.some(
        (candidate) => id.includes(candidate) || owner.includes(candidate)
      )
    )
  })
  if (appModel) return appModel.id

  const generalModel = models.models.find((model) =>
    model.supportedEndpointTypes.some(
      (endpoint) =>
        endpoint === 'openai' ||
        endpoint === 'anthropic' ||
        endpoint === 'gemini' ||
        endpoint === 'openai-response'
    )
  )
  return generalModel?.id ?? ''
}
