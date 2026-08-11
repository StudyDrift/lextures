/**
 * UX.8 — Settings & Admin IA types.
 *
 * Scope model:
 * | Scope        | Meaning                                      | Route prefix              |
 * | me           | Affects only this user                       | /settings/*               |
 * | org          | Affects everyone in the org                  | /settings/org/*           |
 * | platform     | Affects the whole deployment (super-admin)   | /settings/platform/*      |
 * | course       | Affects one course                           | /courses/:code/settings/* |
 * | operations   | Non-configuration admin work                 | /admin/*                  |
 */

export type SettingsScope = 'me' | 'org' | 'platform' | 'course' | 'operations'

/** Page-level classification entry (one destination / route). */
export type ClassifiedPage = {
  /** Stable id used in CI and prefs (e.g. settings.account, admin.scheduled-jobs). */
  id: string
  /** Canonical route after migration (absolute path, no query). */
  route: string
  /** Pre-migration route when different; used for permanent redirects. */
  legacyRoute?: string
  scope: SettingsScope
  /** English label (fallback). */
  label: string
  /** i18n key under common. */
  labelKey: string
  description?: string
  /** Permission required to see this page (null = any authenticated). */
  requiredPermission: string | null
  /** Platform feature flag key, if gated. */
  featureFlag?: string
  /** Section within the scope for nav grouping. */
  section: string
  /** Whether this page is configuration (vs operational work). */
  kind: 'configuration' | 'operations'
}

/** Individual setting control indexed for search (FR-5). */
export type SettingsIndexEntry = {
  key: string
  scope: SettingsScope
  /** Page route (absolute). */
  route: string
  /** Deep-link target within the page (`data-focus-anchor` / `?focus=`). */
  anchor: string | null
  label: string
  labelKey: string
  description: string
  descriptionKey: string
  synonyms: string[]
  requiredPermission: string | null
  featureFlag?: string
  /** Parent page id from the classification table. */
  pageId: string
  section?: string
}

export type BlastRadius = {
  users: number
  courses: number
  orgs: number
  /** When true, counts are approximate and should be labelled as such. */
  approximate?: boolean
}

export type EffectiveSource = 'default' | 'platform' | 'org' | 'course'

export type EffectiveValue = {
  value: unknown
  source: EffectiveSource
  overriddenBy: { scope: string; id: string } | null
}

export const SCOPE_BADGE_LABEL: Record<SettingsScope, string> = {
  me: 'Affects only you',
  org: 'Affects your organisation',
  platform: 'Affects the whole platform',
  course: 'Affects this course',
  operations: 'Operational tools',
}

export const SCOPE_BADGE_LABEL_KEY: Record<SettingsScope, string> = {
  me: 'settings.scope.me',
  org: 'settings.scope.org',
  platform: 'settings.scope.platform',
  course: 'settings.scope.course',
  operations: 'settings.scope.operations',
}
