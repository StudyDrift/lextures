/**
 * Seat license management (plan 18.8)
 *
 *   [x] Super admin can patch license max_seats
 *   [x] Org admin overview includes license utilization
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-seats') {
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

async function enableFeatures(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      adminConsoleEnabled: true,
      seatManagementEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable features: ${await res.text()}`)
  }
}

async function grantOrgAdmin(actorToken: string, orgId: string, userId: string) {
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: authHeaders(actorToken),
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
  if (!res.ok) throw new Error(`grant org_admin failed: ${await res.text()}`)
}

test.describe.serial('Seat license management', () => {
  test('SeatManagement: super admin patch and org overview', async () => {
    const gaEmail = uniqueEmail('ga')
    await apiSignup(gaEmail)
    try {
      await bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const ga = await apiLogin(gaEmail)
    await enableFeatures(ga.access_token)

    const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(ga.access_token) })
    const me = (await meRes.json()) as { id: string; org: { id: string } }
    const orgId = me.org.id

    const patchRes = await fetch(`${API_BASE}/api/v1/admin/licenses/${orgId}`, {
      method: 'PATCH',
      headers: authHeaders(ga.access_token),
      body: JSON.stringify({ maxSeats: 500, tier: 'enterprise' }),
    })
    expect(patchRes.status).toBe(200)
    const patched = (await patchRes.json()) as { maxSeats: number; tier: string }
    expect(patched.maxSeats).toBe(500)
    expect(patched.tier).toBe('enterprise')

    const orgAdminEmail = uniqueEmail('org-admin')
    await apiSignup(orgAdminEmail)
    const orgAdmin = await apiLogin(orgAdminEmail)
    const orgAdminMe = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(orgAdmin.access_token) })
    const orgAdminBody = (await orgAdminMe.json()) as { id: string }
    await grantOrgAdmin(ga.access_token, orgId, orgAdminBody.id)

    const overviewRes = await fetch(`${API_BASE}/api/v1/admin-console/overview`, {
      headers: authHeaders(orgAdmin.access_token),
    })
    expect(overviewRes.status).toBe(200)
    const overview = (await overviewRes.json()) as {
      license?: { maxSeats: number; usedSeats: number; tier: string }
    }
    expect(overview.license?.maxSeats).toBe(500)
    expect(overview.license?.tier).toBe('enterprise')
    expect(typeof overview.license?.usedSeats).toBe('number')
  })
})
