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
*/
import { nanoid } from 'nanoid'
import { useCallback, useEffect, useRef, useState } from 'react'

import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../constants'
import {
  saveConfig,
  saveParameterEnabled,
  applyMessageStateUpdate,
  getInitialParameterEnabled,
  getInitialPlaygroundConfig,
  loadMessagesWithImages,
  type MessageStateUpdater,
} from '../lib'
import {
  loadPlaygroundConversations,
  removePlaygroundConversation,
  savePlaygroundConversation,
} from '../lib/storage/conversation-storage'
import type {
  Message,
  PlaygroundConversation,
  PlaygroundConfig,
  ParameterEnabled,
  ModelOption,
  GroupOption,
} from '../types'

function createConversation(messages: Message[] = []): PlaygroundConversation {
  return {
    id: nanoid(),
    title: '',
    updatedAt: Date.now(),
    messages,
    summary: '',
  }
}

function getConversationTitle(messages: Message[]): string {
  const firstUserMessage = messages.find((message) => message.from === 'user')
  const title = firstUserMessage?.versions[0]?.content.trim() ?? ''
  if (!title) return ''
  return title.length > 40 ? `${title.slice(0, 40)}…` : title
}

function sortConversations(conversations: PlaygroundConversation[]) {
  return [...conversations].sort(
    (left, right) => right.updatedAt - left.updatedAt
  )
}

