export const DEMO_ORIGIN = 'https://demo.lextures.com'
export const SELF_LEARNER_ORIGIN = 'https://self.lextures.com'
export const TENANT_HOST_SUFFIX = 'lextures.com'

/** Primary hosted app origin (self-learner instance). */
export const APP_ORIGIN = SELF_LEARNER_ORIGIN

export function tenantOrigin(schoolCode: string) {
  return `https://${schoolCode}.${TENANT_HOST_SUFFIX}/`
}

export const SITE_LINKS = {
  demo: `${DEMO_ORIGIN}/`,
  selfLearner: `${SELF_LEARNER_ORIGIN}/`,
  github: 'https://github.com/StudyDrift/lextures',
  institutionInquiryEmail: 'chase@lextures.com',
  privacy: '/privacy',
  privacyHistory: '/privacy/history',
  terms: '/terms',
  termsHistory: '/terms/history',
  security: '/security',
  accessibility: '/accessibility',
  accessibilityVpat: '/accessibility/vpat',
  californiaPrivacyRights: '/privacy-rights/california',
} as const