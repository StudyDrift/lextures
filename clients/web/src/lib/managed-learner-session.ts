import { getImpersonationToken } from './auth'

/** JWT `typ` claim of a bearer token, or null when it cannot be read. */
export function jwtTokenType(token: string | null | undefined): string | null {
  const seg = token?.split('.')[1]
  if (!seg) return null
  try {
    const json = atob(seg.replace(/-/g, '+').replace(/_/g, '/'))
    const o = JSON.parse(json) as { typ?: unknown }
    return typeof o.typ === 'string' ? o.typ : null
  } catch {
    return null
  }
}

/**
 * True while a managed learner (kid) is the active bearer — Learn-as from the parent
 * or a family-device sign-in (#771). Admin view-as sessions return false.
 */
export function isManagedLearnerSession(): boolean {
  return jwtTokenType(getImpersonationToken()) === 'managed_learner'
}
