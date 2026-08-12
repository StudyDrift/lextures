import type { JsonLdNode } from '../document-head'
import { absoluteUrl, founderPersonId, logoImageId, organizationId } from './ids'
import { BRAND, VERIFIED_SAME_AS } from './entity'

export function buildLogoImage(siteOrigin?: string): JsonLdNode {
  return {
    '@type': 'ImageObject',
    '@id': logoImageId(siteOrigin),
    url: absoluteUrl(BRAND.logoPath, siteOrigin),
    contentUrl: absoluteUrl(BRAND.logoPath, siteOrigin),
    caption: 'Lextures logo',
  }
}

export function buildFounderPersonStub(siteOrigin?: string): JsonLdNode {
  return {
    '@type': 'Person',
    '@id': founderPersonId(siteOrigin),
    name: 'Chase Willden',
    jobTitle: 'Founder',
    url: absoluteUrl('/about', siteOrigin),
    worksFor: { '@id': organizationId(siteOrigin) },
  }
}

export function buildOrganization(siteOrigin?: string): JsonLdNode {
  return {
    '@type': 'Organization',
    '@id': organizationId(siteOrigin),
    name: BRAND.name,
    alternateName: [...BRAND.alternateName],
    legalName: BRAND.legalName,
    url: absoluteUrl('/', siteOrigin),
    logo: { '@id': logoImageId(siteOrigin) },
    image: { '@id': logoImageId(siteOrigin) },
    description: BRAND.description,
    foundingDate: BRAND.foundingDate,
    founder: { '@id': founderPersonId(siteOrigin) },
    email: BRAND.email,
    contactPoint: [
      {
        '@type': 'ContactPoint',
        contactType: 'sales',
        email: BRAND.salesEmail,
        areaServed: BRAND.areaServed,
        availableLanguage: [...BRAND.availableLanguage],
      },
      {
        '@type': 'ContactPoint',
        contactType: 'customer support',
        email: BRAND.supportEmail,
        areaServed: BRAND.areaServed,
        availableLanguage: [...BRAND.availableLanguage],
      },
    ],
    knowsAbout: [...BRAND.knowsAbout],
    sameAs: [...VERIFIED_SAME_AS],
    inLanguage: 'en',
  }
}
