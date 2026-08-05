/**
 * Stable fragment IDs for checklist source chips (e.g. "OSCQR 7" → "src-oscqr-7").
 * Used by the research dialog, full-page route, and chip hrefs.
 */

import helpCatalog from './checklist-help-data.json'
import { COURSE_DESIGN_RESEARCH_HREF } from './checklist-help'

type HelpCatalog = {
  items: Record<string, { title: string; sources: string[] }>
}

const catalog = helpCatalog as HelpCatalog

/** Normalize a source chip label to a URL-safe anchor id. */
export function sourceToAnchorId(source: string): string {
  const slug = source
    .trim()
    .toLowerCase()
    .replace(/§/g, 's')
    .replace(/&/g, 'and')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return slug ? `src-${slug}` : 'src-unknown'
}

/** Full-page research URL, optionally deep-linked to a source chip. */
export function courseDesignResearchHref(source?: string | null): string {
  if (!source?.trim()) return COURSE_DESIGN_RESEARCH_HREF
  return `${COURSE_DESIGN_RESEARCH_HREF}#${sourceToAnchorId(source)}`
}

export type SourceIndexEntry = {
  source: string
  anchorId: string
  items: Array<{ title: string }>
}

/** Build a standards index from the help catalog (every unique chip label). */
export function buildSourceIndex(): SourceIndexEntry[] {
  const bySource = new Map<string, Set<string>>()
  for (const entry of Object.values(catalog.items)) {
    for (const src of entry.sources ?? []) {
      const key = src.trim()
      if (!key) continue
      let titles = bySource.get(key)
      if (!titles) {
        titles = new Set()
        bySource.set(key, titles)
      }
      titles.add(entry.title)
    }
  }

  return [...bySource.entries()]
    .sort(([a], [b]) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' }))
    .map(([source, titles]) => ({
      source,
      anchorId: sourceToAnchorId(source),
      items: [...titles]
        .sort((x, y) => x.localeCompare(y))
        .map((title) => ({ title })),
    }))
}
