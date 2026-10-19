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
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Trans, useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { CodeBlock } from './components/code-block'
import { DocsShell, type DocsTocItem } from './components/docs-shell'
import { DocsTable } from './components/docs-table'
import { ResponsiveDocsImage } from './components/responsive-docs-image'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const CODEX_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline break-words'
const CODEX_CODE_CLASS = 'bg-muted rounded px-1.5 py-0.5 text-sm break-words'
const IMAGE_BASE = '/static/image/docs/codex'
const AUTH_JSON = `{
  "OPENAI_API_KEY": "sk-your-api-key"
}`

const CODEX_TOC: DocsTocItem[] = [
  {
    id: 'download',
    label: 'ChatGPT download',
    level: 1,
  },
  {
    id: 'windows',
    label: 'Windows version',
    level: 2,
  },
  {
    id: 'windows-standard-download',
    label: 'Standard download',
    level: 3,
  },
  {
    id: 'windows-package-download',
    label: 'Download a Microsoft installation package',
    level: 3,
  },
  {
    id: 'windows-desktop-shortcut',
    label: 'Desktop shortcut',
    level: 3,
  },
  {
    id: 'macos',
    label: 'macOS version',
    level: 2,
  },
  {
    id: 'login',
    label: 'ChatGPT sign-in',
    level: 1,
  },
  {
    id: 'chatgpt-account',
    label: '1. Sign in with an official ChatGPT account',
    level: 2,
  },
  {
    id: 'official-api',
    label: '2. Sign in with the official ChatGPT API',
    level: 2,
  },
  {
    id: 'third-party-login',
    label: '3. Sign in with a third-party API or gateway',
    level: 2,
  },
  {
    id: 'third-party-api',
    label: 'Configure a third-party API in ChatGPT',
    level: 1,
  },
  {
    id: 'cc-switch-import',
    label: '1. One-click import with CC Switch',
    level: 2,
  },
  {
    id: 'install-cc-switch',
    label: '1.1 Install CC Switch',
    level: 3,
  },
  {
    id: 'create-api-key',
    label: '1.2 Create an API key',
    level: 3,
  },
  {
    id: 'configure-import',
    label: '1.3 Configure before importing',
    level: 3,
  },
  {
    id: 'confirm-import',
    label: '1.4 Confirm the configuration in CC Switch',
    level: 3,
  },
  {
    id: 'restart-client',
    label: '1.5 Restart the client',
    level: 3,
  },
  {
    id: 'cc-switch-manual',
    label: 'Manual import in CC Switch',
    level: 2,
  },
  {
    id: 'add-provider',
    label: '2.1 Add a provider',
    level: 3,
  },
  {
    id: 'custom-provider',
    label: '2.2 Custom configuration',
    level: 3,
  },
  {
    id: 'manual-config',
    label: '3. Edit auth.json and config.toml',
    level: 2,
  },
  {
    id: 'edit-auth-json',
    label: '3.1 Edit auth.json',
    level: 3,
  },
  {
    id: 'edit-config-toml',
    label: '3.2 Edit config.toml',
    level: 3,
  },
  {
    id: 'verify-config',
    label: '3.3 Restart and verify',
    level: 3,
  },
]

