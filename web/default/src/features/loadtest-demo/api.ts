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
import { fetchTokenKeysBatch, getApiKeys } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { getPricing } from '@/features/pricing/api'
import type { PricingModel } from '@/features/pricing/types'
import { api } from '@/lib/api'

export const LOAD_TEST_DEFAULT_DURATION_SECONDS = 60
export const LOAD_TEST_MIN_DURATION_SECONDS = 5
export const LOAD_TEST_MAX_DURATION_SECONDS = 600
export const LOAD_TEST_DEFAULT_RPS = 2
export const LOAD_TEST_MIN_RPS = 1
export const LOAD_TEST_MAX_RPS = 20
export const LOAD_TEST_MAX_REQUESTS = 10_000
export const LOAD_TEST_MAX_CONCURRENCY = 10
export const LOAD_TEST_TIMEOUT_MS = 120_000
export const LOAD_TEST_MODEL = 'gpt-5.6-sol'
export const LOAD_TEST_MODELS = [
  'gpt-5.6-sol',
  'gpt-5.6-terra',
  'gpt-5.6-luna',
  'gpt-5.4-mini',
  'gpt-4o',
  'gpt-4.1',
  'claude-opus-4-8',
  'claude-sonnet-4-6',
  'claude-3-7-sonnet',
] as const

const LOAD_TEST_CACHE_PREFIX = Array.from(
  { length: 48 },
  (_, index) =>
    `Stable load-test context section ${index + 1}: keep this deterministic prefix unchanged so provider prompt caching can reuse it across requests. The demo measures gateway routing and usage reporting only.`
).join('\n')

export type LoadTestKey = ApiKey & { secret: string }
export type LoadTestProvider = 'openai' | 'claude'

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

export type LoadTestChannelStats = {
  channel_id: number
  channel_name: string
  cluster_id: number
  pool_name: string
  cost_factor: number
  requests: number
  input_tokens: number
  input_tokens_total: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
}

export function getLoadTestProvider(model: string): LoadTestProvider {
  return model.toLowerCase().startsWith('claude-') ? 'claude' : 'openai'
}

export async function loadLoadTestKeys(): Promise<LoadTestKey[]> {
  const response = await getApiKeys({ p: 1, size: 100 })
  // Keys are already scoped to the authenticated account by the API. Do not
  // infer the provider from a key name/group: ordinary keys commonly use
  // names such as "default" while still being valid for GPT or Claude.
  const candidates = (response.data?.items ?? []).filter(
    (apiKey) => apiKey.status === 1
  )
  if (candidates.length === 0) return []

  // Fetch secrets in one request. Calling the per-key endpoint concurrently
  // can trip the critical rate limit and make the whole selector appear empty.
  const keyResponse = await fetchTokenKeysBatch(candidates.map((apiKey) => apiKey.id))
  const secrets = keyResponse.data?.keys ?? {}

  return candidates.flatMap((apiKey) => {
    const secret = secrets[apiKey.id]?.trim()
    return secret ? [{ ...apiKey, secret }] : []
  })
}

