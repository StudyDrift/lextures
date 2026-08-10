/**
 * UX.7 FR-5 — icon uniqueness + near-duplicate label detection within a scope.
 */

import type { NavDestination, NavScopeKind } from './types'

export type CollisionFinding = {
  scope: NavScopeKind
  kind: 'duplicate-icon' | 'near-duplicate-label' | 'duplicate-id' | 'duplicate-route'
  a: string
  b: string
  detail: string
}

/** Levenshtein distance (small strings only — nav labels). */
export function levenshtein(a: string, b: string): number {
  const s = a.toLowerCase()
  const t = b.toLowerCase()
  if (s === t) return 0
  if (!s.length) return t.length
  if (!t.length) return s.length
  const rows = s.length + 1
  const cols = t.length + 1
  const d: number[][] = Array.from({ length: rows }, () => Array(cols).fill(0))
  for (let i = 0; i < rows; i++) d[i]![0] = i
  for (let j = 0; j < cols; j++) d[0]![j] = j
  for (let i = 1; i < rows; i++) {
    for (let j = 1; j < cols; j++) {
      const cost = s[i - 1] === t[j - 1] ? 0 : 1
      d[i]![j] = Math.min(
        d[i - 1]![j]! + 1,
        d[i]![j - 1]! + 1,
        d[i - 1]![j - 1]! + cost,
      )
    }
  }
  return d[s.length]![t.length]!
}

/**
 * Normalise labels for comparison: strip punctuation, collapse whitespace,
 * drop leading "my "/"the ".
 */
export function normaliseLabel(label: string): string {
  return label
    .toLowerCase()
    .replace(/['’]/g, '')
    .replace(/[^a-z0-9\s]/g, ' ')
    .replace(/\b(my|the|a|an)\b/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

/** Near-duplicate when normalised equal, or Levenshtein ≤2 for labels ≥5 chars. */
export function labelsNearDuplicate(a: string, b: string): boolean {
  const na = normaliseLabel(a)
  const nb = normaliseLabel(b)
  if (!na || !nb) return false
  if (na === nb) return true
  if (Math.min(na.length, nb.length) < 5) return false
  return levenshtein(na, nb) <= 2
}

export function findCollisions(
  scope: NavScopeKind,
  destinations: NavDestination[],
): CollisionFinding[] {
  const findings: CollisionFinding[] = []
  const byId = new Map<string, NavDestination>()
  const byIcon = new Map<string, NavDestination>()
  const byRoute = new Map<string, NavDestination>()

  for (const d of destinations) {
    if (d.utility) continue

    const prevId = byId.get(d.id)
    if (prevId) {
      findings.push({
        scope,
        kind: 'duplicate-id',
        a: prevId.id,
        b: d.id,
        detail: `duplicate destination id "${d.id}"`,
      })
    } else {
      byId.set(d.id, d)
    }

    const prevIcon = byIcon.get(d.icon)
    if (prevIcon) {
      findings.push({
        scope,
        kind: 'duplicate-icon',
        a: prevIcon.id,
        b: d.id,
        detail: `icon "${d.icon}" used by both "${prevIcon.label}" and "${d.label}"`,
      })
    } else {
      byIcon.set(d.icon, d)
    }

    const routeKey = d.route
    const prevRoute = byRoute.get(routeKey)
    if (prevRoute) {
      findings.push({
        scope,
        kind: 'duplicate-route',
        a: prevRoute.id,
        b: d.id,
        detail: `route "${routeKey}" used by both "${prevRoute.id}" and "${d.id}"`,
      })
    } else {
      byRoute.set(routeKey, d)
    }
  }

  const list = destinations.filter((d) => !d.utility)
  for (let i = 0; i < list.length; i++) {
    for (let j = i + 1; j < list.length; j++) {
      const a = list[i]!
      const b = list[j]!
      if (labelsNearDuplicate(a.label, b.label)) {
        findings.push({
          scope,
          kind: 'near-duplicate-label',
          a: a.id,
          b: b.id,
          detail: `labels "${a.label}" ≈ "${b.label}"`,
        })
      }
    }
  }

  return findings
}
