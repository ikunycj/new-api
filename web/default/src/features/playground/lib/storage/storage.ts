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
import { MESSAGE_STATUS, STORAGE_KEYS } from '../../constants'
import type { PlaygroundConfig, ParameterEnabled, Message } from '../../types'
import {
  finalizeMessage,
  isAssistantMessagePending,
  sanitizeMessagesOnLoad,
} from '../message/message-streaming-utils'
import { completeAssistantTiming } from '../message/message-timing-utils'
import { hasMessageContent } from '../message/message-utils'
import {
  clearPlaygroundMessages,
  readPlaygroundMessages,
  writePlaygroundMessages,
} from './indexed-db'
import {
  MAX_LOADED_MESSAGE_CHARS,
  MAX_LOADED_MESSAGES_CHARS,
  MAX_INDEXED_DB_MESSAGES_BYTES,
  MAX_STORED_MESSAGES,
  MAX_STORED_MESSAGES_BYTES,
  STORAGE_VERSION,
  localMessagesSchema,
  messagesSchema,
  parameterEnabledSchema,
  playgroundConfigSchema,
} from './storage-schema'

type StoredEnvelope<T> = {
  version: number
  data: T
  revision?: number
  cleared?: boolean
}

type StoredMessagesCandidate = {
  messages: Message[]
  revision: number
}

const TRUNCATED_CONTENT_SUFFIX = '\n\n[...]'
const MIN_PREFIX_COLLAPSE_LENGTH = 2000
const MIN_REPEATED_SECTION_COUNT = 3
const SECTION_HEADING_LINE_PATTERN = /^#{2,6}\s+\d+\.\s+.+$/gm

let messageStorageWriteQueue: Promise<void> = Promise.resolve()
let latestMessageRevision = 0

function observeMessageRevision(revision: number | undefined): void {
  if (
    revision === undefined ||
    revision < 0 ||
    !Number.isSafeInteger(revision)
  ) {
    return
  }
  latestMessageRevision = Math.max(latestMessageRevision, revision)
}

function nextMessageRevision(): number {
  const nextRevision =
    latestMessageRevision >= Number.MAX_SAFE_INTEGER
      ? Date.now()
      : latestMessageRevision + 1
  latestMessageRevision = Math.max(Date.now(), nextRevision)
  return latestMessageRevision
}

function getStoredRevision(value: unknown): number {
  if (!value || typeof value !== 'object' || !('revision' in value)) {
    return 0
  }

  const revision = (value as { revision?: unknown }).revision
  if (typeof revision !== 'number' || !Number.isSafeInteger(revision)) {
    return 0
  }

  observeMessageRevision(revision)
  return revision
}

function isStoredClearMarker(value: unknown): boolean {
  return Boolean(
    value &&
    typeof value === 'object' &&
    (value as { cleared?: unknown }).cleared === true
  )
}

function readStoredValue(key: string): unknown | null {
  const saved = localStorage.getItem(key)
  if (!saved) return null

  return JSON.parse(saved) as unknown
}

function readStoredMessagesValue(): unknown | null {
  const saved = localStorage.getItem(STORAGE_KEYS.MESSAGES)
  if (!saved) return null

  if (saved.length > MAX_STORED_MESSAGES_BYTES) {
    localStorage.removeItem(STORAGE_KEYS.MESSAGES)
    return null
  }

  return JSON.parse(saved) as unknown
}

function parseStoredMessages(value: unknown): Message[] {
  const unwrapped = unwrapStoredValue(value)
  const fullResult = messagesSchema.safeParse(unwrapped)
  if (fullResult.success) {
    return fullResult.data as Message[]
  }

  return localMessagesSchema.parse(unwrapped) as Message[]
}

function unwrapStoredValue(value: unknown): unknown {
  if (!value || typeof value !== 'object') {
    return value
  }

  if ('version' in value && 'data' in value) {
    return (value as StoredEnvelope<unknown>).data
  }

  return value
}

