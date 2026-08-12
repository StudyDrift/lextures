export const COMPARISON_CRITERIA = [
  'Adaptive delivery', 'Question bank and quizzes', 'Rubrics and peer review',
  'Gradebook and audit trail', 'Standards and outcomes', 'SIS roster sync',
  'LTI support', 'SSO and provisioning', 'Accessibility documentation',
  'Self-hosting', 'Data export and API', 'Pricing transparency',
] as const

export type ComparisonCriterion = (typeof COMPARISON_CRITERIA)[number]
