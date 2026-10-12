/**
 * Family-device learner sign-in state (#771).
 *
 * A parent marks a browser as a family device once; the device secret lives in
 * localStorage so learners can pick themselves from "Who's learning?" without
 * the parent's password. A learner session reuses the impersonation-token slot
 * (it is a managed_learner JWT) but never touches the parent's access/refresh
 * tokens, so signing a learner out leaves the parent session exactly as it was.
 */
import { clearImpersonationToken, getImpersonationToken } from './auth'

const DEVICE_KEY = 'studydrift_family_device'
const FAMILY_SESSION_KEY = 'studydrift_family_session'

export const FAMILY_PICKER_PATH = '/family'

export type StoredFamilyDevice = {
  id: string
  token: string
  label: string
}

let memoryDevice: StoredFamilyDevice | null = null
let memoryFamilySession = false

export function getStoredFamilyDevice(): StoredFamilyDevice | null {
  try {
    const raw = localStorage.getItem(DEVICE_KEY)
    if (!raw) return memoryDevice
    const parsed = JSON.parse(raw) as Partial<StoredFamilyDevice>
    if (typeof parsed?.id === 'string' && typeof parsed?.token === 'string') {
      return { id: parsed.id, token: parsed.token, label: typeof parsed.label === 'string' ? parsed.label : '' }
    }
    return null
  } catch {
    return memoryDevice
  }
}

export function storeFamilyDevice(device: StoredFamilyDevice): void {
  try {
    localStorage.setItem(DEVICE_KEY, JSON.stringify(device))
    memoryDevice = null
  } catch {
    memoryDevice = device
  }
}

export function clearStoredFamilyDevice(): void {
  memoryDevice = null
  try {
    localStorage.removeItem(DEVICE_KEY)
  } catch {
    /* ignore */
  }
}

/** Marks the active impersonation token as a learner's own family-device session. */
export function markFamilySession(active: boolean): void {
  memoryFamilySession = active
  try {
    if (active) localStorage.setItem(FAMILY_SESSION_KEY, '1')
    else localStorage.removeItem(FAMILY_SESSION_KEY)
  } catch {
    /* ignore */
  }
}

/** True while a learner who signed in from the family picker is the active bearer. */
export function isFamilySessionActive(): boolean {
  if (!getImpersonationToken()) return false
  try {
    return localStorage.getItem(FAMILY_SESSION_KEY) === '1' || memoryFamilySession
  } catch {
    return memoryFamilySession
  }
}

/** Drops the learner session locally (parent tokens untouched). */
export function clearFamilySessionLocally(): void {
  markFamilySession(false)
  clearImpersonationToken()
}

/**
 * A 401 during a family-device learner session means the learner token expired
 * or was revoked: return to the picker instead of the parent login.
 * Returns true when handled.
 */
export function handleFamilySessionUnauthorized(): boolean {
  if (!isFamilySessionActive()) return false
  clearFamilySessionLocally()
  if (typeof window !== 'undefined' && window.location.pathname !== FAMILY_PICKER_PATH) {
    window.location.assign(FAMILY_PICKER_PATH)
  }
  return true
}
