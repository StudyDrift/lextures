/**
 * Learner-facing coupon helpers (plan MKTC.5).
 * Normalize input, URL/sessionStorage pending codes, and reason → i18n keys.
 */

export type CouponReason =
  | 'ok'
  | 'not_found'
  | 'inactive'
  | 'not_started'
  | 'expired'
  | 'exhausted'
  | 'already_used'
  | 'currency_mismatch'
  | 'course_free'
  | 'owned'
  | string

/** Max characters retained from a URL or typed code (server normalizes further). */
export const COUPON_INPUT_MAX_LEN = 32

const STORAGE_PREFIX = 'lextures.coupon.'

const REASON_KEYS = new Set([
  'ok',
  'not_found',
  'inactive',
  'not_started',
  'expired',
  'exhausted',
  'already_used',
  'currency_mismatch',
  'course_free',
  'owned',
])

/** Upper-case, strip whitespace, truncate, drop characters outside [A-Z0-9_-]. */
export function normalizeCouponInput(raw: string): string {
  const upper = String(raw ?? '')
    .replace(/\s+/g, '')
    .toUpperCase()
  const filtered = upper.replace(/[^A-Z0-9_-]/g, '')
  return filtered.slice(0, COUPON_INPUT_MAX_LEN)
}

/** Read `?coupon=` from a location search string (case-insensitive key). */
export function readCouponFromLocation(search: string): string | null {
  const q = search.startsWith('?') ? search.slice(1) : search
  if (!q) return null
  const params = new URLSearchParams(q)
  // URLSearchParams keys are case-sensitive; scan for coupon case-insensitively.
  let raw: string | null = null
  for (const [k, v] of params.entries()) {
    if (k.toLowerCase() === 'coupon' && v) {
      raw = v
      break
    }
  }
  if (raw == null) return null
  const code = normalizeCouponInput(raw)
  return code || null
}

export function pendingCouponStorageKey(slug: string): string {
  return `${STORAGE_PREFIX}${slug}`
}

export function rememberPendingCoupon(slug: string, code: string): void {
  const normalized = normalizeCouponInput(code)
  if (!slug || !normalized) return
  try {
    sessionStorage.setItem(pendingCouponStorageKey(slug), normalized)
  } catch {
    // private mode / quota — ignore
  }
}

export function readPendingCoupon(slug: string): string | null {
  if (!slug) return null
  try {
    const raw = sessionStorage.getItem(pendingCouponStorageKey(slug))
    if (!raw) return null
    const code = normalizeCouponInput(raw)
    return code || null
  } catch {
    return null
  }
}

export function clearPendingCoupon(slug: string): void {
  if (!slug) return
  try {
    sessionStorage.removeItem(pendingCouponStorageKey(slug))
  } catch {
    // ignore
  }
}

/** Map a server reason token to an i18n key under marketplace.coupon.reason.*. */
export function couponReasonKey(reason: CouponReason): string {
  const r = String(reason || 'not_found').toLowerCase().trim()
  if (REASON_KEYS.has(r)) {
    return `marketplace.coupon.reason.${r}`
  }
  return 'marketplace.coupon.reason.not_found'
}

/** Truncate for safe display (www + app); never use as HTML/attribute. */
export function displayCouponCode(code: string, maxLen = COUPON_INPUT_MAX_LEN): string {
  const n = normalizeCouponInput(code)
  return n.slice(0, maxLen)
}

/**
 * Strip `coupon` from the current URL via history.replaceState (privacy FR).
 * Leaves other query params intact.
 */
export function replaceCouponOutOfUrl(): void {
  if (typeof window === 'undefined' || !window.history?.replaceState) return
  try {
    const url = new URL(window.location.href)
    let changed = false
    for (const key of [...url.searchParams.keys()]) {
      if (key.toLowerCase() === 'coupon') {
        url.searchParams.delete(key)
        changed = true
      }
    }
    if (changed) {
      const next = `${url.pathname}${url.search}${url.hash}`
      window.history.replaceState(window.history.state, '', next)
    }
  } catch {
    // ignore
  }
}
