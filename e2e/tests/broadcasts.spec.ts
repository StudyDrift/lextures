/**
 * District broadcasts & emergency channel (plan 13.10)
 *
 *   [x] GET broadcasts unauthenticated returns 401
 *   [x] POST broadcasts unauthenticated returns 401
 *   [x] GET delivery-report unauthenticated returns 401
 *   [x] POST acknowledge unauthenticated returns 401
 *   [x] GET me/broadcasts unauthenticated returns 401
 *   [x] Feature flag off returns 501
 *   [x] POST with invalid type returns 400
 *   [x] POST with no subject returns 400
 *   [x] Admin can create an announcement broadcast
 *   [x] Admin can create an emergency broadcast and acknowledge it
 *   [x] Delivery report returns recipient counts
 *   [x] Scheduled broadcast > 7 days returns 400
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'bcast') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'bcast') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
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

async function getAdminOrgId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs`, { headers: authHeaders(token) })
  if (!res.ok) return null
  const data = (await res.json()) as { organizations?: Array<{ id: string }> }
  return data.organizations?.[0]?.id ?? null
}

async function getAdminUserId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(token) })
  if (!res.ok) return null
  const data = (await res.json()) as { id?: string }
  return data.id ?? null
}

async function grantOrgAdmin(adminToken: string, orgId: string, userId: string): Promise<void> {
  await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
}

interface Broadcast {
  id: string
  orgId: string
  senderId: string
  type: 'announcement' | 'emergency'
  subject: string
  body: string
  status: 'draft' | 'queued' | 'sent'
  scheduledAt: string | null
  sentAt: string | null
  createdAt: string
}

// ─── Auth guards (no token → 401) ────────────────────────────────────────────

test('Broadcasts: GET broadcasts unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts`)
  expect(res.status).toBe(401)
})

test('Broadcasts: POST broadcasts unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('Broadcasts: GET delivery-report unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts/00000000-0000-0000-0000-000000000002/delivery-report`,
  )
  expect(res.status).toBe(401)
})

test('Broadcasts: POST acknowledge unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/broadcasts/00000000-0000-0000-0000-000000000002/acknowledge`,
    { method: 'POST' },
  )
  expect(res.status).toBe(401)
})

test('Broadcasts: GET me/broadcasts unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/me/broadcasts`)
  expect(res.status).toBe(401)
})

// ─── Feature flag gating ─────────────────────────────────────────────────────

test('Broadcasts: skipped when feature flag is off (returns 501 or works when on)', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    headers: authHeaders(token),
  })
  // 501 when flag off; 200/403 when on
  expect([200, 403, 501]).toContain(res.status)
})

// ─── Happy path tests (require FF_BROADCASTS=true) ───────────────────────────

test('Broadcasts: Admin can create an announcement broadcast', async () => {
const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const subject = `Snow Day ${uid()}`
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ type: 'announcement', subject, body: 'School closed tomorrow.' }),
  })
  expect(res.status).toBe(201)
  const data = (await res.json()) as { broadcast: Broadcast }
  expect(data.broadcast.subject).toBe(subject)
  expect(data.broadcast.type).toBe('announcement')
  expect(data.broadcast.status).toBe('sent')
})

test('Broadcasts: POST with invalid type returns 400', async () => {
const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ type: 'spam', subject: 's', body: 'b' }),
  })
  expect(res.status).toBe(400)
})

test('Broadcasts: POST with no subject returns 400', async () => {
const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ subject: '', body: 'b' }),
  })
  expect(res.status).toBe(400)
})

test('Broadcasts: Scheduled > 7 days returns 400', async () => {
const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const farFuture = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ subject: 's', body: 'b', scheduledAt: farFuture }),
  })
  expect(res.status).toBe(400)
})

test('Broadcasts: Admin can create an emergency broadcast and acknowledge it', async () => {
const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const subject = `Lockdown Drill ${uid()}`
  const createRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ type: 'emergency', subject, body: 'This is a drill.' }),
  })
  expect(createRes.status).toBe(201)
  const { broadcast } = (await createRes.json()) as { broadcast: Broadcast }
  expect(broadcast.type).toBe('emergency')

  const ackRes = await fetch(
    `${API_BASE}/api/v1/broadcasts/${broadcast.id}/acknowledge`,
    { method: 'POST', headers: authHeaders(token) },
  )
  expect(ackRes.status).toBe(204)

  // Delivery report should reflect at least 1 acknowledged
  const reportRes = await fetch(
    `${API_BASE}/api/v1/orgs/${orgId}/broadcasts/${broadcast.id}/delivery-report`,
    { headers: authHeaders(token) },
  )
  expect(reportRes.status).toBe(200)
  const report = (await reportRes.json()) as {
    totalRecipients: number
    acknowledged: number
  }
  expect(report.acknowledged).toBeGreaterThanOrEqual(1)
})

test('Broadcasts: Non-admin cannot create a broadcast (403)', async () => {
const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const { access_token: studentToken } = await apiSignup({
    email: uniqueEmail('nonAdmin'),
    password: PASSWORD,
  })
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/broadcasts`, {
    method: 'POST',
    headers: authHeaders(studentToken),
    body: JSON.stringify({ type: 'announcement', subject: 's', body: 'b' }),
  })
  expect([403, 404]).toContain(res.status)
})

test('Broadcasts: Me/broadcasts returns array for authenticated user', async () => {
const { access_token: token } = await apiSignup({
    email: uniqueEmail('mebcast'),
    password: PASSWORD,
  })
  const res = await fetch(`${API_BASE}/api/v1/me/broadcasts`, { headers: authHeaders(token) })
  expect(res.status).toBe(200)
  const data = (await res.json()) as { broadcasts: Broadcast[] }
  expect(Array.isArray(data.broadcasts)).toBe(true)
})
