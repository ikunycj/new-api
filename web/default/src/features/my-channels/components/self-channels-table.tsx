import { CheckCircle2, Loader2, XCircle } from 'lucide-react'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { GroupBadge } from '@/components/group-badge'
import { ProviderBadge } from '@/components/provider-badge'
import { StatusBadge, StatusBadgeList } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { TruncatedText } from '@/components/truncated-text'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import { CHANNEL_STATUS_CONFIG } from '../../channels/constants'
import {
  getChannelTypeIcon,
  getChannelTypeLabel,
  parseGroupsList,
  parseModelsList,
} from '../../channels/lib'
import { formatResponseTime } from '../../channels/lib/channel-utils'
import { testSelfChannel } from '../api'
import type { SelfChannel } from '../types'

type SelfChannelsTableProps = {
  channels: SelfChannel[]
}

function SelfChannelMetric(props: { label: string; children: ReactNode }) {
  return (
    <div className='flex min-w-0 items-start justify-between gap-3'>
      <span className='text-muted-foreground min-w-0 shrink-0 text-xs'>
        {props.label}
      </span>
      <div className='min-w-0 flex-1 text-right text-sm'>{props.children}</div>
    </div>
  )
}

function SelfChannelDetailsSheet(props: {
  channel: SelfChannel | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const [testState, setTestState] = useState<
    'idle' | 'testing' | 'success' | 'error'
  >('idle')
  const [testResponseTime, setTestResponseTime] = useState<number>()
  const [testError, setTestError] = useState('')
  const [selectedModel, setSelectedModel] = useState('')
  const channel = props.channel
  const channelId = channel?.id
  const channelTestModel = channel?.test_model
  const channelModels = channel?.models
  const models = parseModelsList(channel?.models ?? '')

  useEffect(() => {
    const availableModels = parseModelsList(channelModels ?? '')
    setSelectedModel(channelTestModel || availableModels[0] || '')
    setTestState('idle')
    setTestResponseTime(undefined)
    setTestError('')
  }, [channelId, channelTestModel, channelModels])
  if (!channel) return null

  const status =
    CHANNEL_STATUS_CONFIG[
      channel.status as keyof typeof CHANNEL_STATUS_CONFIG
    ] || CHANNEL_STATUS_CONFIG[0]
  const typeLabel = t(getChannelTypeLabel(channel.type))
  const groups = parseGroupsList(channel.group)
  let testIcon: ReactNode = null
  if (testState === 'testing') testIcon = <Loader2 className='animate-spin' />
  if (testState === 'success') testIcon = <CheckCircle2 />
  if (testState === 'error') testIcon = <XCircle />
  const testLabel =
    testState === 'testing' ? t('Testing...') : t('Test Channel Connection')

  const handleTest = async () => {
    setTestState('testing')
    setTestResponseTime(undefined)
    setTestError('')
    try {
      const response = await testSelfChannel(channel.id, {
        model: selectedModel || undefined,
      })
      const responseTime =
        response.data?.response_time ??
        (typeof response.time === 'number' ? response.time * 1000 : undefined)
      setTestResponseTime(responseTime)
      if (response.success) {
        setTestState('success')
        return
      }
      setTestError(response.message || t('Failed to test channel'))
      setTestState('error')
    } catch (error: unknown) {
      const responseError = error as {
        response?: { data?: { message?: string } }
      }
      setTestError(
        responseError.response?.data?.message || t('Failed to test channel')
      )
      setTestState('error')
    }
  }

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className='sm:max-w-2xl'>
        <SheetHeader>
          <SheetTitle className='flex items-center gap-3'>
            <ProviderBadge
              iconKey={`${getChannelTypeIcon(channel.type)}.Color`}
              iconSize={22}
              label={typeLabel}
              colorText={false}
              copyable={false}
              showDot={false}
            />
            <span className='truncate'>{channel.name}</span>
          </SheetTitle>
          <SheetDescription>
            {t('Read-only view of channels available to your user group.')}
          </SheetDescription>
        </SheetHeader>
        <div className='min-h-0 flex-1 space-y-4 overflow-y-auto px-4 pb-6'>
          <div className='rounded-lg border p-4'>
            <div className='mb-3 text-base font-semibold'>
              {t('Basic Information')}
            </div>
            <div className='grid gap-3 sm:grid-cols-2'>
              <SelfChannelMetric label={t('ID')}>
                <TableId value={`#${channel.id}`} />
              </SelfChannelMetric>
              <SelfChannelMetric label={t('Status')}>
                <StatusBadge
                  label={t(status.label)}
                  variant={status.variant}
                  copyable={false}
                />
              </SelfChannelMetric>
              <SelfChannelMetric label={t('Type')}>
                <span>{typeLabel}</span>
              </SelfChannelMetric>
              <SelfChannelMetric label={t('Groups')}>
                <div className='flex flex-wrap justify-end gap-1'>
                  {groups.map((group) => (
                    <GroupBadge key={group} group={group} size='sm' />
                  ))}
                </div>
              </SelfChannelMetric>
            </div>
          </div>

          <div className='rounded-lg border p-4'>
            <div className='flex items-center justify-between gap-3'>
              <div className='text-base font-semibold'>{t('API URL')}</div>
              <Button
                type='button'
                size='sm'
                variant='outline'
                disabled={testState === 'testing'}
                onClick={handleTest}
              >
                {testIcon}
                {testLabel}
              </Button>
            </div>
            <TruncatedText
              text={channel.base_url || t('Default endpoint')}
              className='mt-2 max-w-full'
            />
            <div className='mt-3 flex items-center gap-2'>
              <span className='text-muted-foreground shrink-0 text-xs'>
                {t('Model to use for testing')}
              </span>
              <Select
                value={selectedModel}
                onValueChange={(value) => value && setSelectedModel(value)}
              >
                <SelectTrigger size='sm' className='min-w-0 flex-1'>
                  <SelectValue placeholder={t('Model')} />
                </SelectTrigger>
                <SelectContent>
                  {models.map((model) => (
                    <SelectItem key={model} value={model}>
                      <span className='font-mono'>{model}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {testState === 'success' && (
              <p className='text-success mt-2 text-xs'>
                {t('{{target}} test succeeded', { target: channel.name })}
                {testResponseTime != null &&
                  ` · ${formatResponseTime(testResponseTime, t)}`}
              </p>
            )}
            {testState === 'error' && (
              <p className='text-destructive mt-2 text-xs'>{testError}</p>
            )}
          </div>

          <div className='rounded-lg border p-4'>
            <div className='mb-3 text-base font-semibold'>{t('Models')}</div>
            {models.length > 0 ? (
              <div className='flex flex-wrap gap-1.5'>
                {models.map((model) => (
                  <StatusBadge
                    key={model}
                    label={model}
                    autoColor={model}
                    size='sm'
                    className='max-w-full font-mono'
                  />
                ))}
              </div>
            ) : (
              <span className='text-muted-foreground'>-</span>
            )}
          </div>

          {channel.model_mapping && (
            <div className='rounded-lg border p-4'>
              <div className='mb-3 text-base font-semibold'>
                {t('Model Mapping')}
              </div>
              <pre className='bg-muted/50 overflow-x-auto rounded-md p-3 text-xs whitespace-pre-wrap'>
                {channel.model_mapping}
              </pre>
            </div>
          )}

          {channel.remark && (
            <div className='rounded-lg border p-4'>
              <div className='mb-2 text-base font-semibold'>{t('Remark')}</div>
              <p className='text-sm'>{channel.remark}</p>
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function SelfChannelCard(props: { channel: SelfChannel }) {
  const { t } = useTranslation()
  const channel = props.channel
  const status =
    CHANNEL_STATUS_CONFIG[
      channel.status as keyof typeof CHANNEL_STATUS_CONFIG
    ] || CHANNEL_STATUS_CONFIG[0]
  const typeLabel = t(getChannelTypeLabel(channel.type))
  const groups = parseGroupsList(channel.group)
  const models = parseModelsList(channel.models)

  return (
    <div className='flex min-w-0 flex-col gap-3 rounded-xl border bg-(--table-row) p-4'>
      <div className='flex items-center justify-between gap-2'>
        <ProviderBadge
          iconKey={`${getChannelTypeIcon(channel.type)}.Color`}
          iconSize={20}
          label={typeLabel}
          colorText={false}
          copyable={false}
          showDot={false}
        />
        <StatusBadge
          label={t(status.label)}
          variant={status.variant}
          copyable={false}
        />
      </div>

      <div className='min-w-0'>
        <div className='text-muted-foreground text-xs'>
          <TableId value={`#${channel.id}`} />
        </div>
        <div className='truncate text-base font-semibold'>{channel.name}</div>
      </div>

      <div className='grid grid-cols-1 gap-x-5 gap-y-2 sm:grid-cols-2'>
        <SelfChannelMetric label={t('Models')}>
          <StatusBadgeList
            items={models}
            max={3}
            className='justify-end'
            moreLabel={(count) => t('+{{count}} more', { count })}
            renderItem={(model) => (
              <StatusBadge
                label={model}
                autoColor={model}
                size='sm'
                copyable={false}
                className='max-w-28 font-mono'
              />
            )}
          />
        </SelfChannelMetric>
        <SelfChannelMetric label={t('Groups')}>
          <div className='flex flex-wrap justify-end gap-1'>
            {groups.map((group) => (
              <GroupBadge key={group} group={group} size='sm' />
            ))}
          </div>
        </SelfChannelMetric>
        <SelfChannelMetric label={t('Base URL')}>
          <TruncatedText
            text={channel.base_url || t('Default endpoint')}
            className='max-w-full text-xs'
          />
        </SelfChannelMetric>
        <SelfChannelMetric label={t('Status')}>
          <StatusBadge
            label={t(status.label)}
            variant={status.variant}
            copyable={false}
          />
        </SelfChannelMetric>
      </div>

      {channel.remark && (
        <div className='border-border/70 border-t pt-2'>
          <div className='text-muted-foreground mb-1 text-xs'>
            {t('Remark')}
          </div>
          <div className='text-sm'>{channel.remark}</div>
        </div>
      )}
    </div>
  )
}

export function SelfChannelsTable(props: SelfChannelsTableProps) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState('')
  const [selectedChannel, setSelectedChannel] = useState<SelfChannel | null>(
    null
  )

  const filteredChannels = useMemo(() => {
    const keyword = filter.trim().toLowerCase()
    if (!keyword) return props.channels

    return props.channels.filter((channel) =>
      [
        channel.name,
        channel.base_url,
        channel.models,
        channel.group,
        channel.remark,
      ]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword))
    )
  }, [filter, props.channels])

  return (
    <div className='space-y-3'>
      <Input
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
        placeholder={t('Filter by name...')}
        className='max-w-sm'
        aria-label={t('Filter by name...')}
      />
      {filteredChannels.length === 0 ? (
        <div className='text-muted-foreground rounded-lg border p-8 text-center text-sm'>
          {filter ? t('No results found') : t('No available channels')}
        </div>
      ) : (
        <div className='grid grid-cols-1 gap-4 lg:grid-cols-3'>
          {filteredChannels.map((channel) => (
            <button
              key={channel.id}
              type='button'
              className='focus-visible:ring-ring text-left transition hover:-translate-y-0.5 hover:shadow-sm focus-visible:ring-2 focus-visible:outline-none'
              onClick={() => setSelectedChannel(channel)}
              aria-label={`${t('View')} ${channel.name}`}
            >
              <SelfChannelCard channel={channel} />
            </button>
          ))}
        </div>
      )}
      <SelfChannelDetailsSheet
        channel={selectedChannel}
        open={selectedChannel !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedChannel(null)
        }}
      />
    </div>
  )
}
