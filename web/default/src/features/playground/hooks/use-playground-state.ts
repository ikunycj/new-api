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

import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../constants'
import {
  saveConfig,
  saveParameterEnabled,
  saveMessages,
  applyMessageStateUpdate,
  getInitialParameterEnabled,
  getInitialPlaygroundConfig,
  loadMessagesWithImages,
  type MessageStateUpdater,
} from '../lib'
import type {
  Message,
  PlaygroundConfig,
  ParameterEnabled,
  ModelOption,
  GroupOption,
} from '../types'

const MESSAGE_SAVE_DEBOUNCE_MS = 500

/**
 * Main state management hook for playground
 */
export function usePlaygroundState(
  initialModel?: string,
  initialGroup?: string
) {
  // Load initial state from localStorage
  const [config, setConfig] = useState<PlaygroundConfig>(() => ({
    ...getInitialPlaygroundConfig(),
    ...(initialModel ? { model: initialModel } : {}),
    ...(initialGroup ? { group: initialGroup } : {}),
  }))

  const [parameterEnabled, setParameterEnabled] = useState<ParameterEnabled>(
    getInitialParameterEnabled
  )

  const [messages, setMessages] = useState<Message[]>([])
  const [isLoadingMessages, setIsLoadingMessages] = useState(true)
  const messagesSaveTimerRef = useRef<number | null>(null)
  const messagesLoadTimerRef = useRef<number | null>(null)
  const latestMessagesRef = useRef<Message[]>(messages)
  const hasLoadedMessagesRef = useRef(false)
  const pendingMessagesRef = useRef<Message[]>([])
  const hasPendingMessagesRef = useRef(false)

  const [models, setModels] = useState<ModelOption[]>([])
  const [groups, setGroups] = useState<GroupOption[]>([])

  const persistMessages = useCallback((messagesToSave: Message[]) => {
    const previousMessages = latestMessagesRef.current
    latestMessagesRef.current = messagesToSave

    if (!hasLoadedMessagesRef.current) {
      pendingMessagesRef.current = messagesToSave
      hasPendingMessagesRef.current = true
      return
    }

    const nextMessageKeys = new Set(
      messagesToSave.map((message) => message.key)
    )
    const removedMessage = previousMessages.some(
      (message) => !nextMessageKeys.has(message.key)
    )
    const previousMessagesByKey = new Map(
      previousMessages.map((message) => [message.key, message])
    )
    const imagesChanged = messagesToSave.some(
      (message) =>
        message.images !== previousMessagesByKey.get(message.key)?.images
    )

    if (messagesSaveTimerRef.current !== null) {
      window.clearTimeout(messagesSaveTimerRef.current)
    }

    if (removedMessage || imagesChanged) {
      messagesSaveTimerRef.current = null
      void saveMessages(messagesToSave)
      return
    }

    messagesSaveTimerRef.current = window.setTimeout(() => {
      messagesSaveTimerRef.current = null
      void saveMessages(latestMessagesRef.current)
    }, MESSAGE_SAVE_DEBOUNCE_MS)
  }, [])

  useEffect(() => {
    let cancelled = false

    messagesLoadTimerRef.current = window.setTimeout(() => {
      messagesLoadTimerRef.current = null
      void loadMessagesWithImages()
        .then((storedMessages) => {
          if (cancelled) return

          const hasPendingMessages = hasPendingMessagesRef.current
          const loadedMessages = hasPendingMessages
            ? pendingMessagesRef.current
            : (storedMessages ?? [])
          latestMessagesRef.current = loadedMessages
          hasLoadedMessagesRef.current = true
          hasPendingMessagesRef.current = false
          setMessages(loadedMessages)
          setIsLoadingMessages(false)

          if (hasPendingMessages) {
            void saveMessages(loadedMessages)
          }
        })
        .catch((error: unknown) => {
          if (cancelled) return

          // eslint-disable-next-line no-console
          console.error('Failed to initialize playground messages:', error)
          const hasPendingMessages = hasPendingMessagesRef.current
          const pendingMessages = hasPendingMessages
            ? pendingMessagesRef.current
            : []
          latestMessagesRef.current = pendingMessages
          hasLoadedMessagesRef.current = true
          hasPendingMessagesRef.current = false
          setMessages(pendingMessages)
          setIsLoadingMessages(false)
          if (hasPendingMessages) {
            void saveMessages(pendingMessages)
          }
        })
    }, 0)

    return () => {
      cancelled = true
      if (messagesLoadTimerRef.current !== null) {
        window.clearTimeout(messagesLoadTimerRef.current)
        messagesLoadTimerRef.current = null
      }
      if (!hasLoadedMessagesRef.current && hasPendingMessagesRef.current) {
        void saveMessages(pendingMessagesRef.current)
      }
    }
  }, [])

  useEffect(
    () => () => {
      if (messagesSaveTimerRef.current !== null) {
        window.clearTimeout(messagesSaveTimerRef.current)
        void saveMessages(latestMessagesRef.current)
      }
    },
    []
  )

  // Update config with automatic save
  const updateConfig = useCallback(
    <K extends keyof PlaygroundConfig>(key: K, value: PlaygroundConfig[K]) => {
      setConfig((prev) => {
        const updated = { ...prev, [key]: value }
        saveConfig(updated)
        return updated
      })
    },
    []
  )

  // Update parameter enabled with automatic save
  const updateParameterEnabled = useCallback(
    (key: keyof ParameterEnabled, value: boolean) => {
      setParameterEnabled((prev) => {
        const updated = { ...prev, [key]: value }
        saveParameterEnabled(updated)
        return updated
      })
    },
    []
  )

  // Update messages with automatic save
  const updateMessages = useCallback(
    (updater: MessageStateUpdater) => {
      setMessages((prev) => {
        const newMessages = applyMessageStateUpdate(prev, updater)
        persistMessages(newMessages)
        return newMessages
      })
    },
    [persistMessages]
  )

  // Clear all messages
  const clearMessages = useCallback(() => {
    updateMessages([])
  }, [updateMessages])

  // Reset config to defaults
  const resetConfig = useCallback(() => {
    setConfig(DEFAULT_CONFIG)
    setParameterEnabled(DEFAULT_PARAMETER_ENABLED)
    saveConfig(DEFAULT_CONFIG)
    saveParameterEnabled(DEFAULT_PARAMETER_ENABLED)
  }, [])

  return {
    // State
    config,
    parameterEnabled,
    messages,
    isLoadingMessages,
    models,
    groups,

    // Setters
    setModels,
    setGroups,

    // Actions
    updateConfig,
    updateParameterEnabled,
    updateMessages,
    clearMessages,
    resetConfig,
  }
}
