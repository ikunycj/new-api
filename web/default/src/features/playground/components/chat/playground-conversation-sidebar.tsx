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
import { HistoryIcon, PlusIcon, Trash2Icon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { cn } from '@/lib/utils'

import type { PlaygroundConversation } from '../../types'

type PlaygroundConversationSidebarProps = {
  conversations: PlaygroundConversation[]
  activeConversationId: string | null
  onNewConversation: () => void
  onSelectConversation: (id: string) => void
  onDeleteConversation: (id: string) => void
  mobile?: boolean
}

function ConversationList({
  activeConversationId,
  conversations,
  onDeleteConversation,
  onSelectConversation,
}: Omit<PlaygroundConversationSidebarProps, 'mobile' | 'onNewConversation'>) {
  const { t } = useTranslation()

  if (conversations.length === 0) {
    return (
      <div className='text-muted-foreground px-3 py-8 text-center text-sm'>
        {t('No conversation history yet')}
      </div>
    )
  }

  return (
    <ScrollArea className='min-h-0 flex-1'>
      <div className='grid gap-1 p-2'>
        {conversations.map((conversation) => {
          const title = conversation.title || t('New conversation')
          const isActive = conversation.id === activeConversationId
          return (
            <div
              className='flex min-w-0 items-center gap-1'
              key={conversation.id}
            >
              <Button
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'min-w-0 flex-1 justify-start truncate text-left font-normal',
                  isActive && 'bg-muted text-foreground'
                )}
                onClick={() => onSelectConversation(conversation.id)}
                title={title}
                variant='ghost'
              >
                <HistoryIcon className='text-muted-foreground mr-2 size-4 shrink-0' />
                <span className='truncate'>{title}</span>
              </Button>
              <Button
                aria-label={`${t('Delete conversation')}: ${title}`}
                className='text-muted-foreground hover:text-destructive shrink-0'
                onClick={() => onDeleteConversation(conversation.id)}
                size='icon-sm'
                variant='ghost'
              >
                <Trash2Icon />
              </Button>
            </div>
          )
        })}
      </div>
    </ScrollArea>
  )
}

function ConversationHistoryContent(
  props: Omit<PlaygroundConversationSidebarProps, 'mobile'>
) {
  const { t } = useTranslation()
  return (
    <div className='flex min-h-0 flex-1 flex-col'>
      <div className='flex items-center justify-between gap-2 border-b px-3 py-3'>
        <div className='flex min-w-0 items-center gap-2'>
          <HistoryIcon className='text-muted-foreground size-4 shrink-0' />
          <h2 className='truncate text-sm font-semibold'>{t('History')}</h2>
        </div>
        <Button
          aria-label={t('New conversation')}
          onClick={props.onNewConversation}
          size='icon-sm'
          variant='ghost'
        >
          <PlusIcon />
        </Button>
      </div>
      <ConversationList {...props} />
    </div>
  )
}

export function PlaygroundConversationSidebar(
  props: PlaygroundConversationSidebarProps
) {
  const { t } = useTranslation()
  const [mobileOpen, setMobileOpen] = useState(false)

  if (props.mobile) {
    return (
      <Sheet onOpenChange={setMobileOpen} open={mobileOpen}>
        <SheetTrigger
          render={
            <Button aria-label={t('History')} size='icon-sm' variant='ghost'>
              <HistoryIcon />
            </Button>
          }
        />
        <SheetContent className='w-[min(86vw,20rem)] p-0' side='left'>
          <SheetHeader className='sr-only'>
            <SheetTitle>{t('History')}</SheetTitle>
          </SheetHeader>
          <ConversationHistoryContent
            {...props}
            onNewConversation={() => {
              props.onNewConversation()
              setMobileOpen(false)
            }}
            onSelectConversation={(id) => {
              props.onSelectConversation(id)
              setMobileOpen(false)
            }}
          />
        </SheetContent>
      </Sheet>
    )
  }

  return (
    <aside className='bg-muted/10 hidden w-64 shrink-0 flex-col border-r md:flex'>
      <ConversationHistoryContent {...props} />
    </aside>
  )
}
