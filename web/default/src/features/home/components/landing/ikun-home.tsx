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
import { Link } from '@tanstack/react-router'
import {
  ArrowUpRight,
  Check,
  ChevronRight,
  Code2,
  Copy,
  FileCode2,
  KeyRound,
  Layers3,
  Terminal,
  Zap,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import generatedInterfaceGrid from '@/assets/home/generated-interface-grid.webp'
import { Button } from '@/components/ui/button'
import { useHomeCatalog } from '@/features/home/hooks/use-home-catalog'
import { QUOTA_TYPE_VALUES } from '@/features/pricing/constants'
import { formatPrice, formatRequestPrice } from '@/features/pricing/lib/price'
import { useSystemConfig } from '@/hooks/use-system-config'

import { HomeProviderIcon } from './home-provider-icon'

import './ikun-home.css'

interface IkunHomeProps {
  isAuthenticated: boolean
  catalogAvailable: boolean
}

interface DisplayModel {
  name: string
  vendor: string
  icon?: string
  input: string
  output: string
  endpoint: string
}

const FALLBACK_MODELS: DisplayModel[] = [
  {
    name: 'gpt-5.6-sol',
    vendor: 'OpenAI',
    icon: "OpenAI.Avatar.type={'gpt5'}.shape={'square'}",
    input: '$0.53',
    output: '$4.20',
    endpoint: '/v1/responses',
  },
  {
    name: 'claude-fable-5',
    vendor: 'Anthropic',
    icon: 'Claude.Color',
    input: '$1.05',
    output: '$6.48',
    endpoint: '/v1/messages',
  },
  {
    name: 'gemini-3.1-pro',
    vendor: 'Google',
    icon: 'Gemini.Color',
    input: '$0.70',
    output: '$4.20',
    endpoint: '/v1beta/models',
  },
]

function getDisplayModels(
  models: ReturnType<typeof useHomeCatalog>['models']
): DisplayModel[] {
  if (models.length === 0) return FALLBACK_MODELS

  return models.slice(0, 6).map((model) => {
    const requestPriced = model.quota_type === QUOTA_TYPE_VALUES.REQUEST

    return {
      name: model.model_name,
      vendor: model.vendor_name || 'Model provider',
      icon: model.icon || model.vendor_icon,
      input: requestPriced
        ? formatRequestPrice(model)
        : formatPrice(model, 'input', 'M'),
      output: requestPriced
        ? formatRequestPrice(model)
        : formatPrice(model, 'output', 'M'),
      endpoint: model.supported_endpoint_types?.[0] || 'OpenAI compatible',
    }
  })
}

function RequestWorkbench(props: { modelName: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const request = `curl https://api.ikun.love/v1/responses \\\n+  -H "Authorization: Bearer sk-xxxx" \\\n+  -H "Content-Type: application/json" \\\n+  -d '{ "model": "${props.modelName}", "input": "Hello" }'`

  const cleanRequest = request.replaceAll('+  ', '  ')

  const copyRequest = async () => {
    try {
      await navigator.clipboard.writeText(cleanRequest)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1800)
    } catch {
      setCopied(false)
    }
  }

  return (
    <div className='ikun-home-workbench' aria-label={t('Request preview')}>
      <div className='ikun-home-workbench-head'>
        <div className='ikun-home-window-dots' aria-hidden='true'>
          <span className='ikun-home-window-dot is-coral' />
          <span className='ikun-home-window-dot is-lime' />
          <span className='ikun-home-window-dot is-cyan' />
        </div>
        <div className='ikun-home-workbench-meta'>
          <span>{t('Response preview')}</span>
          <span className='ikun-home-status'>
            <span className='ikun-home-status-dot' />
            {t('Operational')}
          </span>
        </div>
      </div>

      <div className='ikun-home-workbench-toolbar'>
        <div className='ikun-home-workbench-tabs' aria-label={t('Protocol')}>
          <span className='is-active'>
            <Terminal className='size-3.5' /> cURL
          </span>
          <span>
            <FileCode2 className='size-3.5' /> JSON
          </span>
          <span>
            <Code2 className='size-3.5' /> SDK
          </span>
        </div>
        <button
          type='button'
          className='ikun-home-copy-button'
          onClick={copyRequest}
          aria-label={t('Copy ready-to-run curl')}
          title={t('Copy ready-to-run curl')}
        >
          {copied ? <Check className='size-4' /> : <Copy className='size-4' />}
        </button>
      </div>

      <div className='ikun-home-workbench-body'>
        <div className='ikun-home-request-heading'>
          <div>
            <span className='ikun-home-request-method'>POST</span>
            <span className='ikun-home-request-path'>/v1/responses</span>
          </div>
          <span className='ikun-home-response-code'>200 OK</span>
        </div>

        <p className='ikun-home-selected-model'>
          <span>{t('Request Model')}</span>
          <strong>{props.modelName}</strong>
        </p>

        <pre className='ikun-home-request-code' aria-label={t('Example')}>
          <code>{cleanRequest}</code>
        </pre>

        <div className='ikun-home-workbench-foot'>
          <div>
            <span>{t('Base URL')}</span>
            <strong>api.ikun.love</strong>
          </div>
          <div>
            <span>{t('Route')}</span>
            <strong>stable</strong>
          </div>
          <div>
            <span>{t('Latency')}</span>
            <strong className='is-lime'>180 ms</strong>
          </div>
        </div>
        <p className='sr-only' aria-live='polite'>
          {copied ? t('Copied!') : ''}
        </p>
      </div>
    </div>
  )
}

function SignalRail(props: {
  models: DisplayModel[]
  catalogAvailable: boolean
  isLoading: boolean
}) {
  const { t } = useTranslation()
  const target = props.catalogAvailable ? '/pricing' : '/docs'

  return (
    <section className='ikun-home-signal-rail' aria-label={t('Live catalog')}>
      <div className='ikun-home-shell ikun-home-signal-inner'>
        <div className='ikun-home-signal-label'>
          <span className='ikun-home-live-mark' />
          <span>{t('Live catalog')}</span>
          <span className='ikun-home-signal-count'>
            {props.isLoading
              ? '--'
              : String(props.models.length).padStart(2, '0')}
          </span>
        </div>
        <div className='ikun-home-signal-list'>
          {props.models.slice(0, 5).map((model) => (
            <Link
              key={model.name}
              to={target}
              className='ikun-home-signal-item group'
            >
              <span className='ikun-home-signal-icon'>
                <HomeProviderIcon
                  icon={model.icon}
                  provider={model.vendor}
                  size={16}
                />
              </span>
              <span className='truncate'>{model.name}</span>
              <ChevronRight className='size-3.5 shrink-0 transition-transform group-hover:translate-x-0.5' />
            </Link>
          ))}
        </div>
        <Link to={target} className='ikun-home-signal-link'>
          {t('Browse and compare')}
          <ArrowUpRight className='size-3.5' />
        </Link>
      </div>
    </section>
  )
}

function ModelDirectory(props: {
  models: DisplayModel[]
  catalogAvailable: boolean
}) {
  const { t } = useTranslation()
  const target = props.catalogAvailable ? '/pricing' : '/docs'

  return (
    <section className='ikun-home-section ikun-home-directory'>
      <div className='ikun-home-shell'>
        <div className='ikun-home-section-head'>
          <div>
            <p className='ikun-home-overline'>{t('Model catalog')}</p>
            <h2 className='ikun-home-section-title'>
              {t('Browse available models and pricing')}
            </h2>
            <p className='ikun-home-section-copy'>
              {t(
                'Compare capabilities, context, endpoints, and pricing before sending a request with one API key.'
              )}
            </p>
          </div>
          <Link to={target} className='ikun-home-section-link'>
            {t('View Pricing')}
            <ArrowUpRight className='size-4' />
          </Link>
        </div>

        <div className='ikun-home-directory-table'>
          <div className='ikun-home-directory-head'>
            <span>{t('Model')}</span>
            <span>{t('Provider')}</span>
            <span>{t('Endpoint')}</span>
            <span>{t('Input / 1M')}</span>
            <span>{t('Output / 1M')}</span>
            <span />
          </div>
          {props.models.slice(0, 5).map((model) => (
            <Link
              key={model.name}
              to={target}
              className='ikun-home-directory-row group'
              aria-label={`${model.name} ${t('Model Square')}`}
            >
              <span className='ikun-home-model-cell'>
                <span className='ikun-home-model-icon'>
                  <HomeProviderIcon
                    icon={model.icon}
                    provider={model.vendor}
                    size={20}
                  />
                </span>
                <span className='min-w-0'>
                  <strong className='block truncate'>{model.name}</strong>
                  <small className='block truncate'>{t('Available now')}</small>
                </span>
              </span>
              <span
                className='ikun-home-directory-value'
                data-label={t('Provider')}
              >
                {model.vendor}
              </span>
              <span
                className='ikun-home-directory-value ikun-home-endpoint'
                data-label={t('Endpoint')}
              >
                {model.endpoint}
              </span>
              <span
                className='ikun-home-directory-value'
                data-label={t('Input')}
              >
                {model.input}
              </span>
              <span
                className='ikun-home-directory-value'
                data-label={t('Output')}
              >
                {model.output}
              </span>
              <ChevronRight className='ikun-home-directory-arrow' />
            </Link>
          ))}
        </div>
      </div>
    </section>
  )
}

function RelaySteps() {
  const { t } = useTranslation()
  const steps = [
    {
      number: '01',
      icon: KeyRound,
      title: t('Create an API key'),
      description: t('Create a key for your app or service'),
    },
    {
      number: '02',
      icon: Layers3,
      title: t('Choose an API protocol'),
      description: t(
        'Choose a model that supports the Responses endpoint and copy its exact model ID.'
      ),
    },
    {
      number: '03',
      icon: Zap,
      title: t('Send a test request'),
      description: t('Create an API key to unlock the real request'),
    },
  ]

  return (
    <section className='ikun-home-section ikun-home-relay'>
      <div className='ikun-home-shell'>
        <div className='ikun-home-section-head ikun-home-section-head-light'>
          <div>
            <p className='ikun-home-overline'>{t('How it works')}</p>
            <h2 className='ikun-home-section-title'>
              {t('From first key to first response.')}
            </h2>
          </div>
          <p className='ikun-home-section-copy'>
            {t(
              'The gateway provides one service address for supported AI models and keeps API keys, model access, billing, and usage records in one console.'
            )}
          </p>
        </div>

        <div className='ikun-home-relay-grid'>
          {steps.map((step) => {
            const Icon = step.icon
            return (
              <div key={step.number} className='ikun-home-relay-step'>
                <div className='ikun-home-relay-step-top'>
                  <span className='ikun-home-relay-number'>{step.number}</span>
                  <span className='ikun-home-relay-icon'>
                    <Icon className='size-4' />
                  </span>
                </div>
                <h3>{step.title}</h3>
                <p>{step.description}</p>
                <Check className='ikun-home-relay-check size-4' />
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}

function ControlPlaneSection() {
  const { t } = useTranslation()
  const capabilities = [
    {
      icon: Code2,
      title: t('OpenAI compatible'),
      copy: t('OpenAI-compatible chat clients and SDKs'),
    },
    {
      icon: Layers3,
      title: t('Protocol adaptation'),
      copy: t('Route, auth, and balance check in one place'),
    },
    {
      icon: Zap,
      title: t('Routing logs'),
      copy: t('Every request leaves a clear record.'),
    },
  ]

  return (
    <section className='ikun-home-section ikun-home-control'>
      <div className='ikun-home-shell ikun-home-control-grid'>
        <div className='ikun-home-control-copy'>
          <p className='ikun-home-overline'>{t('Unified gateway')}</p>
          <h2 className='ikun-home-section-title'>
            {t('Models can change while your product interface stays stable.')}
          </h2>
          <p className='ikun-home-section-copy'>
            {t(
              'One gateway accepts compatible endpoints and routes each request through the current configuration.'
            )}
          </p>
          <div className='ikun-home-capability-list'>
            {capabilities.map((capability) => {
              const Icon = capability.icon
              return (
                <div key={capability.title} className='ikun-home-capability'>
                  <span className='ikun-home-capability-icon'>
                    <Icon className='size-4' />
                  </span>
                  <div>
                    <h3>{capability.title}</h3>
                    <p>{capability.copy}</p>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
        <figure className='ikun-home-control-figure'>
          <div className='ikun-home-control-image-wrap'>
            <img
              src={generatedInterfaceGrid}
              alt={t('Console area')}
              loading='lazy'
              decoding='async'
            />
          </div>
          <figcaption>
            {t('Dashboards, tokens, and usage analytics.')}
          </figcaption>
        </figure>
      </div>
    </section>
  )
}

export function IkunHome(props: IkunHomeProps) {
  const { t } = useTranslation()
  const { systemName } = useSystemConfig()
  const catalog = useHomeCatalog()
  const models = useMemo(
    () => getDisplayModels(catalog.models),
    [catalog.models]
  )
  const primaryPath = props.isAuthenticated ? '/dashboard' : '/sign-up'
  const primaryLabel = props.isAuthenticated
    ? t('Go to Dashboard')
    : t('Get Started')
  const catalogPath = props.catalogAvailable ? '/pricing' : '/docs'
  const displayName = systemName || 'ikun.love'

  return (
    <main className='ikun-home'>
      <section className='ikun-home-hero'>
        <div className='ikun-home-hero-rule' aria-hidden='true' />
        <div className='ikun-home-shell ikun-home-hero-grid'>
          <div className='ikun-home-hero-copy'>
            <div className='ikun-home-brand-lockup'>
              <span className='ikun-home-mascot-wrap'>
                <img src='/ikun-mascot.png' alt='' aria-hidden='true' />
              </span>
              <span>{displayName}</span>
              <span className='ikun-home-brand-divider'>/</span>
              <span>{t('Unified AI gateway')}</span>
            </div>
            <p className='ikun-home-hero-kicker'>
              <span className='ikun-home-kicker-line' />
              {t('Live catalog')}
            </p>
            <h1 className='ikun-home-display'>
              <span>{t('One endpoint.')}</span>
              <span className='is-accent'>{t('Every model.')}</span>
            </h1>
            <p className='ikun-home-hero-copy-text'>
              {t(
                'The gateway provides one service address for supported AI models and keeps API keys, model access, billing, and usage records in one console.'
              )}
            </p>
            <div className='ikun-home-actions'>
              <Button
                size='lg'
                className='ikun-home-primary'
                render={<Link to={primaryPath} />}
              >
                {primaryLabel}
                <ArrowUpRight className='size-4' />
              </Button>
              <Button
                size='lg'
                variant='outline'
                className='ikun-home-outline'
                render={<Link to={catalogPath} />}
              >
                {t('Model Square')}
                <ChevronRight className='size-4' />
              </Button>
            </div>
            <div className='ikun-home-trust-row'>
              <span>
                <Check className='size-3.5' /> {t('OpenAI compatible')}
              </span>
              <span>
                <Check className='size-3.5' /> {t('Live catalog')}
              </span>
              <span>
                <Check className='size-3.5' /> {t('Unified authentication')}
              </span>
            </div>
          </div>

          <div className='ikun-home-hero-panel'>
            <div className='ikun-home-panel-label'>
              <span>{t('Request preview')}</span>
              <span className='ikun-home-panel-index'>01 / 03</span>
            </div>
            <RequestWorkbench modelName={models[0]?.name || 'YOUR_MODEL'} />
            <div className='ikun-home-panel-note'>
              <span className='ikun-home-note-mark' />
              <span>
                {t(
                  'The catalog and availability shown here follow this deployment.'
                )}
              </span>
            </div>
          </div>
        </div>
      </section>

      <SignalRail
        models={models}
        catalogAvailable={props.catalogAvailable}
        isLoading={catalog.isLoading}
      />
      <ModelDirectory
        models={models}
        catalogAvailable={props.catalogAvailable}
      />
      <RelaySteps />
      <ControlPlaneSection />

      <section className='ikun-home-cta'>
        <div className='ikun-home-shell ikun-home-cta-inner'>
          <div>
            <p className='ikun-home-overline'>{t('Ready when you are')}</p>
            <h2>{t('Give your next model request a clearer path.')}</h2>
            <p>
              {t(
                'Create an API key and connect to the models available on this deployment.'
              )}
            </p>
          </div>
          <Button
            size='lg'
            className='ikun-home-cta-button'
            render={<Link to={primaryPath} />}
          >
            {primaryLabel}
            <ArrowUpRight className='size-4' />
          </Button>
        </div>
      </section>
    </main>
  )
}
