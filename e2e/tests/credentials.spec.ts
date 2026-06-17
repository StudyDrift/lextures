/**
 * Completion credentials — LinkedIn share and Open Badges export (plan 15.6).
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Credentials — API auth', () => {
  test('GET /api/v1/me/credentials returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/credentials`)
    expect(res.status).toBe(401)
  })

  test('GET linkedin-params returns 401 without auth', async () => {
    const res = await fetch(
      `${apiBase}/api/v1/credentials/00000000-0000-0000-0000-000000000001/linkedin-params`,
    )
    expect(res.status).toBe(401)
  })

  test('GET verify endpoint is public (not 401)', async () => {
    const res = await fetch(
      `${apiBase}/api/v1/credentials/00000000-0000-0000-0000-000000000099/verify`,
    )
    expect(res.status).not.toBe(401)
  })
})

test.describe('Credentials — authenticated API', () => {
  test('student can load credentials when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${apiBase}/api/v1/me/credentials`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (res.status === 404) {
      test.skip(true, 'ff_completion_credentials not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { credentials: unknown[] }
    expect(Array.isArray(body.credentials)).toBe(true)
  })
})

test.describe('Credentials — UI', () => {
  test('My Credentials page loads when feature enabled', async ({ page, seededCourse }) => {
    const featRes = await fetch(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (!featRes.ok) {
      test.skip(true, 'platform features unavailable')
    }
    const feats = (await featRes.json()) as { ffCompletionCredentials?: boolean }
    if (!feats.ffCompletionCredentials) {
      test.skip(true, 'ff_completion_credentials not enabled in this environment')
    }

    await injectToken(page, seededCourse.studentToken)
    await page.goto('/me/credentials')
    await expect(page.getByRole('heading', { name: /my credentials/i })).toBeVisible({
      timeout: 10_000,
    })
  })

  test('public verify page renders for credential UUID route', async ({ page }) => {
    await page.goto('/verify/00000000-0000-0000-0000-000000000099')
    await expect(page.getByRole('heading', { name: /credential verification/i })).toBeVisible()
  })
})