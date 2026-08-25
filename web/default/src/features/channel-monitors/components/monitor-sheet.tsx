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
import { zodResolver } from '@hookform/resolvers/zod'
import { Activity01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { FieldGroup } from '@/components/ui/field'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
} from '@/components/ui/input-group'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'

import {
  createChannelMonitor,
  getPricingGroupChannelCount,
  runChannelMonitor,
  updateChannelMonitor,
} from '../api'
import {
  applyMonitorAvailabilityBoost,
  formatMonitorAvailability,
} from '../lib/format'
import {
  channelMonitorFormDefaults,
  channelMonitorFormSchema,
  type ChannelMonitorFormInput,
  type ChannelMonitorFormValues,
} from '../lib/schema'
import type {
  ChannelMonitor,
  ChannelMonitorCreatePayload,
  ChannelMonitorSettingsPayload,
  ChannelMonitorRunResponse,
} from '../types'
import { MonitorHistoryBars, MonitorStatusBadge } from './monitor-status'

type ChannelMonitorSheetProps = {
  open: boolean
  monitor: ChannelMonitor | null
  pricingGroupName: string
  onOpenChange: (open: boolean) => void
  embedded?: boolean
  disabled?: boolean
}

type ChannelMonitorFormCardProps = {
  monitor: ChannelMonitor | null
  pricingGroupName: string
  disabled?: boolean
}

export function ChannelMonitorFormCard(props: ChannelMonitorFormCardProps) {
  return (
    <ChannelMonitorSheet
      open
      monitor={props.monitor}
      pricingGroupName={props.pricingGroupName}
      onOpenChange={() => undefined}
      embedded
      disabled={props.disabled}
    />
  )
}

type SaveMutationInput = {
  values: ChannelMonitorFormValues
  runAfterSave: boolean
}

type SaveMutationResult = {
  monitor: ChannelMonitor
  test?: ChannelMonitorRunResponse
  runAfterSave: boolean
}

function buildFormDefaults(
  monitor: ChannelMonitor | null
): ChannelMonitorFormInput {
  if (!monitor) {
    return { ...channelMonitorFormDefaults }
  }
  return {
    test_model: monitor.test_model,
    interval_seconds: monitor.interval_seconds,
    timeout_seconds: monitor.timeout_seconds,
    retry_count: monitor.retry_count,
    enabled: monitor.enabled,
    visible: monitor.visible,
    availability_boost_percent: monitor.availability_boost_percent,
  }
}

