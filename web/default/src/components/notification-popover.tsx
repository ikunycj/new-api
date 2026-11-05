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
import type { TFunction } from 'i18next'
import { Bell, Megaphone } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getAnnouncementColorClass } from '@/lib/colors'
import { formatDateTimeObject } from '@/lib/time'
import { cn } from '@/lib/utils'

interface AnnouncementItem {
  id?: number | string
  type?: string
  content?: string
  extra?: string
  publishDate?: string | Date
}

interface NotificationPopoverProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  unreadCount: number
  activeTab: 'notice' | 'announcements'
  onTabChange: (tab: 'notice' | 'announcements') => void
  notice: string
  announcements: AnnouncementItem[]
  noticeLoading: boolean
  announcementsLoading: boolean
  className?: string
}

/**
 * Get relative time string from a date
 */
function getRelativeTime(publishDate: string | Date, t: TFunction): string {
  if (!publishDate) return ''

  const now = new Date()
  const pubDate = new Date(publishDate)

  // If invalid date, return original string
  if (Number.isNaN(pubDate.getTime())) {
    return typeof publishDate === 'string' ? publishDate : ''
  }

  const diffMs = now.getTime() - pubDate.getTime()
  const diffSeconds = Math.floor(diffMs / 1000)
  const diffMinutes = Math.floor(diffSeconds / 60)
  const diffHours = Math.floor(diffMinutes / 60)
  const diffDays = Math.floor(diffHours / 24)
  const diffWeeks = Math.floor(diffDays / 7)
  const diffMonths = Math.floor(diffDays / 30)
  const diffYears = Math.floor(diffDays / 365)

  // If future time, show specific date
  if (diffMs < 0) return formatDateTimeObject(pubDate)

  // Return relative time based on difference
  if (diffSeconds < 60) return t('Just now')
  if (diffMinutes < 60) {
    return diffMinutes === 1
      ? t('1 minute ago')
      : t('{{count}} minutes ago', { count: diffMinutes })
  }
  if (diffHours < 24) {
    return diffHours === 1
      ? t('1 hour ago')
      : t('{{count}} hours ago', { count: diffHours })
  }
  if (diffDays < 7) {
    return diffDays === 1
      ? t('1 day ago')
      : t('{{count}} days ago', { count: diffDays })
  }
  if (diffWeeks < 4) {
    return diffWeeks === 1
      ? t('1 week ago')
      : t('{{count}} weeks ago', { count: diffWeeks })
  }
  if (diffMonths < 12) {
    return diffMonths === 1
      ? t('1 month ago')
      : t('{{count}} months ago', { count: diffMonths })
  }
  if (diffYears < 2) return t('1 year ago')

  // Over 2 years, show specific date
  return formatDateTimeObject(pubDate)
}

/**
 * Announcement status dot indicator
 */
function AnnouncementDot({ type }: { type?: string }) {
  return (
    <span
      className={cn(
        'mt-1.5 inline-block size-2 shrink-0 rounded-full',
        getAnnouncementColorClass(type)
      )}
    />
  )
}

function getAnnouncementRenderKey(announcement: AnnouncementItem): string {
  if (announcement.id !== undefined && announcement.id !== null) {
    return `id:${announcement.id}`
  }

  return JSON.stringify({
    content: announcement.content ?? '',
    extra: announcement.extra ?? '',
    publishDate: announcement.publishDate ?? '',
    type: announcement.type ?? '',
  })
}

/**
 * Render an announcement card, or empty/loading state, in the notification popover.
 */
function AnnouncementCardContent({
  announcement,
  loading,
  t,
  emptyTitle,
}: {
  announcement?: AnnouncementItem
  loading: boolean
  t: TFunction
  emptyTitle: string
}) {
  if (loading) {
    return <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
  }

  if (!announcement) {
    return <p className='text-muted-foreground text-sm'>{t(emptyTitle)}</p>
  }

  const publishDate = announcement.publishDate
    ? new Date(announcement.publishDate)
    : null
  const relativeTime = publishDate ? getRelativeTime(publishDate, t) : ''
  const absoluteTime = publishDate ? formatDateTimeObject(publishDate) : ''

  return (
    <div className='flex items-start gap-3'>
      <AnnouncementDot type={announcement.type} />
      <div className='flex min-w-0 flex-1 flex-col gap-2'>
        <div className='text-sm'>
          <RichContent breaks content={announcement.content || ''} />
        </div>
        {announcement.extra ? (
          <div className='text-muted-foreground text-xs'>
            <RichContent breaks content={announcement.extra} />
          </div>
        ) : null}
        {absoluteTime ? (
          <div className='text-muted-foreground text-xs'>
            {relativeTime ? `${relativeTime} • ` : null}
            {absoluteTime}
          </div>
        ) : null}
      </div>
    </div>
  )
}

