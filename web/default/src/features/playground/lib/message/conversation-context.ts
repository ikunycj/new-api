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
import { getContextWindow } from 'tokenlens'

import type {
  ChatCompletionMessage,
  Message,
  PlaygroundConfig,
  ParameterEnabled,
} from '../../types'
import { isImageGenerationModel } from '../model/model-utils'
import { formatMessageForAPI, isValidMessage } from './message-utils'

export const CONVERSATION_WINDOW_ROUNDS = 5
export const CONVERSATION_SUMMARY_TOKEN_LIMIT = 2048

export class ConversationContextError extends Error {
  constructor(
    message: 'prompt-too-long' | 'summary-failed' | 'context-limit-unknown'
  ) {
    super(message)
    this.name = 'ConversationContextError'
  }
}

type ConversationRound = ChatCompletionMessage[]

type PrepareConversationContextOptions = {
  model: string
  existingSummary: string
  config: PlaygroundConfig
  parameterEnabled: ParameterEnabled
  compressSummary: (source: string, maxTokens: number) => Promise<string>
}

export type PreparedConversationContext = {
  messages: ChatCompletionMessage[]
  imagePrompt: string
  summary: string
}

function contentToText(content: string | Array<{ text?: string }>): string {
  if (typeof content === 'string') return content
  return content.map((part) => part.text ?? '').join('')
}

function estimateTokens(text: string): number {
  if (!text) return 0
  const cjkCharacters = (text.match(/[\u3040-\u30ff\u3400-\u9fff]/g) ?? [])
    .length
  const otherCharacters = text.length - cjkCharacters
  return Math.ceil(cjkCharacters / 1.8 + otherCharacters / 4) + 4
}

function estimateMessages(messages: ChatCompletionMessage[]): number {
  return messages.reduce((total, message) => {
    const content = Array.isArray(message.content)
      ? message.content
          .map((part) => part.text ?? part.image_url?.url ?? '')
          .join('')
      : message.content
    return total + estimateTokens(content) + 4
  }, 0)
}

function truncateForCompression(text: string, maxTokens: number): string {
  const maxCharacters = Math.max(1000, maxTokens * 4)
  if (text.length <= maxCharacters) return text
  const half = Math.floor((maxCharacters - 80) / 2)
  return `${text.slice(0, half)}\n\n[Earlier context omitted]\n\n${text.slice(-half)}`
}

function splitIntoRounds(
  messages: ChatCompletionMessage[]
): ConversationRound[] {
  const rounds: ConversationRound[] = []
  let current: ConversationRound = []

  for (const message of messages) {
    if (message.role === 'user' && current.length > 0) {
      rounds.push(current)
      current = []
    }
    current.push(message)
  }

  if (current.length > 0) rounds.push(current)
  return rounds
}

function formatRoundsForSummary(rounds: ConversationRound[]): string {
  return rounds
    .map((round, index) => {
      const content = round
        .map((message) => `${message.role}: ${contentToText(message.content)}`)
        .join('\n')
      return `Exchange ${index + 1}\n${content}`
    })
    .join('\n\n')
}

function buildContextMessages(
  summary: string,
  recentRounds: ConversationRound[],
  currentRound: ConversationRound
): ChatCompletionMessage[] {
  const summaryMessage: ChatCompletionMessage[] = summary
    ? [
        {
          role: 'system',
          content: `Conversation summary. Use it as background context and follow the recent messages when they conflict:\n${summary}`,
        },
      ]
    : []
  return [...summaryMessage, ...recentRounds.flat(), ...currentRound]
}

function buildImagePrompt(
  summary: string,
  recentRounds: ConversationRound[],
  currentRound: ConversationRound
): string {
  const recent = [...recentRounds, currentRound]
    .flat()
    .map((message) => `${message.role}: ${contentToText(message.content)}`)
    .join('\n')
  return [
    summary ? `Conversation summary:\n${summary}` : '',
    recent ? `Recent conversation:\n${recent}` : '',
    'Create the image requested by the latest user message. Preserve relevant visual details from the context.',
  ]
    .filter(Boolean)
    .join('\n\n')
}

export async function prepareConversationContext(
  messages: Message[],
  options: PrepareConversationContextOptions
): Promise<PreparedConversationContext> {
  const apiMessages = messages.filter(isValidMessage).map(formatMessageForAPI)
  const rounds = splitIntoRounds(apiMessages)
  const currentRound = rounds.at(-1) ?? []
  const previousRounds = rounds.slice(0, -1)
  let windowSize = Math.min(CONVERSATION_WINDOW_ROUNDS, previousRounds.length)
  const inputBudget = getInputBudget(options)
  let summary = options.existingSummary
  let lastSummarySource = ''

  if (inputBudget === null) {
    if (previousRounds.length > CONVERSATION_WINDOW_ROUNDS) {
      throw new ConversationContextError('context-limit-unknown')
    }
    const contextMessages = buildContextMessages(
      summary,
      previousRounds,
      currentRound
    )
    return {
      messages: contextMessages,
      imagePrompt: buildImagePrompt(summary, previousRounds, currentRound),
      summary,
    }
  }
  if (inputBudget <= 0) {
    throw new ConversationContextError('prompt-too-long')
  }

  while (windowSize >= 0) {
    const olderRounds = previousRounds.slice(
      0,
      previousRounds.length - windowSize
    )
    const recentRounds = windowSize ? previousRounds.slice(-windowSize) : []
    const summarySource = [
      olderRounds.length && summary ? `Existing summary:\n${summary}` : '',
      olderRounds.length ? formatRoundsForSummary(olderRounds) : '',
    ]
      .filter(Boolean)
      .join('\n\n')

    if (summarySource && summarySource !== lastSummarySource) {
      lastSummarySource = summarySource
      const currentTokens = estimateMessages(currentRound)
      const outputBudget = Math.min(
        CONVERSATION_SUMMARY_TOKEN_LIMIT,
        inputBudget - currentTokens - 256
      )
      if (outputBudget < 32) {
        throw new ConversationContextError('prompt-too-long')
      }
      const sourceBudget = Math.max(64, inputBudget - 512)
      try {
        summary = await options.compressSummary(
          truncateForCompression(summarySource, sourceBudget),
          outputBudget
        )
      } catch {
        throw new ConversationContextError('summary-failed')
      }
      if (estimateTokens(summary) > outputBudget) {
        throw new ConversationContextError('summary-failed')
      }
    }

    const contextMessages = buildContextMessages(
      summary,
      recentRounds,
      currentRound
    )
    if (estimateMessages(contextMessages) <= inputBudget) {
      return {
        messages: contextMessages,
        imagePrompt: buildImagePrompt(summary, recentRounds, currentRound),
        summary,
      }
    }

    if (windowSize === 0) {
      throw new ConversationContextError('prompt-too-long')
    }
    windowSize -= 1
  }

  throw new ConversationContextError('prompt-too-long')
}

function getContextLimit(
  options: PrepareConversationContextOptions
): number | null {
  if (options.config.contextWindowTokens > 0) {
    return options.config.contextWindowTokens
  }
  const context = getContextWindow(options.model)
  return context.combinedMax ?? context.inputMax ?? context.totalMax ?? null
}

function getInputBudget(
  options: PrepareConversationContextOptions
): number | null {
  const contextLimit = getContextLimit(options)
  if (contextLimit === null) return null
  let reserveOutput = 4096
  if (isImageGenerationModel(options.model)) {
    reserveOutput = 0
  } else if (options.parameterEnabled.max_tokens) {
    reserveOutput = Math.max(256, options.config.max_tokens)
  }
  return contextLimit - reserveOutput
}
