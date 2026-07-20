import type { PlatformBooleanFeatureKey } from './platform-feature-definitions'

/**
 * Platform boolean flags persisted by PUT /api/v1/settings/platform but managed
 * outside Settings → Global platform (dedicated settings panels).
 */
export const PLATFORM_FEATURE_EXEMPT_KEYS = [
  'ffContentFilterIntegration',
  'ffPlagiarismChecks',
  'ffStudyReminders',
  // Live Quizzes are course-scoped only; platform master is always on (ignored).
  'ffInteractiveQuizzes',
  // Collaboration boards platform master is always on (ignored); course toggle is the real gate.
  'ffVisualBoards',
  // In-app product feedback is always on (platform master removed).
  'ffFeedback',
  // Mobile parity flags are always on (platform masters removed).
  'ffMobileCreateCourse',
  'ffMobileCourseCreateV2',
  'ffMobileCanvasImport',
  'ffMobileAdminConsole',
  'ffMobileEnrollmentAdd',
  'ffMobileLiveQuiz',
  'ffMobileWhiteboardEdit',
  'ffMobileMarketplacePurchase',
  'ffMobileBoardsAdvanced',
  // COLLAPSE (docs/plan/flags.md): folded into ffIqLiveHosting's per-course merge; always-on, no
  // independent Global platform toggle.
  'ffIqLiveHosting',
  'ffIqTeamMode',
  'ffIqStudentPaced',
  'ffIqHomework',
  'ffIqGradebookPush',
  // COLLAPSE: motion kill-switches merged into ffMotionNavigation.
  'ffMotionReveal',
  'ffMotionLists',
  'ffMotionOverlays',
  'ffMotionControls',
  'ffMotionDelight',
  // COLLAPSE: accommodations audit log always follows accommodationsEngineEnabled.
  'ffAccommodationsEngine',
  // COLLAPSE: parent portal V2 sections merged into ffParentPortal.
  'ffParentPortalV2',
] as const satisfies readonly PlatformBooleanFeatureKey[]