/**
 * UX.7 — Navigation registry types.
 *
 * Single source of truth for every shell destination. Sidebars, command palette,
 * breadcrumbs, and the collision CI check all consume this shape.
 */

/** Shell navigation scopes (preferences key prefix uses the same values). */
export type NavScopeKind = 'global' | 'course' | 'settings' | 'course-settings' | 'admin'

/** Who may see a destination (after permission + feature-flag gates). */
export type NavAudience = 'any' | 'student' | 'instructor' | 'admin' | 'parent'

/** Stable section ids. Legacy ids preserve current IA; V2 ids are task-based. */
export type NavSectionId =
  | 'primary'
  | 'content'
  | 'collaboration'
  | 'your-learning'
  | 'assessment'
  | 'grades-insights'
  | 'people'
  | 'manage'
  | 'learning'
  | 'notes-portfolio'
  | 'records'
  | 'family'
  | 'administration'
  | 'account'
  | 'teach'
  | 'engage'
  | 'assess-analyse'
  | 'participate'
  | 'my-work'
  | 'user-settings'
  | 'org-settings'
  | 'platform'
  | 'course-setup'
  | 'grading-outcomes'
  | 'access-localization'
  | 'content-data'
  | 'lifecycle'
  | 'admin-tools'
  | 'pinned'
  | 'recent'
  | 'more'

export type NavDestinationId = string

/**
 * Declarative destination. Visibility predicates cover cases that cannot be
 * expressed as a single permission or feature-flag key.
 */
export type NavDestination = {
  /** Stable id — preferences and telemetry key. Never reuse after retirement. */
  id: NavDestinationId
  /**
   * Route path. Use `:courseCode` for in-course routes.
   * Absolute paths only (start with `/`).
   */
  route: string
  /** i18n key under `common` (e.g. `nav.dest.gradebook`). */
  labelKey: string
  /** English fallback until all locales ship keys. */
  label: string
  /**
   * Lucide icon export name (e.g. `ClipboardList`). Unique within a scope
   * (enforced by CI).
   */
  icon: string
  /** Legacy section (current IA, default when ffNavigationV2 is off). */
  section: NavSectionId
  /** Task-based section when ffNavigationV2 is on. */
  sectionV2: NavSectionId
  /**
   * When true and V2 is on, the destination is listed in the ungrouped primary
   * block (FR-4 budget of ≤7).
   */
  primaryV2?: boolean
  /**
   * Default priority rank within a section (lower = higher). Never alphabetical
   * below 20 items (R-12).
   */
  priority: number
  /** Audiences that may see this destination. Empty = treat as `any`. */
  audience: NavAudience[]
  /**
   * Permission string, or a template with `:courseCode`.
   * When set, destination is hidden unless `allows(permission)` is true.
   */
  permission?: string
  /**
   * Platform feature flag key on `PlatformFeatures` / snapshot
   * (e.g. `ffLibrary`, `ragNotebookEnabled`).
   */
  featureFlag?: string
  /**
   * Course-nav feature key from `useCourseNavFeatures`
   * (e.g. `notebookEnabled`, `filesEnabled`).
   */
  courseFeature?: string
  /** Cannot be hidden via personalisation. */
  essential?: boolean
  /** English synonyms for command-palette fuzzy match (FR-18). */
  synonyms?: string[]
  /** Per-locale synonyms; falls back to `synonyms` for en. */
  synonymsByLocale?: Partial<Record<string, string[]>>
  /** Match NavLink with `end`. */
  end?: boolean
  /**
   * When true, omit from the main link list (utility rows like "Back").
   * Still indexed for palette when `indexInPalette` is true.
   */
  utility?: boolean
  /** Include in command palette (default true). */
  indexInPalette?: boolean
  /** Optional active-path prefix override (settings sub-routes). */
  activePathPrefix?: string
}

export type NavSectionMeta = {
  id: NavSectionId
  labelKey: string
  label: string
  /** Sort order among sections (lower first). */
  order: number
  /** V2 sort order. */
  orderV2: number
  /** Max visible links before "More" when V2 is on (default 6). */
  visibleBudget?: number
}

export type NavPreferences = {
  scope: string
  pinned: string[]
  hidden: string[]
  collapsed: string[]
}

export type NavResolveContext = {
  scope: NavScopeKind
  /** `global` | `course:<code>` | `settings` | `course-settings:<code>` | `admin` */
  preferenceScope: string
  courseCode?: string
  /** Effective audience after "View as" switching. */
  audience: NavAudience
  allows: (permission: string) => boolean
  permLoading: boolean
  /** Platform feature snapshot (boolean flags). */
  platform: Record<string, unknown>
  /** Course nav features (boolean flags). */
  courseFeatures: Record<string, unknown>
  /** When true, use V2 sections / primary / overflow. */
  navigationV2: boolean
  prefs: NavPreferences
  /** Show hidden destinations (customise sheet). */
  showHidden?: boolean
}

export type ResolvedNavItem = {
  dest: NavDestination
  href: string
  label: string
  section: NavSectionId
  isPinned: boolean
  isHidden: boolean
  isPrimary: boolean
  /** Overflowed past section budget (V2). */
  inMore: boolean
}

export type ResolvedNavSection = {
  id: NavSectionId
  label: string
  items: ResolvedNavItem[]
  moreItems: ResolvedNavItem[]
  collapsed: boolean
}

export type ResolvedNavModel = {
  scope: NavScopeKind
  preferenceScope: string
  utility: ResolvedNavItem[]
  pinned: ResolvedNavItem[]
  recent: ResolvedNavItem[]
  primary: ResolvedNavItem[]
  sections: ResolvedNavSection[]
  /** All visible (non-hidden) items for palette/telemetry. */
  allVisible: ResolvedNavItem[]
  /** Hidden items still findable via search. */
  hidden: ResolvedNavItem[]
}
