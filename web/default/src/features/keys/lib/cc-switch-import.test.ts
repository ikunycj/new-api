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
import { runInNewContext } from 'node:vm'

import {
  buildCCSwitchImportUrl,
  getCCSwitchUsageBaseUrl,
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

  test('encodes a scoped account usage script and the configured logo', () => {
    const url = new URL(
      buildCCSwitchImportUrl({
        apiKey: 'inference-key',
        app: 'workbuddy',
        logoUrl: '/logo.webp',
        models: { model: 'gateway-model' },
        name: 'Example API',
        serverAddress: 'https://api.example.com',
        usageAuthorization: { access_token: 'usage-credential', user_id: 42 },
      })
    )

    const params = url.searchParams
    assert.equal(
      params.get('endpoint'),
      'https://api.example.com/v1/chat/completions'
    )
    assert.equal(params.get('iconUrl'), 'https://api.example.com/logo.webp')
    assert.equal(params.get('usageEnabled'), 'true')
    assert.equal(params.get('usageBaseUrl'), 'https://api.example.com')
    assert.equal(params.get('usageAccessToken'), 'usage-credential')
    assert.equal(params.get('usageUserId'), '42')
    assert.equal(params.has('usageApiKey'), false)
    const encodedScript = params.get('usageScript')
    assert.ok(encodedScript)
    const code = atob(encodedScript)
      .replaceAll('{{baseUrl}}', params.get('usageBaseUrl') ?? '')
      .replaceAll('{{accessToken}}', params.get('usageAccessToken') ?? '')
      .replaceAll('{{userId}}', params.get('usageUserId') ?? '')
    const script = runInNewContext(code)
    assert.deepEqual(structuredClone(script.request), {
      url: 'https://api.example.com/api/usage/account/',
      method: 'GET',
      headers: {
        Authorization: 'Bearer usage-credential',
        'New-Api-User': '42',
      },
    })
    const success = script.extractor({
      success: true,
      data: { remaining: 12.5, used: 3.75, unit: 'USD' },
    })
    assert.deepEqual(structuredClone(success), {
      remaining: 12.5,
      used: 3.75,
      unit: 'USD',
    })
    const failure = script.extractor({
      success: false,
      message: 'Unauthorized',
    })
    assert.deepEqual(structuredClone(failure), {
      isValid: false,
      invalidMessage: 'Unauthorized',
    })
  })

  test('only supplies usage authorization to HTTPS or local HTTP endpoints', () => {
    const cases = [
      ['https://api.example.com:9443/proxy/', 'https://api.example.com:9443'],
      ['http://localhost:5173', 'http://localhost:5173'],
      ['http://127.0.0.2:3000', 'http://127.0.0.2:3000'],
      ['http://[::1]:3000', 'http://[::1]:3000'],
      ['http://api.example.com', null],
      ['http://127.0.0.1.example.com', null],
      ['http://192.168.1.8', null],
      ['not a URL', null],
    ] as const

    for (const [serverAddress, expected] of cases) {
      assert.equal(
        getCCSwitchUsageBaseUrl(serverAddress),
        expected,
        serverAddress
      )
      const params = new URL(
        buildCCSwitchImportUrl({
          apiKey: 'inference-key',
          app: 'codex',
          models: {},
          name: 'Example API',
          serverAddress,
          usageAuthorization: { access_token: 'usage-credential', user_id: 42 },
        })
      ).searchParams
      assert.equal(params.get('usageBaseUrl'), expected, serverAddress)
      assert.equal(
        params.has('usageAccessToken'),
        expected !== null,
        serverAddress
      )
      assert.equal(params.has('usageScript'), expected !== null, serverAddress)
    }
  })

  test('omits icon URLs CC Switch cannot fetch', () => {
    for (const logoUrl of [
      'data:image/svg+xml;base64,abcd',
      'http://cdn.example.com/logo.webp',
      'https://cdn.example.com:8443/logo.webp',
      'https://example:password@cdn.example.com/logo.webp',
      'https://127.0.0.1/logo.webp',
      'https://loopback.localhost/logo.webp',
    ]) {
      const params = new URL(
        buildCCSwitchImportUrl({
          apiKey: 'inference-key',
          app: 'codex',
          logoUrl,
          models: {},
          name: 'Example API',
          serverAddress: 'https://api.example.com',
        })
      ).searchParams
      assert.equal(params.has('iconUrl'), false, logoUrl)
      assert.equal(params.has('usageEnabled'), false)
    }
  })
})