const CODEX_SCREENSHOTS = {
  step01: {
    largeSrc: `${IMAGE_BASE}/codex-step-01-1520.76aec47a3542.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-01-760.f1b941db29de.webp`,
    width: 2239,
    height: 1280,
    captionKey: 'The Codex website',
  },
  step02: {
    largeSrc: `${IMAGE_BASE}/codex-step-02-1520.448a7e2c0dae.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-02-760.46088f2608aa.webp`,
    width: 2230,
    height: 1280,
    captionKey: 'The ChatGPT desktop website',
  },
  step03: {
    largeSrc: `${IMAGE_BASE}/codex-step-03-1520.c3cc73933895.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-03-760.9a1d795363b1.webp`,
    width: 1374,
    height: 983,
    captionKey: 'Run the Windows installer',
  },
  step04: {
    largeSrc: `${IMAGE_BASE}/codex-step-04-1520.bca04fa27f61.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-04-760.8e2a3969796f.webp`,
    width: 768,
    height: 444,
    captionKey: 'Wait for the installation to finish',
  },
  step05: {
    largeSrc: `${IMAGE_BASE}/codex-step-05-1520.a00596234f9c.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-05-760.9733f2824b25.webp`,
    width: 2432,
    height: 1280,
    captionKey: 'Microsoft Store package download',
  },
  step06: {
    largeSrc: `${IMAGE_BASE}/codex-step-06-1520.96101482ae98.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-06-760.96101482ae98.webp`,
    width: 626,
    height: 326,
    captionKey: 'Open shell:AppsFolder',
  },
  step07: {
    largeSrc: `${IMAGE_BASE}/codex-step-07-1520.f2a43e5048db.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-07-760.823fc2d63d97.webp`,
    width: 1385,
    height: 1227,
    captionKey: 'Create a desktop shortcut',
  },
  step08: {
    largeSrc: `${IMAGE_BASE}/codex-step-08-1520.cd4f3583f98e.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-08-760.7dab8b37079f.webp`,
    width: 1324,
    height: 796,
    captionKey: 'Install ChatGPT on macOS',
  },
  step09: {
    largeSrc: `${IMAGE_BASE}/codex-step-09-1520.ee21d5f06959.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-09-760.47040c4bacd0.webp`,
    width: 1635,
    height: 1139,
    captionKey: 'ChatGPT sign-in screen',
  },
  step10: {
    largeSrc: `${IMAGE_BASE}/codex-step-10-1520.c3af45bacd6f.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-10-760.d66fd473e7d2.webp`,
    width: 1635,
    height: 1139,
    captionKey: 'API key sign-in screen',
  },
  step11: {
    largeSrc: `${IMAGE_BASE}/codex-step-11-1520.e2c09208f339.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-11-760.39ef9c330bf0.webp`,
    width: 1520,
    height: 1120,
    captionKey: 'CC Switch release downloads',
  },
  step12: {
    largeSrc: `${IMAGE_BASE}/codex-step-12-1520.398671127324.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-12-760.6c6f19b1c305.webp`,
    width: 2944,
    height: 1942,
    captionKey: 'Create and import an API key',
  },
  step13: {
    largeSrc: `${IMAGE_BASE}/codex-step-13-1520.4fdb3368c4e1.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-13-760.8c10f94bb3c9.webp`,
    width: 2932,
    height: 1936,
    captionKey: 'Configure the Codex import',
  },
  step14: {
    largeSrc: `${IMAGE_BASE}/codex-step-14-1520.55261981cb3a.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-14-760.269a2082abbe.webp`,
    width: 2062,
    height: 1446,
    captionKey: 'Confirm the provider in CC Switch',
  },
  step15: {
    largeSrc: `${IMAGE_BASE}/codex-step-15-1520.e69f3ad511b4.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-15-760.08d973bf8658.webp`,
    width: 2080,
    height: 1460,
    captionKey: 'Enable the provider and restart the client',
  },
  step16: {
    largeSrc: `${IMAGE_BASE}/codex-step-16-1520.4fd00ac230fa.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-16-760.eb4d63526f32.webp`,
    width: 2086,
    height: 1472,
    captionKey: 'Add a provider in CC Switch',
  },
  step17: {
    largeSrc: `${IMAGE_BASE}/codex-step-17-1520.271cb69534cd.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-17-760.09dd5d7aae77.webp`,
    width: 2078,
    height: 1462,
    captionKey: 'Choose a custom provider',
  },
  step18: {
    largeSrc: `${IMAGE_BASE}/codex-step-18-1520.7d1d1491d42f.webp`,
    smallSrc: `${IMAGE_BASE}/codex-step-18-760.adc73359849f.webp`,
    width: 2074,
    height: 1454,
    captionKey: 'Enter the API key and request URL',
  },
} as const

function CodexText(props: { textKey: string; links?: Record<string, string> }) {
  const { t } = useTranslation()
  return (
    <Trans
      t={t}
      i18nKey={props.textKey}
      components={{
        code: <code className={CODEX_CODE_CLASS} />,
        ...Object.fromEntries(
          Object.entries(props.links ?? {}).map(([key, href]) => [
            key,
            <a
              key={key}
              href={href}
              target='_blank'
              rel='noopener noreferrer'
              className={CODEX_LINK_CLASS}
            />,
          ])
        ),
      }}
    />
  )
}

