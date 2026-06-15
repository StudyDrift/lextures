/**
 * Research / IRB consent flows (plan 14.15)
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'consent') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'consent') {
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
      ffResearchConsent: true,
      updateMask: ['ffResearchConsent'],
    }),
  })
  expect(res.ok).toBeTruthy()
}

async function createActiveStudy(adminToken: string, title: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/admin/consent-studies`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      title,
      irbProtocol: 'IRB-2026-E2E',
      consentText: '## Consent\nYou may participate in this study.',
      dataUseDescription: 'Learning analytics only.',
      status: 'active',
    }),
  })
  expect(res.status).toBe(201)
  const data = (await res.json()) as { study?: { id: string } }
  if (!data.study?.id) throw new Error('missing study id')
  return data.study.id
}

test('Research consent: unauthenticated endpoints return 401', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)

  const paths = [
    '/api/v1/me/consent-studies',
    '/api/v1/me/consent-studies/history',
    '/api/v1/admin/consent-studies',
  ]
  for (const path of paths) {
    const res = await fetch(`${API_BASE}${path}`)
    expect(res.status).toBe(401)
  }
})

test('Research consent: export gate includes only consenting students (AC-2, AC-3)', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)
  const studyId = await createActiveStudy(adminToken, uid('study'))

  const { access_token: granter } = await apiSignup({ email: uniqueEmail('granter'), password: PASSWORD })
  const { access_token: decliner } = await apiSignup({ email: uniqueEmail('decliner'), password: PASSWORD })
  const { access_token: withdrawer } = await apiSignup({ email: uniqueEmail('withdrawer'), password: PASSWORD })

  async function respond(token: string, decision: string) {
    const res = await fetch(`${API_BASE}/api/v1/me/consent-studies/${studyId}/respond`, {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ decision }),
    })
    expect(res.status).toBe(201)
  }

  await respond(granter, 'granted')
  await respond(decliner, 'declined')
  await respond(withdrawer, 'granted')

  // Before withdrawal: two consenting participants.
  let exportRes = await fetch(`${API_BASE}/api/v1/admin/consent-studies/${studyId}/export`, {
    headers: authHeaders(adminToken),
  })
  expect(exportRes.ok).toBeTruthy()
  let exportData = (await exportRes.json()) as { count: number }
  expect(exportData.count).toBe(2)

  // AC-3: withdrawal removes the student from future exports.
  await respond(withdrawer, 'withdrawn')
  exportRes = await fetch(`${API_BASE}/api/v1/admin/consent-studies/${studyId}/export`, {
    headers: authHeaders(adminToken),
  })
  exportData = (await exportRes.json()) as { count: number }
  expect(exportData.count).toBe(1)

  // AC-4: the audit log records every decision with a timestamp.
  const recordsRes = await fetch(`${API_BASE}/api/v1/admin/consent-studies/${studyId}/records`, {
    headers: authHeaders(adminToken),
  })
  expect(recordsRes.ok).toBeTruthy()
  const recordsData = (await recordsRes.json()) as { records: { decision: string; createdAt: string }[] }
  // granted + declined + granted + withdrawn = 4 ledger rows.
  expect(recordsData.records.length).toBe(4)
  expect(recordsData.records.every((r) => Boolean(r.createdAt))).toBeTruthy()
})

test('Research consent: student history reflects their decision', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)
  const studyId = await createActiveStudy(adminToken, uid('history-study'))

  const { access_token: student } = await apiSignup({ email: uniqueEmail('hist'), password: PASSWORD })
  const respondRes = await fetch(`${API_BASE}/api/v1/me/consent-studies/${studyId}/respond`, {
    method: 'POST',
    headers: authHeaders(student),
    body: JSON.stringify({ decision: 'granted' }),
  })
  expect(respondRes.status).toBe(201)

  const historyRes = await fetch(`${API_BASE}/api/v1/me/consent-studies/history`, {
    headers: authHeaders(student),
  })
  expect(historyRes.ok).toBeTruthy()
  const data = (await historyRes.json()) as { history: { studyId: string; decision: string }[] }
  const entry = data.history.find((h) => h.studyId === studyId)
  expect(entry?.decision).toBe('granted')
})

test('Research consent: a non-privileged student cannot create a study (AC-5 / security)', async () => {
  const adminToken = await getAdminToken()
  await enableFeature(adminToken)

  const { access_token: student } = await apiSignup({ email: uniqueEmail('nope'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/admin/consent-studies`, {
    method: 'POST',
    headers: authHeaders(student),
    body: JSON.stringify({
      title: 'Unauthorized',
      irbProtocol: 'X',
      consentText: 'x',
      dataUseDescription: 'x',
    }),
  })
  expect(res.status).toBe(403)
})
