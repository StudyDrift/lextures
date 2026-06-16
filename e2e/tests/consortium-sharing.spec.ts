/**
 * Multi-campus consortium course sharing (plan 14.18)
 */
import { test, expect } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'consortium') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'consortium') {
  return `${uid(prefix)}@test.invalid`
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
  const paths = [
    '/api/v1/consortium/courses',
    '/api/v1/admin/consortium/agreements?orgId=00000000-0000-0000-0000-000000000001',
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

  const orgsRes = await fetch(`${API_BASE}/api/v1/admin/orgs`, { headers: authHeaders(adminToken) })
  expect(orgsRes.ok).toBeTruthy()
  const hostOrgs = (await orgsRes.json()) as { orgs?: { id: string }[] }
  const hostOrgId = hostOrgs.orgs?.[0]?.id
  expect(hostOrgId).toBeTruthy()

  const createGuestOrg = await fetch(`${API_BASE}/api/v1/admin/orgs`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ name: uid('guest-org'), slug: uid('guest'), orgType: 'higher-ed' }),
  })
  expect(createGuestOrg.status).toBe(201)
  const guestOrg = (await createGuestOrg.json()) as { org?: { id: string } }
  const guestOrgId = guestOrg.org?.id
  expect(guestOrgId).toBeTruthy()

  const createAgreement = await fetch(`${API_BASE}/api/v1/admin/consortium/agreements`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ hostOrgId, guestOrgId, status: 'active' }),
  })
  expect(createAgreement.status).toBe(201)

  const list = await fetch(
    `${API_BASE}/api/v1/admin/consortium/agreements?orgId=${encodeURIComponent(hostOrgId!)}`,
    { headers: authHeaders(adminToken) },
  )
  expect(list.ok).toBeTruthy()
  const data = (await list.json()) as { agreements?: { guestOrgId: string }[] }
  expect(data.agreements?.some((a) => a.guestOrgId === guestOrgId)).toBe(true)
})
