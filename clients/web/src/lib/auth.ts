const ACCESS_TOKEN_KEY = 'studydrift_access_token'
const ACCOUNT_TYPE_KEY = 'studydrift_account_type'
const IMPERSONATION_TOKEN_KEY = 'studydrift_impersonation_token'

/** In-memory fallback when `localStorage` is unavailable (tests, private mode). */
let memoryToken: string | null = null
let memoryImpersonationToken: string | null = null

export function notifyAuthTokenListeners(): void {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event('studydrift-auth-token'))
  }
}

export function setAccessToken(token: string): void {
  try {
    localStorage.setItem(ACCESS_TOKEN_KEY, token)
    memoryToken = null
  } catch {
    memoryToken = token
  }
  notifyAuthTokenListeners()
}

export function getAccessToken(): string | null {
  try {
    return localStorage.getItem(ACCESS_TOKEN_KEY) ?? memoryToken
  } catch {
    return memoryToken
  }
}

/** Active impersonation JWT when an admin is viewing as another user (plan 18.3). */
export function getImpersonationToken(): string | null {
  try {
    return localStorage.getItem(IMPERSONATION_TOKEN_KEY) ?? memoryImpersonationToken
  } catch {
    return memoryImpersonationToken
  }
}

export function setImpersonationToken(token: string): void {
  try {
    localStorage.setItem(IMPERSONATION_TOKEN_KEY, token)
    memoryImpersonationToken = null
  } catch {
    memoryImpersonationToken = token
  }
  notifyAuthTokenListeners()
}

export function clearImpersonationToken(): void {
  memoryImpersonationToken = null
  try {
    localStorage.removeItem(IMPERSONATION_TOKEN_KEY)
  } catch {
    /* ignore */
  }
  notifyAuthTokenListeners()
}

/** Bearer token for API calls — impersonation token takes precedence when active. */
export function getBearerToken(): string | null {
  return getImpersonationToken() ?? getAccessToken()
}

export function clearAccessToken(): void {
  memoryToken = null
  try {
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(ACCOUNT_TYPE_KEY)
  } catch {
    /* ignore */
  }
  notifyAuthTokenListeners()
}

/** Cached account type from last login/signup response (`standard` or `parent`). */
export function getAccountType(): string | null {
  try {
    return localStorage.getItem(ACCOUNT_TYPE_KEY)
  } catch {
    return null
  }
}

export function setCachedAccountTypeFromUser(user?: { accountType?: string | null }): void {
  const v = user?.accountType?.trim()
  try {
    if (v) {
      localStorage.setItem(ACCOUNT_TYPE_KEY, v)
    } else {
      localStorage.removeItem(ACCOUNT_TYPE_KEY)
    }
  } catch {
    /* ignore */
  }
}

/** JWT `sub` claim for the current access token, if parseable. */
export function getJwtSubject(): string | null {
  const t = getAccessToken()
  if (!t) return null
  const seg = t.split('.')[1]
  if (!seg) return null
  try {
    const json = atob(seg.replace(/-/g, '+').replace(/_/g, '/'))
    const o = JSON.parse(json) as { sub?: string }
    return typeof o.sub === 'string' ? o.sub : null
  } catch {
    return null
  }
}