function SystemNoticeContent({
  notice,
  loading,
  t,
}: {
  notice: string
  loading: boolean
  t: TFunction
}) {
  if (loading) {
    return <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
  }

  if (!notice) {
    return (
      <p className='text-muted-foreground text-sm'>
        {t('No announcements at this time')}
      </p>
    )
  }

  return <RichContent breaks content={notice} />
}

function AnnouncementsContent({
  announcements,
  loading,
  t,
}: {
  announcements: AnnouncementItem[]
  loading: boolean
  t: TFunction
}) {
  if (loading) {
    return (
      <p className='text-muted-foreground py-8 text-center text-sm'>
        {t('Loading...')}
      </p>
    )
  }

  if (announcements.length === 0) {
    return (
      <p className='text-muted-foreground py-8 text-center text-sm'>
        {t('No system announcements')}
      </p>
    )
  }

  return (
    <ScrollArea className='h-[min(52vh,28rem)] pr-3'>
      <div className='flex flex-col'>
        {announcements.map((announcement, index) => (
          <div key={getAnnouncementRenderKey(announcement)}>
            <div className='py-3'>
              <AnnouncementCardContent
                announcement={announcement}
                loading={false}
                t={t}
                emptyTitle='No system announcements'
              />
            </div>
            {index < announcements.length - 1 ? <Separator /> : null}
          </div>
        ))}
      </div>
    </ScrollArea>
  )
}

export function NotificationPopover({
  open,
  onOpenChange,
  unreadCount,
  activeTab,
  onTabChange,
  notice,
  announcements,
  noticeLoading,
  announcementsLoading,
  className,
}: NotificationPopoverProps) {
  const { t } = useTranslation()
  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger
        render={
          <Button
            variant='ghost'
            size='sm'
            className={cn('relative h-9 px-2', className)}
            aria-label={t('Announcements')}
          />
        }
      >
        <span className='flex items-center gap-1.5 text-xs font-medium'>
          <Bell className='size-4' />
          {t('Announcements')}
        </span>
        {unreadCount > 0 ? (
          <Badge
            variant='destructive'
            className='absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center px-1 text-[10px] font-semibold tabular-nums'
          >
            {unreadCount > 99 ? '99+' : unreadCount}
          </Badge>
        ) : null}
      </PopoverTrigger>

      <PopoverContent
        align='end'
        sideOffset={8}
        className='w-[min(26rem,calc(100vw-1rem))] gap-3 p-3'
      >
        <PopoverHeader className='gap-1 px-1'>
          <PopoverTitle>{t('Announcements')}</PopoverTitle>
          <p className='text-muted-foreground text-xs'>
            {t('Latest platform updates and notices')}
          </p>
        </PopoverHeader>
        <Tabs
          value={activeTab}
          onValueChange={onTabChange as (value: string) => void}
        >
          <TabsList className='grid w-full grid-cols-2'>
            <TabsTrigger value='notice' className='gap-1.5'>
              <Bell className='size-3.5' />
              {t('Announcements')}
            </TabsTrigger>
            <TabsTrigger value='announcements' className='gap-1.5'>
              <Megaphone className='size-3.5' />
              {t('Timeline')}
            </TabsTrigger>
          </TabsList>
          <TabsContent value='notice' className='mt-2'>
            <ScrollArea className='max-h-[min(64vh,36rem)] pr-3'>
              <div className='flex flex-col gap-3'>
                <section className='rounded-md border p-3'>
                  <div className='mb-2 flex items-center gap-2 text-sm font-medium'>
                    <Megaphone className='size-4' />
                    {t('Latest Announcement')}
                  </div>
                  <AnnouncementCardContent
                    announcement={announcements[0]}
                    loading={announcementsLoading}
                    t={t}
                    emptyTitle='No system announcements'
                  />
                </section>
                <section className='rounded-md border p-3'>
                  <div className='mb-2 flex items-center gap-2 text-sm font-medium'>
                    <Bell className='size-4' />
                    {t('System Notice')}
                  </div>
                  <SystemNoticeContent
                    notice={notice}
                    loading={noticeLoading}
                    t={t}
                  />
                </section>
              </div>
            </ScrollArea>
          </TabsContent>
          <TabsContent value='announcements' className='mt-2'>
            <AnnouncementsContent
              announcements={announcements}
              loading={announcementsLoading}
              t={t}
            />
          </TabsContent>
        </Tabs>

        <div className='flex justify-end'>
          <Button size='sm' onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
