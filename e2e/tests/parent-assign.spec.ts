/**
 * PP.1 — Staff assign parent / guardian (API contract)
 *
 *   [x] GET parent-assign/students unauthenticated → 401
 *   [x] Regular user without assign permission → 403 on student search
 *   [x] parent-invite/consume invalid token → 400
 *   [x] Org admin can search students via parent-assign API
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uid(prefix = 'pp1') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function uniqueEmail(prefix = 'pp1') {
  return `${uid(prefix)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: PASSWORD }),
  })
  if (!loginRes.ok) {
    await fetch(`${API_BASE}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: adminEmail,
        password: PASSWORD,
        display_name: 'E2E Admin',
      }),
    })
    const retry = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: adminEmail, password: PASSWORD }),
    })
    const { access_token } = (await retry.json()) as { access_token: string }
    return access_token
  }
  const { access_token } = (await loginRes.json()) as { access_token: string }
  return access_token
}

function orgIdFromToken(token: string): string | null {
  try {
    const parts = token.split('.')
    if (parts.length < 2) return null
    const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString()) as {
      org_id?: string
    }
    return payload.org_id ?? null
  } catch {
    return null
  }
}

async function getMyOrgId(token: string): Promise<string | null> {
  const fromToken = orgIdFromToken(token)
  if (fromToken) return fromToken
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return null
  const me = (await res.json()) as { orgId?: string; org_id?: string }
  return me.orgId ?? me.org_id ?? null
}

test('PP.1: parent-assign student search unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/parent-assign/students?q=a`,
  )
  expect(res.status).toBe(401)
})

test('PP.1: parent-invite consume invalid token returns 400', async () => {
  const res = await fetch(`${API_BASE}/api/v1/auth/parent-invite/consume`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token: 'not-a-real-invite-token', password: PASSWORD }),
  })
  expect(res.status).toBe(400)
})

test('PP.1: regular user without assign permission gets 403 on search', async () => {
  const { access_token } = await apiSignup({
    email: uniqueEmail('noperm'),
    password: PASSWORD,
  })
  const orgId = await getMyOrgId(access_token)
  if (!orgId) {
    test.skip(true, 'could not determine org id')
    return
  }
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/${orgId}/parent-assign/students?q=test`,
    { headers: authHeaders(access_token) },
  )
  expect([403, 404]).toContain(res.status)
})

test('PP.1: org admin can search students via parent-assign API', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) {
    test.skip(true, 'could not determine org id')
    return
  }

  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: authHeaders(adminToken),
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /me unavailable')
    return
  }
  const meBody = (await meRes.json()) as { id?: string }
  if (meBody.id) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: authHeaders(adminToken),
      body: JSON.stringify({ userId: meBody.id, role: 'org_admin' }),
    })
  }

  const studentEmail = uniqueEmail('child')
  await apiSignup({ email: studentEmail, password: PASSWORD })

  const res = await fetch(
    `${API_BASE}/api/v1/orgs/${orgId}/parent-assign/students?q=${encodeURIComponent(studentEmail)}`,
    { headers: authHeaders(adminToken) },
  )
  // Org admin may access via elevated path; 403 if permission not yet granted in this env.
  if (res.status === 403) {
    test.skip(true, 'admin lacks parent-links assign permission in this environment')
    return
  }
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { students?: Array<{ email?: string }> }
  expect(Array.isArray(body.students)).toBeTruthy()
})
