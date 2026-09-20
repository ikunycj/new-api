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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { ApiKeyModel, ApiKeyModelsResult } from '../api'
import { selectSuggestedModel } from './cc-switch-model-selection'

function model(
  id: string,
  supportedEndpointTypes: ApiKeyModel['supportedEndpointTypes']
): ApiKeyModel {
  return { id, supportedEndpointTypes }
}

function result(models: ApiKeyModel[]): ApiKeyModelsResult {
  return { success: true, models }
}

describe('CC Switch suggested model selection', () => {
  test('uses a preferred model by ID regardless of its endpoint metadata', () => {
    const models = result([
      model('gpt-5.6-sol', ['openai']),
      model('gpt-6-astra', ['anthropic']),
      model('codex-native', ['openai-response']),
    ])

    assert.equal(
      selectSuggestedModel(models, 'codex', ['gpt-6-astra', 'gpt-5.6-sol']),
      'gpt-6-astra'
    )
  })

  test('continues to prefer native Responses models when they are available', () => {
    const models = result([
      model('gpt-6-astra', ['openai']),
      model('gpt-5.3-codex', ['openai-response']),
    ])

    assert.equal(
      selectSuggestedModel(models, 'codex', ['gpt-5.3-codex']),
      'gpt-5.3-codex'
    )
  })
})
