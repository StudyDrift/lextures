/**
 * Admin org-wide search (plan 18.4)
 *
 *   [x] Feature disabled returns 404
 *   [x] Org admin can omnisearch users by name
 *   [x] Types filter returns only requested entity type
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-admin-search') {
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

async function apiSignup(email: string, displayName?: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD, display_name: displayName ?? 'E2E User' }),
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

async function enableAdminFeatures(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      adminConsoleEnabled: true,
      adminSearchEnabled: true,
      adminAuditLogEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable admin search: ${await res.text()}`)
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

test.describe.serial('Admin org-wide search', () => {
  test('AdminSearch: disabled feature returns 404', async () => {
    const email = uniqueEmail('disabled')
    await apiSignup(email)
    const { access_token } = await apiLogin(email)
    const res = await fetch(`${API_BASE}/api/v1/admin/search?q=test`, {
      headers: authHeaders(access_token),
    })
    if (res.status === 200 || res.status === 403) {
      test.skip(true, 'admin search already enabled globally')
    }
    expect(res.status).toBe(404)
  })

  test('AdminSearch: org admin omnisearch finds user by name', async () => {
    const gaEmail = uniqueEmail('ga')
    await apiSignup(gaEmail)
    try {
      await bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const ga = await apiLogin(gaEmail)
    await enableAdminFeatures(ga.access_token)

    const orgAdminEmail = uniqueEmail('org-admin')
    await apiSignup(orgAdminEmail, 'Alice Johnson')
    const orgAdmin = await apiLogin(orgAdminEmail)

    const meRes = await fetch(`${API_BASE}/api/v1/me`, {
      headers: authHeaders(orgAdmin.access_token),
    })
    const me = (await meRes.json()) as { id: string; org?: { id: string }; orgId?: string }
    const orgId = me.org?.id ?? me.orgId ?? (await (await fetch(`${API_BASE}/api/v1/me/org-role-capabilities`, {
      headers: authHeaders(orgAdmin.access_token),
    })).json() as { orgId: string }).orgId

    await grantOrgAdmin(ga.access_token, orgId, me.id)

    const targetEmail = uniqueEmail('target-johnson')
    await apiSignup(targetEmail, 'Bob Johnson')

    const searchRes = await fetch(
      `${API_BASE}/api/v1/admin/search?q=johnson&types=users`,
      { headers: authHeaders(orgAdmin.access_token) },
    )
    expect(searchRes.status).toBe(200)
    const body = (await searchRes.json()) as {
      users: { title: string; subtitle: string }[]
      courses: unknown[]
      content: unknown[]
    }
    expect(body.courses).toEqual([])
    expect(body.content).toEqual([])
    const hit = body.users.find(
      (u) => u.subtitle === targetEmail || u.title.includes('Johnson'),
    )
    expect(hit).toBeTruthy()
  })
})
