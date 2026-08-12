import type { JsonLdNode } from '../document-head'
import { PRICING_TIERS } from '../institution-pricing'
import { absoluteUrl, organizationId, productPricingId } from './ids'
import { HOMESCHOOL_MONTHLY_USD } from './entity'

/**
 * Product + Offer nodes for /pricing (FR-14).
 * Prices derive only from institution-pricing.ts + HOMESCHOOL_MONTHLY_USD.
 */
export function buildPricingProduct(siteOrigin?: string): JsonLdNode[] {
  const product: JsonLdNode = {
    '@type': 'Product',
    '@id': productPricingId(siteOrigin),
    name: 'Lextures',
    description:
      'Adaptive LMS: self-host free under AGPL-3.0, homeschool hosted plans, and per-student institution pricing.',
    brand: { '@id': organizationId(siteOrigin) },
    url: absoluteUrl('/pricing', siteOrigin),
    offers: [
      {
        '@type': 'Offer',
        name: 'Self-host (AGPL-3.0)',
        price: '0',
        priceCurrency: 'USD',
        description: 'No license fees — you pay for your own infrastructure.',
        availability: 'https://schema.org/InStock',
        url: absoluteUrl('/pricing', siteOrigin),
      },
      {
        '@type': 'Offer',
        name: 'Homeschool hosted',
        price: String(HOMESCHOOL_MONTHLY_USD),
        priceCurrency: 'USD',
        priceSpecification: {
          '@type': 'UnitPriceSpecification',
          price: String(HOMESCHOOL_MONTHLY_USD),
          priceCurrency: 'USD',
          unitText: 'month',
        },
        availability: 'https://schema.org/InStock',
        url: absoluteUrl('/pricing', siteOrigin),
      },
      ...PRICING_TIERS.map(tier => ({
        '@type': 'Offer',
        name: `Institution — ${tier.label}`,
        price: String(tier.pricePerStudent),
        priceCurrency: 'USD',
        priceSpecification: {
          '@type': 'UnitPriceSpecification',
          price: String(tier.pricePerStudent),
          priceCurrency: 'USD',
          unitText: 'student',
        },
        availability: 'https://schema.org/InStock',
        eligibleCustomerType: 'https://schema.org/Business',
        url: absoluteUrl('/pricing/calculator', siteOrigin),
      })),
    ],
  }
  return [product]
}
