import type { JsonLdNode } from '../document-head'
import { absoluteUrl, articleId, organizationId } from './ids'
import { personRef } from './person'

export type ArticleInput = {
  path: string
  headline: string
  description: string
  datePublished: string
  dateModified?: string
  authorSlug: string
  reviewedBySlug?: string
  image?: string
  wordCount?: number
  articleSection?: string
  /** Primary sources referenced in the body (URLs or names). */
  citations?: string[]
  /** Use TechArticle for help-center / procedural docs. */
  tech?: boolean
  siteOrigin?: string
}

export function buildArticle(input: ArticleInput): JsonLdNode {
  const type = input.tech ? 'TechArticle' : 'Article'
  const node: JsonLdNode = {
    '@type': type,
    '@id': articleId(input.path, input.siteOrigin),
    headline: input.headline,
    description: input.description,
    datePublished: input.datePublished,
    dateModified: input.dateModified || input.datePublished,
    author: personRef(input.authorSlug, input.siteOrigin),
    publisher: { '@id': organizationId(input.siteOrigin) },
    mainEntityOfPage: {
      '@type': 'WebPage',
      '@id': absoluteUrl(input.path, input.siteOrigin),
    },
    inLanguage: 'en',
  }

  if (input.image) {
    node.image = input.image.startsWith('http')
      ? input.image
      : absoluteUrl(input.image, input.siteOrigin)
  }
  if (typeof input.wordCount === 'number' && input.wordCount > 0) {
    node.wordCount = input.wordCount
  }
  if (input.articleSection) {
    node.articleSection = input.articleSection
  }
  if (input.reviewedBySlug) {
    node.reviewedBy = personRef(input.reviewedBySlug, input.siteOrigin)
  }
  if (input.citations?.length) {
    node.citation = input.citations.map(c =>
      c.startsWith('http')
        ? { '@type': 'CreativeWork', url: c }
        : { '@type': 'CreativeWork', name: c },
    )
  }

  return node
}
