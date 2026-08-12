import type { JsonLdNode } from '../document-head'
import type { Author } from '../authors'
import { absoluteUrl, authorPersonId, organizationId } from './ids'

/**
 * Full Person node for an active author (FR-19).
 * Retired authors must not emit a Person node (AC-10).
 */
export function buildPerson(author: Author, siteOrigin?: string): JsonLdNode | null {
  if (author.status === 'retired') return null

  const node: JsonLdNode = {
    '@type': 'Person',
    '@id': authorPersonId(author.slug, siteOrigin),
    name: author.name,
    jobTitle: author.jobTitle,
    description: author.bio,
    url: absoluteUrl(`/authors/${author.slug}`, siteOrigin),
    worksFor: { '@id': organizationId(siteOrigin) },
    knowsAbout: [...author.knowsAbout],
    inLanguage: 'en',
  }

  if (author.image) {
    node.image = author.image.startsWith('http')
      ? author.image
      : absoluteUrl(author.image, siteOrigin)
  }

  if (author.sameAs.length) {
    node.sameAs = [...author.sameAs]
  }

  if (author.alumniOf?.length) {
    node.alumniOf = author.alumniOf.map(name => ({
      '@type': 'Organization',
      name,
    }))
  }

  if (author.credentials?.length) {
    node.hasCredential = author.credentials.map(c => ({
      '@type': 'EducationalOccupationalCredential',
      name: c,
    }))
  }

  return node
}

/** Compact author reference for Article.author etc. */
export function personRef(slug: string, siteOrigin?: string): JsonLdNode {
  return { '@id': authorPersonId(slug, siteOrigin) }
}
