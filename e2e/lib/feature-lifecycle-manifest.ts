/**
 * E2E.3 — flagged feature rollback & dependency manifest.
 *
 * One representative lifecycle per family. Disabled HTTP statuses are the
 * documented product contract (not permissive unions). Parent→child edges are
 * validated for cycles; truth-table journeys assert parent authority where the
 * product enforces it.
 */

import type { CourseFeatureKey } from './course-feature-matrix.js'

/** Documented disabled HTTP statuses for authenticated feature gates. */
export type DisabledHttpStatus = 403 | 404 | 501 | 503

/** How unauthenticated requests behave when the flag is off. */
export type UnauthDisabledContract =
  | 'auth-first' // 401 before feature disclosure
  | 'feature-first' // feature guard runs before auth (may return disabled status)

export type LifecyclePriority = 1 | 2

export type LifecycleShard =
  | 'collaboration'
  | 'credentials'
  | 'commerce-api'
  | 'ai'
  | 'priority2'

export type FlagKind = 'platform' | 'course'

export type LifecycleFlagRef = {
  kind: FlagKind
  /** Platform settings/runtime key or course feature JSON key. */
  key: string
  /**
   * When true, product merge forces the flag on (kill switch is elsewhere).
   * Documented so tests do not assert platform-off for always-on masters.
   */
  alwaysOn?: boolean
  /** Optional note for operators (known gaps, course-only gates, etc.). */
  notes?: string
}

export type DependencyEdge = {
  parent: LifecycleFlagRef
  child: LifecycleFlagRef
  /**
   * When true, child surface must remain unavailable while parent is off
   * even if the child flag is on.
   */
  parentAuthoritative: boolean
  /** Documented product gap when parent is not yet enforced. */
  knownGap?: string
}

export type ApiProbe = {
  id: string
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  /**
   * Path template. Supports `{courseCode}` substitution.
   * Absolute path under the API origin (starts with /api/ or /webhooks/).
   */
  path: string
  /** Optional JSON body for mutating probes. */
  body?: Record<string, unknown>
  /** Authenticated caller when the master/child kill switch is off. */
  authDisabledStatus: DisabledHttpStatus
  unauthContract: UnauthDisabledContract
  /**
   * Expected status for unauthenticated probe when the flag is off.
   * Required when unauthContract is feature-first; defaults to 401 for auth-first.
   */
  unauthDisabledStatus?: DisabledHttpStatus | 401
  /** Which flag(s) must be off for this probe's disabled assertion. */
  gatedBy: LifecycleFlagRef[]
  /** Soft skip when the endpoint needs org/admin setup beyond lifecycle scope. */
  requiresRole?: 'any' | 'instructor' | 'global-admin' | 'parent'
}

export type WebSurface = {
  /** Direct route under the SPA (supports `{courseCode}`). */
  route: string
  /** Side-nav link name when enabled; null when nav is not asserted. */
  navLinkName?: RegExp | null
  /** Behavior when the master flag is off. */
  offBehavior: 'hidden-nav' | 'inline-disabled' | 'redirect' | 'runtime-only' | 'none'
  disabledMessage?: RegExp
}

export type LifecycleFamily = {
  id: string
  label: string
  priority: LifecyclePriority
  shard: LifecycleShard
  /** Happy-path specs that own deeper product coverage (not duplicated here). */
  linkedHappyPathSpecs: string[]
  masterFlags: LifecycleFlagRef[]
  children: LifecycleFlagRef[]
  edges: DependencyEdge[]
  /** Representative authenticated API probes for off-state + auth contracts. */
  probes: ApiProbe[]
  web?: WebSurface
  /**
   * When set, lifecycle creates a durable record while on, disables the master,
   * then re-enables and asserts the record still exists.
   */
  dataPreservation?: {
    kind: 'boards' | 'quiz-kits' | 'parent-prefs' | 'runtime-only'
    notes?: string
  }
}

