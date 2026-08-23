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
import { z } from 'zod'

const LOAD_TEST_RESULTS_VERSION = 1
const LOAD_TEST_RESULTS_TTL_MS = 7 * 24 * 60 * 60 * 1000
const LOAD_TEST_RESULTS_KEY_PREFIX = 'new-api:loadtest-demo:results:v1'

const runStatsSchema = z.object({
  completed: z.number().int().nonnegative(),
  failures: z.number().int().nonnegative(),
  latencies: z.array(z.number().nonnegative()),
  successes: z.number().int().nonnegative(),
  statusCodes: z.record(z.string(), z.number().int().nonnegative()),
  errorCodes: z.record(z.string(), z.number().int().nonnegative()),
  keyCounts: z.record(z.string(), z.number().int().nonnegative()),
  inputTokens: z.number().int().nonnegative(),
  outputTokens: z.number().int().nonnegative(),
  cacheReadTokens: z.number().int().nonnegative(),
  cacheWriteTokens: z.number().int().nonnegative(),
})

const channelStatsSchema = z.object({
  channel_id: z.number().int(),
  channel_name: z.string(),
  cluster_id: z.number().int(),
  pool_name: z.string(),
  cost_factor: z.number(),
  requests: z.number().int().nonnegative(),
  input_tokens: z.number().int().nonnegative(),
  input_tokens_total: z.number().int().nonnegative(),
  output_tokens: z.number().int().nonnegative(),
  cache_read_tokens: z.number().int().nonnegative(),
  cache_write_tokens: z.number().int().nonnegative(),
  token_id: z.number().int(),
  charged_quota: z.number().int().nonnegative(),
})

const comparisonResultSchema = z.object({
  strategy: z.enum([
    'cost_first',
    'balanced',
    'stability_first',
    'pro_cost_first',
    'pro_stability_first',
  ]),
  runId: z.string().min(1),
  stats: runStatsSchema,
  channelStats: z.array(channelStatsSchema),
  requestIds: z.array(z.string().min(1)),
})

const persistedRunSchema = z.object({
  version: z.literal(LOAD_TEST_RESULTS_VERSION),
  savedAt: z.number().int().nonnegative(),
  model: z.string().min(1),
  results: z.array(comparisonResultSchema),
})

export type RunStats = z.infer<typeof runStatsSchema>
export type LoadTestComparisonResult = z.infer<typeof comparisonResultSchema>
export type PersistedLoadTestRun = z.infer<typeof persistedRunSchema>

function storageKey(userId: number) {
  return `${LOAD_TEST_RESULTS_KEY_PREFIX}:${userId}`
}

export function loadPersistedLoadTestRun(
  userId: number | undefined
): PersistedLoadTestRun | null {
  if (typeof window === 'undefined' || !userId) return null

  const key = storageKey(userId)
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return null
    const parsed = persistedRunSchema.safeParse(JSON.parse(raw))
    if (
      !parsed.success ||
      Date.now() - parsed.data.savedAt > LOAD_TEST_RESULTS_TTL_MS
    ) {
      window.localStorage.removeItem(key)
      return null
    }
    return parsed.data
  } catch {
    window.localStorage.removeItem(key)
    return null
  }
}

export function savePersistedLoadTestRun(
  userId: number | undefined,
  model: string,
  results: LoadTestComparisonResult[]
): void {
  if (typeof window === 'undefined' || !userId) return

  const key = storageKey(userId)
  try {
    if (results.length === 0) {
      window.localStorage.removeItem(key)
      return
    }
    window.localStorage.setItem(
      key,
      JSON.stringify({
        version: LOAD_TEST_RESULTS_VERSION,
        savedAt: Date.now(),
        model,
        results,
      } satisfies PersistedLoadTestRun)
    )
  } catch {
    // Storage may be unavailable or full; the running test should continue.
  }
}