export function ChannelMonitorSheet(props: ChannelMonitorSheetProps) {
  const queryClient = useQueryClient()
  const pricingGroupName = props.pricingGroupName.trim()
  const defaultRetryCountQuery = useQuery({
    queryKey: ['pricing-group-channel-count', pricingGroupName],
    queryFn: () => getPricingGroupChannelCount(pricingGroupName),
    enabled:
      props.open &&
      !props.disabled &&
      props.monitor === null &&
      pricingGroupName !== '',
  })
  const form = useForm<
    ChannelMonitorFormInput,
    unknown,
    ChannelMonitorFormValues
  >({
    resolver: zodResolver(channelMonitorFormSchema),
    defaultValues: buildFormDefaults(props.monitor),
  })
  const formContextKey = `${props.monitor?.id ?? 'new'}:${pricingGroupName}`
  const previousFormContextKey = useRef<string | null>(null)

  useEffect(() => {
    if (!props.open) {
      previousFormContextKey.current = null
      return
    }
    if (previousFormContextKey.current === formContextKey) return

    previousFormContextKey.current = formContextKey
    form.reset(buildFormDefaults(props.monitor))
  }, [form, formContextKey, props.monitor, props.open])

  useEffect(() => {
    if (
      !props.open ||
      props.monitor !== null ||
      defaultRetryCountQuery.data === undefined ||
      form.getFieldState('retry_count').isDirty
    ) {
      return
    }
    form.setValue('retry_count', defaultRetryCountQuery.data)
  }, [defaultRetryCountQuery.data, form, props.monitor, props.open])

  const watchedBoostPercent = useWatch({
    control: form.control,
    name: 'availability_boost_percent',
  })
  const parsedBoostPercent = Number(watchedBoostPercent)
  const boostPercent = Number.isFinite(parsedBoostPercent)
    ? parsedBoostPercent
    : 0

  const saveMutation = useMutation<
    SaveMutationResult,
    Error,
    SaveMutationInput
  >({
    mutationFn: async (input) => {
      const payload: ChannelMonitorSettingsPayload = {
        test_model: input.values.test_model.trim(),
        interval_seconds: input.values.interval_seconds,
        timeout_seconds: input.values.timeout_seconds,
        retry_count: input.values.retry_count,
        enabled: input.values.enabled,
        visible: input.values.visible,
        availability_boost_percent: input.values.availability_boost_percent,
      }
      const saved = props.monitor
        ? await updateChannelMonitor(props.monitor.id, payload)
        : await createChannelMonitor({
            ...payload,
            pricing_group: props.pricingGroupName.trim(),
          } satisfies ChannelMonitorCreatePayload)
      if (!input.runAfterSave) {
        return { monitor: saved, runAfterSave: false }
      }
      const test = await runChannelMonitor(saved.id)
      return { monitor: test.monitor, test, runAfterSave: true }
    },
    onSuccess: async (result) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['channel-monitors'] }),
        queryClient.invalidateQueries({ queryKey: ['group-status'] }),
      ])
      if (result.runAfterSave && result.test && !result.test.result.success) {
        toast.error('监控配置已保存，但可用性测试失败')
        return
      }
      toast.success(
        result.runAfterSave
          ? '监控配置已保存，可用性测试成功'
          : '监控配置已保存'
      )
      if (!props.embedded) props.onOpenChange(false)
    },
    onError: (error) => {
      toast.error(error.message || '操作失败')
    },
  })

  const submit = (values: ChannelMonitorFormValues, runAfterSave: boolean) => {
    if (!props.monitor && !props.pricingGroupName.trim()) {
      toast.error('请先保存定价分组，再配置监控')
      return
    }
    saveMutation.mutate({ values, runAfterSave })
  }

  const formContent = (
    <Form {...form}>
      <form
        className='flex min-h-0 flex-1 flex-col'
        onSubmit={form.handleSubmit((values) => submit(values, false))}
      >
        <fieldset
          disabled={props.disabled || saveMutation.isPending}
          className='contents'
        >
          <div className='flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-5 py-4'>
            {props.monitor && (
              <div className='bg-muted/40 flex flex-col gap-3 rounded-lg border p-3'>
                <div className='flex items-center justify-between gap-3'>
                  <div className='min-w-0'>
                    <p className='truncate text-sm font-medium'>最近测试</p>
                    <p className='text-muted-foreground mt-0.5 text-xs'>
                      {props.monitor.latest_latency_ms == null
                        ? '暂无测试结果'
                        : `${props.monitor.latest_latency_ms} ms`}
                    </p>
                  </div>
                  <MonitorStatusBadge status={props.monitor.status} />
                </div>
                <MonitorHistoryBars
                  results={props.monitor.recent_results}
                  compact
                />
              </div>
            )}

            <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='test_model'
                render={({ field }) => (
                  <FormItem className='sm:col-span-2'>
                    <FormLabel>测试模型</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder='gpt-4.1-mini' />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='interval_seconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>测试间隔</FormLabel>
                    <FormControl>
                      <InputGroup>
                        <InputGroupInput
                          type='number'
                          min={1}
                          max={86400}
                          step={1}
                          value={String(field.value ?? '')}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                        <InputGroupAddon align='inline-end'>
                          <InputGroupText>秒</InputGroupText>
                        </InputGroupAddon>
                      </InputGroup>
                    </FormControl>
                    <FormDescription>可设置 1 至 86400 秒</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='timeout_seconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>请求超时</FormLabel>
                    <FormControl>
                      <InputGroup>
                        <InputGroupInput
                          type='number'
                          min={1}
                          max={120}
                          step={1}
                          value={String(field.value ?? '')}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                        <InputGroupAddon align='inline-end'>
                          <InputGroupText>秒</InputGroupText>
                        </InputGroupAddon>
                      </InputGroup>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='retry_count'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>重试次数</FormLabel>
                    <FormControl>
                      <InputGroup>
                        <InputGroupInput
                          type='number'
                          min={1}
                          max={10000}
                          step={1}
                          value={String(field.value ?? '')}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                        <InputGroupAddon align='inline-end'>
                          <InputGroupText>次</InputGroupText>
                        </InputGroupAddon>
                      </InputGroup>
                    </FormControl>
                    <FormDescription>
                      {defaultRetryCountQuery.isFetching
                        ? '正在读取分组渠道数量'
                        : '默认按分组内渠道数量设置'}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='availability_boost_percent'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>可用率加成</FormLabel>
                    <FormControl>
                      <InputGroup>
                        <InputGroupInput
                          type='number'
                          min={0}
                          max={100}
                          step={0.01}
                          value={String(field.value ?? '')}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                        <InputGroupAddon align='inline-end'>
                          <InputGroupText>%</InputGroupText>
                        </InputGroupAddon>
                      </InputGroup>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {props.monitor && (
                <div className='bg-muted/40 rounded-lg border p-3 sm:col-span-2'>
                  <p className='mb-2 text-sm font-medium'>可用率加成预览</p>
                  <div className='grid gap-2 text-sm'>
                    <AvailabilityPreviewRow
                      period='7 天'
                      raw={props.monitor.raw_availability_7d}
                      boosted={applyMonitorAvailabilityBoost(
                        props.monitor.raw_availability_7d,
                        boostPercent
                      )}
                    />
                    <AvailabilityPreviewRow
                      period='30 天'
                      raw={props.monitor.raw_availability_30d}
                      boosted={applyMonitorAvailabilityBoost(
                        props.monitor.raw_availability_30d,
                        boostPercent
                      )}
                    />
                  </div>
                </div>
              )}
            </FieldGroup>

            <Separator />

            <FieldGroup className='gap-1'>
              <FormField
                control={form.control}
                name='enabled'
                render={({ field }) => (
                  <FormItem className='flex items-center justify-between gap-4 rounded-lg px-1 py-2'>
                    <div>
                      <FormLabel>启用定时测试</FormLabel>
                      <FormDescription>
                        按配置的间隔自动测试当前定价分组
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='visible'
                render={({ field }) => (
                  <FormItem className='flex items-center justify-between gap-4 rounded-lg px-1 py-2'>
                    <div>
                      <FormLabel>向登录用户展示</FormLabel>
                      <FormDescription>
                        在分组状态页面展示此定价分组
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            </FieldGroup>
          </div>

          <SheetFooter className='flex-row justify-end border-t px-5 py-3'>
            <Button
              type='button'
              variant='outline'
              onClick={() => props.onOpenChange(false)}
              className={props.embedded ? 'hidden' : undefined}
            >
              取消
            </Button>
            <Button type='submit'>
              {saveMutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : (
                <HugeiconsIcon icon={Tick02Icon} data-icon='inline-start' />
              )}
              保存
            </Button>
            <Button
              type='button'
              onClick={() =>
                void form.handleSubmit((values) => submit(values, true))()
              }
            >
              {saveMutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : (
                <HugeiconsIcon icon={Activity01Icon} data-icon='inline-start' />
              )}
              保存并测试
            </Button>
          </SheetFooter>
        </fieldset>
      </form>
    </Form>
  )

  if (props.embedded) {
    return (
      <Card className='gap-0 overflow-hidden shadow-none'>
        <CardHeader className='border-b px-4 py-3'>
          <CardTitle className='text-sm'>分组监控</CardTitle>
          <p className='text-muted-foreground text-xs'>
            系统将使用“{props.pricingGroupName}”分组内的渠道凭据测试可用性
          </p>
        </CardHeader>
        <CardContent className='p-0'>{formContent}</CardContent>
      </Card>
    )
  }

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className='gap-0 sm:max-w-xl'>
        <SheetHeader className='border-b px-5 py-4'>
          <SheetTitle>
            {props.monitor ? '编辑分组监控' : '配置分组监控'}
          </SheetTitle>
          <SheetDescription>
            系统将使用“{props.pricingGroupName}”分组内的渠道凭据测试可用性
          </SheetDescription>
        </SheetHeader>
        {formContent}
      </SheetContent>
    </Sheet>
  )
}

type AvailabilityPreviewRowProps = {
  period: string
  raw: number | null
  boosted: number | null
}

function AvailabilityPreviewRow(props: AvailabilityPreviewRowProps) {
  return (
    <div className='grid grid-cols-[5rem_1fr_auto_1fr] items-center gap-2'>
      <span className='text-muted-foreground'>{props.period}</span>
      <span className='text-end tabular-nums'>
        {formatMonitorAvailability(props.raw)}
      </span>
      <span className='text-muted-foreground' aria-hidden='true'>
        →
      </span>
      <strong className='text-end tabular-nums'>
        {formatMonitorAvailability(props.boosted)}
      </strong>
    </div>
  )
}