export async function loadLoadTestPricing(
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

export async function getLoadTestChannelStats(
  requestIds: string[]
): Promise<LoadTestChannelStats[]> {
  if (requestIds.length === 0) return []
  const response = await api.post<{
    success: boolean
    message?: string
    data?: LoadTestChannelStats[]
  }>('/api/log/self/loadtest-stats', { request_ids: requestIds })
  if (!response.data.success) {
    throw new Error(
      response.data.message || 'Failed to load channel statistics'
    )
  }
  return response.data.data ?? []
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

function readUsage(
  payload: unknown,
  provider: LoadTestProvider
): LoadTestUsage | undefined {
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
    if (typeof value === 'number' && Number.isFinite(value) && value >= 0) {
      return value
    }
    if (typeof value === 'string') {
      const parsed = Number(value)
      if (Number.isFinite(parsed) && parsed >= 0) return parsed
    }
    return 0
  }

  if (provider === 'openai') {
    const inputDetails = record.input_tokens_details
    const promptDetails = record.prompt_tokens_details
    const inputDetailsRecord =
      inputDetails &&
      typeof inputDetails === 'object' &&
      !Array.isArray(inputDetails)
        ? (inputDetails as Record<string, unknown>)
        : undefined
    const promptDetailsRecord =
      promptDetails &&
      typeof promptDetails === 'object' &&
      !Array.isArray(promptDetails)
        ? (promptDetails as Record<string, unknown>)
        : undefined
    const cacheReadTokens =
      (promptDetailsRecord
        ? readNumberFromRecord(promptDetailsRecord, 'cached_tokens')
        : 0) ||
      (inputDetailsRecord
        ? readNumberFromRecord(inputDetailsRecord, 'cached_tokens')
        : 0)
    const cacheWriteTokens =
      readNumber('cache_creation_input_tokens') ||
      readNumber('cache_write_tokens')
    const totalInputTokens =
      readNumber('prompt_tokens') || readNumber('input_tokens')
    return {
      inputTokens: Math.max(
        0,
        totalInputTokens - cacheReadTokens - cacheWriteTokens
      ),
      outputTokens:
        readNumber('completion_tokens') || readNumber('output_tokens'),
      cacheReadTokens,
      cacheWriteTokens,
    }
  }

  const cacheCreation = record.cache_creation
  const cacheCreationRecord =
    cacheCreation && typeof cacheCreation === 'object'
      ? (cacheCreation as Record<string, unknown>)
      : undefined
  const cacheWriteTokens =
    readNumber('cache_creation_input_tokens') ||
    (cacheCreationRecord
      ? readNumberFromRecord(cacheCreationRecord, 'ephemeral_5m_input_tokens') +
        readNumberFromRecord(cacheCreationRecord, 'ephemeral_1h_input_tokens')
      : 0)

  return {
    inputTokens: readNumber('input_tokens'),
    outputTokens: readNumber('output_tokens'),
    cacheReadTokens: readNumber('cache_read_input_tokens'),
    cacheWriteTokens,
  }
}

function readNumberFromRecord(record: Record<string, unknown>, key: string) {
  const value = record[key]
  if (typeof value === 'number' && Number.isFinite(value) && value >= 0) {
    return value
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (Number.isFinite(parsed) && parsed >= 0) return parsed
  }
  return 0
}

export async function sendLoadTestRequest(
  baseUrl: string,
  apiKey: LoadTestKey,
  model: string,
  runId: string,
  promptCache: boolean,
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
    const provider = getLoadTestProvider(model)
    const requestBody: Record<string, unknown> = {
      model,
      max_tokens: 32,
      messages: [{ role: 'user', content: 'Reply with OK.' }],
    }
    if (provider === 'claude') {
      if (promptCache) {
        requestBody.system = [
          {
            type: 'text',
            text: LOAD_TEST_CACHE_PREFIX,
            cache_control: { type: 'ephemeral' },
          },
        ]
      }
    } else {
      requestBody.temperature = 0
      requestBody.stream = false
      if (promptCache) {
        requestBody.messages = [
          { role: 'system', content: LOAD_TEST_CACHE_PREFIX },
          { role: 'user', content: 'Reply with OK.' },
        ]
      }
    }

    const endpoint =
      provider === 'claude' ? '/v1/messages' : '/v1/chat/completions'
    const headers: Record<string, string> = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey.secret}`,
      'X-Load-Test-ID': runId,
    }
    if (provider === 'claude') {
      headers['anthropic-version'] = '2023-06-01'
      headers['x-api-key'] = apiKey.secret
    }
    const response = await fetch(`${baseUrl.replace(/\/+$/, '')}${endpoint}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(requestBody),
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
      usage: response.ok ? readUsage(payload, provider) : undefined,
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
