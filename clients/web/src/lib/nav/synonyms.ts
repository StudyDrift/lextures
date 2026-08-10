/**
 * Build a synonym index for command-palette fuzzy matching (FR-17, FR-18).
 */

import type { NavDestination, NavScopeKind } from './types'
import { allRegisteredDestinations, resolvePath } from './resolve'

export type NavSynonymHit = {
  destinationId: string
  scope: NavScopeKind
  label: string
  route: string
  /** Resolved path (courseCode substituted when provided). */
  path: string
  section: string
  /** Lowercase haystack: label + section + synonyms. */
  haystack: string
}

export function buildNavSynonymIndex(opts?: {
  courseCode?: string
  locale?: string
}): NavSynonymHit[] {
  const locale = opts?.locale ?? 'en'
  const out: NavSynonymHit[] = []

  for (const { scope, destinations } of allRegisteredDestinations()) {
    for (const d of destinations) {
      if (d.indexInPalette === false) continue
      if (scope === 'course' && !opts?.courseCode && d.route.includes(':courseCode')) {
        // Still index with placeholder path for global search of course-relative names
      }
      const syns =
        (d.synonymsByLocale && d.synonymsByLocale[locale]) || d.synonyms || []
      const haystack = [d.label, d.section, d.sectionV2, ...syns]
        .join(' ')
        .toLowerCase()
      out.push({
        destinationId: d.id,
        scope,
        label: d.label,
        route: d.route,
        path: resolvePath(d.route, opts?.courseCode),
        section: d.sectionV2 || d.section,
        haystack,
      })
    }
  }
  return out
}

/** Match query against synonym index (substring + token). */
export function matchNavSynonyms(
  query: string,
  index: NavSynonymHit[],
  limit = 12,
): NavSynonymHit[] {
  const q = query.trim().toLowerCase()
  if (!q) return []
  const scored: { hit: NavSynonymHit; score: number }[] = []
  for (const hit of index) {
    if (hit.haystack.includes(q)) {
      const exactLabel = hit.label.toLowerCase() === q
      const starts = hit.label.toLowerCase().startsWith(q)
      scored.push({
        hit,
        score: exactLabel ? 100 : starts ? 80 : 50 + (hit.haystack.startsWith(q) ? 10 : 0),
      })
      continue
    }
    // Token match (e.g. "marks" in synonyms)
    const tokens = q.split(/\s+/).filter(Boolean)
    if (tokens.every((t) => hit.haystack.includes(t))) {
      scored.push({ hit, score: 40 })
    }
  }
  scored.sort((a, b) => b.score - a.score)
  return scored.slice(0, limit).map((s) => s.hit)
}

export function destinationHaystack(d: NavDestination, locale = 'en'): string {
  const syns = (d.synonymsByLocale && d.synonymsByLocale[locale]) || d.synonyms || []
  return [d.label, d.section, d.sectionV2, ...syns].join(' ').toLowerCase()
}
