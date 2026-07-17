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
] as const satisfies readonly PlatformBooleanFeatureKey[]