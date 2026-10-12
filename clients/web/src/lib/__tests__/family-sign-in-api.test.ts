import { http, HttpResponse } from 'msw'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { getAccessToken, getImpersonationToken, setAccessToken } from '../auth'
import { clearFamilySessionLocally, clearStoredFamilyDevice, getStoredFamilyDevice, isFamilySessionActive } from '../family-device'
import {
  FamilySignInError,
  familySignIn,
  fetchFamilyPicker,
  registerThisFamilyDevice,
  revokeFamilyDevice,
  setLearnerSignIn,
  signOutFamilySession,
} from '../family-sign-in-api'
import { server } from '../../test/mocks/server'

const API = 'http://localhost:8080'

describe('family-device learner sign-in (#771)', () => {
  beforeEach(() => {
    setAccessToken('parent-token')
    server.use(
      http.post(`${API}/api/v1/me/dependents/family-devices`, () =>
        HttpResponse.json(
          { deviceToken: 'device-secret', device: { id: 'dev-1', label: 'Family device', createdAt: '2026-10-12T00:00:00Z' } },
          { status: 201 },
        ),
      ),
    )
  })

  afterEach(() => {
    clearFamilySessionLocally()
    clearStoredFamilyDevice()
  })

  it('remembers the device secret after the parent registers this browser', async () => {
    await registerThisFamilyDevice()
    expect(getStoredFamilyDevice()).toEqual({ id: 'dev-1', token: 'device-secret', label: 'Family device' })
  })

  it('returns null for the picker when this browser is not a family device', async () => {
    expect(await fetchFamilyPicker()).toBeNull()
  })

  it('loads the picker with the device secret and forgets a revoked device', async () => {
    await registerThisFamilyDevice()
    let sentToken = ''
    server.use(
      http.post(`${API}/api/v1/auth/family-device/learners`, async ({ request }) => {
        sentToken = ((await request.json()) as { deviceToken: string }).deviceToken
        return HttpResponse.json({
          learners: [{ id: 'kid-1', displayName: 'Alex', signInMethod: 'pin' }],
          pictureChoices: ['cat'],
        })
      }),
    )
    const picker = await fetchFamilyPicker()
    expect(sentToken).toBe('device-secret')
    expect(picker?.learners[0].displayName).toBe('Alex')

    server.use(
      http.post(`${API}/api/v1/auth/family-device/learners`, () =>
        HttpResponse.json({ error: { code: 'UNAUTHORIZED', message: 'nope' } }, { status: 401 }),
      ),
    )
    expect(await fetchFamilyPicker()).toBeNull()
    expect(getStoredFamilyDevice()).toBeNull()
  })

  it('starts a learner session without touching the parent token, and sign-out keeps it', async () => {
    await registerThisFamilyDevice()
    let revoked = false
    server.use(
      http.post(`${API}/api/v1/auth/family-device/sessions`, () =>
        HttpResponse.json({ accessToken: 'kid-token', expiresAt: '2026-10-12T08:00:00Z', target: { id: 'kid-1' } }),
      ),
      http.delete(`${API}/api/v1/me/dependents/sessions/current`, ({ request }) => {
        revoked = request.headers.get('Authorization') === 'Bearer kid-token'
        return new HttpResponse(null, { status: 204 })
      }),
    )
    await familySignIn('kid-1', { pin: '1234' })
    expect(getImpersonationToken()).toBe('kid-token')
    expect(isFamilySessionActive()).toBe(true)
    expect(getAccessToken()).toBe('parent-token')

    await signOutFamilySession()
    expect(revoked).toBe(true)
    expect(getImpersonationToken()).toBeNull()
    expect(isFamilySessionActive()).toBe(false)
    expect(getAccessToken()).toBe('parent-token')
  })

  it('surfaces attempts remaining and lockout details', async () => {
    await registerThisFamilyDevice()
    server.use(
      http.post(`${API}/api/v1/auth/family-device/sessions`, () =>
        HttpResponse.json(
          { error: { code: 'INVALID_CREDENTIALS', message: 'no' }, attemptsRemaining: 3 },
          { status: 401 },
        ),
      ),
    )
    const wrong = await familySignIn('kid-1', { pin: '0000' }).catch((e: unknown) => e)
    expect(wrong).toBeInstanceOf(FamilySignInError)
    expect((wrong as FamilySignInError).attemptsRemaining).toBe(3)
    expect(getImpersonationToken()).toBeNull()

    server.use(
      http.post(`${API}/api/v1/auth/family-device/sessions`, () =>
        HttpResponse.json(
          { error: { code: 'RATE_LIMITED', message: 'locked' }, lockedUntil: '2026-10-12T10:15:00Z' },
          { status: 429 },
        ),
      ),
    )
    const locked = (await familySignIn('kid-1', { pin: '0000' }).catch((e: unknown) => e)) as FamilySignInError
    expect(locked.status).toBe(429)
    expect(locked.lockedUntil).toBe('2026-10-12T10:15:00Z')
  })

  it('sends the PIN / picture payload when a parent sets sign-in', async () => {
    let body: unknown = null
    server.use(
      http.put(`${API}/api/v1/me/dependents/kid-1/sign-in`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ id: 'kid-1', displayName: 'Alex', signInMethod: 'picture' })
      }),
    )
    const updated = await setLearnerSignIn('kid-1', { method: 'picture', picture: ['cat', 'sun', 'tree'] })
    expect(body).toEqual({ method: 'picture', picture: ['cat', 'sun', 'tree'] })
    expect(updated.signInMethod).toBe('picture')
  })

  it('forgets this browser when its device is revoked', async () => {
    await registerThisFamilyDevice()
    server.use(
      http.delete(`${API}/api/v1/me/dependents/family-devices/dev-1`, () => new HttpResponse(null, { status: 204 })),
    )
    await revokeFamilyDevice('dev-1')
    expect(getStoredFamilyDevice()).toBeNull()
  })
})
