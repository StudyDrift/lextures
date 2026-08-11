/**
 * UX.8 — Classification table (FR-2, AC-1).
 *
 * Every settings destination and admin page appears exactly once with a scope.
 * Tiebreak rule: if it changes behaviour for others → configuration (not ops).
 * CI: `npm run settings-ia:check` asserts completeness and uniqueness.
 */

import type { ClassifiedPage } from './types'
import { CLASSIFIED_PAGES } from './classified-pages-data'

export { CLASSIFIED_PAGES }


const byId = new Map(CLASSIFIED_PAGES.map((p) => [p.id, p]))
const byRoute = new Map(CLASSIFIED_PAGES.map((p) => [p.route, p]))
const byLegacy = new Map(
  CLASSIFIED_PAGES.filter((p) => p.legacyRoute).map((p) => [p.legacyRoute!, p]),
)

export function classifiedPageById(id: string): ClassifiedPage | undefined {
  return byId.get(id)
}

export function classifiedPageByRoute(route: string): ClassifiedPage | undefined {
  return byRoute.get(route) ?? byLegacy.get(route)
}

/** Permanent redirects: legacy path → canonical route. */
export function settingsRedirects(): { from: string; to: string }[] {
  return CLASSIFIED_PAGES.filter((p) => p.legacyRoute).map((p) => ({
    from: p.legacyRoute!,
    to: p.route,
  }))
}

export function pagesByScope(scope: ClassifiedPage['scope']): ClassifiedPage[] {
  return CLASSIFIED_PAGES.filter((p) => p.scope === scope)
}

export function configurationPages(): ClassifiedPage[] {
  return CLASSIFIED_PAGES.filter((p) => p.kind === 'configuration')
}

export function operationsPages(): ClassifiedPage[] {
  return CLASSIFIED_PAGES.filter((p) => p.kind === 'operations')
}

/** Assert classification integrity (used by unit tests + CI). */
export function assertClassificationIntegrity(): string[] {
  const errors: string[] = []
  const ids = new Set<string>()
  const routes = new Set<string>()
  for (const p of CLASSIFIED_PAGES) {
    if (ids.has(p.id)) errors.push(`duplicate id: ${p.id}`)
    ids.add(p.id)
    if (routes.has(p.route)) errors.push(`duplicate route: ${p.route}`)
    routes.add(p.route)
    if (p.legacyRoute) {
      if (routes.has(p.legacyRoute) && p.legacyRoute !== p.route) {
        // legacy may not collide with a canonical route as another's home
        const owner = byRoute.get(p.legacyRoute)
        if (owner && owner.id !== p.id) {
          errors.push(`legacy route ${p.legacyRoute} collides with ${owner.id}`)
        }
      }
    }
    if (!['me', 'org', 'platform', 'course', 'operations'].includes(p.scope)) {
      errors.push(`invalid scope on ${p.id}: ${p.scope}`)
    }
    if (p.kind === 'configuration' && p.scope === 'operations') {
      errors.push(`${p.id}: configuration cannot have operations scope`)
    }
    if (p.kind === 'operations' && p.scope !== 'operations') {
      errors.push(`${p.id}: operations pages must use operations scope`)
    }
    // Config under /admin/* is forbidden (AC-3)
    if (p.kind === 'configuration' && p.route.startsWith('/admin/')) {
      errors.push(`${p.id}: configuration still under /admin/* (${p.route})`)
    }
  }
  return errors
}

