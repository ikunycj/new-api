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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { markUserRequestCompleted } from '@/lib/api'

import { generateImage, sendChatCompletion } from '../api'
import { ERROR_MESSAGES } from '../constants'
import {
  applyStreamingChunk,
  buildChatCompletionPayload,
  updateAssistantMessageWithError,
  updateLastAssistantMessage,
  parseRequestErrorDetails,
  applyChatCompletionResponse,
  completeAssistantMessage,
  hasChatCompletionChoice,
  isImageGenerationModel,
  isAssistantMessageFinal,
  isAssistantMessagePending,
  prepareConversationContext,
  ConversationContextError,
} from '../lib'
import type { Message, PlaygroundConfig, ParameterEnabled } from '../types'
import { useStreamRequest } from './use-stream-request'

interface UseChatHandlerOptions {
  config: PlaygroundConfig
  parameterEnabled: ParameterEnabled
  onMessageUpdate: (updater: (prev: Message[]) => Message[]) => void
  conversationSummary: string
  onConversationSummaryUpdate: (summary: string) => void
}

const KNOWN_ERROR_MESSAGES = new Set<string>(Object.values(ERROR_MESSAGES))
const STREAM_UPDATE_FLUSH_MS = 50

type PendingStreamChunks = {
  content: string
  reasoning: string
}

function mergePendingStreamChunk(
  currentChunk: string,
  nextChunk: string
): string {
  if (!currentChunk || !nextChunk.startsWith(currentChunk)) {
    return currentChunk + nextChunk
  }

  return nextChunk
}

/**
 * Hook for handling chat message sending and receiving
 */