export const FEATURE_LIFECYCLE_FAMILIES: readonly LifecycleFamily[] = [
  // ── Priority 1: collaboration ───────────────────────────────────────────
  {
    id: 'visual-boards',
    label: 'Collaboration boards',
    priority: 1,
    shard: 'collaboration',
    linkedHappyPathSpecs: ['course-features-nav-matrix.spec.ts'],
    masterFlags: [
      {
        kind: 'platform',
        key: 'ffVisualBoards',
        alwaysOn: true,
        notes: 'Platform master is always-on in merge; course visualBoardsEnabled is the kill switch',
      },
      { kind: 'course', key: 'visualBoardsEnabled' },
    ],
    children: [
      { kind: 'platform', key: 'ffBoardsRealtime' },
      { kind: 'platform', key: 'ffBoardsExternalSharing' },
    ],
    edges: [
      {
        parent: { kind: 'course', key: 'visualBoardsEnabled' },
        child: { kind: 'platform', key: 'ffBoardsRealtime' },
        parentAuthoritative: true,
        knownGap: 'Realtime WS is platform-gated; course off already 404s board HTTP before WS',
      },
      {
        parent: { kind: 'course', key: 'visualBoardsEnabled' },
        child: { kind: 'platform', key: 'ffBoardsExternalSharing' },
        parentAuthoritative: true,
      },
    ],
    probes: [
      {
        id: 'list-boards',
        method: 'GET',
        path: '/api/v1/courses/{courseCode}/boards',
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'course', key: 'visualBoardsEnabled' }],
        requiresRole: 'instructor',
      },
      {
        id: 'boards-realtime-ws-upgrade',
        method: 'GET',
        path: '/api/v1/courses/{courseCode}/boards/00000000-0000-0000-0000-000000000001/ws',
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffBoardsRealtime' }],
        requiresRole: 'instructor',
        body: undefined,
      },
    ],
    web: {
      route: '/courses/{courseCode}/boards',
      navLinkName: /^Boards$/,
      offBehavior: 'inline-disabled',
      disabledMessage: /Collaboration boards are not enabled/i,
    },
    dataPreservation: { kind: 'boards' },
  },
  {
    id: 'interactive-quizzes',
    label: 'Live Quizzes',
    priority: 1,
    shard: 'collaboration',
    linkedHappyPathSpecs: ['course-features-nav-matrix.spec.ts'],
    masterFlags: [
      {
        kind: 'course',
        key: 'interactiveQuizzesEnabled',
        notes: 'Kit CRUD is course-scoped; no platform master for kits',
      },
      {
        kind: 'platform',
        key: 'ffIqLiveHosting',
        alwaysOn: true,
        notes: 'COLLAPSE (docs/plan/flags.md): hosting is merged into the per-course flag; platform master is always-on',
      },
    ],
    children: [
      {
        kind: 'platform',
        key: 'ffIqTeamMode',
        alwaysOn: true,
        notes: 'COLLAPSE: quiz mode, no longer an independent platform kill-switch; always-on in merge',
      },
      {
        kind: 'platform',
        key: 'ffIqStudentPaced',
        alwaysOn: true,
        notes: 'COLLAPSE: quiz mode, no longer an independent platform kill-switch; always-on in merge',
      },
      {
        kind: 'platform',
        key: 'ffIqHomework',
        alwaysOn: true,
        notes: 'COLLAPSE: quiz mode, no longer an independent platform kill-switch; always-on in merge',
      },
      {
        kind: 'platform',
        key: 'ffIqGradebookPush',
        alwaysOn: true,
        notes: 'COLLAPSE: folded into the per-course Live Quizzes flag; always-on in merge',
      },
      { kind: 'platform', key: 'ffIqPublicKitCatalog' },
      { kind: 'platform', key: 'ffIqGuestJoin' },
      { kind: 'platform', key: 'ffIqAiGeneration' },
    ],
    edges: [
      {
        parent: { kind: 'course', key: 'interactiveQuizzesEnabled' },
        child: { kind: 'platform', key: 'ffIqLiveHosting' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffIqLiveHosting' },
        child: { kind: 'platform', key: 'ffIqGuestJoin' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffIqLiveHosting' },
        child: { kind: 'platform', key: 'ffIqTeamMode' },
        parentAuthoritative: true,
        knownGap: 'Mode flags are checked when starting a mode; hosting off 404s games first',
      },
      {
        parent: { kind: 'platform', key: 'ffIqLiveHosting' },
        child: { kind: 'platform', key: 'ffIqStudentPaced' },
        parentAuthoritative: true,
        knownGap: 'Mode flags are checked when starting a mode; hosting off 404s games first',
      },
      {
        parent: { kind: 'platform', key: 'ffIqLiveHosting' },
        child: { kind: 'platform', key: 'ffIqHomework' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'course', key: 'interactiveQuizzesEnabled' },
        child: { kind: 'platform', key: 'ffIqGradebookPush' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'course', key: 'interactiveQuizzesEnabled' },
        child: { kind: 'platform', key: 'ffIqAiGeneration' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'course', key: 'interactiveQuizzesEnabled' },
        child: { kind: 'platform', key: 'ffIqPublicKitCatalog' },
        parentAuthoritative: false,
        knownGap: 'Public catalog can include org-public kits independent of a single course flag',
      },
    ],
    probes: [
      {
        id: 'list-kits',
        method: 'GET',
        path: '/api/v1/courses/{courseCode}/live-quizzes/kits',
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'course', key: 'interactiveQuizzesEnabled' }],
        requiresRole: 'instructor',
      },
      {
        id: 'create-game',
        method: 'POST',
        path: '/api/v1/courses/{courseCode}/live-quizzes/kits/00000000-0000-0000-0000-000000000001/games',
        body: { mode: 'classic' },
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffIqLiveHosting' }],
        requiresRole: 'instructor',
      },
      {
        id: 'ai-generation',
        method: 'POST',
        path: '/api/v1/courses/{courseCode}/live-quizzes/kits/00000000-0000-0000-0000-000000000001/generate',
        body: { sourceType: 'prompt', sourceRef: { prompt: 'photosynthesis' }, params: {} },
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffIqAiGeneration' }],
        requiresRole: 'instructor',
      },
    ],
    web: {
      route: '/courses/{courseCode}/live-quizzes',
      navLinkName: /^Live Quizzes$/,
      offBehavior: 'inline-disabled',
      disabledMessage: /Live Quizzes are not enabled/i,
    },
    dataPreservation: { kind: 'quiz-kits' },
  },

  // ── Priority 1: credentials / transcripts / parent ──────────────────────
  {
    id: 'transcripts',
    label: 'Transcripts',
    priority: 1,
    shard: 'credentials',
    linkedHappyPathSpecs: ['transcript-fees.spec.ts', 'ccr.spec.ts'],
    masterFlags: [{ kind: 'platform', key: 'ffTranscripts' }],
    children: [{ kind: 'platform', key: 'ffTranscriptInbound' }],
    edges: [
      {
        parent: { kind: 'platform', key: 'ffTranscripts' },
        child: { kind: 'platform', key: 'ffTranscriptInbound' },
        parentAuthoritative: true,
      },
    ],
    probes: [
      {
        id: 'transcripts-config',
        method: 'GET',
        path: '/api/v1/transcripts/config',
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffTranscripts' }],
        requiresRole: 'any',
      },
      {
        id: 'transcript-inbound-list',
        method: 'GET',
        path: '/api/v1/me/transcripts/inbound',
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffTranscriptInbound' }],
        requiresRole: 'any',
      },
    ],
    web: {
      route: '/settings/transcripts',
      offBehavior: 'runtime-only',
    },
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Order fixtures require registrar setup; lifecycle asserts config/runtime restore',
    },
  },
  {
    id: 'parent-portal',
    label: 'Parent portal',
    priority: 1,
    shard: 'credentials',
    linkedHappyPathSpecs: ['parent-portal.spec.ts'],
    masterFlags: [{ kind: 'platform', key: 'ffParentPortal' }],
    children: [
      {
        kind: 'platform',
        key: 'ffParentPortalV2',
        notes: 'COLLAPSE (docs/plan/flags.md): merge now mirrors ffParentPortal exactly; expanded sections are always included',
      },
      { kind: 'platform', key: 'ffReportCards' },
    ],
    edges: [
      {
        parent: { kind: 'platform', key: 'ffParentPortal' },
        child: { kind: 'platform', key: 'ffParentPortalV2' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffParentPortalV2' },
        child: { kind: 'platform', key: 'ffReportCards' },
        parentAuthoritative: false,
        knownGap: 'Report cards section requires both ffParentPortalV2 and ffReportCards in UI',
      },
    ],
    probes: [
      {
        id: 'parent-children',
        method: 'GET',
        path: '/api/v1/parent/children',
        // Product gap: parent API is account-type gated (403), not feature-flag gated.
        authDisabledStatus: 403,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffParentPortal' }],
        requiresRole: 'any',
      },
    ],
    web: {
      route: '/parent',
      offBehavior: 'runtime-only',
    },
    dataPreservation: { kind: 'parent-prefs' },
  },

  // ── Priority 1: commerce / API ──────────────────────────────────────────
  {
    id: 'payments-billing',
    label: 'Payments / billing / tax / revenue share',
    priority: 1,
    shard: 'commerce-api',
    linkedHappyPathSpecs: [
      'billing.spec.ts',
      'revenue-share.spec.ts',
      'course-marketplace-purchase.spec.ts',
    ],
    masterFlags: [
      { kind: 'platform', key: 'ffPaymentsEnabled' },
      { kind: 'platform', key: 'ffStripeBilling' },
    ],
    children: [
      { kind: 'platform', key: 'ffTaxCollection' },
      { kind: 'platform', key: 'ffRevenueShare' },
    ],
    edges: [
      {
        parent: { kind: 'platform', key: 'ffStripeBilling' },
        child: { kind: 'platform', key: 'ffTaxCollection' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffStripeBilling' },
        child: { kind: 'platform', key: 'ffRevenueShare' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffPaymentsEnabled' },
        child: { kind: 'platform', key: 'ffTaxCollection' },
        parentAuthoritative: false,
        knownGap: 'Tax requires Stripe billing specifically; payments abstraction is separate',
      },
    ],
    probes: [
      {
        id: 'my-transactions',
        method: 'GET',
        path: '/api/v1/me/transactions',
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [
          { kind: 'platform', key: 'ffPaymentsEnabled' },
          { kind: 'platform', key: 'ffStripeBilling' },
        ],
        requiresRole: 'any',
      },
      {
        id: 'tax-quote',
        method: 'POST',
        path: '/api/v1/checkout/quote',
        body: { courseId: '00000000-0000-0000-0000-000000000001', address: { country: 'US' } },
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffTaxCollection' }],
        requiresRole: 'any',
      },
      {
        id: 'creator-earnings',
        method: 'GET',
        path: '/api/v1/creator/earnings',
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffRevenueShare' }],
        requiresRole: 'any',
      },
    ],
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Entitlements/transactions require Stripe; lifecycle asserts kill-switch statuses only',
    },
  },
  {
    id: 'public-api-tokens',
    label: 'Public API / API tokens',
    priority: 1,
    shard: 'commerce-api',
    linkedHappyPathSpecs: ['public-api.spec.ts', 'access-keys.spec.ts'],
    masterFlags: [
      { kind: 'platform', key: 'ffPublicApi' },
      { kind: 'platform', key: 'ffApiTokens' },
    ],
    children: [],
    edges: [
      {
        parent: { kind: 'platform', key: 'ffApiTokens' },
        child: { kind: 'platform', key: 'ffPublicApi' },
        parentAuthoritative: false,
        knownGap: 'Public API and access-key management are independently toggled',
      },
    ],
    probes: [
      {
        id: 'access-keys-list',
        method: 'GET',
        path: '/api/v1/me/access-keys',
        authDisabledStatus: 501,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffApiTokens' }],
        requiresRole: 'any',
      },
    ],
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Keys created while on remain in DB; management returns 501 while ffApiTokens off',
    },
  },

  // ── Priority 1: AI ──────────────────────────────────────────────────────
  {
    id: 'ai-capabilities',
    label: 'AI tutor / study buddy / lesson generator',
    priority: 1,
    shard: 'ai',
    linkedHappyPathSpecs: [
      'ai-tutor.spec.ts',
      'study-buddy.spec.ts',
      'lesson-generator.spec.ts',
    ],
    masterFlags: [
      { kind: 'platform', key: 'ffPersistentTutor' },
      { kind: 'platform', key: 'ffAiStudyBuddy' },
      { kind: 'platform', key: 'ffLessonGenerator' },
      { kind: 'course', key: 'aiTutorEnabled' },
    ],
    children: [],
    edges: [
      {
        parent: { kind: 'platform', key: 'ffPersistentTutor' },
        child: { kind: 'course', key: 'aiTutorEnabled' },
        parentAuthoritative: true,
        knownGap: 'Platform off → 404; course off with platform on → 403',
      },
    ],
    probes: [
      {
        id: 'tutor-sessions',
        method: 'GET',
        path: '/api/v1/courses/{courseCode}/tutor/sessions',
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffPersistentTutor' }],
        requiresRole: 'instructor',
      },
      {
        id: 'study-buddy-prompts',
        method: 'GET',
        path: '/api/v1/courses/{courseCode}/study-buddy/prompts',
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffAiStudyBuddy' }],
        requiresRole: 'instructor',
      },
      {
        id: 'lesson-generator',
        method: 'POST',
        path: '/api/v1/courses/{courseCode}/lesson-generator',
        body: {
          learning_objective: 'Identify the main idea',
          grade_level: '4',
          subject: 'ELA',
        },
        authDisabledStatus: 404,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffLessonGenerator' }],
        requiresRole: 'instructor',
      },
    ],
    web: {
      route: '/courses/{courseCode}',
      offBehavior: 'runtime-only',
    },
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Inference stubbed; lifecycle asserts gates and course flag restore without provider calls',
    },
  },

  // ── Priority 2 (representative off-state contracts) ─────────────────────
  {
    id: 'proctoring',
    label: 'Proctoring',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffProctoringIntegration' }],
    children: [],
    edges: [],
    probes: [
      {
        id: 'quiz-proctoring-config',
        method: 'GET',
        path: '/api/v1/courses/{courseCode}/quizzes/00000000-0000-0000-0000-000000000001/proctoring-config',
        authDisabledStatus: 501,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 501,
        gatedBy: [{ kind: 'platform', key: 'ffProctoringIntegration' }],
        requiresRole: 'instructor',
      },
    ],
  },
  {
    id: 'feedback',
    label: 'Product feedback',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffFeedback' }],
    children: [],
    edges: [],
    probes: [
      {
        id: 'post-feedback',
        method: 'POST',
        path: '/api/v1/feedback',
        body: { message: 'e2e lifecycle', category: 'other' },
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffFeedback' }],
        requiresRole: 'any',
      },
    ],
  },
  {
    id: 'gamification',
    label: 'Gamification',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffGamification' }],
    children: [],
    edges: [],
    probes: [
      {
        id: 'my-gamification',
        method: 'GET',
        path: '/api/v1/me/gamification',
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffGamification' }],
        requiresRole: 'any',
      },
    ],
  },
  {
    id: 'motion',
    label: 'Motion',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffMotionNavigation' }],
    children: [
      {
        kind: 'platform',
        key: 'ffMotionLists',
        notes: 'COLLAPSE (docs/plan/flags.md): merged into ffMotionNavigation, the single motion kill-switch',
      },
      {
        kind: 'platform',
        key: 'ffMotionReveal',
        notes: 'COLLAPSE (docs/plan/flags.md): merged into ffMotionNavigation, the single motion kill-switch',
      },
      {
        kind: 'platform',
        key: 'ffMotionOverlays',
        notes: 'COLLAPSE (docs/plan/flags.md): merged into ffMotionNavigation, the single motion kill-switch',
      },
      {
        kind: 'platform',
        key: 'ffMotionControls',
        notes: 'COLLAPSE (docs/plan/flags.md): merged into ffMotionNavigation, the single motion kill-switch',
      },
    ],
    edges: [
      {
        parent: { kind: 'platform', key: 'ffMotionNavigation' },
        child: { kind: 'platform', key: 'ffMotionLists' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffMotionNavigation' },
        child: { kind: 'platform', key: 'ffMotionReveal' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffMotionNavigation' },
        child: { kind: 'platform', key: 'ffMotionOverlays' },
        parentAuthoritative: true,
      },
      {
        parent: { kind: 'platform', key: 'ffMotionNavigation' },
        child: { kind: 'platform', key: 'ffMotionControls' },
        parentAuthoritative: true,
      },
    ],
    probes: [],
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Motion is client-runtime; lifecycle asserts platform runtime payload toggles',
    },
  },
  {
    id: 'learner-profile',
    label: 'Learner profile / adaptivity',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'learnerProfileEnabled' }],
    children: [],
    edges: [],
    probes: [],
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Runtime payload toggle; deep adaptivity child journeys owned by learner specs',
    },
  },
  {
    id: 'intro-onboarding',
    label: 'Intro course / onboarding',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'introCourseEnabled' }],
    children: [],
    edges: [],
    probes: [],
    dataPreservation: { kind: 'runtime-only' },
  },
  {
    id: 'demographics',
    label: 'Demographics',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffDemographics' }],
    children: [],
    edges: [],
    probes: [],
    dataPreservation: { kind: 'runtime-only' },
  },
  {
    id: 'report-cards',
    label: 'Report cards',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffReportCards' }],
    children: [],
    edges: [],
    probes: [],
    dataPreservation: { kind: 'runtime-only' },
  },
  {
    id: 'identity-provisioning',
    label: 'Identity / provisioning (representative)',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffApiTokens' }],
    children: [],
    edges: [],
    probes: [
      {
        id: 'access-keys-scopes',
        method: 'GET',
        path: '/api/v1/me/access-keys/scopes',
        authDisabledStatus: 501,
        unauthContract: 'auth-first',
        gatedBy: [{ kind: 'platform', key: 'ffApiTokens' }],
        requiresRole: 'any',
      },
    ],
  },
  {
    id: 'grading-workflows',
    label: 'Grading workflows (representative)',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: ['gradebook.spec.ts', 'grading-settings.spec.ts'],
    masterFlags: [{ kind: 'platform', key: 'blindGradingEnabled' }],
    children: [],
    edges: [],
    probes: [],
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'Several grading flags are settings-only; contract covered by E2E.2',
    },
  },
  {
    id: 'ses',
    label: 'SES / email (representative)',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: ['admin-email-templates.spec.ts'],
    masterFlags: [{ kind: 'platform', key: 'ffEmailSes' }],
    children: [],
    edges: [],
    probes: [],
    dataPreservation: {
      kind: 'runtime-only',
      notes: 'SES is a provider toggle; lifecycle asserts runtime payload off/on',
    },
  },
  {
    id: 'evaluations',
    label: 'Course evaluations (representative)',
    priority: 2,
    shard: 'priority2',
    linkedHappyPathSpecs: [],
    masterFlags: [{ kind: 'platform', key: 'ffFeedback' }],
    children: [],
    edges: [],
    probes: [
      {
        id: 'feedback-when-evals-proxy',
        method: 'POST',
        path: '/api/v1/feedback',
        body: { message: 'e2e evals proxy', category: 'other' },
        authDisabledStatus: 404,
        unauthContract: 'feature-first',
        unauthDisabledStatus: 404,
        gatedBy: [{ kind: 'platform', key: 'ffFeedback' }],
        requiresRole: 'any',
      },
    ],
  },
]

