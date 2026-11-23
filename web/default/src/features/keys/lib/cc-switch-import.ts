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
  CC_SWITCH_APPS,
  type CCSwitchApp,
  type CCSwitchModelField,
} from './model-catalog'

const accountUsageScript = `({
  request: {
    url: "{{baseUrl}}/api/usage/account/",
    method: "GET",
    headers: {
      "Authorization": "Bearer {{accessToken}}",
      "New-Api-User": "{{userId}}"
    }
  },
  extractor: function (response) {
    if (response.success && response.data) {
      return {
        remaining: response.data.remaining,
        used: response.data.used,
        unit: response.data.unit
      };
    }
    return { isValid: false, invalidMessage: response.message };
  }
})`

export type CCSwitchUsageAuthorization = {
  access_token: string
  user_id: number
}

type CCSwitchImportConfig = {
  apiKey: string
  app: CCSwitchApp
  logoUrl?: string | null
  models: Partial<Record<CCSwitchModelField, string>>
  name: string
  serverAddress: string
  usageAuthorization?: CCSwitchUsageAuthorization
}

export function resolveCCSwitchDefaultName(
  apiKeyName: string | null | undefined,
  fallbackName: string
): string {
  return apiKeyName?.trim() || fallbackName
}

export function getCCSwitchUsageBaseUrl(serverAddress: string): string | null {
  let url: URL
  try {
    url = new URL(serverAddress)
  } catch {
    return null
  }

  if (url.protocol === 'https:') return url.origin

  // Imported usage scripts allow HTTP for loopback hosts during development.
  const isLoopback =
    url.hostname === 'localhost' ||
    /^127(?:\.\d+){3}$/.test(url.hostname) ||
    url.hostname === '[::1]'
  return url.protocol === 'http:' && isLoopback ? url.origin : null
}

function getCCSwitchLogoUrl(
  logoUrl: string | null | undefined,
  serverAddress: string
): string | null {
  if (!logoUrl?.trim()) return null

  try {
    const url = new URL(logoUrl, `${serverAddress}/`)
    const domain = url.hostname.replace(/\.+$/, '')
    if (
      url.protocol !== 'https:' ||
      url.port ||
      url.username ||
      url.password ||
      url.href.length > 2048 ||
      !domain.includes('.') ||
      /^[\d.]+$/.test(domain) ||
      domain.startsWith('[') ||
      domain.endsWith('.localhost')
    ) {
      return null
    }
    return url.href
  } catch {
    return null
  }
}

export function buildCCSwitchImportUrl(config: CCSwitchImportConfig): string {
  const serverAddress = config.serverAddress.trim().replace(/\/+$/, '')
  const apiKey = config.apiKey.trim()
  const normalizedApiKey = apiKey.startsWith('sk-') ? apiKey : `sk-${apiKey}`
  const appConfig = CC_SWITCH_APPS[config.app]
  const params = new URLSearchParams()

  params.set('resource', 'provider')
  params.set('app', config.app)
  params.set('name', config.name)
  params.set('endpoint', `${serverAddress}${appConfig.endpointPath}`)
  params.set('apiKey', normalizedApiKey)
  params.set('apiFormat', appConfig.apiFormat)
  for (const { key } of appConfig.modelFields) {
    const value = config.models[key]?.trim()
    if (value) params.set(key, value)
  }
  params.set('homepage', serverAddress)
  params.set('enabled', 'true')
  const logoUrl = getCCSwitchLogoUrl(config.logoUrl, serverAddress)
  if (logoUrl) params.set('iconUrl', logoUrl)

  const usageBaseUrl = getCCSwitchUsageBaseUrl(serverAddress)
  if (usageBaseUrl && config.usageAuthorization) {
    params.set('usageEnabled', 'true')
    params.set('usageScript', btoa(accountUsageScript))
    params.set('usageBaseUrl', usageBaseUrl)
    params.set('usageAccessToken', config.usageAuthorization.access_token)
    params.set('usageUserId', String(config.usageAuthorization.user_id))
  }

  return `ccswitch://v1/import?${params.toString()}`
}
