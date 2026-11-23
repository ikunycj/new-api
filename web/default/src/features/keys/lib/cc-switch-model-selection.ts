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
import type { ApiKeyModelsResult } from '../api'
import {
  getCCSwitchModelEndpoint,
  matchesCCSwitchModelFamily,
  type CCSwitchApp,
} from './model-catalog'

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

  const endpoint = getCCSwitchModelEndpoint(app)
  const compatibleModels = models.models.filter(
    (model) =>
      model.supportedEndpointTypes.includes(endpoint) ||
      // Codex can also use OpenAI-compatible channels through the gateway.
      (app === 'codex' && model.supportedEndpointTypes.includes('openai'))
  )
  const appModel = compatibleModels.find((model) =>
    matchesCCSwitchModelFamily(model, app)
  )
  if (appModel) return appModel.id
  if (app !== 'claude' && app !== 'codex' && app !== 'gemini') {
    const protocolModel = compatibleModels[0]
    if (protocolModel) return protocolModel.id
  }

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