function CodexScreenshot(props: { name: keyof typeof CODEX_SCREENSHOTS }) {
  const { t } = useTranslation()
  const screenshot = CODEX_SCREENSHOTS[props.name]
  return (
    <ResponsiveDocsImage
      {...screenshot}
      alt={t(screenshot.captionKey)}
      caption={t(screenshot.captionKey)}
    />
  )
}

export function DocsCodex() {
  const { t } = useTranslation()
  const baseUrl = useDocsBaseUrl()
  const codexConfig = `cli_auth_credentials_store = "file"
model = "gpt-5.6-sol"
model_provider = "OpenAI"

[model_providers.OpenAI]
name = "All Token API"
base_url = "${baseUrl}/v1"
requires_openai_auth = true
wire_api = "responses"`

  return (
    <DocsShell
      pageId='codex'
      title={t('ChatGPT beginner installation guide')}
      description={t(
        'Download, install, sign in, and configure third-party APIs with this illustrated guide.'
      )}
      toc={CODEX_TOC.map((item) => ({
        ...item,
        label: t(item.label).replaceAll(/<\/?code>/g, ''),
      }))}
    >
      <section id='download' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          <CodexText textKey='ChatGPT download' />
        </h2>
        <section id='windows' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='Windows version' />
          </h3>
          <section id='windows-standard-download' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='Standard download' />
            </h4>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={1}
            >
              <li>
                <CodexText
                  textKey='Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.'
                  links={{
                    a0: 'https://openai.com/zh-Hans-CN/codex/',
                    a1: 'https://chatgpt.com/zh-Hans-CN/download/',
                  }}
                />
              </li>
            </ol>
            <Alert className='mt-5'>
              <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
              <AlertTitle>
                <CodexText textKey='Tutorial note' />
              </AlertTitle>
              <AlertDescription className='space-y-2'>
                <p>
                  <CodexText textKey='Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.' />
                </p>
              </AlertDescription>
            </Alert>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Codex website:' />
            </p>
            <CodexScreenshot name='step01' />
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='ChatGPT desktop website' />
            </p>
            <CodexScreenshot name='step02' />
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={2}
            >
              <li>
                <CodexText textKey='Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.' />
              </li>
            </ol>
            <CodexScreenshot name='step03' />
            <Alert className='mt-5'>
              <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
              <AlertTitle>
                <CodexText textKey='Tutorial note' />
              </AlertTitle>
              <AlertDescription className='space-y-2'>
                <p>
                  <CodexText textKey='On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.' />
                </p>
                <p>
                  <CodexText textKey='Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.' />
                </p>
                <p>
                  <CodexText textKey='If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.' />
                </p>
                <p>
                  <CodexText textKey='If the problem persists, use the second method below to install the Codex package directly.' />
                </p>
              </AlertDescription>
            </Alert>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={3}
            >
              <li>
                <CodexText textKey='The download has started. Wait for the progress bar to finish.' />
              </li>
            </ol>
            <CodexScreenshot name='step04' />
          </section>
          <section id='windows-package-download' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='Download a Microsoft installation package' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.' />
            </p>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={1}
            >
              <li>
                <CodexText
                  textKey='Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.'
                  links={{ a0: 'https://store.rg-adguard.net/' }}
                />
              </li>
            </ol>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={2}
            >
              <li>
                <CodexText
                  textKey='Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.'
                  links={{
                    a0: 'http://tlu.dl.delivery.mp.microsoft.com/filestreamingservice/files/19f8e311-2a5f-4e79-9b4d-a169d4738ec1?P1=1792170754&P2=404&P3=2&P4=dq7RKlIMhJNj8CkemOXb3hET3P7vxlUbMQnRM7OjgyT%2bhY%2fEWXE50jdfVNOX3mGNkjTltHG813ZfE%2f3jx92Y3Q%3d%3d',
                  }}
                />
              </li>
            </ol>
            <CodexScreenshot name='step05' />
          </section>
          <section id='windows-desktop-shortcut' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='Desktop shortcut' />
            </h4>
            <Alert className='mt-5'>
              <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
              <AlertTitle>
                <CodexText textKey='Why is there no ChatGPT desktop shortcut on Windows?' />
              </AlertTitle>
              <AlertDescription className='space-y-2'>
                <p>
                  <CodexText textKey='The icon is in Microsoft Store, so you need to search for it when opening the app.' />
                </p>
                <p>
                  <CodexText textKey='Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.' />
                </p>
              </AlertDescription>
            </Alert>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={1}
            >
              <li>
                <CodexText textKey='Press Win+R to open the Run dialog.' />
              </li>
            </ol>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={2}
            >
              <li>
                <CodexText textKey='Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.' />
              </li>
            </ol>
            <CodexScreenshot name='step06' />
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={3}
            >
              <li>
                <CodexText textKey='Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.' />
              </li>
            </ol>
            <CodexScreenshot name='step07' />
          </section>
        </section>
        <section id='macos' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='macOS version' />
          </h3>
          <p className='text-muted-foreground mt-3 leading-7'>
            <CodexText textKey='Downloading the macOS version is simpler.' />
          </p>
          <ol className='mt-4 list-decimal space-y-3 pl-6 leading-7' start={1}>
            <li>
              <CodexText
                textKey='Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.'
                links={{
                  a0: 'https://openai.com/zh-Hans-CN/codex/',
                  a1: 'https://chatgpt.com/zh-Hans-CN/download/',
                }}
              />
            </li>
          </ol>
          <Alert className='mt-5'>
            <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
            <AlertTitle>
              <CodexText textKey='Tutorial note' />
            </AlertTitle>
            <AlertDescription className='space-y-2'>
              <p>
                <CodexText textKey='Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.' />
              </p>
            </AlertDescription>
          </Alert>
          <ol className='mt-4 list-decimal space-y-3 pl-6 leading-7' start={2}>
            <li>
              <CodexText textKey='After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.' />
            </li>
          </ol>
          <CodexScreenshot name='step08' />
        </section>
      </section>
      <section id='login' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          <CodexText textKey='ChatGPT sign-in' />
        </h2>
        <section id='chatgpt-account' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='1. Sign in with an official ChatGPT account' />
          </h3>
          <p className='text-muted-foreground mt-3 leading-7'>
            <CodexText textKey='Click Continue to sign in.' />
          </p>
          <CodexScreenshot name='step09' />
        </section>
        <section id='official-api' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='2. Sign in with the official ChatGPT API' />
          </h3>
          <p className='text-muted-foreground mt-3 leading-7'>
            <CodexText textKey='Click Use another sign-in method below, then enter your official OpenAI API key.' />
          </p>
          <CodexScreenshot name='step09' />
        </section>
        <section id='third-party-login' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='3. Sign in with a third-party API or gateway' />
          </h3>
          <ol className='mt-4 list-decimal space-y-3 pl-6 leading-7' start={1}>
            <li>
              <CodexText textKey='Click Use another sign-in method and enter an arbitrary value.' />
            </li>
          </ol>
          <CodexScreenshot name='step09' />
          <CodexScreenshot name='step10' />
          <ol className='mt-4 list-decimal space-y-3 pl-6 leading-7' start={2}>
            <li>
              <CodexText textKey='Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.' />
            </li>
          </ol>
        </section>
      </section>
      <section id='third-party-api' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>
          <CodexText textKey='Configure a third-party API in ChatGPT' />
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          <CodexText textKey='Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.' />
        </p>
        <section id='cc-switch-import' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='1. One-click import with CC Switch' />
          </h3>
          <p className='text-muted-foreground mt-3 leading-7'>
            <CodexText textKey='CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.' />
          </p>
          <section id='install-cc-switch' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='1.1 Install CC Switch' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText
                textKey='Download and install the app from the <a0>CC Switch releases page</a0>.'
                links={{
                  a0: 'https://github.com/farion1231/cc-switch/releases',
                }}
              />
            </p>
            <CodexScreenshot name='step11' />
          </section>
          <section id='create-api-key' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='1.2 Create an API key' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText
                textKey='Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.'
                links={{
                  a0: 'https://ccswitch.io/zh/docs?section=getting-started&item=quickstart',
                }}
              />
            </p>
            <Alert className='mt-5'>
              <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
              <AlertTitle>
                <CodexText textKey='Tutorial note' />
              </AlertTitle>
              <AlertDescription className='space-y-2'>
                <p>
                  <CodexText textKey='The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.' />
                </p>
              </AlertDescription>
            </Alert>
            <CodexScreenshot name='step12' />
          </section>
          <section id='configure-import' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='1.3 Configure before importing' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.' />
            </p>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Name: the configuration name.' />
            </p>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Primary model: the default model for new windows. You can change it later.' />
            </p>
            <CodexScreenshot name='step13' />
          </section>
          <section id='confirm-import' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='1.4 Confirm the configuration in CC Switch' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.' />
            </p>
            <CodexScreenshot name='step14' />
          </section>
          <section id='restart-client' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='1.5 Restart the client' />
            </h4>
            <CodexScreenshot name='step15' />
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.' />
            </p>
          </section>
        </section>
        <section id='cc-switch-manual' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='Manual import in CC Switch' />
          </h3>
          <section id='add-provider' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='2.1 Add a provider' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Click Add provider.' />
            </p>
            <CodexScreenshot name='step16' />
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Then choose Custom configuration.' />
            </p>
            <CodexScreenshot name='step17' />
          </section>
          <section id='custom-provider' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='2.2 Custom configuration' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Fill in the fields as prompted. Make sure the API key and API request URL are correct.' />
            </p>
            <CodexScreenshot name='step18' />
          </section>
        </section>
        <section id='manual-config' className='mt-8 scroll-mt-28'>
          <h3 className='text-xl font-semibold'>
            <CodexText textKey='3. Edit auth.json and config.toml' />
          </h3>
          <p className='text-muted-foreground mt-3 leading-7'>
            <CodexText textKey='Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.' />
          </p>
          <div className='mt-4'>
            <DocsTable
              minWidth='sm'
              headers={[t('System'), 'auth.json', 'config.toml']}
              rows={[
                {
                  key: 'windows',
                  cells: [
                    'Windows',
                    <code key='auth-path' className='break-all'>
                      {'%USERPROFILE%\\.codex\\auth.json'}
                    </code>,
                    <code key='config-path' className='break-all'>
                      {'%USERPROFILE%\\.codex\\config.toml'}
                    </code>,
                  ],
                },
                {
                  key: 'macos-linux',
                  cells: [
                    'macOS / Linux',
                    <code key='auth-path'>~/.codex/auth.json</code>,
                    <code key='config-path'>~/.codex/config.toml</code>,
                  ],
                },
              ]}
            />
          </div>
          <section id='edit-auth-json' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='3.1 Edit auth.json' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.' />
            </p>
            <div className='mt-4'>
              <CodeBlock code={AUTH_JSON} label='auth.json' />
            </div>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.' />
            </p>
          </section>
          <section id='edit-config-toml' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='3.2 Edit config.toml' />
            </h4>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.' />
            </p>
            <div className='mt-4'>
              <CodeBlock code={codexConfig} label='config.toml' />
            </div>
            <Alert className='mt-5'>
              <HugeiconsIcon icon={InformationCircleIcon} aria-hidden='true' />
              <AlertTitle>
                <CodexText textKey='Conversation history issues' />
              </AlertTitle>
              <AlertDescription className='space-y-2'>
                <p>
                  <CodexText textKey='If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.' />
                </p>
              </AlertDescription>
            </Alert>
          </section>
          <section id='verify-config' className='mt-8 scroll-mt-28'>
            <h4 className='text-lg font-semibold'>
              <CodexText textKey='3.3 Restart and verify' />
            </h4>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={1}
            >
              <li>
                <CodexText textKey='Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.' />
              </li>
            </ol>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={2}
            >
              <li>
                <CodexText textKey='Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.' />
              </li>
            </ol>
            <ol
              className='mt-4 list-decimal space-y-3 pl-6 leading-7'
              start={3}
            >
              <li>
                <CodexText textKey='If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.' />
              </li>
            </ol>
            <div className='mt-4'>
              <CodeBlock code='codex' label='codex' />
            </div>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText textKey='Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.' />
            </p>
            <p className='text-muted-foreground mt-3 leading-7'>
              <CodexText
                textKey='References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>'
                links={{
                  a0: 'https://developers.openai.com/codex/auth',
                  a1: 'https://developers.openai.com/codex/config-reference',
                }}
              />
            </p>
          </section>
        </section>
      </section>
    </DocsShell>
  )
}
