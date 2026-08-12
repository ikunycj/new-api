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
import { Footer } from '@/components/layout/components/footer'

import { useHomeCatalog } from '../../hooks/use-home-catalog'
import { AiClientsSection } from './ai-clients-section'
import { CapabilitiesSection } from './capabilities-section'
import { ConsolePreviewSection } from './console-preview-section'
import { DeferUntilVisible } from './defer-until-visible'
import { FaqSection } from './faq-section'
import { FeaturedModelsSection } from './featured-models-section'
import { GatewaySection } from './gateway-section'
import { HomeCtaSection } from './home-cta-section'
import { PricingPreviewSection } from './pricing-preview-section'

interface BelowFoldHomeProps {
  catalogAvailable: boolean
  isAuthenticated: boolean
}

function CatalogSections() {
  const catalog = useHomeCatalog()

  return (
    <>
      <ConsolePreviewSection models={catalog.models} />
      <GatewaySection />
      <PricingPreviewSection
        models={catalog.models}
        isLoading={catalog.isLoading}
      />
    </>
  )
}

export function BelowFoldHome(props: BelowFoldHomeProps) {
  return (
    <>
      <AiClientsSection />
      <FeaturedModelsSection catalogAvailable={props.catalogAvailable} />
      <CapabilitiesSection />
      <DeferUntilVisible>
        <CatalogSections />
      </DeferUntilVisible>
      <FaqSection />
      <HomeCtaSection isAuthenticated={props.isAuthenticated} />
      <Footer />
    </>
  )
}
