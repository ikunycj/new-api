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
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { getPricing } from '@/features/pricing/api'
import type { PricingModel } from '@/features/pricing/types'

export const LOAD_TEST_DURATION_MS = 60_000
export const LOAD_TEST_INTERVAL_MS = 500
export const LOAD_TEST_TIMEOUT_MS = 120_000
export const LOAD_TEST_MODEL = 'claude-opus-4-8'

export type LoadTestKey = ApiKey & { secret: string }

export type LoadTestRequestResult = {
  keyName: string
  latency: number
  status: number
  errorCode?: string
  requestId?: string
  usage?: LoadTestUsage
  success: boolean
}

export type LoadTestUsage = {
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
}

export type LoadTestPricing = {
  model: PricingModel
  groupRatio: number
  group: string
}

function isClaudeLoadTestKey(apiKey: ApiKey) {
  const searchable = [
    apiKey.name,
    apiKey.group ?? '',
    apiKey.model_limits ?? '',
    ...apiKey.group_candidates,
  ]
    .join(' ')
    .toLowerCase()

  return searchable.includes('claude') || searchable.includes('anthropic')
}

export async function loadClaudeLoadTestKeys(): Promise<LoadTestKey[]> {
  const response = await getApiKeys({ p: 1, size: 100 })
  const candidates = (response.data?.items ?? []).filter(
    (apiKey) => apiKey.status === 1 && isClaudeLoadTestKey(apiKey)
  )

  const loaded = await Promise.all(
    candidates.map(async (apiKey) => {
      const response = await fetchTokenKey(apiKey.id)
      const secret = response.data?.key?.trim()
      return secret ? { ...apiKey, secret } : null
    })
  )

  return loaded.filter((apiKey): apiKey is LoadTestKey => apiKey !== null)
}

export async function loadClaudeLoadTestPricing(
  modelName: string,
  group: string
): Promise<LoadTestPricing | null> {
  const pricing = await getPricing()
  const model = pricing.data?.find((item) => item.model_name === modelName)
  if (!model) return null

  const configuredRatio = pricing.group_ratio?.[group]
  return {
    model,
    group,
    groupRatio:
      typeof configuredRatio === 'number' && Number.isFinite(configuredRatio)
        ? configuredRatio
        : 1,
  }
}

function readRequestId(response: Response) {
  for (const header of ['x-request-id', 'x-oneapi-request-id', 'request-id']) {
    const value = response.headers.get(header)?.trim()
    if (value) return value
  }
  return undefined
}

function readErrorCode(payload: unknown) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return undefined
  }

  const record = payload as Record<string, unknown>
  const error = record.error
  if (typeof error === 'string' && error.trim()) {
    return error.trim()
  }
  if (error && typeof error === 'object' && !Array.isArray(error)) {
    const errorRecord = error as Record<string, unknown>
    for (const value of [
      errorRecord.code,
      errorRecord.type,
      errorRecord.message,
    ]) {
      if (typeof value === 'string' && value.trim()) {
        return value.trim()
      }
      if (typeof value === 'number' && Number.isFinite(value)) {
        return String(value)
      }
    }
  }
  if (typeof record.code === 'string' && record.code.trim()) {
    return record.code.trim()
  }
  if (typeof record.code === 'number' && Number.isFinite(record.code)) {
    return String(record.code)
  }
  if (typeof record.message === 'string' && record.message.trim()) {
    return record.message.trim()
  }
  return undefined
}

function readUsage(payload: unknown): LoadTestUsage | undefined {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return undefined
  }

  const usage = (payload as Record<string, unknown>).usage
  if (!usage || typeof usage !== 'object' || Array.isArray(usage)) {
    return undefined
  }

  const record = usage as Record<string, unknown>
  const readNumber = (key: string) => {
    const value = record[key]
    return typeof value === 'number' && Number.isFinite(value) && value >= 0
      ? value
      : 0
  }

  return {
    inputTokens: readNumber('input_tokens'),
    outputTokens: readNumber('output_tokens'),
    cacheReadTokens: readNumber('cache_read_input_tokens'),
    cacheWriteTokens: readNumber('cache_creation_input_tokens'),
  }
}

export async function sendClaudeLoadTestRequest(
  baseUrl: string,
  apiKey: LoadTestKey,
  model: string,
  runId: string,
  signal?: AbortSignal
): Promise<LoadTestRequestResult> {
  const startedAt = performance.now()
  const controller = new AbortController()
  const timeoutId = window.setTimeout(
    () => controller.abort(),
    LOAD_TEST_TIMEOUT_MS
  )
  const abortRequest = () => controller.abort()
  signal?.addEventListener('abort', abortRequest, { once: true })

  try {
    const response = await fetch(`${baseUrl.replace(/\/+$/, '')}/v1/messages`, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'anthropic-version': '2023-06-01',
        Authorization: `Bearer ${apiKey.secret}`,
        'x-api-key': apiKey.secret,
        'X-Load-Test-ID': runId,
      },
      body: JSON.stringify({
        model,
        max_tokens: 32,
        messages: [{ role: 'user', content: 'Reply with OK.' }],
      }),
      credentials: 'omit',
      cache: 'no-store',
      signal: controller.signal,
    })
    const payload = await response.json().catch(() => null)
    return {
      keyName: apiKey.name,
      latency: Math.max(1, Math.round(performance.now() - startedAt)),
      status: response.status,
      errorCode: response.ok ? undefined : readErrorCode(payload),
      requestId: readRequestId(response),
      usage: response.ok ? readUsage(payload) : undefined,
      success: response.ok,
    }
  } catch (error) {
    const isAbort = error instanceof DOMException && error.name === 'AbortError'
    return {
      keyName: apiKey.name,
      latency: Math.max(1, Math.round(performance.now() - startedAt)),
      status: 0,
      errorCode: isAbort ? 'timeout' : 'network_error',
      success: false,
    }
  } finally {
    window.clearTimeout(timeoutId)
    signal?.removeEventListener('abort', abortRequest)
  }
}
