import type { JsonLdNode } from '../document-head'
import { PRICING_TIERS } from '../institution-pricing'
import { absoluteUrl, organizationId, softwareApplicationId } from './ids'
import { BRAND, HOMESCHOOL_MONTHLY_USD, SOFTWARE_FEATURES } from './entity'

/**
 * Product SoftwareApplication with AggregateOffer from real published pricing (FR-9, FR-14).
 * Homeschool $20/mo and institution per-student tiers come from constants / institution-pricing.ts.
 * Self-host $0 is license fees only (not a free commercial plan claim for paid segments).
 */
export function buildSoftwareApplication(siteOrigin?: string): JsonLdNode {
  const low = Math.min(HOMESCHOOL_MONTHLY_USD, ...PRICING_TIERS.map(t => t.pricePerStudent))
  const high = Math.max(HOMESCHOOL_MONTHLY_USD, ...PRICING_TIERS.map(t => t.pricePerStudent))

  return {
    '@type': 'SoftwareApplication',
    '@id': softwareApplicationId(siteOrigin),
    name: BRAND.name,
    applicationCategory: 'EducationalApplication',
    operatingSystem: 'Web, iOS, Android',
    description: BRAND.description,
    url: absoluteUrl('/', siteOrigin),
    featureList: [...SOFTWARE_FEATURES],
    softwareHelp: absoluteUrl('/docs', siteOrigin),
    offers: {
      '@type': 'AggregateOffer',
      priceCurrency: 'USD',
      lowPrice: String(low),
      highPrice: String(high),
      offerCount: String(2 + PRICING_TIERS.length),
      offers: [
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
          name: `Institution hosted — ${tier.label}`,
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
          url: absoluteUrl('/pricing', siteOrigin),
        })),
      ],
    },
    provider: { '@id': organizationId(siteOrigin) },
    inLanguage: 'en',
  }
}