export function familiesForShard(shard: LifecycleShard): LifecycleFamily[] {
  return FEATURE_LIFECYCLE_FAMILIES.filter((f) => f.shard === shard)
}

export function priorityFamilies(priority: LifecyclePriority): LifecycleFamily[] {
  return FEATURE_LIFECYCLE_FAMILIES.filter((f) => f.priority === priority)
}

export function flagKey(ref: LifecycleFlagRef): string {
  return `${ref.kind}:${ref.key}`
}

/** Detect cycles in parent→child edges (within and across families). */
export function detectParentCycles(
  families: readonly LifecycleFamily[] = FEATURE_LIFECYCLE_FAMILIES,
): string[] {
  const adj = new Map<string, Set<string>>()
  for (const family of families) {
    for (const edge of family.edges) {
      const p = flagKey(edge.parent)
      const c = flagKey(edge.child)
      if (!adj.has(p)) adj.set(p, new Set())
      adj.get(p)!.add(c)
      if (!adj.has(c)) adj.set(c, new Set())
    }
  }

  const cycles: string[] = []
  const visiting = new Set<string>()
  const visited = new Set<string>()
  const stack: string[] = []

  function dfs(node: string) {
    if (visiting.has(node)) {
      const start = stack.indexOf(node)
      const path = start >= 0 ? stack.slice(start).concat(node) : [node, node]
      cycles.push(path.join(' → '))
      return
    }
    if (visited.has(node)) return
    visiting.add(node)
    stack.push(node)
    for (const next of adj.get(node) ?? []) {
      dfs(next)
    }
    stack.pop()
    visiting.delete(node)
    visited.add(node)
  }

  for (const node of adj.keys()) {
    dfs(node)
  }
  return [...new Set(cycles)]
}

