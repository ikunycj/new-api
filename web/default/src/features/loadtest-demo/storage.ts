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

const LOAD_TEST_RESULTS_VERSION = 2
const LOAD_TEST_RESULTS_TTL_MS = 7 * 24 * 60 * 60 * 1000
const LOAD_TEST_RESULTS_KEY_PREFIX = 'new-api:loadtest-demo:result:v1'
const MAX_PERSISTED_RUNS = 50

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
  billing_group: z.string(),
  cost_factor: z.number().nonnegative(),
  requests: z.number().int().nonnegative(),
  input_tokens: z.number().int().nonnegative(),
  input_tokens_total: z.number().int().nonnegative(),
  output_tokens: z.number().int().nonnegative(),
  cache_read_tokens: z.number().int().nonnegative(),
  cache_write_tokens: z.number().int().nonnegative(),
})

const persistedRunSchema = z.object({
  version: z.literal(LOAD_TEST_RESULTS_VERSION),
  savedAt: z.number().int().nonnegative(),
  completedAt: z.number().int().nonnegative(),
  model: z.string().min(1),
  runId: z.string().min(1),
  durationSeconds: z.number().nonnegative(),
  requestsPerSecond: z.number().nonnegative(),
  estimatedCost: z.number().nonnegative(),
  stats: runStatsSchema,
  channelStats: z.array(channelStatsSchema),
  requestIds: z.array(z.string().min(1)),
})

const legacyPersistedRunSchema = z.object({
  version: z.literal(1),
  savedAt: z.number().int().nonnegative(),
  model: z.string().min(1),
  runId: z.string().min(1),
  stats: runStatsSchema,
  channelStats: z.array(channelStatsSchema),
  requestIds: z.array(z.string().min(1)),
})

const persistedRunsSchema = z.object({
  version: z.literal(LOAD_TEST_RESULTS_VERSION),
  runs: z.array(persistedRunSchema),
})

export type RunStats = z.infer<typeof runStatsSchema>
export type LoadTestRunResult = Omit<
  z.infer<typeof persistedRunSchema>,
  'version' | 'savedAt' | 'completedAt'
>
export type PersistedLoadTestRun = z.infer<typeof persistedRunSchema>

function storageKey(userId: number) {
  return `${LOAD_TEST_RESULTS_KEY_PREFIX}:${userId}`
}

export function loadPersistedLoadTestRuns(
  userId: number | undefined
): PersistedLoadTestRun[] {
  if (typeof window === 'undefined' || !userId) return []

  const key = storageKey(userId)
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return []
    const value: unknown = JSON.parse(raw)
    const parsedRuns = persistedRunsSchema.safeParse(value)
    const legacyRun = legacyPersistedRunSchema.safeParse(value)
    let runs: PersistedLoadTestRun[] = []
    if (parsedRuns.success) {
      runs = parsedRuns.data.runs
    } else if (legacyRun.success) {
      runs = [
        {
          ...legacyRun.data,
          version: LOAD_TEST_RESULTS_VERSION as 2,
          completedAt: legacyRun.data.savedAt,
          durationSeconds: 0,
          requestsPerSecond: 0,
          estimatedCost: 0,
        },
      ]
    }
    const validRuns = runs.filter(
      (run) => Date.now() - run.savedAt <= LOAD_TEST_RESULTS_TTL_MS
    )
    if (validRuns.length !== runs.length) {
      if (validRuns.length === 0) window.localStorage.removeItem(key)
      else {
        window.localStorage.setItem(
          key,
          JSON.stringify({ version: LOAD_TEST_RESULTS_VERSION, runs: validRuns })
        )
      }
    }
    return validRuns.sort((a, b) => b.completedAt - a.completedAt)
  } catch {
    window.localStorage.removeItem(key)
    return []
  }
}

export function loadPersistedLoadTestRun(
  userId: number | undefined
): PersistedLoadTestRun | null {
  return loadPersistedLoadTestRuns(userId)[0] ?? null
}

export function savePersistedLoadTestRun(
  userId: number | undefined,
  result: LoadTestRunResult
): void {
  if (typeof window === 'undefined' || !userId) return

  try {
    const runs = loadPersistedLoadTestRuns(userId).filter(
      (run) => run.runId !== result.runId
    )
    const now = Date.now()
    window.localStorage.setItem(
      storageKey(userId),
      JSON.stringify({
        version: LOAD_TEST_RESULTS_VERSION,
        runs: [
          {
            version: LOAD_TEST_RESULTS_VERSION,
            savedAt: now,
            completedAt: now,
            ...result,
          },
          ...runs,
        ].slice(0, MAX_PERSISTED_RUNS),
      })
    )
  } catch {
    // Storage failures must not interrupt an active load test.
  }
}

export function clearPersistedLoadTestRuns(userId: number | undefined): void {
  if (typeof window === 'undefined' || !userId) return

  try {
    window.localStorage.removeItem(storageKey(userId))
  } catch {
    // Storage failures must not interrupt an active load test.
  }
}
