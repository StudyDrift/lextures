/** i18n-style keys for at-risk alerts UI (plan 9.2). */
export const atRiskI18n = {
  title: 'At-risk students',
  score: 'At-risk score',
  missingWork: 'Missing work',
  inactive: 'Inactive',
  dismiss: 'Dismiss',
  snooze: 'Snooze',
  snooze7: 'Snooze 7 days',
  snooze14: 'Snooze 14 days',
  supported: 'Being supported',
  addNote: 'Add note',
  viewProgress: 'View progress',
  resolved: 'Resolved',
  empty: 'No at-risk students — all students are on track.',
  severityModerate: 'Moderate risk',
  severityHigh: 'High risk',
  primaryFactor: 'Primary concern',
  notePlaceholder: 'e.g. Student contacted — plans to submit by Friday',
} as const

export function atRiskFeatureEnabled(): boolean {
  return import.meta.env.VITE_FEATURE_AT_RISK_ALERTS === 'true'
}
