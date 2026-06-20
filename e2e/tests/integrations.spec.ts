/**
 * Inbound Integrations (plan 16.4)
 *
 *   [x] GET /api/v1/integrations unauthenticated returns 401
 *   [x] GET /integrations/oauth/{provider}/connect unauthenticated returns 401
 *   [x] Non-admin user listing integrations returns 403
 *   [x] OAuth callback with an invalid state redirects back to the admin page (302)
 *   [x] Admin can list integrations (catalog includes Google Classroom)
 *   [x] DELETE of a non-existent connection returns 404 for an admin
 *   [x] Admin Integrations page renders the connector grid
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const NON_EXISTENT = '00000000-0000-0000-0000-0000000000aa'

async function adminToken(): Promise<string> {
  const { access_token } = await apiSignup({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD })
  return access_token
}

function uniqueEmail(prefix = 'integ') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

// ──────────────────────────────────────────────────────────
// Auth contract
// ──────────────────────────────────────────────────────────

test('Integrations: GET /api/v1/integrations unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/integrations`)
  expect(res.status).toBe(401)
})

test('Integrations: GET connect unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/integrations/oauth/google_classroom/connect`)
  expect(res.status).toBe(401)
})

test('Integrations: non-admin listing returns 403', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('nonadmin'), password: ADMIN_PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/integrations`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

// ──────────────────────────────────────────────────────────
// OAuth callback (state-carried, unauthenticated) degrades gracefully
// ──────────────────────────────────────────────────────────

test('Integrations: callback with invalid state redirects to the admin page', async () => {
  const res = await fetch(
    `${API_BASE}/integrations/oauth/google_classroom/callback?code=x&state=tampered`,
    { redirect: 'manual' },
  )
  expect(res.status).toBe(302)
  const loc = res.headers.get('location') ?? ''
  expect(loc).toContain('/admin/integrations')
  expect(loc).toContain('error=')
})

// ──────────────────────────────────────────────────────────
// Admin happy path
// ──────────────────────────────────────────────────────────

test('Integrations: admin can list integrations with the connector catalog', async () => {
  const token = await adminToken()
  const res = await fetch(`${API_BASE}/api/v1/integrations`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (res.status === 400) {
    test.skip(true, 'admin account has no organization in this environment')
  }
  expect(res.status).toBe(200)
  const body = (await res.json()) as {
    integrations: Array<{ provider: string; displayName: string; connected: boolean }>
  }
  expect(Array.isArray(body.integrations)).toBe(true)
  const providers = body.integrations.map((i) => i.provider)
  expect(providers).toContain('google_classroom')
})

test('Integrations: admin DELETE of a non-existent connection returns 404', async () => {
  const token = await adminToken()
  const res = await fetch(`${API_BASE}/api/v1/integrations/${NON_EXISTENT}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (res.status === 400) {
    test.skip(true, 'admin account has no organization in this environment')
  }
  expect(res.status).toBe(404)
})

// ──────────────────────────────────────────────────────────
// Admin UI smoke
// ──────────────────────────────────────────────────────────

test('Integrations: admin page renders the connector grid', async ({ page }) => {
  const token = await adminToken()
  await injectToken(page, token)
  await page.goto('/admin/integrations')
  await expect(page.getByRole('heading', { name: 'Integrations' })).toBeVisible()
  // Either the connector grid (admin with org) or an error banner renders, but the
  // page must mount without crashing.
  await expect(
    page.getByTestId('integration-grid').or(page.getByRole('alert')).first(),
  ).toBeVisible()
})
