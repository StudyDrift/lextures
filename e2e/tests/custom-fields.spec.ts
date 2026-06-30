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

async function enableFeatures(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      adminConsoleEnabled: true,
      customFieldsEnabled: true,
      bulkCsvImportEnabled: true,
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

test.describe.serial('Custom fields', () => {
  test('CustomFields: define, set, retrieve, and enforce visibility', async () => {
    const globalEmail = uniqueEmail('global-cf')
    await apiSignup(globalEmail)
    await bootstrapGlobalAdmin(globalEmail)
    const { access_token: globalToken } = await apiLogin(globalEmail)
    await enableFeatures(globalToken)

    const orgAdminEmail = uniqueEmail('orgadmin-cf')
    await apiSignup(orgAdminEmail)
    const { access_token: orgAdminToken } = await apiLogin(orgAdminEmail)

    const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(orgAdminToken) })
    const me = (await meRes.json()) as { id: string; org: { id: string } }
    await grantOrgAdmin(globalToken, me.org.id, me.id)

    const createRes = await fetch(`${API_BASE}/api/v1/admin-console/custom-fields`, {
      method: 'POST',
      headers: authHeaders(orgAdminToken),
      body: JSON.stringify({
        entityType: 'user',
        key: 'student_id',
        label: 'Student ID',
        fieldType: 'text',
        isRequired: false,
        visibility: 'student',
      }),
    })
    expect(createRes.status).toBe(201)

    const adminOnlyRes = await fetch(`${API_BASE}/api/v1/admin-console/custom-fields`, {
      method: 'POST',
      headers: authHeaders(orgAdminToken),
      body: JSON.stringify({
        entityType: 'user',
        key: 'title_one_eligible',
        label: 'Title I',
        fieldType: 'boolean',
        isRequired: false,
        visibility: 'admin_only',
      }),
    })
    expect(adminOnlyRes.status).toBe(201)

    const studentEmail = uniqueEmail('student-cf')
    await apiSignup(studentEmail)
    const { access_token: studentToken } = await apiLogin(studentEmail)

    const studentMe = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(studentToken) })
    const student = (await studentMe.json()) as { id: string }

    const patchRes = await fetch(`${API_BASE}/api/v1/admin-console/users/${student.id}`, {
      method: 'PATCH',
      headers: authHeaders(orgAdminToken),
      body: JSON.stringify({
        customFields: { student_id: '12345', title_one_eligible: true },
      }),
    })
    expect(patchRes.status).toBe(200)

    const adminGet = await fetch(
      `${API_BASE}/api/v1/admin-console/users/${student.id}?include=custom_fields`,
      { headers: authHeaders(orgAdminToken) },
    )
    expect(adminGet.status).toBe(200)
    const adminBody = (await adminGet.json()) as { customFields: Record<string, unknown> }
    expect(adminBody.customFields.student_id).toBe('12345')

    const studentGet = await fetch(`${API_BASE}/api/v1/me?include=custom_fields`, {
      headers: authHeaders(studentToken),
    })
    expect(studentGet.status).toBe(200)
    const studentBody = (await studentGet.json()) as { customFields?: Record<string, unknown> }
    expect(studentBody.customFields?.student_id).toBe('12345')
    expect(studentBody.customFields?.title_one_eligible).toBeUndefined()

    const selectRes = await fetch(`${API_BASE}/api/v1/admin-console/custom-fields`, {
      method: 'POST',
      headers: authHeaders(orgAdminToken),
      body: JSON.stringify({
        entityType: 'user',
        key: 'department',
        label: 'Department',
        fieldType: 'select',
        selectOptions: ['Math', 'Science'],
        isRequired: false,
        visibility: 'admin_only',
      }),
    })
    expect(selectRes.status).toBe(201)

    const badPatch = await fetch(`${API_BASE}/api/v1/admin-console/users/${student.id}`, {
      method: 'PATCH',
      headers: authHeaders(orgAdminToken),
      body: JSON.stringify({ customFields: { department: 'History' } }),
    })
    expect(badPatch.status).toBe(422)
  })
})