const ALLOWED_DISABLED = new Set<number>([403, 404, 501, 503])

export function validateFeatureLifecycleManifest(
  families: readonly LifecycleFamily[] = FEATURE_LIFECYCLE_FAMILIES,
): string[] {
  const errors: string[] = []
  const ids = new Set<string>()
  const probeIds = new Set<string>()

  const requiredPriority1 = [
    'visual-boards',
    'interactive-quizzes',
    'transcripts',
    'parent-portal',
    'payments-billing',
    'public-api-tokens',
    'ai-capabilities',
  ]
  for (const id of requiredPriority1) {
    if (!families.some((f) => f.id === id && f.priority === 1)) {
      errors.push(`missing Priority 1 family: ${id}`)
    }
  }

  const requiredPriority2Topics = [
    'identity-provisioning',
    'grading-workflows',
    'learner-profile',
    'intro-onboarding',
    'proctoring',
    'feedback',
    'ses',
    'motion',
    'gamification',
    'evaluations',
    'demographics',
    'report-cards',
  ]
  for (const id of requiredPriority2Topics) {
    if (!families.some((f) => f.id === id && f.priority === 2)) {
      errors.push(`missing Priority 2 family: ${id}`)
    }
  }

  for (const family of families) {
    if (ids.has(family.id)) errors.push(`duplicate family id: ${family.id}`)
    ids.add(family.id)

    if (!family.label.trim()) errors.push(`${family.id}: empty label`)
    if (family.masterFlags.length === 0) errors.push(`${family.id}: needs at least one master flag`)

    for (const edge of family.edges) {
      const parentKnown =
        family.masterFlags.some((f) => flagKey(f) === flagKey(edge.parent)) ||
        family.children.some((f) => flagKey(f) === flagKey(edge.parent))
      const childKnown =
        family.masterFlags.some((f) => flagKey(f) === flagKey(edge.child)) ||
        family.children.some((f) => flagKey(f) === flagKey(edge.child))
      if (!parentKnown) {
        errors.push(`${family.id}: edge parent ${flagKey(edge.parent)} not listed on family`)
      }
      if (!childKnown) {
        errors.push(`${family.id}: edge child ${flagKey(edge.child)} not listed on family`)
      }
      if (!edge.parentAuthoritative && !edge.knownGap?.trim()) {
        errors.push(
          `${family.id}: non-authoritative edge ${flagKey(edge.parent)}→${flagKey(edge.child)} needs knownGap`,
        )
      }
    }

    for (const probe of family.probes) {
      const pid = `${family.id}/${probe.id}`
      if (probeIds.has(pid)) errors.push(`duplicate probe id: ${pid}`)
      probeIds.add(pid)

      if (!ALLOWED_DISABLED.has(probe.authDisabledStatus)) {
        errors.push(`${pid}: authDisabledStatus ${probe.authDisabledStatus} not in contract set`)
      }
      if (probe.unauthContract === 'auth-first') {
        if (probe.unauthDisabledStatus != null && probe.unauthDisabledStatus !== 401) {
          errors.push(`${pid}: auth-first probes must use unauth 401 (or omit)`)
        }
      } else if (probe.unauthDisabledStatus == null) {
        errors.push(`${pid}: feature-first probes must set unauthDisabledStatus`)
      } else if (
        probe.unauthDisabledStatus !== 401 &&
        !ALLOWED_DISABLED.has(probe.unauthDisabledStatus)
      ) {
        errors.push(`${pid}: invalid unauthDisabledStatus ${probe.unauthDisabledStatus}`)
      }
      if (probe.gatedBy.length === 0) {
        errors.push(`${pid}: gatedBy must list at least one flag`)
      }
      if (!probe.path.startsWith('/')) {
        errors.push(`${pid}: path must be absolute`)
      }
    }

    for (const flag of [...family.masterFlags, ...family.children]) {
      if (flag.kind === 'course') {
        // Soft-check known course keys; unknown keys are still allowed for future flags.
        const known: CourseFeatureKey[] = [
          'visualBoardsEnabled',
          'interactiveQuizzesEnabled',
          'aiTutorEnabled',
          'reportCardsEnabled',
        ]
        if (!known.includes(flag.key as CourseFeatureKey) && !flag.notes) {
          // not an error — keep extensible
        }
      }
      if (flag.alwaysOn && !flag.notes?.trim()) {
        errors.push(`${family.id}: alwaysOn flag ${flagKey(flag)} needs notes`)
      }
    }
  }

  for (const cycle of detectParentCycles(families)) {
    errors.push(`dependency cycle: ${cycle}`)
  }

  return errors
}
