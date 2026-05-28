/**
 * Time zones (plan 11.4): per-user timezone, course timezone, deadline display.
 */
import { test, expect } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Time zones', () => {
  test.use({ timezoneId: 'Asia/Tokyo' })

  test('signup with browser timezone stores Asia/Tokyo', async ({ page }) => {
    const email = `e2e-tz-${Date.now()}@test.invalid`
    const password = 'E2eTestPass1!'

    await page.goto('/signup')
    await page.getByLabel(/^email$/i).fill(email)
    await page.getByLabel(/^password$/i).fill(password)
    await expect(page.getByText(/We detected your time zone as Asia\/Tokyo/i)).toBeVisible({
      timeout: 8000,
    })
    await page.getByRole('button', { name: /create account/i }).click()
    await page.waitForURL(/\/(dashboard)?/, { timeout: 15000 })

    const token = await page.evaluate(() => localStorage.getItem('studydrift_access_token'))
    expect(token).toBeTruthy()

    const res = await fetch(`${apiBase}/api/v1/settings/timezone`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.ok).toBe(true)
    const data = (await res.json()) as { timezone?: string }
    expect(data.timezone).toBe('Asia/Tokyo')
  })

  test('PUT invalid timezone returns 422', async ({ authedToken }) => {
    const res = await fetch(`${apiBase}/api/v1/settings/timezone`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${authedToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ timezone: 'Not/A_Real_Zone' }),
    })
    expect(res.status).toBe(422)
    const body = (await res.json()) as { error?: { message?: string } }
    expect(body.error?.message).toMatch(/Invalid IANA timezone/i)
  })

  test('public timezones list includes Asia/Kolkata', async () => {
    const res = await fetch(`${apiBase}/api/v1/timezones`)
    expect(res.ok).toBe(true)
    const data = (await res.json()) as { timezones?: { id: string }[] }
    expect(data.timezones?.some((t) => t.id === 'Asia/Kolkata')).toBe(true)
  })

  test('account settings shows time zone section', async ({ authedPage: page }) => {
    await page.goto('/settings/account')
    await expect(page.getByText('Time zone', { exact: true })).toBeVisible({ timeout: 8000 })
    await expect(page.getByRole('searchbox')).toBeVisible()
  })
})
