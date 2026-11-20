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
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'

import { getChannelIQTestHistory } from '../../api'
import { channelsQueryKeys } from '../../lib/channel-actions'
import type { Channel, ChannelIQTestRun } from '../../types'
import {
  ChannelIQRunResult,
  ChannelIQTestPreview,
} from './channel-iq-test-preview'

const sourceLabels = {
  manual: '手动测试',
  automatic: '自动探测',
  snapshot: '历史快照',
}

export function ChannelIQTestHistoryDialog({
  channel,
  model,
  onClose,
}: {
  channel: Channel
  model: string
  onClose: () => void
}) {
  const [page, setPage] = useState(1)
  const [preview, setPreview] = useState<ChannelIQTestRun | null>(null)
  const history = useQuery({
    queryKey: [
      ...channelsQueryKeys.iqTests(channel.id),
      'history',
      model,
      page,
    ],
    queryFn: ({ signal }) =>
      getChannelIQTestHistory(channel.id, model, page, signal),
    staleTime: 0,
  })
  const data = history.data
  const pages = data ? Math.ceil(data.total / data.page_size) : 0

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
      title='智商测试历史'
      description={`${channel.name} · ${model}`}
      descriptionClassName='break-all'
      contentClassName='sm:max-w-xl'
      footer={
        (pages > 1 || page > 1) && (
          <nav
            aria-label='智商测试历史分页'
            className='flex w-full items-center gap-2'
          >
            <span className='text-muted-foreground mr-auto text-xs tabular-nums'>
              {data
                ? `共 ${data.total} 条 · 第 ${page} / ${pages} 页`
                : `第 ${page} 页`}
            </span>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setPage(page - 1)}
              disabled={page <= 1}
            >
              上一页
            </Button>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setPage(page + 1)}
              disabled={page >= pages}
            >
              下一页
            </Button>
          </nav>
        )
      }
    >
      {history.isPending && (
        <p role='status' className='text-muted-foreground py-4 text-sm'>
          正在读取历史…
        </p>
      )}
      {history.isError && (
        <div role='alert' className='flex items-center gap-2 py-2'>
          <span className='text-destructive'>读取智商测试历史失败</span>
          <Button
            variant='link'
            size='sm'
            onClick={() => void history.refetch()}
          >
            重试
          </Button>
        </div>
      )}
      {data &&
        (data.items.length === 0 ? (
          <p className='text-muted-foreground py-4 text-sm'>
            此模型暂无智商测试记录。
          </p>
        ) : (
          <ol className='divide-y' aria-label='测试记录'>
            {data.items.map((run) => (
              <li
                key={run.id}
                className='flex flex-wrap items-center justify-between gap-x-4 gap-y-2 py-3 first:pt-1 last:pb-1'
              >
                <div className='space-y-1'>
                  <time
                    className='text-sm tabular-nums'
                    dateTime={new Date(run.completed_at * 1000).toISOString()}
                  >
                    {formatTimestampToDate(run.completed_at)}
                  </time>
                  <p className='text-muted-foreground text-xs'>
                    {run.engine === 'fingerprint' ? '指纹测试' : '鹈鹕测试'} ·{' '}
                    {sourceLabels[run.source]}
                  </p>
                </div>
                <div className='max-w-full min-w-0 sm:max-w-[55%]'>
                  <ChannelIQRunResult run={run} onPreview={setPreview} />
                </div>
              </li>
            ))}
          </ol>
        ))}
      {preview && (
        <ChannelIQTestPreview
          channelName={channel.name}
          run={preview}
          onClose={() => setPreview(null)}
        />
      )}
    </Dialog>
  )
}
