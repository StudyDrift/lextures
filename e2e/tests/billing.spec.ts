/**
 * Stripe billing (plan 15.3).
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Billing — API auth', () => {
  test('GET /api/v1/me/entitlements returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/entitlements`)
    expect(res.status).toBe(401)
  })

  test('POST /api/v1/billing/checkout returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/billing/checkout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        successUrl: 'http://localhost:5173/checkout/success',
        cancelUrl: 'http://localhost:5173/checkout/cancel',
      }),
    })
    expect(res.status).toBe(401)
  })
})

test.describe('Billing — authenticated API', () => {
  test('student can list entitlements when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${apiBase}/api/v1/me/entitlements`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (res.status === 404) {
      test.skip(true, 'ff_stripe_billing not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { entitlements: unknown[] }
    expect(Array.isArray(body.entitlements)).toBe(true)
  })
})

test.describe('Billing — UI', () => {
  test('billing settings page loads when feature enabled', async ({ page, seededCourse }) => {
    const featRes = await fetch(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (!featRes.ok) {
      test.skip(true, 'platform features unavailable')
    }
    const feats = (await featRes.json()) as { ffStripeBilling?: boolean }
    if (!feats.ffStripeBilling) {
      test.skip(true, 'ff_stripe_billing not enabled in this environment')
    }

    await injectToken(page, seededCourse.studentToken)
    await page.goto('/me/billing')
    await expect(page.getByRole('heading', { name: /billing/i })).toBeVisible({
      timeout: 10_000,
    })
  })
})
