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
  fetchTokenKeysBatch,
  getApiKeys,
  type ApiKeyModel,
} from '@/features/keys/api'
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
export const LOAD_TEST_DEFAULT_CONCURRENCY = 10
export const LOAD_TEST_MIN_CONCURRENCY = 1
export const LOAD_TEST_MAX_CONCURRENCY = 10
export const LOAD_TEST_DEFAULT_PROMPT = 'Reply with OK.'
export const LOAD_TEST_MAX_PROMPT_CHARS = 8000
export const LOAD_TEST_TIMEOUT_MS = 120_000
const LOAD_TEST_CACHE_PREFIX = Array.from(
  { length: 48 },
  (_, index) =>
    `Stable load-test context section ${index + 1}: keep this deterministic prefix unchanged so provider prompt caching can reuse it across requests. The demo measures gateway routing and usage reporting only.`
).join('\n')

export type LoadTestKey = ApiKey & { secret: string }
export type LoadTestProvider = 'openai' | 'claude'
export type LoadTestEndpoint =
  | 'openai'
  | 'openai-response'
  | 'openai-response-compact'
  | 'anthropic'

export type LoadTestModel = {
  id: string
  provider: LoadTestProvider
  endpoint: LoadTestEndpoint
}

export function getLoadTestEndpointPath(endpoint: LoadTestEndpoint): string {
  return {
    anthropic: '/v1/messages',
    openai: '/v1/chat/completions',
    'openai-response': '/v1/responses',
    'openai-response-compact': '/v1/responses/compact',
  }[endpoint]
}

export function buildLoadTestRequestBody(
  model: string,
  prompt: string,
  promptCache: boolean,
  endpoint: LoadTestEndpoint
): Record<string, unknown> {
  const isResponsesEndpoint =
    endpoint === 'openai-response' || endpoint === 'openai-response-compact'
  const requestBody: Record<string, unknown> = isResponsesEndpoint
    ? {
        model,
        input: promptCache
          ? [
              { role: 'system', content: LOAD_TEST_CACHE_PREFIX },
              { role: 'user', content: prompt },
            ]
          : [{ role: 'user', content: prompt }],
      }
    : {
        model,
        max_tokens: 32,
        messages: [{ role: 'user', content: prompt }],
      }

  if (endpoint === 'anthropic') {
    if (promptCache) {
      requestBody.system = [
        {
          type: 'text',
          text: LOAD_TEST_CACHE_PREFIX,
          cache_control: { type: 'ephemeral' },
        },
      ]
    }
  } else if (endpoint === 'openai') {
    requestBody.temperature = 0
    requestBody.stream = false
    if (promptCache) {
      requestBody.messages = [
        { role: 'system', content: LOAD_TEST_CACHE_PREFIX },
        { role: 'user', content: prompt },
      ]
    }
  } else if (endpoint === 'openai-response') {
    requestBody.max_output_tokens = 32
    requestBody.stream = false
  }

  return requestBody
}

export function getLoadTestApiBaseUrl(serverAddress: string): string {
  const normalizedAddress = serverAddress.trim().replace(/\/+$/, '')
  return normalizedAddress.endsWith('/v1')
    ? normalizedAddress
    : `${normalizedAddress}/v1`
}

export function getLoadTestModels(models: ApiKeyModel[]): LoadTestModel[] {
  const seen = new Set<string>()
  const loadTestModels: LoadTestModel[] = []

  for (const model of models) {
    const supportsOpenAI = model.supportedEndpointTypes.includes('openai')
    const supportsOpenAIResponses =
      model.supportedEndpointTypes.includes('openai-response')
    const supportsOpenAICompact = model.supportedEndpointTypes.includes(
      'openai-response-compact'
    )
    const supportsClaude = model.supportedEndpointTypes.includes('anthropic')
    if (
      !supportsOpenAI &&
      !supportsOpenAIResponses &&
      !supportsOpenAICompact &&
      !supportsClaude
    ) {
      continue
    }
    if (seen.has(model.id)) continue

    const provider = getLoadTestProviderFromModel(model)
    let endpoint: LoadTestEndpoint | undefined
    if (provider === 'claude' && supportsClaude) {
      endpoint = 'anthropic'
    } else if (supportsOpenAI) {
      endpoint = 'openai'
    } else if (supportsOpenAIResponses) {
      endpoint = 'openai-response'
    } else if (supportsOpenAICompact) {
      endpoint = 'openai-response-compact'
    }
    if (!endpoint) continue

    seen.add(model.id)
    loadTestModels.push({
      id: model.id,
      provider: endpoint === 'anthropic' ? 'claude' : 'openai',
      endpoint,
    })
  }

  return loadTestModels
}

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

