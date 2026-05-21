/**
 * Public legal pages & acknowledgement banner (plan 20.1)
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Legal pages — public access', () => {
  test('privacy policy loads without login and includes FERPA and GDPR', async ({ page }) => {
    await page.goto('/privacy')
    await expect(page.getByRole('heading', { level: 1, name: /privacy policy/i })).toBeVisible({
      timeout: 8000,
    })
    await expect(page.getByText(/May 21, 2026/i).first()).toBeVisible()
    await expect(page.getByText(/FERPA/i).first()).toBeVisible()
    await expect(page.getByRole('heading', { name: /your rights under gdpr/i })).toBeVisible()
    await expect(page.getByText(/Anthropic/i).first()).toBeVisible()
    await expect(page.getByText(/privacy@lextures.com/i).first()).toBeVisible()
    await expect(page.getByRole('navigation', { name: /table of contents/i })).toBeVisible()
  })

  test('terms of service loads without login', async ({ page }) => {
    await page.goto('/terms')
    await expect(page.getByRole('heading', { level: 1, name: /terms of service/i })).toBeVisible({
      timeout: 8000,
    })
    await expect(page.getByRole('heading', { name: /acceptable use/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /dmca copyright policy/i })).toBeVisible()
  })

  test('privacy history page is reachable', async ({ page }) => {
    await page.goto('/privacy/history')
    await expect(page.getByRole('heading', { name: /history of changes/i })).toBeVisible({
      timeout: 8000,
    })
    await expect(page.getByRole('link', { name: /back to privacy policy/i })).toBeVisible()
  })
})

test.describe('Legal acknowledgement banner', () => {
  test('authenticated user sees banner and can acknowledge', async ({ page }) => {
    const email = `e2e-legal-${Date.now()}@test.invalid`
    const password = 'E2eTestPass1!Extra'

    const signupRes = await fetch(`${apiBase}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, displayName: 'Legal E2E' }),
    })
    expect(signupRes.ok).toBeTruthy()
    const { access_token: token } = (await signupRes.json()) as { access_token: string }

    await injectToken(page, token)
    await page.goto('/dashboard')

    const banner = page.getByRole('region', { name: /legal policy update/i })
    await expect(banner).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: /i acknowledge/i }).click()
    await expect(banner).not.toBeVisible({ timeout: 8000 })

    const pendingRes = await fetch(`${apiBase}/api/v1/legal/pending`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(pendingRes.ok).toBeTruthy()
    const pending = (await pendingRes.json()) as { documents: unknown[] }
    expect(pending.documents ?? []).toHaveLength(0)
  })
})
