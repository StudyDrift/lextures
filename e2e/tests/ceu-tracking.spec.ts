/**
 * CEU seat-time tracking (plan 14.17).
 */
import { test, expect } from '@playwright/test'
import { injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('CEU tracking — API auth', () => {
  test('GET /api/v1/me/ce-transcript returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/ce-transcript`)
    expect(res.status).toBe(401)
  })

  test('POST /api/v1/seat-time/heartbeat returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/seat-time/heartbeat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        contentItemId: '00000000-0000-0000-0000-000000000001',
        sessionToken: 'e2e-token',
      }),
    })
    expect(res.status).toBe(401)
  })
})

test.describe('CEU tracking — authenticated API', () => {
  test('student can load CE transcript when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${apiBase}/api/v1/me/ce-transcript`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (res.status === 404) {
      test.skip(true, 'ff_ceu_tracking not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { awards: unknown[] }
    expect(Array.isArray(body.awards)).toBe(true)
  })
})

test.describe('CEU tracking — UI', () => {
  test('CE transcript page loads for student when feature enabled', async ({ page, seededCourse }) => {
    const featRes = await fetch(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (!featRes.ok) {
      test.skip(true, 'platform features unavailable')
    }
    const feats = (await featRes.json()) as { ffCeuTracking?: boolean }
    if (!feats.ffCeuTracking) {
      test.skip(true, 'ff_ceu_tracking not enabled in this environment')
    }

    await injectToken(page, seededCourse.studentToken)
    await page.goto('/me/ce-transcript')
    await expect(page.getByRole('heading', { name: /continuing education transcript/i })).toBeVisible({
      timeout: 10_000,
    })
  })
})
