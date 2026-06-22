import type { PlatformBooleanFeatureKey } from './platform-feature-definitions'

/**
 * Platform boolean flags persisted by PUT /api/v1/settings/platform but managed
 * outside Settings → Global platform (dedicated settings panels).
 */
export const PLATFORM_FEATURE_EXEMPT_KEYS = [
  'ffContentFilterIntegration',
  'ffPlagiarismChecks',
  'ffStudyReminders',
] as const satisfies readonly PlatformBooleanFeatureKey[]