function writeStoredValue<T>(
  key: string,
  data: T,
  metadata: Pick<StoredEnvelope<T>, 'revision' | 'cleared'> = {}
): void {
  const payload: StoredEnvelope<T> = {
    version: STORAGE_VERSION,
    ...metadata,
    data,
  }

  localStorage.setItem(key, JSON.stringify(payload))
}

function trimMessages(messages: Message[]): Message[] {
  if (messages.length <= MAX_STORED_MESSAGES) {
    return messages
  }

  return messages.slice(-MAX_STORED_MESSAGES)
}

function getSerializedByteLength(value: unknown): number {
  const serialized = JSON.stringify(value) ?? ''
  if (typeof TextEncoder !== 'undefined') {
    return new TextEncoder().encode(serialized).byteLength
  }

  return serialized.length * 2
}

function removeOldestImage(messages: Message[]): Message[] | null {
  for (let index = 0; index < messages.length; index++) {
    const images = messages[index].images
    if (!images || images.length === 0) continue

    const next = [...messages]
    const nextImages = images.slice(1)
    next[index] = nextImages.length
      ? { ...messages[index], images: nextImages }
      : { ...messages[index], images: undefined }
    return next
  }

  return null
}

function prepareMessagesForIndexedDb(messages: Message[]): Message[] {
  let prepared = messages

  if (getSerializedByteLength(prepared) <= MAX_INDEXED_DB_MESSAGES_BYTES) {
    return prepared
  }

  // Images are the dominant part of a generated-image snapshot. Remove the
  // oldest image payloads first so the text conversation remains available.
  while (getSerializedByteLength(prepared) > MAX_INDEXED_DB_MESSAGES_BYTES) {
    const next = removeOldestImage(prepared)
    if (!next) break
    prepared = next
  }

  // If text alone is still too large, retain the most recent messages. The
  // normal load-time limits still apply when the snapshot is read back.
  while (
    prepared.length > 1 &&
    getSerializedByteLength(prepared) > MAX_INDEXED_DB_MESSAGES_BYTES
  ) {
    prepared = prepared.slice(1)
  }

  return prepared
}

function getMessageSize(message: Message): number {
  const versionsSize = message.versions.reduce(
    (total, version) => total + version.content.length,
    0
  )
  const reasoningSize = message.reasoning?.content.length ?? 0

  return versionsSize + reasoningSize
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text
  }

  if (maxLength <= TRUNCATED_CONTENT_SUFFIX.length) {
    return text.slice(0, maxLength)
  }

  return `${text.slice(0, maxLength - TRUNCATED_CONTENT_SUFFIX.length)}${TRUNCATED_CONTENT_SUFFIX}`
}

type SectionOccurrence = {
  heading: string
  index: number
}

function getSectionOccurrences(text: string): SectionOccurrence[] {
  const occurrences: SectionOccurrence[] = []
  const matches = text.matchAll(SECTION_HEADING_LINE_PATTERN)
  for (const match of matches) {
    const index = match.index
    if (index === undefined) {
      continue
    }

    occurrences.push({
      heading: match[0],
      index,
    })
  }

  return occurrences
}

function getHeadingCounts(
  occurrences: SectionOccurrence[]
): Map<string, number> {
  const counts = new Map<string, number>()

  for (const occurrence of occurrences) {
    counts.set(occurrence.heading, (counts.get(occurrence.heading) ?? 0) + 1)
  }

  return counts
}

function findLastRepeatedSectionRunStart(text: string): number {
  const occurrences = getSectionOccurrences(text)
  const headingCounts = getHeadingCounts(occurrences)
  const lastRepeatedIndexes: number[] = []
  const seenHeadings = new Set<string>()

  for (let index = occurrences.length - 1; index >= 0; index--) {
    const occurrence = occurrences[index]
    const count = headingCounts.get(occurrence.heading) ?? 0

    if (
      count < MIN_REPEATED_SECTION_COUNT ||
      seenHeadings.has(occurrence.heading)
    ) {
      continue
    }

    seenHeadings.add(occurrence.heading)
    lastRepeatedIndexes.push(occurrence.index)
  }

  if (lastRepeatedIndexes.length === 0) {
    return -1
  }

  return Math.min(...lastRepeatedIndexes)
}

