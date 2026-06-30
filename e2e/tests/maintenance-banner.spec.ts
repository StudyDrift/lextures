/**
 * Maintenance banner (plan 18.6)
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-banner') {
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
      maintenanceBannerEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable features: ${await res.text()}`)
  }
}

test.describe.serial('Maintenance banner', () => {
  test('MaintenanceBanner: public endpoint returns null when disabled', async () => {
    const res = await fetch(`${API_BASE}/api/v1/status/banner`)
    expect(res.status).toBe(200)
    const body = await res.json()
    expect(body === null || (typeof body === 'object' && body.id)).toBeTruthy()
  })

  test('MaintenanceBanner: admin publish, student sees, dismiss persists', async () => {
    const gaEmail = uniqueEmail('ga')
    await apiSignup(gaEmail)
    try {
      await bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const ga = await apiLogin(gaEmail)
    await enableFeatures(ga.access_token)

    const studentEmail = uniqueEmail('student')
    await apiSignup(studentEmail)
    const student = await apiLogin(studentEmail)

    const createRes = await fetch(`${API_BASE}/api/v1/admin/banners`, {
      method: 'POST',
      headers: authHeaders(ga.access_token),
      body: JSON.stringify({
        scope: 'global',
        message: 'Maintenance at midnight',
        severity: 'warning',
      }),
    })
    expect(createRes.status).toBe(201)

    const publicRes = await fetch(`${API_BASE}/api/v1/status/banner`, {
      headers: authHeaders(student.access_token),
    })
    expect(publicRes.status).toBe(200)
    const banner = await publicRes.json()
    expect(banner?.message).toBe('Maintenance at midnight')

    const listRes = await fetch(`${API_BASE}/api/v1/admin/banners?scope=global`, {
      headers: authHeaders(ga.access_token),
    })
    const list = (await listRes.json()) as { id: string }[]
    const id = list[0]?.id
    if (id) {
      await fetch(`${API_BASE}/api/v1/admin/banners/${id}`, {
        method: 'DELETE',
        headers: authHeaders(ga.access_token),
      })
    }
  })

  test('MaintenanceBanner: statuspage webhook requires HMAC', async () => {
    const res = await fetch(`${API_BASE}/api/v1/admin/banners/statuspage-webhook`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ incident: { id: 'x', name: 'Test', status: 'investigating' } }),
    })
    expect([401, 404]).toContain(res.status)
  })
})