export function usePlaygroundState(
  initialModel?: string,
  initialGroup?: string
) {
  const [config, setConfig] = useState<PlaygroundConfig>(() => ({
    ...getInitialPlaygroundConfig(),
    ...(initialModel ? { model: initialModel } : {}),
    ...(initialGroup ? { group: initialGroup } : {}),
  }))
  const [parameterEnabled, setParameterEnabled] = useState<ParameterEnabled>(
    getInitialParameterEnabled
  )
  const [messages, setMessages] = useState<Message[]>([])
  const [conversationSummary, setConversationSummary] = useState('')
  const [conversations, setConversations] = useState<PlaygroundConversation[]>(
    []
  )
  const [activeConversationId, setActiveConversationId] = useState<
    string | null
  >(null)
  const [isLoadingMessages, setIsLoadingMessages] = useState(true)
  const [models, setModels] = useState<ModelOption[]>([])
  const [groups, setGroups] = useState<GroupOption[]>([])

  const messagesRef = useRef(messages)
  const summaryRef = useRef(conversationSummary)
  const conversationsRef = useRef(conversations)
  const activeConversationIdRef = useRef(activeConversationId)
  const pendingPersistRef = useRef<PlaygroundConversation | null>(null)
  const persistTimerRef = useRef<number | null>(null)

  const persistConversation = useCallback(
    (conversationId: string, nextMessages: Message[], nextSummary: string) => {
      const existing = conversationsRef.current.find(
        (conversation) => conversation.id === conversationId
      )
      const title = existing?.title || getConversationTitle(nextMessages)
      const nextConversation: PlaygroundConversation = {
        id: conversationId,
        title,
        updatedAt: Date.now(),
        messages: nextMessages,
        summary: nextSummary,
      }
      const nextConversations = sortConversations([
        nextConversation,
        ...conversationsRef.current.filter(
          (conversation) => conversation.id !== conversationId
        ),
      ])
      conversationsRef.current = nextConversations
      setConversations(nextConversations)
      pendingPersistRef.current = nextConversation
      if (persistTimerRef.current !== null) {
        window.clearTimeout(persistTimerRef.current)
      }
      persistTimerRef.current = window.setTimeout(() => {
        persistTimerRef.current = null
        const pending = pendingPersistRef.current
        pendingPersistRef.current = null
        if (pending) void savePlaygroundConversation(pending)
      }, 300)
    },
    []
  )

  useEffect(() => {
    let cancelled = false
    void (async () => {
      let loaded = await loadPlaygroundConversations()
      if (loaded.length === 0) {
        const legacyMessages = (await loadMessagesWithImages()) ?? []
        const migrated = createConversation(legacyMessages)
        loaded = [migrated]
        void savePlaygroundConversation(migrated)
      }

      if (cancelled) return
      const sorted = sortConversations(loaded)
      const active = sorted[0]
      conversationsRef.current = sorted
      setConversations(sorted)
      if (active) {
        activeConversationIdRef.current = active.id
        messagesRef.current = active.messages
        summaryRef.current = active.summary
        setActiveConversationId(active.id)
        setMessages(active.messages)
        setConversationSummary(active.summary)
      }
      setIsLoadingMessages(false)
    })().catch((error: unknown) => {
      if (cancelled) return
      // eslint-disable-next-line no-console
      console.error('Failed to initialize playground conversations:', error)
      setIsLoadingMessages(false)
    })

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(
    () => () => {
      if (persistTimerRef.current !== null) {
        window.clearTimeout(persistTimerRef.current)
      }
      if (pendingPersistRef.current) {
        void savePlaygroundConversation(pendingPersistRef.current)
      }
    },
    []
  )

  const updateMessages = useCallback(
    (updater: MessageStateUpdater) => {
      setMessages((previousMessages) => {
        const nextMessages = applyMessageStateUpdate(previousMessages, updater)
        messagesRef.current = nextMessages
        const conversationId = activeConversationIdRef.current
        if (conversationId) {
          persistConversation(conversationId, nextMessages, summaryRef.current)
        }
        return nextMessages
      })
    },
    [persistConversation]
  )

  const updateConversationSummary = useCallback(
    (summary: string) => {
      summaryRef.current = summary
      setConversationSummary(summary)
      const conversationId = activeConversationIdRef.current
      if (conversationId) {
        persistConversation(conversationId, messagesRef.current, summary)
      }
    },
    [persistConversation]
  )

  const createNewConversation = useCallback(() => {
    const next = createConversation()
    const nextConversations = sortConversations([
      next,
      ...conversationsRef.current,
    ])
    conversationsRef.current = nextConversations
    activeConversationIdRef.current = next.id
    messagesRef.current = []
    summaryRef.current = ''
    setConversations(nextConversations)
    setActiveConversationId(next.id)
    setMessages([])
    setConversationSummary('')
    void savePlaygroundConversation(next)
  }, [])

  const selectConversation = useCallback((conversationId: string) => {
    const selected = conversationsRef.current.find(
      (conversation) => conversation.id === conversationId
    )
    if (!selected) return

    activeConversationIdRef.current = selected.id
    messagesRef.current = selected.messages
    summaryRef.current = selected.summary
    setActiveConversationId(selected.id)
    setMessages(selected.messages)
    setConversationSummary(selected.summary)
  }, [])

  const deleteConversation = useCallback((conversationId: string) => {
    if (pendingPersistRef.current?.id === conversationId) {
      pendingPersistRef.current = null
      if (persistTimerRef.current !== null) {
        window.clearTimeout(persistTimerRef.current)
        persistTimerRef.current = null
      }
    }
    const remaining = conversationsRef.current.filter(
      (conversation) => conversation.id !== conversationId
    )
    void removePlaygroundConversation(conversationId)

    if (remaining.length === 0) {
      const next = createConversation()
      conversationsRef.current = [next]
      activeConversationIdRef.current = next.id
      messagesRef.current = []
      summaryRef.current = ''
      setConversations([next])
      setActiveConversationId(next.id)
      setMessages([])
      setConversationSummary('')
      void savePlaygroundConversation(next)
      return
    }

    const nextConversations = sortConversations(remaining)
    conversationsRef.current = nextConversations
    setConversations(nextConversations)
    if (activeConversationIdRef.current === conversationId) {
      const next = nextConversations[0]
      activeConversationIdRef.current = next.id
      messagesRef.current = next.messages
      summaryRef.current = next.summary
      setActiveConversationId(next.id)
      setMessages(next.messages)
      setConversationSummary(next.summary)
    }
  }, [])

  const updateConfig = useCallback(
    <K extends keyof PlaygroundConfig>(key: K, value: PlaygroundConfig[K]) => {
      setConfig((previous) => {
        const updated = { ...previous, [key]: value }
        saveConfig(updated)
        return updated
      })
    },
    []
  )

  const updateParameterEnabled = useCallback(
    (key: keyof ParameterEnabled, value: boolean) => {
      setParameterEnabled((previous) => {
        const updated = { ...previous, [key]: value }
        saveParameterEnabled(updated)
        return updated
      })
    },
    []
  )

  const clearMessages = useCallback(() => {
    summaryRef.current = ''
    setConversationSummary('')
    updateMessages([])
  }, [updateMessages])

  const resetConfig = useCallback(() => {
    setConfig(DEFAULT_CONFIG)
    setParameterEnabled(DEFAULT_PARAMETER_ENABLED)
    saveConfig(DEFAULT_CONFIG)
    saveParameterEnabled(DEFAULT_PARAMETER_ENABLED)
  }, [])

  return {
    config,
    parameterEnabled,
    messages,
    conversationSummary,
    conversations,
    activeConversationId,
    isLoadingMessages,
    models,
    groups,
    setModels,
    setGroups,
    updateConfig,
    updateParameterEnabled,
    updateMessages,
    updateConversationSummary,
    clearMessages,
    resetConfig,
    createNewConversation,
    selectConversation,
    deleteConversation,
  }
}