function collapseRepeatedSectionSnapshots(text: string): string {
  if (text.length < MIN_PREFIX_COLLAPSE_LENGTH) {
    return text
  }

  const lastRepeatedRunStart = findLastRepeatedSectionRunStart(text)
  if (lastRepeatedRunStart === -1) {
    return text
  }

  return text.slice(lastRepeatedRunStart)
}

function normalizeStoredMessageForLoad(message: Message): Message {
  let changed = false
  const versions = message.versions.map((version) => {
    const collapsedContent = collapseRepeatedSectionSnapshots(version.content)
    const content = truncateText(collapsedContent, MAX_LOADED_MESSAGE_CHARS)

    if (content === version.content && collapsedContent === version.content) {
      return version
    }

    changed = true
    return {
      ...version,
      content,
    }
  })

  const reasoning = message.reasoning
    ? {
        ...message.reasoning,
        content: truncateText(
          message.reasoning.content,
          MAX_LOADED_MESSAGE_CHARS
        ),
      }
    : undefined

  if (reasoning?.content !== message.reasoning?.content) {
    changed = true
  }

  const normalized = changed ? { ...message, versions, reasoning } : message

  if (!isAssistantMessagePending(normalized)) {
    return normalized
  }

  const hasContent = hasMessageContent(normalized)
  const hasReasoning = normalized.reasoning?.content.trim()

  if (!hasContent && !hasReasoning) {
    return normalized
  }

  const completedAt =
    normalized.completedAt ??
    normalized.reasoning?.completedAt ??
    normalized.startedAt ??
    normalized.createdAt ??
    Date.now()

  return completeAssistantTiming(
    {
      ...finalizeMessage(normalized),
      status: MESSAGE_STATUS.COMPLETE,
      isReasoningStreaming: false,
    },
    completedAt
  )
}

function trimMessagesByContentSize(messages: Message[]): Message[] {
  let totalSize = 0
  const result: Message[] = []

  for (let index = messages.length - 1; index >= 0; index--) {
    const message = messages[index]
    const messageSize = getMessageSize(message)

    if (
      result.length > 0 &&
      totalSize + messageSize > MAX_LOADED_MESSAGES_CHARS
    ) {
      break
    }

    totalSize += messageSize
    result.push(message)
  }

  return result.reverse()
}

type NormalizedMessages = {
  changed: boolean
  messages: Message[]
}

function normalizeLoadedMessages(messages: Message[]): NormalizedMessages {
  const normalized = messages.map(normalizeStoredMessageForLoad)
  const normalizedChanged = normalized.some(
    (message, index) => message !== messages[index]
  )
  const trimmed = trimMessages(normalized)
  const sizeTrimmed = trimMessagesByContentSize(trimmed)
  const sanitized = sanitizeMessagesOnLoad(sizeTrimmed)

  return {
    changed:
      normalizedChanged ||
      trimmed.length !== normalized.length ||
      sizeTrimmed.length !== trimmed.length ||
      sanitized !== sizeTrimmed,
    messages: sanitized,
  }
}

function mergeDurableImages(
  messages: Message[],
  durableMessages: Message[]
): Message[] {
  const durableImagesByKey = new Map(
    durableMessages
      .filter((message) => message.images && message.images.length > 0)
      .map((message) => [message.key, message.images])
  )

  return messages.map((message) => {
    if (message.images && message.images.length > 0) {
      return message
    }

    const durableImages = durableImagesByKey.get(message.key)
    return durableImages ? { ...message, images: durableImages } : message
  })
}

function readMessagesClearMarkerRevision(): number {
  try {
    const saved = readStoredValue(STORAGE_KEYS.MESSAGES_CLEARED)
    if (!isStoredClearMarker(saved)) return 0

    const revision = getStoredRevision(saved)
    return revision > 0 ? revision : 1
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to read playground clear marker:', error)
    return 0
  }
}

