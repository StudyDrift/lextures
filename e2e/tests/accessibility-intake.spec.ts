/**
 * Accessibility services intake workflow (plan 14.16)
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'acc') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'acc') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? PASSWORD
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (!loginRes.ok) {
    await fetch(`${API_BASE}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: adminEmail, password: adminPassword, display_name: 'E2E Admin' }),
    })
    const retry = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: adminEmail, password: adminPassword }),
    })
    const { access_token } = (await retry.json()) as { access_token: string }
    return access_token
  }
  const { access_token } = (await loginRes.json()) as { access_token: string }
  return access_token
}

async function enableFeature(adminToken: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffAccessibilityIntake: true,
      updateMask: ['ffAccessibilityIntake'],
    }),
  })
  expect(res.ok).toBeTruthy()
}

async function myUserId(token: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(token) })
  expect(res.ok).toBeTruthy()
  const { id } = (await res.json()) as { id: string }
  return id
}

test('Accessibility intake: unauthenticated endpoints return 401', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)

  const paths = [
    '/api/v1/accessibility/profiles',
    '/api/v1/me/accommodation-profiles',
  ]
  for (const path of paths) {
    const res = await fetch(`${API_BASE}${path}`)
    expect(res.status).toBe(401)
  }
})

test('Accessibility intake: coordinator profile propagates extended time and deactivation removes it (AC-1, AC-3, AC-5)', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)

  const student = await apiSignup({ email: uniqueEmail('student'), password: PASSWORD })
  const studentId = await myUserId(student.access_token)

  // Coordinator (admin) creates a 1.5x extended-time profile.
  const createRes = await fetch(`${API_BASE}/api/v1/accessibility/profiles`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      studentId,
      accommodations: ['extended_time_1_5x', 'reduced_distraction'],
    }),
  })
  expect(createRes.status).toBe(201)
  const { profile } = (await createRes.json()) as { profile: { id: string; labels: string[] } }
  expect(profile.labels).toContain('1.5x Extended Time')

  // AC-3: student sees the profile and affected courses payload.
  const mineRes = await fetch(`${API_BASE}/api/v1/me/accommodation-profiles`, {
    headers: authHeaders(student.access_token),
  })
  expect(mineRes.ok).toBeTruthy()
  const mine = (await mineRes.json()) as { profiles: { labels: string[] }[]; affectedCourses: unknown[] }
  expect(mine.profiles.length).toBe(1)
  expect(mine.profiles[0].labels).toContain('1.5x Extended Time')

  // AC-1 proxy: the 2.11 engine now reports extended time for the student (global override).
  const effRes = await fetch(`${API_BASE}/api/v1/me/accommodations`, {
    headers: authHeaders(student.access_token),
  })
  expect(effRes.ok).toBeTruthy()
  const eff = (await effRes.json()) as { accommodations: { hasExtendedTime: boolean }[] }
  expect(eff.accommodations.some((a) => a.hasExtendedTime)).toBeTruthy()

  // AC-5: deactivating the profile removes the override.
  const patchRes = await fetch(`${API_BASE}/api/v1/accessibility/profiles/${profile.id}`, {
    method: 'PATCH',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ isActive: false }),
  })
  expect(patchRes.ok).toBeTruthy()

  const effAfter = await fetch(`${API_BASE}/api/v1/me/accommodations`, {
    headers: authHeaders(student.access_token),
  })
  const effAfterData = (await effAfter.json()) as { accommodations: { hasExtendedTime: boolean }[] }
  expect(effAfterData.accommodations.some((a) => a.hasExtendedTime)).toBeFalsy()
})

test('Accessibility intake: instructor cannot read accommodation profiles (AC-4 / security)', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)

  const student = await apiSignup({ email: uniqueEmail('s2'), password: PASSWORD })
  const studentId = await myUserId(student.access_token)
  const createRes = await fetch(`${API_BASE}/api/v1/accessibility/profiles`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ studentId, accommodations: ['extended_time_2x'] }),
  })
  expect(createRes.status).toBe(201)
  const { profile } = (await createRes.json()) as { profile: { id: string } }

  // A non-privileged user (no accommodations:manage permission) is forbidden.
  const other = await apiSignup({ email: uniqueEmail('instructor'), password: PASSWORD })
  const forbidden = await fetch(`${API_BASE}/api/v1/accessibility/profiles/${profile.id}`, {
    headers: authHeaders(other.access_token),
  })
  expect(forbidden.status).toBe(403)

  const forbiddenList = await fetch(`${API_BASE}/api/v1/accessibility/profiles`, {
    headers: authHeaders(other.access_token),
  })
  expect(forbiddenList.status).toBe(403)
})

test('Accessibility intake: instructor notification letter omits disability details (AC-2)', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)

  const student = await apiSignup({ email: uniqueEmail('s3'), password: PASSWORD })
  const studentId = await myUserId(student.access_token)
  const createRes = await fetch(`${API_BASE}/api/v1/accessibility/profiles`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ studentId, accommodations: ['extended_time_1_5x'] }),
  })
  expect(createRes.status).toBe(201)
  const { profile } = (await createRes.json()) as { profile: { id: string } }

  const notifyRes = await fetch(`${API_BASE}/api/v1/accessibility/profiles/${profile.id}/notify-instructors`, {
    method: 'POST',
    headers: authHeaders(adminToken),
  })
  expect(notifyRes.ok).toBeTruthy()
  const notify = (await notifyRes.json()) as { letter: string; notifiedInstructorCount: number }
  expect(notify.letter).toContain('1.5x Extended Time')
  expect(notify.letter.toLowerCase()).not.toContain('diagnosis')
})
