/**
 * SIS Integration (plan 13.7)
 *
 *   [x] GET sis/connections unauthenticated returns 401
 *   [x] POST sis/connections unauthenticated returns 401
 *   [x] PATCH sis/connections/:id unauthenticated returns 401
 *   [x] POST sis/connections/:id/sync unauthenticated returns 401
 *   [x] GET sis/sync-logs unauthenticated returns 401
 *   [x] POST sis/grade-passback unauthenticated returns 401
 *   [x] Admin can create a SIS connection (PowerSchool)
 *   [x] Admin can create a SIS connection (Infinite Campus)
 *   [x] Admin can create a SIS connection (Skyward)
 *   [x] Admin can create a SIS connection (Aeries)
 *   [x] Create SIS connection with invalid vendor returns 400
 *   [x] Create SIS connection missing baseUrl returns 400
 *   [x] Create SIS connection missing clientIdRef returns 400
 *   [x] Create SIS connection defaults syncSchedule to nightly cron
 *   [x] Admin can list SIS connections for org
 *   [x] Admin can update (PATCH) a SIS connection
 *   [x] Admin can trigger a manual sync
 *   [x] Manual sync returns a log entry with a status
 *   [x] Admin can view sync logs
 *   [x] Sync log shows connection_id and started_at
 *   [x] Admin can trigger grade passback
 *   [x] Grade passback with missing connectionId returns 400
 *   [x] Grade passback with missing gradingPeriod returns 400
 *   [x] Non-admin cannot access SIS endpoints (403)
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'sis') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'sis') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guard checks (no token → 401)
// ─────────────────────────────────────────────────────────────────────────────

test('SIS: GET sis/connections unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections`,
  )
  expect(res.status).toBe(401)
})

test('SIS: POST sis/connections unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('SIS: PATCH sis/connections/:id unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002`,
    { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('SIS: POST sis/connections/:id/sync unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002/sync`,
    { method: 'POST' },
  )
  expect(res.status).toBe(401)
})

test('SIS: GET sis/sync-logs unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/sync-logs`,
  )
  expect(res.status).toBe(401)
})

test('SIS: POST sis/grade-passback unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/grade-passback`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

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
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return null
  const data = (await res.json()) as { organizations?: Array<{ id: string }> }
  return data.organizations?.[0]?.id ?? null
}

async function getAdminUserId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: authHeaders(token),
  })
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

interface SISConnection {
  id: string
  orgId: string
  vendor: string
  baseUrl: string
  clientIdRef: string
  clientSecretRef: string
  syncSchedule: string
  syncMode: string
  active: boolean
  lastSyncAt: string | null
  createdAt: string
}

async function createSISConnection(
  token: string,
  orgId: string,
  payload: {
    vendor: string
    baseUrl?: string
    clientIdRef?: string
    clientSecretRef?: string
    syncSchedule?: string
    syncMode?: string
  },
): Promise<{ status: number; body: unknown }> {
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({
      vendor: payload.vendor,
      baseUrl: payload.baseUrl ?? 'https://ps.district.k12.example.com',
      clientIdRef: payload.clientIdRef ?? 'secrets/sis-client-id',
      clientSecretRef: payload.clientSecretRef ?? 'secrets/sis-client-secret',
      syncSchedule: payload.syncSchedule,
      syncMode: payload.syncMode,
    }),
  })
  return { status: res.status, body: await res.json() }
}

// ─────────────────────────────────────────────────────────────────────────────
// Connection CRUD
// ─────────────────────────────────────────────────────────────────────────────

test('SIS: Admin can create a SIS connection (PowerSchool)', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, body } = await createSISConnection(token, orgId, { vendor: 'powerschool' })
  expect(status).toBe(201)
  const conn = (body as { connection: SISConnection }).connection
  expect(conn.vendor).toBe('powerschool')
  expect(conn.syncSchedule).toBe('0 2 * * *')
  expect(conn.syncMode).toBe('incremental')
  expect(conn.active).toBe(true)
  expect(conn.id).toBeTruthy()
})

test('SIS: Admin can create a SIS connection (Infinite Campus)', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, body } = await createSISConnection(token, orgId, { vendor: 'infinite_campus' })
  expect(status).toBe(201)
  const conn = (body as { connection: SISConnection }).connection
  expect(conn.vendor).toBe('infinite_campus')
})

test('SIS: Admin can create a SIS connection (Skyward)', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, body } = await createSISConnection(token, orgId, { vendor: 'skyward' })
  expect(status).toBe(201)
  const conn = (body as { connection: SISConnection }).connection
  expect(conn.vendor).toBe('skyward')
})

test('SIS: Admin can create a SIS connection (Aeries)', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, body } = await createSISConnection(token, orgId, { vendor: 'aeries' })
  expect(status).toBe(201)
  const conn = (body as { connection: SISConnection }).connection
  expect(conn.vendor).toBe('aeries')
})

test('SIS: Create SIS connection with invalid vendor returns 400', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status } = await createSISConnection(token, orgId, { vendor: 'canvas' })
  expect(status).toBe(400)
})

test('SIS: Create SIS connection missing baseUrl returns 400', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({
      vendor: 'powerschool',
      baseUrl: '',
      clientIdRef: 'ref/id',
      clientSecretRef: 'ref/secret',
    }),
  })
  expect(res.status).toBe(400)
})

test('SIS: Create SIS connection missing clientIdRef returns 400', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({
      vendor: 'powerschool',
      baseUrl: 'https://ps.example.com',
      clientIdRef: '',
      clientSecretRef: 'ref/secret',
    }),
  })
  expect(res.status).toBe(400)
})

test('SIS: Create SIS connection defaults syncSchedule to nightly cron', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, body } = await createSISConnection(token, orgId, { vendor: 'powerschool' })
  expect(status).toBe(201)
  const conn = (body as { connection: SISConnection }).connection
  expect(conn.syncSchedule).toBe('0 2 * * *')
})

test('SIS: Admin can list SIS connections for org', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  await createSISConnection(token, orgId, { vendor: 'powerschool' })

  const listRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections`, {
    headers: authHeaders(token),
  })
  expect(listRes.status).toBe(200)
  const data = (await listRes.json()) as { connections: SISConnection[] }
  expect(Array.isArray(data.connections)).toBe(true)
  expect(data.connections.length).toBeGreaterThan(0)
  const conn = data.connections[0]
  expect(conn.id).toBeTruthy()
  expect(conn.vendor).toBeTruthy()
})

test('SIS: Admin can update (PATCH) a SIS connection', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'infinite_campus' })
  const connId = (created as { connection: SISConnection }).connection.id

  const patchRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections/${connId}`,
    {
      method: 'PATCH',
      headers: authHeaders(token),
      body: JSON.stringify({ syncMode: 'full', active: false }),
    },
  )
  expect(patchRes.status).toBe(200)
  const updated = (await patchRes.json()) as { connection: SISConnection }
  expect(updated.connection.syncMode).toBe('full')
  expect(updated.connection.active).toBe(false)
})

// ─────────────────────────────────────────────────────────────────────────────
// Sync trigger
// ─────────────────────────────────────────────────────────────────────────────

test('SIS: Admin can trigger a manual sync', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'powerschool' })
  const connId = (created as { connection: SISConnection }).connection.id

  const syncRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections/${connId}/sync`,
    { method: 'POST', headers: authHeaders(token) },
  )
  expect(syncRes.status).toBe(200)
})

test('SIS: Manual sync returns a log entry with a status', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'skyward' })
  const connId = (created as { connection: SISConnection }).connection.id

  const syncRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections/${connId}/sync`,
    { method: 'POST', headers: authHeaders(token) },
  )
  expect(syncRes.status).toBe(200)
  const result = (await syncRes.json()) as {
    logId: string
    status: string
    summary: Record<string, number>
  }
  expect(result.logId).toBeTruthy()
  expect(['success', 'partial', 'failed']).toContain(result.status)
  expect(typeof result.summary).toBe('object')
})

test('SIS: Admin can view sync logs', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'aeries' })
  const connId = (created as { connection: SISConnection }).connection.id

  await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections/${connId}/sync`,
    { method: 'POST', headers: authHeaders(token) },
  )

  const logsRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/sync-logs`,
    { headers: authHeaders(token) },
  )
  expect(logsRes.status).toBe(200)
  const data = (await logsRes.json()) as { logs: Array<{ id: string; connectionId: string; status: string; startedAt: string }> }
  expect(Array.isArray(data.logs)).toBe(true)
})

test('SIS: Sync log shows connection_id and started_at', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'powerschool' })
  const connId = (created as { connection: SISConnection }).connection.id

  const syncResult = (await (
    await fetch(
      `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections/${connId}/sync`,
      { method: 'POST', headers: authHeaders(token) },
    )
  ).json()) as { logId: string }

  const logsRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/sync-logs`,
    { headers: authHeaders(token) },
  )
  const data = (await logsRes.json()) as {
    logs: Array<{ id: string; connectionId: string; status: string; startedAt: string }>
  }
  const log = data.logs.find((l) => l.id === syncResult.logId)
  expect(log).toBeTruthy()
  expect(log?.connectionId).toBe(connId)
  expect(log?.startedAt).toBeTruthy()
  expect(['success', 'partial', 'failed']).toContain(log?.status)
})

// ─────────────────────────────────────────────────────────────────────────────
// Grade passback
// ─────────────────────────────────────────────────────────────────────────────

test('SIS: Admin can trigger grade passback', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'infinite_campus' })
  const connId = (created as { connection: SISConnection }).connection.id

  const pbRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/grade-passback`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ connectionId: connId, gradingPeriod: 'Q1-2026' }),
    },
  )
  expect(pbRes.status).toBe(200)
  const result = (await pbRes.json()) as {
    logId: string
    status: string
    gradingPeriod: string
    recordsSent: number
  }
  expect(result.logId).toBeTruthy()
  expect(result.status).toBe('success')
  expect(result.gradingPeriod).toBe('Q1-2026')
  expect(typeof result.recordsSent).toBe('number')
})

test('SIS: Grade passback with missing connectionId returns 400', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const pbRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/grade-passback`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ gradingPeriod: 'Q1-2026' }),
    },
  )
  expect(pbRes.status).toBe(400)
})

test('SIS: Grade passback with missing gradingPeriod returns 400', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { body: created } = await createSISConnection(token, orgId, { vendor: 'powerschool' })
  const connId = (created as { connection: SISConnection }).connection.id

  const pbRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sis/grade-passback`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ connectionId: connId }),
    },
  )
  expect(pbRes.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// RBAC: non-admin is forbidden
// ─────────────────────────────────────────────────────────────────────────────

test('SIS: Non-admin cannot access SIS endpoints (403)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const student = await apiSignup({ email: uniqueEmail('student'), password: PASSWORD })
  const studentToken = student.access_token

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/sis/connections`, {
    headers: authHeaders(studentToken),
  })
  expect([403, 404]).toContain(res.status)
})
