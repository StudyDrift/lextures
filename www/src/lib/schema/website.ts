import type { JsonLdNode } from '../document-head'
import { absoluteUrl, organizationId, websiteId } from './ids'
import { BRAND } from './entity'

/**
 * WebSite node. SearchAction is omitted until site search exists (FR-8).
 */
export function buildWebsite(siteOrigin?: string): JsonLdNode {
  return {
    '@type': 'WebSite',
    '@id': websiteId(siteOrigin),
    name: BRAND.name,
    url: absoluteUrl('/', siteOrigin),
    description: BRAND.description,
    publisher: { '@id': organizationId(siteOrigin) },
    inLanguage: 'en',
  }
}