export type LoadTestLimits = {
  min_duration_seconds: number
  max_duration_seconds: number
  min_rps: number
  max_rps: number
  min_concurrency: number
  max_concurrency: number
}

export type LoadTestAgent = {
  id: string
  name: string
  platform: string
  version: string
  last_seen_at: number
  created_at: number
}

export type LoadTestAgentList = {
  agents: LoadTestAgent[]
  online_before: number
}

export type LoadTestAgentState = LoadTestAgentList & {
  runs: LoadTestAgentRun[]
}

export type LoadTestAgentRunStatus =
  | 'queued'
  | 'dispatched'
  | 'running'
  | 'cancel_requested'
  | 'completed'
  | 'failed'
  | 'cancelled'

export type LoadTestAgentRun = {
  id: string
  agent_id: string
  token_id: number
  key_name: string
  package_name: string
  model: string
  endpoint: LoadTestEndpoint
  prompt: string
  prompt_cache: boolean
  duration_seconds: number
  requests_per_second: number
  concurrency: number
  status: LoadTestAgentRunStatus
  sent: number
  completed: number
  successes: number
  failures: number
  dropped: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  current_rps: number
  p50_ms: number
  p95_ms: number
  p99_ms: number
  error_counts: Record<string, number>
  error_message: string
  created_at: number
  started_at: number
  finished_at: number
  updated_at: number
}

export type CreateLoadTestAgentRun = {
  agent_id: string
  token_id: number
  model: string
  endpoint: LoadTestEndpoint
  prompt: string
  prompt_cache: boolean
  duration_seconds: number
  requests_per_second: number
  concurrency: number
}

export const DEFAULT_LOAD_TEST_LIMITS: LoadTestLimits = {
  min_duration_seconds: LOAD_TEST_MIN_DURATION_SECONDS,
  max_duration_seconds: LOAD_TEST_MAX_DURATION_SECONDS,
  min_rps: LOAD_TEST_MIN_RPS,
  max_rps: LOAD_TEST_MAX_RPS,
  min_concurrency: LOAD_TEST_MIN_CONCURRENCY,
  max_concurrency: LOAD_TEST_MAX_CONCURRENCY,
}

export async function getLoadTestLimits(): Promise<LoadTestLimits> {
  const response = await api.get<{
    success: boolean
    message?: string
    data?: LoadTestLimits
  }>('/api/loadtest/config')
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to load load-test limits')
  }
  return response.data.data
}

export async function createLoadTestAgentPairing(): Promise<{
  agent_id: string
  code: string
  expires_at: number
}> {
  const response = await api.post<{
    success: boolean
    message?: string
    data?: { agent_id: string; code: string; expires_at: number }
  }>('/api/loadtest/agents/pairing', undefined, {
    skipBusinessError: true,
    skipErrorHandler: true,
  })
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to create pairing code')
  }
  return response.data.data
}

export async function getLoadTestAgents(): Promise<LoadTestAgentList> {
  const response = await api.get<{
    success: boolean
    message?: string
    data?: LoadTestAgentList
  }>('/api/loadtest/agents', {
    skipBusinessError: true,
    skipErrorHandler: true,
  })
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to load agents')
  }
  return response.data.data
}

export async function deleteLoadTestAgent(agentId: string): Promise<void> {
  const response = await api.delete<{ success: boolean; message?: string }>(
    `/api/loadtest/agents/${encodeURIComponent(agentId)}`,
    { skipBusinessError: true, skipErrorHandler: true }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to remove agent')
  }
}

export async function createLoadTestAgentRun(
  request: CreateLoadTestAgentRun
): Promise<LoadTestAgentRun> {
  const response = await api.post<{
    success: boolean
    message?: string
    data?: LoadTestAgentRun
  }>('/api/loadtest/runs', request, {
    skipBusinessError: true,
    skipErrorHandler: true,
  })
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to create agent run')
  }
  return response.data.data
}

