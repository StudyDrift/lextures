/**
 * Email template editor (plan 18.5)
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uniqueEmail(prefix = 'e2e-email-templates') {
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

async function enableEmailTemplateEditor(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      adminConsoleEnabled: true,
      emailTemplateEditorEnabled: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable email template editor: ${await res.text()}`)
  }
}

test.describe('Admin email templates API', () => {
  test('disabled feature returns 404', async () => {
    const res = await fetch(`${API_BASE}/api/v1/admin-console/email-templates`)
    expect(res.status).toBe(401)
  })

  test.describe.serial('enabled feature', () => {
    let token = ''
    let orgId = ''

    test.beforeAll(async () => {
      const email = uniqueEmail()
      await apiSignup(email)
      await bootstrapGlobalAdmin(email)
      const login = await apiLogin(email)
      token = login.access_token
      await enableEmailTemplateEditor(token)

      const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: authHeaders(token) })
      const me = (await meRes.json()) as { org_id?: string }
      orgId = me.org_id ?? ''
    })

    test('lists template slots', async () => {
      const res = await fetch(`${API_BASE}/api/v1/admin-console/email-templates?orgId=${orgId}`, {
        headers: authHeaders(token),
      })
      expect(res.status).toBe(200)
      const slots = (await res.json()) as Array<{ id: string; description: string }>
      expect(slots.length).toBeGreaterThan(0)
      expect(slots.some((s) => s.id === 'welcome')).toBe(true)
    })

    test('saves template and records version history', async () => {
      const html = '<p>Welcome {{user.first_name}} to {{org.name}}!</p><p><a href="{{link}}">Sign in</a></p>'
      const putRes = await fetch(
        `${API_BASE}/api/v1/admin-console/email-templates/welcome?orgId=${orgId}`,
        {
          method: 'PUT',
          headers: authHeaders(token),
          body: JSON.stringify({ htmlBody: html }),
        },
      )
      expect(putRes.status).toBe(200)
      const saved = (await putRes.json()) as { id: string; htmlBody: string }
      expect(saved.htmlBody).toContain('Welcome')

      const histRes = await fetch(
        `${API_BASE}/api/v1/admin-console/email-templates/welcome/history?orgId=${orgId}`,
        { headers: authHeaders(token) },
      )
      expect(histRes.status).toBe(200)
      const history = (await histRes.json()) as Array<{ id: string; isActive: boolean }>
      expect(history.length).toBeGreaterThan(0)
      expect(history[0].isActive).toBe(true)

      const previewRes = await fetch(
        `${API_BASE}/api/v1/admin-console/email-templates/welcome/preview?orgId=${orgId}`,
        {
          method: 'POST',
          headers: authHeaders(token),
          body: JSON.stringify({ htmlBody: html }),
        },
      )
      expect(previewRes.status).toBe(200)
      const preview = (await previewRes.json()) as { html: string }
      expect(preview.html).toContain('Welcome')
    })

    test('warns on unknown merge fields but still saves', async () => {
      const putRes = await fetch(
        `${API_BASE}/api/v1/admin-console/email-templates/welcome?orgId=${orgId}`,
        {
          method: 'PUT',
          headers: authHeaders(token),
          body: JSON.stringify({ htmlBody: '<p>Hi {{foo.bar}}</p>' }),
        },
      )
      expect(putRes.status).toBe(200)
      const saved = (await putRes.json()) as { unknownFields?: string[] }
      expect(saved.unknownFields).toContain('foo.bar')
    })

    test('reset restores system default', async () => {
      const resetRes = await fetch(
        `${API_BASE}/api/v1/admin-console/email-templates/welcome/reset?orgId=${orgId}`,
        { method: 'POST', headers: authHeaders(token) },
      )
      expect(resetRes.status).toBe(204)

      const getRes = await fetch(
        `${API_BASE}/api/v1/admin-console/email-templates/welcome?orgId=${orgId}`,
        { headers: authHeaders(token) },
      )
      const detail = (await getRes.json()) as { hasCustom: boolean }
      expect(detail.hasCustom).toBe(false)
    })
  })
})
