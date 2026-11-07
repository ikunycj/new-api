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
import { z } from 'zod'

import type { PlaygroundConversation } from '../../types'
import {
  deletePlaygroundConversation,
  loadPlaygroundConversations as loadRecords,
  savePlaygroundConversation as saveRecord,
} from './indexed-db'
import { prepareMessagesForIndexedDb } from './storage'
import { messagesSchema } from './storage-schema'

const conversationRecordSchema = z.object({
  id: z.string().min(1),
  title: z.string(),
  updatedAt: z.number(),
  data: messagesSchema,
  summary: z.string(),
})

export async function loadPlaygroundConversations(): Promise<
  PlaygroundConversation[]
> {
  const records = await loadRecords()
  return records
    .flatMap((record) => {
      const parsed = conversationRecordSchema.safeParse(record)
      if (!parsed.success) return []

      return [
        {
          id: parsed.data.id,
          title: parsed.data.title,
          updatedAt: parsed.data.updatedAt,
          messages: parsed.data.data,
          summary: parsed.data.summary,
        },
      ]
    })
    .sort((left, right) => right.updatedAt - left.updatedAt)
}

export function savePlaygroundConversation(
  conversation: PlaygroundConversation
): Promise<boolean> {
  const durableMessages = prepareMessagesForIndexedDb(conversation.messages)
  const parsed = conversationRecordSchema.safeParse({
    id: conversation.id,
    title: conversation.title,
    updatedAt: conversation.updatedAt,
    data: durableMessages,
    summary: conversation.summary,
  })
  if (!parsed.success) return Promise.resolve(false)
  return saveRecord(parsed.data)
}

export function removePlaygroundConversation(id: string): Promise<boolean> {
  return deletePlaygroundConversation(id)
}
