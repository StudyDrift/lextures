/**
 * Custom fields (plan 18.7)
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-custom-fields') {
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

async function enableCustomFields(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      adminConsoleEnabled: true,
      customFieldsEnabled: true,
      bulkCsvImportEnabled: true,
      adminAuditLogEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable custom fields: ${await res.text()}`)
  }
}

test.describe('Admin custom fields API', () => {
  test('CustomFields: disabled feature returns 404', async () => {
    const email = uniqueEmail('disabled')
    await apiSignup(email)
    const { access_token } = await apiLogin(email)
    const res = await fetch(`${API_BASE}/api/v1/admin-console/custom-fields?entity_type=user`, {
      headers: authHeaders(access_token),
    })
    if (res.status === 200 || res.status === 403) {
      test.skip(true, 'custom fields already enabled globally')
    }
    expect(res.status).toBe(404)
  })

  test('CustomFields: define field, set value, retrieve with include', async () => {
    const email = uniqueEmail('admin')
    await apiSignup(email)
    await bootstrapGlobalAdmin(email)
    const { access_token } = await apiLogin(email)
    await enableCustomFields(access_token)

    const createRes = await fetch(`${API_BASE}/api/v1/admin-console/custom-fields`, {
      method: 'POST',
      headers: authHeaders(access_token),
      body: JSON.stringify({
        entityType: 'user',
        key: 'student_id',
        label: 'Student ID',
        fieldType: 'text',
        visibility: 'admin_only',
      }),
    })
    expect(createRes.status).toBe(201)

    const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(access_token) })
    expect(meRes.ok).toBeTruthy()
    const me = (await meRes.json()) as { id: string; org?: { id: string } }
    const userId = me.id

    const patchRes = await fetch(`${API_BASE}/api/v1/admin-console/users/${userId}`, {
      method: 'PATCH',
      headers: authHeaders(access_token),
      body: JSON.stringify({ customFields: { student_id: '12345' } }),
    })
    expect(patchRes.status).toBe(200)

    const getRes = await fetch(
      `${API_BASE}/api/v1/admin-console/users/${userId}?include=custom_fields`,
      { headers: authHeaders(access_token) },
    )
    expect(getRes.ok).toBeTruthy()
    const user = (await getRes.json()) as { customFields?: Record<string, string> }
    expect(user.customFields?.student_id).toBe('12345')
  })

  test('CustomFields: select validation returns 422', async () => {
    const email = uniqueEmail('select')
    await apiSignup(email)
    await bootstrapGlobalAdmin(email)
    const { access_token } = await apiLogin(email)
    await enableCustomFields(access_token)

    await fetch(`${API_BASE}/api/v1/admin-console/custom-fields`, {
      method: 'POST',
      headers: authHeaders(access_token),
      body: JSON.stringify({
        entityType: 'user',
        key: 'department',
        label: 'Department',
        fieldType: 'select',
        selectOptions: ['Math', 'Science'],
        visibility: 'admin_only',
      }),
    })

    const me = (await (await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(access_token) })).json()) as {
      id: string
    }

    const patchRes = await fetch(`${API_BASE}/api/v1/admin-console/users/${me.id}`, {
      method: 'PATCH',
      headers: authHeaders(access_token),
      body: JSON.stringify({ customFields: { department: 'History' } }),
    })
    expect(patchRes.status).toBe(422)
  })
})