export function useChatHandler({
  config,
  parameterEnabled,
  onMessageUpdate,
  conversationSummary,
  onConversationSummaryUpdate,
}: UseChatHandlerOptions) {
  const { t } = useTranslation()
  const { sendStreamRequest, stopStream, isStreaming } = useStreamRequest()
  const [isRequesting, setIsRequesting] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const requestIdRef = useRef(0)
  const completedRequestIdsRef = useRef<Set<number>>(new Set())
  const pendingStreamChunksRef = useRef<PendingStreamChunks>({
    content: '',
    reasoning: '',
  })
  const streamFlushTimerRef = useRef<number | null>(null)

  const flushStreamUpdates = useCallback(() => {
    if (streamFlushTimerRef.current !== null) {
      window.clearTimeout(streamFlushTimerRef.current)
      streamFlushTimerRef.current = null
    }

    const pendingChunks = pendingStreamChunksRef.current
    if (!pendingChunks.reasoning && !pendingChunks.content) {
      return
    }

    pendingStreamChunksRef.current = { content: '', reasoning: '' }
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) => {
        let updatedMessage = message

        if (pendingChunks.reasoning) {
          updatedMessage = applyStreamingChunk(
            updatedMessage,
            'reasoning',
            pendingChunks.reasoning
          )
        }

        if (pendingChunks.content) {
          updatedMessage = applyStreamingChunk(
            updatedMessage,
            'content',
            pendingChunks.content
          )
        }

        return updatedMessage
      })
    )
  }, [onMessageUpdate])

  const scheduleStreamFlush = useCallback(() => {
    if (streamFlushTimerRef.current !== null) {
      return
    }

    streamFlushTimerRef.current = window.setTimeout(
      flushStreamUpdates,
      STREAM_UPDATE_FLUSH_MS
    )
  }, [flushStreamUpdates])

  useEffect(
    () => () => {
      if (streamFlushTimerRef.current !== null) {
        window.clearTimeout(streamFlushTimerRef.current)
      }
    },
    []
  )

  const getDisplayError = useCallback(
    (error: string) => {
      if (KNOWN_ERROR_MESSAGES.has(error)) {
        return t(error)
      }

      const connectionClosedSuffix = `: ${ERROR_MESSAGES.CONNECTION_CLOSED}`
      if (error.endsWith(connectionClosedSuffix)) {
        return `${error.slice(0, -ERROR_MESSAGES.CONNECTION_CLOSED.length)}${t(
          ERROR_MESSAGES.CONNECTION_CLOSED
        )}`
      }

      return error
    },
    [t]
  )

  // A stream can deliver its terminal event more than once while it is being
  // closed. Keep the optimistic onboarding counter idempotent per request.
  const markRequestCompletedOnce = useCallback((requestId: number) => {
    if (completedRequestIdsRef.current.has(requestId)) return
    completedRequestIdsRef.current.add(requestId)
    if (completedRequestIdsRef.current.size > 64) {
      const oldestRequestId = [...completedRequestIdsRef.current][0]
      if (oldestRequestId !== undefined) {
        completedRequestIdsRef.current.delete(oldestRequestId)
      }
    }
    markUserRequestCompleted()
  }, [])

  // Handle stream update
  const handleStreamUpdate = useCallback(
    (type: 'reasoning' | 'content', chunk: string) => {
      pendingStreamChunksRef.current[type] = mergePendingStreamChunk(
        pendingStreamChunksRef.current[type],
        chunk
      )
      scheduleStreamFlush()
    },
    [scheduleStreamFlush]
  )

  // Handle stream complete
  const handleStreamComplete = useCallback(
    (requestId: number) => {
      flushStreamUpdates()
      setIsRequesting(false)
      abortControllerRef.current = null
      markRequestCompletedOnce(requestId)
      onMessageUpdate((prev) =>
        updateLastAssistantMessage(prev, (message) =>
          isAssistantMessageFinal(message)
            ? message
            : completeAssistantMessage(message)
        )
      )
    },
    [flushStreamUpdates, markRequestCompletedOnce, onMessageUpdate]
  )

  // Handle stream error
  const handleStreamError = useCallback(
    (error: string, errorCode?: string) => {
      flushStreamUpdates()
      setIsRequesting(false)
      abortControllerRef.current = null
      const displayError = getDisplayError(error)
      toast.error(displayError)
      const errorTitle = t(ERROR_MESSAGES.API_REQUEST_ERROR)
      onMessageUpdate((prev) =>
        updateAssistantMessageWithError(
          prev,
          displayError,
          errorCode,
          errorTitle
        )
      )
    },
    [flushStreamUpdates, getDisplayError, onMessageUpdate, t]
  )

  const prepareContext = useCallback(
    async (messages: Message[], signal: AbortSignal) => {
      if (!config.conversationContext) {
        const preparedMessages = messages
          .filter(
            (message) =>
              message.from !== 'assistant' || message.versions[0]?.content
          )
          .map((message) => ({
            role: message.from,
            content: message.versions[0]?.content ?? '',
          }))
        const latestUserMessage = [...preparedMessages]
          .reverse()
          .find((message) => message.role === 'user')
        return {
          messages: preparedMessages,
          imagePrompt: latestUserMessage?.content ?? '',
          summary: conversationSummary,
        }
      }

      const prepared = await prepareConversationContext(messages, {
        model: config.model,
        existingSummary: conversationSummary,
        config,
        parameterEnabled,
        compressSummary: async (source, maxTokens) => {
          const response = await sendChatCompletion(
            {
              model: config.model,
              group: config.group,
              messages: [
                {
                  role: 'system',
                  content:
                    'Summarize the conversation for future context. Preserve user goals, decisions, constraints, important facts, visual details, and unresolved questions. Return only the concise summary.',
                },
                { role: 'user', content: source },
              ],
              stream: false,
              temperature: 0,
              max_tokens: maxTokens,
            },
            signal
          )
          const summary = response.choices?.[0]?.message?.content?.trim()
          if (!summary) throw new Error('summary-failed')
          return summary
        },
      })
      if (prepared.summary !== conversationSummary) {
        onConversationSummaryUpdate(prepared.summary)
      }
      return prepared
    },
    [config, conversationSummary, onConversationSummaryUpdate, parameterEnabled]
  )

  const getContextErrorMessage = useCallback(
    (error: unknown) => {
      if (error instanceof ConversationContextError) {
        if (error.message === 'summary-failed') {
          return t('Failed to compress conversation history')
        }
        if (error.message === 'context-limit-unknown') {
          return t(
            'The selected model does not publish a context limit. Set one to enable safe history compression.'
          )
        }
        return t('The current prompt exceeds the model context limit.')
      }
      return parseRequestErrorDetails(error).errorMessage
    },
    [t]
  )

  // Send streaming chat request
  const sendStreamingChat = useCallback(
    async (messages: Message[]) => {
      const requestId = requestIdRef.current + 1
      const abortController = new AbortController()
      requestIdRef.current = requestId
      abortControllerRef.current = abortController
      setIsRequesting(true)

      try {
        const prepared = await prepareContext(messages, abortController.signal)
        if (abortController.signal.aborted) {
          abortControllerRef.current = null
          setIsRequesting(false)
          return
        }
        const payload = buildChatCompletionPayload(
          prepared.messages,
          config,
          parameterEnabled
        )
        sendStreamRequest(
          payload,
          handleStreamUpdate,
          () => handleStreamComplete(requestId),
          handleStreamError
        )
      } catch (error: unknown) {
        if (abortController.signal.aborted) {
          abortControllerRef.current = null
          setIsRequesting(false)
          return
        }
        abortControllerRef.current = null
        setIsRequesting(false)
        handleStreamError(getContextErrorMessage(error))
      }
    },
    [
      config,
      getContextErrorMessage,
      handleStreamComplete,
      handleStreamError,
      handleStreamUpdate,
      parameterEnabled,
      prepareContext,
      sendStreamRequest,
    ]
  )

  // Send non-streaming chat request
  const sendNonStreamingChat = useCallback(
    async (messages: Message[]) => {
      const requestId = requestIdRef.current + 1
      const abortController = new AbortController()

      requestIdRef.current = requestId
      abortControllerRef.current = abortController
      try {
        setIsRequesting(true)
        const prepared = await prepareContext(messages, abortController.signal)
        if (abortController.signal.aborted) return
        const payload = buildChatCompletionPayload(
          prepared.messages,
          config,
          parameterEnabled
        )
        const response = await sendChatCompletion(
          payload,
          abortController.signal
        )
        if (abortController.signal.aborted) return

        if (!hasChatCompletionChoice(response)) {
          handleStreamError(ERROR_MESSAGES.API_REQUEST_ERROR)
          return
        }

        onMessageUpdate((prev) =>
          updateLastAssistantMessage(prev, (message) => {
            const updatedMessage = applyChatCompletionResponse(
              message,
              response
            )
            return updatedMessage ?? message
          })
        )
        markRequestCompletedOnce(requestId)
      } catch (error: unknown) {
        if (abortController.signal.aborted) return
        const details = parseRequestErrorDetails(error)
        handleStreamError(
          error instanceof ConversationContextError
            ? getContextErrorMessage(error)
            : details.errorMessage,
          details.errorCode
        )
      } finally {
        if (requestIdRef.current === requestId) {
          abortControllerRef.current = null
          setIsRequesting(false)
        }
      }
    },
    [
      config,
      getContextErrorMessage,
      handleStreamError,
      markRequestCompletedOnce,
      onMessageUpdate,
      parameterEnabled,
      prepareContext,
    ]
  )

  const sendImageGeneration = useCallback(
    async (messages: Message[]) => {
      const requestId = requestIdRef.current + 1
      const abortController = new AbortController()
      requestIdRef.current = requestId
      abortControllerRef.current = abortController

      try {
        setIsRequesting(true)
        const prepared = await prepareContext(messages, abortController.signal)
        const prompt = prepared.imagePrompt.trim()
        if (!prompt) {
          handleStreamError(ERROR_MESSAGES.API_REQUEST_ERROR)
          return
        }
        const response = await generateImage(
          {
            model: config.model,
            group: config.group,
            prompt,
            n: Math.min(128, Math.max(1, Math.trunc(config.imageN))),
            size: config.imageSize,
            quality: config.imageQuality,
            response_format: 'b64_json',
          },
          abortController.signal
        )

        if (abortController.signal.aborted) return

        const images = (response.data ?? [])
          .map((image) => ({
            url:
              image.url ||
              (image.b64_json ? `data:image/png;base64,${image.b64_json}` : ''),
            ...(typeof image.revised_prompt === 'string'
              ? { revisedPrompt: image.revised_prompt }
              : {}),
          }))
          .filter((image) => image.url)

        if (images.length === 0) {
          throw new Error(t('Image generation returned no images'))
        }

        onMessageUpdate((prev) =>
          updateLastAssistantMessage(prev, (message) =>
            completeAssistantMessage({ ...message, images })
          )
        )
        markRequestCompletedOnce(requestId)
      } catch (error: unknown) {
        if (abortController.signal.aborted) return

        const { errorCode, errorMessage } = parseRequestErrorDetails(error)
        handleStreamError(
          error instanceof ConversationContextError
            ? getContextErrorMessage(error)
            : errorMessage,
          errorCode
        )
      } finally {
        if (requestIdRef.current === requestId) {
          abortControllerRef.current = null
          setIsRequesting(false)
        }
      }
    },
    [
      config,
      getContextErrorMessage,
      handleStreamError,
      markRequestCompletedOnce,
      onMessageUpdate,
      prepareContext,
      t,
    ]
  )

  // Send chat request (stream or non-stream based on config)
  const sendChat = useCallback(
    (messages: Message[]) => {
      if (isImageGenerationModel(config.model)) {
        void sendImageGeneration(messages)
        return
      }

      if (config.stream) {
        sendStreamingChat(messages)
      } else {
        sendNonStreamingChat(messages)
      }
    },
    [
      config.model,
      config.stream,
      sendImageGeneration,
      sendStreamingChat,
      sendNonStreamingChat,
    ]
  )

  // Stop generation
  const stopGeneration = useCallback(() => {
    stopStream()
    flushStreamUpdates()
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setIsRequesting(false)
    onMessageUpdate((prev) =>
      updateLastAssistantMessage(prev, (message) =>
        isAssistantMessagePending(message)
          ? completeAssistantMessage(message)
          : message
      )
    )
  }, [stopStream, flushStreamUpdates, onMessageUpdate])

  return {
    sendChat,
    stopGeneration,
    isGenerating: isStreaming || isRequesting,
  }
}
