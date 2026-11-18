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
import { Check, Copy, Download } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import type { ChannelIQTestAnalysis } from '../../types'

export type ChannelIQTestPreviewProps = {
  open: boolean
  channelName: string
  model: string
  result: Omit<ChannelIQTestResult, 'model' | 'response'> | null
  source?: string
  onOpenChange: (open: boolean) => void
}

export type ChannelIQTestResult = {
  status: 'testing' | 'success' | 'error'
  model: string
  response?: string
  responseBytes?: number
  finishReason?: string
  error?: string
  responseTime?: number
  completedAt?: number
  engine?: string
  analysis?: ChannelIQTestAnalysis
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MiB`
}

function buildSandboxedSrcDoc(source: string): string {
  const cspContent =
    "default-src 'none'; base-uri 'none'; form-action 'none'; object-src 'none'; img-src data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:; media-src data: blob:; connect-src 'none'; frame-src 'none'; child-src 'none';"
  if (typeof DOMParser !== 'undefined') {
    const parsed = new DOMParser().parseFromString(source, 'text/html')
    const meta = parsed.createElement('meta')
    meta.httpEquiv = 'Content-Security-Policy'
    meta.content = cspContent
    parsed.head.prepend(meta)
    return `<!doctype html>${parsed.documentElement.outerHTML}`
  }
  return `<!doctype html><html><head><meta http-equiv="Content-Security-Policy" content="${cspContent}"></head><body>${source}</body></html>`
}

export function ChannelIQTestPreview(props: ChannelIQTestPreviewProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('preview')
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const analysis = props.result?.analysis
  const isFingerprint =
    props.result?.engine === 'fingerprint' || Boolean(analysis)
  const source = props.source ?? ''
  const sourceBytes =
    props.result?.responseBytes ?? new TextEncoder().encode(source).byteLength
  const srcDoc = useMemo(
    () => (activeTab === 'preview' ? buildSandboxedSrcDoc(source) : ''),
    [activeTab, source]
  )

  const handleDownload = () => {
    if (!source) return
    const url = URL.createObjectURL(new Blob([source], { type: 'text/html' }))
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'channel-iq-test.html'
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 0)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='flex max-h-[min(86vh,720px)] w-[min(94vw,1200px)] max-w-none flex-col gap-3 overflow-hidden p-4 sm:p-5'>
        <DialogHeader className='pr-9'>
          <DialogTitle>{t('IQ test result')}</DialogTitle>
          <DialogDescription>
            {props.channelName} · {props.model}
          </DialogDescription>
        </DialogHeader>

        {isFingerprint ? (
          <FingerprintResult analysis={analysis} error={props.result?.error} />
        ) : (
          <Tabs
            value={activeTab}
            onValueChange={(value) => setActiveTab(value ?? 'preview')}
            className='min-h-0'
          >
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <TabsList aria-label={t('IQ test result view')}>
                <TabsTrigger value='preview'>{t('Preview')}</TabsTrigger>
                <TabsTrigger value='source'>{t('Source')}</TabsTrigger>
              </TabsList>
              <div className='flex flex-wrap items-center gap-2'>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => void copyToClipboard(source)}
                  disabled={!source}
                  aria-label={t('Copy source')}
                  title={t('Copy source')}
                >
                  {copiedText === source ? (
                    <Check data-icon='inline-start' />
                  ) : (
                    <Copy data-icon='inline-start' />
                  )}
                  {t('Copy source')}
                </Button>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={handleDownload}
                  disabled={!source}
                  aria-label={t('Download')}
                  title={t('Download')}
                >
                  <Download data-icon='inline-start' />
                  {t('Download')}
                </Button>
                <span
                  className='text-muted-foreground text-xs'
                  title={`${t('Size:')} ${formatBytes(sourceBytes)}`}
                >
                  {t('Size:')} {formatBytes(sourceBytes)}
                </span>
              </div>
            </div>
            <TabsContent
              value='preview'
              className='mt-3 flex h-[min(56vh,520px)] min-h-[280px] items-center justify-center'
            >
              <div className='flex h-full w-full items-center justify-center overflow-auto rounded-md border bg-white p-2 text-neutral-950'>
                {source && activeTab === 'preview' ? (
                  <iframe
                    className='aspect-video max-h-full min-h-0 w-full border-0'
                    sandbox='allow-scripts'
                    srcDoc={srcDoc}
                    title={t('Preview')}
                  />
                ) : (
                  <p className='text-muted-foreground p-4 text-sm'>
                    {props.result?.error ?? t('No result available')}
                  </p>
                )}
              </div>
            </TabsContent>
            <TabsContent
              value='source'
              className='mt-3 h-[min(56vh,520px)] min-h-[280px]'
            >
              {activeTab === 'source' && (
                <pre className='bg-muted/30 h-full overflow-auto rounded-md border p-3 font-mono text-xs break-words whitespace-pre-wrap'>
                  {source}
                </pre>
              )}
            </TabsContent>
          </Tabs>
        )}
      </DialogContent>
    </Dialog>
  )
}

function FingerprintResult({
  analysis,
  error,
}: {
  analysis?: ChannelIQTestAnalysis
  error?: string
}) {
  const { t } = useTranslation()
  const probability = analysis?.probability
  const prediction = [
    analysis?.prediction
      ? analysis.prediction_name || analysis.prediction
      : t('Unavailable'),
    analysis?.family_prediction_name,
  ]
    .filter(Boolean)
    .join(' · ')

  if (!analysis) {
    return (
      <div className='text-muted-foreground rounded-md border p-4 text-sm'>
        {error ?? t('No result available')}
      </div>
    )
  }

  return (
    <div className='bg-muted/20 space-y-4 rounded-lg border p-4 sm:p-5'>
      <div className='grid gap-3 sm:grid-cols-2'>
        <ResultItem label={t('Prediction')} value={prediction} />
        <ResultItem
          label={t('Decision')}
          value={getDecisionLabel(analysis.decision, t)}
        />
        {probability !== null && probability !== undefined && (
          <ResultItem
            label={t('Reference-set probability')}
            value={`${(probability * 100).toFixed(1)}%`}
          />
        )}
      </div>
      {analysis.evidence?.label && (
        <p className='text-muted-foreground text-sm'>
          {getEvidenceLabel(analysis.evidence.label, t)}
        </p>
      )}
    </div>
  )
}

function ResultItem({ label, value }: { label: string; value: string }) {
  return (
    <div className='min-w-0 space-y-1'>
      <p className='text-muted-foreground text-xs'>{label}</p>
      <p className='text-sm font-medium break-words'>{value}</p>
    </div>
  )
}

function getEvidenceLabel(label: string, t: (key: string) => string): string {
  switch (label) {
    case 'Calibrated ranking':
      return t('Calibrated ranking')
    case 'Uncalibrated ranking':
      return t('Uncalibrated ranking')
    case 'Partial sample ranking':
      return t('Partial sample ranking')
    case 'Not enough valid samples':
      return t('Not enough valid samples')
    case 'Not like a language model reply':
      return t('Not a model reply')
    default:
      return t('Fingerprint analysis complete')
  }
}

function getDecisionLabel(
  decision: string,
  t: (key: string) => string
): string {
  switch (decision) {
    case 'scored':
      return t('Scored')
    case 'partial':
      return t('Partial')
    case 'unscorable':
      return t('Not scored')
    case 'not_model':
      return t('Not a model reply')
    default:
      return decision
  }
}
