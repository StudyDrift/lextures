/**
 * Focus-anchor registry (CC.8) — addressable controls across course surfaces.
 *
 * ## Two-registry rule
 * - Assignment / quiz **editor controls** live in PS.1 (`settings-registry.ts`).
 * - Every other addressable spot lives here.
 * - Section-level editor anchors (e.g. `assignment.scheduling`, `quiz.outcomes`)
 *   are declared here and open the matching accordion before focusing.
 *
 * ## ID contract
 * - Format: `^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$` (same shape as CC.1 / PS.1).
 * - Entity rows use `kind: 'entity'` + `?focusEntity=` (not a colon in the ID).
 * - Rename: add old → new in `FOCUS_ANCHOR_ALIASES`. Retire: `RETIRED_FOCUS_ANCHOR_IDS`.
 */

import {
  getSettingById,
  resolveSettingId,
  SETTINGS_SECTION_TITLES,
  type SettingsSurface,
} from './settings-registry'
import {
  FOCUS_ANCHORS,
  type FocusAnchor,
} from './focus-anchors-registry'

export type {
  FocusAnchor,
  FocusAnchorKind,
  FocusContainer,
} from './focus-anchors-registry'
export { FOCUS_ANCHORS } from './focus-anchors-registry'

/** Same shape as CC.1 item IDs and PS.1 setting IDs (without the surface lock). */
export const FOCUS_ANCHOR_ID_REGEX =
  /^[a-z][a-z0-9]*(?:\.[a-z0-9-]+){1,3}$/

export const FOCUS_ANCHOR_ID_MAX_LENGTH = 96

/** Legacy / alternate names → canonical registry id. */
export const FOCUS_ANCHOR_ALIASES: Record<string, string> = {
  // Server pre-CC.8 / FR-17 naming drift
  'course.general.hero': 'course.general.hero-image',
  'grading.groups': 'course.grading.groups',
  'course.feed.announcements': 'feed.channel.announcements',
  'feed.channel.announcements': 'feed.channel.announcements',
  // Entity base renames (colon form → base id; entity travels in focusEntity)
  'modules.module': 'modules.module',
  'modules.item': 'modules.item',
  // Invitation resend affordance
  resend: 'enrollments.invitations',
  // FR-17 spellings (hyphen in first segment is invalid under FR-2)
  'office-hours.slots': 'office.hours.slots',
  'standards-coverage.grid': 'standards.coverage.grid',
}

/** Permanently removed; resolve returns null. */
export const RETIRED_FOCUS_ANCHOR_IDS: ReadonlySet<string> = new Set()

const byId: Map<string, FocusAnchor> = new Map()
for (const a of FOCUS_ANCHORS) {
  byId.set(a.id, a)
}

export function isValidFocusAnchorIdFormat(id: string): boolean {
  return id.length <= FOCUS_ANCHOR_ID_MAX_LENGTH && FOCUS_ANCHOR_ID_REGEX.test(id)
}

/**
 * Strip a trailing `:{entityId}` (or legacy `prefix:uuid`) into base + entity.
 * Does not mutate the registry — pure parse helper for server/client drift.
 */
export function parseCompositeAnchor(
  raw: string,
): { baseId: string; entityId?: string } {
  const colon = raw.indexOf(':')
  if (colon <= 0) return { baseId: raw }
  const left = raw.slice(0, colon)
  const right = raw.slice(colon + 1)
  if (!right) return { baseId: raw }

  // Legacy: `module:{uuid}`, `item:{uuid}`, `outcome:{uuid}`, `publish:{uuid}`, …
  const legacyMap: Record<string, string> = {
    module: 'modules.module',
    item: 'modules.item',
    outcome: 'course.outcomes.item',
    attribution: 'modules.item',
    publish: 'modules.item',
  }
  if (legacyMap[left]) {
    return { baseId: legacyMap[left], entityId: right }
  }

  // FR-17 style: `modules.item:{id}`, `syllabus.section:{id}`, `files.item:{id}`
  if (FOCUS_ANCHOR_ID_REGEX.test(left) || byId.has(left) || FOCUS_ANCHOR_ALIASES[left]) {
    return { baseId: left, entityId: right }
  }

  return { baseId: raw }
}

function synthesizeFromSettings(canonicalId: string): FocusAnchor | null {
  const d = getSettingById(canonicalId)
  if (!d) return null
  const surface = d.surface as SettingsSurface
  const sectionTitle =
    SETTINGS_SECTION_TITLES[surface][d.section] ?? d.section
  const route =
    surface === 'assignment'
      ? '/courses/{courseCode}/modules/assignment/{itemId}'
      : '/courses/{courseCode}/modules/quiz/{itemId}'
  return {
    id: canonicalId,
    route,
    labelKey: `focusAnchor.${canonicalId}`,
    label: d.label || sectionTitle,
    kind: 'control',
    container: { type: 'accordion', id: d.section },
    fromSettingsRegistry: true,
  }
}

/**
 * Resolve a possibly-aliased, composite, or PS.1 setting ID to a FocusAnchor.
 * Returns null for unknown / retired IDs.
 */
export function resolveFocusAnchor(rawId: string): FocusAnchor | null {
  if (!rawId || RETIRED_FOCUS_ANCHOR_IDS.has(rawId)) return null

  const { baseId: parsedBase, entityId: _entity } = parseCompositeAnchor(rawId)
  void _entity

  if (RETIRED_FOCUS_ANCHOR_IDS.has(parsedBase)) return null

  const aliased = FOCUS_ANCHOR_ALIASES[parsedBase]
  const candidate = aliased ?? parsedBase

  if (RETIRED_FOCUS_ANCHOR_IDS.has(candidate)) return null

  const fromRegistry = byId.get(candidate)
  if (fromRegistry) return fromRegistry

  // PS.1 control IDs (full three-segment form)
  const settingCanonical = resolveSettingId(candidate)
  if (settingCanonical) {
    return synthesizeFromSettings(settingCanonical)
  }

  // Section-level editor IDs already in FOCUS_ANCHORS above; if someone passes a
  // bare PS.1 section that we do not list, do not invent one.
  return null
}

/** O(1) lookup of a canonical registry row (no PS.1 synthesis). */
export function getFocusAnchorById(id: string): FocusAnchor | undefined {
  return byId.get(id)
}

/**
 * Whether an anchor id is known to either registry (for integrity tests).
 * Accepts aliases, composite entity forms, and full PS.1 setting IDs.
 */
export function isKnownFocusAnchor(rawId: string): boolean {
  return resolveFocusAnchor(rawId) != null
}

/** All canonical registry IDs (non-PS.1). */
export function listFocusAnchorIds(): string[] {
  return FOCUS_ANCHORS.map((a) => a.id)
}
