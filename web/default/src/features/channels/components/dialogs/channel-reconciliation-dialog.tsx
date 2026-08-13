import { BarChart3, Loader2, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CartesianGrid, Line, LineChart, XAxis } from 'recharts'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { formatTimestampToDate } from '@/lib/format'

import {
  createChannelCostEntry,
  deleteChannelCostEntry,
  getChannelReconciliation,
} from '../../api'
import type { ChannelReconciliationBucket } from '../../types'
import { useChannels } from '../channels-provider'

type Props = { open: boolean; onOpenChange: (open: boolean) => void }
type RangeKey = '24h' | '7d' | '30d'

const chartConfig = {
  user_charge_usd: { color: 'var(--chart-2)' },
  estimated_cost_usd: { color: 'var(--chart-4)' },
  actual_cost_usd: { color: 'var(--chart-1)' },
} satisfies ChartConfig

function rangeFor(key: RangeKey) {
  const end = Math.floor(Date.now() / 1000)
  const secondsByRange: Record<RangeKey, number> = {
    '24h': 86400,
    '7d': 604800,
    '30d': 2592000,
  }
  const seconds = secondsByRange[key]
  return { start_timestamp: end - seconds, end_timestamp: end }
}

function money(value: number) {
  return formatBillingCurrencyFromUSD(value || 0, {
    digitsLarge: 4,
    digitsSmall: 6,
  })
}

