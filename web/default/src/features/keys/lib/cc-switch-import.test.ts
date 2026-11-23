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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  buildCCSwitchImportUrl,
  resolveCCSwitchDefaultName,
} from './cc-switch-import'
import { CC_SWITCH_APPS, type CCSwitchApp } from './model-catalog'

describe('CC Switch import URL', () => {
  test('uses the API key name as the default provider name', () => {
    assert.equal(
      resolveCCSwitchDefaultName(' Production API ', 'My Claude'),
      'Production API'
    )
    assert.equal(resolveCCSwitchDefaultName('', 'My Codex'), 'My Codex')
    assert.equal(resolveCCSwitchDefaultName(null, 'My Gemini'), 'My Gemini')
  })

  test('encodes every application endpoint and model mapping correctly', () => {
    const providers: Record<CCSwitchApp, readonly [string, string]> = {
      claude: ['', 'anthropic'],
      'claude-desktop': ['', 'anthropic'],
      codex: ['/v1', 'openai_responses'],
      gemini: ['', 'gemini_native'],
      grokbuild: ['/v1', 'openai_responses'],
      agy: ['', 'gemini_native'],
      copilot: ['/v1', 'openai_chat'],
      opencode: ['/v1', 'openai_chat'],
      openclaw: ['/v1', 'openai_chat'],
      hermes: ['/v1', 'openai_chat'],
      pi: ['/v1', 'openai_chat'],
      mcode: ['/v1', 'openai_chat'],
      kimi: ['/v1', 'openai_chat'],
      dsh: ['/v1', 'openai_chat'],
      omp: ['/v1', 'openai_chat'],
      workbuddy: ['/v1/chat/completions', 'openai_chat'],
    }

    for (const app of Object.keys(CC_SWITCH_APPS) as CCSwitchApp[]) {
      const [endpointPath, apiFormat] = providers[app]
      const apiKey = app === 'codex' ? 'sk-existing-token' : 'raw-token'
      const url = new URL(
        buildCCSwitchImportUrl({
          apiKey,
          app,
          models: {
            model: ' gateway-model ',
            haikuModel: ' small-model ',
            sonnetModel: '',
            opusModel: 'large-model',
          },
          name: '配置 & test',
          serverAddress: ' https://api.example.com/ ',
        })
      )

      assert.equal(url.protocol, 'ccswitch:')
      assert.equal(url.hostname, 'v1')
      assert.equal(url.pathname, '/import')
      assert.equal(url.searchParams.get('resource'), 'provider')
      assert.equal(url.searchParams.get('app'), app)
      assert.equal(url.searchParams.get('name'), '配置 & test')
      assert.equal(
        url.searchParams.get('endpoint'),
        `https://api.example.com${endpointPath}`
      )
      assert.equal(url.searchParams.get('homepage'), 'https://api.example.com')
      assert.equal(
        url.searchParams.get('apiKey'),
        app === 'codex' ? 'sk-existing-token' : 'sk-raw-token'
      )
      assert.equal(url.searchParams.get('apiFormat'), apiFormat)
      assert.equal(url.searchParams.get('enabled'), 'true')
      assert.equal(url.searchParams.get('model'), 'gateway-model')
      const supportsClaudeAliases = app === 'claude' || app === 'claude-desktop'
      assert.equal(
        url.searchParams.get('haikuModel'),
        supportsClaudeAliases ? 'small-model' : null
      )
      assert.equal(url.searchParams.has('sonnetModel'), false)
      assert.equal(
        url.searchParams.get('opusModel'),
        supportsClaudeAliases ? 'large-model' : null
      )
      assert.equal(url.searchParams.has('group'), false)
      assert.equal(url.searchParams.has('group_candidates'), false)
    }
  })
})
