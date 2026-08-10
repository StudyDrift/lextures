/**
 * UX.7 — Resolve registry destinations into a renderable nav model.
 * Filters by audience → permission → feature flag → personalisation.
 * Orders by priority (never alphabetical below 20 items).
 */

import {
  courseEnrollmentsReadPermission,
  courseGradebookViewPermission,
  courseItemCreatePermission,
  courseItemsCreatePermission,
} from '../courses-api'
import {
  atRiskFeatureEnabled,
  outcomesReportFeatureEnabled,
  studentProgressFeatureEnabled,
  xapiEmissionFeatureEnabled,
} from '../platform-features'
import {
  PERM_ACCOMMODATIONS_MANAGE,
  PERM_PARENT_DASHBOARD,
  PERM_PARENT_LINKS_MANAGE,
  PERM_RBAC_MANAGE,
  PERM_REPORTS_VIEW,
} from '../rbac-api'
import { NAV_SECTIONS, sectionOrder } from './sections'
import type {
  NavAudience,
  NavDestination,
  NavPreferences,
  NavResolveContext,
  NavScopeKind,
  NavSectionId,
  ResolvedNavItem,
  ResolvedNavModel,
  ResolvedNavSection,
} from './types'
import { COURSE_NAV } from './registry-course'
import { GLOBAL_NAV } from './registry-global'

const DEFAULT_VISIBLE_BUDGET = 6
const PRIMARY_BUDGET = 7

export function emptyPreferences(scope: string): NavPreferences {
  return { scope, pinned: [], hidden: [], collapsed: [] }
}

export function preferenceScopeFor(
  kind: NavScopeKind,
  courseCode?: string,
): string {
  switch (kind) {
    case 'course':
      return courseCode ? `course:${courseCode}` : 'course'
    case 'course-settings':
      return courseCode ? `course-settings:${courseCode}` : 'course-settings'
    case 'global':
      return 'global'
    case 'settings':
      return 'settings'
    case 'admin':
      return 'admin'
    default:
      return 'global'
  }
}

export function resolvePath(route: string, courseCode?: string): string {
  if (!route.includes(':courseCode')) return route
  if (!courseCode) return route
  return route.split(':courseCode').join(encodeURIComponent(courseCode))
}

function flagOn(map: Record<string, unknown>, key: string | undefined): boolean {
  if (!key) return true
  // Composite / helper flags
  if (key === '_walletComposite') {
    return Boolean(
      map.ffTranscripts ||
        map.ffCoCurricularTranscript ||
        map.ffCompetencyBadges ||
        map.ffCompletionCredentials ||
        map.ffCeuTracking ||
        map.ffDiplomas,
    )
  }
  if (key === '_atRisk') return atRiskFeatureEnabled()
  if (key === '_xapi') return xapiEmissionFeatureEnabled()
  if (key === '_outcomesReport') return outcomesReportFeatureEnabled()
  if (key === '_studentProgress') return studentProgressFeatureEnabled()
  return map[key] === true
}

