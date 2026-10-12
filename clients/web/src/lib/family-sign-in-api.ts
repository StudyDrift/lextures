import { apiUrl, authorizedFetch } from './api'
import { getImpersonationToken, setImpersonationToken } from './auth'
import {
  clearFamilySessionLocally,
  clearStoredFamilyDevice,
  getStoredFamilyDevice,
  markFamilySession,
  storeFamilyDevice,
} from './family-device'
import type { ManagedDependent } from './managed-learners-api'

/** Family-device learner sign-in (#771). */

export type SignInMethod = 'none' | 'pin' | 'picture'

export type FamilyDevice = {
  id: string
  label: string
  createdAt: string
  lastUsedAt?: string | null
}

export type FamilyPickerLearner = {
  id: string
  displayName: string
  signInMethod: SignInMethod
  lockedUntil?: string | null
}

export type FamilyPicker = {
  learners: FamilyPickerLearner[]
  pictureChoices: string[]
}

/** Thrown by {@link familySignIn} with the server's lockout / attempts detail. */
export class FamilySignInError extends Error {
  status: number
  attemptsRemaining?: number
  lockedUntil?: string
  constructor(message: string, status: number, extra: { attemptsRemaining?: number; lockedUntil?: string } = {}) {
    super(message)
    this.name = 'FamilySignInError'
    this.status = status
    this.attemptsRemaining = extra.attemptsRemaining
    this.lockedUntil = extra.lockedUntil
  }
}

type ErrorBody = { error?: { message?: string }; attemptsRemaining?: number; lockedUntil?: string }

async function errorMessage(res: Response, fallback: string): Promise<string> {
  const raw = (await res.json().catch(() => ({}))) as ErrorBody
  return raw?.error?.message || fallback
}

export async function listFamilyDevices(): Promise<FamilyDevice[]> {
  const res = await authorizedFetch('/api/v1/me/dependents/family-devices')
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to load family devices'))
  const data = (await res.json()) as { devices?: FamilyDevice[] }
  return data.devices ?? []
}

/** Marks this browser as a family device and remembers its secret locally. */
export async function registerThisFamilyDevice(label?: string): Promise<FamilyDevice> {
  const res = await authorizedFetch('/api/v1/me/dependents/family-devices', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ label: label ?? '' }),
  })
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to set up family device'))
  const data = (await res.json()) as { deviceToken: string; device: FamilyDevice }
  storeFamilyDevice({ id: data.device.id, token: data.deviceToken, label: data.device.label })
  return data.device
}

/** Turns off learner sign-in on a device (forgets it locally when it is this browser). */
export async function revokeFamilyDevice(deviceId: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/dependents/family-devices/${encodeURIComponent(deviceId)}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204 && res.status !== 404) {
    throw new Error(await errorMessage(res, 'Failed to turn off family device'))
  }
  if (getStoredFamilyDevice()?.id === deviceId) clearStoredFamilyDevice()
}

export async function setLearnerSignIn(
  dependentId: string,
  input: { method: 'pin'; pin: string } | { method: 'picture'; picture: string[] },
): Promise<ManagedDependent> {
  const res = await authorizedFetch(`/api/v1/me/dependents/${encodeURIComponent(dependentId)}/sign-in`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to save sign-in'))
  return (await res.json()) as ManagedDependent
}

export async function disableLearnerSignIn(dependentId: string): Promise<ManagedDependent> {
  const res = await authorizedFetch(`/api/v1/me/dependents/${encodeURIComponent(dependentId)}/sign-in`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to turn off sign-in'))
  return (await res.json()) as ManagedDependent
}

/** "Who's learning?" list for this device; null when this browser is not a family device. */
export async function fetchFamilyPicker(): Promise<FamilyPicker | null> {
  const device = getStoredFamilyDevice()
  if (!device) return null
  const res = await fetch(apiUrl('/api/v1/auth/family-device/learners'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ deviceToken: device.token }),
  })
  if (res.status === 401) {
    clearStoredFamilyDevice()
    return null
  }
  if (!res.ok) throw new Error(await errorMessage(res, "Failed to load who's learning"))
  const data = (await res.json()) as Partial<FamilyPicker>
  return { learners: data.learners ?? [], pictureChoices: data.pictureChoices ?? [] }
}

/** Signs a learner in from the picker; the parent's own tokens are left untouched. */
export async function familySignIn(
  learnerId: string,
  secret: { pin: string } | { picture: string[] },
): Promise<void> {
  const device = getStoredFamilyDevice()
  if (!device) throw new FamilySignInError('This device is not set up for learner sign-in.', 401)
  const res = await fetch(apiUrl('/api/v1/auth/family-device/sessions'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ deviceToken: device.token, learnerId, ...secret }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as ErrorBody
    throw new FamilySignInError(raw?.error?.message || 'Sign-in failed', res.status, {
      attemptsRemaining: raw.attemptsRemaining,
      lockedUntil: raw.lockedUntil,
    })
  }
  const data = (await res.json()) as { accessToken: string }
  markFamilySession(true)
  setImpersonationToken(data.accessToken)
}

/** Ends the learner's session (server revoke + local clear). Parent session is unaffected. */
export async function signOutFamilySession(): Promise<void> {
  const token = getImpersonationToken()
  try {
    if (token) {
      await fetch(apiUrl('/api/v1/me/dependents/sessions/current'), {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
    }
  } catch {
    /* network failure — still clear the learner session locally */
  } finally {
    clearFamilySessionLocally()
  }
}
