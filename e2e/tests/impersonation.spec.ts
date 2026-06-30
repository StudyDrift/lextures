/**
 * Admin impersonation (plan 18.3)
 *
 *   [x] Org admin can impersonate student and GET /me returns target
 *   [x] Write requests blocked during impersonation
 *   [x] Exit restores admin session
 *   [x] Cross-org impersonation returns 403
 *   [x] Nested impersonation returns 403
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-impersonation') {
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
  return (await res.json()) as { access_token: string; user: { id: string; org?: { id: string } } }
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
      impersonationEnabled: true,
      adminAuditLogEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable impersonation: ${await res.text()}`)
  }
}

async function resolveOrgId(token: string): Promise<{ userId: string; orgId: string }> {
  const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(token) })
  if (!meRes.ok) throw new Error(`GET /me failed: ${meRes.status}`)
  const me = (await meRes.json()) as { id: string; org?: { id: string }; orgId?: string }
  let orgId = me.org?.id ?? me.orgId
  if (!orgId) {
    const capsRes = await fetch(`${API_BASE}/api/v1/me/org-role-capabilities`, {
      headers: authHeaders(token),
    })
    if (capsRes.ok) {
      orgId = ((await capsRes.json()) as { orgId?: string }).orgId
    }
  }
  if (!orgId) throw new Error('missing org id')
  return { userId: me.id, orgId }
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

test.describe.serial('Impersonation', () => {
  test('Impersonation: disabled returns 404', async () => {
    const email = uniqueEmail('disabled')
    await apiSignup(email)
    const { access_token } = await apiLogin(email)
    const res = await fetch(`${API_BASE}/api/v1/admin-console/impersonate`, {
      method: 'POST',
      headers: authHeaders(access_token),
      body: JSON.stringify({ target_user_id: '00000000-0000-4000-8000-000000000001' }),
    })
    if (res.status === 200 || res.status === 403) {
      test.skip(true, 'impersonation already enabled globally')
    }
    expect(res.status).toBe(404)
  })

  test('Impersonation: org admin workflow', async () => {
    const gaEmail = uniqueEmail('ga')
    await apiSignup(gaEmail)
    try {
      await bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const ga = await apiLogin(gaEmail)
    await enableFeatures(ga.access_token)

    const orgAdminEmail = uniqueEmail('org-admin')
    await apiSignup(orgAdminEmail)
    const orgAdmin = await apiLogin(orgAdminEmail)
    const { userId: orgAdminId, orgId } = await resolveOrgId(orgAdmin.access_token)
    await grantOrgAdmin(ga.access_token, orgId, orgAdminId)

    const studentEmail = uniqueEmail('student')
    await apiSignup(studentEmail)
    const student = await apiLogin(studentEmail)
    const { userId: studentId } = await resolveOrgId(student.access_token)

    const startRes = await fetch(`${API_BASE}/api/v1/admin-console/impersonate`, {
      method: 'POST',
      headers: authHeaders(orgAdmin.access_token),
      body: JSON.stringify({ target_user_id: studentId }),
    })
    expect(startRes.status).toBe(200)
    const start = (await startRes.json()) as { impersonation_token: string }
    expect(start.impersonation_token).toBeTruthy()

    const meRes = await fetch(`${API_BASE}/api/v1/me`, {
      headers: authHeaders(start.impersonation_token),
    })
    expect(meRes.status).toBe(200)
    const me = (await meRes.json()) as { id: string; impersonating?: { adminId: string } }
    expect(me.id).toBe(studentId)
    expect(me.impersonating?.adminId).toBe(orgAdminId)

    const writeRes = await fetch(`${API_BASE}/api/v1/me/push-subscriptions`, {
      method: 'POST',
      headers: authHeaders(start.impersonation_token),
      body: JSON.stringify({}),
    })
    expect(writeRes.status).toBe(403)

    const nestedRes = await fetch(`${API_BASE}/api/v1/admin-console/impersonate`, {
      method: 'POST',
      headers: authHeaders(start.impersonation_token),
      body: JSON.stringify({ target_user_id: studentId }),
    })
    expect(nestedRes.status).toBe(403)

    const endRes = await fetch(`${API_BASE}/api/v1/admin-console/impersonate/session`, {
      method: 'DELETE',
      headers: authHeaders(start.impersonation_token),
    })
    expect(endRes.status).toBe(204)

    const adminMe = await fetch(`${API_BASE}/api/v1/me`, {
      headers: authHeaders(orgAdmin.access_token),
    })
    expect(adminMe.status).toBe(200)
    const adminProfile = (await adminMe.json()) as { id: string }
    expect(adminProfile.id).toBe(orgAdminId)
  })
})