function permissionAllows(dest: NavDestination, ctx: NavResolveContext): boolean {
  const p = dest.permission
  if (!p) return true
  if (ctx.permLoading) {
    // While loading, hide permission-gated destinations (matches prior behaviour).
    return false
  }
  const code = ctx.courseCode ?? ''

  switch (p) {
    case '_courseGradebook':
      return Boolean(code) && ctx.allows(courseGradebookViewPermission(code))
    case '_courseManage':
      return Boolean(code) && ctx.allows(courseItemCreatePermission(code))
    case '_courseQuestionBank':
      return Boolean(code) && ctx.allows(courseItemsCreatePermission(code))
    case '_courseEnrollments':
      // Enrollment visibility is pre-computed into platform/course context by the caller
      // via `platform._canViewEnrollments` when available.
      if (typeof ctx.platform._canViewEnrollments === 'boolean') {
        return ctx.platform._canViewEnrollments === true
      }
      return Boolean(code) && ctx.allows(courseEnrollmentsReadPermission(code))
    case '_courseMyGrades':
      if (typeof ctx.platform._canViewMyGrades === 'boolean') {
        return ctx.platform._canViewMyGrades === true
      }
      return ctx.audience === 'student'
    case '_courseStandards':
      return (
        Boolean(code) &&
        (ctx.allows(courseGradebookViewPermission(code)) ||
          ctx.allows(courseItemCreatePermission(code)))
      )
    case '_assignParents':
      return (
        ctx.allows(PERM_PARENT_LINKS_MANAGE) || ctx.allows(PERM_RBAC_MANAGE)
      )
    case 'parent.dashboard':
    case PERM_PARENT_DASHBOARD:
      return ctx.allows(PERM_PARENT_DASHBOARD)
    case 'reports.view':
    case PERM_REPORTS_VIEW:
      return ctx.allows(PERM_REPORTS_VIEW)
    case 'accommodations.manage':
    case PERM_ACCOMMODATIONS_MANAGE:
      return ctx.allows(PERM_ACCOMMODATIONS_MANAGE)
    default: {
      const resolved = code ? p.replace(':courseCode', code) : p
      return ctx.allows(resolved)
    }
  }
}

function audienceAllows(dest: NavDestination, audience: NavAudience): boolean {
  if (!dest.audience.length || dest.audience.includes('any')) return true
  // Ambiguous audience (permissions still loading / dual role): let permission gates decide.
  if (audience === 'any') return true
  // Explicit student preview must not see instructor-only destinations (FR-8, AC-3, AC-4).
  if (audience === 'student') {
    return dest.audience.includes('student') || dest.audience.includes('any')
  }
  return dest.audience.includes(audience)
}

export function destinationsForScope(scope: NavScopeKind): NavDestination[] {
  switch (scope) {
    case 'global':
      return GLOBAL_NAV
    case 'course':
      return COURSE_NAV
    default:
      return []
  }
}

function isVisible(dest: NavDestination, ctx: NavResolveContext): boolean {
  if (!audienceAllows(dest, ctx.audience)) return false
  if (!permissionAllows(dest, ctx)) return false
  if (dest.featureFlag && !flagOn(ctx.platform, dest.featureFlag)) return false
  if (dest.courseFeature && !flagOn(ctx.courseFeatures, dest.courseFeature)) return false
  return true
}

function itemFrom(
  dest: NavDestination,
  ctx: NavResolveContext,
  extras: Partial<ResolvedNavItem> = {},
): ResolvedNavItem {
  const section = ctx.navigationV2 ? dest.sectionV2 : dest.section
  const isPinned = ctx.prefs.pinned.includes(dest.id)
  const isHidden = ctx.prefs.hidden.includes(dest.id)
  return {
    dest,
    href: resolvePath(dest.route, ctx.courseCode),
    label: dest.label,
    section,
    isPinned,
    isHidden,
    isPrimary: Boolean(ctx.navigationV2 && dest.primaryV2),
    inMore: false,
    ...extras,
  }
}

/**
 * Build the full nav model for a scope. Pure + memo-friendly.
 */
