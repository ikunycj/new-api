/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { RotateCcw, Save, Settings2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export type LoadTestConfigPanelValues = {
  durationSeconds: string
  requestsPerSecond: string
  concurrency: string
  maxOutputTokens: string
  promptCache: boolean
  streamMode: boolean
  requestTimeoutSeconds: number
}

export type LoadTestConfigPanelLimits = {
  minDurationSeconds: number
  maxDurationSeconds: number
  minRps: number
  maxRps: number
  minConcurrency: number
  maxConcurrency: number
  minOutputTokens: number
  maxOutputTokens: number
}

type Props = {
  values: LoadTestConfigPanelValues
  saved: boolean
  disabled: boolean
  limits: LoadTestConfigPanelLimits
  onDurationChange: (value: string) => void
  onRequestsPerSecondChange: (value: string) => void
  onConcurrencyChange: (value: string) => void
  onMaxOutputTokensChange: (value: string) => void
  onSave: () => void
  onReset: () => void
}

export function LoadTestConfigPanel(props: Props) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <CardTitle className='flex items-center gap-2'>
              <Settings2 className='text-primary size-5' />
              {t('Load Test Configuration')}
            </CardTitle>
            <CardDescription className='mt-1'>
              {t(
                'Configure a bounded duration and request rate for a controlled test.'
              )}
            </CardDescription>
          </div>
          <div className='flex items-center gap-2'>
            {props.saved && <Badge variant='secondary'>{t('Saved')}</Badge>}
            <Button
              disabled={props.disabled}
              onClick={props.onReset}
              size='sm'
              title={t('Reset')}
              variant='outline'
            >
              <RotateCcw className='size-4' />
              {t('Reset')}
            </Button>
            <Button
              disabled={props.disabled}
              onClick={props.onSave}
              size='sm'
              title={t('Save')}
            >
              <Save className='size-4' />
              {t('Save')}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3'>
        <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
          <div className='space-y-1.5'>
            <Label htmlFor='load-test-config-duration'>{t('Duration')}</Label>
            <Input
              disabled={props.disabled}
              id='load-test-config-duration'
              max={props.limits.maxDurationSeconds}
              min={props.limits.minDurationSeconds}
              onChange={(event) => props.onDurationChange(event.target.value)}
              step={1}
              type='number'
              value={props.values.durationSeconds}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Allowed range: {{min}}-{{max}} seconds', {
                min: props.limits.minDurationSeconds,
                max: props.limits.maxDurationSeconds,
              })}
            </p>
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='load-test-config-rps'>
              {t('Requests per second')}
            </Label>
            <Input
              disabled={props.disabled}
              id='load-test-config-rps'
              max={props.limits.maxRps}
              min={props.limits.minRps}
              onChange={(event) =>
                props.onRequestsPerSecondChange(event.target.value)
              }
              step={1}
              type='number'
              value={props.values.requestsPerSecond}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Allowed range: {{min}}-{{max}} RPS', {
                min: props.limits.minRps,
                max: props.limits.maxRps,
              })}
            </p>
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='load-test-config-concurrency'>
              {t('Maximum concurrency')}
            </Label>
            <Input
              disabled={props.disabled}
              id='load-test-config-concurrency'
              max={props.limits.maxConcurrency}
              min={props.limits.minConcurrency}
              onChange={(event) =>
                props.onConcurrencyChange(event.target.value)
              }
              step={1}
              type='number'
              value={props.values.concurrency}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Allowed range: {{min}}-{{max}} concurrent requests', {
                min: props.limits.minConcurrency,
                max: props.limits.maxConcurrency,
              })}
            </p>
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='load-test-config-output-tokens'>
              {t('Max output tokens')}
            </Label>
            <Input
              disabled={props.disabled}
              id='load-test-config-output-tokens'
              max={props.limits.maxOutputTokens}
              min={props.limits.minOutputTokens}
              onChange={(event) =>
                props.onMaxOutputTokensChange(event.target.value)
              }
              step={1}
              type='number'
              value={props.values.maxOutputTokens}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Allowed range: {{min}}-{{max}} tokens', {
                min: props.limits.minOutputTokens,
                max: props.limits.maxOutputTokens,
              })}
            </p>
          </div>
        </div>
        <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
          <span>
            {t('Request timeout')}: {props.values.requestTimeoutSeconds}{' '}
            {t('seconds')}
          </span>
          <span>
            {t('Prompt Cache')}:{' '}
            {props.values.promptCache ? t('Enabled') : t('Disabled')}
          </span>
          <span>
            {t('Stream Mode')}:{' '}
            {props.values.streamMode ? t('Enabled') : t('Disabled')}
          </span>
        </div>
      </CardContent>
    </Card>
  )
}
