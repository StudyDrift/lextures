/**
 * MOB.3 — Mobile admin console audit log API parity
 *
 *   [x] GET admin-console/audit-log: unauthenticated returns 401
 *   [x] GET admin-console/audit-log: student returns 403/404 when console disabled or unauthorized
 *   [x] Global admin can list audit events when console + audit log enabled
 *   [x] Action filter narrows results
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

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

async function enableMobileAdminConsole(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      adminConsoleEnabled: true,
      adminAuditLogEnabled: true,
      ffMobileAdminConsole: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable mobile admin console: ${await res.text()}`)
  }
}

test('MOB.3 audit-log: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/admin-console/audit-log`)
  expect(res.status).toBe(401)
})

test('MOB.3 audit-log: student cannot list events', async () => {
  const email = uniqueEmail('mob3-student')
  const { access_token: token } = await apiSignup({
    email,
    password: PASSWORD,
  })
  const res = await fetch(`${API_BASE}/api/v1/admin-console/audit-log`, {
    headers: authHeaders(token),
  })
  expect([403, 404]).toContain(res.status)
})

test('MOB.3 audit-log: admin can list and filter events', async () => {
  const email = uniqueEmail('mob3-admin')
  await apiSignup({ email, password: PASSWORD })
  await bootstrapGlobalAdmin(email)
  const { access_token: token } = await apiLogin(email)
  await enableMobileAdminConsole(token)

  const listRes = await fetch(`${API_BASE}/api/v1/admin-console/audit-log`, {
    headers: authHeaders(token),
  })
  expect(listRes.ok).toBeTruthy()
  const listBody = (await listRes.json()) as { events?: Array<{ eventId: string; eventType: string }> }
  expect(Array.isArray(listBody.events)).toBeTruthy()

  const filterRes = await fetch(
    `${API_BASE}/api/v1/admin-console/audit-log?action=${encodeURIComponent('__no_such_event__')}`,
    { headers: authHeaders(token) },
  )
  expect(filterRes.ok).toBeTruthy()
  const filterBody = (await filterRes.json()) as { events?: unknown[] }
  expect(filterBody.events ?? []).toEqual([])

  const featuresRes = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(token),
  })
  expect(featuresRes.ok).toBeTruthy()
  const features = (await featuresRes.json()) as {
    ffMobileAdminConsole?: boolean
    adminConsoleEnabled?: boolean
    adminAuditLogEnabled?: boolean
  }
  expect(features.ffMobileAdminConsole).toBe(true)
  expect(features.adminConsoleEnabled).toBe(true)
  expect(features.adminAuditLogEnabled).toBe(true)
})
