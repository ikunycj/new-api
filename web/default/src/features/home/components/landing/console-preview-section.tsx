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
import {
  Activity01Icon,
  Alert02Icon,
  CheckmarkCircle02Icon,
  Key01Icon,
  RefreshIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import { SectionHeading } from './section-heading'

const LOG_SAMPLES = [
  {
    time: '10:42:18',
    model: 'gpt-6-astra',
    group: 'ChatGPT Pro',
    tokens: '1,824',
    cost: '$0.012',
    duration: '412 ms',
  },
  {
    time: '10:41:52',
    model: 'claude-sonnet-5',
    group: 'Claude Economy',
    tokens: '2,306',
    cost: '$0.021',
    duration: '638 ms',
  },
  {
    time: '10:40:09',
    model: 'deepseek-flash',
    group: 'DeepSeek Standard',
    tokens: '968',
    cost: '$0.004',
    duration: '295 ms',
  },
  {
    time: '10:38:44',
    model: 'gpt-image-2.5-sunburst',
    group: 'ChatGPT Plus',
    tokens: '1,476',
    cost: '$0.015',
    duration: '526 ms',
  },
]

const GROUP_STATUS_SAMPLES = [
  {
    name: 'ChatGPT Plus',
    availability: '99.98%',
    latency: '412 ms',
    status: 'operational',
  },
  {
    name: 'ChatGPT Pro',
    availability: '99.99%',
    latency: '318 ms',
    status: 'operational',
  },
  {
    name: 'Claude Economy',
    availability: '92.40%',
    latency: '--',
    status: 'failed',
  },
  {
    name: 'Claude Priority',
    availability: '99.95%',
    latency: '465 ms',
    status: 'operational',
  },
] as const

export function ConsolePreviewSection() {
  const { t } = useTranslation()
  const metrics = [
    { label: t("Today's usage"), value: '$24.68', hint: t('Usage') },
    { label: t('Requests'), value: '18,420', hint: t('Last 24 hours') },
    { label: t('Average TPM'), value: '3,280', hint: t('Tokens per minute') },
    { label: t('Available groups'), value: '3 / 4', hint: t('Operational') },
  ]

  return (
    <section className='px-4 py-20 sm:px-6 sm:py-24 lg:py-28'>
      <div className='mx-auto w-full max-w-6xl'>
        <SectionHeading
          eyebrow={t('Console and monitoring')}
          title={t('Clear call records, visible group status')}
          description={t(
            'Review model, group, token, cost, and latency for every call, then check group availability from the same console.'
          )}
        />

        <Card className='gap-0 rounded-lg py-0 shadow-xs'>
          <header className='border-border flex min-h-13 flex-col justify-center gap-1 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5'>
            <div className='flex items-center gap-2 text-sm font-semibold'>
              <span className='bg-primary size-2 rounded-[3px]' />
              {t('Sample workspace')}
            </div>
            <span className='text-muted-foreground text-xs'>
              {t('Last 24 hours · sample data')}
            </span>
          </header>

          <div className='bg-border grid grid-cols-2 gap-px border-b lg:grid-cols-4'>
            {metrics.map((metric) => (
              <div key={metric.label} className='bg-card p-4 sm:p-5'>
                <p className='text-muted-foreground text-xs'>{metric.label}</p>
                <p className='mt-2 font-mono text-xl font-semibold tabular-nums sm:text-2xl'>
                  {metric.value}
                </p>
                <p className='text-muted-foreground mt-1 text-[11px]'>
                  {metric.hint}
                </p>
              </div>
            ))}
          </div>

          <div className='grid lg:grid-cols-[1.18fr_0.82fr]'>
            <section className='border-border min-w-0 p-4 sm:p-5 lg:border-e'>
              <header className='flex items-start justify-between gap-4'>
                <div className='flex items-center gap-2'>
                  <HugeiconsIcon
                    icon={Activity01Icon}
                    className='text-muted-foreground size-4'
                  />
                  <div>
                    <h3 className='text-sm font-semibold'>
                      {t('Recent calls')}
                    </h3>
                    <p className='text-muted-foreground mt-0.5 text-xs'>
                      {t('Model, group, tokens, cost and latency')}
                    </p>
                  </div>
                </div>
                <span className='text-muted-foreground text-xs'>
                  {t('Sample data')}
                </span>
              </header>

              <Table className='mt-3'>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Time / model')}</TableHead>
                    <TableHead>{t('Group')}</TableHead>
                    <TableHead className='hidden sm:table-cell'>
                      {t('Tokens / cost')}
                    </TableHead>
                    <TableHead>{t('Duration')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {LOG_SAMPLES.map((sample) => (
                    <TableRow key={sample.time}>
                      <TableCell>
                        <span className='block font-mono text-xs font-semibold'>
                          {sample.time}
                        </span>
                        <span className='text-muted-foreground block max-w-44 truncate font-mono text-[11px]'>
                          {sample.model}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant='outline' className='max-w-32 truncate'>
                          {sample.group}
                        </Badge>
                      </TableCell>
                      <TableCell className='hidden font-mono text-xs sm:table-cell'>
                        <span className='block'>{sample.tokens}</span>
                        <span className='text-muted-foreground mt-0.5 block'>
                          {sample.cost}
                        </span>
                      </TableCell>
                      <TableCell className='font-mono text-xs'>
                        {sample.duration}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </section>

            <section className='border-border border-t p-4 sm:p-5 lg:border-t-0'>
              <header className='flex items-start justify-between gap-4'>
                <div className='flex items-center gap-2'>
                  <HugeiconsIcon
                    icon={RefreshIcon}
                    className='text-muted-foreground size-4'
                  />
                  <div>
                    <h3 className='text-sm font-semibold'>
                      {t('Group status query')}
                    </h3>
                    <p className='text-muted-foreground mt-0.5 text-xs leading-5'>
                      {t(
                        'Check available groups, latency, and recent test results for an API key.'
                      )}
                    </p>
                  </div>
                </div>
                <Badge
                  variant='secondary'
                  className='bg-success/10 text-success'
                >
                  3 / 4 {t('Operational')}
                </Badge>
              </header>

              <div className='border-border bg-background mt-4 flex items-center gap-2 rounded-lg border p-2'>
                <HugeiconsIcon
                  icon={Key01Icon}
                  className='text-muted-foreground ml-1 size-4 shrink-0'
                  aria-hidden='true'
                />
                <code className='text-muted-foreground min-w-0 flex-1 truncate font-mono text-xs'>
                  sk-all-models-••••••
                </code>
                <span className='bg-primary text-primary-foreground shrink-0 rounded-md px-3 py-1.5 text-xs font-medium'>
                  {t('Check status')}
                </span>
              </div>

              <div className='mt-3 space-y-2'>
                {GROUP_STATUS_SAMPLES.map((group) => {
                  const operational = group.status === 'operational'
                  return (
                    <div
                      key={group.name}
                      className='border-border bg-card flex items-center gap-3 rounded-lg border px-3 py-3'
                    >
                      <span
                        className={cn(
                          'flex size-8 shrink-0 items-center justify-center rounded-md',
                          operational
                            ? 'bg-success/10 text-success'
                            : 'bg-destructive/10 text-destructive'
                        )}
                      >
                        <HugeiconsIcon
                          icon={
                            operational ? CheckmarkCircle02Icon : Alert02Icon
                          }
                          className='size-4'
                          aria-hidden='true'
                        />
                      </span>
                      <span className='min-w-0 flex-1'>
                        <span className='block truncate text-sm font-medium'>
                          {group.name}
                        </span>
                        <span className='text-muted-foreground mt-1 flex flex-wrap gap-x-3 text-[11px]'>
                          <span>
                            {t('Availability')} {group.availability}
                          </span>
                          <span>
                            {t('Request latency')} {group.latency}
                          </span>
                        </span>
                      </span>
                      <Badge
                        variant={operational ? 'secondary' : 'destructive'}
                        className={cn(
                          operational && 'bg-success/10 text-success'
                        )}
                      >
                        {operational ? t('Operational') : t('Failed')}
                      </Badge>
                    </div>
                  )
                })}
              </div>
            </section>
          </div>
        </Card>
      </div>
    </section>
  )
}
