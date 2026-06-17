/**
 * Creator revenue share & affiliate tracking (plan 15.8).
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Revenue share — API auth', () => {
  test('GET /api/v1/creator/earnings returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/creator/earnings`)
    expect(res.status).toBe(401)
  })

  test('POST /api/v1/creator/affiliate-codes returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/creator/affiliate-codes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    })
    expect(res.status).toBe(401)
  })
})

test.describe('Revenue share — authenticated API', () => {
  test('teacher can load earnings summary when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${apiBase}/api/v1/creator/earnings`, {
      headers: { Authorization: `Bearer ${seededCourse.teacherToken}` },
    })
    if (res.status === 404) {
      test.skip(true, 'ff_revenue_share not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { pendingCents: number; currency: string }
    expect(typeof body.pendingCents).toBe('number')
    expect(typeof body.currency).toBe('string')
  })
})

test.describe('Revenue share — UI', () => {
  test('creator earnings page loads when feature enabled', async ({ page, seededCourse }) => {
    const featRes = await fetch(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.teacherToken}` },
    })
    if (!featRes.ok) {
      test.skip(true, 'platform features unavailable')
    }
    const feats = (await featRes.json()) as { ffRevenueShare?: boolean }
    if (!feats.ffRevenueShare) {
      test.skip(true, 'ff_revenue_share not enabled in this environment')
    }

    await injectToken(page, seededCourse.teacherToken)
    await page.goto('/me/creator/earnings')
    await expect(page.getByRole('heading', { name: /creator earnings/i })).toBeVisible({
      timeout: 10_000,
    })
  })
})
