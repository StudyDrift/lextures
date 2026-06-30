import { apiUrl } from './api'
import {
  clearImpersonationToken,
  getAccessToken,
  getBearerToken,
  getImpersonationToken,
  notifyAuthTokenListeners,
  setAccessToken,
  setImpersonationToken,
} from './auth'

const ADMIN_BACKUP_TOKEN_KEY = 'studydrift_admin_session_backup'

let memoryBackup: string | null = null

export type MeImpersonation = {
  adminId: string
}

export type MeProfile = {
  id: string
  email: string
  displayName?: string | null
  impersonating?: MeImpersonation
}

function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* ignore */
  }
}

function removeStorage(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    /* ignore */
  }
}

export { getBearerToken, getImpersonationToken }

export function isImpersonating(): boolean {
  return getImpersonationToken() != null
}

/** Store impersonation JWT and back up the admin session token. */
export function startImpersonationSession(impersonationToken: string): void {
  const current = getAccessToken()
  if (current) {
    writeStorage(ADMIN_BACKUP_TOKEN_KEY, current)
    memoryBackup = null
  }
  setImpersonationToken(impersonationToken)
}

/** Clear impersonation state and restore the backed-up admin token. */
export function endImpersonationSession(): void {
  clearImpersonationToken()
  const backup = readStorage(ADMIN_BACKUP_TOKEN_KEY) ?? memoryBackup
  memoryBackup = null
  removeStorage(ADMIN_BACKUP_TOKEN_KEY)
  if (backup) {
    setAccessToken(backup)
  }
  notifyAuthTokenListeners()
}

/** End impersonation on the server then restore the admin session locally. */
export async function exitImpersonation(): Promise<void> {
  const token = getImpersonationToken()
  if (token) {
    await fetch(apiUrl('/api/v1/admin-console/impersonate/session'), {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    })
  }
  endImpersonationSession()
}

export async function fetchMeProfile(): Promise<MeProfile | null> {
  const token = getBearerToken()
  if (!token) return null
  const res = await fetch(apiUrl('/api/v1/me'), {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return null
  return (await res.json()) as MeProfile
}