function readLocalMessagesCandidate(): StoredMessagesCandidate | null {
  const saved = readStoredMessagesValue()
  if (!saved || isStoredClearMarker(saved)) return null

  const parsed = parseStoredMessages(saved)
  const normalized = normalizeLoadedMessages(parsed)

  if (normalized.changed) {
    writeLocalMessagesBackup(
      normalized.messages,
      getStoredRevision(saved),
      false
    )
  }

  return {
    messages: normalized.messages,
    revision: getStoredRevision(saved),
  }
}

function writeLocalMessagesBackup(
  messages: Message[],
  revision: number = nextMessageRevision(),
  clearMarker = true
): void {
  const localMessages = localMessagesSchema.parse(messages)
  writeStoredValue(STORAGE_KEYS.MESSAGES, localMessages, { revision })
  if (clearMarker) {
    localStorage.removeItem(STORAGE_KEYS.MESSAGES_CLEARED)
  }
}

function enqueueMessageStorageWrite(
  operation: () => Promise<boolean>
): Promise<void> {
  const queuedWrite = messageStorageWriteQueue.then(async () => {
    const succeeded = await operation()
    if (!succeeded) {
      // eslint-disable-next-line no-console
      console.warn('IndexedDB playground message write was not completed')
    }
  })

  messageStorageWriteQueue = queuedWrite.catch((error: unknown) => {
    // eslint-disable-next-line no-console
    console.error('Failed to persist playground messages:', error)
  })
  return messageStorageWriteQueue
}

/**
 * Load playground config from localStorage
 */
export function loadConfig(): Partial<PlaygroundConfig> {
  try {
    const saved = readStoredValue(STORAGE_KEYS.CONFIG)
    if (!saved) return {}

    return playgroundConfigSchema.parse(unwrapStoredValue(saved))
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to load config:', error)
  }
  return {}
}

/**
 * Save playground config to localStorage
 */
export function saveConfig(config: Partial<PlaygroundConfig>): void {
  try {
    const parsed = playgroundConfigSchema.parse(config)
    writeStoredValue(STORAGE_KEYS.CONFIG, parsed)
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to save config:', error)
  }
}

/**
 * Load parameter enabled state from localStorage
 */
export function loadParameterEnabled(): Partial<ParameterEnabled> {
  try {
    const saved = readStoredValue(STORAGE_KEYS.PARAMETER_ENABLED)
    if (!saved) return {}

    return parameterEnabledSchema.parse(unwrapStoredValue(saved))
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to load parameter enabled:', error)
  }
  return {}
}

/**
 * Save parameter enabled state to localStorage
 */
export function saveParameterEnabled(
  parameterEnabled: Partial<ParameterEnabled>
): void {
  try {
    const parsed = parameterEnabledSchema.parse(parameterEnabled)
    writeStoredValue(STORAGE_KEYS.PARAMETER_ENABLED, parsed)
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to save parameter enabled:', error)
  }
}

/**
 * Load messages from localStorage
 */
export function loadMessages(): Message[] | null {
  try {
    const localCandidate = readLocalMessagesCandidate()
    const clearMarkerRevision = readMessagesClearMarkerRevision()
    if (
      clearMarkerRevision > 0 &&
      clearMarkerRevision >= (localCandidate?.revision ?? 0)
    ) {
      return null
    }
    return localCandidate?.messages ?? null
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to load messages:', error)
  }
  return null
}

/**
 * Load the durable IndexedDB snapshot, falling back to the lightweight
 * localStorage copy when IndexedDB is unavailable or has no data yet.
 */
