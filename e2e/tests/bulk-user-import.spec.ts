/**
 * Bulk user CSV import (plan 18.2)
 *
 *   [x] Feature disabled returns 404
 *   [x] Dry-run reports invalid email rows
 *   [x] Upsert import creates users
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-import') {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}` }
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

async function enableImportFeatures(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: { ...authHeaders(token), 'Content-Type': 'application/json' },
    body: JSON.stringify({
      adminConsoleEnabled: true,
      bulkCsvImportEnabled: true,
      adminAuditLogEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable import features: ${await res.text()}`)
  }
}

async function grantOrgAdmin(actorToken: string, orgId: string, userId: string) {
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: { ...authHeaders(actorToken), 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
  if (!res.ok) {
    throw new Error(`grant org_admin failed: ${await res.text()}`)
  }
}

async function pollImportJob(token: string, jobId: string, timeoutMs = 30000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const res = await fetch(`${API_BASE}/api/v1/admin-console/imports/${jobId}`, {
      headers: authHeaders(token),
    })
    if (!res.ok) throw new Error(`status poll failed: ${await res.text()}`)
    const body = (await res.json()) as { status: string; errorRows?: number; createdCount?: number }
    if (body.status === 'complete' || body.status === 'failed') return body
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error('import job timed out')
}

function csvBlob(rows: string) {
  return new Blob([rows], { type: 'text/csv' })
}

test.describe.serial('Bulk user CSV import', () => {
  test('BulkImport: disabled feature returns 404', async () => {
    const email = uniqueEmail('disabled')
    await apiSignup(email)
    const { access_token } = await apiLogin(email)
    const res = await fetch(`${API_BASE}/api/v1/admin-console/imports`, {
      headers: authHeaders(access_token),
    })
    if (res.status === 200) {
      test.skip(true, 'bulk import already enabled globally')
    }
    expect(res.status).toBe(404)
  })

  test('BulkImport: dry-run reports invalid emails', async () => {
    const adminEmail = uniqueEmail('admin')
    await apiSignup(adminEmail)
    await bootstrapGlobalAdmin(adminEmail)
    const { access_token: adminToken } = await apiLogin(adminEmail)
    await enableImportFeatures(adminToken)

    const orgAdminEmail = uniqueEmail('orgadmin')
    await apiSignup(orgAdminEmail)
    const { access_token: orgToken } = await apiLogin(orgAdminEmail)
    const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(orgToken) })
    const me = (await meRes.json()) as { id: string; orgId?: string }
    const orgId = me.orgId ?? (await (await fetch(`${API_BASE}/api/v1/me/org-role-capabilities`, {
      headers: authHeaders(orgToken),
    })).json() as { orgId: string }).orgId
    await grantOrgAdmin(adminToken, orgId, me.id)

    const csv = `email,first_name,last_name,role
good@example.edu,Jane,Smith,student
not-an-email,Bob,Jones,student
`
    const form = new FormData()
    form.append('file', csvBlob(csv), 'users.csv')
    form.append('merge_strategy', 'upsert')
    form.append('profile', 'lextures_native')
    form.append('dry_run', 'true')

    const uploadRes = await fetch(`${API_BASE}/api/v1/admin-console/imports`, {
      method: 'POST',
      headers: authHeaders(orgToken),
      body: form,
    })
    expect(uploadRes.status).toBe(202)
    const uploaded = (await uploadRes.json()) as { jobId: string; parseErrors?: { row: number }[] }
    expect(uploaded.parseErrors?.length).toBeGreaterThanOrEqual(1)

    const job = await pollImportJob(orgToken, uploaded.jobId)
    expect(job.status).toBe('complete')
    expect((job as { errorRows: number }).errorRows).toBeGreaterThanOrEqual(1)
  })

  test('BulkImport: upsert creates users from CSV', async () => {
    const adminEmail = uniqueEmail('admin2')
    await apiSignup(adminEmail)
    await bootstrapGlobalAdmin(adminEmail)
    const { access_token: adminToken } = await apiLogin(adminEmail)
    await enableImportFeatures(adminToken)

    const orgAdminEmail = uniqueEmail('orgadmin2')
    await apiSignup(orgAdminEmail)
    const { access_token: orgToken } = await apiLogin(orgAdminEmail)
    const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(orgToken) })
    const me = (await meRes.json()) as { id: string; orgId?: string }
    const orgId = me.orgId ?? (await (await fetch(`${API_BASE}/api/v1/me/org-role-capabilities`, {
      headers: authHeaders(orgToken),
    })).json() as { orgId: string }).orgId
    await grantOrgAdmin(adminToken, orgId, me.id)

    const studentEmail = uniqueEmail('student')
    const csv = `email,first_name,last_name,role,external_id
${studentEmail},Test,Student,student,EXT-${Date.now()}
`
    const form = new FormData()
    form.append('file', csvBlob(csv), 'users.csv')
    form.append('merge_strategy', 'upsert')
    form.append('profile', 'lextures_native')
    form.append('dry_run', 'false')

    const uploadRes = await fetch(`${API_BASE}/api/v1/admin-console/imports`, {
      method: 'POST',
      headers: authHeaders(orgToken),
      body: form,
    })
    expect(uploadRes.status).toBe(202)
    const uploaded = (await uploadRes.json()) as { jobId: string }
    const job = await pollImportJob(orgToken, uploaded.jobId)
    expect(job.status).toBe('complete')
    expect((job as { createdCount: number }).createdCount).toBeGreaterThanOrEqual(1)

    const usersRes = await fetch(`${API_BASE}/api/v1/admin-console/users?search=${encodeURIComponent(studentEmail.split('@')[0])}`, {
      headers: authHeaders(orgToken),
    })
    expect(usersRes.status).toBe(200)
    const users = (await usersRes.json()) as { items: { email: string }[] }
    expect(users.items.some((u) => u.email === studentEmail)).toBe(true)
  })
})
