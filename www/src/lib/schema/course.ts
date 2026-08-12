import type { JsonLdNode } from '../document-head'
import { absoluteUrl, courseId, organizationId } from './ids'

const DESC_MAX = 300

export function truncateCourseDescription(text: string, maxLen = DESC_MAX): string {
  const cleaned = String(text || '')
    .replace(/\s+/g, ' ')
    .trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

/**
 * Strip @context from server-built course JSON-LD and ensure @id + provider ref (FR / §9).
 */
export function normalizeServerCourseJsonLd(
  raw: Record<string, unknown> | null | undefined,
  slug: string,
  siteOrigin?: string,
): JsonLdNode | null {
  if (!raw || typeof raw !== 'object') return null
  const { ['@context']: _ctx, ...rest } = raw
  const node: JsonLdNode = { ...rest }
  if (!node['@id']) {
    node['@id'] = courseId(slug, siteOrigin)
  }
  if (!node['@type']) {
    node['@type'] = 'Course'
  }
  // Prefer graph Organization @id over inline duplicate
  node.provider = { '@id': organizationId(siteOrigin) }
  if (typeof node.description === 'string') {
    node.description = truncateCourseDescription(node.description)
  }
  if (!node.url) {
    node.url = absoluteUrl(`/courses/${slug}`, siteOrigin)
  }
  return node
}

export type CourseListItem = {
  slug: string
  title: string
  url?: string
}

/** ItemList for /courses carousel (FR-15) — ≥3 ListItems when available. */
export function buildCourseItemList(
  courses: CourseListItem[],
  siteOrigin?: string,
): JsonLdNode | null {
  if (courses.length < 3) return null
  return {
    '@type': 'ItemList',
    '@id': `${absoluteUrl('/courses', siteOrigin)}#itemlist`,
    name: 'Lextures course marketplace',
    numberOfItems: courses.length,
    itemListElement: courses.map((c, i) => ({
      '@type': 'ListItem',
      position: i + 1,
      name: c.title,
      url: c.url || absoluteUrl(`/courses/${c.slug}`, siteOrigin),
    })),
  }
}

/**
 * Extend a Course node with educationalLevel / teaches when known (FR-15).
 */
export function extendCourseNode(
  node: JsonLdNode,
  extras?: { educationalLevel?: string | null; teaches?: string | null },
): JsonLdNode {
  const out = { ...node }
  if (extras?.educationalLevel) {
    out.educationalLevel = extras.educationalLevel
  }
  if (extras?.teaches) {
    out.teaches = extras.teaches
  }
  return out
}
