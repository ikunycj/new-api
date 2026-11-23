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

type CCSwitchImportConfig = {
  apiKey: string
  app: CCSwitchApp
  models: Partial<Record<CCSwitchModelField, string>>
  name: string
  serverAddress: string
}

export function resolveCCSwitchDefaultName(
  apiKeyName: string | null | undefined,
  fallbackName: string
): string {
  return apiKeyName?.trim() || fallbackName
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

  return `ccswitch://v1/import?${params.toString()}`
}