export async function getLoadTestAgentRuns(): Promise<LoadTestAgentRun[]> {
  const response = await api.get<{
    success: boolean
    message?: string
    data?: LoadTestAgentRun[]
  }>('/api/loadtest/runs?limit=30', {
    skipBusinessError: true,
    skipErrorHandler: true,
  })
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to load agent runs')
  }
  return response.data.data ?? []
}

export async function cancelLoadTestAgentRun(runId: string): Promise<void> {
  const response = await api.post<{ success: boolean; message?: string }>(
    `/api/loadtest/runs/${encodeURIComponent(runId)}/cancel`,
    undefined,
    { skipBusinessError: true, skipErrorHandler: true }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to cancel agent run')
  }
}

export async function getLoadTestAgentState(): Promise<LoadTestAgentState> {
  const response = await api.get<{
    success: boolean
    message?: string
    data?: LoadTestAgentState
  }>('/api/loadtest/state', {
    skipBusinessError: true,
    skipErrorHandler: true,
  })
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to load agent state')
  }
  return response.data.data
}

export type LoadTestChannelStats = {
  channel_id: number
  channel_name: string
  billing_group: string
  cost_factor: number
  requests: number
  input_tokens: number
  input_tokens_total: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
}

export function getLoadTestProvider(model: string): LoadTestProvider {
  const normalizedModel = model.toLowerCase()
  return normalizedModel.includes('claude') ||
    normalizedModel.includes('anthropic')
    ? 'claude'
    : 'openai'
}

function getLoadTestProviderFromModel(model: ApiKeyModel): LoadTestProvider {
  const owner = model.ownedBy?.toLowerCase() ?? ''
  if (owner.includes('claude') || owner.includes('anthropic')) return 'claude'
  return getLoadTestProvider(model.id)
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
  const keyResponse = await fetchTokenKeysBatch(
    candidates.map((apiKey) => apiKey.id)
  )
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
  const merged = new Map<string, LoadTestChannelStats>()
  for (let offset = 0; offset < requestIds.length; offset += 10_000) {
    const response = await api.post<{
      success: boolean
      message?: string
      data?: LoadTestChannelStats[]
    }>('/api/log/self/loadtest-stats', {
      request_ids: requestIds.slice(offset, offset + 10_000),
    })
    if (!response.data.success) {
      throw new Error(
        response.data.message || 'Failed to load channel statistics'
      )
    }
    for (const channel of response.data.data ?? []) {
      const key = `${channel.channel_id}:${channel.billing_group}`
      const existing = merged.get(key)
      if (!existing) {
        merged.set(key, { ...channel })
        continue
      }
      existing.requests += channel.requests
      existing.input_tokens += channel.input_tokens
      existing.input_tokens_total += channel.input_tokens_total
      existing.output_tokens += channel.output_tokens
      existing.cache_read_tokens += channel.cache_read_tokens
      existing.cache_write_tokens += channel.cache_write_tokens
    }
  }
  return [...merged.values()].sort(
    (left, right) => right.requests - left.requests || left.channel_id - right.channel_id
  )
}

function readRequestId(response: Response) {
  // The gateway ID is the consume-log key. Upstream providers may also return
  // X-Request-Id, which is intentionally copied through for diagnostics.
  for (const header of ['x-oneapi-request-id', 'x-request-id', 'request-id']) {
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
  prompt: string,
  promptCache: boolean,
  signal?: AbortSignal,
  providerOverride?: LoadTestProvider,
  endpointOverride?: LoadTestEndpoint
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
    const provider = providerOverride ?? getLoadTestProvider(model)
    const endpoint =
      endpointOverride ?? (provider === 'claude' ? 'anthropic' : 'openai')
    const requestBody = buildLoadTestRequestBody(
      model,
      prompt,
      promptCache,
      endpoint
    )
    const endpointPath = getLoadTestEndpointPath(endpoint)
    const headers: Record<string, string> = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey.secret}`,
      'X-Load-Test-ID': runId,
    }
    if (endpoint === 'anthropic') {
      headers['anthropic-version'] = '2023-06-01'
      headers['x-api-key'] = apiKey.secret
    }
    const response = await fetch(
      `${baseUrl.replace(/\/+$/, '')}${endpointPath}`,
      {
        method: 'POST',
        headers,
        body: JSON.stringify(requestBody),
        credentials: 'omit',
        cache: 'no-store',
        signal: controller.signal,
      }
    )
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
