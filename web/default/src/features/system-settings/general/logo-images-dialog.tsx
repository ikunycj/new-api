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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AxiosError } from 'axios'
import { useRef, useState } from 'react'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Spinner } from '@/components/ui/spinner'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import {
  deleteLogoImage,
  getLogoImages,
  uploadLogoImage,
  type LogoImage,
} from './logo-images-api'

const LOGO_IMAGES_QUERY_KEY = ['system-settings', 'logo-images'] as const

function reportLogoImageError(error: Error): void {
  if (!(error instanceof AxiosError)) toast.error(error.message)
}

export function LogoImagesDialog(props: {
  serverAddress: string
  selectedUrl: string
  onSelect: (url: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<LogoImage | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()
  const { copiedText, copyToClipboard } = useCopyToClipboard({
    successMessage: 'URL 已复制',
    errorMessage: '复制 URL 失败',
  })
  const images = useQuery({
    queryKey: LOGO_IMAGES_QUERY_KEY,
    queryFn: getLogoImages,
    enabled: open,
  })
  const upload = useMutation({
    mutationFn: uploadLogoImage,
    onSuccess: async () => {
      toast.success('图片已上传')
      await queryClient.invalidateQueries({ queryKey: LOGO_IMAGES_QUERY_KEY })
    },
    onError: reportLogoImageError,
  })
  const remove = useMutation({
    mutationFn: deleteLogoImage,
    onSuccess: async () => {
      setDeleteTarget(null)
      toast.success('图片已删除')
      await queryClient.invalidateQueries({ queryKey: LOGO_IMAGES_QUERY_KEY })
    },
    onError: reportLogoImageError,
  })
  const busy = upload.isPending || remove.isPending
  const serverAddress = props.serverAddress.trim().replace(/\/+$/, '')

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (busy) return
        setOpen(nextOpen)
      }}
    >
      <DialogTrigger render={<Button type='button' variant='outline' />}>
        上传图片
      </DialogTrigger>
      <DialogContent className='flex max-h-[85dvh] flex-col sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>徽标图片</DialogTitle>
          <DialogDescription className='overflow-x-auto text-xs whitespace-nowrap'>
            支持 PNG、JPG、WebP 和 GIF，单张最大 5 MiB，最多 10
            张。点“使用”后保存系统信息即可更新徽标。
          </DialogDescription>
        </DialogHeader>
        <div className='flex items-center gap-3'>
          <input
            ref={fileInput}
            type='file'
            className='hidden'
            accept='.png,.jpg,.jpeg,.webp,.gif'
            aria-label='选择徽标图片'
            onChange={(event) => {
              const file = event.target.files?.[0]
              event.target.value = ''
              if (file) upload.mutate(file)
            }}
          />
          <Button
            type='button'
            disabled={busy}
            onClick={() => fileInput.current?.click()}
          >
            {upload.isPending ? <Spinner /> : null}
            {upload.isPending ? '上传中' : '添加'}
          </Button>
          <p className='text-muted-foreground text-xs'>
            图片地址为 /logo/图片名
          </p>
        </div>
        <div className='min-h-0 overflow-y-auto'>
          {images.isPending ? (
            <div
              className='flex items-center justify-center gap-2 py-12'
              role='status'
            >
              <Spinner />
              加载图片中
            </div>
          ) : null}
          {images.isError ? (
            <div className='flex items-center justify-between gap-3 py-4'>
              <p role='alert' className='text-destructive'>
                加载图片失败，请重试。
              </p>
              <Button
                type='button'
                variant='outline'
                disabled={images.isFetching}
                onClick={() => void images.refetch()}
              >
                重试
              </Button>
            </div>
          ) : null}
          {images.isSuccess && images.data.length === 0 ? (
            <Empty className='border'>
              <EmptyHeader>
                <EmptyTitle>暂无上传图片</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : null}
          {images.data?.length ? (
            <ul className='divide-y rounded-md border'>
              {images.data.map((image) => (
                <li
                  key={image.name}
                  className='flex flex-wrap items-center gap-3 p-3'
                >
                  <div className='bg-muted flex size-14 shrink-0 items-center justify-center rounded-md border p-1'>
                    <img
                      src={image.url}
                      alt={image.name}
                      className='max-h-full max-w-full object-contain'
                      loading='lazy'
                    />
                  </div>
                  <div className='min-w-0 flex-1'>
                    <p className='truncate font-medium' title={image.name}>
                      {image.name}
                    </p>
                    <p className='text-muted-foreground text-xs break-all'>
                      {image.url}
                    </p>
                  </div>
                  <div className='flex w-full justify-end gap-2 sm:w-auto'>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        void copyToClipboard(`${serverAddress}${image.url}`)
                      }
                    >
                      {copiedText === `${serverAddress}${image.url}`
                        ? '已复制'
                        : '复制 URL'}
                    </Button>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={busy || props.selectedUrl === image.url}
                      onClick={() => {
                        props.onSelect(image.url)
                        setOpen(false)
                      }}
                    >
                      {props.selectedUrl === image.url ? '已选择' : '使用'}
                    </Button>
                    <Button
                      type='button'
                      variant='destructive'
                      size='sm'
                      disabled={busy}
                      onClick={() => setDeleteTarget(image)}
                    >
                      删除
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            disabled={busy}
            onClick={() => setOpen(false)}
          >
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(nextOpen) => {
          if (!nextOpen && !remove.isPending) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除图片</AlertDialogTitle>
            <AlertDialogDescription className='break-all'>
              确定删除“{deleteTarget?.name}”？引用此图片的 URL 将无法访问。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              type='button'
              variant='destructive'
              disabled={remove.isPending}
              onClick={() => {
                if (deleteTarget) remove.mutate(deleteTarget.name)
              }}
            >
              {remove.isPending ? '删除中' : '删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Dialog>
  )
}
