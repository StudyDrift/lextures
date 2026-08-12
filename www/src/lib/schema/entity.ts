/**
 * Brand entity constants for Organization / sameAs (SEO.3 FR-6, FR-7).
 *
 * sameAs lists only verified, owned or claimed profiles. Unclaimed third-party
 * listings must not be added (coordinate new profiles with SEO.13).
 */
import { SITE_ORIGIN } from '../site-origin'
import { SITE_LINKS } from '../site-links'

export const BRAND = {
  name: 'Lextures',
  alternateName: ['StudyDrift'],
  legalName: 'Lextures LLC',
  description:
    'Adaptive learning management system for K–12, higher education, and homeschool: IRT-routed quizzes, gradebook, roster sync, self-hosting under AGPL-3.0, and a public course marketplace.',
  /** ISO date — inception of the Lextures product / company. */
  foundingDate: '2024-01-01',
  email: 'chase@lextures.com',
  supportEmail: 'support@lextures.com',
  salesEmail: SITE_LINKS.institutionInquiryEmail,
  pressEmail: SITE_LINKS.pressEmail,
  accessibilityEmail: 'accessibility@lextures.com',
  url: SITE_ORIGIN.replace(/\/$/, '') + '/',
  logoPath: '/logo.svg',
  /** Topics we actually publish and product for (must stay truthful). */
  knowsAbout: [
    'adaptive learning',
    'Item Response Theory',
    'learning management systems',
    'spaced repetition',
    'standards-aligned assessment',
    'WCAG accessibility',
    'FERPA-compliant student data',
    'LTI 1.3',
    'SCIM provisioning',
    'K-12 gradebook',
    'higher education LMS',
    'homeschool curriculum software',
    'open source education software',
    'misconception detection',
    'Bloom\'s taxonomy in AI-era assessment',
    'rubric design',
  ],
  availableLanguage: ['en'],
  areaServed: 'Worldwide',
} as const

/**
 * Verified sameAs allowlist. Only URLs we own or have claimed.
 * Add Wikidata QID and review-site profiles after SEO.13 claims them.
 */
export const VERIFIED_SAME_AS: readonly string[] = [
  SITE_LINKS.github,
  // Add once claimed (SEO.13):
  // 'https://www.linkedin.com/company/lextures',
  // 'https://x.com/lextures',
  // 'https://www.youtube.com/@lextures',
  // 'https://www.crunchbase.com/organization/lextures',
  // 'https://www.g2.com/products/lextures/...',
  // 'https://www.capterra.com/p/.../Lextures/',
  // 'https://www.wikidata.org/wiki/Q…',
]

export const SOFTWARE_FEATURES: readonly string[] = [
  'IRT-routed adaptive quizzes',
  'Spaced-repetition review scheduling',
  'Gradebook with audit trail',
  'Roster sync via SCIM, Clever, and ClassLink',
  'LTI 1.3 tool provider and platform consumer',
  'SAML/OIDC single sign-on',
  'Self-hosting under AGPL-3.0',
  'Native iOS and Android apps',
  'Public course marketplace',
  'Parent portal for K–12',
]

/** Homeschool hosted plan list price (USD / month) — matches pricing page. */
export const HOMESCHOOL_MONTHLY_USD = 20
