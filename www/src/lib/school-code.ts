import { API_BASE } from './api-base'

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

export type SchoolLookupResult =
  | { ok: true; slug: string; name: string }
  | { ok: false; reason: 'not_found' | 'unreachable' | 'invalid' }

/** Message when a syntactically valid school code is not registered. */
export const SCHOOL_NOT_FOUND_MESSAGE =
  "We couldn't find a school with that code. Double-check the spelling and try again."

/** Message when the lookup request fails (network / server). */
export const SCHOOL_LOOKUP_UNREACHABLE_MESSAGE =
  "We couldn't verify that school code right now. Check your connection and try again."

export function schoolLookupUrl(code: string): string {
  const slug = normalizeSchoolCode(code)
  return `${API_BASE}/api/v1/public/orgs/by-slug/${encodeURIComponent(slug)}`
}

/**
 * Confirms a school code maps to an active organization before redirecting
 * to the tenant subdomain (avoids sending users to non-existent hosts).
 */
export async function lookupSchoolCode(
  code: string,
  fetchImpl: typeof fetch = fetch,
): Promise<SchoolLookupResult> {
  if (!isValidSchoolCode(code)) {
    return { ok: false, reason: 'invalid' }
  }
  const normalized = normalizeSchoolCode(code)
  try {
    const res = await fetchImpl(schoolLookupUrl(normalized), {
      headers: { Accept: 'application/json' },
    })
    if (res.status === 404) {
      return { ok: false, reason: 'not_found' }
    }
    if (!res.ok) {
      return { ok: false, reason: 'unreachable' }
    }
    const raw: unknown = await res.json().catch(() => ({}))
    const data = raw as { slug?: string; name?: string }
    return {
      ok: true,
      slug: data.slug?.trim() || normalized,
      name: data.name?.trim() || normalized,
    }
  } catch {
    return { ok: false, reason: 'unreachable' }
  }
}
