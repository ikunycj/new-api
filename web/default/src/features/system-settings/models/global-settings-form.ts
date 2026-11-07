/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import * as z from 'zod'

export type ChatCompletionsToResponsesPolicyValues = {
  enabled: boolean
  all_channels: boolean
  channel_ids: number[]
  channel_types: number[]
  model_patterns: string[]
}

// Keep the shorter name available to callers that do not use the form-specific
// suffix. The values are intentionally the same structured shape used by the
// editor, not the optional-field shape persisted in the backend JSON.
export type ChatCompletionsToResponsesPolicy =
  ChatCompletionsToResponsesPolicyValues

export type GlobalModelSettingsFormValues = {
  global: {
    pass_through_request_enabled: boolean
    thinking_model_blacklist: string[]
    chat_completions_to_responses_policy: ChatCompletionsToResponsesPolicyValues
  }
  PreferredModels: string[]
  general_setting: {
    ping_interval_enabled: boolean
    ping_interval_seconds: number
  }
}

export type GlobalSettingsFormValues = GlobalModelSettingsFormValues

export type GlobalModelSettingsRawValues = {
  'global.pass_through_request_enabled': boolean
  'global.thinking_model_blacklist': string
  'global.chat_completions_to_responses_policy': string
  PreferredModels: string
  'general_setting.ping_interval_enabled': boolean
  'general_setting.ping_interval_seconds': number
}

export type FlatGlobalModelSettings = GlobalModelSettingsRawValues

export const DEFAULT_THINKING_MODEL_BLACKLIST = [
  'moonshotai/kimi-k2-thinking',
  'kimi-k2-thinking',
] as const

export const DEFAULT_PREFERRED_MODELS = [
  'gpt-5.6-sol',
  'claude-fable-5',
] as const

export const DEFAULT_CHAT_COMPLETIONS_TO_RESPONSES_POLICY = {
  enabled: false,
  all_channels: true,
  channel_ids: [],
  channel_types: [],
  model_patterns: [],
} satisfies ChatCompletionsToResponsesPolicyValues

const positiveSafeIntegerSchema = z
  .number()
  .refine(Number.isSafeInteger, 'Must be a safe integer')
  .positive('Must be greater than 0')

export const chatCompletionsToResponsesPolicySchema = z
  .object({
    enabled: z.boolean(),
    all_channels: z.boolean(),
    channel_ids: z.array(positiveSafeIntegerSchema),
    channel_types: z.array(positiveSafeIntegerSchema),
    model_patterns: z.array(z.string().trim().min(1)),
  })
  .superRefine((policy, context) => {
    if (policy.enabled && policy.model_patterns.length === 0) {
      context.addIssue({
        code: 'custom',
        path: ['model_patterns'],
        message: 'Add at least one model pattern when the policy is enabled',
      })
    }

    if (
      policy.enabled &&
      !policy.all_channels &&
      policy.channel_ids.length === 0 &&
      policy.channel_types.length === 0
    ) {
      context.addIssue({
        code: 'custom',
        path: ['channel_ids'],
        message:
          'Select a channel or channel type when using selected channels',
      })
    }

    const seenPatterns = new Set<string>()
    for (const [index, pattern] of policy.model_patterns.entries()) {
      if (seenPatterns.has(pattern)) {
        context.addIssue({
          code: 'custom',
          path: ['model_patterns', index],
          message: 'Duplicate model patterns are not allowed',
        })
        continue
      }
      seenPatterns.add(pattern)
      try {
        // Go's regexp syntax is close enough for the common patterns entered here.
        new RegExp(pattern)
      } catch {
        context.addIssue({
          code: 'custom',
          path: ['model_patterns', index],
          message: 'Enter a valid regular expression',
        })
      }
    }
  })

export const globalModelSettingsFormSchema = z.object({
  global: z.object({
    pass_through_request_enabled: z.boolean(),
    thinking_model_blacklist: z.array(z.string().trim().min(1)),
    chat_completions_to_responses_policy:
      chatCompletionsToResponsesPolicySchema,
  }),
  PreferredModels: z.array(z.string().trim().min(1)),
  general_setting: z.object({
    ping_interval_enabled: z.boolean(),
    ping_interval_seconds: z.coerce.number().int().min(1),
  }),
})

