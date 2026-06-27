/**
 * Classroom bots (plan 16.6)
 *
 *   [x] GET /api/v1/bots unauthenticated returns 401
 *   [x] Non-admin listing bots returns 403
 *   [x] POST /integrations/slack/events with invalid signature returns 401
 *   [x] POST /integrations/discord/interactions with invalid signature returns 401
 *   [x] GET /api/v1/me/bot-links unauthenticated returns 401
 *   [x] Admin Integrations page renders classroom bot cards
 *   [x] Settings account page shows messaging app link panel
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

async function adminToken(): Promise<string> {
  const { access_token } = await apiSignup({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD })
  return access_token
}

function uniqueEmail(prefix = 'bots') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function expectUnauthenticatedOrDisabled(status: number) {
  expect([401, 501]).toContain(status)
}

// ──────────────────────────────────────────────────────────
// Auth contract
// ──────────────────────────────────────────────────────────

test('Bots: GET /api/v1/bots unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/bots`)
  expectUnauthenticatedOrDisabled(res.status)
})

test('Bots: non-admin listing returns 403', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('nonadmin'), password: ADMIN_PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/bots`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  if (res.status === 501) {
    test.skip(true, 'classroom bots not enabled in this environment')
  }
  expect(res.status).toBe(403)
})

test('Bots: GET /api/v1/me/bot-links unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/me/bot-links`)
  expectUnauthenticatedOrDisabled(res.status)
})

// ──────────────────────────────────────────────────────────
// Inbound signing verification
// ──────────────────────────────────────────────────────────

test('Bots: Slack events with invalid signature returns 401', async () => {
  const res = await fetch(`${API_BASE}/integrations/slack/events`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Slack-Request-Timestamp': '1',
      'X-Slack-Signature': 'v0=invalid',
    },
    body: JSON.stringify({ command: '/lextures', text: 'upcoming' }),
  })
  if (res.status === 501) {
    test.skip(true, 'Slack bot not enabled in this environment')
  }
  expect(res.status).toBe(401)
})

test('Bots: Discord interactions with invalid signature returns 401', async () => {
  const res = await fetch(`${API_BASE}/integrations/discord/interactions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Signature-Timestamp': '1',
      'X-Signature-Ed25519': 'deadbeef',
    },
    body: JSON.stringify({ type: 2, data: { name: 'lextures' } }),
  })
  if (res.status === 501) {
    test.skip(true, 'Discord bot not enabled in this environment')
  }
  expect(res.status).toBe(401)
})

// ──────────────────────────────────────────────────────────
// Admin API happy path
// ──────────────────────────────────────────────────────────

test('Bots: admin can list bot connections', async () => {
  const token = await adminToken()
  const res = await fetch(`${API_BASE}/api/v1/bots`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (res.status === 501) {
    test.skip(true, 'classroom bots not enabled in this environment')
  }
  if (res.status === 400) {
    test.skip(true, 'admin account has no organization in this environment')
  }
  expect(res.status).toBe(200)
  const body = (await res.json()) as { connections: unknown[] }
  expect(Array.isArray(body.connections)).toBe(true)
})

// ──────────────────────────────────────────────────────────
// UI smoke
// ──────────────────────────────────────────────────────────

test('Bots: admin integrations page renders classroom bot grid', async ({ page }) => {
  const token = await adminToken()
  await injectToken(page, token)
  await page.goto('/admin/integrations')
  await expect(page.getByRole('heading', { name: 'Integrations' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Classroom bots' })).toBeVisible()
  await expect(page.getByTestId('bot-grid')).toBeVisible()
  await expect(page.getByTestId('bot-card-slack')).toBeVisible()
  await expect(page.getByTestId('bot-card-discord')).toBeVisible()
  await expect(page.getByTestId('bot-card-teams')).toBeVisible()
})

test('Bots: account settings shows messaging apps panel', async ({ authedPage: page }) => {
  await page.goto('/settings/account')
  await expect(page.getByRole('heading', { name: 'Messaging apps' })).toBeVisible({ timeout: 8000 })
  await expect(page.getByText(/lextures upcoming/i)).toBeVisible()
})