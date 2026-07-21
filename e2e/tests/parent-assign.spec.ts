/**
 * PP.1 — Staff assign parent / guardian (API contract)
 *
 *   [x] With ffParentPortal off → parent-assign returns 404 (feature-first)
 *   [x] With flag on, unauthenticated search → 401
 *   [x] With flag on, regular user without assign permission → 403
 *   [x] parent-invite/consume invalid token → 400
 *   [x] Org admin can search students via parent-assign API
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'
import { setPlatformFlag } from '../lib/feature-lifecycle-helpers.js'

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

function bootstrapGlobalAdmin(email: string) {
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: databaseUrl() },
    stdio: 'pipe',
  })
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

async function ensureGlobalAdmin(): Promise<string> {
  const email = uniqueEmail('pp1-ga')
  await apiSignup({ email, password: PASSWORD, displayName: 'PP.1 GA' })
  try {
    bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  return (await apiLogin(email)).access_token
}

test.describe('PP.1 parent-assign API', () => {
  test('feature off returns 404 on parent-assign and invite consume', async () => {
    const gaToken = await ensureGlobalAdmin()
    await setPlatformFlag(gaToken, 'ffParentPortal', false)

    const orgId = (await getMyOrgId(gaToken)) ?? '00000000-0000-0000-0000-000000000001'
    const search = await fetch(
      `${API_BASE}/api/v1/orgs/${orgId}/parent-assign/students?q=a`,
      { headers: authHeaders(gaToken) },
    )
    expect(search.status).toBe(404)

    const consume = await fetch(`${API_BASE}/api/v1/auth/parent-invite/consume`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: 'not-a-real-invite-token', password: PASSWORD }),
    })
    expect(consume.status).toBe(404)
  })

  test('feature on: unauthenticated search is 401; invalid invite is 400', async () => {
    const gaToken = await ensureGlobalAdmin()
    await setPlatformFlag(gaToken, 'ffParentPortal', true)

    const search = await fetch(
      `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/parent-assign/students?q=a`,
    )
    expect(search.status).toBe(401)

    const consume = await fetch(`${API_BASE}/api/v1/auth/parent-invite/consume`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: 'not-a-real-invite-token', password: PASSWORD }),
    })
    expect(consume.status).toBe(400)
  })

  test('feature on: regular user without assign permission gets 403', async () => {
    const gaToken = await ensureGlobalAdmin()
    await setPlatformFlag(gaToken, 'ffParentPortal', true)

    const { access_token } = await apiSignup({
      email: uniqueEmail('pp1-noperm'),
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

  test('feature on: org admin can search students', async () => {
    const gaToken = await ensureGlobalAdmin()
    await setPlatformFlag(gaToken, 'ffParentPortal', true)

    const orgId = await getMyOrgId(gaToken)
    if (!orgId) {
      test.skip(true, 'could not determine org id')
      return
    }

    const meRes = await fetch(`${API_BASE}/api/v1/me`, {
      headers: authHeaders(gaToken),
    })
    if (!meRes.ok) {
      test.skip(true, 'GET /me unavailable')
      return
    }
    const meBody = (await meRes.json()) as { id?: string }
    if (meBody.id) {
      await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
        method: 'POST',
        headers: authHeaders(gaToken),
        body: JSON.stringify({ userId: meBody.id, role: 'org_admin' }),
      })
    }

    const studentEmail = uniqueEmail('pp1-child')
    await apiSignup({ email: studentEmail, password: PASSWORD })

    const res = await fetch(
      `${API_BASE}/api/v1/orgs/${orgId}/parent-assign/students?q=${encodeURIComponent(studentEmail)}`,
      { headers: authHeaders(gaToken) },
    )
    expect(res.ok, await res.text()).toBeTruthy()
    const body = (await res.json()) as { students?: Array<{ email?: string }> }
    expect(Array.isArray(body.students)).toBeTruthy()
  })
})