function BucketTable({ buckets }: { buckets: ChannelReconciliationBucket[] }) {
  const { t } = useTranslation()
  if (!buckets.length) {
    return (
      <div className='text-muted-foreground py-8 text-center text-sm'>
        {t('No data')}
      </div>
    )
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Name')}</TableHead>
          <TableHead>{t('Requests')}</TableHead>
          <TableHead>{t('Tokens')}</TableHead>
          <TableHead>{t('Estimated cost')}</TableHead>
          <TableHead>{t('Actual cost')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {buckets.map((bucket) => (
          <TableRow key={bucket.name}>
            <TableCell className='max-w-72 truncate font-medium'>
              {bucket.name}
            </TableCell>
            <TableCell>{bucket.requests}</TableCell>
            <TableCell>
              {(
                bucket.prompt_tokens + bucket.completion_tokens
              ).toLocaleString()}
            </TableCell>
            <TableCell>{money(bucket.estimated_cost_usd)}</TableCell>
            <TableCell>{money(bucket.actual_cost_usd)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function ChannelReconciliationDialog(props: Props) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const [rangeKey, setRangeKey] = useState<RangeKey>('7d')
  const [data, setData] =
    useState<Awaited<ReturnType<typeof getChannelReconciliation>>['data']>()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>()
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    start_at: '',
    end_at: '',
    amount_usd: '',
    note: '',
  })

  const load = useCallback(async () => {
    if (!currentRow) return
    setLoading(true)
    setError(undefined)
    try {
      const response = await getChannelReconciliation(
        currentRow.id,
        rangeFor(rangeKey)
      )
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Request failed'))
      }
      setData(response.data)
    } catch (cause) {
      const message =
        cause instanceof Error ? cause.message : t('Request failed')
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [currentRow, rangeKey, t])

  useEffect(() => {
    if (props.open) void load()
  }, [props.open, load])

  const summary = data?.summary
  const formValid = useMemo(() => {
    const start = Date.parse(form.start_at)
    const end = Date.parse(form.end_at)
    const amount = Number(form.amount_usd)
    return (
      Number.isFinite(start) &&
      Number.isFinite(end) &&
      end > start &&
      Number.isFinite(amount) &&
      amount >= 0
    )
  }, [form])

  const saveCost = async () => {
    if (!currentRow || !formValid) return
    try {
      const response = await createChannelCostEntry(currentRow.id, {
        start_at: Math.floor(Date.parse(form.start_at) / 1000),
        end_at: Math.floor(Date.parse(form.end_at) / 1000),
        amount_usd: Number(form.amount_usd),
        currency: 'USD',
        source: 'manual',
        note: form.note,
      })
      if (!response.success) {
        throw new Error(response.message || t('Request failed'))
      }
      toast.success(t('Cost entry saved'))
      setShowForm(false)
      setForm({ start_at: '', end_at: '', amount_usd: '', note: '' })
      await load()
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : t('Request failed'))
    }
  }

  const removeCost = async (id: number) => {
    if (!currentRow) return
    try {
      const response = await deleteChannelCostEntry(currentRow.id, id)
      if (!response.success) {
        throw new Error(response.message || t('Request failed'))
      }
      await load()
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : t('Request failed'))
    }
  }

  if (!currentRow) return null
  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Channel Reconciliation')}
      description={`${currentRow.name} (#${currentRow.id})`}
      contentClassName='sm:max-w-6xl'
      bodyClassName='space-y-5'
    >
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <ToggleGroup
          value={[rangeKey]}
          onValueChange={(values) => {
            const value = values[0] as RangeKey | undefined
            if (value) setRangeKey(value)
          }}
          variant='outline'
          size='sm'
        >
          <ToggleGroupItem value='24h'>{t('24 hours')}</ToggleGroupItem>
          <ToggleGroupItem value='7d'>{t('7 days')}</ToggleGroupItem>
          <ToggleGroupItem value='30d'>{t('30 days')}</ToggleGroupItem>
        </ToggleGroup>
        <div className='flex gap-2'>
          <Button
            variant='outline'
            size='sm'
            onClick={() => void load()}
            disabled={loading}
          >
            <RefreshCw className={loading ? 'size-4 animate-spin' : 'size-4'} />
            {t('Refresh')}
          </Button>
          <Button size='sm' onClick={() => setShowForm((value) => !value)}>
            <Plus className='size-4' />
            {t('Record cost')}
          </Button>
        </div>
      </div>
      {showForm && (
        <div className='bg-muted/30 grid gap-3 rounded-lg border p-4 sm:grid-cols-2 lg:grid-cols-4'>
          <div>
            <Label>{t('Start')}</Label>
            <Input
              type='datetime-local'
              value={form.start_at}
              onChange={(event) =>
                setForm({ ...form, start_at: event.target.value })
              }
            />
          </div>
          <div>
            <Label>{t('End')}</Label>
            <Input
              type='datetime-local'
              value={form.end_at}
              onChange={(event) =>
                setForm({ ...form, end_at: event.target.value })
              }
            />
          </div>
          <div>
            <Label>{t('Actual cost (USD)')}</Label>
            <Input
              type='number'
              min='0'
              step='0.000001'
              value={form.amount_usd}
              onChange={(event) =>
                setForm({ ...form, amount_usd: event.target.value })
              }
            />
          </div>
          <div>
            <Label>{t('Note')}</Label>
            <Input
              value={form.note}
              onChange={(event) =>
                setForm({ ...form, note: event.target.value })
              }
            />
          </div>
          <div className='flex justify-end gap-2 sm:col-span-2 lg:col-span-4'>
            <Button variant='ghost' onClick={() => setShowForm(false)}>
              {t('Cancel')}
            </Button>
            <Button onClick={() => void saveCost()} disabled={!formValid}>
              {t('Save')}
            </Button>
          </div>
        </div>
      )}
      {error && (
        <Alert variant='destructive'>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {loading && !data ? (
        <div className='text-muted-foreground flex items-center justify-center py-16'>
          <Loader2 className='mr-2 size-4 animate-spin' />
          {t('Loading')}
        </div>
      ) : (
        data && (
          <>
            <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
              {[
                [t('Requests'), summary?.requests],
                [
                  t('Tokens'),
                  (summary?.prompt_tokens || 0) +
                    (summary?.completion_tokens || 0),
                ],
                [t('User charge'), money(summary?.user_charge_usd || 0)],
                [t('Estimated cost'), money(summary?.estimated_cost_usd || 0)],
                [t('Actual cost'), money(summary?.actual_cost_usd || 0)],
                [t('Gross margin'), money(summary?.gross_margin_usd || 0)],
                [
                  t('Estimate variance'),
                  money(summary?.estimate_variance_usd || 0),
                ],
                [
                  t('Average latency'),
                  `${Math.round(summary?.average_latency_ms || 0)} ms`,
                ],
              ].map(([label, value]) => (
                <div
                  className='border-border/70 bg-muted/20 rounded-lg border px-3 py-2'
                  key={String(label)}
                >
                  <div className='text-muted-foreground text-xs'>{label}</div>
                  <div className='mt-1 text-lg font-semibold tabular-nums'>
                    {value}
                  </div>
                </div>
              ))}
            </div>
            <div className='rounded-lg border p-3'>
              <div className='mb-3 text-sm font-medium'>
                {t('Daily cost trend')}
              </div>
              {data.daily.length ? (
                <ChartContainer
                  config={chartConfig}
                  className='aspect-auto h-64 w-full'
                >
                  <LineChart data={data.daily} margin={{ left: 8, right: 8 }}>
                    <CartesianGrid vertical={false} />
                    <XAxis
                      dataKey='name'
                      tickLine={false}
                      axisLine={false}
                      tickMargin={8}
                      minTickGap={24}
                    />
                    <ChartTooltip
                      content={
                        <ChartTooltipContent
                          formatter={(value) => money(Number(value))}
                        />
                      }
                    />
                    <Line
                      dataKey='user_charge_usd'
                      name={t('User charge')}
                      type='monotone'
                      stroke='var(--color-user_charge_usd)'
                      strokeWidth={2}
                      dot={false}
                    />
                    <Line
                      dataKey='estimated_cost_usd'
                      name={t('Estimated cost')}
                      type='monotone'
                      stroke='var(--color-estimated_cost_usd)'
                      strokeWidth={2}
                      dot={false}
                    />
                    <Line
                      dataKey='actual_cost_usd'
                      name={t('Actual cost')}
                      type='monotone'
                      stroke='var(--color-actual_cost_usd)'
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ChartContainer>
              ) : (
                <div className='text-muted-foreground py-12 text-center text-sm'>
                  {t('No data')}
                </div>
              )}
            </div>
            <div className='flex items-center gap-2 text-sm font-medium'>
              <BarChart3 className='size-4' />
              {t('Usage breakdown')}
            </div>
            <Tabs defaultValue='models'>
              <TabsList>
                <TabsTrigger value='models'>{t('Models')}</TabsTrigger>
                <TabsTrigger value='inbound'>
                  {t('Inbound endpoints')}
                </TabsTrigger>
                <TabsTrigger value='upstream'>
                  {t('Upstream endpoints')}
                </TabsTrigger>
                <TabsTrigger value='daily'>{t('Daily')}</TabsTrigger>
              </TabsList>
              <TabsContent value='models'>
                <BucketTable buckets={data.models} />
              </TabsContent>
              <TabsContent value='inbound'>
                <BucketTable buckets={data.inbound_endpoints} />
              </TabsContent>
              <TabsContent value='upstream'>
                <BucketTable buckets={data.upstream_endpoints} />
              </TabsContent>
              <TabsContent value='daily'>
                <BucketTable buckets={data.daily} />
              </TabsContent>
            </Tabs>
            <div>
              <div className='mb-2 text-sm font-medium'>
                {t('Cost entries')}
              </div>
              {data.cost_entries.length ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Period')}</TableHead>
                      <TableHead>{t('Amount')}</TableHead>
                      <TableHead>{t('Source')}</TableHead>
                      <TableHead>{t('Note')}</TableHead>
                      <TableHead />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.cost_entries.map((entry) => (
                      <TableRow key={entry.id}>
                        <TableCell>
                          {formatTimestampToDate(entry.start_at)} ~{' '}
                          {formatTimestampToDate(entry.end_at)}
                        </TableCell>
                        <TableCell>{money(entry.amount_usd)}</TableCell>
                        <TableCell>{entry.source}</TableCell>
                        <TableCell className='max-w-72 truncate'>
                          {entry.note || '-'}
                        </TableCell>
                        <TableCell className='text-right'>
                          <Button
                            variant='ghost'
                            size='icon-sm'
                            aria-label={t('Delete')}
                            onClick={() => void removeCost(entry.id)}
                          >
                            <Trash2 className='text-destructive size-4' />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className='text-muted-foreground rounded-lg border py-8 text-center text-sm'>
                  {t('No cost entries')}
                </div>
              )}
            </div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.'
              )}
            </div>
          </>
        )
      )}
    </Dialog>
  )
}
