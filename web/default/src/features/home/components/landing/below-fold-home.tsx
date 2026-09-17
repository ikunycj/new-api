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
import { lazy, type ReactNode, Suspense } from 'react'

import { AiClientsSection } from './ai-clients-section'
import { CapabilitiesSection } from './capabilities-section'
import { DeferUntilVisible } from './defer-until-visible'
import { FaqSection } from './faq-section'
import { FeaturedModelsSection } from './featured-models-section'
import { HomeCtaSection } from './home-cta-section'

const GatewaySection = lazy(() =>
  import('./gateway-section').then((module) => ({
    default: module.GatewaySection,
  }))
)

const ConsolePreviewSection = lazy(() =>
  import('./console-preview-section').then((module) => ({
    default: module.ConsolePreviewSection,
  }))
)

interface BelowFoldHomeProps {
  catalogAvailable: boolean
  isAuthenticated: boolean
}

function DeferredLandingSection(props: {
  children: ReactNode
  placeholderClassName: string
}) {
  const fallback = (
    <div className={props.placeholderClassName} aria-hidden='true' />
  )

  return (
    <DeferUntilVisible placeholderClassName={props.placeholderClassName}>
      <Suspense fallback={fallback}>{props.children}</Suspense>
    </DeferUntilVisible>
  )
}

export function BelowFoldHome(props: BelowFoldHomeProps) {
  return (
    <>
      <FeaturedModelsSection catalogAvailable={props.catalogAvailable} />
      <AiClientsSection />
      <DeferredLandingSection placeholderClassName='min-h-[132rem] bg-transparent md:min-h-[96rem]'>
        <GatewaySection />
      </DeferredLandingSection>
      <DeferredLandingSection placeholderClassName='min-h-[60rem] bg-transparent'>
        <ConsolePreviewSection />
      </DeferredLandingSection>
      <CapabilitiesSection />
      <FaqSection />
      <HomeCtaSection isAuthenticated={props.isAuthenticated} />
    </>
  )
}
