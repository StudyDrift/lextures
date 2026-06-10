e2e/tests/ccr.spec.ts
/**
 * Co-Curricular Transcript (plan 14.13).
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('CCR — API auth', () => {
  test('GET /api/v1/me/ccr returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/ccr`)
    expect(res.status).toBe(401)
  })

  test('POST /api/v1/me/ccr/generate returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/ccr/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sharePublicly: false }),
    })
    expect(res.status).toBe(401)
  })

  test('GET verify endpoint is public (404/500 without token data, not 401)', async () => {
    const res = await fetch(`${apiBase}/api/v1/verify/00000000-0000-0000-0000-000000000099`)
    expect(res.status).not.toBe(401)
  })
})

test.describe('CCR — authenticated API', () => {
  test('student can load CCR summary when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${apiBase}/api/v1/me/ccr`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (res.status === 404) {
      test.skip(true, 'ff_co_curricular_transcript not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { achievements: unknown[]; documents: unknown[] }
    expect(Array.isArray(body.achievements)).toBe(true)
    expect(Array.isArray(body.documents)).toBe(true)
  })
})

test.describe('CCR — UI', () => {
  test('My CCR page loads for student when feature enabled', async ({ page, seededCourse }) => {
    await injectToken(page, seededCourse.studentToken)
    await page.goto('/me/ccr')
    await expect(page.getByRole('heading', { name: /comprehensive learner record/i })).toBeVisible({
      timeout: 10_000,
    })
  })

  test('public verify page renders', async ({ page }) => {
    await page.goto('/verify/test-token')
    await expect(page.getByRole('heading', { name: /credential verification/i })).toBeVisible()
  })
})
