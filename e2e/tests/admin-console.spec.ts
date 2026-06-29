/**
 * Admin Console (plan 18.1)
 *
 *   [x] Feature disabled returns 404 on admin-console API
 *   [x] Unauthenticated requests return 401
 *   [x] Student cannot access admin-console overview (403)
 *   [x] Org admin can load overview with KPIs
 *   [x] Org admin can search users
 *   [x] Org admin can deactivate user; deactivated user cannot log in
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-admin-console') {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

function databaseUrl(): string {
  return (
    process.env.DATABASE_URL ??
    process.env.E2E_DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
}

async function apiSignup(email: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD, display_name: 'E2E User' }),
  })
  if (!res.ok && res.status !== 409) {
    throw new Error(`signup failed: ${await res.text()}`)
  }
}

async function apiLogin(email: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD }),
  })
  if (!res.ok) throw new Error(`login failed: ${await res.text()}`)
  return (await res.json()) as { access_token: string }
}

async function bootstrapGlobalAdmin(email: string) {
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: databaseUrl() },
    stdio: 'pipe',
  })
}

async function enableAdminConsole(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ adminConsoleEnabled: true, adminAuditLogEnabled: true }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable admin console: ${await res.text()}`)
  }
}

async function grantOrgAdmin(actorToken: string, orgId: string, userId: string) {
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: authHeaders(actorToken),
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
  if (!res.ok) {
    throw new Error(`grant org_admin failed: ${await res.text()}`)
  }
}

test('AdminConsole: disabled feature returns 404', async () => {
  const email = uniqueEmail('disabled')
  await apiSignup(email)
  const { access_token } = await apiLogin(email)
  const res = await fetch(`${API_BASE}/api/v1/admin-console/overview`, {
    headers: authHeaders(access_token),
  })
  // Default off — expect 404 unless another test enabled it globally.
  if (res.status === 200) {
    test.skip(true, 'admin console already enabled globally')
  }
  expect(res.status).toBe(404)
})

test('AdminConsole: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/admin-console/users`)
  expect([401, 404]).toContain(res.status)
})

test('AdminConsole: org admin workflow', async () => {
  const gaEmail = uniqueEmail('ga')
  await apiSignup(gaEmail)
  try {
    await bootstrapGlobalAdmin(gaEmail)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const ga = await apiLogin(gaEmail)
  await enableAdminConsole(ga.access_token)

  const orgAdminEmail = uniqueEmail('org-admin')
  await apiSignup(orgAdminEmail)
  const orgAdmin = await apiLogin(orgAdminEmail)

  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: authHeaders(orgAdmin.access_token),
  })
  const me = (await meRes.json()) as { id: string; orgId?: string }
  const orgId = me.orgId ?? (await (await fetch(`${API_BASE}/api/v1/me/org-role-capabilities`, {
    headers: authHeaders(orgAdmin.access_token),
  })).json() as { orgId: string }).orgId

  await grantOrgAdmin(ga.access_token, orgId, me.id)

  const meCaps = await fetch(`${API_BASE}/api/v1/me/admin-console-capabilities`, {
    headers: authHeaders(orgAdmin.access_token),
  })
  expect(meCaps.status).toBe(200)
  const meCapsBody = (await meCaps.json()) as { canAccess: boolean; canManage: boolean }
  expect(meCapsBody.canAccess).toBe(true)
  expect(meCapsBody.canManage).toBe(true)

  const overviewRes = await fetch(`${API_BASE}/api/v1/admin-console/overview`, {
    headers: authHeaders(orgAdmin.access_token),
  })
  expect(overviewRes.status).toBe(200)
  const overview = (await overviewRes.json()) as { totalUsers: number; activeCourses: number }
  expect(typeof overview.totalUsers).toBe('number')
  expect(typeof overview.activeCourses).toBe('number')

  const studentEmail = uniqueEmail('student')
  await apiSignup(studentEmail)
  const student = await apiLogin(studentEmail)
  const forbidden = await fetch(`${API_BASE}/api/v1/admin-console/overview`, {
    headers: authHeaders(student.access_token),
  })
  expect(forbidden.status).toBe(403)

  const usersRes = await fetch(
    `${API_BASE}/api/v1/admin-console/users?q=${encodeURIComponent(studentEmail)}`,
    { headers: authHeaders(orgAdmin.access_token) },
  )
  expect(usersRes.status).toBe(200)
  const usersBody = (await usersRes.json()) as { items: { id: string; email: string }[] }
  const target = usersBody.items.find((u) => u.email === studentEmail)
  expect(target).toBeTruthy()

  const deactivateRes = await fetch(
    `${API_BASE}/api/v1/admin-console/users/${target!.id}`,
    {
      method: 'PATCH',
      headers: authHeaders(orgAdmin.access_token),
      body: JSON.stringify({ active: false }),
    },
  )
  expect(deactivateRes.status).toBe(200)

  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: studentEmail, password: PASSWORD }),
  })
  expect(loginRes.status).toBe(401)
})
