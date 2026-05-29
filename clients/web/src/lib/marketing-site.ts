export const MARKETING_SITE_ORIGIN = 'https://lextures.com'

export const MARKETING_SITE_URLS = {
  privacy: `${MARKETING_SITE_ORIGIN}/privacy`,
  privacyHistory: `${MARKETING_SITE_ORIGIN}/privacy/history`,
  terms: `${MARKETING_SITE_ORIGIN}/terms`,
  termsHistory: `${MARKETING_SITE_ORIGIN}/terms/history`,
  security: `${MARKETING_SITE_ORIGIN}/security`,
  accessibility: `${MARKETING_SITE_ORIGIN}/accessibility`,
  accessibilityVpat: `${MARKETING_SITE_ORIGIN}/accessibility/vpat`,
  californiaPrivacyRights: `${MARKETING_SITE_ORIGIN}/privacy-rights/california`,
} as const

/** @deprecated Prefer MARKETING_SITE_URLS */
export const MARKETING_LEGAL_URLS = MARKETING_SITE_URLS