export async function loadMessagesWithImages(): Promise<Message[] | null> {
  try {
    await messageStorageWriteQueue
    const stored = await readPlaygroundMessages()

    const localCandidate = (() => {
      try {
        return readLocalMessagesCandidate()
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Failed to load local message backup:', error)
        return null
      }
    })()
    const clearMarkerRevision = readMessagesClearMarkerRevision()

    if (stored.status === 'success') {
      const recordRevision = getStoredRevision(stored.record)
      const localCandidateRevision = localCandidate?.revision ?? 0
      const clearMarkerWins =
        clearMarkerRevision > 0 &&
        clearMarkerRevision >= recordRevision &&
        clearMarkerRevision >= localCandidateRevision
      const newerLocalCandidate =
        localCandidate &&
        localCandidate.revision > recordRevision &&
        localCandidate.revision > clearMarkerRevision
          ? localCandidate
          : null
      if (
        clearMarkerWins ||
        (stored.record.cleared === true && !newerLocalCandidate)
      ) {
        return null
      }

      if (newerLocalCandidate) {
        try {
          const durableMessages = stored.record.cleared
            ? []
            : (messagesSchema.parse(stored.record.data) as Message[])
          const reconciledMessages = mergeDurableImages(
            newerLocalCandidate.messages,
            durableMessages
          )
          await saveMessages(reconciledMessages)
          return reconciledMessages
        } catch (error) {
          // eslint-disable-next-line no-console
          console.error(
            'Failed to merge images from the durable playground snapshot:',
            error
          )
          await saveMessages(newerLocalCandidate.messages)
          return newerLocalCandidate.messages
        }
      }

      if (stored.record.version > STORAGE_VERSION) {
        // Keep an unknown newer snapshot intact for a future frontend version.
        return localCandidate?.messages ?? null
      }

      try {
        const parsed = messagesSchema.parse(stored.record.data) as Message[]
        const normalized = normalizeLoadedMessages(parsed)

        if (
          normalized.changed ||
          stored.record.version !== STORAGE_VERSION ||
          recordRevision === 0
        ) {
          await saveMessages(normalized.messages)
        } else {
          try {
            writeLocalMessagesBackup(normalized.messages, recordRevision)
          } catch (error) {
            // eslint-disable-next-line no-console
            console.error('Failed to save local message backup:', error)
          }
        }

        return normalized.messages
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Failed to load messages from IndexedDB:', error)
        return localCandidate?.messages ?? null
      }
    }

    if (stored.status === 'empty') {
      if (
        clearMarkerRevision > 0 &&
        clearMarkerRevision >= (localCandidate?.revision ?? 0)
      ) {
        return null
      }
      if (localCandidate) {
        await saveMessages(localCandidate.messages)
        return localCandidate.messages
      }
    }

    // An unavailable/error IndexedDB must not be treated as an empty store;
    // doing so could overwrite a valid durable snapshot with an older backup.
    return clearMarkerRevision > 0 &&
      clearMarkerRevision >= (localCandidate?.revision ?? 0)
      ? null
      : (localCandidate?.messages ?? null)
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to restore playground messages:', error)
    return loadMessages()
  }
}

/**
 * Save a full snapshot to IndexedDB and a text-only fallback to localStorage.
 */
export function saveMessages(messages: Message[]): Promise<void> {
  const revision = nextMessageRevision()
  let parsed: Message[]
  try {
    const trimmed = trimMessages(messages)
    parsed = messagesSchema.parse(trimmed) as Message[]
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to save messages:', error)
    return Promise.resolve()
  }

  try {
    writeLocalMessagesBackup(parsed, revision)
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to save local message backup:', error)
  }

  const durableMessages = prepareMessagesForIndexedDb(parsed)

  return enqueueMessageStorageWrite(() =>
    writePlaygroundMessages({
      version: STORAGE_VERSION,
      revision,
      data: durableMessages,
    })
  )
}

/**
 * Clear all playground data
 */
export function clearPlaygroundData(): Promise<void> {
  const revision = nextMessageRevision()
  try {
    // Write the tombstone first. If a later removeItem call is interrupted,
    // the stale IndexedDB snapshot is still prevented from being restored.
    writeStoredValue(STORAGE_KEYS.MESSAGES_CLEARED, null, {
      revision,
      cleared: true,
    })
    localStorage.removeItem(STORAGE_KEYS.CONFIG)
    localStorage.removeItem(STORAGE_KEYS.PARAMETER_ENABLED)
    localStorage.removeItem(STORAGE_KEYS.MESSAGES)
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to clear playground data:', error)
  }

  return enqueueMessageStorageWrite(() =>
    clearPlaygroundMessages({
      version: STORAGE_VERSION,
      revision,
      cleared: true,
      data: null,
    })
  )
}
