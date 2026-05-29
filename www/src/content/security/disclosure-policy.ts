/** Public responsible-disclosure policy content (plan 10.16). */

export const SECURITY_CONTACT_EMAIL = 'security@lextures.io'

export const PGP_FINGERPRINT = 'E3F4 9A12 7B6C 8D01 4F2E 91A3 5C7D 0E8B 2A4F 6B9C'

export const PGP_KEY_URL = 'https://keys.openpgp.org/search?q=security%40lextures.io'

export const COORDINATED_DISCLOSURE_DAYS = 90

export const PATCH_SLA_ROWS = [
  { severity: 'Critical', days: '7 calendar days' },
  { severity: 'High', days: '30 calendar days' },
  { severity: 'Medium', days: '90 calendar days' },
  { severity: 'Low', days: 'Next scheduled release' },
] as const

export const IN_SCOPE_ITEMS = [
  'Lextures web application and official API endpoints',
  'Authentication, authorization, and tenant isolation',
  'Student data confidentiality controls',
  'Infrastructure misconfigurations on systems we operate',
] as const

export const OUT_OF_SCOPE_ITEMS = [
  'Denial-of-service against production',
  'Social engineering of staff or customers',
  'Third-party services (report to the vendor)',
  'Issues requiring physical access to a user device',
  'Scanner output without a demonstrated exploit',
] as const
