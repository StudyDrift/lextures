const SCHOOL_CODE_PATTERN = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/

const RESERVED_SCHOOL_CODES = new Set([
  'admin',
  'api',
  'app',
  'default',
  'demo',
  'login',
  'magic-link',
  'mfa',
  'self',
  'signup',
  'www',
])

export function normalizeSchoolCode(raw: string): string {
  return raw.trim().toLowerCase()
}

export function schoolCodeError(code: string): string | null {
  const normalized = normalizeSchoolCode(code)
  if (!normalized) return 'Enter your school code.'
  if (normalized.length < 2) return 'School codes are at least 2 characters.'
  if (normalized.length > 32) return 'School codes are at most 32 characters.'
  if (!SCHOOL_CODE_PATTERN.test(normalized)) {
    return 'Use lowercase letters, numbers, and hyphens only.'
  }
  if (RESERVED_SCHOOL_CODES.has(normalized)) {
    return 'That code is reserved.'
  }
  return null
}

export function isValidSchoolCode(code: string): boolean {
  return schoolCodeError(code) === null
}