export type GlobalModelSettingsFormInput = z.input<
  typeof globalModelSettingsFormSchema
>

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) return []
  const seen = new Set<string>()
  const result: string[] = []
  for (const value of values) {
    if (typeof value !== 'string') continue
    const trimmed = value.trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    result.push(trimmed)
  }
  return result
}

function uniquePositiveIntegers(values: unknown): number[] {
  if (!Array.isArray(values)) return []
  const seen = new Set<number>()
  const result: number[] = []
  for (const value of values) {
    let numberValue = Number.NaN
    if (typeof value === 'number') {
      numberValue = value
    } else if (typeof value === 'string' && value.trim() !== '') {
      numberValue = Number(value)
    }
    if (!Number.isSafeInteger(numberValue) || numberValue < 1) continue
    if (seen.has(numberValue)) continue
    seen.add(numberValue)
    result.push(numberValue)
  }
  return result.sort((left, right) => left - right)
}

function parseJson(raw: unknown, fallback: unknown): unknown {
  if (typeof raw !== 'string') return raw ?? fallback
  if (!raw.trim()) return fallback
  try {
    return JSON.parse(raw)
  } catch {
    return fallback
  }
}

function parseBoolean(raw: unknown, fallback: boolean): boolean {
  if (typeof raw === 'boolean') return raw
  if (typeof raw !== 'string') return fallback

  switch (raw.trim().toLowerCase()) {
    case 'true':
    case '1':
      return true
    case 'false':
    case '0':
      return false
    default:
      return fallback
  }
}

function parsePolicy(
  raw: unknown,
  fallback: ChatCompletionsToResponsesPolicyValues = DEFAULT_CHAT_COMPLETIONS_TO_RESPONSES_POLICY
): ChatCompletionsToResponsesPolicyValues {
  const parsed = parseJson(raw, fallback)
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    return {
      enabled: fallback.enabled,
      all_channels: fallback.all_channels,
      channel_ids: [...fallback.channel_ids],
      channel_types: [...fallback.channel_types],
      model_patterns: [...fallback.model_patterns],
    }
  }
  const value = parsed as Record<string, unknown>
  return {
    enabled: parseBoolean(value.enabled, fallback.enabled),
    all_channels: parseBoolean(value.all_channels, fallback.all_channels),
    channel_ids:
      'channel_ids' in value
        ? uniquePositiveIntegers(value.channel_ids)
        : [...fallback.channel_ids],
    channel_types:
      'channel_types' in value
        ? uniquePositiveIntegers(value.channel_types)
        : [...fallback.channel_types],
    model_patterns:
      'model_patterns' in value
        ? uniqueStrings(value.model_patterns)
        : [...fallback.model_patterns],
  }
}

export function parseStringList(
  value: unknown,
  fallback: readonly string[] = []
): string[] {
  const parsed = parseJson(value, undefined)
  if (!Array.isArray(parsed)) return [...fallback]
  return uniqueStrings(parsed)
}

export function normalizeStringList(value: unknown): string[] {
  return parseStringList(value)
}

export function parseThinkingModelBlacklist(value: unknown): string[] {
  return parseStringList(value, DEFAULT_THINKING_MODEL_BLACKLIST)
}

export function parsePreferredModels(value: unknown): string[] {
  return parseStringList(value, DEFAULT_PREFERRED_MODELS)
}

export function normalizeChatCompletionsToResponsesPolicy(
  value: unknown,
  fallback: ChatCompletionsToResponsesPolicyValues = DEFAULT_CHAT_COMPLETIONS_TO_RESPONSES_POLICY
): ChatCompletionsToResponsesPolicyValues {
  return parsePolicy(value, fallback)
}

export function serializeStringList(value: readonly string[]): string {
  return JSON.stringify(uniqueStrings(value))
}

export function serializeThinkingModelBlacklist(
  value: readonly string[]
): string {
  return serializeStringList(value)
}

export function serializePreferredModels(value: readonly string[]): string {
  return serializeStringList(value)
}

export function serializeChatCompletionsToResponsesPolicy(
  value: ChatCompletionsToResponsesPolicyValues
): string {
  const normalized = parsePolicy(value)
  const serialized: Record<string, unknown> = {
    enabled: normalized.enabled,
    all_channels: normalized.all_channels,
  }
  if (!normalized.all_channels) {
    if (normalized.channel_ids.length > 0) {
      serialized.channel_ids = normalized.channel_ids
    }
    if (normalized.channel_types.length > 0) {
      serialized.channel_types = normalized.channel_types
    }
  }
  if (normalized.model_patterns.length > 0) {
    serialized.model_patterns = normalized.model_patterns
  }
  return JSON.stringify(serialized)
}

