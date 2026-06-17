/**
 * Multi-campus consortium course sharing (plan 14.18)
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const DEFAULT_ORG_ID = '00000000-0000-0000-0000-000000000001'

function uid(prefix = 'consortium') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function uniqueEmail(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}@test.invalid`
}

/** URL-safe org slug within the 32-character server limit. */
function uniqueOrgSlug(prefix = 'guest') {
  return `e2e-${prefix}-${Date.now().toString(36).slice(-8)}`
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

function databaseUrlForBootstrap(): string | null {
  return (
    process.env.DATABASE_URL ??
    process.env.E2E_DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
}

/** Provision a guest org without moving the shared host admin into that tenant. */
async function createGuestOrgViaProvisioner(): Promise<string> {
  const email = uniqueEmail('e2e-consortium-prov')
  const signupRes = await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD, display_name: 'E2E Consortium Provisioner' }),
  })
  if (!signupRes.ok && signupRes.status !== 409) {
    const body = await signupRes.text()
    throw new Error(`Provisioner signup failed (${signupRes.status}): ${body}`)
  }

  const dsn = databaseUrlForBootstrap()
  if (!dsn) {
    test.skip(true, 'DATABASE_URL unavailable for Global Admin bootstrap')
  }
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: dsn },
    stdio: 'pipe',
  })

  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD }),
  })
  if (!loginRes.ok) {
    const body = await loginRes.text()
    throw new Error(`Provisioner login failed (${loginRes.status}): ${body}`)
  }
  const { access_token: provisionerToken } = (await loginRes.json()) as { access_token: string }

  const createGuestOrg = await fetch(`${API_BASE}/api/v1/admin/orgs`, {
    method: 'POST',
    headers: authHeaders(provisionerToken),
    body: JSON.stringify({ name: uid('guest-org'), slug: uniqueOrgSlug('guest') }),
  })
  if (!createGuestOrg.ok) {
    const body = await createGuestOrg.text()
    throw new Error(`Create guest org failed (${createGuestOrg.status}): ${body}`)
  }
  const guestOrg = (await createGuestOrg.json()) as { id: string }
  return guestOrg.id
}

async function enableFeatures(adminToken: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffConsortiumSharing: true,
      ffSisIntegration: true,
      updateMask: ['ffConsortiumSharing', 'ffSisIntegration'],
    }),
  })
  expect(res.ok).toBeTruthy()
}

test('Consortium: unauthenticated endpoints return 401', async () => {
  const adminToken = await getAdminToken()
  await enableFeatures(adminToken)
  const orgId = (await getAdminOrgId(adminToken)) ?? DEFAULT_ORG_ID

  const paths = [
    '/api/v1/consortium/courses',
    `/api/v1/admin/consortium/agreements?orgId=${encodeURIComponent(orgId)}`,
  ]
  for (const path of paths) {
    const res = await fetch(`${API_BASE}${path}`)
    expect(res.status).toBe(401)
  }
})

test('Consortium: feature off returns 404', async () => {
  const adminToken = await getAdminToken()
  await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ ffConsortiumSharing: false, updateMask: ['ffConsortiumSharing'] }),
  })
  const res = await fetch(`${API_BASE}/api/v1/consortium/courses`, {
    headers: authHeaders(adminToken),
  })
  expect(res.status).toBe(404)
})

test('Consortium: admin can create agreement when feature enabled', async () => {
  const adminToken = await getAdminToken()
  await enableFeatures(adminToken)

  const hostOrgId = (await getAdminOrgId(adminToken)) ?? DEFAULT_ORG_ID
  expect(hostOrgId).toBeTruthy()
  const adminId = await getAdminUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, hostOrgId, adminId)

  const guestOrgId = await createGuestOrgViaProvisioner()
  expect(guestOrgId).toBeTruthy()

  const createAgreement = await fetch(`${API_BASE}/api/v1/admin/consortium/agreements`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ hostOrgId, guestOrgId, status: 'active' }),
  })
  expect(createAgreement.status).toBe(201)

  const list = await fetch(
    `${API_BASE}/api/v1/admin/consortium/agreements?orgId=${encodeURIComponent(hostOrgId)}`,
    { headers: authHeaders(adminToken) },
  )
  expect(list.ok).toBeTruthy()
  const data = (await list.json()) as { agreements?: { guestOrgId: string }[] }
  expect(data.agreements?.some((a) => a.guestOrgId === guestOrgId)).toBe(true)
})
