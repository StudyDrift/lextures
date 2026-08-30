import { getAccountType } from './auth'

const AUTH_LOOP_PREFIXES = [
  '/login',
  '/signup',
  '/forgot-password',
  '/reset-password',
  '/activate-parent',
  '/saml-callback',
  '/sso-error',
  '/onboarding',
] as const

function pathnameOf(raw: string): string {
  return raw.split(/[?#]/, 1)[0] ?? raw
}

function isAuthLoopPath(pathname: string): boolean {
  return AUTH_LOOP_PREFIXES.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`))
}

/**
 * Same-origin in-app path only. Rejects protocol-relative URLs, auth pages, and junk.
 */
export function sanitizeReturnPath(preferred: string | null | undefined): string {
  if (preferred == null) return '/'
  const raw = preferred.trim()
  if (raw === '') return '/'
  if (!raw.startsWith('/') || raw.startsWith('//') || raw.includes('\\') || raw.includes('@')) {
    return '/'
  }
  const pathname = pathnameOf(raw)
  if (pathname.includes('://') || isAuthLoopPath(pathname)) {
    return '/'
  }
  return raw
}

/** Builds `/login?next=…` (or `/signup`, `/onboarding`) so a refresh still has the destination. */
export function authHandoffHref(page: string, returnPath: string): string {
  const dest = sanitizeReturnPath(returnPath)
  if (dest === '/') return page
  const join = page.includes('?') ? '&' : '?'
  return `${page}${join}next=${encodeURIComponent(dest)}`
}

export function returnPathFromAuthLocation(location: { search: string; state: unknown }): string {
  const stateFrom =
    location.state && typeof location.state === 'object' && 'from' in location.state
      ? (location.state as { from?: unknown }).from
      : undefined
  const next = new URLSearchParams(location.search).get('next')
  const fromState = typeof stateFrom === 'string' ? stateFrom : undefined
  return sanitizeReturnPath(fromState || next)
}

/** After login or signup, parent accounts land on the parent dashboard unless they already asked for a `/parent` path. */
export function pickPostAuthPath(preferred: string): string {
  const dest = sanitizeReturnPath(preferred)
  if (getAccountType() === 'parent') {
    if (dest.startsWith('/parent')) {
      return dest
    }
    return '/parent'
  }
  return dest
}
