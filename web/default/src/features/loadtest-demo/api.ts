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
import { isAxiosError } from 'axios'

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
export const LOAD_TEST_DEFAULT_MAX_OUTPUT_TOKENS = 256
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

export function supportsLoadTestStreaming(
  endpoint: LoadTestEndpoint | undefined
) {
  return endpoint !== undefined && endpoint !== 'openai-response-compact'
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
  endpoint: LoadTestEndpoint,
  streamMode = false,
  maxOutputTokens = LOAD_TEST_DEFAULT_MAX_OUTPUT_TOKENS
): Record<string, unknown> {
  const streaming = streamMode && supportsLoadTestStreaming(endpoint)
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
        stream: streaming,
      }
    : {
        model,
        max_tokens: maxOutputTokens,
        messages: [{ role: 'user', content: prompt }],
      }

  if (endpoint === 'anthropic') {
    if (streaming) requestBody.stream = true
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
    requestBody.stream = streaming
    if (streaming) {
      requestBody.stream_options = { include_usage: true }
    }
    if (promptCache) {
      requestBody.messages = [
        { role: 'system', content: LOAD_TEST_CACHE_PREFIX },
        { role: 'user', content: prompt },
      ]
    }
  } else if (endpoint === 'openai-response') {
    requestBody.max_output_tokens = maxOutputTokens
    requestBody.stream = streaming
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
  firstTokenLatency?: number
  outputTokensPerSecond?: number
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
  min_output_tokens: number
  max_output_tokens: number
}

export type LoadTestAgent = {
  id: string
  managed: boolean
  name: string
  platform: string
  version: string
  cpu_cores: number
  memory_bytes: number
  max_rps: number
  max_concurrency: number
  last_seen_at: number
  created_at: number
}