export function parseGlobalModelSettings(
  values: Partial<GlobalModelSettingsRawValues>
): GlobalModelSettingsFormValues {
  return {
    global: {
      pass_through_request_enabled: parseBoolean(
        values['global.pass_through_request_enabled'],
        false
      ),
      thinking_model_blacklist: parseThinkingModelBlacklist(
        values['global.thinking_model_blacklist'] ?? ''
      ),
      chat_completions_to_responses_policy:
        normalizeChatCompletionsToResponsesPolicy(
          values['global.chat_completions_to_responses_policy'] ?? ''
        ),
    },
    PreferredModels: parsePreferredModels(values.PreferredModels ?? ''),
    general_setting: {
      ping_interval_enabled: parseBoolean(
        values['general_setting.ping_interval_enabled'],
        false
      ),
      ping_interval_seconds: Number.isFinite(
        Number(values['general_setting.ping_interval_seconds'])
      )
        ? Math.max(
            1,
            Math.trunc(Number(values['general_setting.ping_interval_seconds']))
          )
        : 60,
    },
  }
}

export function serializeGlobalModelSettings(
  values: GlobalModelSettingsFormValues
): GlobalModelSettingsRawValues {
  return {
    'global.pass_through_request_enabled':
      values.global.pass_through_request_enabled,
    'global.thinking_model_blacklist': serializeThinkingModelBlacklist(
      values.global.thinking_model_blacklist
    ),
    'global.chat_completions_to_responses_policy':
      serializeChatCompletionsToResponsesPolicy(
        values.global.chat_completions_to_responses_policy
      ),
    PreferredModels: serializePreferredModels(values.PreferredModels),
    'general_setting.ping_interval_enabled':
      values.general_setting.ping_interval_enabled,
    'general_setting.ping_interval_seconds': Math.max(
      1,
      Math.trunc(values.general_setting.ping_interval_seconds)
    ),
  }
}

export function parseGlobalSettingsDefaults(
  values: Partial<GlobalModelSettingsRawValues>
): GlobalSettingsFormValues {
  return parseGlobalModelSettings(values)
}

export function flattenGlobalValues(
  values: GlobalSettingsFormValues
): FlatGlobalModelSettings {
  return serializeGlobalModelSettings(values)
}

export function canonicalizeGlobalModelSettings(
  values: GlobalModelSettingsRawValues | GlobalModelSettingsFormValues
): GlobalModelSettingsRawValues {
  if ('global.pass_through_request_enabled' in values) {
    return serializeGlobalModelSettings(parseGlobalModelSettings(values))
  }
  return serializeGlobalModelSettings(values)
}

export const canonicalizeGlobalSettings = canonicalizeGlobalModelSettings

/** Canonicalize arbitrary JSON text by sorting object keys recursively. */
export function canonicalizeJson(value: unknown, fallback = '{}'): string {
  const parsed = parseJson(value, undefined)
  if (parsed === undefined) return fallback
  return JSON.stringify(sortJsonKeys(parsed)) ?? fallback
}

function sortJsonKeys(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sortJsonKeys)
  if (typeof value !== 'object' || value === null) return value

  return Object.keys(value as Record<string, unknown>)
    .sort()
    .reduce<Record<string, unknown>>((result, key) => {
      const record = value as Record<string, unknown>
      result[key] = sortJsonKeys(record[key])
      return result
    }, {})
}

export function isValidRegularExpression(pattern: unknown): pattern is string {
  if (typeof pattern !== 'string' || !pattern.trim()) return false
  try {
    new RegExp(pattern)
    return true
  } catch {
    return false
  }
}

export const isValidModelPattern = isValidRegularExpression

export const positiveChannelIdSchema = positiveSafeIntegerSchema
export const modelPatternSchema = z
  .string()
  .trim()
  .min(1, 'Pattern is required')
  .refine(isValidRegularExpression, 'Enter a valid regular expression')

export const globalSettingsFormSchema = globalModelSettingsFormSchema