export function resolveNavModel(ctx: NavResolveContext): ResolvedNavModel {
  const destinations = destinationsForScope(ctx.scope)
  const visible: ResolvedNavItem[] = []
  const hidden: ResolvedNavItem[] = []
  const utility: ResolvedNavItem[] = []

  for (const dest of destinations) {
    if (!isVisible(dest, ctx)) continue
    const item = itemFrom(dest, ctx)
    if (dest.utility) {
      utility.push(item)
      continue
    }
    if (item.isHidden && !ctx.showHidden) {
      hidden.push(item)
      continue
    }
    if (item.isHidden && ctx.showHidden) {
      hidden.push(item)
    }
    visible.push(item)
  }

  // Pinned group (ordered by prefs.pinned)
  const byId = new Map(visible.map((i) => [i.dest.id, i]))
  const pinned: ResolvedNavItem[] = []
  for (const id of ctx.prefs.pinned) {
    const item = byId.get(id)
    if (item) pinned.push({ ...item, isPinned: true })
  }

  // Recent is supplied by caller via platform._recentIds (client-local)
  const recent: ResolvedNavItem[] = []
  const recentIds = Array.isArray(ctx.platform._recentIds)
    ? (ctx.platform._recentIds as string[])
    : []
  for (const id of recentIds) {
    if (ctx.prefs.pinned.includes(id)) continue
    const item = byId.get(id)
    if (item) recent.push({ ...item, section: 'recent' })
  }

  const pinnedIds = new Set(pinned.map((i) => i.dest.id))

  // Primary (V2 only)
  let primary: ResolvedNavItem[] = []
  if (ctx.navigationV2) {
    primary = visible
      .filter((i) => i.isPrimary && !pinnedIds.has(i.dest.id))
      .sort((a, b) => a.dest.priority - b.dest.priority)
      .slice(0, PRIMARY_BUDGET)
  }
  const primaryIds = new Set(primary.map((i) => i.dest.id))

  // Group remaining into sections
  const sectionMap = new Map<NavSectionId, ResolvedNavItem[]>()
  for (const item of visible) {
    if (pinnedIds.has(item.dest.id)) continue
    if (primaryIds.has(item.dest.id)) continue
    // Legacy mode: primary section items render ungrouped at top
    const sec = item.section
    if (!ctx.navigationV2 && sec === 'primary') {
      primary.push(item)
      continue
    }
    const list = sectionMap.get(sec) ?? []
    list.push(item)
    sectionMap.set(sec, list)
  }

  // Sort primary by priority
  primary.sort((a, b) => a.dest.priority - b.dest.priority)

  const sections: ResolvedNavSection[] = []
  for (const [secId, items] of sectionMap) {
    items.sort((a, b) => a.dest.priority - b.dest.priority)
    const meta = NAV_SECTIONS[secId]
    const budget =
      ctx.navigationV2 && items.length >= 20
        ? items.length
        : ctx.navigationV2
          ? (meta.visibleBudget ?? DEFAULT_VISIBLE_BUDGET)
          : items.length
    const head = items.slice(0, budget).map((i) => ({ ...i, inMore: false }))
    const more = items.slice(budget).map((i) => ({ ...i, inMore: true }))
    sections.push({
      id: secId,
      label: meta.label,
      items: head,
      moreItems: more,
      collapsed: ctx.prefs.collapsed.includes(secId),
    })
  }

  sections.sort(
    (a, b) => sectionOrder(a.id, ctx.navigationV2) - sectionOrder(b.id, ctx.navigationV2),
  )

  // Drop empty sections (FR-10)
  const nonEmpty = sections.filter((s) => s.items.length + s.moreItems.length > 0)

  return {
    scope: ctx.scope,
    preferenceScope: ctx.preferenceScope,
    utility,
    pinned,
    recent: recent.slice(0, 5),
    primary,
    sections: nonEmpty,
    allVisible: visible,
    hidden,
  }
}

/** All destinations across scopes (for CI collision + palette index). */
export function allRegisteredDestinations(): {
  scope: NavScopeKind
  destinations: NavDestination[]
}[] {
  return [
    { scope: 'global', destinations: GLOBAL_NAV },
    { scope: 'course', destinations: COURSE_NAV },
  ]
}

/** Lookup by id. */
export function findDestination(id: string): NavDestination | undefined {
  for (const { destinations } of allRegisteredDestinations()) {
    const found = destinations.find((d) => d.id === id)
    if (found) return found
  }
  return undefined
}

/** Known destination id set (for server-side drop of unknown prefs). */
export function allDestinationIds(): Set<string> {
  const ids = new Set<string>()
  for (const { destinations } of allRegisteredDestinations()) {
    for (const d of destinations) ids.add(d.id)
  }
  return ids
}