export type LoadTestAgentList = {
  local_agents: LoadTestAgent[]
  managed_agents: LoadTestAgent[]
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

export type LoadTestExecutionMode = 'single' | 'shared'

export type LoadTestRunWorker = {
  id: string
  run_id: string
  agent_id: string
  worker_id: string
  slot: number
  name: string
  platform: string
  cpu_cores: number
  memory_bytes: number
  max_rps: number
  max_concurrency: number
  assigned_rps: number
  assigned_concurrency: number
  status: string
  sent: number
  completed: number
  successes: number
  failures: number
  dropped: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  usage_missing: number
  token_stats_source: string
  current_rps: number
  p50_ms: number
  p95_ms: number
  p99_ms: number
  error_counts: Record<string, number>
  error_message: string
  last_seen_at: number
  started_at: number
  finished_at: number
}

export type LoadTestMockChannel = {
  slot: number
  failure_rate: number
  failure_status: number
  latency_ms: number
}

export type LoadTestAgentRun = {
  id: string
  agent_id: string
  execution_mode: LoadTestExecutionMode
  expected_workers: number
  join_deadline_at: number
  start_at: number
  token_id: number
  key_name: string
  package_name: string
  model: string
  endpoint: LoadTestEndpoint
  prompt: string
  prompt_cache: boolean
  stream_mode: boolean
  mock_enabled: boolean
  mock_failure_rate: number
  mock_failure_status: number
  mock_latency_ms: number
  mock_channels: LoadTestMockChannel[]
  workers: LoadTestRunWorker[]
  duration_seconds: number
  requests_per_second: number
  concurrency: number
  max_output_tokens: number
  agent_managed: boolean
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
  usage_missing: number
  token_stats_source: string
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
  execution_mode?: LoadTestExecutionMode
  expected_workers?: number
  token_id: number
  model: string
  endpoint: LoadTestEndpoint
  prompt: string
  prompt_cache: boolean
  stream_mode: boolean
  mock_enabled?: boolean
  mock_failure_rate?: number
  mock_failure_status?: number
  mock_latency_ms?: number
  mock_channels?: LoadTestMockChannel[]
  duration_seconds: number
  requests_per_second: number
  concurrency: number
  max_output_tokens: number
}

export const DEFAULT_LOAD_TEST_LIMITS: LoadTestLimits = {
  min_duration_seconds: LOAD_TEST_MIN_DURATION_SECONDS,
  max_duration_seconds: LOAD_TEST_MAX_DURATION_SECONDS,
  min_rps: LOAD_TEST_MIN_RPS,
  max_rps: LOAD_TEST_MAX_RPS,
  min_concurrency: LOAD_TEST_MIN_CONCURRENCY,
  max_concurrency: LOAD_TEST_MAX_CONCURRENCY,
  min_output_tokens: 1,
  max_output_tokens: LOAD_TEST_DEFAULT_MAX_OUTPUT_TOKENS,
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

export async function createLoadTestAgentPairing(managed = false): Promise<{
  agent_id: string
  code: string
  expires_at: number
}> {
  const response = await api.post<{
    success: boolean
    message?: string
    data?: { agent_id: string; code: string; expires_at: number }
  }>(
    managed
      ? '/api/loadtest/managed-agents/pairing'
      : '/api/loadtest/agents/pairing',
    undefined,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
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

export async function deleteLoadTestAgent(
  agentId: string,
  managed = false
): Promise<void> {
  const response = await api.delete<{ success: boolean; message?: string }>(
    managed
      ? `/api/loadtest/managed-agents/${encodeURIComponent(agentId)}`
      : `/api/loadtest/agents/${encodeURIComponent(agentId)}`,
    { skipBusinessError: true, skipErrorHandler: true }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to remove agent')
  }
}

export async function updateManagedLoadTestAgentCapacity(
  agentId: string,
  capacity: { max_rps: number; max_concurrency: number }
): Promise<LoadTestAgent> {
  const response = await api.put<{
    success: boolean
    message?: string
    data?: LoadTestAgent
  }>(
    `/api/loadtest/managed-agents/${encodeURIComponent(agentId)}/capacity`,
    capacity,
    { skipBusinessError: true, skipErrorHandler: true }
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to update agent capacity')
  }
  return response.data.data
}

export async function createLoadTestAgentRun(
  request: CreateLoadTestAgentRun
): Promise<LoadTestAgentRun> {
  let response
  try {
    response = await api.post<{
      success: boolean
      message?: string
      data?: LoadTestAgentRun
    }>('/api/loadtest/runs', request, {
      skipBusinessError: true,
      skipErrorHandler: true,
    })
  } catch (error) {
    if (isAxiosError<{ message?: string }>(error)) {
      throw new Error(
        error.response?.data.message || 'Failed to create agent run'
      )
    }
    throw error
  }
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
    (left, right) =>
      right.requests - left.requests || left.channel_id - right.channel_id
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
  const has = (key: string) => Object.hasOwn(record, key)
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
      promptDetailsRecord && Object.hasOwn(promptDetailsRecord, 'cached_tokens')
        ? readNumberFromRecord(promptDetailsRecord, 'cached_tokens')
        : inputDetailsRecord && Object.hasOwn(inputDetailsRecord, 'cached_tokens')
          ? readNumberFromRecord(inputDetailsRecord, 'cached_tokens')
          : 0
    const cacheWriteTokens = has('cache_creation_input_tokens')
      ? readNumber('cache_creation_input_tokens')
      : has('cache_write_tokens')
        ? readNumber('cache_write_tokens')
        : 0
    const totalInputTokens = has('input_tokens')
      ? readNumber('input_tokens')
      : readNumber('prompt_tokens')
    return {
      inputTokens: Math.max(
        0,
        totalInputTokens - cacheReadTokens - cacheWriteTokens
      ),
      outputTokens: has('output_tokens')
        ? readNumber('output_tokens')
        : readNumber('completion_tokens'),
      cacheReadTokens,
      cacheWriteTokens,
    }
  }

  const cacheCreation = record.cache_creation
  const cacheCreationRecord =
    cacheCreation && typeof cacheCreation === 'object'
      ? (cacheCreation as Record<string, unknown>)
      : undefined
  const cacheWriteTokens = has('cache_creation_input_tokens')
    ? readNumber('cache_creation_input_tokens')
    : cacheCreationRecord
      ? readNumberFromRecord(cacheCreationRecord, 'ephemeral_5m_input_tokens') +
        readNumberFromRecord(cacheCreationRecord, 'ephemeral_1h_input_tokens')
      : 0

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
  endpointOverride?: LoadTestEndpoint,
  streamMode = false,
  maxOutputTokens = LOAD_TEST_DEFAULT_MAX_OUTPUT_TOKENS
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
    const requestUsesStreaming =
      streamMode && supportsLoadTestStreaming(endpoint)
    const requestBody = buildLoadTestRequestBody(
      model,
      prompt,
      promptCache,
      endpoint,
      requestUsesStreaming,
      maxOutputTokens
    )
    const endpointPath = getLoadTestEndpointPath(endpoint)
    const headers: Record<string, string> = {
      Accept: requestUsesStreaming ? 'text/event-stream' : 'application/json',
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
    if (requestUsesStreaming && response.ok && response.body) {
      const streamed = await readStreamingResponse(
        response,
        provider,
        startedAt
      )
      return {
        keyName: apiKey.name,
        latency: streamed.latency,
        status: response.status,
        requestId: readRequestId(response),
        usage: streamed.usage,
        firstTokenLatency: streamed.firstTokenLatency,
        outputTokensPerSecond: streamed.outputTokensPerSecond,
        success: true,
      }
    }
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

async function readStreamingResponse(
  response: Response,
  provider: LoadTestProvider,
  startedAt: number
): Promise<{
  latency: number
  firstTokenLatency: number
  outputTokensPerSecond: number
  usage?: LoadTestUsage
}> {
  const reader = response.body?.getReader()
  if (!reader) {
    return { latency: 0, firstTokenLatency: 0, outputTokensPerSecond: 0 }
  }
  const decoder = new TextDecoder()
  let buffer = ''
  let firstTokenAt = 0
  let outputTokenEstimate = 0
  let usage: LoadTestUsage | undefined
  let done = false
  while (!done) {
    const chunk = await reader.read()
    done = chunk.done
    if (chunk.value) buffer += decoder.decode(chunk.value, { stream: !done })
    const events = buffer.split(/\r?\n\r?\n/)
    buffer = events.pop() ?? ''
    for (const event of events) {
      const data = event
        .split(/\r?\n/)
        .filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trim())
        .join('\n')
      if (!data || data === '[DONE]') continue
      try {
        const payload = JSON.parse(data) as Record<string, unknown>
        const eventUsage = readUsage(
          payload.usage
            ? payload
            : payload.message &&
                typeof payload.message === 'object' &&
                !Array.isArray(payload.message) &&
                (payload.message as Record<string, unknown>).usage
              ? { usage: (payload.message as Record<string, unknown>).usage }
              : payload,
          provider
        )
        if (eventUsage) usage = mergeLoadTestUsage(usage, eventUsage)
        if (JSON.stringify(payload).includes('delta')) {
          if (!firstTokenAt) firstTokenAt = performance.now()
          outputTokenEstimate += 1
        }
      } catch {
        // Ignore provider keep-alives and non-JSON SSE lines.
      }
    }
  }
  if (buffer.trim()) {
    const data = buffer
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trim())
      .join('\n')
    if (data && data !== '[DONE]') {
      try {
        const payload = JSON.parse(data) as Record<string, unknown>
        const eventUsage = readUsage(
          payload.usage
            ? payload
            : payload.message &&
                typeof payload.message === 'object' &&
                !Array.isArray(payload.message) &&
                (payload.message as Record<string, unknown>).usage
              ? { usage: (payload.message as Record<string, unknown>).usage }
              : payload,
          provider
        )
        if (eventUsage) usage = mergeLoadTestUsage(usage, eventUsage)
        if (JSON.stringify(payload).includes('delta')) {
          if (!firstTokenAt) firstTokenAt = performance.now()
          outputTokenEstimate += 1
        }
      } catch {
        // Ignore an incomplete provider keep-alive at end of stream.
      }
    }
  }
  const finishedAt = performance.now()
  const latency = Math.max(1, Math.round(finishedAt - startedAt))
  const firstTokenLatency = firstTokenAt
    ? Math.max(1, Math.round(firstTokenAt - startedAt))
    : latency
  const outputTokens = usage?.outputTokens ?? outputTokenEstimate
  const outputTokensPerSecond =
    outputTokens > 0 && latency > firstTokenLatency
      ? (outputTokens * 1000) / (latency - firstTokenLatency)
      : 0
  return { latency, firstTokenLatency, outputTokensPerSecond, usage }
}

function mergeLoadTestUsage(
  previous: LoadTestUsage | undefined,
  next: LoadTestUsage
): LoadTestUsage {
  if (!previous) return next
  return {
    inputTokens: next.inputTokens > 0 ? next.inputTokens : previous.inputTokens,
    outputTokens:
      next.outputTokens > 0 ? next.outputTokens : previous.outputTokens,
    cacheReadTokens:
      next.cacheReadTokens > 0
        ? next.cacheReadTokens
        : previous.cacheReadTokens,
    cacheWriteTokens:
      next.cacheWriteTokens > 0
        ? next.cacheWriteTokens
        : previous.cacheWriteTokens,
  }
}
