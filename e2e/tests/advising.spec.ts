/**
 * Advising & degree planner (plan 14.14)
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'advising') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'advising') {
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

async function getMeUserId(token: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(token) })
  expect(res.ok).toBeTruthy()
  const data = (await res.json()) as { id?: string }
  if (!data.id) throw new Error('missing user id')
  return data.id
}

async function enableAdvisingFeature(adminToken: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffAdvisingIntegration: true,
      updateMask: ['ffAdvisingIntegration'],
    }),
  })
  expect(res.ok).toBeTruthy()
}

test('Advising: unauthenticated endpoints return 401', async () => {
  const adminToken = await getAdminToken()
  await enableAdvisingFeature(adminToken)

  const paths = [
    '/api/v1/me/degree-progress',
    '/api/v1/me/advising-notes',
    '/api/v1/admin/advising/config',
  ]
  for (const path of paths) {
    const res = await fetch(`${API_BASE}${path}`)
    expect(res.status).toBe(401)
  }
})

test('Advising: admin configures appointment URL and student sees it', async () => {
  const adminToken = await getAdminToken()
  await enableAdvisingFeature(adminToken)

  const apptUrl = 'https://navigate.example.edu/book'
  const configRes = await fetch(`${API_BASE}/api/v1/admin/advising/config`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      appointmentUrl: apptUrl,
      degreeAuditProvider: 'degreeworks',
      degreeAuditBaseUrl: 'https://degreeworks.example.edu/api',
    }),
  })
  expect(configRes.ok).toBeTruthy()

  const { access_token: studentToken } = await apiSignup({
    email: uniqueEmail('student'),
    password: PASSWORD,
    displayName: 'E2E Student',
  })

  const progressRes = await fetch(`${API_BASE}/api/v1/me/degree-progress`, {
    headers: authHeaders(studentToken),
  })
  expect(progressRes.ok).toBeTruthy()
  const progress = (await progressRes.json()) as { appointmentUrl?: string; configured?: boolean }
  expect(progress.appointmentUrl).toBe(apptUrl)
  expect(progress.configured).toBe(true)
})

test('Advising: admin creates note for student and student reads it', async () => {
  const adminToken = await getAdminToken()
  await enableAdvisingFeature(adminToken)

  const { access_token: studentToken } = await apiSignup({
    email: uniqueEmail('student-note'),
    password: PASSWORD,
    displayName: 'E2E Student',
  })
  const studentId = await getMeUserId(studentToken)

  const noteRes = await fetch(`${API_BASE}/api/v1/advisor/students/${studentId}/notes`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ content: 'Register for MATH201 next term.' }),
  })
  expect(noteRes.status).toBe(201)

  const listRes = await fetch(`${API_BASE}/api/v1/me/advising-notes`, {
    headers: authHeaders(studentToken),
  })
  expect(listRes.ok).toBeTruthy()
  const data = (await listRes.json()) as { notes?: { content: string }[] }
  expect(data.notes?.length).toBeGreaterThan(0)
  expect(data.notes?.[0]?.content).toContain('MATH201')
})

test('Advising: student cannot write notes for another student', async () => {
  const adminToken = await getAdminToken()
  await enableAdvisingFeature(adminToken)

  const { access_token: studentAToken } = await apiSignup({
    email: uniqueEmail('student-a'),
    password: PASSWORD,
  })
  const { access_token: studentBToken } = await apiSignup({
    email: uniqueEmail('student-b'),
    password: PASSWORD,
  })
  const studentBId = await getMeUserId(studentBToken)

  const res = await fetch(`${API_BASE}/api/v1/advisor/students/${studentBId}/notes`, {
    method: 'POST',
    headers: authHeaders(studentAToken),
    body: JSON.stringify({ content: 'Should not work' }),
  })
  expect(res.status).toBe(403)
})
