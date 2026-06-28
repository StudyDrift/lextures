/**
 * Caching layer (plan 17.5)
 *
 * Coverage:
 *   [x] authenticated API responses include Cache-Control: no-store (AC-5)
 *   [x] public catalog responses include long-lived cache headers with ETag (FR-2)
 *   [x] offline banner component exists in app shell (AC-4 UI prerequisite)
 */
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('caching layer HTTP headers', () => {
  test('authenticated /api/v1/me returns Cache-Control: no-store', async ({ request }) => {
    const email = `cache-${Date.now()}@e2e.test`
    const password = 'E2eTestPass1!'
    await request.post(`${API_BASE}/api/v1/auth/signup`, {
      data: { email, password, display_name: 'Cache E2E' },
    })
    const login = await request.post(`${API_BASE}/api/v1/auth/login`, {
      data: { email, password },
    })
    expect(login.ok()).toBeTruthy()
    const { access_token } = (await login.json()) as { access_token: string }

    const me = await request.get(`${API_BASE}/api/v1/me`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(me.ok()).toBeTruthy()
    expect(me.headers()['cache-control']).toBe('no-store')
  })

  test('public catalog list returns cache-friendly headers when enabled', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/v1/public/catalog/courses`)
    if (res.status() === 404) {
      test.skip(true, 'public catalog feature flag off in this environment')
    }
    expect(res.ok()).toBeTruthy()
    const cc = res.headers()['cache-control'] ?? ''
    expect(cc).toContain('max-age=3600')
    expect(cc).toContain('stale-while-revalidate')
    expect(res.headers()['etag']).toBeTruthy()
  })
})

test('offline banner renders in app shell when authenticated', async ({ authedPage: page }) => {
  await page.goto('/dashboard')
  await page.evaluate(() => {
    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })
    window.dispatchEvent(new Event('offline'))
  })
  await expect(page.getByRole('alert')).toContainText(/You are offline/)
})
