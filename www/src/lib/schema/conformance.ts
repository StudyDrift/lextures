import type { JsonLdNode } from '../document-head'
import { absoluteUrl, organizationId, vpatCreativeWorkId, webpageId } from './ids'

/** WCAG 2.2 AA — used only where the page states this target (FR-16). */
export const WCAG_22_AA = 'https://www.w3.org/TR/WCAG22/'

/**
 * Accessibility + VPAT pages: WebPage + CreativeWork for the VPAT document.
 * No unearned certification claims (FR-16).
 */
export function buildAccessibilityGraph(opts?: {
  path?: string
  includeVpat?: boolean
  siteOrigin?: string
}): JsonLdNode[] {
  const path = opts?.path || '/accessibility'
  const nodes: JsonLdNode[] = [
    {
      '@type': 'WebPage',
      '@id': webpageId(path, opts?.siteOrigin),
      name: 'Accessibility — Lextures',
      url: absoluteUrl(path, opts?.siteOrigin),
      isPartOf: { '@id': `${(opts?.siteOrigin || '').replace(/\/$/, '') || 'https://lextures.com'}/#website` },
      about: { '@id': organizationId(opts?.siteOrigin) },
      accessibilityAPI: ['ARIA'],
      accessibilityFeature: [
        'alternativeText',
        'readingOrder',
        'structuralNavigation',
        'tableOfContents',
      ],
      // Page states WCAG 2.1 AA target; VPAT covers Section 508 / EN 301 549.
      // We encode the published target without claiming full conformance audit seal.
      inLanguage: 'en',
    },
  ]

  if (opts?.includeVpat !== false) {
    nodes.push({
      '@type': 'CreativeWork',
      '@id': vpatCreativeWorkId(opts?.siteOrigin),
      name: 'Lextures VPAT 2.5 International Edition',
      url: absoluteUrl('/accessibility/vpat', opts?.siteOrigin),
      encodingFormat: 'text/html',
      about: { '@id': organizationId(opts?.siteOrigin) },
      // Documented accessibility features from the public VPAT page — not a certification.
      accessibilityFeature: ['alternativeText', 'readingOrder', 'structuralNavigation'],
      inLanguage: 'en',
    })
  }

  return nodes
}

/**
 * Security page WebPage. hasCredential only when real certifications exist.
 * Today we emit no unearned SOC2/ISO claims.
 */
export function buildSecurityWebPage(siteOrigin?: string): JsonLdNode {
  return {
    '@type': 'WebPage',
    '@id': webpageId('/security', siteOrigin),
    name: 'Security — Lextures',
    url: absoluteUrl('/security', siteOrigin),
    about: { '@id': organizationId(siteOrigin) },
    inLanguage: 'en',
  }
}

export function buildDigitalDocument(opts: {
  path: string
  name: string
  dateModified: string
  siteOrigin?: string
}): JsonLdNode {
  return {
    '@type': 'DigitalDocument',
    '@id': `${absoluteUrl(opts.path, opts.siteOrigin)}#document`,
    name: opts.name,
    url: absoluteUrl(opts.path, opts.siteOrigin),
    dateModified: opts.dateModified,
    publisher: { '@id': organizationId(opts.siteOrigin) },
    inLanguage: 'en',
  }
}
